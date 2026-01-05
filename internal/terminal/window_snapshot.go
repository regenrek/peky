package terminal

import (
	"errors"
	"time"
)

// PlainSnapshot captures a text-only VT snapshot.
type PlainSnapshot struct {
	CapturedAt    time.Time
	Cols          int
	Rows          int
	CursorX       int
	CursorY       int
	CursorVisible bool
	AltScreen     bool
	ScreenLines   []string
	Scrollback    []string
}

// PlainSnapshotOptions controls snapshot capture.
type PlainSnapshotOptions struct {
	MaxScrollbackLines int
}

// SnapshotPlain captures a text-only snapshot of screen + scrollback.
func (w *Window) SnapshotPlain(opts PlainSnapshotOptions) (PlainSnapshot, error) {
	if w == nil {
		return PlainSnapshot{}, errors.New("terminal: window is nil")
	}
	w.termMu.Lock()
	term := w.term
	if term == nil {
		w.termMu.Unlock()
		return PlainSnapshot{}, errors.New("terminal: terminal is nil")
	}
	cols := term.Width()
	rows := term.Height()
	if cols < 0 {
		cols = 0
	}
	if rows < 0 {
		rows = 0
	}
	sbLen := term.ScrollbackLen()
	start := 0
	if max := opts.MaxScrollbackLines; max > 0 && sbLen > max {
		start = sbLen - max
	}
	scrollback := snapshotScrollbackSegment(term, cols, start, sbLen)
	screen := make([]string, rows)
	for y := 0; y < rows; y++ {
		screen[y] = screenLine(term, cols, y)
	}
	cursor := term.CursorPosition()
	alt := w.altScreen.Load()
	cursorVisible := w.cursorVisible.Load()
	w.termMu.Unlock()
	return PlainSnapshot{
		CapturedAt:    time.Now().UTC(),
		Cols:          cols,
		Rows:          rows,
		CursorX:       cursor.X,
		CursorY:       cursor.Y,
		CursorVisible: cursorVisible,
		AltScreen:     alt,
		ScreenLines:   screen,
		Scrollback:    scrollback,
	}, nil
}
