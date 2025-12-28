package pathutil

import (
	"os"
	"path/filepath"
	"strings"
)

// ExpandUser expands a leading ~ to the current user's home directory.
func ExpandUser(path string) string {
	if path == "" {
		return path
	}
	if strings.HasPrefix(path, "~") {
		home, err := os.UserHomeDir()
		if err == nil {
			if path == "~" {
				return home
			}
			if strings.HasPrefix(path, "~/") {
				return filepath.Join(home, path[2:])
			}
		}
	}
	return path
}

// ShortenUser replaces the current user's home directory prefix with ~.
func ShortenUser(path string) string {
	if path == "" {
		return path
	}
	home, err := os.UserHomeDir()
	if err == nil && strings.HasPrefix(path, home) {
		return "~" + strings.TrimPrefix(path, home)
	}
	return path
}
