package logo

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestRenderBasic(t *testing.T) {
	out := Render(0, false)
	if strings.TrimSpace(out) == "" {
		t.Fatalf("Render() returned empty output")
	}
	lines := strings.Split(out, "\n")
	if got := len(lines); got != FullHeight() {
		t.Fatalf("Render() lines = %d, want %d", got, FullHeight())
	}
	width := utf8.RuneCountInString(lines[0])
	if width != FullWidth() {
		t.Fatalf("Render() width = %d, want %d", width, FullWidth())
	}
}

func TestRenderTruncates(t *testing.T) {
	out := Render(10, false)
	lines := strings.Split(out, "\n")
	for _, line := range lines {
		if got := utf8.RuneCountInString(line); got > 10 {
			t.Fatalf("Render() line width = %d, want <= 10", got)
		}
	}
}

func TestSmallRender(t *testing.T) {
	out := SmallRender(0)
	if strings.TrimSpace(out) == "" {
		t.Fatalf("SmallRender() returned empty output")
	}
	if strings.Contains(out, "\n") {
		t.Fatalf("SmallRender() should be single-line")
	}
	if out != "PEKY" {
		t.Fatalf("SmallRender() = %q, want %q", out, "PEKY")
	}
}
