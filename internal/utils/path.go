package utils

import (
	"runtime"
	"strings"
)

// NormalizePathForShortcut normalizes a file path for use in directory shortcuts.
// It converts backslashes to forward slashes, trims whitespace, and removes trailing slashes.
// On Windows, it also converts to lowercase and handles drive roots (e.g., "C:" or "C:/").
// On other platforms, it preserves case for case-sensitive filesystems.
func NormalizePathForShortcut(path string) string {
	normalized := strings.ReplaceAll(path, "\\", "/")
	normalized = strings.TrimSpace(normalized)

	if runtime.GOOS == "windows" {
		normalized = strings.ToLower(normalized)
	}

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
