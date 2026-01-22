package sessionpolicy

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/regenrek/peakypanes/internal/limits"
	"github.com/regenrek/peakypanes/internal/userpath"
)

func ValidateSessionName(name string) (string, error) {
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

func ValidateOptionalSessionName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", nil
	}
	return ValidateSessionName(name)
}

func ValidatePaneIndex(index string) (string, error) {
	index = strings.TrimSpace(index)
	if index == "" {
		return "", errors.New("pane index is required")
	}
	return index, nil
}

func ValidatePaneBackground(value int) (int, error) {
	if value < limits.PaneBackgroundMin || value > limits.PaneBackgroundMax {
		return 0, fmt.Errorf("pane background must be %d-%d", limits.PaneBackgroundMin, limits.PaneBackgroundMax)
	}
	return value, nil
}

func ValidatePath(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", errors.New("path is required")
	}
	path = userpath.ExpandUser(path)
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

// ValidateEnvList validates env entries as KEY=VALUE strings.
func ValidateEnvList(entries []string) ([]string, error) {
	if len(entries) == 0 {
		return nil, nil
	}
	out := make([]string, 0, len(entries))
	for _, entry := range entries {
		trimmed := strings.TrimSpace(entry)
		if trimmed == "" {
			continue
		}
		key, _, found := strings.Cut(trimmed, "=")
		key = strings.TrimSpace(key)
		if !found || key == "" {
			return nil, fmt.Errorf("invalid env entry %q", entry)
		}
		if err := validateEnvKey(key); err != nil {
			return nil, err
		}
		out = append(out, trimmed)
	}
	return out, nil
}

func ValidatePaneCount(count int) (int, error) {
	if count == 0 {
		return 0, nil
	}
	if count < 0 {
		return 0, errors.New("pane count must be positive")
	}
	if count > limits.MaxPanes {
		return 0, fmt.Errorf("pane count %d exceeds max %d", count, limits.MaxPanes)
	}
	return count, nil
}

func validateEnvKey(key string) error {
	for i, r := range key {
		if i == 0 {
			if r != '_' && !unicode.IsLetter(r) {
				return fmt.Errorf("invalid env key %q", key)
			}
			continue
		}
		if r != '_' && !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return fmt.Errorf("invalid env key %q", key)
		}
	}
	return nil
}
