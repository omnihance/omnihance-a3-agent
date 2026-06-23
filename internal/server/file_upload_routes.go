package server

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"

	"github.com/omnihance/omnihance-a3-agent/internal/config"
	"github.com/omnihance/omnihance-a3-agent/internal/constants"
	"github.com/omnihance/omnihance-a3-agent/internal/logger"
	"github.com/omnihance/omnihance-a3-agent/internal/permissions"
	"github.com/omnihance/omnihance-a3-agent/internal/services"
	"github.com/omnihance/omnihance-a3-agent/internal/utils"
)

const (
	fileUploadTempDirName            = ".omnihance-upload-temp"
	fileUploadSessionTTL             = 2 * time.Hour
	fileUploadCleanupInterval        = 15 * time.Minute
	fileUploadDefaultRegistryDirName = ".revisions"
	fileUploadRegistryFileName       = "file-upload-temp-registry.json"
	fileUploadMaxChunkSize           = 8 * 1024 * 1024
)

type fileUploadManager struct {
	fileEditor   services.FileEditorService
	log          logger.Logger
	registryPath string
	maxFileSize  int64

	registryMu   sync.Mutex
	mu           sync.Mutex
	sessions     map[string]*fileUploadSession
	reservations map[string]string
	stopCh       chan struct{}
	stopDoneCh   chan struct{}
	started      bool
}

type fileUploadSession struct {
	ID              string
	DestinationPath string
	TempRoot        string
	CreatedAt       time.Time
	ExpiresAt       time.Time
	LastSeenAt      time.Time
	Files           map[string]*fileUploadFile
	OrderedFiles    []*fileUploadFile
	Reservations    []string
	Cancelled       bool
}

type fileUploadFile struct {
	ID                   string
	ClientFileID         string
	RelativePath         string
	ResolvedRelativePath string
	TargetPath           string
	TempPath             string
	Size                 int64
	ChunkSize            int64
	TotalChunks          int
	ReceivedChunks       map[int]int64
	Completed            bool
	FinalPath            string
}

func newFileUploadManager(registryDir string, fileEditor services.FileEditorService, log logger.Logger, maxFileSize int64) *fileUploadManager {
	if registryDir == "" {
		registryDir = fileUploadDefaultRegistryDirName
	}

	if maxFileSize <= 0 {
		maxFileSize = (&config.EnvVars{}).MaxFileUploadSizeBytes()
	}

	return &fileUploadManager{
		fileEditor:   fileEditor,
		log:          log,
		registryPath: filepath.Join(registryDir, fileUploadRegistryFileName),
		maxFileSize:  maxFileSize,
		sessions:     make(map[string]*fileUploadSession),
		reservations: make(map[string]string),
	}
}

func (m *fileUploadManager) Start() error {
	m.mu.Lock()
	if m.started {
		m.mu.Unlock()
		return nil
	}

	m.mu.Unlock()

	if err := m.cleanupRegisteredTempRoots(); err != nil {
		return err
	}

	stopCh := make(chan struct{})
	stopDoneCh := make(chan struct{})

	m.mu.Lock()
	if m.started {
		m.mu.Unlock()
		return nil
	}

	m.stopCh = stopCh
	m.stopDoneCh = stopDoneCh
	m.started = true
	m.mu.Unlock()

	go func() {
		defer close(stopDoneCh)

		ticker := time.NewTicker(fileUploadCleanupInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				m.cleanupExpiredSessions(time.Now())
			case <-stopCh:
				return
			}
		}
	}()

	return nil
}

func (m *fileUploadManager) Stop() {
	m.mu.Lock()
	if !m.started {
		m.mu.Unlock()
		return
	}

	stopCh := m.stopCh
	stopDoneCh := m.stopDoneCh
	m.started = false
	m.stopCh = nil
	m.stopDoneCh = nil
	close(stopCh)
	m.mu.Unlock()

	<-stopDoneCh
}

func (s *Server) ensureUploadManager() *fileUploadManager {
	if s.uploadManager != nil {
		return s.uploadManager
	}

	registryDir := fileUploadDefaultRegistryDirName
	if s.cfg != nil && s.cfg.RevisionsDirectory != "" {
		registryDir = s.cfg.RevisionsDirectory
	}

	maxFileSize := (&config.EnvVars{}).MaxFileUploadSizeBytes()
	if s.cfg != nil {
		maxFileSize = s.cfg.MaxFileUploadSizeBytes()
	}

	s.uploadManager = newFileUploadManager(registryDir, s.fileEditor, s.log, maxFileSize)
	if err := s.uploadManager.Start(); err != nil && s.log != nil {
		s.log.Error("Could not start file upload manager", logger.Field{Key: "error", Value: err})
	}

	return s.uploadManager
}

func (s *Server) handleCreateFileUpload(w http.ResponseWriter, r *http.Request) {
	if !s.requireUserPermission(w, r, permissions.ActionEditFiles) {
		return
	}

	var req CreateFileUploadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeFileUploadError(w, newFileUploadHTTPError(http.StatusBadRequest, constants.ErrorCodeBadRequest, "Invalid request body: "+err.Error()))
		return
	}

	validate := validator.New()
	if err := validate.Struct(req); err != nil {
		var validationErrors []string
		for _, fieldErr := range err.(validator.ValidationErrors) {
			validationErrors = append(validationErrors, fieldErr.Field()+" is required")
		}

		writeFileUploadError(w, newFileUploadHTTPError(http.StatusBadRequest, constants.ErrorCodeBadRequest, strings.Join(validationErrors, ", ")))
		return
	}

	response, err := s.ensureUploadManager().CreateSession(req)
	if err != nil {
		writeFileUploadError(w, fileUploadErrorFor(err))
		return
	}

	_ = utils.WriteJSONResponseWithStatus(w, http.StatusCreated, response)
}

func (s *Server) handleUploadFileChunk(w http.ResponseWriter, r *http.Request) {
	if !s.requireUserPermission(w, r, permissions.ActionEditFiles) {
		return
	}

	chunkIndex, err := strconv.Atoi(chi.URLParam(r, "chunk_index"))
	if err != nil || chunkIndex < 0 {
		writeFileUploadError(w, newFileUploadHTTPError(http.StatusBadRequest, constants.ErrorCodeBadRequest, "Invalid chunk index"))
		return
	}

	response, err := s.ensureUploadManager().UploadChunk(
		chi.URLParam(r, "upload_id"),
		chi.URLParam(r, "file_id"),
		chunkIndex,
		r.Body,
	)
	if err != nil {
		writeFileUploadError(w, fileUploadErrorFor(err))
		return
	}

	_ = utils.WriteJSONResponse(w, response)
}

func (s *Server) handleCompleteUploadedFile(w http.ResponseWriter, r *http.Request) {
	if !s.requireUserPermission(w, r, permissions.ActionEditFiles) {
		return
	}

	var req CompleteFileUploadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeFileUploadError(w, newFileUploadHTTPError(http.StatusBadRequest, constants.ErrorCodeBadRequest, "Invalid request body: "+err.Error()))
		return
	}

	validate := validator.New()
	if err := validate.Struct(req); err != nil {
		writeFileUploadError(w, newFileUploadHTTPError(http.StatusBadRequest, constants.ErrorCodeBadRequest, "sha256 is required"))
		return
	}

	response, err := s.ensureUploadManager().CompleteFile(chi.URLParam(r, "upload_id"), chi.URLParam(r, "file_id"), req.SHA256)
	if err != nil {
		writeFileUploadError(w, fileUploadErrorFor(err))
		return
	}

	_ = utils.WriteJSONResponse(w, response)
}

func (s *Server) handleFileUploadHeartbeat(w http.ResponseWriter, r *http.Request) {
	if !s.requireUserPermission(w, r, permissions.ActionEditFiles) {
		return
	}

	response, err := s.ensureUploadManager().Heartbeat(chi.URLParam(r, "upload_id"))
	if err != nil {
		writeFileUploadError(w, fileUploadErrorFor(err))
		return
	}

	_ = utils.WriteJSONResponse(w, response)
}

func (s *Server) handleCancelFileUpload(w http.ResponseWriter, r *http.Request) {
	if !s.requireUserPermission(w, r, permissions.ActionEditFiles) {
		return
	}

	if err := s.ensureUploadManager().CancelSession(chi.URLParam(r, "upload_id")); err != nil {
		writeFileUploadError(w, fileUploadErrorFor(err))
		return
	}

	_ = utils.WriteJSONResponse(w, map[string]interface{}{
		"message": "Upload cancelled",
	})
}

func (m *fileUploadManager) CreateSession(req CreateFileUploadRequest) (*CreateFileUploadResponse, error) {
	now := time.Now()
	m.cleanupExpiredSessions(now)

	destinationPath := filepath.Clean(strings.TrimSpace(req.DestinationPath))
	if destinationPath == "" || destinationPath == "." {
		return nil, newFileUploadHTTPError(http.StatusBadRequest, constants.ErrorCodeBadRequest, "Destination path is required")
	}

	info, err := m.fileEditor.Stat(destinationPath)
	if err != nil {
		if m.fileEditor.IsNotExist(err) {
			return nil, newFileUploadHTTPError(http.StatusNotFound, constants.ErrorCodeNotFound, "Destination path not found")
		}

		return nil, classifyFileUploadError(err, "Cannot read destination path")
	}

	if !info.IsDir() {
		return nil, newFileUploadHTTPError(http.StatusBadRequest, constants.ErrorCodeBadRequest, "Destination path must be a directory")
	}

	cleanFiles, err := cleanUploadFiles(req, m.maxFileSize)
	if err != nil {
		return nil, err
	}

	uploadID, err := generateUploadID()
	if err != nil {
		return nil, classifyFileUploadError(err, "Failed to create upload ID")
	}

	tempRoot := filepath.Join(destinationPath, fileUploadTempDirName, uploadID)
	if err := m.fileEditor.MkdirAll(tempRoot, 0700); err != nil {
		return nil, classifyFileUploadError(err, "Failed to create upload temp directory")
	}

	if err := m.registerTempRoot(tempRoot); err != nil {
		_ = m.fileEditor.RemoveAll(tempRoot)
		return nil, classifyFileUploadError(err, "Failed to register upload temp directory")
	}

	session := &fileUploadSession{
		ID:              uploadID,
		DestinationPath: destinationPath,
		TempRoot:        tempRoot,
		CreatedAt:       now,
		ExpiresAt:       now.Add(fileUploadSessionTTL),
		LastSeenAt:      now,
		Files:           make(map[string]*fileUploadFile, len(cleanFiles)),
		OrderedFiles:    make([]*fileUploadFile, 0, len(cleanFiles)),
		Reservations:    []string{},
	}

	response := &CreateFileUploadResponse{
		UploadID:  uploadID,
		ExpiresAt: session.ExpiresAt.Format(time.RFC3339),
		Files:     make([]CreateFileUploadResponseFile, 0, len(cleanFiles)),
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	topLevelResolvedNames := map[string]string{}
	requestReservations := map[string]bool{}
	for _, file := range cleanFiles {
		topLevelName := uploadTopLevelName(file.CleanRelativePath)
		if _, ok := topLevelResolvedNames[topLevelName]; ok {
			continue
		}

		targetPath, resolvedName, err := m.resolveAvailableTargetLocked(destinationPath, topLevelName, requestReservations)
		if err != nil {
			m.cancelSessionLocked(session)
			return nil, err
		}

		topLevelResolvedNames[topLevelName] = resolvedName
		reservationKey := m.reserveUploadPathLocked(session, targetPath)
		requestReservations[reservationKey] = true
	}

	for index, cleanFile := range cleanFiles {
		fileID := strconv.Itoa(index + 1)
		resolvedRelativePath := resolveUploadRelativePath(cleanFile.CleanRelativePath, topLevelResolvedNames)
		targetPath := filepath.Join(destinationPath, resolvedRelativePath)
		reservationKey := normalizeUploadReservationPath(targetPath)

		if reservedUploadID, reserved := m.reservations[reservationKey]; reserved && reservedUploadID != uploadID {
			targetPath, resolvedRelativePath, err = m.resolveAvailableNestedTargetLocked(destinationPath, resolvedRelativePath)
			if err != nil {
				m.cancelSessionLocked(session)
				return nil, err
			}
		}

		m.reserveUploadPathLocked(session, targetPath)

		uploadFile := &fileUploadFile{
			ID:                   fileID,
			ClientFileID:         cleanFile.ClientFileID,
			RelativePath:         cleanFile.OriginalRelativePath,
			ResolvedRelativePath: resolvedRelativePath,
			TargetPath:           targetPath,
			TempPath:             filepath.Join(tempRoot, fileID+".part"),
			Size:                 cleanFile.Size,
			ChunkSize:            req.ChunkSize,
			TotalChunks:          totalUploadChunks(cleanFile.Size, req.ChunkSize),
			ReceivedChunks:       map[int]int64{},
		}

		session.Files[fileID] = uploadFile
		session.OrderedFiles = append(session.OrderedFiles, uploadFile)
		response.Files = append(response.Files, CreateFileUploadResponseFile{
			ClientFileID:         uploadFile.ClientFileID,
			FileID:               uploadFile.ID,
			RelativePath:         uploadFile.RelativePath,
			ResolvedRelativePath: uploadFile.ResolvedRelativePath,
			TargetPath:           uploadFile.TargetPath,
			Size:                 uploadFile.Size,
			ChunkSize:            uploadFile.ChunkSize,
			TotalChunks:          uploadFile.TotalChunks,
		})
	}

	m.sessions[uploadID] = session
	return response, nil
}

func (m *fileUploadManager) UploadChunk(uploadID string, fileID string, chunkIndex int, body io.Reader) (*FileUploadChunkResponse, error) {
	m.mu.Lock()
	session, file, err := m.activeFileLocked(uploadID, fileID, time.Now())
	if err != nil {
		m.mu.Unlock()
		return nil, err
	}

	if file.Completed {
		m.mu.Unlock()
		return nil, newFileUploadHTTPError(http.StatusConflict, constants.ErrorCodeBadRequest, "File is already complete")
	}

	if chunkIndex >= file.TotalChunks {
		m.mu.Unlock()
		return nil, newFileUploadHTTPError(http.StatusBadRequest, constants.ErrorCodeBadRequest, "Chunk index is out of range")
	}

	expectedSize := expectedUploadChunkSize(file.Size, file.ChunkSize, chunkIndex)
	chunkSize := file.ChunkSize
	tempPath := file.TempPath
	tempRoot := session.TempRoot
	m.mu.Unlock()

	chunkData, err := io.ReadAll(io.LimitReader(body, expectedSize+1))
	if err != nil {
		return nil, classifyFileUploadError(err, "Failed to read upload chunk")
	}

	if int64(len(chunkData)) != expectedSize {
		return nil, newFileUploadHTTPError(http.StatusBadRequest, constants.ErrorCodeBadRequest, "Chunk size does not match expected size")
	}

	if err := m.fileEditor.MkdirAll(filepath.Dir(tempPath), 0700); err != nil {
		return nil, classifyFileUploadError(err, "Failed to create upload temp directory")
	}

	tempFile, err := m.fileEditor.OpenFile(tempPath, os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return nil, classifyFileUploadError(err, "Failed to open upload temp file")
	}

	_, writeErr := tempFile.WriteAt(chunkData, int64(chunkIndex)*chunkSize)
	closeErr := tempFile.Close()
	if writeErr != nil {
		return nil, classifyFileUploadError(writeErr, "Failed to write upload chunk")
	}

	if closeErr != nil {
		return nil, classifyFileUploadError(closeErr, "Failed to close upload temp file")
	}

	m.mu.Lock()
	session, file, err = m.activeFileLocked(uploadID, fileID, time.Now())
	if err != nil {
		m.mu.Unlock()
		m.removeTempRoot(tempRoot)
		return nil, err
	}

	if file.Completed {
		m.mu.Unlock()
		_ = m.fileEditor.Remove(tempPath)
		return nil, newFileUploadHTTPError(http.StatusConflict, constants.ErrorCodeBadRequest, "File is already complete")
	}

	if chunkIndex >= file.TotalChunks {
		m.mu.Unlock()
		_ = m.fileEditor.Remove(tempPath)
		return nil, newFileUploadHTTPError(http.StatusBadRequest, constants.ErrorCodeBadRequest, "Chunk index is out of range")
	}

	file.ReceivedChunks[chunkIndex] = expectedSize
	session.LastSeenAt = time.Now()
	session.ExpiresAt = session.LastSeenAt.Add(fileUploadSessionTTL)
	response := &FileUploadChunkResponse{
		Message:        "Chunk uploaded",
		ReceivedChunks: len(file.ReceivedChunks),
		TotalChunks:    file.TotalChunks,
	}
	m.mu.Unlock()

	return response, nil
}

func (m *fileUploadManager) CompleteFile(uploadID string, fileID string, clientSHA256 string) (*CompleteFileUploadResponse, error) {
	m.mu.Lock()
	session, file, err := m.activeFileLocked(uploadID, fileID, time.Now())
	if err != nil {
		m.mu.Unlock()
		return nil, err
	}

	if file.Completed {
		response := &CompleteFileUploadResponse{
			Message:              "File already uploaded",
			FileID:               file.ID,
			RelativePath:         file.RelativePath,
			ResolvedRelativePath: file.ResolvedRelativePath,
			FinalPath:            file.FinalPath,
			SHA256:               clientSHA256,
		}
		m.mu.Unlock()
		return response, nil
	}

	if len(file.ReceivedChunks) != file.TotalChunks {
		m.mu.Unlock()
		return nil, newFileUploadHTTPError(http.StatusBadRequest, constants.ErrorCodeBadRequest, "Not all chunks have been uploaded")
	}

	if file.Size == 0 {
		emptyFile, err := m.fileEditor.OpenFile(file.TempPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
		if err != nil {
			m.mu.Unlock()
			return nil, classifyFileUploadError(err, "Failed to create empty upload temp file")
		}

		if err := emptyFile.Close(); err != nil {
			m.mu.Unlock()
			return nil, classifyFileUploadError(err, "Failed to close empty upload temp file")
		}
	}

	serverSHA256, err := hashUploadFile(file.TempPath)
	if err != nil {
		m.mu.Unlock()
		return nil, classifyFileUploadError(err, "Failed to hash uploaded file")
	}

	normalizedClientHash := strings.ToLower(strings.TrimSpace(clientSHA256))
	if serverSHA256 != normalizedClientHash {
		_ = m.fileEditor.Remove(file.TempPath)
		m.releaseFileReservationLocked(session, file)
		m.mu.Unlock()
		return nil, newFileUploadHTTPError(http.StatusConflict, constants.ErrorCodeHashMismatch, "Uploaded file failed integrity check")
	}

	finalPath := file.TargetPath
	if _, err := m.fileEditor.Stat(finalPath); err == nil {
		resolvedPath, resolvedRelativePath, err := m.resolveAvailableNestedTargetLocked(session.DestinationPath, file.ResolvedRelativePath)
		if err != nil {
			m.mu.Unlock()
			return nil, err
		}

		m.releaseFileReservationLocked(session, file)
		finalPath = resolvedPath
		file.TargetPath = resolvedPath
		file.ResolvedRelativePath = resolvedRelativePath
		m.reserveUploadPathLocked(session, finalPath)
	} else if !m.fileEditor.IsNotExist(err) {
		m.mu.Unlock()
		return nil, classifyFileUploadError(err, "Cannot check final upload path")
	}

	if err := m.fileEditor.MkdirAll(filepath.Dir(finalPath), 0755); err != nil {
		m.mu.Unlock()
		return nil, classifyFileUploadError(err, "Failed to create upload destination directory")
	}

	if err := os.Rename(file.TempPath, finalPath); err != nil {
		m.mu.Unlock()
		return nil, classifyFileUploadError(err, "Failed to finalize uploaded file")
	}

	file.Completed = true
	file.FinalPath = finalPath
	session.LastSeenAt = time.Now()
	session.ExpiresAt = session.LastSeenAt.Add(fileUploadSessionTTL)
	m.releaseFileReservationLocked(session, file)
	m.removeSessionIfCompleteLocked(session)
	response := &CompleteFileUploadResponse{
		Message:              "File uploaded successfully",
		FileID:               file.ID,
		RelativePath:         file.RelativePath,
		ResolvedRelativePath: file.ResolvedRelativePath,
		FinalPath:            finalPath,
		SHA256:               serverSHA256,
	}
	m.mu.Unlock()

	return response, nil
}

func (m *fileUploadManager) Heartbeat(uploadID string) (*FileUploadHeartbeatResponse, error) {
	now := time.Now()
	m.mu.Lock()
	defer m.mu.Unlock()

	session, ok := m.sessions[uploadID]
	if !ok {
		return nil, newFileUploadHTTPError(http.StatusNotFound, constants.ErrorCodeNotFound, "Upload session not found")
	}

	if now.Sub(session.LastSeenAt) > fileUploadSessionTTL {
		m.cancelSessionLocked(session)
		return nil, newFileUploadHTTPError(http.StatusGone, constants.ErrorCodeUploadExpired, "Upload session expired")
	}

	session.LastSeenAt = now
	session.ExpiresAt = now.Add(fileUploadSessionTTL)
	return &FileUploadHeartbeatResponse{
		UploadID:  session.ID,
		ExpiresAt: session.ExpiresAt.Format(time.RFC3339),
	}, nil
}

func (m *fileUploadManager) CancelSession(uploadID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, ok := m.sessions[uploadID]
	if !ok {
		return newFileUploadHTTPError(http.StatusNotFound, constants.ErrorCodeNotFound, "Upload session not found")
	}

	m.cancelSessionLocked(session)
	return nil
}

func (m *fileUploadManager) cleanupExpiredSessions(now time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, session := range m.sessions {
		if now.Sub(session.LastSeenAt) > fileUploadSessionTTL {
			m.cancelSessionLocked(session)
		}
	}
}

func (m *fileUploadManager) cleanupRegisteredTempRoots() error {
	m.registryMu.Lock()
	defer m.registryMu.Unlock()

	tempRoots, err := m.readRegisteredTempRoots()
	if err != nil {
		return err
	}

	for _, tempRoot := range tempRoots {
		if err := m.fileEditor.RemoveAll(tempRoot); err != nil && !m.fileEditor.IsNotExist(err) && m.log != nil {
			m.log.Warn("Failed to cleanup upload temp directory", logger.Field{Key: "path", Value: tempRoot}, logger.Field{Key: "error", Value: err})
		}
	}

	return m.writeRegisteredTempRoots(nil)
}

func (m *fileUploadManager) activeFileLocked(uploadID string, fileID string, now time.Time) (*fileUploadSession, *fileUploadFile, error) {
	session, ok := m.sessions[uploadID]
	if !ok {
		return nil, nil, newFileUploadHTTPError(http.StatusNotFound, constants.ErrorCodeNotFound, "Upload session not found")
	}

	if now.Sub(session.LastSeenAt) > fileUploadSessionTTL {
		m.cancelSessionLocked(session)
		return nil, nil, newFileUploadHTTPError(http.StatusGone, constants.ErrorCodeUploadExpired, "Upload session expired")
	}

	file, ok := session.Files[fileID]
	if !ok {
		return nil, nil, newFileUploadHTTPError(http.StatusNotFound, constants.ErrorCodeNotFound, "Upload file not found")
	}

	session.LastSeenAt = now
	session.ExpiresAt = now.Add(fileUploadSessionTTL)
	return session, file, nil
}

func (m *fileUploadManager) cancelSessionLocked(session *fileUploadSession) {
	session.Cancelled = true
	m.releaseSessionReservationsLocked(session)

	delete(m.sessions, session.ID)
	m.removeTempRoot(session.TempRoot)
}

func (m *fileUploadManager) removeSessionIfCompleteLocked(session *fileUploadSession) {
	for _, file := range session.OrderedFiles {
		if !file.Completed {
			return
		}
	}

	m.releaseSessionReservationsLocked(session)
	delete(m.sessions, session.ID)
	m.removeTempRoot(session.TempRoot)
}

func (m *fileUploadManager) reserveUploadPathLocked(session *fileUploadSession, path string) string {
	reservation := normalizeUploadReservationPath(path)
	if m.reservations[reservation] != session.ID {
		m.reservations[reservation] = session.ID
		session.Reservations = append(session.Reservations, reservation)
	}

	return reservation
}

func (m *fileUploadManager) releaseFileReservationLocked(session *fileUploadSession, file *fileUploadFile) {
	reservation := normalizeUploadReservationPath(file.TargetPath)
	if m.reservations[reservation] == session.ID {
		delete(m.reservations, reservation)
	}
}

func (m *fileUploadManager) releaseSessionReservationsLocked(session *fileUploadSession) {
	for _, reservation := range session.Reservations {
		if m.reservations[reservation] == session.ID {
			delete(m.reservations, reservation)
		}
	}
}

func (m *fileUploadManager) resolveAvailableTargetLocked(parentPath string, name string, requestReservations map[string]bool) (string, string, error) {
	trimmedName := strings.TrimSpace(name)
	if trimmedName == "" {
		return "", "", newFileUploadHTTPError(http.StatusBadRequest, constants.ErrorCodeBadRequest, "Upload file name is required")
	}

	baseName, extension := splitUploadFileName(trimmedName)
	candidateName := trimmedName
	counter := 0
	for {
		candidatePath := filepath.Join(parentPath, candidateName)
		available, err := m.uploadPathAvailableLocked(candidatePath, requestReservations)
		if err != nil {
			return "", "", err
		}

		if available {
			return candidatePath, candidateName, nil
		}

		counter++
		if counter == 1 {
			candidateName = fmt.Sprintf("%s (copy)%s", baseName, extension)
		} else {
			candidateName = fmt.Sprintf("%s (copy %d)%s", baseName, counter, extension)
		}
	}
}

func (m *fileUploadManager) resolveAvailableNestedTargetLocked(destinationPath string, relativePath string) (string, string, error) {
	parentRelativePath := filepath.Dir(relativePath)
	fileName := filepath.Base(relativePath)
	parentPath := destinationPath
	if parentRelativePath != "." {
		parentPath = filepath.Join(destinationPath, parentRelativePath)
	}

	targetPath, resolvedName, err := m.resolveAvailableTargetLocked(parentPath, fileName, map[string]bool{})
	if err != nil {
		return "", "", err
	}

	if parentRelativePath == "." {
		return targetPath, resolvedName, nil
	}

	return targetPath, filepath.Join(parentRelativePath, resolvedName), nil
}

func (m *fileUploadManager) uploadPathAvailableLocked(path string, requestReservations map[string]bool) (bool, error) {
	reservationKey := normalizeUploadReservationPath(path)
	if m.reservations[reservationKey] != "" || requestReservations[reservationKey] {
		return false, nil
	}

	if _, err := m.fileEditor.Stat(path); err == nil {
		return false, nil
	} else if !m.fileEditor.IsNotExist(err) {
		return false, classifyFileUploadError(err, "Cannot check upload destination")
	}

	return true, nil
}

func (m *fileUploadManager) registerTempRoot(tempRoot string) error {
	m.registryMu.Lock()
	defer m.registryMu.Unlock()

	tempRoots, err := m.readRegisteredTempRoots()
	if err != nil {
		return err
	}

	for _, existing := range tempRoots {
		if filepath.Clean(existing) == filepath.Clean(tempRoot) {
			return nil
		}
	}

	tempRoots = append(tempRoots, tempRoot)
	return m.writeRegisteredTempRoots(tempRoots)
}

func (m *fileUploadManager) unregisterTempRoot(tempRoot string) error {
	m.registryMu.Lock()
	defer m.registryMu.Unlock()

	tempRoots, err := m.readRegisteredTempRoots()
	if err != nil {
		return err
	}

	filteredTempRoots := make([]string, 0, len(tempRoots))
	for _, existing := range tempRoots {
		if filepath.Clean(existing) == filepath.Clean(tempRoot) {
			continue
		}

		filteredTempRoots = append(filteredTempRoots, existing)
	}

	return m.writeRegisteredTempRoots(filteredTempRoots)
}

func (m *fileUploadManager) removeTempRoot(tempRoot string) {
	if err := m.fileEditor.RemoveAll(tempRoot); err != nil && !m.fileEditor.IsNotExist(err) && m.log != nil {
		m.log.Warn("Failed to remove upload temp directory", logger.Field{Key: "path", Value: tempRoot}, logger.Field{Key: "error", Value: err})
	}

	if err := m.unregisterTempRoot(tempRoot); err != nil && m.log != nil {
		m.log.Warn("Failed to update upload temp registry", logger.Field{Key: "path", Value: tempRoot}, logger.Field{Key: "error", Value: err})
	}
}

func (m *fileUploadManager) readRegisteredTempRoots() ([]string, error) {
	content, err := os.ReadFile(m.registryPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}

		return nil, err
	}

	var tempRoots []string
	if err := json.Unmarshal(content, &tempRoots); err != nil {
		return []string{}, nil
	}

	return tempRoots, nil
}

func (m *fileUploadManager) writeRegisteredTempRoots(tempRoots []string) error {
	if err := os.MkdirAll(filepath.Dir(m.registryPath), 0700); err != nil {
		return err
	}

	if tempRoots == nil {
		tempRoots = []string{}
	}

	content, err := json.Marshal(tempRoots)
	if err != nil {
		return err
	}

	return os.WriteFile(m.registryPath, content, 0600)
}

func cleanUploadFiles(req CreateFileUploadRequest, maxFileSize int64) ([]cleanUploadFile, error) {
	if len(req.Files) == 0 {
		return nil, newFileUploadHTTPError(http.StatusBadRequest, constants.ErrorCodeBadRequest, "At least one file is required")
	}

	if req.ChunkSize <= 0 {
		return nil, newFileUploadHTTPError(http.StatusBadRequest, constants.ErrorCodeBadRequest, "chunk_size must be greater than zero")
	}

	if req.ChunkSize > fileUploadMaxChunkSize {
		return nil, newFileUploadHTTPError(http.StatusBadRequest, constants.ErrorCodeBadRequest, "chunk_size exceeds maximum allowed size")
	}

	cleanFiles := make([]cleanUploadFile, 0, len(req.Files))
	seenRelativePaths := map[string]bool{}
	for _, file := range req.Files {
		cleanRelativePath, err := cleanUploadRelativePath(file.RelativePath)
		if err != nil {
			return nil, err
		}

		relativeKey := normalizeUploadReservationPath(cleanRelativePath)
		if seenRelativePaths[relativeKey] {
			return nil, newFileUploadHTTPError(http.StatusBadRequest, constants.ErrorCodeBadRequest, "Duplicate upload relative path: "+file.RelativePath)
		}

		if file.Size < 0 {
			return nil, newFileUploadHTTPError(http.StatusBadRequest, constants.ErrorCodeBadRequest, "Upload file size cannot be negative")
		}

		if maxFileSize > 0 && file.Size > maxFileSize {
			return nil, newFileUploadHTTPError(http.StatusRequestEntityTooLarge, constants.ErrorCodeFileTooLarge, uploadFileTooLargeMessage(cleanRelativePath, maxFileSize))
		}

		clientFileID := strings.TrimSpace(file.ClientFileID)
		if clientFileID == "" {
			return nil, newFileUploadHTTPError(http.StatusBadRequest, constants.ErrorCodeBadRequest, "Upload client file ID is required")
		}

		seenRelativePaths[relativeKey] = true
		cleanFiles = append(cleanFiles, cleanUploadFile{
			ClientFileID:         clientFileID,
			OriginalRelativePath: file.RelativePath,
			CleanRelativePath:    cleanRelativePath,
			Size:                 file.Size,
		})
	}

	return cleanFiles, nil
}

func cleanUploadRelativePath(relativePath string) (string, error) {
	trimmedPath := strings.TrimSpace(relativePath)
	if trimmedPath == "" {
		return "", newFileUploadHTTPError(http.StatusBadRequest, constants.ErrorCodeBadRequest, "Upload relative path is required")
	}

	normalizedPath := strings.ReplaceAll(trimmedPath, "\\", string(filepath.Separator))
	normalizedPath = filepath.FromSlash(normalizedPath)
	cleanPath := filepath.Clean(normalizedPath)
	if cleanPath == "." || filepath.IsAbs(cleanPath) || filepath.VolumeName(cleanPath) != "" {
		return "", newFileUploadHTTPError(http.StatusBadRequest, constants.ErrorCodeBadRequest, "Upload relative path is invalid")
	}

	parts := strings.Split(cleanPath, string(filepath.Separator))
	for _, part := range parts {
		if part == "" || part == "." || part == ".." {
			return "", newFileUploadHTTPError(http.StatusBadRequest, constants.ErrorCodeBadRequest, "Upload relative path is invalid")
		}
	}

	return filepath.Join(parts...), nil
}

func resolveUploadRelativePath(cleanRelativePath string, topLevelResolvedNames map[string]string) string {
	parts := strings.Split(cleanRelativePath, string(filepath.Separator))
	if len(parts) == 0 {
		return cleanRelativePath
	}

	resolvedTopLevel, ok := topLevelResolvedNames[parts[0]]
	if !ok {
		return cleanRelativePath
	}

	parts[0] = resolvedTopLevel
	return filepath.Join(parts...)
}

func uploadTopLevelName(cleanRelativePath string) string {
	parts := strings.Split(cleanRelativePath, string(filepath.Separator))
	return parts[0]
}

func splitUploadFileName(name string) (string, string) {
	lastDotIndex := strings.LastIndex(name, ".")
	if lastDotIndex <= 0 {
		return name, ""
	}

	return name[:lastDotIndex], name[lastDotIndex:]
}

func totalUploadChunks(size int64, chunkSize int64) int {
	if size == 0 {
		return 0
	}

	return int(math.Ceil(float64(size) / float64(chunkSize)))
}

func expectedUploadChunkSize(size int64, chunkSize int64, chunkIndex int) int64 {
	offset := int64(chunkIndex) * chunkSize
	remaining := size - offset
	if remaining < chunkSize {
		return remaining
	}

	return chunkSize
}

func hashUploadFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer func() {
		_ = file.Close()
	}()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

func generateUploadID() (string, error) {
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}

	return hex.EncodeToString(raw), nil
}

func normalizeUploadReservationPath(path string) string {
	return strings.ToLower(filepath.Clean(path))
}

func classifyFileUploadError(err error, prefix string) error {
	if os.IsPermission(err) {
		return newFileUploadHTTPError(http.StatusForbidden, constants.ErrorCodeForbidden, prefix+": permission denied")
	}

	if errors.Is(err, syscall.ENOSPC) || strings.Contains(strings.ToLower(err.Error()), "no space") || strings.Contains(strings.ToLower(err.Error()), "disk full") {
		return newFileUploadHTTPError(http.StatusInsufficientStorage, constants.ErrorCodeDiskFull, prefix+": not enough disk space")
	}

	return newFileUploadHTTPError(http.StatusInternalServerError, constants.ErrorCodeInternalServerError, prefix+": "+err.Error())
}

func writeFileUploadError(w http.ResponseWriter, err *fileUploadHTTPError) {
	_ = utils.WriteJSONResponseWithStatus(w, err.status, map[string]interface{}{
		"errorCode": err.errorCode,
		"context":   "file-upload",
		"errors":    []string{err.message},
	})
}

func fileUploadErrorFor(err error) *fileUploadHTTPError {
	var uploadErr *fileUploadHTTPError
	if errors.As(err, &uploadErr) {
		return uploadErr
	}

	return newFileUploadHTTPError(http.StatusInternalServerError, constants.ErrorCodeInternalServerError, err.Error())
}

func newFileUploadHTTPError(status int, errorCode string, message string) *fileUploadHTTPError {
	return &fileUploadHTTPError{
		status:    status,
		errorCode: errorCode,
		message:   message,
	}
}

func (e *fileUploadHTTPError) Error() string {
	return e.message
}

type fileUploadHTTPError struct {
	status    int
	errorCode string
	message   string
}

type cleanUploadFile struct {
	ClientFileID         string
	OriginalRelativePath string
	CleanRelativePath    string
	Size                 int64
}

type CreateFileUploadRequest struct {
	DestinationPath string                        `json:"destination_path" validate:"required"`
	ChunkSize       int64                         `json:"chunk_size" validate:"required,min=1"`
	Files           []CreateFileUploadRequestFile `json:"files" validate:"required,min=1,dive"`
}

type CreateFileUploadRequestFile struct {
	ClientFileID string `json:"client_file_id" validate:"required"`
	RelativePath string `json:"relative_path" validate:"required"`
	Size         int64  `json:"size" validate:"min=0"`
}

type CreateFileUploadResponse struct {
	UploadID  string                         `json:"upload_id"`
	ExpiresAt string                         `json:"expires_at"`
	Files     []CreateFileUploadResponseFile `json:"files"`
}

type CreateFileUploadResponseFile struct {
	ClientFileID         string `json:"client_file_id"`
	FileID               string `json:"file_id"`
	RelativePath         string `json:"relative_path"`
	ResolvedRelativePath string `json:"resolved_relative_path"`
	TargetPath           string `json:"target_path"`
	Size                 int64  `json:"size"`
	ChunkSize            int64  `json:"chunk_size"`
	TotalChunks          int    `json:"total_chunks"`
}

type FileUploadChunkResponse struct {
	Message        string `json:"message"`
	ReceivedChunks int    `json:"received_chunks"`
	TotalChunks    int    `json:"total_chunks"`
}

type CompleteFileUploadRequest struct {
	SHA256 string `json:"sha256" validate:"required,len=64,hexadecimal"`
}

type CompleteFileUploadResponse struct {
	Message              string `json:"message"`
	FileID               string `json:"file_id"`
	RelativePath         string `json:"relative_path"`
	ResolvedRelativePath string `json:"resolved_relative_path"`
	FinalPath            string `json:"final_path"`
	SHA256               string `json:"sha256"`
}

type FileUploadHeartbeatResponse struct {
	UploadID  string `json:"upload_id"`
	ExpiresAt string `json:"expires_at"`
}
