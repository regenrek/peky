package vt

import (
	"strings"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
)

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

	for i := range line {
		c := line[i]
		if c.IsZero() {
			continue
		}

		if c.Equal(&uv.EmptyCell) {
			if pendingSpaces == 0 {
				renderResetForSpace(b, &pen, &link)
			}
			pendingSpaces++
			continue
		}

		if pendingSpaces > 0 {
			renderSpaces(b, pendingSpaces)
			pendingSpaces = 0
		}

		renderApplyStyle(b, &pen, c.Style)
		renderApplyLink(b, &link, c.Link)
		b.WriteString(c.String())
	}

	renderFinalizeLine(b, &pen, &link)
}

func renderSpaces(b *strings.Builder, n int) {
	for n > 0 {
		_ = b.WriteByte(' ')
		n--
	}
}

func renderResetForSpace(b *strings.Builder, pen *uv.Style, link *uv.Link) {
	if pen != nil && !pen.IsZero() {
		b.WriteString(ansi.ResetStyle)
		*pen = uv.Style{}
	}
	if link != nil && link.URL != "" {
		b.WriteString(ansi.ResetHyperlink())
		*link = uv.Link{}
	}
}

func renderFinalizeLine(b *strings.Builder, pen *uv.Style, link *uv.Link) {
	if link != nil && link.URL != "" {
		b.WriteString(ansi.ResetHyperlink())
		*link = uv.Link{}
	}
	if pen != nil && !pen.IsZero() {
		b.WriteString(ansi.ResetStyle)
		*pen = uv.Style{}
	}
}

func renderApplyStyle(b *strings.Builder, pen *uv.Style, next uv.Style) {
	if pen == nil {
		return
	}
	if next.IsZero() {
		if !pen.IsZero() {
			b.WriteString(ansi.ResetStyle)
			*pen = uv.Style{}
		}
		return
	}
	if next.Equal(pen) {
		return
	}
	b.WriteString(next.Diff(pen))
	*pen = next
}

func renderApplyLink(b *strings.Builder, link *uv.Link, next uv.Link) {
	if link == nil {
		return
	}
	if next == *link {
		return
	}

	if link.URL != "" {
		b.WriteString(ansi.ResetHyperlink())
		*link = uv.Link{}
	}
	if next.URL == "" {
		return
	}
	b.WriteString(ansi.SetHyperlink(next.URL, next.Params))
	*link = next
}
