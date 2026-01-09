//go:build linux

package terminal

import (
	"os"
	"path/filepath"
)

func pidCwd(pid int) (string, bool) {
	if pid <= 0 {
		return "", false
	}
	path, err := os.Readlink(filepath.Join("/proc", pidString(pid), "cwd"))
	if err != nil || path == "" {
		return "", false
	}
	return path, true
}
