//go:build windows

package logging

import (
	"fmt"
	"os"
)

func ensureLogDir(dir string, _ bool) error {
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
	return nil
}
