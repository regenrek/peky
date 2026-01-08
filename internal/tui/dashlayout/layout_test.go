package dashlayout

import "testing"

func TestClampColumns(t *testing.T) {
	start, end, sel := ClampColumns(10, 5, 80, 2)
	if start < 0 || end <= start || sel < 0 {
		t.Fatalf("ClampColumns=%d..%d sel=%d", start, end, sel)
	}
	start, end, sel = ClampColumns(2, 1, 80, 2)
	if start != 0 || end != 2 || sel != 1 {
		t.Fatalf("ClampColumns small=%d..%d sel=%d", start, end, sel)
	}
}

func TestColumnWidth(t *testing.T) {
	if got := ColumnWidth(0, 1, 2); got != 1 {
		t.Fatalf("ColumnWidth=%d", got)
	}
	if got := ColumnWidth(10, 2, 2); got < 1 {
		t.Fatalf("ColumnWidth=%d", got)
	}
}

func TestPaneRange(t *testing.T) {
	start, end := PaneRange(true, 5, 3, 10)
	if start != 3 || end != 6 {
		t.Fatalf("PaneRange=%d..%d", start, end)
	}
}
