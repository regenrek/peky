package filelist

import (
	"os"
	"path/filepath"
	"testing"
)

func TestListRespectsGitignoreAndHidden(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ".gitignore"), []byte("ignored.txt\n"), 0o600); err != nil {
		t.Fatalf("write gitignore: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "kept.txt"), []byte("ok"), 0o600); err != nil {
		t.Fatalf("write kept: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "ignored.txt"), []byte("nope"), 0o600); err != nil {
		t.Fatalf("write ignored: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, ".hidden.txt"), []byte("hidden"), 0o600); err != nil {
		t.Fatalf("write hidden: %v", err)
	}
	cfg := Config{MaxDepth: 2, MaxItems: 0, IncludeHidden: false}
	entries, _, err := List(root, cfg)
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if hasEntry(entries, ".hidden.txt") {
		t.Fatalf("expected hidden file filtered")
	}
	if hasEntry(entries, "ignored.txt") {
		t.Fatalf("expected gitignored file filtered")
	}
	if !hasEntry(entries, "kept.txt") {
		t.Fatalf("expected kept.txt in results")
	}
}

func TestListIncludesHiddenWhenConfigured(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ".hidden.txt"), []byte("hidden"), 0o600); err != nil {
		t.Fatalf("write hidden: %v", err)
	}
	cfg := Config{IncludeHidden: true}
	entries, _, err := List(root, cfg)
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if !hasEntry(entries, ".hidden.txt") {
		t.Fatalf("expected hidden file included")
	}
}

func TestListRespectsDepthAndLimit(t *testing.T) {
	root := t.TempDir()
	sub := filepath.Join(root, "dir")
	if err := os.MkdirAll(sub, 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sub, "file.txt"), []byte("ok"), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}
	cfg := Config{MaxDepth: 1, MaxItems: 1}
	entries, truncated, err := List(root, cfg)
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if !truncated {
		t.Fatalf("expected truncated results")
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
}

func hasEntry(entries []Entry, path string) bool {
	for _, entry := range entries {
		if entry.Path == path {
			return true
		}
	}
	return false
}
