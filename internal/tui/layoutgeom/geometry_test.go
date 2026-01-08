package layoutgeom

import (
	"testing"

	"github.com/regenrek/peakypanes/internal/layout"
	"github.com/regenrek/peakypanes/internal/tui/mouse"
)

func TestDividerCellsJunctions(t *testing.T) {
	segments := []DividerSegment{
		{Axis: SegmentVertical, Rect: mouse.Rect{X: 2, Y: 0, W: 1, H: 3}},
		{Axis: SegmentHorizontal, Rect: mouse.Rect{X: 0, Y: 1, W: 5, H: 1}},
	}
	cells := DividerCells(segments)
	if len(cells) == 0 {
		t.Fatalf("expected divider cells")
	}
	glyphs := make(map[[2]int]rune)
	for _, cell := range cells {
		glyphs[[2]int{cell.X, cell.Y}] = cell.Rune
	}
	if got := glyphs[[2]int{2, 1}]; got != '┼' {
		t.Fatalf("junction rune = %q", got)
	}
	if got := glyphs[[2]int{2, 0}]; got != '│' {
		t.Fatalf("vertical rune = %q", got)
	}
	if got := glyphs[[2]int{0, 1}]; got != '─' {
		t.Fatalf("horizontal rune = %q", got)
	}
}

func TestBuildGeometryDividers(t *testing.T) {
	rects := map[string]layout.Rect{
		"p1": {X: 0, Y: 0, W: 500, H: 1000},
		"p2": {X: 500, Y: 0, W: 500, H: 1000},
	}
	geom, ok := Build(mouse.Rect{X: 0, Y: 0, W: 80, H: 20}, rects)
	if !ok {
		t.Fatalf("expected geometry")
	}
	if len(geom.Dividers) == 0 {
		t.Fatalf("expected divider segments")
	}
}

func TestBuildGeometryFrameCorners(t *testing.T) {
	rects := map[string]layout.Rect{
		"p1": {X: 0, Y: 0, W: 1000, H: 1000},
	}
	preview := mouse.Rect{X: 0, Y: 0, W: 10, H: 6}
	geom, ok := Build(preview, rects)
	if !ok {
		t.Fatalf("expected geometry")
	}
	cells := DividerCells(geom.Dividers)
	if len(cells) == 0 {
		t.Fatalf("expected divider cells")
	}
	glyphs := make(map[[2]int]rune, len(cells))
	for _, cell := range cells {
		glyphs[[2]int{cell.X, cell.Y}] = cell.Rune
	}
	topLeft := glyphs[[2]int{preview.X, preview.Y}]
	topRight := glyphs[[2]int{preview.X + preview.W - 1, preview.Y}]
	botLeft := glyphs[[2]int{preview.X, preview.Y + preview.H - 1}]
	botRight := glyphs[[2]int{preview.X + preview.W - 1, preview.Y + preview.H - 1}]
	if topLeft != '┌' {
		t.Fatalf("top-left rune = %q", topLeft)
	}
	if topRight != '┐' {
		t.Fatalf("top-right rune = %q", topRight)
	}
	if botLeft != '└' {
		t.Fatalf("bottom-left rune = %q", botLeft)
	}
	if botRight != '┘' {
		t.Fatalf("bottom-right rune = %q", botRight)
	}
}

func TestContentRectInsetsForFrame(t *testing.T) {
	rects := map[string]layout.Rect{
		"p1": {X: 0, Y: 0, W: 1000, H: 1000},
	}
	preview := mouse.Rect{X: 0, Y: 0, W: 12, H: 8}
	geom, ok := Build(preview, rects)
	if !ok {
		t.Fatalf("expected geometry")
	}
	outer := mouse.Rect{X: 0, Y: 0, W: 12, H: 8}
	content := ContentRect(geom, outer)
	want := mouse.Rect{X: 1, Y: 1, W: 10, H: 6}
	if content != want {
		t.Fatalf("content rect = %+v want %+v", content, want)
	}
}

func TestBuildGeometryInternalDividerStopsAtFrame(t *testing.T) {
	rects := map[string]layout.Rect{
		"p1": {X: 0, Y: 0, W: 500, H: 1000},
		"p2": {X: 500, Y: 0, W: 500, H: 1000},
	}
	preview := mouse.Rect{X: 0, Y: 0, W: 10, H: 6}
	geom, ok := Build(preview, rects)
	if !ok {
		t.Fatalf("expected geometry")
	}
	cells := DividerCells(geom.Dividers)
	glyphs := make(map[[2]int]rune, len(cells))
	for _, cell := range cells {
		glyphs[[2]int{cell.X, cell.Y}] = cell.Rune
	}

	dividerX := ScaleLayoutPos(preview.X, preview.W, 500)
	if dividerX < preview.X || dividerX >= preview.X+preview.W {
		t.Fatalf("divider x out of bounds: %d", dividerX)
	}
	bottom := preview.Y + preview.H - 1
	if got := glyphs[[2]int{dividerX, bottom}]; got != '┴' {
		t.Fatalf("bottom junction rune = %q", got)
	}
}

func TestBuildGeometryHorizontalDividerStopsAtVerticalDivider(t *testing.T) {
	rects := map[string]layout.Rect{
		"p1": {X: 0, Y: 0, W: 500, H: 500},
		"p2": {X: 0, Y: 500, W: 500, H: 500},
		"p3": {X: 500, Y: 0, W: 500, H: 1000},
	}
	preview := mouse.Rect{X: 0, Y: 0, W: 20, H: 10}
	geom, ok := Build(preview, rects)
	if !ok {
		t.Fatalf("expected geometry")
	}
	cells := DividerCells(geom.Dividers)
	glyphs := make(map[[2]int]rune, len(cells))
	for _, cell := range cells {
		glyphs[[2]int{cell.X, cell.Y}] = cell.Rune
	}

	dividerX := ScaleLayoutPos(preview.X, preview.W, 500)
	dividerY := ScaleLayoutPos(preview.Y, preview.H, 500)
	if dividerX < preview.X || dividerX >= preview.X+preview.W {
		t.Fatalf("divider x out of bounds: %d", dividerX)
	}
	if dividerY < preview.Y || dividerY >= preview.Y+preview.H {
		t.Fatalf("divider y out of bounds: %d", dividerY)
	}

	if got := glyphs[[2]int{dividerX, dividerY}]; got != '┤' {
		t.Fatalf("junction rune = %q", got)
	}
	if _, ok := glyphs[[2]int{dividerX + 1, dividerY}]; ok {
		t.Fatalf("unexpected divider cell to the right of junction at x=%d y=%d", dividerX+1, dividerY)
	}
}
