package clone

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestNormalizeRepoURL(t *testing.T) {
	if got := normalizeRepoURL("regenrek/peakypanes"); got != "https://github.com/regenrek/peakypanes" {
		t.Fatalf("got=%q", got)
	}
	if got := normalizeRepoURL("https://example.com/x"); got != "https://example.com/x" {
		t.Fatalf("got=%q", got)
	}
}

func TestExtractRepoName(t *testing.T) {
	if got := extractRepoName("https://github.com/regenrek/peakypanes.git"); got != "peakypanes" {
		t.Fatalf("got=%q", got)
	}
	if got := extractRepoName("x"); got != "x" {
		t.Fatalf("got=%q", got)
	}
}

func TestEnsureClonePath(t *testing.T) {
	if _, err := ensureClonePath(" "); err == nil {
		t.Fatalf("expected error for empty path")
	}
	got, err := ensureClonePath("a/b")
	if err != nil {
		t.Fatalf("ensureClonePath error: %v", err)
	}
	if !filepath.IsAbs(got) {
		t.Fatalf("expected abs, got %q", got)
	}
	if runtime.GOOS != "windows" && strings.Contains(got, "\\") {
		t.Fatalf("unexpected backslashes: %q", got)
	}
}
