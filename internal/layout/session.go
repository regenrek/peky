package layout

import (
	"path/filepath"
	"strings"
)

// SanitizeSessionName normalizes a session name for defaulting purposes.
func SanitizeSessionName(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	if name == "" {
		return "session"
	}
	var b strings.Builder
	lastDash := false
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
			lastDash = false
		case r >= '0' && r <= '9':
			b.WriteRune(r)
			lastDash = false
		case r == '-' || r == '_' || r == ' ':
			if !lastDash {
				b.WriteRune('-')
				lastDash = true
			}
		}
	}
	result := strings.Trim(b.String(), "-")
	if result == "" {
		return "session"
	}
	return result
}

// ResolveSessionName resolves the session name for a project start.
func ResolveSessionName(projectPath, requested string, projectConfig *ProjectLocalConfig) string {
	requested = strings.TrimSpace(requested)
	if requested != "" {
		return requested
	}
	if projectConfig != nil {
		if session := strings.TrimSpace(projectConfig.Session); session != "" {
			return session
		}
	}
	return SanitizeSessionName(filepath.Base(projectPath))
}
