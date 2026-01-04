package terminal

import uv "github.com/charmbracelet/ultraviolet"

func (w *Window) adjustToNonContinuation(absY, x, dir int) int {
	if w == nil {
		return x
	}
	x = clampInt(x, 0, maxInt(0, w.cols-1))
	if dir == 0 {
		dir = -1
	}

	w.termMu.Lock()
	defer w.termMu.Unlock()

	term := w.term
	if term == nil {
		return x
	}

	sbLen := term.ScrollbackLen()
	if absY < sbLen {
		return w.adjustToNonContinuationScrollbackLocked(term, absY, x, dir)
	}

	screenY := absY - sbLen
	if screenY < 0 || screenY >= w.rows {
		return x
	}
	return w.adjustToNonContinuationScreenLocked(term, screenY, x, dir)
}

func (w *Window) adjustToNonContinuationScrollbackLocked(term vtEmulator, absY, x, dir int) int {
	sbRow := make([]uv.Cell, w.cols)
	if ok := term.CopyScrollbackRow(absY, sbRow); !ok {
		return x
	}
	return adjustToNonContinuationLoop(x, dir, w.cols, func(ix int) *uv.Cell { return &sbRow[ix] })
}

func (w *Window) adjustToNonContinuationScreenLocked(term vtEmulator, screenY, x, dir int) int {
	return adjustToNonContinuationLoop(x, dir, w.cols, func(ix int) *uv.Cell { return term.CellAt(ix, screenY) })
}

func adjustToNonContinuationLoop(x, dir, cols int, cellAt func(int) *uv.Cell) int {
	maxX := maxInt(0, cols-1)
	for x > 0 && x < cols {
		cell := cellAt(x)
		if cell == nil || cell.Width != 0 {
			return x
		}
		x += dir
		if x < 0 {
			return 0
		}
		if x >= cols {
			return maxX
		}
	}
	return x
}
