package server

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/omnihance/omnihance-a3-agent/internal/constants"
	"github.com/omnihance/omnihance-a3-agent/internal/logger"
	"github.com/omnihance/omnihance-a3-agent/internal/mw"
	"github.com/omnihance/omnihance-a3-agent/internal/services"
	"github.com/omnihance/omnihance-a3-agent/internal/utils"
)

func (s *Server) InitializeFileSystemRoutes(r *chi.Mux) {
	r.Route("/api/file-tree", func(r chi.Router) {
		r.Use(mw.CheckCookie(s.internalDB, s.cfg.CookieSecret))
		r.Get("/", s.handleFileTree)
		r.Get("/npc-file", s.handleNPCFileData)
		r.Put("/npc-file", s.handleUpdateNPCFile)
		r.Get("/text-file", s.handleTextFileData)
		r.Put("/text-file", s.handleUpdateTextFile)
		r.Get("/spawn-file", s.handleSpawnFileData)
		r.Put("/spawn-file", s.handleUpdateSpawnFile)
		r.Post("/revert-file", s.handleRevertFile)
		r.Get("/revision-summary", s.handleRevisionSummary)
	})
}

func (s *Server) handleFileTree(w http.ResponseWriter, r *http.Request) {
	pathParam := r.URL.Query().Get("path")
	showDotfiles, _ := strconv.ParseBool(r.URL.Query().Get("show_dotfiles"))

	var rootNode *FileNode
	var err error

	if pathParam == "" {
		rootNode, err = s.getSystemRoots(showDotfiles)
	} else {
		cleanPath := filepath.Clean(pathParam)
		rootNode, err = s.getDirectoryNode(cleanPath, showDotfiles)
	}

	if err != nil {
		if os.IsNotExist(err) {
			_ = utils.WriteJSONResponseWithStatus(w, http.StatusNotFound, map[string]interface{}{
				"errorCode": constants.ErrorCodeNotFound,
				"context":   "file-system",
				"errors":    []string{"Path not found"},
			})
			return
		}

		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeInternalServerError,
			"context":   "file-system",
			"errors":    []string{err.Error()},
		})
		return
	}

	response := FileTreeResponse{
		OS:       runtime.GOOS,
		FileTree: rootNode,
	}

	_ = utils.WriteJSONResponse(w, response)
}

func (s *Server) getSystemRoots(showDotfiles bool) (*FileNode, error) {
	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "A3 Online Server"
	}

	root := &FileNode{
		ID:       "root",
		Name:     hostname,
		Kind:     "directory",
		Depth:    0,
		Children: []*FileNode{},
	}

	if runtime.GOOS == "windows" {
		for _, drive := range "ABCDEFGHIJKLMNOPQRSTUVWXYZ" {
			drivePath := string(drive) + ":\\"
			if info, err := os.Stat(drivePath); err == nil {
				modTime := info.ModTime()
				node := &FileNode{
					ID:           utils.GenerateMD5Hash(drivePath),
					Name:         string(drive) + ":",
					Kind:         "directory",
					Depth:        1,
					LastModified: &modTime,
					Permissions:  info.Mode().String(),
					Children:     []*FileNode{},
				}
				root.Children = append(root.Children, node)
			}
		}
	} else {
		rootPath := "/"
		entries, err := os.ReadDir(rootPath)
		if err != nil {
			return nil, err
		}

		for _, entry := range entries {
			if !showDotfiles && len(entry.Name()) > 0 && entry.Name()[0] == '.' {
				continue
			}

			node := s.createNodeFromEntry(rootPath, entry, 1)
			root.Children = append(root.Children, node)
		}
	}

	return root, nil
}

func (s *Server) getDirectoryNode(path string, showDotfiles bool) (*FileNode, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	name := info.Name()
	if runtime.GOOS == "windows" && len(path) == 3 && path[1] == ':' && (path[2] == '\\' || path[2] == '/') {
		name = path[:2]
	}

	modTime := info.ModTime()
	node := &FileNode{
		ID:           utils.GenerateMD5Hash(path),
		Name:         name,
		Kind:         "file",
		Depth:        0,
		LastModified: &modTime,
		Permissions:  info.Mode().String(),
		Children:     []*FileNode{},
	}

	if info.IsDir() {
		node.Kind = "directory"
		entries, err := os.ReadDir(path)
		if err == nil {
			for _, entry := range entries {
				if !showDotfiles && len(entry.Name()) > 0 && entry.Name()[0] == '.' {
					continue
				}

				child := s.createNodeFromEntry(path, entry, 1)
				node.Children = append(node.Children, child)
			}
		}
	} else {
		node.FileSize = info.Size()
		node.FileExtension = filepath.Ext(name)
		node.MimeType = mime.TypeByExtension(node.FileExtension)
		node.FileType = s.fileEditor.GetFileType(path, info)
		node.IsEditable = s.fileEditor.IsFileEditable(path, info)
		node.IsViewable = s.fileEditor.IsFileViewable(path, info)
		node.APIEndpoint = s.fileEditor.GetFileAPIEndpoint(path, info)
	}

	return node, nil
}

func (s *Server) createNodeFromEntry(parentPath string, entry os.DirEntry, depth int) *FileNode {
	kind := "file"
	if entry.IsDir() {
		kind = "directory"
	}

	fullPath := filepath.Join(parentPath, entry.Name())

	node := &FileNode{
		ID:       utils.GenerateMD5Hash(fullPath),
		Name:     entry.Name(),
		Kind:     kind,
		Depth:    depth,
		Children: []*FileNode{},
	}

	info, err := entry.Info()
	if err == nil {
		modTime := info.ModTime()
		node.LastModified = &modTime
		node.Permissions = info.Mode().String()
		if !entry.IsDir() {
			node.FileSize = info.Size()
			node.FileExtension = filepath.Ext(entry.Name())
			node.MimeType = mime.TypeByExtension(node.FileExtension)
			node.FileType = s.fileEditor.GetFileType(fullPath, info)
			node.IsEditable = s.fileEditor.IsFileEditable(fullPath, info)
			node.IsViewable = s.fileEditor.IsFileViewable(fullPath, info)
			node.APIEndpoint = s.fileEditor.GetFileAPIEndpoint(fullPath, info)
		}
	}

	return node
}

func (s *Server) handleNPCFileData(w http.ResponseWriter, r *http.Request) {
	pathParam := r.URL.Query().Get("path")

	if pathParam == "" {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusBadRequest, map[string]interface{}{
			"errorCode": constants.ErrorCodeBadRequest,
			"context":   "file-system",
			"errors":    []string{"Path parameter is required"},
		})
		return
	}

	cleanPath := filepath.Clean(pathParam)
	info, err := os.Stat(cleanPath)

	if err != nil {
		if os.IsNotExist(err) {
			_ = utils.WriteJSONResponseWithStatus(w, http.StatusNotFound, map[string]interface{}{
				"errorCode": constants.ErrorCodeNotFound,
				"context":   "file-system",
				"errors":    []string{"Path not found"},
			})
			return
		}

		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeFileReadError,
			"context":   "file-system",
			"errors":    []string{"Cannot read file: " + err.Error()},
		})
		return
	}

	if info.IsDir() {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusBadRequest, map[string]interface{}{
			"errorCode": constants.ErrorCodePathIsDirectory,
			"context":   "file-system",
			"errors":    []string{"Path is a directory, not a file"},
		})
		return
	}

	fileType := s.fileEditor.GetFileType(cleanPath, info)
	if fileType != services.FileTypeNPC {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusBadRequest, map[string]interface{}{
			"errorCode": constants.ErrorCodeFileNotViewable,
			"context":   "file-system",
			"errors":    []string{"File is not an NPC file"},
		})
		return
	}

	npcData, err := s.fileEditor.ReadNPCFileData(cleanPath)
	if err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeFileReadError,
			"context":   "file-system",
			"errors":    []string{"Failed to read NPC file data: " + err.Error()},
		})
		return
	}

	id := npcData.Id
	respawnRate := npcData.RespawnRate
	attackTypeInfo := npcData.AttackTypeInfo
	targetSelectionInfo := npcData.TargetSelectionInfo
	defense := npcData.Defense
	additionalDefense := npcData.AdditionalDefense
	attackSpeedLow := npcData.AttackSpeedLow
	attackSpeedHigh := npcData.AttackSpeedHigh
	movementSpeed := npcData.MovementSpeed
	level := npcData.Level
	playerExp := npcData.PlayerExp
	appearance := npcData.Appearance
	hp := npcData.HP
	blueAttackDefense := npcData.BlueAttackDefense
	redAttackDefense := npcData.RedAttackDefense
	greyAttackDefense := npcData.GreyAttackDefense
	mercenaryExp := npcData.MercenaryExp
	apiData := NPCFileAPIData{
		Name:                utils.ReadStringFromBytes(npcData.Name[:]),
		Id:                  &id,
		RespawnRate:         &respawnRate,
		AttackTypeInfo:      &attackTypeInfo,
		TargetSelectionInfo: &targetSelectionInfo,
		Defense:             &defense,
		AdditionalDefense:   &additionalDefense,
		Attacks:             npcData.Attacks[:],
		AttackSpeedLow:      &attackSpeedLow,
		AttackSpeedHigh:     &attackSpeedHigh,
		MovementSpeed:       &movementSpeed,
		Level:               &level,
		PlayerExp:           &playerExp,
		Appearance:          &appearance,
		HP:                  &hp,
		BlueAttackDefense:   &blueAttackDefense,
		RedAttackDefense:    &redAttackDefense,
		GreyAttackDefense:   &greyAttackDefense,
		MercenaryExp:        &mercenaryExp,
	}

	_ = utils.WriteJSONResponse(w, apiData)
}

func (s *Server) handleTextFileData(w http.ResponseWriter, r *http.Request) {
	pathParam := r.URL.Query().Get("path")

	if pathParam == "" {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusBadRequest, map[string]interface{}{
			"errorCode": constants.ErrorCodeBadRequest,
			"context":   "file-system",
			"errors":    []string{"Path parameter is required"},
		})
		return
	}

	cleanPath := filepath.Clean(pathParam)
	info, err := os.Stat(cleanPath)

	if err != nil {
		if os.IsNotExist(err) {
			_ = utils.WriteJSONResponseWithStatus(w, http.StatusNotFound, map[string]interface{}{
				"errorCode": constants.ErrorCodeNotFound,
				"context":   "file-system",
				"errors":    []string{"Path not found"},
			})
			return
		}

		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeFileReadError,
			"context":   "file-system",
			"errors":    []string{"Cannot read file: " + err.Error()},
		})
		return
	}

	if info.IsDir() {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusBadRequest, map[string]interface{}{
			"errorCode": constants.ErrorCodePathIsDirectory,
			"context":   "file-system",
			"errors":    []string{"Path is a directory, not a file"},
		})
		return
	}

	fileType := s.fileEditor.GetFileType(cleanPath, info)
	if fileType != services.FileTypeText {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusBadRequest, map[string]interface{}{
			"errorCode": constants.ErrorCodeFileNotViewable,
			"context":   "file-system",
			"errors":    []string{"File is not a text file"},
		})
		return
	}

	content, err := os.ReadFile(cleanPath)
	if err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeFileReadError,
			"context":   "file-system",
			"errors":    []string{"Failed to read text file: " + err.Error()},
		})
		return
	}

	apiData := TextFileAPIData{
		Content: string(content),
	}

	_ = utils.WriteJSONResponse(w, apiData)
}

func (s *Server) handleSpawnFileData(w http.ResponseWriter, r *http.Request) {
	pathParam := r.URL.Query().Get("path")
	if pathParam == "" {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusBadRequest, map[string]interface{}{
			"errorCode": constants.ErrorCodeBadRequest,
			"context":   "file-system",
			"errors":    []string{"Path parameter is required"},
		})
		return
	}

	cleanPath := filepath.Clean(pathParam)
	info, err := os.Stat(cleanPath)
	if err != nil {
		if os.IsNotExist(err) {
			_ = utils.WriteJSONResponseWithStatus(w, http.StatusNotFound, map[string]interface{}{
				"errorCode": constants.ErrorCodeNotFound,
				"context":   "file-system",
				"errors":    []string{"Path not found"},
			})
			return
		}

		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeFileReadError,
			"context":   "file-system",
			"errors":    []string{"Cannot read file: " + err.Error()},
		})
		return
	}

	if info.IsDir() {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusBadRequest, map[string]interface{}{
			"errorCode": constants.ErrorCodePathIsDirectory,
			"context":   "file-system",
			"errors":    []string{"Path is a directory, not a file"},
		})
		return
	}

	fileType := s.fileEditor.GetFileType(cleanPath, info)
	if fileType != services.FileTypeSpawn {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusBadRequest, map[string]interface{}{
			"errorCode": constants.ErrorCodeFileNotViewable,
			"context":   "file-system",
			"errors":    []string{"File is not a spawn file"},
		})
		return
	}

	spawnData, err := s.fileEditor.ReadSpawnFileData(cleanPath)
	if err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeFileReadError,
			"context":   "file-system",
			"errors":    []string{"Failed to read spawn file data: " + err.Error()},
		})
		return
	}

	apiSpawns := make([]NPCSpawnAPIData, len(spawnData))
	for i, spawn := range spawnData {
		id := spawn.Id
		x := spawn.X
		y := spawn.Y
		unknown1 := spawn.Unknown1
		orientation := spawn.Orientation
		spwanStep := spawn.SpwanStep
		apiSpawns[i] = NPCSpawnAPIData{
			Id:          &id,
			X:           &x,
			Y:           &y,
			Unknown1:    &unknown1,
			Orientation: &orientation,
			SpwanStep:   &spwanStep,
		}
	}

	apiData := SpawnFileAPIData{
		Spawns: apiSpawns,
	}

	_ = utils.WriteJSONResponse(w, apiData)
}

func (s *Server) validateFileUpdateRequest(w http.ResponseWriter, r *http.Request, expectedFileType services.FileType, fileTypeName string) (*fileUpdateContext, bool) {
	userID, ok := utils.GetUserIdFromContext(r.Context())
	if !ok {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusUnauthorized, map[string]interface{}{
			"errorCode": constants.ErrorCodeUnauthorized,
			"context":   "file-system",
			"errors":    []string{"User ID not found in context"},
		})
		return nil, false
	}

	pathParam := r.URL.Query().Get("path")
	if pathParam == "" {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusBadRequest, map[string]interface{}{
			"errorCode": constants.ErrorCodeBadRequest,
			"context":   "file-system",
			"errors":    []string{"Path parameter is required"},
		})
		return nil, false
	}

	cleanPath := filepath.Clean(pathParam)
	info, err := os.Stat(cleanPath)
	if err != nil {
		if os.IsNotExist(err) {
			_ = utils.WriteJSONResponseWithStatus(w, http.StatusNotFound, map[string]interface{}{
				"errorCode": constants.ErrorCodeNotFound,
				"context":   "file-system",
				"errors":    []string{"Path not found"},
			})
			return nil, false
		}

		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeFileReadError,
			"context":   "file-system",
			"errors":    []string{"Cannot read file: " + err.Error()},
		})
		return nil, false
	}

	if info.IsDir() {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusBadRequest, map[string]interface{}{
			"errorCode": constants.ErrorCodePathIsDirectory,
			"context":   "file-system",
			"errors":    []string{"Path is a directory, not a file"},
		})
		return nil, false
	}

	fileType := s.fileEditor.GetFileType(cleanPath, info)
	if fileType != expectedFileType {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusBadRequest, map[string]interface{}{
			"errorCode": constants.ErrorCodeFileNotViewable,
			"context":   "file-system",
			"errors":    []string{"File is not a " + fileTypeName + " file"},
		})
		return nil, false
	}

	if !s.fileEditor.IsFileEditable(cleanPath, info) {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusBadRequest, map[string]interface{}{
			"errorCode": constants.ErrorCodeBadRequest,
			"context":   "file-system",
			"errors":    []string{"File is not editable"},
		})
		return nil, false
	}

	return &fileUpdateContext{
		userID:    userID,
		cleanPath: cleanPath,
		info:      info,
		fileID:    utils.GenerateMD5Hash(cleanPath),
	}, true
}

func (s *Server) createFileRevision(w http.ResponseWriter, ctx *fileUpdateContext, previousData []byte, currentData []byte) (int64, bool) {
	previousHash := utils.CalculateFileHash(previousData)
	currentHash := utils.CalculateFileHash(currentData)

	if previousHash == currentHash {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusBadRequest, map[string]interface{}{
			"errorCode": constants.ErrorCodeBadRequest,
			"context":   "file-system",
			"errors":    []string{"No changes detected. The file content is identical to the existing content."},
		})
		return 0, false
	}

	lockPath, err := s.acquireFileLock(ctx.fileID)
	if err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusConflict, map[string]interface{}{
			"errorCode": constants.ErrorCodeBadRequest,
			"context":   "file-system",
			"errors":    []string{err.Error()},
		})
		return 0, false
	}

	defer s.releaseFileLock(lockPath)

	tx, err := s.internalDB.BeginTx()
	if err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeInternalServerError,
			"context":   "file-system",
			"errors":    []string{"Failed to begin transaction: " + err.Error()},
		})
		return 0, false
	}

	defer func() {
		if err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				s.log.Error("Failed to rollback transaction", logger.Field{Key: "error", Value: rollbackErr})
			}
		}
	}()

	revisionID, err := s.internalDB.CreateFileRevision(tx, ctx.fileID, ctx.cleanPath, "", previousHash, currentHash, ctx.userID)
	if err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeInternalServerError,
			"context":   "file-system",
			"errors":    []string{"Failed to create revision: " + err.Error()},
		})
		return 0, false
	}

	revisionDir := filepath.Join(s.cfg.RevisionsDirectory, ctx.fileID, strconv.FormatInt(revisionID, 10))
	if err = os.MkdirAll(revisionDir, 0755); err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeInternalServerError,
			"context":   "file-system",
			"errors":    []string{"Failed to create revision directory: " + err.Error()},
		})
		return 0, false
	}

	epochTime := time.Now().Unix()
	fileName := filepath.Base(ctx.cleanPath)
	revisionFileName := strconv.FormatInt(epochTime, 10) + "_" + fileName
	revisionPath := filepath.Join(revisionDir, revisionFileName)
	if err = os.WriteFile(revisionPath, previousData, 0644); err != nil {
		if removeErr := os.RemoveAll(revisionDir); removeErr != nil {
			s.log.Error("Failed to remove revision directory during cleanup", logger.Field{Key: "error", Value: removeErr})
		}

		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeInternalServerError,
			"context":   "file-system",
			"errors":    []string{"Failed to save revision copy: " + err.Error()},
		})
		return 0, false
	}

	if err = s.internalDB.UpdateFileRevisionPath(tx, revisionID, revisionPath, ctx.userID); err != nil {
		if removeErr := os.Remove(revisionPath); removeErr != nil {
			s.log.Error("Failed to remove revision file during cleanup", logger.Field{Key: "error", Value: removeErr})
		}

		if removeErr := os.RemoveAll(revisionDir); removeErr != nil {
			s.log.Error("Failed to remove revision directory during cleanup", logger.Field{Key: "error", Value: removeErr})
		}

		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeInternalServerError,
			"context":   "file-system",
			"errors":    []string{"Failed to update revision path: " + err.Error()},
		})
		return 0, false
	}

	if err = s.internalDB.UpdateFileRevisionStatus(tx, revisionID, "completed", ctx.userID); err != nil {
		if removeErr := os.Remove(revisionPath); removeErr != nil {
			s.log.Error("Failed to remove revision file during cleanup", logger.Field{Key: "error", Value: removeErr})
		}

		if removeErr := os.RemoveAll(revisionDir); removeErr != nil {
			s.log.Error("Failed to remove revision directory during cleanup", logger.Field{Key: "error", Value: removeErr})
		}

		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeInternalServerError,
			"context":   "file-system",
			"errors":    []string{"Failed to update revision status: " + err.Error()},
		})
		return 0, false
	}

	if err = tx.Commit(); err != nil {
		if removeErr := os.Remove(revisionPath); removeErr != nil {
			s.log.Error("Failed to remove revision file during cleanup", logger.Field{Key: "error", Value: removeErr})
		}

		if removeErr := os.RemoveAll(revisionDir); removeErr != nil {
			s.log.Error("Failed to remove revision directory during cleanup", logger.Field{Key: "error", Value: removeErr})
		}

		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeInternalServerError,
			"context":   "file-system",
			"errors":    []string{"Failed to commit transaction: " + err.Error()},
		})
		return 0, false
	}

	return revisionID, true
}

func (s *Server) handleUpdateNPCFile(w http.ResponseWriter, r *http.Request) {
	ctx, ok := s.validateFileUpdateRequest(w, r, services.FileTypeNPC, "NPC")
	if !ok {
		return
	}

	var req NPCFileAPIData
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusBadRequest, map[string]interface{}{
			"errorCode": constants.ErrorCodeBadRequest,
			"context":   "file-system",
			"errors":    []string{"Invalid request body: " + err.Error()},
		})
		return
	}

	validate := validator.New()
	if err := validate.Struct(req); err != nil {
		var errors []string
		for _, err := range err.(validator.ValidationErrors) {
			errors = append(errors, err.Field()+" is required")
		}

		_ = utils.WriteJSONResponseWithStatus(w, http.StatusBadRequest, map[string]interface{}{
			"errorCode": constants.ErrorCodeBadRequest,
			"context":   "file-system",
			"errors":    errors,
		})
		return
	}

	previousData, err := os.ReadFile(ctx.cleanPath)
	if err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeFileReadError,
			"context":   "file-system",
			"errors":    []string{"Failed to read file: " + err.Error()},
		})
		return
	}

	var nameBytes [0x14]byte
	copy(nameBytes[:], []byte(req.Name))

	var attacks [0x3]services.NPCAttack
	copy(attacks[:], req.Attacks)

	npcData := &services.NPCFileData{
		Name:                nameBytes,
		Id:                  *req.Id,
		RespawnRate:         *req.RespawnRate,
		AttackTypeInfo:      *req.AttackTypeInfo,
		TargetSelectionInfo: *req.TargetSelectionInfo,
		Defense:             *req.Defense,
		AdditionalDefense:   *req.AdditionalDefense,
		Attacks:             attacks,
		AttackSpeedLow:      *req.AttackSpeedLow,
		AttackSpeedHigh:     *req.AttackSpeedHigh,
		MovementSpeed:       *req.MovementSpeed,
		Level:               *req.Level,
		PlayerExp:           *req.PlayerExp,
		Appearance:          *req.Appearance,
		HP:                  *req.HP,
		BlueAttackDefense:   *req.BlueAttackDefense,
		RedAttackDefense:    *req.RedAttackDefense,
		GreyAttackDefense:   *req.GreyAttackDefense,
		MercenaryExp:        *req.MercenaryExp,
		Unknown:             0,
	}

	var currentDataBuffer bytes.Buffer
	if err := binary.Write(&currentDataBuffer, binary.LittleEndian, npcData); err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeInternalServerError,
			"context":   "file-system",
			"errors":    []string{"Failed to serialize NPC data: " + err.Error()},
		})
		return
	}

	currentData := currentDataBuffer.Bytes()

	revisionID, ok := s.createFileRevision(w, ctx, previousData, currentData)
	if !ok {
		return
	}

	if err = s.fileEditor.WriteNPCFileData(ctx.cleanPath, npcData); err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeInternalServerError,
			"context":   "file-system",
			"errors":    []string{"Failed to write file: " + err.Error()},
		})
		return
	}

	_ = utils.WriteJSONResponse(w, map[string]interface{}{
		"message":     "File updated successfully",
		"revision_id": revisionID,
	})
}

func (s *Server) handleUpdateTextFile(w http.ResponseWriter, r *http.Request) {
	ctx, ok := s.validateFileUpdateRequest(w, r, services.FileTypeText, "text")
	if !ok {
		return
	}

	var req TextFileAPIData
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusBadRequest, map[string]interface{}{
			"errorCode": constants.ErrorCodeBadRequest,
			"context":   "file-system",
			"errors":    []string{"Invalid request body: " + err.Error()},
		})
		return
	}

	validate := validator.New()
	if err := validate.Struct(req); err != nil {
		var errors []string
		for _, err := range err.(validator.ValidationErrors) {
			errors = append(errors, err.Field()+" is required")
		}
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusBadRequest, map[string]interface{}{
			"errorCode": constants.ErrorCodeBadRequest,
			"context":   "file-system",
			"errors":    errors,
		})
		return
	}

	previousData, err := os.ReadFile(ctx.cleanPath)
	if err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeFileReadError,
			"context":   "file-system",
			"errors":    []string{"Failed to read file: " + err.Error()},
		})
		return
	}

	currentData := []byte(req.Content)

	revisionID, ok := s.createFileRevision(w, ctx, previousData, currentData)
	if !ok {
		return
	}

	if err = s.fileEditor.WriteTextFileData(ctx.cleanPath, req.Content); err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeInternalServerError,
			"context":   "file-system",
			"errors":    []string{"Failed to write file: " + err.Error()},
		})
		return
	}

	_ = utils.WriteJSONResponse(w, map[string]interface{}{
		"message":     "File updated successfully",
		"revision_id": revisionID,
	})
}

func (s *Server) handleUpdateSpawnFile(w http.ResponseWriter, r *http.Request) {
	ctx, ok := s.validateFileUpdateRequest(w, r, services.FileTypeSpawn, "spawn")
	if !ok {
		return
	}

	var req SpawnFileAPIData
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusBadRequest, map[string]interface{}{
			"errorCode": constants.ErrorCodeBadRequest,
			"context":   "file-system",
			"errors":    []string{"Invalid request body: " + err.Error()},
		})
		return
	}

	validate := validator.New()
	if err := validate.Struct(req); err != nil {
		var errors []string
		for _, err := range err.(validator.ValidationErrors) {
			errors = append(errors, err.Field()+" is required")
		}

		_ = utils.WriteJSONResponseWithStatus(w, http.StatusBadRequest, map[string]interface{}{
			"errorCode": constants.ErrorCodeBadRequest,
			"context":   "file-system",
			"errors":    errors,
		})
		return
	}

	previousData, err := os.ReadFile(ctx.cleanPath)
	if err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeFileReadError,
			"context":   "file-system",
			"errors":    []string{"Failed to read file: " + err.Error()},
		})
		return
	}

	spawnData := make([]services.NPCSpawnData, len(req.Spawns))
	for i, spawn := range req.Spawns {
		spawnData[i] = services.NPCSpawnData{
			Id:          *spawn.Id,
			X:           *spawn.X,
			Y:           *spawn.Y,
			Unknown1:    *spawn.Unknown1,
			Orientation: *spawn.Orientation,
			SpwanStep:   *spawn.SpwanStep,
		}
	}

	var currentDataBuffer bytes.Buffer
	if err := binary.Write(&currentDataBuffer, binary.LittleEndian, spawnData); err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeInternalServerError,
			"context":   "file-system",
			"errors":    []string{"Failed to serialize spawn data: " + err.Error()},
		})
		return
	}

	currentData := currentDataBuffer.Bytes()

	revisionID, ok := s.createFileRevision(w, ctx, previousData, currentData)
	if !ok {
		return
	}

	if err = s.fileEditor.WriteSpawnFileData(ctx.cleanPath, spawnData); err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeInternalServerError,
			"context":   "file-system",
			"errors":    []string{"Failed to write file: " + err.Error()},
		})
		return
	}

	_ = utils.WriteJSONResponse(w, map[string]interface{}{
		"message":     "File updated successfully",
		"revision_id": revisionID,
	})
}

func (s *Server) acquireFileLock(fileID string) (string, error) {
	locksDir := filepath.Join(s.cfg.RevisionsDirectory, "locks")
	if err := os.MkdirAll(locksDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create locks directory: %w", err)
	}

	lockPath := filepath.Join(locksDir, fileID+".lock")
	lockFile, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		if os.IsExist(err) {
			return "", fmt.Errorf("file is currently being edited by another process")
		}
		return "", fmt.Errorf("failed to create lock file: %w", err)
	}
	if err := lockFile.Close(); err != nil {
		s.log.Error("Failed to close lock file", logger.Field{Key: "error", Value: err})
	}

	return lockPath, nil
}

func (s *Server) handleRevertFile(w http.ResponseWriter, r *http.Request) {
	userID, ok := utils.GetUserIdFromContext(r.Context())
	if !ok {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusUnauthorized, map[string]interface{}{
			"errorCode": constants.ErrorCodeUnauthorized,
			"context":   "file-system",
			"errors":    []string{"User ID not found in context"},
		})
		return
	}

	pathParam := r.URL.Query().Get("path")
	if pathParam == "" {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusBadRequest, map[string]interface{}{
			"errorCode": constants.ErrorCodeBadRequest,
			"context":   "file-system",
			"errors":    []string{"Path parameter is required"},
		})
		return
	}

	cleanPath := filepath.Clean(pathParam)
	info, err := os.Stat(cleanPath)
	if err != nil {
		if os.IsNotExist(err) {
			_ = utils.WriteJSONResponseWithStatus(w, http.StatusNotFound, map[string]interface{}{
				"errorCode": constants.ErrorCodeNotFound,
				"context":   "file-system",
				"errors":    []string{"Path not found"},
			})
			return
		}

		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeFileReadError,
			"context":   "file-system",
			"errors":    []string{"Cannot read file: " + err.Error()},
		})
		return
	}

	if info.IsDir() {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusBadRequest, map[string]interface{}{
			"errorCode": constants.ErrorCodePathIsDirectory,
			"context":   "file-system",
			"errors":    []string{"Path is a directory, not a file"},
		})
		return
	}

	fileID := utils.GenerateMD5Hash(cleanPath)

	lockPath, err := s.acquireFileLock(fileID)
	if err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusConflict, map[string]interface{}{
			"errorCode": constants.ErrorCodeBadRequest,
			"context":   "file-system",
			"errors":    []string{err.Error()},
		})
		return
	}

	defer s.releaseFileLock(lockPath)

	revision, err := s.internalDB.GetLastCompletedFileRevision(fileID)
	if err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeInternalServerError,
			"context":   "file-system",
			"errors":    []string{"Failed to query file revisions: " + err.Error()},
		})
		return
	}

	if revision == nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusNotFound, map[string]interface{}{
			"errorCode": constants.ErrorCodeNotFound,
			"context":   "file-system",
			"errors":    []string{"No completed revisions found for this file"},
		})
		return
	}

	if _, err := os.Stat(revision.RevisionPath); os.IsNotExist(err) {
		tx, err := s.internalDB.BeginTx()
		if err != nil {
			_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
				"errorCode": constants.ErrorCodeInternalServerError,
				"context":   "file-system",
				"errors":    []string{"Failed to begin transaction: " + err.Error()},
			})
			return
		}

		defer func() {
			if err != nil {
				if rollbackErr := tx.Rollback(); rollbackErr != nil {
					s.log.Error("Failed to rollback transaction", logger.Field{Key: "error", Value: rollbackErr})
				}
			}
		}()

		if err = s.internalDB.UpdateFileRevisionStatus(tx, revision.ID, "corrupted", userID); err != nil {
			_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
				"errorCode": constants.ErrorCodeInternalServerError,
				"context":   "file-system",
				"errors":    []string{"Failed to update revision status: " + err.Error()},
			})
			return
		}

		if err = tx.Commit(); err != nil {
			_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
				"errorCode": constants.ErrorCodeInternalServerError,
				"context":   "file-system",
				"errors":    []string{"Failed to commit transaction: " + err.Error()},
			})
			return
		}

		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeInternalServerError,
			"context":   "file-system",
			"errors":    []string{"Revision file is missing or corrupted"},
		})
		return
	}

	revisionData, err := os.ReadFile(revision.RevisionPath)
	if err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeFileReadError,
			"context":   "file-system",
			"errors":    []string{"Failed to read revision file: " + err.Error()},
		})
		return
	}

	tx, err := s.internalDB.BeginTx()
	if err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeInternalServerError,
			"context":   "file-system",
			"errors":    []string{"Failed to begin transaction: " + err.Error()},
		})
		return
	}

	defer func() {
		if err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				s.log.Error("Failed to rollback transaction", logger.Field{Key: "error", Value: rollbackErr})
			}
		}
	}()

	if err = s.internalDB.UpdateFileRevisionStatus(tx, revision.ID, "reverted", userID); err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeInternalServerError,
			"context":   "file-system",
			"errors":    []string{"Failed to update revision status: " + err.Error()},
		})
		return
	}

	if err = tx.Commit(); err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeInternalServerError,
			"context":   "file-system",
			"errors":    []string{"Failed to commit transaction: " + err.Error()},
		})
		return
	}

	err = nil

	if err = os.WriteFile(cleanPath, revisionData, 0644); err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeInternalServerError,
			"context":   "file-system",
			"errors":    []string{"Failed to write file: " + err.Error()},
		})
		return
	}

	_ = utils.WriteJSONResponse(w, map[string]interface{}{
		"message":     "File reverted successfully",
		"revision_id": revision.ID,
	})
}

func (s *Server) handleRevisionSummary(w http.ResponseWriter, r *http.Request) {
	pathParam := r.URL.Query().Get("path")
	if pathParam == "" {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusBadRequest, map[string]interface{}{
			"errorCode": constants.ErrorCodeBadRequest,
			"context":   "file-system",
			"errors":    []string{"Path parameter is required"},
		})
		return
	}

	cleanPath := filepath.Clean(pathParam)
	info, err := os.Stat(cleanPath)
	if err != nil {
		if os.IsNotExist(err) {
			_ = utils.WriteJSONResponseWithStatus(w, http.StatusNotFound, map[string]interface{}{
				"errorCode": constants.ErrorCodeNotFound,
				"context":   "file-system",
				"errors":    []string{"Path not found"},
			})
			return
		}

		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeFileReadError,
			"context":   "file-system",
			"errors":    []string{"Cannot read file: " + err.Error()},
		})
		return
	}

	if info.IsDir() {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusBadRequest, map[string]interface{}{
			"errorCode": constants.ErrorCodePathIsDirectory,
			"context":   "file-system",
			"errors":    []string{"Path is a directory, not a file"},
		})
		return
	}

	fileID := utils.GenerateMD5Hash(cleanPath)
	summary, err := s.internalDB.GetRevisionSummary(fileID)
	if err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeInternalServerError,
			"context":   "file-system",
			"errors":    []string{"Failed to get revision summary: " + err.Error()},
		})
		return
	}

	_ = utils.WriteJSONResponse(w, summary)
}

func (s *Server) releaseFileLock(lockPath string) {
	if lockPath != "" {
		if err := os.Remove(lockPath); err != nil {
			s.log.Error("Failed to remove lock file", logger.Field{Key: "error", Value: err})
		}
	}
}

type FileNode struct {
	ID            string            `json:"id"`
	Name          string            `json:"name"`
	Kind          string            `json:"kind"`
	Depth         int               `json:"depth"`
	LastModified  *time.Time        `json:"last_modified,omitempty"`
	Permissions   string            `json:"permissions,omitempty"`
	FileSize      int64             `json:"file_size,omitempty"`
	FileExtension string            `json:"file_extension,omitempty"`
	FileType      services.FileType `json:"file_type,omitempty"`
	MimeType      string            `json:"mime_type,omitempty"`
	IsEditable    bool              `json:"is_editable"`
	IsViewable    bool              `json:"is_viewable"`
	APIEndpoint   string            `json:"api_endpoint,omitempty"`
	Children      []*FileNode       `json:"children"`
}

type FileTreeResponse struct {
	OS       string    `json:"os"`
	FileTree *FileNode `json:"file_tree"`
}

type NPCFileAPIData struct {
	Name                string               `json:"name" validate:"required"`
	Id                  *uint16              `json:"id" validate:"required"`
	RespawnRate         *uint16              `json:"respawn_rate" validate:"required"`
	AttackTypeInfo      *byte                `json:"attack_type_info" validate:"required"`
	TargetSelectionInfo *byte                `json:"target_selection_info" validate:"required"`
	Defense             *byte                `json:"defense" validate:"required"`
	AdditionalDefense   *byte                `json:"additional_defense" validate:"required"`
	Attacks             []services.NPCAttack `json:"attacks" validate:"required,len=3"`
	AttackSpeedLow      *uint16              `json:"attack_speed_low" validate:"required"`
	AttackSpeedHigh     *uint16              `json:"attack_speed_high" validate:"required"`
	MovementSpeed       *uint32              `json:"movement_speed" validate:"required"`
	Level               *byte                `json:"level" validate:"required"`
	PlayerExp           *uint16              `json:"player_exp" validate:"required"`
	Appearance          *byte                `json:"appearance" validate:"required"`
	HP                  *uint32              `json:"hp" validate:"required"`
	BlueAttackDefense   *uint16              `json:"blue_attack_defense" validate:"required"`
	RedAttackDefense    *uint16              `json:"red_attack_defense" validate:"required"`
	GreyAttackDefense   *uint16              `json:"grey_attack_defense" validate:"required"`
	MercenaryExp        *uint16              `json:"mercenary_exp" validate:"required"`
}

type TextFileAPIData struct {
	Content string `json:"content" validate:"required"`
}

type SpawnFileAPIData struct {
	Spawns []NPCSpawnAPIData `json:"spawns" validate:"required"`
}

type NPCSpawnAPIData struct {
	Id          *uint16 `json:"id" validate:"required"`
	X           *byte   `json:"x" validate:"required"`
	Y           *byte   `json:"y" validate:"required"`
	Unknown1    *uint16 `json:"unknown1" validate:"required"`
	Orientation *byte   `json:"orientation" validate:"required"`
	SpwanStep   *byte   `json:"spwan_step" validate:"required"`
}

type fileUpdateContext struct {
	userID    int64
	cleanPath string
	info      os.FileInfo
	fileID    string
}
