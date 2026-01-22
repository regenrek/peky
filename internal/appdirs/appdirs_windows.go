//go:build windows

package appdirs

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/regenrek/peakypanes/internal/identity"
	"github.com/regenrek/peakypanes/internal/runenv"
)

func RuntimeDir() (string, error) {
	dir, err := RuntimeDirPath()
	if err != nil {
		return "", err
	}
	return ensureRuntimeDir(dir)
}

// RuntimeDirPath returns the runtime directory path without creating it.
func RuntimeDirPath() (string, error) {
	if override := runenv.RuntimeDir(); override != "" {
		return override, nil
	}
	if local := strings.TrimSpace(os.Getenv("LOCALAPPDATA")); local != "" {
		return filepath.Join(local, identity.AppSlug), nil
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve config dir: %w", err)
	}
	return filepath.Join(dir, identity.AppSlug), nil
}

// DataDirPath returns the data directory path without creating it.
func DataDirPath() (string, error) {
	if override := runenv.DataDir(); override != "" {
		return override, nil
	}
	if local := strings.TrimSpace(os.Getenv("LOCALAPPDATA")); local != "" {
		return filepath.Join(local, identity.AppSlug), nil
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve config dir: %w", err)
	}
	return filepath.Join(dir, identity.AppSlug), nil
}

// DataDir returns the directory used for persistent data (snapshots, caches).
func DataDir() (string, error) {
	dir, err := DataDirPath()
	if err != nil {
		return "", err
	}
	return ensureDataDir(dir)
}

func ensureRuntimeDir(dir string) (string, error) {
	if dir == "" {
		return "", fmt.Errorf("runtime dir is empty")
	}
	info, err := os.Stat(dir)
	if err != nil {
		if !os.IsNotExist(err) {
			return "", fmt.Errorf("stat runtime dir: %w", err)
		}
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return "", fmt.Errorf("create runtime dir: %w", err)
		}
		return dir, nil
	}
	if !info.IsDir() {
		return "", fmt.Errorf("runtime dir %q is not a directory", dir)
	}
	return dir, nil
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
