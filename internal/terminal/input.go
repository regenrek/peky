package terminal

import (
	"context"
	"errors"
	"fmt"
	"io"
)

// SendInput writes bytes to the underlying PTY.
// This is what your Bubble Tea model should call for focused pane input.
func (w *Window) SendInput(input []byte) error {
	if w == nil {
		return errors.New("terminal: nil window")
	}
	if len(input) == 0 {
		return nil
	}
	if w.closed.Load() {
		return errors.New("terminal: window closed")
	}
	w.ptyMu.Lock()
	pty := w.pty
	w.ptyMu.Unlock()
	if pty == nil {
		return errors.New("terminal: no pty")
	}

	w.writeMu.Lock()
	defer w.writeMu.Unlock()

	n, err := pty.Write(input)
	if err != nil {
		return fmt.Errorf("terminal: pty write: %w", err)
	}
	if n != len(input) {
		return fmt.Errorf("terminal: partial write: wrote %d of %d", n, len(input))
	}

	// Input often changes the screen (echo, app updates).
	w.markDirty()
	return nil
}

func (w *Window) startIO(ctx context.Context) {
	w.startPtyToVt(ctx)
	w.startVtToPty(ctx)
}

// looksLikeCPR checks for ESC[{row};{col}R
func looksLikeCPR(data []byte) bool {
	if len(data) < 6 {
		return false
	}
	if data[0] != 0x1b || data[1] != '[' {
		return false
	}
	if data[len(data)-1] != 'R' {
		return false
	}
	// Must contain ';'
	for _, b := range data {
		if b == ';' {
			return true
		}
	}
	return false
}

func (w *Window) startPtyToVt(ctx context.Context) {
	// PTY -> VT (screen updates)
	w.wg.Add(1)
	go func() {
		defer w.wg.Done()

		buf := make([]byte, 32*1024)
		for {
			pty := w.currentPTY()
			if pty == nil {
				return
			}

			n, err := pty.Read(buf)
			if n > 0 {
				w.handleTerminalWrite(buf[:n])
			}
			if err != nil {
				// Best-effort: treat read errors as exit.
				return
			}
			if ctxDone(ctx) {
				return
			}
		}
	}()
}

func (w *Window) startVtToPty(ctx context.Context) {
	// VT -> PTY (terminal query responses like DSR/DA)
	// This is critical for apps like vim/htop/ncurses that query terminal state.
	w.wg.Add(1)
	go func() {
		defer w.wg.Done()

		buf := make([]byte, 4096)
		for {
			term := w.currentTerminal()
			pty := w.currentPTY()
			if term == nil || pty == nil {
				return
			}

			n, err := term.Read(buf)
			if n > 0 {
				data := w.translateCPR(term, buf[:n])
				w.writeToPTY(pty, data)
			}
			if err != nil {
				// Best-effort: treat read errors as exit.
				return
			}
			if ctxDone(ctx) {
				return
			}
		}
	}()
}

func (w *Window) currentPTY() io.ReadWriter {
	w.ptyMu.Lock()
	defer w.ptyMu.Unlock()
	return w.pty
}

func (w *Window) currentTerminal() vtEmulator {
	w.termMu.Lock()
	defer w.termMu.Unlock()
	return w.term
}

func (w *Window) handleTerminalWrite(data []byte) {
	// Track scrollback growth so scrollback view stays stable.
	oldSB := 0
	newSB := 0

	w.termMu.Lock()
	if w.term != nil {
		oldSB = w.term.ScrollbackLen()
		_, _ = w.term.Write(data)
		newSB = w.term.ScrollbackLen()
	}
	w.termMu.Unlock()

	if newSB > oldSB {
		w.onScrollbackGrew(newSB - oldSB)
	}
	w.markDirty()
}

func (w *Window) translateCPR(term vtEmulator, data []byte) []byte {
	if !looksLikeCPR(data) {
		return data
	}
	w.termMu.Lock()
	pos := term.CursorPosition()
	w.termMu.Unlock()
	return []byte(fmt.Sprintf("\x1b[%d;%dR", pos.Y+1, pos.X+1))
}

func (w *Window) writeToPTY(pty io.Writer, data []byte) {
	w.writeMu.Lock()
	_, _ = pty.Write(data)
	w.writeMu.Unlock()
}

func ctxDone(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	select {
	case <-ctx.Done():
		return true
	default:
		return false
	}
}
