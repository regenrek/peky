package pekyconfig

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDefaultsWhenMissing(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "missing.toml")
	loader := NewLoader(path)
	cfg, err := loader.Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.Agent.Model != defaultModel {
		t.Fatalf("Agent.Model=%q want %q", cfg.Agent.Model, defaultModel)
	}
	if cfg.Agent.Provider != defaultProvider {
		t.Fatalf("Agent.Provider=%q want %q", cfg.Agent.Provider, defaultProvider)
	}
	if cfg.QuickReply.Files.MaxDepth != defaultMaxDepth {
		t.Fatalf("Files.MaxDepth=%d want %d", cfg.QuickReply.Files.MaxDepth, defaultMaxDepth)
	}
	if cfg.QuickReply.Files.MaxItems != defaultMaxItems {
		t.Fatalf("Files.MaxItems=%d want %d", cfg.QuickReply.Files.MaxItems, defaultMaxItems)
	}
}

func TestLoadOverrides(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	data := []byte(`
[agent]
provider = "google"
model = "gemini-3-flash"

[quick_reply.files]
show_hidden = true
max_depth = 2
max_items = 99
`)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	loader := NewLoader(path)
	cfg, err := loader.Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.Agent.Model != "gemini-3-flash" {
		t.Fatalf("Agent.Model=%q", cfg.Agent.Model)
	}
	if cfg.Agent.Provider != "google" {
		t.Fatalf("Agent.Provider=%q", cfg.Agent.Provider)
	}
	if cfg.QuickReply.Files.MaxDepth != 2 {
		t.Fatalf("Files.MaxDepth=%d", cfg.QuickReply.Files.MaxDepth)
	}
	if cfg.QuickReply.Files.MaxItems != 99 {
		t.Fatalf("Files.MaxItems=%d", cfg.QuickReply.Files.MaxItems)
	}
	if cfg.QuickReply.Files.ShowHidden == nil || !*cfg.QuickReply.Files.ShowHidden {
		t.Fatalf("Files.ShowHidden=%v want true", cfg.QuickReply.Files.ShowHidden)
	}
}
