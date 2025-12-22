package zellijctl

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type sessionMetadata struct {
	Path      string    `json:"path"`
	UpdatedAt time.Time `json:"updated_at"`
}

type sessionMetadataStore struct {
	Sessions map[string]sessionMetadata `json:"sessions"`
}

// RecordSessionPath stores metadata for a session created by Peaky Panes.
func RecordSessionPath(session, path string) error {
	if session == "" {
		return fmt.Errorf("session name is required")
	}
	metaPath, err := DefaultSessionMetadataPath()
	if err != nil {
		return err
	}
	store, _ := loadSessionMetadata(metaPath)
	if store.Sessions == nil {
		store.Sessions = make(map[string]sessionMetadata)
	}
	store.Sessions[session] = sessionMetadata{Path: path, UpdatedAt: time.Now()}
	return saveSessionMetadata(metaPath, store)
}

// LoadSessionPaths returns the stored session->path mappings.
func LoadSessionPaths() (map[string]string, error) {
	metaPath, err := DefaultSessionMetadataPath()
	if err != nil {
		return nil, err
	}
	store, err := loadSessionMetadata(metaPath)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]string{}, nil
		}
		return nil, err
	}
	out := make(map[string]string)
	for name, meta := range store.Sessions {
		if meta.Path != "" {
			out[name] = meta.Path
		}
	}
	return out, nil
}

// DefaultSessionMetadataPath returns the metadata path for zellij sessions.
func DefaultSessionMetadataPath() (string, error) {
	dir, err := DefaultBridgeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "sessions.json"), nil
}

func loadSessionMetadata(path string) (sessionMetadataStore, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return sessionMetadataStore{}, err
	}
	var store sessionMetadataStore
	if err := json.Unmarshal(data, &store); err != nil {
		return sessionMetadataStore{}, fmt.Errorf("parse session metadata: %w", err)
	}
	return store, nil
}

func saveSessionMetadata(path string, store sessionMetadataStore) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create metadata dir: %w", err)
	}
	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return fmt.Errorf("encode session metadata: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write session metadata: %w", err)
	}
	return nil
}
