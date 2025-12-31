package terminal

import (
	"strings"

	uv "github.com/charmbracelet/ultraviolet"
)

// SnapshotScrollback returns a plain-text snapshot of the scrollback buffer
// plus current screen. It returns whether output was truncated.
func (w *Window) SnapshotScrollback(rows int) (string, bool) {
	if w == nil || rows == 0 {
		return "", false
	}
	if rows < 0 {
		rows = 0
	}
	w.termMu.Lock()
	term := w.term
	w.termMu.Unlock()
	if term == nil {
		return "", false
	}
	cols := term.Width()
	if cols < 0 {
		cols = 0
	}
	sbLen := term.ScrollbackLen()
	screenRows := term.Height()
	if screenRows < 0 {
		screenRows = 0
	}
	total := sbLen + screenRows
	if total == 0 {
		return "", false
	}
	if rows <= 0 || rows > total {
		rows = total
	}
	start := total - rows
	truncated := rows < total
	lines := make([]string, 0, rows)
	for i := 0; i < rows; i++ {
		abs := start + i
		if abs < sbLen {
			lines = append(lines, lineFromCells(term.ScrollbackLine(abs), cols))
			continue
		}
		row := abs - sbLen
		lines = append(lines, screenLine(term, cols, row))
	}
	return strings.Join(lines, "\n"), truncated
}

func screenLine(term vtEmulator, cols, row int) string {
	if term == nil || cols <= 0 || row < 0 {
		return ""
	}
	cells := make([]uv.Cell, cols)
	for x := 0; x < cols; x++ {
		if cell := term.CellAt(x, row); cell != nil {
			cells[x] = *cell
		} else {
			cells[x] = uv.EmptyCell
		}
	}
	return lineFromCells(cells, cols)
}

func lineFromCells(cells []uv.Cell, width int) string {
	if width <= 0 {
		width = len(cells)
	}
	var b strings.Builder
	b.Grow(width)
	for i := 0; i < width; i++ {
		if i < len(cells) && cells[i].Content != "" {
			b.WriteString(cells[i].Content)
			continue
		}
		b.WriteByte(' ')
	}
	return strings.TrimRight(b.String(), " ")
}
