package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"path/filepath"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/omnihance/omnihance-a3-agent/internal/constants"
	"github.com/omnihance/omnihance-a3-agent/internal/db"
	"github.com/omnihance/omnihance-a3-agent/internal/mw"
	"github.com/omnihance/omnihance-a3-agent/internal/permissions"
	"github.com/omnihance/omnihance-a3-agent/internal/services"
	"github.com/omnihance/omnihance-a3-agent/internal/utils"
)

const backupsErrorContext = "backups"

const (
	defaultBackupRunPageSize = 10
	maxBackupRunPageSize     = 100
)

func (s *Server) InitializeBackupRoutes(r *chi.Mux) {
	r.Route("/api/backups", func(r chi.Router) {
		r.Use(mw.CheckCookie(s.internalDB, s.cfg.CookieSecret))
		r.Get("/jobs", s.handleListBackupJobs)
		r.Post("/jobs", s.handleCreateBackupJob)
		r.Get("/jobs/{id}", s.handleGetBackupJob)
		r.Put("/jobs/{id}", s.handleUpdateBackupJob)
		r.Delete("/jobs/{id}", s.handleDeleteBackupJob)
		r.Post("/jobs/{id}/run", s.handleRunBackupJob)
		r.Post("/jobs/{id}/cancel", s.handleCancelBackupJob)
		r.Get("/jobs/{id}/runs", s.handleListBackupRuns)
		r.Get("/runs/{run_id}", s.handleGetBackupRun)
		r.Get("/runs/{run_id}/files/{file_id}/download", s.handleDownloadBackupRunFile)
		r.Get("/path-search", s.handleBackupPathSearch)
		r.Get("/defaults/sql-server", s.handleBackupSQLServerDefaults)
	})
}

func (s *Server) handleListBackupJobs(w http.ResponseWriter, r *http.Request) {
	if !s.requireUserPermission(w, r, permissions.ActionManageServer) {
		return
	}

	jobs, err := s.backupService.GetJobs()
	if err != nil {
		writeBackupError(w, http.StatusInternalServerError, constants.ErrorCodeInternalServerError, err.Error())
		return
	}

	_ = utils.WriteJSONResponse(w, map[string]interface{}{"jobs": jobs})
}

func (s *Server) handleCreateBackupJob(w http.ResponseWriter, r *http.Request) {
	if !s.requireUserPermission(w, r, permissions.ActionManageServer) {
		return
	}

	var req BackupJobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBackupError(w, http.StatusBadRequest, constants.ErrorCodeBadRequest, "Invalid request body")
		return
	}

	job, err := s.backupService.CreateJob(r.Context(), req.toPayload(), backupUserID(r))
	if err != nil {
		writeBackupServiceError(w, err)
		return
	}

	_ = utils.WriteJSONResponse(w, job)
}

func (s *Server) handleGetBackupJob(w http.ResponseWriter, r *http.Request) {
	if !s.requireUserPermission(w, r, permissions.ActionManageServer) {
		return
	}

	id, ok := backupIDParam(w, r, "id", "Invalid backup job ID")
	if !ok {
		return
	}

	job, err := s.backupService.GetJob(id)
	if err != nil {
		writeBackupServiceError(w, err)
		return
	}

	_ = utils.WriteJSONResponse(w, job)
}

func (s *Server) handleUpdateBackupJob(w http.ResponseWriter, r *http.Request) {
	if !s.requireUserPermission(w, r, permissions.ActionManageServer) {
		return
	}

	id, ok := backupIDParam(w, r, "id", "Invalid backup job ID")
	if !ok {
		return
	}

	var req BackupJobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBackupError(w, http.StatusBadRequest, constants.ErrorCodeBadRequest, "Invalid request body")
		return
	}

	job, err := s.backupService.UpdateJob(r.Context(), id, req.toPayload(), backupUserID(r))
	if err != nil {
		writeBackupServiceError(w, err)
		return
	}

	_ = utils.WriteJSONResponse(w, job)
}

func (s *Server) handleDeleteBackupJob(w http.ResponseWriter, r *http.Request) {
	if !s.requireUserPermission(w, r, permissions.ActionManageServer) {
		return
	}

	id, ok := backupIDParam(w, r, "id", "Invalid backup job ID")
	if !ok {
		return
	}

	if err := s.backupService.DeleteJob(r.Context(), id, backupUserID(r)); err != nil {
		writeBackupServiceError(w, err)
		return
	}

	_ = utils.WriteJSONResponse(w, map[string]interface{}{"message": "Backup job deleted successfully"})
}

func (s *Server) handleRunBackupJob(w http.ResponseWriter, r *http.Request) {
	if !s.requireUserPermission(w, r, permissions.ActionManageServer) {
		return
	}

	id, ok := backupIDParam(w, r, "id", "Invalid backup job ID")
	if !ok {
		return
	}

	run, err := s.backupService.RunJob(r.Context(), id, db.BackupRunTriggerManual, backupUserID(r))
	if err != nil {
		writeBackupServiceError(w, err)
		return
	}

	_ = utils.WriteJSONResponse(w, run)
}

func (s *Server) handleCancelBackupJob(w http.ResponseWriter, r *http.Request) {
	if !s.requireUserPermission(w, r, permissions.ActionManageServer) {
		return
	}

	id, ok := backupIDParam(w, r, "id", "Invalid backup job ID")
	if !ok {
		return
	}

	run, err := s.backupService.CancelJob(r.Context(), id)
	if err != nil {
		writeBackupServiceError(w, err)
		return
	}

	_ = utils.WriteJSONResponse(w, run)
}

func (s *Server) handleListBackupRuns(w http.ResponseWriter, r *http.Request) {
	if !s.requireUserPermission(w, r, permissions.ActionManageServer) {
		return
	}

	id, ok := backupIDParam(w, r, "id", "Invalid backup job ID")
	if !ok {
		return
	}

	page, pageSize := backupPaginationParams(r)
	runs, totalCount, err := s.backupService.GetRuns(id, page, pageSize)
	if err != nil {
		writeBackupServiceError(w, err)
		return
	}

	_ = utils.WriteJSONResponse(w, BackupRunsResponse{
		Runs: runs,
		Pagination: PaginationInfo{
			TotalCount: totalCount,
			Page:       page,
			PageSize:   pageSize,
		},
	})
}

func (s *Server) handleGetBackupRun(w http.ResponseWriter, r *http.Request) {
	if !s.requireUserPermission(w, r, permissions.ActionManageServer) {
		return
	}

	runID, ok := backupIDParam(w, r, "run_id", "Invalid backup run ID")
	if !ok {
		return
	}

	run, err := s.backupService.GetRunDetails(runID)
	if err != nil {
		writeBackupServiceError(w, err)
		return
	}

	_ = utils.WriteJSONResponse(w, run)
}

func (s *Server) handleDownloadBackupRunFile(w http.ResponseWriter, r *http.Request) {
	if !s.requireUserPermission(w, r, permissions.ActionManageServer) {
		return
	}

	runID, ok := backupIDParam(w, r, "run_id", "Invalid backup run ID")
	if !ok {
		return
	}

	fileID, ok := backupIDParam(w, r, "file_id", "Invalid backup file ID")
	if !ok {
		return
	}

	file, err := s.backupService.GetRunFile(fileID)
	if err != nil {
		writeBackupServiceError(w, err)
		return
	}

	if file.RunID != runID {
		writeBackupError(w, http.StatusNotFound, constants.ErrorCodeNotFound, "Backup file not found")
		return
	}

	info, err := s.fileEditor.Stat(file.FilePath)
	if err != nil {
		writeBackupError(w, http.StatusNotFound, constants.ErrorCodeNotFound, "Backup file is missing")
		return
	}

	if info.IsDir() {
		writeBackupError(w, http.StatusBadRequest, constants.ErrorCodeBadRequest, "Backup output is a directory")
		return
	}

	w.Header().Set("Content-Disposition", `attachment; filename="`+filepath.Base(file.FilePath)+`"`)
	http.ServeFile(w, r, file.FilePath)
}

func (s *Server) handleBackupPathSearch(w http.ResponseWriter, r *http.Request) {
	if !s.requireUserPermission(w, r, permissions.ActionManageServer) {
		return
	}

	query := r.URL.Query().Get("query")
	kind := r.URL.Query().Get("kind")
	results, err := s.backupService.SearchPaths(query, kind)
	if err != nil {
		writeBackupServiceError(w, err)
		return
	}

	_ = utils.WriteJSONResponse(w, map[string]interface{}{"results": results})
}

func (s *Server) handleBackupSQLServerDefaults(w http.ResponseWriter, r *http.Request) {
	if !s.requireUserPermission(w, r, permissions.ActionManageServer) {
		return
	}

	_ = utils.WriteJSONResponse(w, s.backupService.GetSQLServerDefaults())
}

func writeBackupServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, services.ErrBackupNotFound):
		writeBackupError(w, http.StatusNotFound, constants.ErrorCodeNotFound, err.Error())
	case errors.Is(err, services.ErrBackupJobRunning):
		writeBackupError(w, http.StatusConflict, constants.ErrorCodeBadRequest, err.Error())
	case errors.Is(err, services.ErrBackupNoRunningJob):
		writeBackupError(w, http.StatusBadRequest, constants.ErrorCodeBadRequest, err.Error())
	case errors.Is(err, services.ErrBackupInvalid), errors.Is(err, services.ErrBackupRemoteSQLHost):
		writeBackupError(w, http.StatusBadRequest, constants.ErrorCodeBadRequest, err.Error())
	default:
		writeBackupError(w, http.StatusInternalServerError, constants.ErrorCodeInternalServerError, err.Error())
	}
}

func writeBackupError(w http.ResponseWriter, status int, errorCode string, message string) {
	_ = utils.WriteJSONResponseWithStatus(w, status, map[string]interface{}{
		"errorCode": errorCode,
		"context":   backupsErrorContext,
		"errors":    []string{message},
	})
}

func backupIDParam(w http.ResponseWriter, r *http.Request, name string, message string) (int64, bool) {
	id, err := strconv.ParseInt(chi.URLParam(r, name), 10, 64)
	if err != nil {
		writeBackupError(w, http.StatusBadRequest, constants.ErrorCodeBadRequest, message)
		return 0, false
	}

	return id, true
}

func backupUserID(r *http.Request) *int64 {
	userID, ok := utils.GetUserIdFromContext(r.Context())
	if !ok {
		return nil
	}

	return &userID
}

func backupPaginationParams(r *http.Request) (int, int) {
	page := 1
	pageSize := defaultBackupRunPageSize

	if value := r.URL.Query().Get("page"); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil && parsed >= 1 {
			page = parsed
		}
	}

	pageSizeValue := r.URL.Query().Get("pageSize")
	if pageSizeValue == "" {
		pageSizeValue = r.URL.Query().Get("page_size")
	}

	if pageSizeValue != "" {
		if parsed, err := strconv.Atoi(pageSizeValue); err == nil && parsed >= 1 && parsed <= maxBackupRunPageSize {
			pageSize = parsed
		}
	}

	return page, pageSize
}

type BackupRunsResponse struct {
	Runs       []db.BackupRun `json:"runs"`
	Pagination PaginationInfo `json:"pagination"`
}

type BackupJobRequest struct {
	JobType              string  `json:"job_type"`
	Name                 string  `json:"name"`
	Status               string  `json:"status"`
	CronExpression       *string `json:"cron_expression"`
	DestinationDirectory string  `json:"destination_directory"`
	ArchivePassword      *string `json:"archive_password"`
	SourcePath           *string `json:"source_path"`
	SQLHost              *string `json:"sql_host"`
	SQLPort              *int    `json:"sql_port"`
	SQLUsername          *string `json:"sql_username"`
	SQLPassword          *string `json:"sql_password"`
	SQLDatabaseNames     *string `json:"sql_database_names"`
}

func (r BackupJobRequest) toPayload() db.BackupJobPayload {
	return db.BackupJobPayload{
		JobType:              r.JobType,
		Name:                 r.Name,
		Status:               r.Status,
		CronExpression:       r.CronExpression,
		DestinationDirectory: r.DestinationDirectory,
		ArchivePassword:      r.ArchivePassword,
		SourcePath:           r.SourcePath,
		SQLHost:              r.SQLHost,
		SQLPort:              r.SQLPort,
		SQLUsername:          r.SQLUsername,
		SQLPassword:          r.SQLPassword,
		SQLDatabaseNames:     r.SQLDatabaseNames,
	}
}
