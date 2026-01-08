package utils

import (
	"runtime"
	"strings"
)

// NormalizePathForShortcut normalizes a file path for use in directory shortcuts.
// It converts backslashes to forward slashes, trims whitespace, converts to lowercase,
// and removes trailing slashes. On Windows, it handles drive roots (e.g., "C:" or "C:/").
func NormalizePathForShortcut(path string) string {
	normalized := strings.ReplaceAll(path, "\\", "/")
	normalized = strings.TrimSpace(normalized)
	normalized = strings.ToLower(normalized)
	normalized = strings.TrimSuffix(normalized, "/")

	if runtime.GOOS == "windows" {
		if len(normalized) == 2 && normalized[1] == ':' {
			return normalized
		}
		if len(normalized) == 3 && normalized[1] == ':' && normalized[2] == '/' {
			return normalized[:2]
		}
	}

	return normalized
}
