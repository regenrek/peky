package panelayout

import "testing"

func TestCompute(t *testing.T) {
	g := Compute(5, 100, 20)
	if g.Cols < 1 || g.Rows < 1 || g.TileWidth < 1 || g.BaseHeight < 1 {
		t.Fatalf("grid=%#v", g)
	}
	if g.RowHeight(g.Rows-1) != g.BaseHeight+g.ExtraHeight {
		t.Fatalf("last row height")
	}
	if g.RowY(0, 0) != 0 {
		t.Fatalf("row 0 y")
	}
}

func TestTileMetricsFor(t *testing.T) {
	m := TileMetricsFor(10, 5, TileBorders{Top: true, Left: true, Right: true, Bottom: true})
	if m.ContentWidth < 0 || m.InnerHeight < 0 {
		t.Fatalf("metrics=%#v", m)
	}
}

func TestDashboardTilePreviewLines(t *testing.T) {
	if got := DashboardTilePreviewLines(1, 10); got != 0 {
		t.Fatalf("preview=%d", got)
	}
	if got := DashboardTilePreviewLines(10, 3); got != 3 {
		t.Fatalf("preview=%d", got)
	}
}
