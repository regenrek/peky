package zellijctl

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteLayoutFileAndSanitize(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmp, "xdg"))

	layoutDir, err := DefaultLayoutDir()
	if err != nil {
		t.Fatalf("DefaultLayoutDir: %v", err)
	}

	name := "My Layout@1"
	content := "layout {}"
	path, err := WriteLayoutFile(layoutDir, name, content)
	if err != nil {
		t.Fatalf("WriteLayoutFile: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("layout file missing: %v", err)
	}
	if filepath.Base(path) != sanitizeLayoutName(name)+".kdl" {
		t.Fatalf("unexpected layout filename: %s", filepath.Base(path))
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read layout file: %v", err)
	}
	if string(got) != content {
		t.Fatalf("layout content mismatch: %q", string(got))
	}
}

func TestWriteLayoutFileErrors(t *testing.T) {
	if _, err := WriteLayoutFile("", "", ""); err == nil {
		t.Fatalf("expected error for empty layout content and name")
	}
	if _, err := WriteLayoutFile("", "name", ""); err == nil {
		t.Fatalf("expected error for empty content")
	}
	if _, err := WriteLayoutFile("", "", "layout {}"); err == nil {
		t.Fatalf("expected error for empty name")
	}
}
