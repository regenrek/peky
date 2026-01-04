package vt

import uv "github.com/charmbracelet/ultraviolet"

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
			continue
		}
		fillRowArea(g.Row(row), area.Min.X, area.Max.X, c)
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
			continue
		}
		fillRowArea(g.Row(row), area.Min.X, area.Max.X, c)
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
	area, n, ok := g.prepareCellOp(x, y, n, area)
	if !ok {
		return
	}

	row := g.Row(y)
	g.deleteCellFixWideStart(row, x, c)
	g.deleteCellShiftLeft(row, x, n, area.Max.X)
	fillRowArea(row, area.Max.X-n, area.Max.X, c)
	normalizeWideRow(row)
}

func (g *cellGrid) prepareCellOp(x, y, n int, area uv.Rectangle) (uv.Rectangle, int, bool) {
	if g.cols <= 0 || g.rows <= 0 || n <= 0 {
		return uv.Rectangle{}, 0, false
	}
	area = area.Intersect(g.Bounds())
	if area.Empty() || y < area.Min.Y || y >= area.Max.Y || x < area.Min.X || x >= area.Max.X {
		return uv.Rectangle{}, 0, false
	}
	if n > area.Max.X-x {
		n = area.Max.X - x
	}
	if n <= 0 {
		return uv.Rectangle{}, 0, false
	}
	return area, n, true
}

func (g *cellGrid) deleteCellFixWideStart(row []uv.Cell, x int, c *uv.Cell) {
	if x < 0 || x >= len(row) {
		return
	}
	prev := row[x]
	if prev.IsZero() || prev.Width > 1 {
		uv.Line(row).Set(x, c)
	}
}

func (g *cellGrid) deleteCellShiftLeft(row []uv.Cell, x, n, areaMaxX int) {
	if x+n >= areaMaxX {
		return
	}
	src := row[x+n : areaMaxX]
	dst := row[x : areaMaxX-n]
	copy(dst, src)
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
