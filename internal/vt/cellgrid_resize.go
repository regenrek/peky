package vt

import uv "github.com/charmbracelet/ultraviolet"

type cellGridState struct {
	cols    int
	rows    int
	stride  int
	capRows int
	cells   []uv.Cell
}

func (g *cellGrid) snapshot() cellGridState {
	return cellGridState{
		cols:    g.cols,
		rows:    g.rows,
		stride:  g.stride,
		capRows: g.capRows,
		cells:   g.cells,
	}
}

func sanitizeGridSize(cols, rows int) (int, int) {
	if cols < 0 {
		cols = 0
	}
	if rows < 0 {
		rows = 0
	}
	return cols, rows
}

func (g *cellGrid) resetBacking() {
	g.cells = nil
	g.stride = 0
	g.capRows = 0
}

func (g *cellGrid) Resize(cols, rows int) {
	cols, rows = sanitizeGridSize(cols, rows)
	if cols == g.cols && rows == g.rows {
		return
	}

	old := g.snapshot()
	g.cols = cols
	g.rows = rows

	if cols == 0 || rows == 0 {
		g.resetBacking()
		return
	}

	if g.canReuseBacking(old, cols, rows) {
		g.resizeWithinBacking(old, cols, rows)
		return
	}

	g.resizeRealloc(old, cols, rows)
}

func (g *cellGrid) canReuseBacking(old cellGridState, cols, rows int) bool {
	if cols > old.stride || rows > old.capRows {
		return false
	}
	if old.stride <= 0 || old.capRows <= 0 {
		return false
	}
	if len(old.cells) != old.stride*old.capRows {
		return false
	}
	return true
}

func (g *cellGrid) resizeWithinBacking(old cellGridState, cols, rows int) {
	if shouldShrinkCapacity(cols, rows, old.stride, old.capRows) {
		g.compactTo(cols, rows)
		return
	}

	g.stride = old.stride
	g.capRows = old.capRows
	g.cells = old.cells

	g.fillNewVisibleSpace(old, cols, rows)
}

func (g *cellGrid) fillNewVisibleSpace(old cellGridState, cols, rows int) {
	// Fill newly visible columns in existing rows.
	if cols > old.cols {
		maxRows := old.rows
		if rows < maxRows {
			maxRows = rows
		}
		for y := 0; y < maxRows; y++ {
			rowStart := y * g.stride
			row := g.cells[rowStart : rowStart+g.stride]
			fillEmptyCells(row[old.cols:cols])
		}
	}

	// Fill newly visible rows.
	if rows > old.rows {
		for y := old.rows; y < rows; y++ {
			rowStart := y * g.stride
			row := g.cells[rowStart : rowStart+g.stride]
			fillEmptyCells(row[:cols])
		}
	}
}

func (g *cellGrid) resizeRealloc(old cellGridState, cols, rows int) {
	newStride := roundUpMultiple(cols, 8)
	newCapRows := roundUpMultiple(rows, 8)
	if newStride < cols {
		newStride = cols
	}
	if newCapRows < rows {
		newCapRows = rows
	}

	newCells := make([]uv.Cell, newStride*newCapRows)
	fillEmptyCells(newCells)

	copyCols, copyRows := copyIntersection(cols, rows, old)
	for y := 0; y < copyRows; y++ {
		src := y * old.stride
		dst := y * newStride
		copy(newCells[dst:dst+copyCols], old.cells[src:src+copyCols])
	}

	g.cells = newCells
	g.stride = newStride
	g.capRows = newCapRows
}

func copyIntersection(cols, rows int, old cellGridState) (int, int) {
	if len(old.cells) == 0 || old.cols <= 0 || old.rows <= 0 || old.stride <= 0 || old.capRows <= 0 {
		return 0, 0
	}

	copyRows := old.rows
	if rows < copyRows {
		copyRows = rows
	}
	copyCols := old.cols
	if cols < copyCols {
		copyCols = cols
	}
	return copyCols, copyRows
}

func (g *cellGrid) Compact() {
	if g == nil || g.cols <= 0 || g.rows <= 0 {
		return
	}
	targetStride := roundUpMultiple(g.cols, 8)
	targetRows := roundUpMultiple(g.rows, 8)
	if g.stride == targetStride && g.capRows == targetRows {
		return
	}
	g.compactTo(g.cols, g.rows)
}

func (g *cellGrid) compactTo(cols, rows int) {
	if cols <= 0 || rows <= 0 {
		g.resetBacking()
		return
	}

	targetStride := roundUpMultiple(cols, 8)
	targetRows := roundUpMultiple(rows, 8)
	if targetStride < cols {
		targetStride = cols
	}
	if targetRows < rows {
		targetRows = rows
	}

	newCells := make([]uv.Cell, targetStride*targetRows)
	fillEmptyCells(newCells)

	copyCols := cols
	if g.cols < copyCols {
		copyCols = g.cols
	}
	copyRows := rows
	if g.rows < copyRows {
		copyRows = g.rows
	}
	for y := 0; y < copyRows; y++ {
		src := y * g.stride
		dst := y * targetStride
		copy(newCells[dst:dst+copyCols], g.cells[src:src+copyCols])
	}

	g.cells = newCells
	g.stride = targetStride
	g.capRows = targetRows
}

func shouldShrinkCapacity(cols, rows, stride, capRows int) bool {
	targetStride := roundUpMultiple(cols, 8)
	targetRows := roundUpMultiple(rows, 8)
	if targetStride < cols {
		targetStride = cols
	}
	if targetRows < rows {
		targetRows = rows
	}
	if targetStride == stride && targetRows == capRows {
		return false
	}

	// Shrink aggressively if we are holding more than 4x the active cell count.
	active := cols * rows
	capacity := stride * capRows
	if active <= 0 || capacity <= 0 {
		return false
	}
	return capacity > active*4
}

func roundUpMultiple(n, multiple int) int {
	if multiple <= 1 {
		return n
	}
	if n <= 0 {
		return 0
	}
	r := n % multiple
	if r == 0 {
		return n
	}
	return n + (multiple - r)
}
