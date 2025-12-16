package server

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/omnihance/omnihance-a3-agent/internal/constants"
	"github.com/omnihance/omnihance-a3-agent/internal/logger"
	"github.com/omnihance/omnihance-a3-agent/internal/mw"
	"github.com/omnihance/omnihance-a3-agent/internal/utils"
	"golang.org/x/crypto/bcrypt"
)

func (s *Server) InitializeSessionRoutes(r *chi.Mux) {
	r.Route("/api/session", func(r chi.Router) {
		r.Use(mw.CheckCookie(s.internalDB, s.cfg.CookieSecret))
		r.Get("/", s.getSessionHandler)
		r.Delete("/sign-out", s.signOutHandler)
		r.Post("/update-password", s.updatePasswordHandler)
	})
}

func (s *Server) getSessionHandler(w http.ResponseWriter, r *http.Request) {
	sessionID, _ := utils.GetSessionIDFromContext(r.Context())
	email, _ := utils.GetUserEmailFromContext(r.Context())
	roles, _ := utils.GetUserRolesFromContext(r.Context())
	userId, _ := utils.GetUserIdFromContext(r.Context())
	session, err := s.internalDB.GetSession(sessionID)
	if err != nil || session == nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusUnauthorized, map[string]interface{}{
			"errorCode": constants.ErrorCodeUnauthorized,
			"context":   "authentication",
			"errors":    []string{"Unauthorized"},
		})
		return
	}

	_ = utils.WriteJSONResponse(w, GetSessionResponse{
		SessionID: session.SessionID,
		UserID:    userId,
		Email:     email,
		Roles:     roles,
		CreatedAt: time.Unix(session.CreatedAt, 0),
		ExpiresAt: time.Unix(session.ExpiresAt, 0),
	})
}

func (s *Server) signOutHandler(w http.ResponseWriter, r *http.Request) {
	sessionID, _ := utils.GetSessionIDFromContext(r.Context())
	cookie, _ := r.Cookie(constants.CookieName)
	err := s.internalDB.DeleteSession(sessionID)
	if err != nil {
		s.log.Error(
			"failed to delete session",
			logger.Field{Key: "session_id", Value: sessionID},
			logger.Field{Key: "error", Value: err},
		)
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeInternalServerError,
			"context":   "authentication",
			"errors":    []string{"Failed to sign out"},
		})
	}

	cookie.Expires = time.Now().Add(-time.Hour * 24)
	http.SetCookie(w, cookie)

	_ = utils.WriteJSONResponse(w, map[string]interface{}{
		"success": true,
		"message": "Signed out successfully",
	})
}

func (s *Server) updatePasswordHandler(w http.ResponseWriter, r *http.Request) {
	var req UpdatePasswordRequest
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

	userID, _ := utils.GetUserIdFromContext(r.Context())
	sessionID, _ := utils.GetSessionIDFromContext(r.Context())

	user, err := s.internalDB.GetUserByID(userID)
	if err != nil {
		s.log.Error(
			"failed to get user",
			logger.Field{Key: "user_id", Value: userID},
			logger.Field{Key: "error", Value: err},
		)
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeInternalServerError,
			"context":   "password-update",
			"errors":    []string{"Failed to update password"},
		})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.CurrentPassword)); err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusBadRequest, map[string]interface{}{
			"errorCode": constants.ErrorCodeBadRequest,
			"context":   "password-update",
			"errors":    []string{"Current password is incorrect"},
		})
		return
	}

	if req.CurrentPassword == req.NewPassword {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusBadRequest, map[string]interface{}{
			"errorCode": constants.ErrorCodeBadRequest,
			"context":   "password-update",
			"errors":    []string{"New password must be different from current password"},
		})
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		s.log.Error(
			"failed to hash password",
			logger.Field{Key: "user_id", Value: userID},
			logger.Field{Key: "error", Value: err},
		)
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeInternalServerError,
			"context":   "password-update",
			"errors":    []string{"Failed to update password"},
		})
		return
	}

	if err := s.internalDB.UpdateUserPassword(userID, string(hashedPassword), userID); err != nil {
		s.log.Error(
			"failed to update user password",
			logger.Field{Key: "user_id", Value: userID},
			logger.Field{Key: "error", Value: err},
		)
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeInternalServerError,
			"context":   "password-update",
			"errors":    []string{"Failed to update password"},
		})
		return
	}

	if err := s.internalDB.DeleteUserSessionsExcept(userID, sessionID); err != nil {
		s.log.Error(
			"failed to delete other sessions",
			logger.Field{Key: "user_id", Value: userID},
			logger.Field{Key: "session_id", Value: sessionID},
			logger.Field{Key: "error", Value: err},
		)
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeInternalServerError,
			"context":   "password-update",
			"errors":    []string{"Failed to update password"},
		})
		return
	}

	_ = utils.WriteJSONResponse(w, map[string]interface{}{
		"success": true,
		"message": "Password updated successfully",
	})
}

type GetSessionResponse struct {
	SessionID string    `json:"session_id"`
	UserID    int64     `json:"user_id"`
	Email     string    `json:"email"`
	Roles     []string  `json:"roles"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

type UpdatePasswordRequest struct {
	CurrentPassword string `json:"current_password" validate:"required"`
	NewPassword     string `json:"new_password" validate:"required,min=6"`
}
