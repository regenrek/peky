package atomicfile

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Save writes bytes to disk using an atomic rename.
func Save(path string, data []byte, perm os.FileMode) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return errors.New("atomicfile: path is required")
	}
	if perm == 0 {
		perm = 0o600
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("atomicfile: create dir: %w", err)
	}
	tmp, err := os.CreateTemp(dir, "atomic-*.tmp")
	if err != nil {
		return fmt.Errorf("atomicfile: create temp: %w", err)
	}
	name := tmp.Name()
	success := false
	defer func() {
		if !success {
			_ = os.Remove(name)
		}
	}()
	if err := tmp.Chmod(perm); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("atomicfile: chmod temp: %w", err)
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("atomicfile: write temp: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("atomicfile: sync temp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("atomicfile: close temp: %w", err)
	}
	if err := os.Rename(name, path); err != nil {
		if removeErr := os.Remove(path); removeErr == nil || os.IsNotExist(removeErr) {
			if retryErr := os.Rename(name, path); retryErr == nil {
				success = true
				_ = os.Chmod(path, perm)
				return nil
			} else {
				err = retryErr
			}
		}
		return fmt.Errorf("atomicfile: replace file: %w", err)
	}
	success = true
	_ = os.Chmod(path, perm)
	return nil
}
