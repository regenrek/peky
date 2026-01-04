//go:build !windows

package logging

import (
	"fmt"
	"log/slog"
	"os"
	"sync"
	"syscall"
)

var logDirPermsWarnOnce sync.Once

func ensureLogDir(dir string, isOverride bool) error {
	if dir == "" || dir == "." {
		return nil
	}
	info, err := os.Stat(dir)
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("logging: stat log dir: %w", err)
		}
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return fmt.Errorf("logging: create log dir: %w", err)
		}
		return nil
	}
	if !info.IsDir() {
		return fmt.Errorf("logging: log dir %q is not a directory", dir)
	}
	mode := info.Mode().Perm()
	if mode&0o077 == 0 {
		return nil
	}
	if isOverride {
		logDirPermsWarnOnce.Do(func() {
			slog.Warn("log dir is group/world accessible; consider chmod 0700", "path", dir, "mode", mode.String())
		})
		return nil
	}
	if ownedByCurrentUser(info) {
		if err := os.Chmod(dir, 0o700); err != nil {
			return fmt.Errorf("logging: chmod log dir: %w", err)
		}
		return nil
	}
	logDirPermsWarnOnce.Do(func() {
		slog.Warn("log dir is not owned by current user; permissions unchanged", "path", dir, "mode", mode.String())
	})
	return nil
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
