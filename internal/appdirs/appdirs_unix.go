//go:build !windows

package appdirs

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"

	"github.com/regenrek/peakypanes/internal/runenv"
	"log/slog"
)

var runtimePermsWarnOnce sync.Once
var dataPermsWarnOnce sync.Once

// RuntimeDir returns the directory used for runtime state (socket/pid/logs).
func RuntimeDir() (string, error) {
	if override := runenv.RuntimeDir(); override != "" {
		return ensureRuntimeDir(override, true)
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve config dir: %w", err)
	}
	dir = filepath.Join(dir, "peakypanes")
	return ensureRuntimeDir(dir, false)
}

// DataDir returns the directory used for persistent data (snapshots, caches).
func DataDir() (string, error) {
	if override := runenv.DataDir(); override != "" {
		return ensureDataDir(override, true)
	}
	if runtime.GOOS == "darwin" {
		dir, err := os.UserConfigDir()
		if err != nil {
			return "", fmt.Errorf("resolve config dir: %w", err)
		}
		dir = filepath.Join(dir, "peakypanes")
		return ensureDataDir(dir, false)
	}
	if xdg := strings.TrimSpace(os.Getenv("XDG_DATA_HOME")); xdg != "" {
		dir := filepath.Join(xdg, "peakypanes")
		return ensureDataDir(dir, false)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home dir: %w", err)
	}
	dir := filepath.Join(home, ".local", "share", "peakypanes")
	return ensureDataDir(dir, false)
}

func ensureRuntimeDir(dir string, isOverride bool) (string, error) {
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
	mode := info.Mode().Perm()
	if mode&0o077 == 0 {
		return dir, nil
	}
	if isOverride {
		runtimePermsWarnOnce.Do(func() {
			slog.Warn("runtime dir is group/world accessible; consider chmod 0700", "path", dir, "mode", mode.String())
		})
		return dir, nil
	}
	if ownedByCurrentUser(info) {
		if err := os.Chmod(dir, 0o700); err != nil {
			return "", fmt.Errorf("chmod runtime dir: %w", err)
		}
		return dir, nil
	}
	runtimePermsWarnOnce.Do(func() {
		slog.Warn("runtime dir is not owned by current user; permissions unchanged", "path", dir, "mode", mode.String())
	})
	return dir, nil
}

func ensureDataDir(dir string, isOverride bool) (string, error) {
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
	mode := info.Mode().Perm()
	if mode&0o077 == 0 {
		return dir, nil
	}
	if isOverride {
		dataPermsWarnOnce.Do(func() {
			slog.Warn("data dir is group/world accessible; consider chmod 0700", "path", dir, "mode", mode.String())
		})
		return dir, nil
	}
	if ownedByCurrentUser(info) {
		if err := os.Chmod(dir, 0o700); err != nil {
			return "", fmt.Errorf("chmod data dir: %w", err)
		}
		return dir, nil
	}
	dataPermsWarnOnce.Do(func() {
		slog.Warn("data dir is not owned by current user; permissions unchanged", "path", dir, "mode", mode.String())
	})
	return dir, nil
}

func ownedByCurrentUser(info os.FileInfo) bool {
	if info == nil {
		return false
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return false
	}
	return stat.Uid == uint32(os.Getuid())
}
