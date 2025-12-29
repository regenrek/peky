package devwatch

import "testing"

func TestNormalizeExts(t *testing.T) {
	got := normalizeExts([]string{"go", ".Go", "", "md", ".md"})
	if len(got) != 2 {
		t.Fatalf("expected 2 unique extensions, got %v", got)
	}
	if got[0] != ".go" {
		t.Fatalf("expected .go first, got %q", got[0])
	}
	if got[1] != ".md" {
		t.Fatalf("expected .md second, got %q", got[1])
	}
}

func TestNormalizeExtsDefaults(t *testing.T) {
	got := normalizeExts(nil)
	if len(got) == 0 {
		t.Fatalf("expected default extensions")
	}
}
