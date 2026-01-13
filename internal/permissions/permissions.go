package permissions

import (
	"strings"

	"github.com/omnihance/omnihance-a3-agent/internal/constants"
)

type PermissionAction string

const (
	ActionViewFiles      PermissionAction = "view_files"
	ActionEditFiles      PermissionAction = "edit_files"
	ActionRevertFiles    PermissionAction = "revert_files"
	ActionUploadGameData PermissionAction = "upload_game_data"
	ActionManageUsers    PermissionAction = "manage_users"
	ActionViewMetrics    PermissionAction = "view_metrics"
	ActionViewGameData   PermissionAction = "view_game_data"
	ActionManageServer   PermissionAction = "manage_server"
)

var rolePermissions = map[PermissionAction][]string{
	ActionViewFiles:      {constants.RoleSuperAdmin, constants.RoleAdmin, constants.RoleUser},
	ActionEditFiles:      {constants.RoleSuperAdmin, constants.RoleAdmin},
	ActionRevertFiles:    {constants.RoleSuperAdmin, constants.RoleAdmin},
	ActionUploadGameData: {constants.RoleSuperAdmin, constants.RoleAdmin},
	ActionManageUsers:    {constants.RoleSuperAdmin},
	ActionViewMetrics:    {constants.RoleSuperAdmin, constants.RoleAdmin, constants.RoleUser},
	ActionViewGameData:   {constants.RoleSuperAdmin, constants.RoleAdmin, constants.RoleUser},
	ActionManageServer:   {constants.RoleSuperAdmin, constants.RoleAdmin},
}

func normalizeRole(role string) string {
	return strings.TrimSpace(strings.ToLower(role))
}

func IsAllowed(action PermissionAction, roles []string) bool {
	allowedRoles, exists := rolePermissions[action]
	if !exists {
		return false
	}

	allowedRolesMap := make(map[string]bool)
	for _, role := range allowedRoles {
		allowedRolesMap[normalizeRole(role)] = true
	}

	for _, userRole := range roles {
		normalizedRole := normalizeRole(userRole)
		if allowedRolesMap[normalizedRole] {
			return true
		}
	}

	return false
}
