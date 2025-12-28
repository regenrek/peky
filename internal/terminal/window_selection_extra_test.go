package terminal

import (
	"testing"

	uv "github.com/charmbracelet/ultraviolet"
)

func TestNormalizeSelectionSwap(t *testing.T) {
	sx, sy, ex, ey := normalizeSelection(5, 3, 1, 2)
	if sx != 1 || ex != 5 || sy != 2 || ey != 3 {
		t.Fatalf("expected selection swapped, got %d,%d to %d,%d", sx, sy, ex, ey)
	}
}

func TestExtractTextAndLineText(t *testing.T) {
	emu := &fakeEmu{
		cols: 4,
		rows: 1,
		screen: [][]uv.Cell{
			{
				{Content: "A", Width: 1},
				{Content: "", Width: 0},
				{Content: "B", Width: 1},
				{Content: "", Width: 1},
			},
		},
	}
	w := &Window{
		term: emu,
		cols: 4,
		rows: 1,
	}
	got := w.extractText(0, 0, 3, 0)
	if got != "AB" {
		t.Fatalf("expected selection text 'AB', got %q", got)
	}

	if envKey("NOVALUE") != "" {
		t.Fatalf("expected empty envKey for missing '='")
	}
}
