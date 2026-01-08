package contextpack

import (
	"strings"
	"testing"

	"github.com/regenrek/peakypanes/internal/cli/output"
)

func TestNormalizeIncludesDefaults(t *testing.T) {
	got := normalizeIncludes(nil)
	for _, key := range []string{"panes", "snapshot", "git", "errors"} {
		if !got[key] {
			t.Fatalf("missing %q", key)
		}
	}
}

func TestNormalizeIncludesExplicit(t *testing.T) {
	got := normalizeIncludes([]string{"git", "errors"})
	if got["panes"] || got["snapshot"] {
		t.Fatalf("unexpected snapshot defaults: %#v", got)
	}
	if !got["git"] || !got["errors"] {
		t.Fatalf("expected git+errors: %#v", got)
	}
}

func TestIncludeSnapshot(t *testing.T) {
	if !includeSnapshot(map[string]bool{"panes": true}) {
		t.Fatalf("panes should include snapshot")
	}
	if includeSnapshot(map[string]bool{"git": true}) {
		t.Fatalf("git should not include snapshot")
	}
}

func TestAtoi(t *testing.T) {
	cases := []struct {
		in   string
		want int
	}{
		{"", 0},
		{"  ", 0},
		{"1", 1},
		{"12", 12},
		{"x", 0},
		{"1x", 0},
	}
	for _, tc := range cases {
		if got := atoi(tc.in); got != tc.want {
			t.Fatalf("atoi(%q)=%d want %d", tc.in, got, tc.want)
		}
	}
}

func TestShrinkToFitTruncates(t *testing.T) {
	largeErrors := make([]string, 0, 200)
	for i := 0; i < 200; i++ {
		largeErrors = append(largeErrors, strings.Repeat("x", 50))
	}
	pack := output.ContextPack{Errors: largeErrors, MaxBytes: 200}
	shrunk := shrinkToFit(pack)
	if !shrunk.Truncated {
		t.Fatalf("expected truncated")
	}
	if shrunk.Errors != nil {
		t.Fatalf("expected errors dropped")
	}
}
