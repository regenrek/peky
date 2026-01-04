package terminal

import (
	"io"
	"testing"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/regenrek/peakypanes/internal/vt"
)

type scrollbackSetEmu struct {
	maxBytes int64
}

func (e *scrollbackSetEmu) Read(p []byte) (int, error)  { return 0, io.EOF }
func (e *scrollbackSetEmu) Write(p []byte) (int, error) { return len(p), nil }
func (e *scrollbackSetEmu) Close() error                { return nil }
func (e *scrollbackSetEmu) Resize(int, int)             {}
func (e *scrollbackSetEmu) Render() string              { return "" }
func (e *scrollbackSetEmu) CellAt(int, int) *uv.Cell    { return nil }
func (e *scrollbackSetEmu) CursorPosition() uv.Position { return uv.Position{} }
func (e *scrollbackSetEmu) SendMouse(uv.MouseEvent)     {}
func (e *scrollbackSetEmu) SetCallbacks(vt.Callbacks)   {}
func (e *scrollbackSetEmu) Height() int                 { return 0 }
func (e *scrollbackSetEmu) Width() int                  { return 0 }
func (e *scrollbackSetEmu) IsAltScreen() bool           { return false }
func (e *scrollbackSetEmu) Cwd() string                 { return "" }
func (e *scrollbackSetEmu) ScrollbackLen() int          { return 0 }
func (e *scrollbackSetEmu) CopyScrollbackRow(int, []uv.Cell) bool {
	return false
}
func (e *scrollbackSetEmu) ClearScrollback() {}
func (e *scrollbackSetEmu) SetScrollbackMaxBytes(maxBytes int64) {
	e.maxBytes = maxBytes
}

func TestWindowSetScrollbackMaxBytesCallsEmulator(t *testing.T) {
	emu := &scrollbackSetEmu{}
	w := &Window{term: emu}
	w.SetScrollbackMaxBytes(1234)
	if got, want := emu.maxBytes, int64(1234); got != want {
		t.Fatalf("maxBytes=%d want %d", got, want)
	}
}
