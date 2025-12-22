package peakypanes

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNextSessionName(t *testing.T) {
	existing := []string{"myapp", "myapp-2", "myapp-3"}
	got := nextSessionName("myapp", existing)
	if got != "myapp-4" {
		t.Fatalf("nextSessionName() = %q", got)
	}
}

func TestValidateProjectPath(t *testing.T) {
	if err := validateProjectPath(""); err != nil {
		t.Fatalf("validateProjectPath(empty) error: %v", err)
	}

	file := filepath.Join(t.TempDir(), "file.txt")
	if err := os.WriteFile(file, []byte("data"), 0o644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	if err := validateProjectPath(file); err == nil {
		t.Fatalf("validateProjectPath(file) expected error")
	}

	missing := filepath.Join(t.TempDir(), "missing")
	if err := validateProjectPath(missing); err == nil {
		t.Fatalf("validateProjectPath(missing) expected error")
	}
}

func TestSelfExecutableNotEmpty(t *testing.T) {
	if exe := selfExecutable(); exe == "" {
		t.Fatalf("selfExecutable() returned empty")
	}
}
