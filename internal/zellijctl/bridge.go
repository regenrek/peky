package zellijctl

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
)

const bridgeFilename = "peakypanes-bridge.wasm"

// EnsureBridgePlugin writes the embedded bridge plugin to disk and returns its path.
func EnsureBridgePlugin() (string, error) {
	dir, err := DefaultBridgeDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create zellij bridge dir: %w", err)
	}
	path := filepath.Join(dir, bridgeFilename)
	if ok, err := fileMatchesHash(path, bridgeWasm); err == nil && ok {
		return path, nil
	}
	if err := os.WriteFile(path, bridgeWasm, 0o644); err != nil {
		return "", fmt.Errorf("write zellij bridge: %w", err)
	}
	return path, nil
}

// DefaultBridgeDir returns the directory used for the embedded zellij bridge plugin.
func DefaultBridgeDir() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "peakypanes", "zellij"), nil
}

func fileMatchesHash(path string, content []byte) (bool, error) {
	existing, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}
	return hashBytes(existing) == hashBytes(content), nil
}

func hashBytes(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
