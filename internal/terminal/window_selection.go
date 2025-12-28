package terminal

import "strings"

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

	for y := startAbsY; y <= endAbsY; y++ {
		x0, x1 := selectionRangeForLine(startAbsY, endAbsY, startX, endX, width, y)
		out.WriteString(lineText(term, sbLen, rows, x0, x1, y))
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

func lineText(term vtEmulator, sbLen, rows, x0, x1, y int) string {
	var line strings.Builder
	for x := x0; x <= x1; x++ {
		cell := cellAtAbs(term, sbLen, rows, x, y)
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
