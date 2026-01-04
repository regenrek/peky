package vt

import uv "github.com/charmbracelet/ultraviolet"

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
