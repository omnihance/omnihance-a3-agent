package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/omnihance/omnihance-a3-agent/internal/config"
	"github.com/omnihance/omnihance-a3-agent/internal/constants"
	"github.com/omnihance/omnihance-a3-agent/internal/db"
	"github.com/omnihance/omnihance-a3-agent/internal/services"
	"github.com/omnihance/omnihance-a3-agent/internal/utils"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestCreateFileDownloadLinkReusesUnchangedFile(t *testing.T) {
	server := newFileDownloadTestServer(t)
	filePath := writeDownloadTestFile(t, t.TempDir(), "server.txt", []byte("server data"))

	first := createFileDownloadLinkForTest(t, server, filePath, constants.RoleAdmin, 1)
	require.False(t, first.Reused)
	require.Equal(t, int64(0), first.DownloadCount)
	require.True(t, strings.HasPrefix(first.DownloadURL, "/api/file-tree/download/"))

	second := createFileDownloadLinkForTest(t, server, filePath, constants.RoleAdmin, 1)
	require.True(t, second.Reused)
	require.Equal(t, first.DownloadURL, second.DownloadURL)
}

func TestCreateFileDownloadLinkRejectsViewerAndDirectory(t *testing.T) {
	server := newFileDownloadTestServer(t)
	filePath := writeDownloadTestFile(t, t.TempDir(), "server.txt", []byte("server data"))

	req := downloadRequest(http.MethodPost, "/api/file-tree/download-link?path="+url.QueryEscape(filePath), nil, constants.RoleUser, 1)
	rr := httptest.NewRecorder()
	server.handleCreateFileDownloadLink(rr, req)
	require.Equal(t, http.StatusForbidden, rr.Code)

	req = downloadRequest(http.MethodPost, "/api/file-tree/download-link?path="+url.QueryEscape(t.TempDir()), nil, constants.RoleAdmin, 1)
	rr = httptest.NewRecorder()
	server.handleCreateFileDownloadLink(rr, req)
	require.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestDownloadLinkedFileStreamsAttachmentAndTracksCount(t *testing.T) {
	server := newFileDownloadTestServer(t)
	fileContent := []byte("download body")
	filePath := writeDownloadTestFile(t, t.TempDir(), "server.txt", fileContent)
	link := createFileDownloadLinkForTest(t, server, filePath, constants.RoleAdmin, 1)

	rr := downloadLinkedFileForTest(t, server, link.DownloadURL, constants.RoleAdmin, 1)

	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())
	require.Equal(t, fileContent, rr.Body.Bytes())
	require.Contains(t, rr.Header().Get("Content-Disposition"), "attachment")
	require.Contains(t, rr.Header().Get("Content-Disposition"), "server.txt")

	publicID, _, err := server.verifyDownloadToken(strings.TrimPrefix(link.DownloadURL, "/api/file-tree/download/"))
	require.NoError(t, err)
	updatedLink, err := server.internalDB.GetFileDownloadLinkByPublicID(publicID)
	require.NoError(t, err)
	require.Equal(t, int64(1), updatedLink.DownloadCount)
}

func TestDownloadLinkedFileRejectsWrongUserExpiredChangedAndMissingFile(t *testing.T) {
	server := newFileDownloadTestServer(t)
	filePath := writeDownloadTestFile(t, t.TempDir(), "server.txt", []byte("server data"))
	link := createFileDownloadLinkForTest(t, server, filePath, constants.RoleAdmin, 1)

	rr := downloadLinkedFileForTest(t, server, link.DownloadURL, constants.RoleAdmin, 2)
	require.Equal(t, http.StatusForbidden, rr.Code)

	require.NoError(t, os.WriteFile(filePath, []byte("changed data"), 0600))
	rr = downloadLinkedFileForTest(t, server, link.DownloadURL, constants.RoleAdmin, 1)
	require.Equal(t, http.StatusGone, rr.Code)

	missingLink := createFileDownloadLinkForTest(t, server, filePath, constants.RoleAdmin, 1)
	require.NoError(t, os.Remove(filePath))
	rr = downloadLinkedFileForTest(t, server, missingLink.DownloadURL, constants.RoleAdmin, 1)
	require.Equal(t, http.StatusNotFound, rr.Code)

	expiredPath := writeDownloadTestFile(t, t.TempDir(), "expired.txt", []byte("expired data"))
	expiredLink := createExpiredDownloadLinkForTest(t, server, expiredPath)
	rr = downloadLinkedFileForTest(t, server, expiredLink, constants.RoleAdmin, 1)
	require.Equal(t, http.StatusGone, rr.Code)
}

func TestCreateDirectoryDownloadLinkReturnsInProgress(t *testing.T) {
	server := newFileDownloadTestServer(t)
	backupService := services.NewMockBackupService(t)
	server.backupService = backupService
	sourceDir := t.TempDir()
	runID := int64(77)

	backupService.EXPECT().
		PrepareDirectoryDownload(mock.Anything, filepath.Clean(sourceDir), mock.Anything).
		Return(&services.DirectoryDownloadResult{
			Status:  services.DirectoryDownloadStatusInProgress,
			Message: "This directory download is already in progress. Keep this page open; the download will start when ready.",
			JobID:   55,
			RunID:   runID,
		}, nil)

	req := downloadRequest(http.MethodPost, "/api/file-tree/directory-download-link?path="+url.QueryEscape(sourceDir), nil, constants.RoleAdmin, 1)
	rr := httptest.NewRecorder()
	server.handleCreateDirectoryDownloadLink(rr, req)

	require.Equal(t, http.StatusAccepted, rr.Code, rr.Body.String())
	var response DirectoryDownloadResponse
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &response))
	require.Equal(t, services.DirectoryDownloadStatusInProgress, response.Status)
	require.Equal(t, runID, response.RunID)
}

func TestCreateDirectoryDownloadLinkUsesSecureDownloadTableWhenReady(t *testing.T) {
	server := newFileDownloadTestServer(t)
	backupService := services.NewMockBackupService(t)
	server.backupService = backupService
	dir := t.TempDir()
	archivePath := writeDownloadTestFile(t, dir, "server.zip", []byte("zip data"))
	sourcePath := filepath.Join(dir, "source")

	job, err := server.internalDB.CreateBackupJob(db.BackupJobPayload{
		JobType:              db.BackupJobTypeFile,
		Name:                 "Directory download: source",
		Status:               db.BackupJobStatusActive,
		DestinationDirectory: dir,
		SourcePath:           &sourcePath,
	}, nil)
	require.NoError(t, err)
	run, err := server.internalDB.CreateBackupRun(job.ID, db.BackupRunTriggerDirectoryDownload, db.BackupJobStatusActive, nil)
	require.NoError(t, err)
	file, err := server.internalDB.CreateBackupRunFile(run.ID, "source", archivePath, 8)
	require.NoError(t, err)

	backupService.EXPECT().
		PrepareDirectoryDownload(mock.Anything, filepath.Clean(sourcePath), mock.Anything).
		Return(&services.DirectoryDownloadResult{
			Status:      services.DirectoryDownloadStatusReady,
			JobID:       job.ID,
			RunID:       run.ID,
			FileID:      &file.ID,
			ArchivePath: archivePath,
		}, nil)

	req := downloadRequest(http.MethodPost, "/api/file-tree/directory-download-link?path="+url.QueryEscape(sourcePath), nil, constants.RoleAdmin, 1)
	rr := httptest.NewRecorder()
	server.handleCreateDirectoryDownloadLink(rr, req)

	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())
	var response DirectoryDownloadResponse
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &response))
	require.Equal(t, services.DirectoryDownloadStatusReady, response.Status)
	require.True(t, strings.HasPrefix(response.DownloadURL, "/api/file-tree/download/"))

	publicID, _, err := server.verifyDownloadToken(strings.TrimPrefix(response.DownloadURL, "/api/file-tree/download/"))
	require.NoError(t, err)
	link, err := server.internalDB.GetFileDownloadLinkByPublicID(publicID)
	require.NoError(t, err)
	require.Equal(t, db.FileDownloadSourceDirectoryDownload, link.SourceType)
	require.Equal(t, run.ID, *link.BackupRunID)
	require.Equal(t, file.ID, *link.BackupFileID)
}

func newFileDownloadTestServer(t *testing.T) *Server {
	t.Helper()

	internalDB := newTestInternalDB(t)
	_, err := internalDB.CreateUser("download-admin@example.com", "password", constants.RoleAdmin, nil)
	require.NoError(t, err)
	_, err = internalDB.CreateUser("download-admin-2@example.com", "password", constants.RoleAdmin, nil)
	require.NoError(t, err)

	return &Server{
		cfg:        &config.EnvVars{CookieSecret: "test-secret"},
		internalDB: internalDB,
		fileEditor: services.NewFileEditorService(nil),
	}
}

func createFileDownloadLinkForTest(t *testing.T, server *Server, filePath string, role string, userID int64) DownloadLinkResponse {
	t.Helper()

	req := downloadRequest(http.MethodPost, "/api/file-tree/download-link?path="+url.QueryEscape(filePath), nil, role, userID)
	rr := httptest.NewRecorder()
	server.handleCreateFileDownloadLink(rr, req)
	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())

	var response DownloadLinkResponse
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &response))
	return response
}

func createExpiredDownloadLinkForTest(t *testing.T, server *Server, filePath string) string {
	t.Helper()

	fingerprint, err := server.buildFileDownloadFingerprint(filePath)
	require.NoError(t, err)
	link, err := server.internalDB.CreateFileDownloadLink(db.FileDownloadLinkPayload{
		PublicID:       "expired-link",
		UserID:         1,
		FileID:         fingerprint.fileID,
		SourceType:     db.FileDownloadSourceFileBrowser,
		OriginalPath:   fingerprint.path,
		FileName:       fingerprint.fileName,
		FileSize:       fingerprint.fileSize,
		FileHash:       fingerprint.fileHash,
		FileModifiedAt: fingerprint.fileModifiedAt,
		ExpiresAt:      time.Now().Add(-time.Hour),
	})
	require.NoError(t, err)

	downloadURL, err := server.downloadURLForLink(link)
	require.NoError(t, err)
	return downloadURL
}

func downloadLinkedFileForTest(t *testing.T, server *Server, downloadURL string, role string, userID int64) *httptest.ResponseRecorder {
	t.Helper()

	token := strings.TrimPrefix(downloadURL, "/api/file-tree/download/")
	req := downloadRequest(http.MethodGet, downloadURL, nil, role, userID)
	req = withURLParam(req, "token", token)
	rr := httptest.NewRecorder()
	server.handleDownloadLinkedFile(rr, req)
	return rr
}

func downloadRequest(method string, target string, body *bytes.Buffer, role string, userID int64) *http.Request {
	var reader *strings.Reader
	if body == nil {
		reader = strings.NewReader("")
	} else {
		reader = strings.NewReader(body.String())
	}

	req := httptest.NewRequest(method, target, reader)
	ctx := req.Context()
	ctx = utils.SetUserIdInContext(ctx, userID)
	ctx = utils.SetUserRolesInContext(ctx, []string{role})
	return req.WithContext(ctx)
}

func writeDownloadTestFile(t *testing.T, dir string, name string, content []byte) string {
	t.Helper()

	path := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(path, content, 0600))
	return path
}
