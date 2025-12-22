package zellijctl

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnsureConfigWithBridgeAddsPluginOnce(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmp, "xdg"))

	baseConfig := filepath.Join(tmp, "config.kdl")
	if err := os.WriteFile(baseConfig, []byte("keybinds {}\n"), 0o644); err != nil {
		t.Fatalf("write base config: %v", err)
	}

	pluginPath := filepath.Join(tmp, "plugins", "bridge.wasm")
	if err := os.MkdirAll(filepath.Dir(pluginPath), 0o755); err != nil {
		t.Fatalf("mkdir plugin dir: %v", err)
	}
	if err := os.WriteFile(pluginPath, []byte("wasm"), 0o644); err != nil {
		t.Fatalf("write plugin: %v", err)
	}

	outPath, err := EnsureConfigWithBridge(baseConfig, pluginPath)
	if err != nil {
		t.Fatalf("EnsureConfigWithBridge: %v", err)
	}
	content, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read output config: %v", err)
	}

	pluginURL := "file:" + filepath.ToSlash(pluginPath)
	if !strings.Contains(string(content), pluginURL) {
		t.Fatalf("expected plugin url %q in output config", pluginURL)
	}

	// Run again to confirm it does not duplicate.
	outPath, err = EnsureConfigWithBridge(baseConfig, pluginPath)
	if err != nil {
		t.Fatalf("EnsureConfigWithBridge second run: %v", err)
	}
	content, err = os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read output config second run: %v", err)
	}
	if strings.Count(string(content), pluginURL) != 1 {
		t.Fatalf("expected plugin url once, got %d", strings.Count(string(content), pluginURL))
	}
}

func TestEnsureConfigWithBridgeUsesEnvDefault(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmp, "xdg"))

	baseConfig := filepath.Join(tmp, "zellij-config.kdl")
	if err := os.WriteFile(baseConfig, []byte("keybinds {}\n"), 0o644); err != nil {
		t.Fatalf("write base config: %v", err)
	}
	t.Setenv("ZELLIJ_CONFIG_FILE", baseConfig)

	pluginPath := filepath.Join(tmp, "bridge.wasm")
	if err := os.WriteFile(pluginPath, []byte("wasm"), 0o644); err != nil {
		t.Fatalf("write plugin: %v", err)
	}

	outPath, err := EnsureConfigWithBridge("", pluginPath)
	if err != nil {
		t.Fatalf("EnsureConfigWithBridge: %v", err)
	}
	content, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read output config: %v", err)
	}

	pluginURL := "file:" + filepath.ToSlash(pluginPath)
	if !strings.Contains(string(content), pluginURL) {
		t.Fatalf("expected plugin url %q in output config", pluginURL)
	}
}

func TestDefaultZellijLayoutDirUsesEnv(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("ZELLIJ_CONFIG_DIR", filepath.Join(tmp, "zellij"))

	dir, err := DefaultZellijLayoutDir()
	if err != nil {
		t.Fatalf("DefaultZellijLayoutDir: %v", err)
	}
	expected := filepath.Join(filepath.Join(tmp, "zellij"), "layouts")
	if dir != expected {
		t.Fatalf("expected %q, got %q", expected, dir)
	}
}
