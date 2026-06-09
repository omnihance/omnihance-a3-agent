package server

import (
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/omnihance/omnihance-a3-agent/internal/constants"
	"github.com/omnihance/omnihance-a3-agent/internal/db"
	"github.com/omnihance/omnihance-a3-agent/internal/permissions"
	"github.com/omnihance/omnihance-a3-agent/internal/utils"
)

const (
	fileDownloadLinkLifetime = 24 * time.Hour
	fileDownloadErrorContext = "file-system"
)

func (s *Server) handleCreateFileDownloadLink(w http.ResponseWriter, r *http.Request) {
	if !s.requireUserPermission(w, r, permissions.ActionDownloadFiles) {
		return
	}

	pathParam := r.URL.Query().Get("path")
	if pathParam == "" {
		writeFileDownloadError(w, http.StatusBadRequest, constants.ErrorCodeBadRequest, fileDownloadErrorContext, "Path parameter is required")
		return
	}

	response, err := s.createDownloadLinkForPath(r, filepath.Clean(pathParam), downloadLinkSource{
		sourceType: db.FileDownloadSourceFileBrowser,
	})
	if err != nil {
		writeDownloadLinkError(w, fileDownloadErrorContext, err)
		return
	}

	_ = utils.WriteJSONResponse(w, response)
}

func (s *Server) handleDownloadLinkedFile(w http.ResponseWriter, r *http.Request) {
	if !s.requireUserPermission(w, r, permissions.ActionDownloadFiles) {
		return
	}

	userID, ok := utils.GetUserIdFromContext(r.Context())
	if !ok {
		writeFileDownloadError(w, http.StatusUnauthorized, constants.ErrorCodeUnauthorized, fileDownloadErrorContext, "User not found in context")
		return
	}

	token := chi.URLParam(r, "token")
	publicID, tokenExpiresAt, err := s.verifyDownloadToken(token)
	if err != nil {
		writeFileDownloadError(w, http.StatusForbidden, constants.ErrorCodeForbidden, fileDownloadErrorContext, "Download link is invalid")
		return
	}

	now := time.Now()
	if !tokenExpiresAt.After(now) {
		writeFileDownloadError(w, http.StatusGone, constants.ErrorCodeBadRequest, fileDownloadErrorContext, "Download link has expired")
		return
	}

	link, err := s.internalDB.GetFileDownloadLinkByPublicID(publicID)
	if err != nil {
		if !errors.Is(err, constants.ErrNotFound) {
			writeFileDownloadError(w, http.StatusInternalServerError, constants.ErrorCodeInternalServerError, fileDownloadErrorContext, "Failed to load download link")
			return
		}

		writeFileDownloadError(w, http.StatusNotFound, constants.ErrorCodeNotFound, fileDownloadErrorContext, "Download link not found")
		return
	}

	if link.UserID != userID {
		writeFileDownloadError(w, http.StatusForbidden, constants.ErrorCodeForbidden, fileDownloadErrorContext, "Download link belongs to another user")
		return
	}

	if link.ExpiresAt.Unix() != tokenExpiresAt.Unix() || !link.ExpiresAt.After(now) {
		writeFileDownloadError(w, http.StatusGone, constants.ErrorCodeBadRequest, fileDownloadErrorContext, "Download link has expired")
		return
	}

	file, fingerprint, err := s.openFileDownload(link.OriginalPath)
	if err != nil {
		writeDownloadLinkError(w, fileDownloadErrorContext, err)
		return
	}
	defer func() {
		_ = file.Close()
	}()

	if !downloadFingerprintMatchesLink(fingerprint, link) {
		writeFileDownloadError(w, http.StatusGone, constants.ErrorCodeBadRequest, fileDownloadErrorContext, "Download link is no longer valid because the file changed")
		return
	}

	userAgent := optionalString(r.UserAgent())
	ipAddress := optionalString(downloadRequestIP(r))
	if err := s.internalDB.RecordFileDownload(link, userID, userAgent, ipAddress); err != nil {
		writeFileDownloadError(w, http.StatusInternalServerError, constants.ErrorCodeInternalServerError, fileDownloadErrorContext, "Failed to record file download")
		return
	}

	contentDisposition := mime.FormatMediaType("attachment", map[string]string{"filename": link.FileName})
	if contentDisposition == "" {
		contentDisposition = `attachment; filename="` + filepath.Base(link.FileName) + `"`
	}

	w.Header().Set("Content-Disposition", contentDisposition)
	http.ServeContent(w, r, link.FileName, fingerprint.modTime, file)
}

func (s *Server) createDownloadLinkForPath(r *http.Request, path string, source downloadLinkSource) (*DownloadLinkResponse, error) {
	userID, ok := utils.GetUserIdFromContext(r.Context())
	if !ok {
		return nil, newDownloadLinkError(http.StatusUnauthorized, constants.ErrorCodeUnauthorized, "User not found in context")
	}

	fingerprint, err := s.buildFileDownloadFingerprint(path)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	payload := db.FileDownloadLinkPayload{
		UserID:         userID,
		FileID:         fingerprint.fileID,
		SourceType:     source.sourceType,
		BackupRunID:    source.backupRunID,
		BackupFileID:   source.backupFileID,
		OriginalPath:   fingerprint.path,
		FileName:       fingerprint.fileName,
		FileSize:       fingerprint.fileSize,
		FileHash:       fingerprint.fileHash,
		FileModifiedAt: fingerprint.fileModifiedAt,
	}

	link, err := s.internalDB.GetReusableFileDownloadLink(payload, now)
	if err != nil {
		return nil, newDownloadLinkError(http.StatusInternalServerError, constants.ErrorCodeInternalServerError, "Failed to get download link")
	}

	reused := link != nil
	if link == nil {
		payload.PublicID = uuid.NewString()
		payload.ExpiresAt = now.Add(fileDownloadLinkLifetime)
		link, err = s.internalDB.CreateFileDownloadLink(payload)
		if err != nil {
			return nil, newDownloadLinkError(http.StatusInternalServerError, constants.ErrorCodeInternalServerError, "Failed to create download link")
		}
	}

	downloadURL, err := s.downloadURLForLink(link)
	if err != nil {
		return nil, newDownloadLinkError(http.StatusInternalServerError, constants.ErrorCodeInternalServerError, "Failed to create download URL")
	}

	return &DownloadLinkResponse{
		DownloadURL:   downloadURL,
		ExpiresAt:     link.ExpiresAt,
		Reused:        reused,
		DownloadCount: link.DownloadCount,
	}, nil
}

func (s *Server) buildFileDownloadFingerprint(path string) (*downloadFingerprint, error) {
	file, fingerprint, err := s.openFileDownload(path)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = file.Close()
	}()

	return fingerprint, nil
}

func (s *Server) openFileDownload(path string) (*os.File, *downloadFingerprint, error) {
	cleanPath := filepath.Clean(path)
	info, err := s.fileEditor.Stat(cleanPath)
	if err != nil {
		if s.fileEditor.IsNotExist(err) {
			return nil, nil, newDownloadLinkError(http.StatusNotFound, constants.ErrorCodeNotFound, "Path not found")
		}

		return nil, nil, newDownloadLinkError(http.StatusInternalServerError, constants.ErrorCodeFileReadError, "Cannot read file: "+err.Error())
	}

	if info.IsDir() {
		return nil, nil, newDownloadLinkError(http.StatusBadRequest, constants.ErrorCodePathIsDirectory, "Path is a directory, not a file")
	}

	file, err := s.fileEditor.OpenFile(cleanPath, os.O_RDONLY, 0)
	if err != nil {
		if s.fileEditor.IsNotExist(err) {
			return nil, nil, newDownloadLinkError(http.StatusNotFound, constants.ErrorCodeNotFound, "Path not found")
		}

		return nil, nil, newDownloadLinkError(http.StatusInternalServerError, constants.ErrorCodeFileReadError, "Cannot read file: "+err.Error())
	}

	success := false
	defer func() {
		if !success {
			_ = file.Close()
		}
	}()

	info, err = file.Stat()
	if err != nil {
		return nil, nil, newDownloadLinkError(http.StatusInternalServerError, constants.ErrorCodeFileReadError, "Cannot read file: "+err.Error())
	}

	if info.IsDir() {
		return nil, nil, newDownloadLinkError(http.StatusBadRequest, constants.ErrorCodePathIsDirectory, "Path is a directory, not a file")
	}

	fileHash, err := hashFile(file)
	if err != nil {
		return nil, nil, newDownloadLinkError(http.StatusInternalServerError, constants.ErrorCodeFileReadError, "Failed to hash file: "+err.Error())
	}

	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return nil, nil, newDownloadLinkError(http.StatusInternalServerError, constants.ErrorCodeFileReadError, "Failed to prepare file download: "+err.Error())
	}

	success = true
	return file, &downloadFingerprint{
		path:           cleanPath,
		fileID:         utils.GenerateMD5Hash(cleanPath),
		fileName:       filepath.Base(cleanPath),
		fileSize:       info.Size(),
		fileHash:       fileHash,
		fileModifiedAt: info.ModTime().UnixNano(),
		modTime:        info.ModTime(),
	}, nil
}

func hashFile(file *os.File) (string, error) {
	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

func (s *Server) downloadURLForLink(link *db.FileDownloadLink) (string, error) {
	token, err := s.signDownloadToken(link.PublicID, link.ExpiresAt)
	if err != nil {
		return "", err
	}

	return "/api/file-tree/download/" + token, nil
}

func (s *Server) signDownloadToken(publicID string, expiresAt time.Time) (string, error) {
	secret, err := s.downloadTokenSecret()
	if err != nil {
		return "", err
	}

	expiresUnix := strconv.FormatInt(expiresAt.Unix(), 10)
	message := publicID + "." + expiresUnix
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(message))
	signature := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return message + "." + signature, nil
}

func (s *Server) verifyDownloadToken(token string) (string, time.Time, error) {
	secret, err := s.downloadTokenSecret()
	if err != nil {
		return "", time.Time{}, err
	}

	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return "", time.Time{}, errors.New("invalid token shape")
	}

	expiresUnix, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return "", time.Time{}, err
	}

	message := parts[0] + "." + parts[1]
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(message))
	expected := mac.Sum(nil)
	actual, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return "", time.Time{}, err
	}

	if !hmac.Equal(actual, expected) {
		return "", time.Time{}, errors.New("invalid token signature")
	}

	return parts[0], time.Unix(expiresUnix, 0), nil
}

func (s *Server) downloadTokenSecret() (string, error) {
	if s.cfg == nil || strings.TrimSpace(s.cfg.CookieSecret) == "" {
		return "", errors.New("download token secret is not configured")
	}

	return s.cfg.CookieSecret, nil
}

func downloadFingerprintMatchesLink(fingerprint *downloadFingerprint, link *db.FileDownloadLink) bool {
	return fingerprint.fileID == link.FileID &&
		fingerprint.path == link.OriginalPath &&
		fingerprint.fileSize == link.FileSize &&
		fingerprint.fileHash == link.FileHash &&
		fingerprint.fileModifiedAt == link.FileModifiedAt
}

func writeDownloadLinkError(w http.ResponseWriter, context string, err error) {
	var downloadErr *downloadLinkError
	if errors.As(err, &downloadErr) {
		writeFileDownloadError(w, downloadErr.status, downloadErr.errorCode, context, downloadErr.message)
		return
	}

	writeFileDownloadError(w, http.StatusInternalServerError, constants.ErrorCodeInternalServerError, context, err.Error())
}

func writeFileDownloadError(w http.ResponseWriter, status int, errorCode string, context string, message string) {
	_ = utils.WriteJSONResponseWithStatus(w, status, map[string]interface{}{
		"errorCode": errorCode,
		"context":   context,
		"errors":    []string{message},
	})
}

func newDownloadLinkError(status int, errorCode string, message string) error {
	return &downloadLinkError{
		status:    status,
		errorCode: errorCode,
		message:   message,
	}
}

func downloadRequestIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}

	return r.RemoteAddr
}

func optionalString(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}

	return &value
}

type DownloadLinkResponse struct {
	DownloadURL   string    `json:"download_url"`
	ExpiresAt     time.Time `json:"expires_at"`
	Reused        bool      `json:"reused"`
	DownloadCount int64     `json:"download_count"`
}

type downloadLinkSource struct {
	sourceType   string
	backupRunID  *int64
	backupFileID *int64
}

type downloadFingerprint struct {
	path           string
	fileID         string
	fileName       string
	fileSize       int64
	fileHash       string
	fileModifiedAt int64
	modTime        time.Time
}

type downloadLinkError struct {
	status    int
	errorCode string
	message   string
}

func (e *downloadLinkError) Error() string {
	return fmt.Sprintf("%s: %s", e.errorCode, e.message)
}
