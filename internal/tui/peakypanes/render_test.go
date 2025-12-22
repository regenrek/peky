package peakypanes

import (
	"reflect"
	"strings"
	"testing"
)

func TestCompactPreviewLines(t *testing.T) {
	input := []string{"", "hello", "   ", "world", "\t", "done"}
	want := []string{"hello", "world", "done"}

	got := compactPreviewLines(input)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("compactPreviewLines() = %#v, want %#v", got, want)
	}
}

func TestOverlayCenteredBasic(t *testing.T) {
	base := "hello\nworld"
	overlay := "POP"
	out := overlayCentered(base, overlay, 12, 4)
	if !strings.Contains(out, "hello") {
		t.Fatalf("overlayCentered() missing base content: %q", out)
	}
	if !strings.Contains(out, "POP") {
		t.Fatalf("overlayCentered() missing overlay content: %q", out)
	}
}
