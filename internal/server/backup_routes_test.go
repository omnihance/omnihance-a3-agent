package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/omnihance/omnihance-a3-agent/internal/config"
	"github.com/omnihance/omnihance-a3-agent/internal/constants"
	"github.com/omnihance/omnihance-a3-agent/internal/db"
	"github.com/omnihance/omnihance-a3-agent/internal/services"
	"github.com/stretchr/testify/require"
)

func TestCreateBackupRunFileDownloadLinkUsesSharedDownloadLink(t *testing.T) {
	server, backupService, fixture := newBackupDownloadTestServer(t, db.BackupRunStatusSucceeded, true)
	backupService.EXPECT().GetRunDetails(fixture.runID).Return(backupDownloadRunDetails(fixture, db.BackupRunStatusSucceeded), nil)

	req := backupRunFileRequest(http.MethodPost, "/download-link", constants.RoleAdmin, fixture)
	rr := httptest.NewRecorder()

	server.handleCreateBackupRunFileDownloadLink(rr, req)

	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())
	var response DownloadLinkResponse
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &response))
	require.True(t, strings.HasPrefix(response.DownloadURL, "/api/file-tree/download/"))

	publicID, _, err := server.verifyDownloadToken(strings.TrimPrefix(response.DownloadURL, "/api/file-tree/download/"))
	require.NoError(t, err)
	link, err := server.internalDB.GetFileDownloadLinkByPublicID(publicID)
	require.NoError(t, err)
	require.Equal(t, db.FileDownloadSourceBackup, link.SourceType)
	require.Equal(t, fixture.runID, *link.BackupRunID)
	require.Equal(t, fixture.fileID, *link.BackupFileID)
}

func TestCreateBackupRunFileDownloadLinkRejectsFailedRunAndWrongFile(t *testing.T) {
	server, backupService, fixture := newBackupDownloadTestServer(t, db.BackupRunStatusFailed, true)
	backupService.EXPECT().GetRunDetails(fixture.runID).Return(backupDownloadRunDetails(fixture, db.BackupRunStatusFailed), nil)

	req := backupRunFileRequest(http.MethodPost, "/download-link", constants.RoleAdmin, fixture)
	rr := httptest.NewRecorder()
	server.handleCreateBackupRunFileDownloadLink(rr, req)
	require.Equal(t, http.StatusBadRequest, rr.Code)

	server, backupService, fixture = newBackupDownloadTestServer(t, db.BackupRunStatusRunning, true)
	backupService.EXPECT().GetRunDetails(fixture.runID).Return(backupDownloadRunDetails(fixture, db.BackupRunStatusRunning), nil)

	req = backupRunFileRequest(http.MethodPost, "/download-link", constants.RoleAdmin, fixture)
	rr = httptest.NewRecorder()
	server.handleCreateBackupRunFileDownloadLink(rr, req)
	require.Equal(t, http.StatusBadRequest, rr.Code)

	server, backupService, fixture = newBackupDownloadTestServer(t, db.BackupRunStatusSucceeded, true)
	backupService.EXPECT().GetRunDetails(fixture.runID).Return(&services.BackupRunDetails{
		Run: db.BackupRun{ID: fixture.runID, Status: db.BackupRunStatusSucceeded},
		Files: []db.BackupRunFile{{
			ID:       fixture.fileID,
			RunID:    fixture.runID + 1,
			ItemName: "server",
			FilePath: fixture.filePath,
			FileSize: 4,
		}},
	}, nil)

	req = backupRunFileRequest(http.MethodPost, "/download-link", constants.RoleAdmin, fixture)
	rr = httptest.NewRecorder()
	server.handleCreateBackupRunFileDownloadLink(rr, req)
	require.Equal(t, http.StatusNotFound, rr.Code)
}

func TestGetBackupRunMarksDownloadAvailability(t *testing.T) {
	server, backupService, fixture := newBackupDownloadTestServer(t, db.BackupRunStatusSucceeded, true)
	missingPath := filepath.Join(t.TempDir(), "missing.zip")
	backupService.EXPECT().GetRunDetails(fixture.runID).Return(&services.BackupRunDetails{
		Run: db.BackupRun{ID: fixture.runID, Status: db.BackupRunStatusSucceeded},
		Files: []db.BackupRunFile{
			{ID: fixture.fileID, RunID: fixture.runID, ItemName: "available", FilePath: fixture.filePath, FileSize: 4, CreatedAt: time.Now()},
			{ID: fixture.fileID + 1, RunID: fixture.runID, ItemName: "missing", FilePath: missingPath, FileSize: 4, CreatedAt: time.Now()},
		},
	}, nil)

	req := backupDownloadRequest(http.MethodGet, "/api/backups/runs/"+strconv.FormatInt(fixture.runID, 10), constants.RoleAdmin)
	req = withURLParam(req, "run_id", strconv.FormatInt(fixture.runID, 10))
	rr := httptest.NewRecorder()
	server.handleGetBackupRun(rr, req)

	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())
	var response BackupRunDetailsResponse
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &response))
	require.True(t, response.Files[0].DownloadAvailable)
	require.False(t, response.Files[1].DownloadAvailable)
}

func TestOldBackupDownloadRouteRedirectsThroughTempLink(t *testing.T) {
	server, backupService, fixture := newBackupDownloadTestServer(t, db.BackupRunStatusSucceeded, true)
	backupService.EXPECT().GetRunDetails(fixture.runID).Return(backupDownloadRunDetails(fixture, db.BackupRunStatusSucceeded), nil)

	req := backupRunFileRequest(http.MethodGet, "/download", constants.RoleAdmin, fixture)
	rr := httptest.NewRecorder()
	server.handleDownloadBackupRunFile(rr, req)

	require.Equal(t, http.StatusFound, rr.Code, rr.Body.String())
	require.True(t, strings.HasPrefix(rr.Header().Get("Location"), "/api/file-tree/download/"))
}

func newBackupDownloadTestServer(t *testing.T, status string, createFile bool) (*Server, *services.MockBackupService, backupDownloadFixture) {
	t.Helper()

	internalDB := newTestInternalDB(t)
	_, err := internalDB.CreateUser("backup-download-admin@example.com", "password", constants.RoleAdmin, nil)
	require.NoError(t, err)
	backupService := services.NewMockBackupService(t)
	dir := t.TempDir()
	filePath := filepath.Join(dir, "backup.zip")
	if createFile {
		require.NoError(t, os.WriteFile(filePath, []byte("data"), 0600))
	}

	sourcePath := filePath
	job, err := internalDB.CreateBackupJob(db.BackupJobPayload{
		JobType:              db.BackupJobTypeFile,
		Name:                 "Backup",
		Status:               db.BackupJobStatusActive,
		DestinationDirectory: dir,
		SourcePath:           &sourcePath,
	}, nil)
	require.NoError(t, err)
	run, err := internalDB.CreateBackupRun(job.ID, db.BackupRunTriggerManual, db.BackupJobStatusActive, nil)
	require.NoError(t, err)
	require.NoError(t, internalDB.FinishBackupRun(run.ID, job.ID, status, db.BackupJobStatusActive, nil, nil))
	file, err := internalDB.CreateBackupRunFile(run.ID, "server", filePath, 4)
	require.NoError(t, err)

	return &Server{
			cfg:           &config.EnvVars{CookieSecret: "test-secret"},
			internalDB:    internalDB,
			fileEditor:    services.NewFileEditorService(nil),
			backupService: backupService,
		}, backupService, backupDownloadFixture{
			runID:    run.ID,
			fileID:   file.ID,
			filePath: filePath,
		}
}

func backupDownloadRunDetails(fixture backupDownloadFixture, status string) *services.BackupRunDetails {
	return &services.BackupRunDetails{
		Run: db.BackupRun{ID: fixture.runID, Status: status},
		Files: []db.BackupRunFile{{
			ID:       fixture.fileID,
			RunID:    fixture.runID,
			ItemName: "server",
			FilePath: fixture.filePath,
			FileSize: 4,
		}},
	}
}

func backupRunFileRequest(method string, suffix string, role string, fixture backupDownloadFixture) *http.Request {
	runID := strconv.FormatInt(fixture.runID, 10)
	fileID := strconv.FormatInt(fixture.fileID, 10)
	req := backupDownloadRequest(method, "/api/backups/runs/"+runID+"/files/"+fileID+suffix, role)
	return withBackupRunFileParams(req, runID, fileID)
}

func backupDownloadRequest(method string, target string, role string) *http.Request {
	return downloadRequest(method, target, nil, role, 1)
}

func withBackupRunFileParams(req *http.Request, runID string, fileID string) *http.Request {
	routeContext := chi.NewRouteContext()
	routeContext.URLParams.Add("run_id", runID)
	routeContext.URLParams.Add("file_id", fileID)
	return req.WithContext(contextWithChiRoute(req, routeContext))
}

func contextWithChiRoute(req *http.Request, routeContext *chi.Context) context.Context {
	return context.WithValue(req.Context(), chi.RouteCtxKey, routeContext)
}

type backupDownloadFixture struct {
	runID    int64
	fileID   int64
	filePath string
}
