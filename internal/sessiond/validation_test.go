package sessiond

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateSessionName(t *testing.T) {
	name, err := validateSessionName("  demo ")
	if err != nil {
		t.Fatalf("expected valid name: %v", err)
	}
	if name != "demo" {
		t.Fatalf("expected trimmed name, got %q", name)
	}

	if _, err := validateSessionName(""); err == nil {
		t.Fatalf("expected error for empty name")
	}

	if _, err := validateSessionName("bad\x00name"); err == nil {
		t.Fatalf("expected error for control chars")
	}
}

func TestValidateOptionalSessionName(t *testing.T) {
	name, err := validateOptionalSessionName("  ")
	if err != nil {
		t.Fatalf("expected nil error for empty optional name: %v", err)
	}
	if name != "" {
		t.Fatalf("expected empty name, got %q", name)
	}

	if _, err := validateOptionalSessionName("bad\x00"); err == nil {
		t.Fatalf("expected error for invalid optional name")
	}
}

func TestValidatePaneIndex(t *testing.T) {
	idx, err := validatePaneIndex(" 1 ")
	if err != nil {
		t.Fatalf("expected valid pane index: %v", err)
	}
	if idx != "1" {
		t.Fatalf("expected trimmed index, got %q", idx)
	}

	if _, err := validatePaneIndex("  "); err == nil {
		t.Fatalf("expected error for empty pane index")
	}
}

func TestValidatePath(t *testing.T) {
	root := t.TempDir()
	abs, err := validatePath(root)
	if err != nil {
		t.Fatalf("expected valid path: %v", err)
	}
	if !filepath.IsAbs(abs) {
		t.Fatalf("expected absolute path, got %q", abs)
	}

	filePath := filepath.Join(root, "file.txt")
	if err := os.WriteFile(filePath, []byte("x"), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}
	if _, err := validatePath(filePath); err == nil {
		t.Fatalf("expected error for file path")
	}

	missing := filepath.Join(root, "missing")
	if _, err := validatePath(missing); err == nil {
		t.Fatalf("expected error for missing path")
	}

	if _, err := validatePath(strings.TrimSpace("")); err == nil {
		t.Fatalf("expected error for empty path")
	}
}
