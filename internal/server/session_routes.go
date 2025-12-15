package server

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/omnihance/omnihance-a3-agent/internal/constants"
	"github.com/omnihance/omnihance-a3-agent/internal/logger"
	"github.com/omnihance/omnihance-a3-agent/internal/mw"
	"github.com/omnihance/omnihance-a3-agent/internal/utils"
)

func (s *Server) InitializeSessionRoutes(r *chi.Mux) {
	r.Route("/api/session", func(r chi.Router) {
		r.Use(mw.CheckCookie(s.internalDB, s.cfg.CookieSecret))
		r.Get("/", s.getSessionHandler)
		r.Delete("/sign-out", s.signOutHandler)
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

type GetSessionResponse struct {
	SessionID string    `json:"session_id"`
	UserID    int64     `json:"user_id"`
	Email     string    `json:"email"`
	Roles     []string  `json:"roles"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}
