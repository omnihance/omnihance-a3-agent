package server

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/omnihance/omnihance-a3-agent/internal/constants"
	"github.com/omnihance/omnihance-a3-agent/internal/logger"
	"github.com/omnihance/omnihance-a3-agent/internal/mw"
	"github.com/omnihance/omnihance-a3-agent/internal/permissions"
	"github.com/omnihance/omnihance-a3-agent/internal/utils"
	"golang.org/x/crypto/bcrypt"
)

var allowedUserStatuses = map[string]bool{
	constants.UserStatusPending:  true,
	constants.UserStatusActive:   true,
	constants.UserStatusInactive: true,
	constants.UserStatusBanned:   true,
}

func (s *Server) InitializeUserManagementRoutes(r *chi.Mux) {
	r.Route("/api/users", func(r chi.Router) {
		r.Use(mw.CheckCookie(s.internalDB, s.cfg.CookieSecret))
		r.Get("/", s.handleListUsers)
		r.Get("/statuses", s.handleGetUserStatuses)
		r.Patch("/{id}/status", s.handleUpdateUserStatus)
		r.Patch("/{id}/password", s.handleSetUserPassword)
	})
}

func (s *Server) handleListUsers(w http.ResponseWriter, r *http.Request) {
	if !s.requireUserPermission(w, r, permissions.ActionManageUsers) {
		return
	}

	page := 1
	pageSize := 10
	search := ""

	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		if parsed, err := strconv.Atoi(pageStr); err == nil && parsed >= 1 {
			page = parsed
		}
	}

	if pageSizeStr := r.URL.Query().Get("pageSize"); pageSizeStr != "" {
		if parsed, err := strconv.Atoi(pageSizeStr); err == nil && parsed >= 1 && parsed <= 100 {
			pageSize = parsed
		}
	}

	if searchStr := r.URL.Query().Get("s"); searchStr != "" {
		search = searchStr
	}

	users, totalCount, err := s.internalDB.GetUsersPaginated(page, pageSize, search)
	if err != nil {
		s.log.Error(
			"failed to get paginated users",
			logger.Field{Key: "error", Value: err},
		)
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeInternalServerError,
			"context":   "user-management",
			"errors":    []string{"Failed to retrieve users"},
		})
		return
	}

	userList := make([]UserListItem, 0, len(users))
	for _, user := range users {
		roles := strings.Split(user.Roles, ",")
		trimmedRoles := make([]string, 0, len(roles))
		for _, role := range roles {
			trimmedRole := strings.TrimSpace(role)
			if trimmedRole != "" {
				trimmedRoles = append(trimmedRoles, trimmedRole)
			}
		}

		userList = append(userList, UserListItem{
			ID:        user.ID,
			Email:     user.Email,
			Roles:     trimmedRoles,
			Status:    user.Status,
			CreatedAt: user.CreatedAt,
		})
	}

	_ = utils.WriteJSONResponse(w, ListUsersResponse{
		Data: userList,
		Pagination: PaginationInfo{
			TotalCount: totalCount,
			Page:       page,
			PageSize:   pageSize,
		},
	})
}

func (s *Server) handleGetUserStatuses(w http.ResponseWriter, r *http.Request) {
	if !s.requireUserPermission(w, r, permissions.ActionManageUsers) {
		return
	}

	statuses := make([]string, 0, len(allowedUserStatuses))
	for status := range allowedUserStatuses {
		statuses = append(statuses, status)
	}

	_ = utils.WriteJSONResponse(w, GetUserStatusesResponse{
		Statuses: statuses,
	})
}

func (s *Server) handleUpdateUserStatus(w http.ResponseWriter, r *http.Request) {
	if !s.requireUserPermission(w, r, permissions.ActionManageUsers) {
		return
	}

	userIDStr := chi.URLParam(r, "id")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusBadRequest, map[string]interface{}{
			"errorCode": constants.ErrorCodeBadRequest,
			"context":   "user-management",
			"errors":    []string{"Invalid user ID"},
		})
		return
	}

	var req UpdateUserStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusBadRequest, map[string]interface{}{
			"errorCode": constants.ErrorCodeBadRequest,
			"context":   "form-validation",
			"errors":    []string{"Invalid request body"},
		})
		return
	}

	validate := validator.New()
	if err := validate.Struct(req); err != nil {
		var errors []string
		for _, err := range err.(validator.ValidationErrors) {
			switch err.Tag() {
			case "required":
				errors = append(errors, err.Field()+" is required")
			case "oneof":
				errors = append(errors, "Invalid status value")
			default:
				errors = append(errors, "Invalid "+err.Field())
			}
		}

		_ = utils.WriteJSONResponseWithStatus(w, http.StatusBadRequest, map[string]interface{}{
			"errorCode": constants.ErrorCodeBadRequest,
			"context":   "form-validation",
			"errors":    errors,
		})
		return
	}

	if !isValidUserStatus(req.Status) {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusBadRequest, map[string]interface{}{
			"errorCode": constants.ErrorCodeBadRequest,
			"context":   "form-validation",
			"errors":    []string{"Invalid status value"},
		})
		return
	}

	user, err := s.internalDB.GetUserByID(userID)
	if err != nil {
		s.log.Error(
			"failed to get user",
			logger.Field{Key: "user_id", Value: userID},
			logger.Field{Key: "error", Value: err},
		)
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusNotFound, map[string]interface{}{
			"errorCode": constants.ErrorCodeNotFound,
			"context":   "user-management",
			"errors":    []string{"User not found"},
		})
		return
	}

	roles := strings.Split(user.Roles, ",")
	for _, role := range roles {
		if strings.TrimSpace(role) == constants.RoleSuperAdmin {
			_ = utils.WriteJSONResponseWithStatus(w, http.StatusBadRequest, map[string]interface{}{
				"errorCode": constants.ErrorCodeBadRequest,
				"context":   "user-management",
				"errors":    []string{"Cannot update status of Super Admin user"},
			})
			return
		}
	}

	if user.Status == req.Status {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusBadRequest, map[string]interface{}{
			"errorCode": constants.ErrorCodeBadRequest,
			"context":   "user-management",
			"errors":    []string{"New status must be different from current status"},
		})
		return
	}

	updatedBy, _ := utils.GetUserIdFromContext(r.Context())
	if err := s.internalDB.UpdateUserStatus(userID, req.Status, updatedBy); err != nil {
		s.log.Error(
			"failed to update user status",
			logger.Field{Key: "user_id", Value: userID},
			logger.Field{Key: "status", Value: req.Status},
			logger.Field{Key: "error", Value: err},
		)
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeInternalServerError,
			"context":   "user-management",
			"errors":    []string{"Failed to update user status"},
		})
		return
	}

	_ = utils.WriteJSONResponse(w, map[string]interface{}{
		"success": true,
		"message": "User status updated successfully",
	})
}

func (s *Server) handleSetUserPassword(w http.ResponseWriter, r *http.Request) {
	if !s.requireUserPermission(w, r, permissions.ActionManageUsers) {
		return
	}

	userIDStr := chi.URLParam(r, "id")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusBadRequest, map[string]interface{}{
			"errorCode": constants.ErrorCodeBadRequest,
			"context":   "user-management",
			"errors":    []string{"Invalid user ID"},
		})
		return
	}

	var req SetUserPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusBadRequest, map[string]interface{}{
			"errorCode": constants.ErrorCodeBadRequest,
			"context":   "form-validation",
			"errors":    []string{"Invalid request body"},
		})
		return
	}

	validate := validator.New()
	if err := validate.Struct(req); err != nil {
		var errors []string
		for _, err := range err.(validator.ValidationErrors) {
			switch err.Tag() {
			case "required":
				errors = append(errors, err.Field()+" is required")
			case "min":
				errors = append(errors, err.Field()+" is too short")
			default:
				errors = append(errors, "Invalid "+err.Field())
			}
		}

		_ = utils.WriteJSONResponseWithStatus(w, http.StatusBadRequest, map[string]interface{}{
			"errorCode": constants.ErrorCodeBadRequest,
			"context":   "form-validation",
			"errors":    errors,
		})
		return
	}

	_, err = s.internalDB.GetUserByID(userID)
	if err != nil {
		s.log.Error(
			"failed to get user",
			logger.Field{Key: "user_id", Value: userID},
			logger.Field{Key: "error", Value: err},
		)
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusNotFound, map[string]interface{}{
			"errorCode": constants.ErrorCodeNotFound,
			"context":   "user-management",
			"errors":    []string{"User not found"},
		})
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		s.log.Error(
			"failed to hash password",
			logger.Field{Key: "user_id", Value: userID},
			logger.Field{Key: "error", Value: err},
		)
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeInternalServerError,
			"context":   "user-management",
			"errors":    []string{"Failed to set password"},
		})
		return
	}

	updatedBy, _ := utils.GetUserIdFromContext(r.Context())
	if err := s.internalDB.UpdateUserPassword(userID, string(hashedPassword), updatedBy); err != nil {
		s.log.Error(
			"failed to update user password",
			logger.Field{Key: "user_id", Value: userID},
			logger.Field{Key: "error", Value: err},
		)
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeInternalServerError,
			"context":   "user-management",
			"errors":    []string{"Failed to set password"},
		})
		return
	}

	_ = utils.WriteJSONResponse(w, map[string]interface{}{
		"success": true,
		"message": "Password set successfully",
	})
}

type ListUsersResponse struct {
	Data       []UserListItem `json:"data"`
	Pagination PaginationInfo `json:"pagination"`
}

type PaginationInfo struct {
	TotalCount int64 `json:"totalCount"`
	Page       int   `json:"page"`
	PageSize   int   `json:"pageSize"`
}

type UserListItem struct {
	ID        int64     `json:"id"`
	Email     string    `json:"email"`
	Roles     []string  `json:"roles"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

type UpdateUserStatusRequest struct {
	Status string `json:"status" validate:"required,oneof=pending active inactive banned"`
}

func isValidUserStatus(status string) bool {
	return allowedUserStatuses[status]
}

type SetUserPasswordRequest struct {
	Password string `json:"password" validate:"required,min=6"`
}

type GetUserStatusesResponse struct {
	Statuses []string `json:"statuses"`
}
