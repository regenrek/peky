package zellijctl

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// DefaultLayoutDir returns the directory used for generated zellij layouts.
func DefaultLayoutDir() (string, error) {
	dir, err := DefaultBridgeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "layouts"), nil
}

// WriteLayoutFile writes a layout file and returns its path.
func WriteLayoutFile(layoutDir, name, content string) (string, error) {
	if strings.TrimSpace(content) == "" {
		return "", fmt.Errorf("layout content is required")
	}
	if strings.TrimSpace(name) == "" {
		return "", fmt.Errorf("layout name is required")
	}
	if strings.TrimSpace(layoutDir) == "" {
		var err error
		layoutDir, err = DefaultLayoutDir()
		if err != nil {
			return "", err
		}
	}
	if err := os.MkdirAll(layoutDir, 0o755); err != nil {
		return "", fmt.Errorf("create zellij layout dir: %w", err)
	}
	fileName := sanitizeLayoutName(name) + ".kdl"
	path := filepath.Join(layoutDir, fileName)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return "", fmt.Errorf("write zellij layout: %w", err)
	}
	return path, nil
}

func sanitizeLayoutName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "layout"
	}
	out := strings.Builder{}
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z':
			out.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			out.WriteRune(r + ('a' - 'A'))
		case r >= '0' && r <= '9':
			out.WriteRune(r)
		case r == '-' || r == '_' || r == '.':
			out.WriteRune(r)
		default:
			out.WriteRune('-')
		}
	}
	result := strings.Trim(out.String(), "-")
	if result == "" {
		return "layout"
	}
	return result
}
