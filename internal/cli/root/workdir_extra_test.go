package root

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNormalizeWorkDir(t *testing.T) {
	if _, err := normalizeWorkDir(" "); err == nil {
		t.Fatalf("expected error for empty")
	}
	dir := t.TempDir()
	got, err := normalizeWorkDir(dir)
	if err != nil {
		t.Fatalf("normalizeWorkDir error: %v", err)
	}
	if !filepath.IsAbs(got) {
		t.Fatalf("expected abs path, got %q", got)
	}
	file := filepath.Join(dir, "file.txt")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	if _, err := normalizeWorkDir(file); err == nil {
		t.Fatalf("expected error for file path")
	}
}

func TestResolveWorkDirPrecedence(t *testing.T) {
	depDir := t.TempDir()
	overrideDir := t.TempDir()
	ctx := CommandContext{Deps: Dependencies{WorkDir: depDir}}
	got, err := ResolveWorkDir(ctx)
	if err != nil {
		t.Fatalf("ResolveWorkDir error: %v", err)
	}
	if got != depDir {
		t.Fatalf("expected dep workdir, got %q", got)
	}
	ctx.WorkDir = overrideDir
	got, err = ResolveWorkDir(ctx)
	if err != nil {
		t.Fatalf("ResolveWorkDir override error: %v", err)
	}
	if got != overrideDir {
		t.Fatalf("expected override workdir, got %q", got)
	}
}
