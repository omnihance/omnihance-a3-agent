package permissions

import (
	"testing"

	"github.com/omnihance/omnihance-a3-agent/internal/constants"
)

func TestIsAllowed(t *testing.T) {
	tests := []struct {
		name     string
		action   PermissionAction
		roles    []string
		expected bool
	}{
		{
			name:     "super_admin can view files",
			action:   ActionViewFiles,
			roles:    []string{constants.RoleSuperAdmin},
			expected: true,
		},
		{
			name:     "admin can view files",
			action:   ActionViewFiles,
			roles:    []string{constants.RoleAdmin},
			expected: true,
		},
		{
			name:     "viewer can view files",
			action:   ActionViewFiles,
			roles:    []string{constants.RoleUser},
			expected: true,
		},
		{
			name:     "super_admin can edit files",
			action:   ActionEditFiles,
			roles:    []string{constants.RoleSuperAdmin},
			expected: true,
		},
		{
			name:     "admin can edit files",
			action:   ActionEditFiles,
			roles:    []string{constants.RoleAdmin},
			expected: true,
		},
		{
			name:     "viewer cannot edit files",
			action:   ActionEditFiles,
			roles:    []string{constants.RoleUser},
			expected: false,
		},
		{
			name:     "super_admin can revert files",
			action:   ActionRevertFiles,
			roles:    []string{constants.RoleSuperAdmin},
			expected: true,
		},
		{
			name:     "admin can revert files",
			action:   ActionRevertFiles,
			roles:    []string{constants.RoleAdmin},
			expected: true,
		},
		{
			name:     "viewer cannot revert files",
			action:   ActionRevertFiles,
			roles:    []string{constants.RoleUser},
			expected: false,
		},
		{
			name:     "super_admin can upload game data",
			action:   ActionUploadGameData,
			roles:    []string{constants.RoleSuperAdmin},
			expected: true,
		},
		{
			name:     "admin can upload game data",
			action:   ActionUploadGameData,
			roles:    []string{constants.RoleAdmin},
			expected: true,
		},
		{
			name:     "viewer cannot upload game data",
			action:   ActionUploadGameData,
			roles:    []string{constants.RoleUser},
			expected: false,
		},
		{
			name:     "super_admin can manage users",
			action:   ActionManageUsers,
			roles:    []string{constants.RoleSuperAdmin},
			expected: true,
		},
		{
			name:     "admin cannot manage users",
			action:   ActionManageUsers,
			roles:    []string{constants.RoleAdmin},
			expected: false,
		},
		{
			name:     "viewer cannot manage users",
			action:   ActionManageUsers,
			roles:    []string{constants.RoleUser},
			expected: false,
		},
		{
			name:     "super_admin can view metrics",
			action:   ActionViewMetrics,
			roles:    []string{constants.RoleSuperAdmin},
			expected: true,
		},
		{
			name:     "admin can view metrics",
			action:   ActionViewMetrics,
			roles:    []string{constants.RoleAdmin},
			expected: true,
		},
		{
			name:     "viewer can view metrics",
			action:   ActionViewMetrics,
			roles:    []string{constants.RoleUser},
			expected: true,
		},
		{
			name:     "super_admin can view game data",
			action:   ActionViewGameData,
			roles:    []string{constants.RoleSuperAdmin},
			expected: true,
		},
		{
			name:     "admin can view game data",
			action:   ActionViewGameData,
			roles:    []string{constants.RoleAdmin},
			expected: true,
		},
		{
			name:     "viewer can view game data",
			action:   ActionViewGameData,
			roles:    []string{constants.RoleUser},
			expected: true,
		},
		{
			name:     "multiple roles with one allowed grants access",
			action:   ActionEditFiles,
			roles:    []string{constants.RoleUser, constants.RoleAdmin},
			expected: true,
		},
		{
			name:     "multiple roles with none allowed denies access",
			action:   ActionEditFiles,
			roles:    []string{constants.RoleUser, "unknown_role"},
			expected: false,
		},
		{
			name:     "empty roles denies access",
			action:   ActionViewFiles,
			roles:    []string{},
			expected: false,
		},
		{
			name:     "unknown action denies access",
			action:   PermissionAction("unknown_action"),
			roles:    []string{constants.RoleSuperAdmin},
			expected: false,
		},
		{
			name:     "role with whitespace is normalized",
			action:   ActionViewFiles,
			roles:    []string{"  " + constants.RoleSuperAdmin + "  "},
			expected: true,
		},
		{
			name:     "role case is normalized",
			action:   ActionViewFiles,
			roles:    []string{"SUPER_ADMIN"},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsAllowed(tt.action, tt.roles)
			if result != tt.expected {
				t.Errorf("IsAllowed(%v, %v) = %v, expected %v", tt.action, tt.roles, result, tt.expected)
			}
		})
	}
}

func TestNormalizeRole(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "normalize lowercase",
			input:    "super_admin",
			expected: "super_admin",
		},
		{
			name:     "normalize uppercase",
			input:    "SUPER_ADMIN",
			expected: "super_admin",
		},
		{
			name:     "normalize mixed case",
			input:    "Super_Admin",
			expected: "super_admin",
		},
		{
			name:     "normalize with whitespace",
			input:    "  super_admin  ",
			expected: "super_admin",
		},
		{
			name:     "normalize empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeRole(tt.input)
			if result != tt.expected {
				t.Errorf("normalizeRole(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

