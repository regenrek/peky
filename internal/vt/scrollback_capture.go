package vt

import (
	uv "github.com/charmbracelet/ultraviolet"
)

func appendGlyphsFromCells(dst *cellStore, line []uv.Cell) int {
	end := len(line)
	for end > 0 {
		c := line[end-1]
		if c.Width == 0 {
			end--
			continue
		}
		if isBlankCell(c) {
			end--
			continue
		}
		break
	}
	if end <= 0 {
		return 0
	}

	n := 0
	for i := 0; i < end; i++ {
		c := line[i]
		if c.Width == 0 {
			continue
		}
		if c.Width <= 0 {
			c.Width = 1
		}
		if c.Content == "" {
			c.Content = " "
		}
		dst.AppendCell(c)
		n++
	}
	return n
}

func appendGlyphsFromBuffer(dst *cellStore, buf *uv.Buffer, y, width int) int {
	if buf == nil || width <= 0 {
		return 0
	}

	endX := -1
	for x := width - 1; x >= 0; x-- {
		c := cellAtOrEmpty(buf, x, y)
		if c.Width == 0 {
			continue
		}
		if isBlankCell(c) {
			continue
		}
		endX = x
		break
	}
	if endX < 0 {
		return 0
	}

	n := 0
	for x := 0; x <= endX; x++ {
		c := cellAtOrEmpty(buf, x, y)
		if c.Width == 0 {
			continue
		}
		if c.Width <= 0 {
			c.Width = 1
		}
		if c.Content == "" {
			c.Content = " "
		}
		dst.AppendCell(c)
		n++
	}
	return n
}

func guessSoftWrappedRow(buf *uv.Buffer, y, width int) bool {
	if buf == nil || width <= 0 {
		return false
	}

	for x := width - 1; x >= 0; x-- {
		c := cellAtOrEmpty(buf, x, y)
		if c.Width == 0 {
			continue
		}
		if isBlankCell(c) {
			continue
		}
		w := c.Width
		if w <= 0 {
			w = 1
		}
		end := x + w - 1
		return end >= width-1
	}
	return false
}

func cellAtOrEmpty(buf *uv.Buffer, x, y int) uv.Cell {
	if buf == nil {
		return uv.EmptyCell
	}
	if cell := buf.CellAt(x, y); cell != nil {
		return *cell
	}
	return uv.EmptyCell
}

func isBlankCell(c uv.Cell) bool {
	if c.Content != "" && c.Content != " " {
		return false
	}
	if c.Width == 0 {
		return true
	}
	var zeroStyle uv.Style
	if !c.Style.Equal(&zeroStyle) {
		return false
	}
	if c.Link != (uv.Link{}) {
		return false
	}
	return true
}
