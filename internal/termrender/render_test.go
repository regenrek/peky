package termrender

import (
	"strings"
	"testing"

	"github.com/charmbracelet/colorprofile"
	"github.com/charmbracelet/x/ansi"

	"github.com/regenrek/peakypanes/internal/termframe"
)

func TestRenderPlainFrameNoCursor(t *testing.T) {
	frame := termframe.Frame{
		Cols: 2,
		Rows: 1,
		Cells: []termframe.Cell{
			{Content: "A", Width: 1},
			{Content: "B", Width: 1},
		},
	}
	out := Render(frame, Options{Profile: colorprofile.TrueColor, ShowCursor: false})
	if out != "AB" {
		t.Fatalf("expected plain output, got %q", out)
	}
	if strings.Contains(out, "\x1b[") {
		t.Fatalf("unexpected ansi sequence in plain output")
	}
}

func TestRenderCursorAddsStyle(t *testing.T) {
	frame := termframe.Frame{
		Cols: 2,
		Rows: 1,
		Cells: []termframe.Cell{
			{Content: "A", Width: 1},
			{Content: "B", Width: 1},
		},
		Cursor: termframe.Cursor{X: 0, Y: 0, Visible: true},
	}
	out := Render(frame, Options{Profile: colorprofile.TrueColor, ShowCursor: true})
	if !strings.Contains(out, "A") {
		t.Fatalf("expected content in output, got %q", out)
	}
	if !strings.Contains(out, "\x1b[") || !strings.Contains(out, ansi.ResetStyle) {
		t.Fatalf("expected cursor styling in output, got %q", out)
	}
}

func TestRenderIncludesHyperlink(t *testing.T) {
	url := "https://example.com"
	frame := termframe.Frame{
		Cols: 1,
		Rows: 1,
		Cells: []termframe.Cell{
			{Content: "L", Width: 1, Link: termframe.Link{URL: url}},
		},
	}
	out := Render(frame, Options{Profile: colorprofile.TrueColor, ShowCursor: false})
	if !strings.Contains(out, ansi.SetHyperlink(url, "")) {
		t.Fatalf("expected hyperlink start in output")
	}
	if !strings.Contains(out, ansi.ResetHyperlink()) {
		t.Fatalf("expected hyperlink reset in output")
	}
}
