package termframe

import "testing"

func TestFrameCellAtAndEmpty(t *testing.T) {
	frame := Frame{
		Cols:  2,
		Rows:  2,
		Cells: make([]Cell, 4),
	}
	frame.Cells[3] = Cell{Content: "Z", Width: 1}

	if frame.Empty() {
		t.Fatalf("expected non-empty frame")
	}
	if frame.CellAt(1, 1) == nil || frame.CellAt(1, 1).Content != "Z" {
		t.Fatalf("expected cell at 1,1")
	}
	if frame.CellAt(-1, 0) != nil || frame.CellAt(0, -1) != nil {
		t.Fatalf("expected out of bounds to return nil")
	}
	if frame.CellAt(2, 0) != nil || frame.CellAt(0, 2) != nil {
		t.Fatalf("expected out of bounds to return nil")
	}

	empty := Frame{Cols: 0, Rows: 0}
	if !empty.Empty() {
		t.Fatalf("expected empty frame")
	}
}
