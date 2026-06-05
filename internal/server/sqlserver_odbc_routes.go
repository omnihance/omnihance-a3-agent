package server

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/omnihance/omnihance-a3-agent/internal/constants"
	"github.com/omnihance/omnihance-a3-agent/internal/mw"
	"github.com/omnihance/omnihance-a3-agent/internal/odbc/sqlserverdsn"
	"github.com/omnihance/omnihance-a3-agent/internal/permissions"
	"github.com/omnihance/omnihance-a3-agent/internal/utils"
	"github.com/omnihance/omnihance-a3-agent/internal/utils/userdsn"
)

func (s *Server) InitializeSQLServerODBCDSNRoutes(r *chi.Mux) {
	service := sqlserverdsn.NewService(userdsn.New())
	r.Route("/api/odbc/sqlserver-dsns", func(r chi.Router) {
		r.Use(mw.CheckCookie(s.internalDB, s.cfg.CookieSecret))
		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			if !s.requireUserPermission(w, r, permissions.ActionManageServer) {
				return
			}

			dsns, err := service.List()
			if err != nil {
				writeODBCError(w, http.StatusInternalServerError, err)
				return
			}

			_ = utils.WriteJSONResponse(w, map[string]interface{}{"dsns": dsns})
		})
		r.Get("/{name}", func(w http.ResponseWriter, r *http.Request) {
			if !s.requireUserPermission(w, r, permissions.ActionManageServer) {
				return
			}

			name := chi.URLParam(r, "name")
			dsn, err := service.Get(name)
			if err != nil {
				if errors.Is(err, userdsn.ErrDSNNotFound) {
					writeODBCError(w, http.StatusNotFound, err)
					return
				}
				writeODBCError(w, http.StatusInternalServerError, err)
				return
			}
			_ = utils.WriteJSONResponse(w, dsn)
		})
		r.Post("/", func(w http.ResponseWriter, r *http.Request) {
			if !s.requireUserPermission(w, r, permissions.ActionManageServer) {
				return
			}
			var req sqlServerDSNRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				writeODBCValidationError(w, "Invalid request body")
				return
			}
			if err := validator.New().Struct(req); err != nil {
				writeODBCValidationError(w, err.Error())
				return
			}
			err := service.Add(req.toModel())
			if err != nil {
				switch {
				case errors.Is(err, userdsn.ErrDSNAlreadyExists):
					writeODBCError(w, http.StatusBadRequest, err)
				case isODBCValidationError(err):
					writeODBCValidationError(w, err.Error())
				default:
					writeODBCError(w, http.StatusInternalServerError, err)
				}
				return
			}

			dsn, err := service.Get(req.Name)
			if err != nil {
				writeODBCError(w, http.StatusInternalServerError, err)
				return
			}
			_ = utils.WriteJSONResponse(w, dsn)
		})
		r.Post("/defaults", func(w http.ResponseWriter, r *http.Request) {
			if !s.requireUserPermission(w, r, permissions.ActionManageServer) {
				return
			}

			var req sqlServerDefaultDSNRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				writeODBCValidationError(w, "Invalid request body")
				return
			}
			if err := validator.New().Struct(req); err != nil {
				writeODBCValidationError(w, err.Error())
				return
			}

			result, err := service.CreateDefaults(req.Server, req.LoginID)
			if err != nil {
				if isODBCValidationError(err) {
					writeODBCValidationError(w, err.Error())
					return
				}
				writeODBCError(w, http.StatusInternalServerError, err)
				return
			}

			dsns, err := service.List()
			if err != nil {
				writeODBCError(w, http.StatusInternalServerError, err)
				return
			}
			_ = utils.WriteJSONResponse(w, map[string]interface{}{
				"created": result.Created,
				"skipped": result.Skipped,
				"dsns":    dsns,
			})
		})
		r.Put("/{name}", func(w http.ResponseWriter, r *http.Request) {
			if !s.requireUserPermission(w, r, permissions.ActionManageServer) {
				return
			}
			pathName := chi.URLParam(r, "name")
			var req sqlServerDSNRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				writeODBCValidationError(w, "Invalid request body")
				return
			}
			req.Name = pathName
			if err := validator.New().Struct(req); err != nil {
				writeODBCValidationError(w, err.Error())
				return
			}
			err := service.Update(req.toModel())
			if err != nil {
				switch {
				case errors.Is(err, userdsn.ErrDSNNotFound):
					writeODBCError(w, http.StatusNotFound, err)
				case isODBCValidationError(err):
					writeODBCValidationError(w, err.Error())
				default:
					writeODBCError(w, http.StatusInternalServerError, err)
				}
				return
			}
			dsn, err := service.Get(req.Name)
			if err != nil {
				writeODBCError(w, http.StatusInternalServerError, err)
				return
			}
			_ = utils.WriteJSONResponse(w, dsn)
		})
		r.Delete("/{name}", func(w http.ResponseWriter, r *http.Request) {
			if !s.requireUserPermission(w, r, permissions.ActionManageServer) {
				return
			}
			name := chi.URLParam(r, "name")
			err := service.Delete(name)
			if err != nil {
				switch {
				case errors.Is(err, userdsn.ErrDSNNotFound):
					writeODBCError(w, http.StatusNotFound, err)
				case isODBCValidationError(err):
					writeODBCValidationError(w, err.Error())
				default:
					writeODBCError(w, http.StatusInternalServerError, err)
				}
				return
			}
			_ = utils.WriteJSONResponse(w, map[string]interface{}{"message": "DSN deleted successfully"})
		})
		r.Post("/test", func(w http.ResponseWriter, r *http.Request) {
			if !s.requireUserPermission(w, r, permissions.ActionManageServer) {
				return
			}
			var req sqlServerDSNTestRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				writeODBCValidationError(w, "Invalid request body")
				return
			}
			if err := validator.New().Struct(req); err != nil {
				writeODBCValidationError(w, err.Error())
				return
			}

			if err := service.TestConnection(r.Context(), req.toModel()); err != nil {
				writeODBCError(w, http.StatusBadRequest, err)
				return
			}
			_ = utils.WriteJSONResponse(w, map[string]interface{}{"message": "Connection successful"})
		})
	})
}

type sqlServerDSNRequest struct {
	Name     string `json:"name" validate:"required"`
	Server   string `json:"server" validate:"required"`
	Database string `json:"database" validate:"required"`
	LoginID  string `json:"login_id" validate:"required"`
}

type sqlServerDSNTestRequest struct {
	Name     string `json:"name" validate:"required"`
	Server   string `json:"server" validate:"required"`
	Database string `json:"database" validate:"required"`
	LoginID  string `json:"login_id" validate:"required"`
	Password string `json:"password" validate:"required"`
}

type sqlServerDefaultDSNRequest struct {
	Server  string `json:"server" validate:"required"`
	LoginID string `json:"login_id" validate:"required"`
}

func (r sqlServerDSNRequest) toModel() sqlserverdsn.DSN {
	return sqlserverdsn.DSN{
		Name:     r.Name,
		Server:   r.Server,
		Database: r.Database,
		LoginID:  r.LoginID,
	}
}

func (r sqlServerDSNTestRequest) toModel() sqlserverdsn.DSN {
	return sqlserverdsn.DSN{
		Name:     r.Name,
		Server:   r.Server,
		Database: r.Database,
		LoginID:  r.LoginID,
		Password: r.Password,
	}
}

func writeODBCValidationError(w http.ResponseWriter, message string) {
	_ = utils.WriteJSONResponseWithStatus(w, http.StatusBadRequest, map[string]interface{}{
		"errorCode": constants.ErrorCodeBadRequest,
		"context":   "odbc-sqlserver-dsns",
		"errors":    []string{message},
	})
}

func writeODBCError(w http.ResponseWriter, status int, err error) {
	errorCode := constants.ErrorCodeInternalServerError
	if status == http.StatusBadRequest {
		errorCode = constants.ErrorCodeBadRequest
	}
	if status == http.StatusNotFound {
		errorCode = constants.ErrorCodeNotFound
	}
	_ = utils.WriteJSONResponseWithStatus(w, status, map[string]interface{}{
		"errorCode": errorCode,
		"context":   "odbc-sqlserver-dsns",
		"errors":    []string{err.Error()},
	})
}

func isODBCValidationError(err error) bool {
	return errors.Is(err, sqlserverdsn.ErrNameRequired) ||
		errors.Is(err, sqlserverdsn.ErrServerRequired) ||
		errors.Is(err, sqlserverdsn.ErrDatabaseRequired) ||
		errors.Is(err, sqlserverdsn.ErrLoginIDRequired) ||
		errors.Is(err, sqlserverdsn.ErrPasswordRequired)
}
