package server

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/omnihance/omnihance-a3-agent/internal/constants"
	"github.com/omnihance/omnihance-a3-agent/internal/mw"
	"github.com/omnihance/omnihance-a3-agent/internal/utils"
)

func (s *Server) InitializeDirectoryShortcutsRoutes(r *chi.Mux) {
	r.Route("/api/directory-shortcuts", func(r chi.Router) {
		r.Use(mw.CheckCookie(s.internalDB, s.cfg.CookieSecret))
		r.Get("/", s.handleGetDirectoryShortcuts)
		r.Post("/", s.handleCreateDirectoryShortcut)
		r.Delete("/{id}", s.handleDeleteDirectoryShortcut)
	})
}

func (s *Server) handleGetDirectoryShortcuts(w http.ResponseWriter, r *http.Request) {
	userID, ok := utils.GetUserIdFromContext(r.Context())
	if !ok {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusUnauthorized, map[string]interface{}{
			"errorCode": constants.ErrorCodeUnauthorized,
			"context":   "directory-shortcuts",
			"errors":    []string{"User ID not found in context"},
		})
		return
	}

	shortcuts, err := s.internalDB.GetDirectoryShortcuts(userID)
	if err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeInternalServerError,
			"context":   "directory-shortcuts",
			"errors":    []string{err.Error()},
		})
		return
	}

	count, err := s.internalDB.GetDirectoryShortcutCount(userID)
	if err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeInternalServerError,
			"context":   "directory-shortcuts",
			"errors":    []string{err.Error()},
		})
		return
	}

	limit := s.cfg.DirectoryShortcutsLimit
	overLimitBy := int64(0)
	if count > int64(limit) {
		overLimitBy = count - int64(limit)
	}

	_ = utils.WriteJSONResponse(w, map[string]interface{}{
		"shortcuts":     shortcuts,
		"limit":         limit,
		"over_limit_by": overLimitBy,
	})
}

func (s *Server) handleCreateDirectoryShortcut(w http.ResponseWriter, r *http.Request) {
	userID, ok := utils.GetUserIdFromContext(r.Context())
	if !ok {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusUnauthorized, map[string]interface{}{
			"errorCode": constants.ErrorCodeUnauthorized,
			"context":   "directory-shortcuts",
			"errors":    []string{"User ID not found in context"},
		})
		return
	}

	var req CreateDirectoryShortcutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusBadRequest, map[string]interface{}{
			"errorCode": constants.ErrorCodeBadRequest,
			"context":   "directory-shortcuts",
			"errors":    []string{"Invalid request body"},
		})
		return
	}

	validate := validator.New()
	if err := validate.Struct(req); err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusBadRequest, map[string]interface{}{
			"errorCode": constants.ErrorCodeBadRequest,
			"context":   "directory-shortcuts",
			"errors":    []string{err.Error()},
		})
		return
	}

	cleanPath := filepath.Clean(req.Path)
	if s.isRootPath(cleanPath) {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusBadRequest, map[string]interface{}{
			"errorCode": constants.ErrorCodeBadRequest,
			"context":   "directory-shortcuts",
			"errors":    []string{"Cannot pin root directory"},
		})
		return
	}

	normalizedPath := utils.NormalizePathForShortcut(cleanPath)
	existing, err := s.internalDB.GetDirectoryShortcutByNormalizedPath(userID, normalizedPath)
	if err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeInternalServerError,
			"context":   "directory-shortcuts",
			"errors":    []string{err.Error()},
		})
		return
	}

	if existing != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusBadRequest, map[string]interface{}{
			"errorCode": constants.ErrorCodeBadRequest,
			"context":   "directory-shortcuts",
			"errors":    []string{"Directory already in shortcuts"},
		})
		return
	}

	count, err := s.internalDB.GetDirectoryShortcutCount(userID)
	if err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeInternalServerError,
			"context":   "directory-shortcuts",
			"errors":    []string{err.Error()},
		})
		return
	}

	limit := s.cfg.DirectoryShortcutsLimit
	if limit > 0 && count >= int64(limit) {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusBadRequest, map[string]interface{}{
			"errorCode": constants.ErrorCodeBadRequest,
			"context":   "directory-shortcuts",
			"errors":    []string{"Shortcut limit reached"},
		})
		return
	}

	shortcut, err := s.internalDB.CreateDirectoryShortcut(userID, req.Name, cleanPath)
	if err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeInternalServerError,
			"context":   "directory-shortcuts",
			"errors":    []string{err.Error()},
		})
		return
	}

	_ = utils.WriteJSONResponse(w, shortcut)
}

func (s *Server) handleDeleteDirectoryShortcut(w http.ResponseWriter, r *http.Request) {
	userID, ok := utils.GetUserIdFromContext(r.Context())
	if !ok {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusUnauthorized, map[string]interface{}{
			"errorCode": constants.ErrorCodeUnauthorized,
			"context":   "directory-shortcuts",
			"errors":    []string{"User ID not found in context"},
		})
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusBadRequest, map[string]interface{}{
			"errorCode": constants.ErrorCodeBadRequest,
			"context":   "directory-shortcuts",
			"errors":    []string{"Invalid shortcut ID"},
		})
		return
	}

	shortcut, err := s.internalDB.GetDirectoryShortcut(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || strings.Contains(strings.ToLower(err.Error()), "not found") {
			_ = utils.WriteJSONResponseWithStatus(w, http.StatusNotFound, map[string]interface{}{
				"errorCode": constants.ErrorCodeNotFound,
				"context":   "directory-shortcuts",
				"errors":    []string{"Shortcut not found"},
			})
			return
		}

		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeInternalServerError,
			"context":   "directory-shortcuts",
			"errors":    []string{err.Error()},
		})
		return
	}

	if shortcut.UserID != userID {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusForbidden, map[string]interface{}{
			"errorCode": constants.ErrorCodeUnauthorized,
			"context":   "directory-shortcuts",
			"errors":    []string{"Not authorized to delete this shortcut"},
		})
		return
	}

	if err := s.internalDB.DeleteDirectoryShortcut(id, userID); err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeInternalServerError,
			"context":   "directory-shortcuts",
			"errors":    []string{err.Error()},
		})
		return
	}

	_ = utils.WriteJSONResponse(w, map[string]interface{}{
		"message": "Shortcut deleted successfully",
	})
}

func (s *Server) isRootPath(path string) bool {
	if path == "" {
		return true
	}

	normalized := utils.NormalizePathForShortcut(path)
	if normalized == "" {
		return true
	}

	if normalized == "/" {
		return true
	}

	if len(normalized) == 2 && normalized[1] == ':' {
		firstChar := normalized[0]
		if firstChar >= 'a' && firstChar <= 'z' {
			return true
		}
	}

	return false
}

type CreateDirectoryShortcutRequest struct {
	Name string `json:"name" validate:"required"`
	Path string `json:"path" validate:"required"`
}
