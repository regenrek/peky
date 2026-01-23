package views

import (
	"strings"
	"testing"
)

func TestResizeOverlayAndContextMenu(t *testing.T) {
	base := ""
	resize := ResizeOverlay{
		Active:      true,
		SnapEnabled: true,
		SnapActive:  true,
		ModeKey:     "r",
		EdgeLabel:   "right",
		Label:       "80x24",
		LabelX:      2,
		LabelY:      1,
	}

	withLabel := overlayResizeLabel(base, 40, 10, resize)
	if !strings.Contains(withLabel, "80x24") {
		t.Fatalf("expected resize label in overlay")
	}
	withHUD := overlayResizeHUD(base, 40, 10, 1, 1, 7, resize)
	if !strings.Contains(withHUD, "Resize mode") {
		t.Fatalf("expected resize hud content")
	}
	if renderResizeHUD(ResizeOverlay{}) != "" {
		t.Fatalf("expected empty hud when inactive")
	}
	if renderResizeLabel(" ") != "" {
		t.Fatalf("expected empty label for blank text")
	}

	menu := ContextMenu{
		Items: []ContextMenuItem{
			{Label: "Open", Enabled: true},
			{Label: "Delete", Enabled: false},
		},
		Selected: 1,
	}
	out := renderContextMenu(menu)
	if !strings.Contains(out, "Open") || !strings.Contains(out, "Delete") {
		t.Fatalf("expected menu labels")
	}
	if width := contextMenuWidth(menu.Items); width < 10 {
		t.Fatalf("expected menu width >= 10")
	}
}

func TestRightAlignLine(t *testing.T) {
	if out := rightAlignLine("hi", 1); out == "" {
		t.Fatalf("expected aligned output")
	}
	if out := rightAlignLine("hi", 4); !strings.HasSuffix(out, "hi") {
		t.Fatalf("expected suffix alignment")
	}
}
