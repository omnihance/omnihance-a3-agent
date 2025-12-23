package server

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/omnihance/omnihance-a3-agent/internal/constants"
	"github.com/omnihance/omnihance-a3-agent/internal/logger"
	"github.com/omnihance/omnihance-a3-agent/internal/utils"
	"golang.org/x/crypto/bcrypt"
)

func (s *Server) InitializeAuthRoutes(r *chi.Mux) {
	r.Route("/api/auth", func(r chi.Router) {
		r.Post("/sign-in", s.signInHandler)
		r.Post("/sign-up", s.signUpHandler)
	})
}

func (s *Server) signInHandler(w http.ResponseWriter, r *http.Request) {
	var req AuthRequest
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
			case "email":
				errors = append(errors, "Invalid email")
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

	user, err := s.internalDB.GetUserByEmail(req.Email)
	if err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusUnauthorized, map[string]interface{}{
			"errorCode": constants.ErrorCodeUnauthorized,
			"context":   "authentication",
			"errors":    []string{"Invalid email or password"},
		})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusUnauthorized, map[string]interface{}{
			"errorCode": constants.ErrorCodeUnauthorized,
			"context":   "authentication",
			"errors":    []string{"Invalid email or password"},
		})
		return
	}

	if user.Status == constants.UserStatusBanned {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusUnauthorized, map[string]interface{}{
			"errorCode": constants.ErrorCodeUnauthorized,
			"context":   "authentication",
			"errors":    []string{"Account is banned"},
		})
		return
	}

	if user.Status != constants.UserStatusActive {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusUnauthorized, map[string]interface{}{
			"errorCode": constants.ErrorCodeUnauthorized,
			"context":   "authentication",
			"errors":    []string{"Account is not active"},
		})
		return
	}

	userAgent := r.UserAgent()
	ipAddress := r.RemoteAddr
	expiresAt := time.Now().Add(time.Duration(s.cfg.SessionTimeoutSeconds) * time.Second)

	session, err := s.internalDB.CreateSession(user.ID, expiresAt, &userAgent, &ipAddress)
	if err != nil {
		s.log.Error(
			"failed to create session",
			logger.Field{Key: "user_id", Value: user.ID},
			logger.Field{Key: "error", Value: err},
		)
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeInternalServerError,
			"context":   "session-creation",
			"errors":    []string{"Failed to create session"},
		})
		return
	}

	signedValue := utils.SignCookie(session.SessionID, s.cfg.CookieSecret)
	cookie := &http.Cookie{
		Name:     constants.CookieName,
		Value:    signedValue,
		Path:     constants.CookiePath,
		Expires:  expiresAt,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}
	http.SetCookie(w, cookie)

	_ = utils.WriteJSONResponse(w, AuthResponse{
		Success: true,
		Message: constants.SignInSuccessMessage,
	})
}

func (s *Server) signUpHandler(w http.ResponseWriter, r *http.Request) {
	var req AuthRequest
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
			case "email":
				errors = append(errors, "Invalid email")
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

	_, err := s.internalDB.GetUserByEmail(req.Email)
	if err == nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusBadRequest, map[string]interface{}{
			"errorCode": constants.ErrorCodeBadRequest,
			"context":   "form-validation",
			"errors":    []string{"Email already exists"},
		})
		return
	}

	adminUserCount, err := s.internalDB.GetAdminUserCount()
	if err != nil {
		s.log.Error(
			"failed to get admin user count",
			logger.Field{Key: "error", Value: err},
		)
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeInternalServerError,
			"context":   "user-creation",
			"errors":    []string{"Failed to create account"},
		})
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		s.log.Error(
			"failed to hash password",
			logger.Field{Key: "error", Value: err},
		)
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeInternalServerError,
			"context":   "user-creation",
			"errors":    []string{"Failed to create account"},
		})
		return
	}

	var role, status, message string
	if adminUserCount == 0 {
		role = constants.RoleSuperAdmin
		status = constants.UserStatusActive
		message = constants.SignUpSuccessMessageActive
	} else {
		role = constants.RoleUser
		status = constants.UserStatusPending
		message = constants.SignUpSuccessMessagePending
	}

	_, err = s.internalDB.CreateUserWithStatus(req.Email, string(hashedPassword), role, status, nil)
	if err != nil {
		s.log.Error(
			"failed to create user",
			logger.Field{Key: "email", Value: req.Email},
			logger.Field{Key: "error", Value: err},
		)
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeInternalServerError,
			"context":   "user-creation",
			"errors":    []string{"Failed to create account"},
		})
		return
	}

	_ = utils.WriteJSONResponse(w, AuthResponse{
		Success: true,
		Message: message,
	})
}

type AuthRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6"`
}

type AuthResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}
