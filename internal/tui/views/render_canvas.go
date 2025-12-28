package views

import "strings"

// ===== Rendering helpers =====

type canvas struct {
	w     int
	h     int
	cells []rune
}

func newCanvas(w, h int) *canvas {
	c := &canvas{w: w, h: h, cells: make([]rune, w*h)}
	for i := range c.cells {
		c.cells[i] = ' '
	}
	return c
}

func (c *canvas) set(x, y int, r rune) {
	if x < 0 || y < 0 || x >= c.w || y >= c.h {
		return
	}
	c.cells[y*c.w+x] = r
}

func (c *canvas) drawBox(x, y, w, h int) {
	if w < 2 || h < 2 {
		return
	}
	x2 := x + w - 1
	y2 := y + h - 1
	for ix := x + 1; ix < x2; ix++ {
		c.set(ix, y, '-')
		c.set(ix, y2, '-')
	}
	for iy := y + 1; iy < y2; iy++ {
		c.set(x, iy, '|')
		c.set(x2, iy, '|')
	}
	c.set(x, y, '+')
	c.set(x2, y, '+')
	c.set(x, y2, '+')
	c.set(x2, y2, '+')
}

func (c *canvas) write(x, y int, text string, max int) {
	if y < 0 || y >= c.h || max <= 0 {
		return
	}
	trimmed := truncateLine(text, max)
	for i, r := range []rune(trimmed) {
		if x+i >= c.w {
			break
		}
		c.set(x+i, y, r)
	}
}

func (c *canvas) String() string {
	lines := make([]string, c.h)
	for y := 0; y < c.h; y++ {
		lines[y] = string(c.cells[y*c.w : (y+1)*c.w])
	}
	return strings.Join(lines, "\n")
}
