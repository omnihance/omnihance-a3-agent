package mw

import (
	"net/http"
	"strings"

	"github.com/omnihance/omnihance-a3-agent/internal/constants"
	"github.com/omnihance/omnihance-a3-agent/internal/db"
	"github.com/omnihance/omnihance-a3-agent/internal/utils"
)

func CheckCookie(db db.InternalDB, cookieSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(constants.CookieName)
			if err != nil {
				_ = utils.WriteJSONResponseWithStatus(w, http.StatusUnauthorized, map[string]interface{}{
					"errorCode": constants.ErrorCodeUnauthorized,
					"context":   "authentication",
					"errors":    []string{"Unauthorized"},
				})
				return
			}

			sessionID, err := utils.VerifyCookie(cookie.Value, cookieSecret)
			if err != nil {
				_ = utils.WriteJSONResponseWithStatus(w, http.StatusUnauthorized, map[string]interface{}{
					"errorCode": constants.ErrorCodeUnauthorized,
					"context":   "authentication",
					"errors":    []string{"Unauthorized"},
				})
				return
			}

			session, err := db.GetSession(sessionID)
			if err != nil || session == nil {
				_ = utils.WriteJSONResponseWithStatus(w, http.StatusUnauthorized, map[string]interface{}{
					"errorCode": constants.ErrorCodeUnauthorized,
					"context":   "authentication",
					"errors":    []string{"Unauthorized"},
				})
				return
			}

			user, err := db.GetActiveUserByID(session.UserID)
			if err != nil || user == nil {
				_ = utils.WriteJSONResponseWithStatus(w, http.StatusUnauthorized, map[string]interface{}{
					"errorCode": constants.ErrorCodeUnauthorized,
					"context":   "authentication",
					"errors":    []string{"Unauthorized"},
				})
				return
			}

			ctx := r.Context()
			ctx = utils.SetUserIdInContext(ctx, user.ID)
			ctx = utils.SetUserRolesInContext(ctx, strings.Split(user.Roles, ","))
			ctx = utils.SetUserEmailInContext(ctx, user.Email)
			ctx = utils.SetSessionIDInContext(ctx, session.SessionID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
