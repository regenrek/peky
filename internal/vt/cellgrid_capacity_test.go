package vt

import "testing"

func TestCellGridResizeZeroFreesBacking(t *testing.T) {
	g := newCellGrid(80, 24)
	if g.stride == 0 || g.capRows == 0 || len(g.cells) == 0 {
		t.Fatalf("expected allocated backing; stride=%d capRows=%d len=%d", g.stride, g.capRows, len(g.cells))
	}

	g.Resize(0, 0)
	if g.cols != 0 || g.rows != 0 {
		t.Fatalf("expected empty dims; cols=%d rows=%d", g.cols, g.rows)
	}
	if g.stride != 0 || g.capRows != 0 || g.cells != nil {
		t.Fatalf("expected backing freed; stride=%d capRows=%d cellsNil=%v", g.stride, g.capRows, g.cells == nil)
	}
}

func TestCellGridResizeWithinCapacityRetainsBacking(t *testing.T) {
	g := newCellGrid(80, 24)
	oldStride := g.stride
	oldCapRows := g.capRows
	oldLen := len(g.cells)
	oldFirst := &g.cells[0]

	g.Resize(79, 23)

	if g.stride != oldStride || g.capRows != oldCapRows {
		t.Fatalf("expected capacity retained; stride=%d/%d capRows=%d/%d", g.stride, oldStride, g.capRows, oldCapRows)
	}
	if len(g.cells) != oldLen {
		t.Fatalf("expected backing len retained; len=%d/%d", len(g.cells), oldLen)
	}
	if &g.cells[0] != oldFirst {
		t.Fatalf("expected backing reused")
	}
}

func TestCellGridResizeShrinkTriggersCompaction(t *testing.T) {
	g := newCellGrid(200, 200)
	if g.stride != 200 || g.capRows != 200 {
		t.Fatalf("unexpected initial capacity; stride=%d capRows=%d", g.stride, g.capRows)
	}

	g.Resize(20, 20)
	if g.stride != 24 || g.capRows != 24 {
		t.Fatalf("expected compacted capacity; stride=%d capRows=%d", g.stride, g.capRows)
	}
	if len(g.cells) != 24*24 {
		t.Fatalf("expected compacted backing len; len=%d", len(g.cells))
	}
}

func TestCellGridCompactShrinksCapacity(t *testing.T) {
	g := newCellGrid(80, 24)
	g.Resize(81, 25) // force larger backing
	if g.stride != 88 || g.capRows != 32 {
		t.Fatalf("unexpected grown capacity; stride=%d capRows=%d", g.stride, g.capRows)
	}

	g.Resize(80, 24) // within capacity; should not auto-shrink
	if g.stride != 88 || g.capRows != 32 {
		t.Fatalf("expected capacity retained pre-compact; stride=%d capRows=%d", g.stride, g.capRows)
	}

	g.Compact()
	if g.stride != 80 || g.capRows != 24 {
		t.Fatalf("expected compacted capacity; stride=%d capRows=%d", g.stride, g.capRows)
	}
	if len(g.cells) != 80*24 {
		t.Fatalf("expected compacted backing len; len=%d", len(g.cells))
	}
}
