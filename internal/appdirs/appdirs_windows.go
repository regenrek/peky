//go:build windows

package appdirs

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/regenrek/peakypanes/internal/identity"
	"github.com/regenrek/peakypanes/internal/runenv"
)

func RuntimeDir() (string, error) {
	return "", errors.New("runtime dirs are not supported on windows yet")
}

// DataDir returns the directory used for persistent data (snapshots, caches).
func DataDir() (string, error) {
	if override := runenv.DataDir(); override != "" {
		return ensureDataDir(override)
	}
	if local := strings.TrimSpace(os.Getenv("LOCALAPPDATA")); local != "" {
		return ensureDataDir(filepath.Join(local, identity.AppSlug))
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve config dir: %w", err)
	}
	return ensureDataDir(filepath.Join(dir, identity.AppSlug))
}

func ensureDataDir(dir string) (string, error) {
	if dir == "" {
		return "", fmt.Errorf("data dir is empty")
	}
	info, err := os.Stat(dir)
	if err != nil {
		if !os.IsNotExist(err) {
			return "", fmt.Errorf("stat data dir: %w", err)
		}
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return "", fmt.Errorf("create data dir: %w", err)
		}
		return dir, nil
	}
	if !info.IsDir() {
		return "", fmt.Errorf("data dir %q is not a directory", dir)
	}
	return dir, nil
}
