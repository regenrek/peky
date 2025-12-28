package sessiond

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/regenrek/peakypanes/internal/pathutil"
)

func validateSessionName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", errors.New("session name is required")
	}
	for _, r := range name {
		if r == 0 || unicode.IsControl(r) {
			return "", fmt.Errorf("invalid session name %q", name)
		}
	}
	return name, nil
}

func validateOptionalSessionName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", nil
	}
	return validateSessionName(name)
}

func validatePaneIndex(index string) (string, error) {
	index = strings.TrimSpace(index)
	if index == "" {
		return "", errors.New("pane index is required")
	}
	return index, nil
}

func validatePath(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", errors.New("path is required")
	}
	path = pathutil.ExpandUser(path)
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve path %q: %w", path, err)
	}
	info, err := os.Stat(abs)
	if err != nil {
		return "", fmt.Errorf("stat %q: %w", abs, err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("%s is not a directory", abs)
	}
	return abs, nil
}
