package terminal

import (
	"errors"

	"github.com/regenrek/peakypanes/internal/limits"
)

// Resize resizes both the VT and PTY (PTY resize is best-effort).
func (w *Window) Resize(cols, rows int) error {
	if w == nil {
		return errors.New("terminal: nil window")
	}
	if cols <= 0 || rows <= 0 {
		return nil
	}
	cols, rows = limits.Clamp(cols, rows)
	if w.closed.Load() {
		return errors.New("terminal: window closed")
	}

	if cols == w.cols && rows == w.rows {
		return nil
	}
	w.cols, w.rows = cols, rows

	w.termMu.Lock()
	if w.term != nil {
		w.term.Resize(cols, rows)
	}
	w.termMu.Unlock()

	w.ptyMu.Lock()
	pty := w.pty
	w.ptyMu.Unlock()
	if pty != nil {
		_ = pty.Resize(cols, rows)
	}

	w.clampViewState()
	w.markDirty()
	return nil
}
