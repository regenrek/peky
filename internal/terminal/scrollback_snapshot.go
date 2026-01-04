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
	return snapshotScrollbackFromTerm(term, rows)
}

func snapshotScrollbackFromTerm(term vtEmulator, rows int) (string, bool) {
	if term == nil || rows == 0 {
		return "", false
	}
	if rows < 0 {
		rows = 0
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
	if total <= 0 {
		return "", false
	}
	if rows <= 0 || rows > total {
		rows = total
	}

	start := total - rows
	truncated := rows < total
	lines := snapshotScrollbackLines(term, cols, sbLen, start, rows)
	return strings.Join(lines, "\n"), truncated
}

func snapshotScrollbackLines(term vtEmulator, cols, sbLen, start, rows int) []string {
	if term == nil || rows <= 0 {
		return nil
	}

	end := start + rows
	lines := make([]string, 0, rows)

	sbEnd := end
	if sbEnd > sbLen {
		sbEnd = sbLen
	}
	if start < sbEnd {
		lines = append(lines, snapshotScrollbackSegment(term, cols, start, sbEnd)...)
	}

	screenStart := start
	if screenStart < sbLen {
		screenStart = sbLen
	}
	if screenStart < end {
		lines = append(lines, snapshotScreenSegment(term, cols, sbLen, screenStart, end)...)
	}

	return lines
}

func snapshotScrollbackSegment(term vtEmulator, cols, startAbs, endAbs int) []string {
	if term == nil || startAbs >= endAbs {
		return nil
	}
	out := make([]string, 0, endAbs-startAbs)
	if cols <= 0 {
		for i := startAbs; i < endAbs; i++ {
			out = append(out, "")
		}
		return out
	}
	sbRow := make([]uv.Cell, cols)
	for abs := startAbs; abs < endAbs; abs++ {
		if ok := term.CopyScrollbackRow(abs, sbRow); ok {
			out = append(out, lineFromCells(sbRow, cols))
		} else {
			out = append(out, "")
		}
	}
	return out
}

func snapshotScreenSegment(term vtEmulator, cols, sbLen, startAbs, endAbs int) []string {
	if term == nil || startAbs >= endAbs {
		return nil
	}
	out := make([]string, 0, endAbs-startAbs)
	for abs := startAbs; abs < endAbs; abs++ {
		out = append(out, screenLine(term, cols, abs-sbLen))
	}
	return out
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
