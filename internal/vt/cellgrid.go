package vt

import (
	"strings"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
)

type cellGrid struct {
	cols    int
	rows    int
	stride  int
	capRows int
	cells   []uv.Cell
}

func newCellGrid(cols, rows int) cellGrid {
	var g cellGrid
	g.Resize(cols, rows)
	return g
}

func (g *cellGrid) Width() int  { return g.cols }
func (g *cellGrid) Height() int { return g.rows }

func (g *cellGrid) Bounds() uv.Rectangle {
	return uv.Rect(0, 0, g.cols, g.rows)
}

func (g *cellGrid) Row(y int) []uv.Cell {
	if y < 0 || y >= g.rows || g.cols <= 0 || g.stride <= 0 {
		return nil
	}
	start := y * g.stride
	if start < 0 || start+g.cols > len(g.cells) {
		return nil
	}
	return g.cells[start : start+g.cols]
}

func (g *cellGrid) CellAt(x, y int) *uv.Cell {
	if x < 0 || y < 0 || x >= g.cols || y >= g.rows {
		return nil
	}
	idx := y*g.stride + x
	if idx < 0 || idx >= len(g.cells) {
		return nil
	}
	return &g.cells[idx]
}

func (g *cellGrid) SetCell(x, y int, c *uv.Cell) {
	if x < 0 || y < 0 || x >= g.cols || y >= g.rows {
		return
	}
	row := uv.Line(g.Row(y))
	row.Set(x, c)
}

func (g *cellGrid) Clear() { g.ClearArea(g.Bounds()) }

func (g *cellGrid) ClearArea(area uv.Rectangle) {
	if g.cols <= 0 || g.rows <= 0 || area.Empty() {
		return
	}
	area = area.Intersect(g.Bounds())
	if area.Empty() {
		return
	}
	for y := area.Min.Y; y < area.Max.Y; y++ {
		if area.Min.X == 0 && area.Max.X == g.cols {
			row := g.Row(y)
			fillEmptyCells(row)
			continue
		}
		row := uv.Line(g.Row(y))
		for x := area.Min.X; x < area.Max.X; x++ {
			row.Set(x, nil)
		}
	}
}

func (g *cellGrid) FillArea(c *uv.Cell, area uv.Rectangle) {
	if g.cols <= 0 || g.rows <= 0 || area.Empty() {
		return
	}
	area = area.Intersect(g.Bounds())
	if area.Empty() {
		return
	}

	cellWidth := 1
	if c != nil && c.Width > 1 {
		cellWidth = c.Width
	}
	for y := area.Min.Y; y < area.Max.Y; y++ {
		row := uv.Line(g.Row(y))
		for x := area.Min.X; x < area.Max.X; x += cellWidth {
			row.Set(x, c)
		}
	}
}

func (g *cellGrid) Resize(cols, rows int) {
	if cols < 0 {
		cols = 0
	}
	if rows < 0 {
		rows = 0
	}
	if cols == g.cols && rows == g.rows {
		return
	}

	oldCols := g.cols
	oldRows := g.rows
	oldStride := g.stride
	oldCapRows := g.capRows
	oldCells := g.cells

	g.cols = cols
	g.rows = rows

	if cols == 0 || rows == 0 {
		g.cells = nil
		g.stride = 0
		g.capRows = 0
		return
	}

	if cols <= oldStride && rows <= oldCapRows && len(oldCells) == oldStride*oldCapRows {
		if shouldShrinkCapacity(cols, rows, oldStride, oldCapRows) {
			g.compactTo(cols, rows)
			return
		}

		g.stride = oldStride
		g.capRows = oldCapRows

		// Fill newly visible columns in existing rows.
		if cols > oldCols {
			for y := 0; y < oldRows && y < rows; y++ {
				row := g.cells[y*g.stride : y*g.stride+g.stride]
				fillEmptyCells(row[oldCols:cols])
			}
		}

		// Fill newly visible rows.
		if rows > oldRows {
			for y := oldRows; y < rows; y++ {
				row := g.cells[y*g.stride : y*g.stride+g.stride]
				fillEmptyCells(row[:cols])
			}
		}

		return
	}

	newStride := roundUpMultiple(cols, 8)
	newCapRows := roundUpMultiple(rows, 8)
	if newStride < cols {
		newStride = cols
	}
	if newCapRows < rows {
		newCapRows = rows
	}
	newSize := newStride * newCapRows

	newCells := make([]uv.Cell, newSize)
	fillEmptyCells(newCells)

	if len(oldCells) > 0 && oldCols > 0 && oldRows > 0 && oldStride > 0 && oldCapRows > 0 {
		minRows := oldRows
		if rows < minRows {
			minRows = rows
		}
		minCols := oldCols
		if cols < minCols {
			minCols = cols
		}
		for y := 0; y < minRows; y++ {
			src := y * oldStride
			dst := y * newStride
			copy(newCells[dst:dst+minCols], oldCells[src:src+minCols])
		}
	}
	g.cells = newCells
	g.stride = newStride
	g.capRows = newCapRows
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
		g.cells = nil
		g.stride = 0
		g.capRows = 0
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

	minCols := cols
	if g.cols < minCols {
		minCols = g.cols
	}
	minRows := rows
	if g.rows < minRows {
		minRows = g.rows
	}
	for y := 0; y < minRows; y++ {
		src := y * g.stride
		dst := y * targetStride
		if src < 0 || dst < 0 || src+minCols > len(g.cells) || dst+minCols > len(newCells) {
			break
		}
		copy(newCells[dst:dst+minCols], g.cells[src:src+minCols])
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

func (g *cellGrid) InsertLineArea(y, n int, c *uv.Cell, area uv.Rectangle) {
	if g.cols <= 0 || g.rows <= 0 || n <= 0 {
		return
	}
	area = area.Intersect(g.Bounds())
	if area.Empty() || y < area.Min.Y || y >= area.Max.Y {
		return
	}
	if y+n > area.Max.Y {
		n = area.Max.Y - y
	}
	if n <= 0 {
		return
	}

	for row := area.Max.Y - 1; row >= y+n; row-- {
		dstRow := g.Row(row)[area.Min.X:area.Max.X]
		srcRow := g.Row(row - n)[area.Min.X:area.Max.X]
		copy(dstRow, srcRow)
	}

	for row := y; row < y+n; row++ {
		if area.Min.X == 0 && area.Max.X == g.cols && (c == nil || c.Width <= 1) {
			dst := g.Row(row)
			fillCells(dst, c)
		} else {
			fillRowArea(g.Row(row), area.Min.X, area.Max.X, c)
		}
	}
}

func (g *cellGrid) DeleteLineArea(y, n int, c *uv.Cell, area uv.Rectangle) {
	if g.cols <= 0 || g.rows <= 0 || n <= 0 {
		return
	}
	area = area.Intersect(g.Bounds())
	if area.Empty() || y < area.Min.Y || y >= area.Max.Y {
		return
	}
	if n > area.Max.Y-y {
		n = area.Max.Y - y
	}
	if n <= 0 {
		return
	}

	for row := y; row < area.Max.Y-n; row++ {
		dstRow := g.Row(row)[area.Min.X:area.Max.X]
		srcRow := g.Row(row + n)[area.Min.X:area.Max.X]
		copy(dstRow, srcRow)
	}

	for row := area.Max.Y - n; row < area.Max.Y; row++ {
		if area.Min.X == 0 && area.Max.X == g.cols && (c == nil || c.Width <= 1) {
			dst := g.Row(row)
			fillCells(dst, c)
		} else {
			fillRowArea(g.Row(row), area.Min.X, area.Max.X, c)
		}
	}
}

func (g *cellGrid) InsertCellArea(x, y, n int, c *uv.Cell, area uv.Rectangle) {
	if g.cols <= 0 || g.rows <= 0 || n <= 0 {
		return
	}
	area = area.Intersect(g.Bounds())
	if area.Empty() || y < area.Min.Y || y >= area.Max.Y || x < area.Min.X || x >= area.Max.X {
		return
	}
	if x+n > area.Max.X {
		n = area.Max.X - x
	}
	if n <= 0 {
		return
	}

	row := g.Row(y)
	shiftStart := x + n
	if shiftStart < area.Max.X {
		src := row[x : area.Max.X-n]
		dst := row[shiftStart:area.Max.X]
		copy(dst, src)
	}

	line := uv.Line(row)
	cellWidth := 1
	if c != nil && c.Width > 1 {
		cellWidth = c.Width
	}
	for pos := x; pos < x+n; pos += cellWidth {
		line.Set(pos, c)
	}
	normalizeWideRow(row)
}

func (g *cellGrid) DeleteCellArea(x, y, n int, c *uv.Cell, area uv.Rectangle) {
	if g.cols <= 0 || g.rows <= 0 || n <= 0 {
		return
	}
	area = area.Intersect(g.Bounds())
	if area.Empty() || y < area.Min.Y || y >= area.Max.Y || x < area.Min.X || x >= area.Max.X {
		return
	}
	if n > area.Max.X-x {
		n = area.Max.X - x
	}
	if n <= 0 {
		return
	}

	row := g.Row(y)
	if x >= 0 && x < len(row) {
		prev := row[x]
		if prev.IsZero() || prev.Width > 1 {
			uv.Line(row).Set(x, c)
		}
	}
	if x+n < area.Max.X {
		src := row[x+n : area.Max.X]
		dst := row[x : area.Max.X-n]
		copy(dst, src)
	}
	fillRowArea(row, area.Max.X-n, area.Max.X, c)
	normalizeWideRow(row)
}

func (g *cellGrid) String() string {
	if g.cols <= 0 || g.rows <= 0 {
		return ""
	}
	var b strings.Builder
	for y := 0; y < g.rows; y++ {
		line := uv.Line(g.Row(y))
		b.WriteString(line.String())
		if y < g.rows-1 {
			b.WriteByte('\n')
		}
	}
	return b.String()
}

func (g *cellGrid) Render() string {
	if g.cols <= 0 || g.rows <= 0 {
		return ""
	}

	var b strings.Builder
	// Conservative pre-grow: each cell may contribute at least 1 byte; ANSI adds more.
	b.Grow(g.cols * g.rows)

	for y := 0; y < g.rows; y++ {
		renderLine(&b, g.Row(y))
		if y < g.rows-1 {
			_ = b.WriteByte('\n')
		}
	}
	return b.String()
}

func renderLine(b *strings.Builder, line []uv.Cell) {
	var pen uv.Style
	var link uv.Link
	pendingSpaces := 0

	flushSpaces := func() {
		for pendingSpaces > 0 {
			_ = b.WriteByte(' ')
			pendingSpaces--
		}
	}

	for i := range line {
		c := line[i]
		if c.IsZero() {
			continue
		}
		if c.Equal(&uv.EmptyCell) {
			if !pen.IsZero() {
				b.WriteString(ansi.ResetStyle)
				pen = uv.Style{}
			}
			if !link.IsZero() {
				b.WriteString(ansi.ResetHyperlink())
				link = uv.Link{}
			}
			pendingSpaces++
			continue
		}

		if pendingSpaces > 0 {
			flushSpaces()
		}

		if c.Style.IsZero() && !pen.IsZero() {
			b.WriteString(ansi.ResetStyle)
			pen = uv.Style{}
		}
		if !c.Style.Equal(&pen) {
			seq := c.Style.Diff(&pen)
			b.WriteString(seq)
			pen = c.Style
		}

		if c.Link != link && link.URL != "" {
			b.WriteString(ansi.ResetHyperlink())
			link = uv.Link{}
		}
		if c.Link != link {
			b.WriteString(ansi.SetHyperlink(c.Link.URL, c.Link.Params))
			link = c.Link
		}

		b.WriteString(c.String())
	}

	if link.URL != "" {
		b.WriteString(ansi.ResetHyperlink())
	}
	if !pen.IsZero() {
		b.WriteString(ansi.ResetStyle)
	}
}

func fillEmptyCells(dst []uv.Cell) {
	for i := range dst {
		dst[i] = uv.EmptyCell
	}
}

func fillCells(dst []uv.Cell, c *uv.Cell) {
	if c == nil {
		fillEmptyCells(dst)
		return
	}
	val := *c
	for i := range dst {
		dst[i] = val
	}
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

func fillRowArea(row []uv.Cell, start, end int, c *uv.Cell) {
	if start < 0 {
		start = 0
	}
	if end > len(row) {
		end = len(row)
	}
	if start >= end {
		return
	}
	line := uv.Line(row)
	step := 1
	if c != nil && c.Width > 1 {
		step = c.Width
	}
	for x := start; x < end; x += step {
		line.Set(x, c)
	}
}

func normalizeWideRow(row []uv.Cell) {
	if len(row) == 0 {
		return
	}
	basePos := -1
	baseWidth := 0
	cover := 0

	for x := 0; x < len(row); x++ {
		c := row[x]
		if cover > 0 {
			if c.IsZero() {
				cover--
				if cover == 0 {
					basePos = -1
					baseWidth = 0
				}
				continue
			}
			blankWide(row, basePos, baseWidth)
			basePos = -1
			baseWidth = 0
			cover = 0
			x--
			continue
		}

		if c.IsZero() {
			row[x] = uv.EmptyCell
			continue
		}

		w := c.Width
		if w <= 1 {
			continue
		}
		if x+w > len(row) {
			row[x] = uv.EmptyCell
			continue
		}
		basePos = x
		baseWidth = w
		cover = w - 1
	}

	if cover > 0 && basePos >= 0 {
		blankWide(row, basePos, baseWidth)
	}
}

func blankWide(row []uv.Cell, basePos, width int) {
	if basePos < 0 || width <= 0 {
		return
	}
	for i := 0; i < width && basePos+i < len(row); i++ {
		row[basePos+i] = uv.EmptyCell
	}
}
