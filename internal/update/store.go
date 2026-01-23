package update

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/regenrek/peakypanes/internal/runenv"
)

var ErrStateDisabled = errors.New("update state disabled")

// Store loads and saves update state.
type Store interface {
	Load(ctx context.Context) (State, error)
	Save(ctx context.Context, state State) error
}

// FileStore persists update state to a JSON file.
type FileStore struct {
	Path string
}

// DefaultStatePath resolves the update state file path.
func DefaultStatePath() (string, error) {
	if runenv.FreshConfigEnabled() {
		return "", ErrStateDisabled
	}
	if dir := runenv.ConfigDir(); dir != "" {
		return filepath.Join(dir, "update-state.json"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "peakypanes", "update-state.json"), nil
}

// Load reads update state from disk.
func (s FileStore) Load(ctx context.Context) (State, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return State{}, err
	}
	path, err := cleanStatePath(s.Path)
	if err != nil {
		return State{}, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return State{}, nil
		}
		return State{}, fmt.Errorf("read update state: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return State{}, err
	}
	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return State{}, fmt.Errorf("parse update state: %w", err)
	}
	return state, nil
}

// Save writes update state to disk atomically.
func (s FileStore) Save(ctx context.Context, state State) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	path, err := cleanStatePath(s.Path)
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if dir == "." || dir == "" {
		return fmt.Errorf("update state path missing directory")
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create update state dir: %w", err)
	}
	payload, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal update state: %w", err)
	}
	payload = append(payload, '\n')
	if err := ctx.Err(); err != nil {
		return err
	}
	tempFile, err := os.CreateTemp(dir, "update-state-*.json")
	if err != nil {
		return fmt.Errorf("create update state temp file: %w", err)
	}
	tempName := tempFile.Name()
	defer func() {
		_ = tempFile.Close()
		_ = os.Remove(tempName)
	}()
	if _, err := tempFile.Write(payload); err != nil {
		return fmt.Errorf("write update state temp file: %w", err)
	}
	if err := tempFile.Sync(); err != nil {
		return fmt.Errorf("sync update state temp file: %w", err)
	}
	if err := tempFile.Close(); err != nil {
		return fmt.Errorf("close update state temp file: %w", err)
	}
	if err := os.Rename(tempName, path); err != nil {
		return fmt.Errorf("rename update state temp file: %w", err)
	}
	return nil
}

func cleanStatePath(path string) (string, error) {
	if path == "" {
		return "", ErrStateDisabled
	}
	cleaned := filepath.Clean(path)
	if !filepath.IsAbs(cleaned) {
		return "", fmt.Errorf("update state path must be absolute")
	}
	return cleaned, nil
}
