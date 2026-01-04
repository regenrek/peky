//go:build !windows

package appdirs

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"syscall"

	"github.com/regenrek/peakypanes/internal/runenv"
	"log/slog"
)

var runtimePermsWarnOnce sync.Once

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
