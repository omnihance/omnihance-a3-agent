package server

import (
	"net/http"

	"github.com/omnihance/omnihance-a3-agent/internal/constants"
	"github.com/omnihance/omnihance-a3-agent/internal/permissions"
	"github.com/omnihance/omnihance-a3-agent/internal/utils"
)

func (s *Server) requireUserPermission(w http.ResponseWriter, r *http.Request, action permissions.PermissionAction) bool {
	userRoles, ok := utils.GetUserRolesFromContext(r.Context())
	if !ok {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusForbidden, map[string]interface{}{
			"errorCode": constants.ErrorCodeUnauthorized,
			"context":   "authorization",
			"errors":    []string{"User roles not found in context"},
		})
		return false
	}

	if !permissions.IsAllowed(action, userRoles) {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusForbidden, map[string]interface{}{
			"errorCode": constants.ErrorCodeUnauthorized,
			"context":   "authorization",
			"errors":    []string{"Insufficient permissions"},
		})
		return false
	}

	return true
}

