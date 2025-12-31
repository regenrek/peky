package sessionpolicy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateSessionName(t *testing.T) {
	name, err := ValidateSessionName("  demo ")
	if err != nil {
		t.Fatalf("expected valid name: %v", err)
	}
	if name != "demo" {
		t.Fatalf("expected trimmed name, got %q", name)
	}

	if _, err := ValidateSessionName(""); err == nil {
		t.Fatalf("expected error for empty name")
	}

	if _, err := ValidateSessionName("bad\x00name"); err == nil {
		t.Fatalf("expected error for control chars")
	}
}

func TestValidateOptionalSessionName(t *testing.T) {
	name, err := ValidateOptionalSessionName("  ")
	if err != nil {
		t.Fatalf("expected nil error for empty optional name: %v", err)
	}
	if name != "" {
		t.Fatalf("expected empty name, got %q", name)
	}

	if _, err := ValidateOptionalSessionName("bad\x00"); err == nil {
		t.Fatalf("expected error for invalid optional name")
	}
}

func TestValidatePaneIndex(t *testing.T) {
	idx, err := ValidatePaneIndex(" 1 ")
	if err != nil {
		t.Fatalf("expected valid pane index: %v", err)
	}
	if idx != "1" {
		t.Fatalf("expected trimmed index, got %q", idx)
	}

	if _, err := ValidatePaneIndex("  "); err == nil {
		t.Fatalf("expected error for empty pane index")
	}
}

func TestValidatePath(t *testing.T) {
	root := t.TempDir()
	abs, err := ValidatePath(root)
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
	if _, err := ValidatePath(filePath); err == nil {
		t.Fatalf("expected error for file path")
	}

	missing := filepath.Join(root, "missing")
	if _, err := ValidatePath(missing); err == nil {
		t.Fatalf("expected error for missing path")
	}

	if _, err := ValidatePath(strings.TrimSpace("")); err == nil {
		t.Fatalf("expected error for empty path")
	}
}

func TestValidateEnvList(t *testing.T) {
	entries, err := ValidateEnvList([]string{" FOO=bar ", "BAZ=qux", "EMPTY="})
	if err != nil {
		t.Fatalf("expected valid env list: %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}
	if entries[0] != "FOO=bar" {
		t.Fatalf("expected trimmed entry, got %q", entries[0])
	}

	if _, err := ValidateEnvList([]string{"NOEQUALS"}); err == nil {
		t.Fatalf("expected error for missing equals")
	}
	if _, err := ValidateEnvList([]string{"=VALUE"}); err == nil {
		t.Fatalf("expected error for missing key")
	}
	if _, err := ValidateEnvList([]string{"1BAD=VALUE"}); err == nil {
		t.Fatalf("expected error for invalid key")
	}
}
