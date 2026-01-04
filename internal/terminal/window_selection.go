package terminal

import (
	"strings"

	uv "github.com/charmbracelet/ultraviolet"
)

func (w *Window) extractText(startX, startAbsY, endX, endAbsY int) string {
	if w == nil {
		return ""
	}
	width := w.cols
	if width <= 0 {
		return ""
	}

	w.termMu.Lock()
	defer w.termMu.Unlock()
	term := w.term
	if term == nil {
		return ""
	}

	startX, startAbsY, endX, endAbsY = normalizeSelection(startX, startAbsY, endX, endAbsY)

	sbLen := term.ScrollbackLen()
	total := sbLen + w.rows
	if total <= 0 {
		return ""
	}

	startAbsY, endAbsY = clampSelectionY(startAbsY, endAbsY, total-1)
	startX, endX = clampSelectionX(startX, endX, width)

	return buildSelectionText(term, sbLen, w.rows, width, startAbsY, endAbsY, startX, endX)
}

func normalizeSelection(startX, startAbsY, endX, endAbsY int) (int, int, int, int) {
	if startAbsY > endAbsY || (startAbsY == endAbsY && startX > endX) {
		startX, endX = endX, startX
		startAbsY, endAbsY = endAbsY, startAbsY
	}
	return startX, startAbsY, endX, endAbsY
}

func clampSelectionY(startAbsY, endAbsY, maxY int) (int, int) {
	startAbsY = clampInt(startAbsY, 0, maxY)
	endAbsY = clampInt(endAbsY, 0, maxY)
	return startAbsY, endAbsY
}

func clampSelectionX(startX, endX, width int) (int, int) {
	if width <= 0 {
		return 0, 0
	}
	maxX := maxInt(0, width-1)
	startX = clampInt(startX, 0, maxX)
	endX = clampInt(endX, 0, maxX)
	return startX, endX
}

func buildSelectionText(term vtEmulator, sbLen, rows, width, startAbsY, endAbsY, startX, endX int) string {
	var out strings.Builder
	var sbRow []uv.Cell
	if sbLen > 0 && width > 0 {
		sbRow = make([]uv.Cell, width)
	}

	for y := startAbsY; y <= endAbsY; y++ {
		x0, x1 := selectionRangeForLine(startAbsY, endAbsY, startX, endX, width, y)
		if y < sbLen {
			if sbRow == nil || len(sbRow) != width {
				sbRow = make([]uv.Cell, width)
			}
			if ok := term.CopyScrollbackRow(y, sbRow); ok {
				out.WriteString(lineTextCells(sbRow, x0, x1))
			} else {
				out.WriteString("")
			}
		} else {
			screenY := y - sbLen
			out.WriteString(lineTextScreen(term, screenY, x0, x1))
		}
		if y != endAbsY {
			out.WriteByte('\n')
		}
	}

	return strings.TrimRight(out.String(), "\n")
}

func selectionRangeForLine(startAbsY, endAbsY, startX, endX, width, y int) (int, int) {
	x0 := 0
	x1 := width - 1
	if y == startAbsY {
		x0 = startX
	}
	if y == endAbsY {
		x1 = endX
	}
	x0 = clampInt(x0, 0, width-1)
	x1 = clampInt(x1, 0, width-1)
	return x0, x1
}

func lineTextCells(cells []uv.Cell, x0, x1 int) string {
	var line strings.Builder
	for x := x0; x <= x1; x++ {
		if x < 0 || x >= len(cells) {
			line.WriteByte(' ')
			continue
		}
		cell := &cells[x]
		if cell != nil && cell.Width == 0 {
			continue
		}
		if cell != nil && cell.Content != "" {
			line.WriteString(cell.Content)
		} else {
			line.WriteByte(' ')
		}
	}
	return strings.TrimRight(line.String(), " ")
}

func lineTextScreen(term vtEmulator, row, x0, x1 int) string {
	var line strings.Builder
	for x := x0; x <= x1; x++ {
		cell := term.CellAt(x, row)
		if cell != nil && cell.Width == 0 {
			continue
		}
		if cell != nil && cell.Content != "" {
			line.WriteString(cell.Content)
		} else {
			line.WriteByte(' ')
		}
	}
	return strings.TrimRight(line.String(), " ")
}
