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

func guessSoftWrappedLine(line []uv.Cell, width int) bool {
	if width <= 0 || len(line) == 0 {
		return false
	}

	for x := width - 1; x >= 0; x-- {
		c := line[x]
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
