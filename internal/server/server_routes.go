package server

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/omnihance/omnihance-a3-agent/internal/constants"
	"github.com/omnihance/omnihance-a3-agent/internal/db"
	"github.com/omnihance/omnihance-a3-agent/internal/mw"
	"github.com/omnihance/omnihance-a3-agent/internal/permissions"
	"github.com/omnihance/omnihance-a3-agent/internal/utils"
)

func (s *Server) InitializeServerRoutes(r *chi.Mux) {
	r.Route("/api/server", func(r chi.Router) {
		r.Use(mw.CheckCookie(s.internalDB, s.cfg.CookieSecret))
		r.Get("/processes", s.handleGetServerProcesses)
		r.Post("/processes", s.handleCreateServerProcess)
		r.Get("/processes/{id}", s.handleGetServerProcess)
		r.Put("/processes/{id}", s.handleUpdateServerProcess)
		r.Delete("/processes/{id}", s.handleDeleteServerProcess)
		r.Post("/processes/reorder", s.handleReorderServerProcesses)
		r.Post("/start", s.handleStartFullServer)
		r.Post("/stop", s.handleStopFullServer)
		r.Post("/processes/{id}/start", s.handleStartProcess)
		r.Post("/processes/{id}/stop", s.handleStopProcess)
		r.Get("/processes/{id}/status", s.handleGetProcessStatus)
	})
}

func (s *Server) handleGetServerProcesses(w http.ResponseWriter, r *http.Request) {
	processes, err := s.internalDB.GetServerProcesses()
	if err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeInternalServerError,
			"context":   "server",
			"errors":    []string{err.Error()},
		})
		return
	}

	_ = utils.WriteJSONResponse(w, map[string]interface{}{
		"processes": processes,
	})
}

func (s *Server) handleCreateServerProcess(w http.ResponseWriter, r *http.Request) {
	if !s.requireUserPermission(w, r, permissions.ActionManageServer) {
		return
	}

	var req CreateServerProcessRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusBadRequest, map[string]interface{}{
			"errorCode": constants.ErrorCodeBadRequest,
			"context":   "server",
			"errors":    []string{"Invalid request body"},
		})
		return
	}

	validate := validator.New()
	if err := validate.Struct(req); err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusBadRequest, map[string]interface{}{
			"errorCode": constants.ErrorCodeBadRequest,
			"context":   "server",
			"errors":    []string{err.Error()},
		})
		return
	}

	cleanPath := filepath.Clean(req.Path)
	if !s.validateServerProcessPath(w, cleanPath, nil) {
		return
	}

	maxOrder, err := s.internalDB.GetMaxSequenceOrder()
	if err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeInternalServerError,
			"context":   "server",
			"errors":    []string{err.Error()},
		})
		return
	}

	process, err := s.internalDB.CreateServerProcess(req.Name, cleanPath, req.Port, maxOrder+1)
	if err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeInternalServerError,
			"context":   "server",
			"errors":    []string{err.Error()},
		})
		return
	}

	_ = utils.WriteJSONResponse(w, process)
}

func (s *Server) handleGetServerProcess(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusBadRequest, map[string]interface{}{
			"errorCode": constants.ErrorCodeBadRequest,
			"context":   "server",
			"errors":    []string{"Invalid process ID"},
		})
		return
	}

	process, err := s.internalDB.GetServerProcess(id)
	if err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusNotFound, map[string]interface{}{
			"errorCode": constants.ErrorCodeNotFound,
			"context":   "server",
			"errors":    []string{err.Error()},
		})
		return
	}

	_ = utils.WriteJSONResponse(w, process)
}

func (s *Server) handleUpdateServerProcess(w http.ResponseWriter, r *http.Request) {
	if !s.requireUserPermission(w, r, permissions.ActionManageServer) {
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusBadRequest, map[string]interface{}{
			"errorCode": constants.ErrorCodeBadRequest,
			"context":   "server",
			"errors":    []string{"Invalid process ID"},
		})
		return
	}

	var req UpdateServerProcessRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusBadRequest, map[string]interface{}{
			"errorCode": constants.ErrorCodeBadRequest,
			"context":   "server",
			"errors":    []string{"Invalid request body"},
		})
		return
	}

	validate := validator.New()
	if err := validate.Struct(req); err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusBadRequest, map[string]interface{}{
			"errorCode": constants.ErrorCodeBadRequest,
			"context":   "server",
			"errors":    []string{err.Error()},
		})
		return
	}

	cleanPath := filepath.Clean(req.Path)
	if !s.validateServerProcessPath(w, cleanPath, &id) {
		return
	}

	if err := s.internalDB.UpdateServerProcess(id, req.Name, cleanPath, req.Port); err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeInternalServerError,
			"context":   "server",
			"errors":    []string{err.Error()},
		})
		return
	}

	process, err := s.internalDB.GetServerProcess(id)
	if err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeInternalServerError,
			"context":   "server",
			"errors":    []string{err.Error()},
		})
		return
	}

	_ = utils.WriteJSONResponse(w, process)
}

func (s *Server) handleDeleteServerProcess(w http.ResponseWriter, r *http.Request) {
	if !s.requireUserPermission(w, r, permissions.ActionManageServer) {
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusBadRequest, map[string]interface{}{
			"errorCode": constants.ErrorCodeBadRequest,
			"context":   "server",
			"errors":    []string{"Invalid process ID"},
		})
		return
	}

	proc, err := s.internalDB.GetServerProcess(id)
	if err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusNotFound, map[string]interface{}{
			"errorCode": constants.ErrorCodeNotFound,
			"context":   "server",
			"errors":    []string{err.Error()},
		})
		return
	}

	isRunning, err := s.processService.IsProcessRunning(proc.Path)
	if err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeInternalServerError,
			"context":   "server",
			"errors":    []string{"Failed to check process status: " + err.Error()},
		})
		return
	}

	if isRunning {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusBadRequest, map[string]interface{}{
			"errorCode": constants.ErrorCodeBadRequest,
			"context":   "server",
			"errors":    []string{"Cannot delete a process that is currently running. Please stop the process first."},
		})
		return
	}

	if err := s.internalDB.DeleteServerProcess(id); err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeInternalServerError,
			"context":   "server",
			"errors":    []string{err.Error()},
		})
		return
	}

	_ = utils.WriteJSONResponse(w, map[string]interface{}{
		"message": "Process deleted successfully",
	})
}

func (s *Server) handleReorderServerProcesses(w http.ResponseWriter, r *http.Request) {
	if !s.requireUserPermission(w, r, permissions.ActionManageServer) {
		return
	}

	var req ReorderServerProcessesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusBadRequest, map[string]interface{}{
			"errorCode": constants.ErrorCodeBadRequest,
			"context":   "server",
			"errors":    []string{"Invalid request body"},
		})
		return
	}

	validate := validator.New()
	if err := validate.Struct(req); err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusBadRequest, map[string]interface{}{
			"errorCode": constants.ErrorCodeBadRequest,
			"context":   "server",
			"errors":    []string{err.Error()},
		})
		return
	}

	if err := s.internalDB.ReorderServerProcesses(req.Updates); err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeInternalServerError,
			"context":   "server",
			"errors":    []string{err.Error()},
		})
		return
	}

	_ = utils.WriteJSONResponse(w, map[string]interface{}{
		"message": "Processes reordered successfully",
	})
}

func (s *Server) handleStartFullServer(w http.ResponseWriter, r *http.Request) {
	if !s.requireUserPermission(w, r, permissions.ActionManageServer) {
		return
	}

	if err := s.serverManagerService.StartServerSequence(); err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeInternalServerError,
			"context":   "server",
			"errors":    []string{err.Error()},
		})
		return
	}

	_ = utils.WriteJSONResponse(w, map[string]interface{}{
		"message": "Server started successfully",
	})
}

func (s *Server) handleStopFullServer(w http.ResponseWriter, r *http.Request) {
	if !s.requireUserPermission(w, r, permissions.ActionManageServer) {
		return
	}

	if err := s.serverManagerService.StopServerSequence(); err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeInternalServerError,
			"context":   "server",
			"errors":    []string{err.Error()},
		})
		return
	}

	_ = utils.WriteJSONResponse(w, map[string]interface{}{
		"message": "Server stopped successfully",
	})
}

func (s *Server) handleStartProcess(w http.ResponseWriter, r *http.Request) {
	if !s.requireUserPermission(w, r, permissions.ActionManageServer) {
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusBadRequest, map[string]interface{}{
			"errorCode": constants.ErrorCodeBadRequest,
			"context":   "server",
			"errors":    []string{"Invalid process ID"},
		})
		return
	}

	if err := s.serverManagerService.StartProcess(id); err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeInternalServerError,
			"context":   "server",
			"errors":    []string{err.Error()},
		})
		return
	}

	_ = utils.WriteJSONResponse(w, map[string]interface{}{
		"message": "Process started successfully",
	})
}

func (s *Server) handleStopProcess(w http.ResponseWriter, r *http.Request) {
	if !s.requireUserPermission(w, r, permissions.ActionManageServer) {
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusBadRequest, map[string]interface{}{
			"errorCode": constants.ErrorCodeBadRequest,
			"context":   "server",
			"errors":    []string{"Invalid process ID"},
		})
		return
	}

	if err := s.serverManagerService.StopProcess(id); err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeInternalServerError,
			"context":   "server",
			"errors":    []string{err.Error()},
		})
		return
	}

	_ = utils.WriteJSONResponse(w, map[string]interface{}{
		"message": "Process stopped successfully",
	})
}

func (s *Server) handleGetProcessStatus(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusBadRequest, map[string]interface{}{
			"errorCode": constants.ErrorCodeBadRequest,
			"context":   "server",
			"errors":    []string{"Invalid process ID"},
		})
		return
	}

	status, err := s.serverManagerService.GetProcessStatus(id)
	if err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeInternalServerError,
			"context":   "server",
			"errors":    []string{err.Error()},
		})
		return
	}

	_ = utils.WriteJSONResponse(w, status)
}

func (s *Server) validateServerProcessPath(w http.ResponseWriter, path string, excludeID *int64) bool {
	cleanPath := filepath.Clean(path)

	info, err := s.fileEditor.Stat(cleanPath)
	if err != nil {
		if s.fileEditor.IsNotExist(err) {
			_ = utils.WriteJSONResponseWithStatus(w, http.StatusBadRequest, map[string]interface{}{
				"errorCode": constants.ErrorCodeBadRequest,
				"context":   "server",
				"errors":    []string{"Path does not exist"},
			})
			return false
		}

		_ = utils.WriteJSONResponseWithStatus(w, http.StatusBadRequest, map[string]interface{}{
			"errorCode": constants.ErrorCodeBadRequest,
			"context":   "server",
			"errors":    []string{"Cannot access path: " + err.Error()},
		})
		return false
	}

	if info.IsDir() {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusBadRequest, map[string]interface{}{
			"errorCode": constants.ErrorCodeBadRequest,
			"context":   "server",
			"errors":    []string{"Path is a directory, not a file"},
		})
		return false
	}

	ext := strings.ToLower(filepath.Ext(cleanPath))
	validExtensions := []string{".exe", ".bat", ".cmd"}
	isValidExtension := false
	for _, validExt := range validExtensions {
		if ext == validExt {
			isValidExtension = true
			break
		}
	}

	if !isValidExtension {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusBadRequest, map[string]interface{}{
			"errorCode": constants.ErrorCodeBadRequest,
			"context":   "server",
			"errors":    []string{"Path must be an executable (.exe) or batch file (.bat, .cmd)"},
		})
		return false
	}

	existingProcess, err := s.internalDB.GetServerProcessByPath(cleanPath)
	if err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeInternalServerError,
			"context":   "server",
			"errors":    []string{"Failed to check for duplicate path: " + err.Error()},
		})
		return false
	}

	if existingProcess != nil {
		if excludeID == nil || existingProcess.ID != *excludeID {
			_ = utils.WriteJSONResponseWithStatus(w, http.StatusBadRequest, map[string]interface{}{
				"errorCode": constants.ErrorCodeBadRequest,
				"context":   "server",
				"errors":    []string{"A process with this path already exists"},
			})
			return false
		}
	}

	return true
}

type CreateServerProcessRequest struct {
	Name string `json:"name" validate:"required"`
	Path string `json:"path" validate:"required"`
	Port *int   `json:"port"`
}

type UpdateServerProcessRequest struct {
	Name string `json:"name" validate:"required"`
	Path string `json:"path" validate:"required"`
	Port *int   `json:"port"`
}

type ReorderServerProcessesRequest struct {
	Updates []db.ReorderUpdate `json:"updates" validate:"required"`
}
