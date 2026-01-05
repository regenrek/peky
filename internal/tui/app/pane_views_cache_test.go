package app

import (
	"strings"
	"testing"

	"github.com/charmbracelet/colorprofile"

	"github.com/regenrek/peakypanes/internal/termframe"
)

func TestPaneViewCachesByCursorState(t *testing.T) {
	m := newTestModelLite()
	m.paneViewProfile = colorprofile.TrueColor

	frame := termframe.Frame{
		Cols: 2,
		Rows: 1,
		Cells: []termframe.Cell{
			{Content: "A", Width: 1},
			{Content: "B", Width: 1},
		},
		Cursor: termframe.Cursor{X: 0, Y: 0, Visible: true},
	}
	key := paneViewKey{PaneID: "p1", Cols: 2, Rows: 1}
	m.paneViews[key] = paneViewEntry{frame: frame}

	plain := m.paneView("p1", 2, 1, false)
	if plain == "" || strings.Contains(plain, "\x1b[") {
		t.Fatalf("expected plain rendered view, got %q", plain)
	}
	cursor := m.paneView("p1", 2, 1, true)
	if cursor == "" || cursor == plain {
		t.Fatalf("expected distinct cursor view")
	}
	entry := m.paneViews[key]
	if len(entry.rendered) != 2 {
		t.Fatalf("expected two cached variants, got %#v", entry.rendered)
	}
	if entry.rendered[false] != plain || entry.rendered[true] != cursor {
		t.Fatalf("unexpected cached entries")
	}
}
