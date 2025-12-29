package state

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ErrUnknownSchema is returned when a persisted state uses an unsupported version.
var ErrUnknownSchema = errors.New("sessiond: unknown state schema")

// Load reads persisted runtime state from disk.
func Load(path string) (*RuntimeState, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, errors.New("sessiond: state path is required")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("sessiond: read state: %w", err)
	}
	if len(data) == 0 {
		return nil, nil
	}
	var st RuntimeState
	if err := json.Unmarshal(data, &st); err != nil {
		return nil, fmt.Errorf("sessiond: decode state: %w", err)
	}
	if st.SchemaVersion != CurrentSchemaVersion {
		return nil, ErrUnknownSchema
	}
	st.Normalize()
	return &st, nil
}

// SaveAtomic writes bytes to disk using an atomic rename.
func SaveAtomic(path string, data []byte, perm os.FileMode) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return errors.New("sessiond: state path is required")
	}
	if perm == 0 {
		perm = 0o600
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("sessiond: create state dir: %w", err)
	}
	tmp, err := os.CreateTemp(dir, "state-*.tmp")
	if err != nil {
		return fmt.Errorf("sessiond: create state temp file: %w", err)
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
		return fmt.Errorf("sessiond: chmod state temp: %w", err)
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("sessiond: write state temp: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("sessiond: sync state temp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("sessiond: close state temp: %w", err)
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
		return fmt.Errorf("sessiond: replace state file: %w", err)
	}
	success = true
	_ = os.Chmod(path, perm)
	return nil
}
