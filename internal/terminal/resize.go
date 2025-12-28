package terminal

import "errors"

// Resize resizes both the VT and PTY (PTY resize is best-effort).
func (w *Window) Resize(cols, rows int) error {
	if w == nil {
		return errors.New("terminal: nil window")
	}
	if cols <= 0 || rows <= 0 {
		return nil
	}
	if w.closed.Load() {
		return errors.New("terminal: window closed")
	}

	changed := (cols != w.cols) || (rows != w.rows)
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

	if changed {
		w.clampViewState()
		w.markDirty()
	}
	return nil
}
