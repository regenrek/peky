package terminal

import (
	"io"
	"strings"
	"testing"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/muesli/termenv"

	"github.com/regenrek/peakypanes/internal/vt"
)

type fakeEmu struct {
	cols, rows  int
	sb          [][]uv.Cell
	screen      [][]uv.Cell
	alt         bool
	cursor      uv.Position
	renderValue string
	renderCalls int
	sentMouse   uv.MouseEvent
}

func (f *fakeEmu) Read(p []byte) (int, error)  { return 0, io.EOF }
func (f *fakeEmu) Write(p []byte) (int, error) { return len(p), nil }
func (f *fakeEmu) Close() error                { return nil }
func (f *fakeEmu) Resize(cols, rows int)       { f.cols, f.rows = cols, rows }
func (f *fakeEmu) Render() string              { f.renderCalls++; return f.renderValue }
func (f *fakeEmu) CursorPosition() uv.Position { return f.cursor }
func (f *fakeEmu) SendMouse(event uv.MouseEvent) {
	f.sentMouse = event
}
func (f *fakeEmu) SetCallbacks(vt.Callbacks) {}
func (f *fakeEmu) Height() int               { return f.rows }
func (f *fakeEmu) Width() int                { return f.cols }
func (f *fakeEmu) IsAltScreen() bool         { return f.alt }
func (f *fakeEmu) Cwd() string               { return "" }

func (f *fakeEmu) ScrollbackLen() int { return len(f.sb) }
func (f *fakeEmu) CopyScrollbackRow(i int, dst []uv.Cell) bool {
	if i < 0 || i >= len(f.sb) || len(dst) < f.cols {
		return false
	}
	copy(dst[:f.cols], f.sb[i])
	return true
}
func (f *fakeEmu) ClearScrollback()            { f.sb = nil }
func (f *fakeEmu) SetScrollbackMaxBytes(int64) {}

func (f *fakeEmu) CellAt(x, y int) *uv.Cell {
	if y < 0 || y >= len(f.screen) {
		return nil
	}
	if x < 0 || x >= len(f.screen[y]) {
		return nil
	}
	return &f.screen[y][x]
}

func mkCellsLine(text string, width int) []uv.Cell {
	r := []rune(text)
	out := make([]uv.Cell, width)
	for i := 0; i < width; i++ {
		c := uv.EmptyCell
		c.Width = 1
		if i < len(r) {
			c.Content = string(r[i])
		}
		out[i] = c
	}
	return out
}

func trimLines(s string) []string {
	raw := strings.Split(s, "\n")
	out := make([]string, 0, len(raw))
	for _, ln := range raw {
		out = append(out, strings.TrimRight(ln, " "))
	}
	return out
}

func TestScrollbackOffsetAndRender(t *testing.T) {
	emu := &fakeEmu{
		cols: 4,
		rows: 3,
		sb: [][]uv.Cell{
			mkCellsLine("S0", 4),
			mkCellsLine("S1", 4),
			mkCellsLine("S2", 4),
			mkCellsLine("S3", 4),
			mkCellsLine("S4", 4),
		},
		screen: [][]uv.Cell{
			mkCellsLine("A0", 4),
			mkCellsLine("A1", 4),
			mkCellsLine("A2", 4),
		},
	}

	w := &Window{
		term:    emu,
		cols:    4,
		rows:    3,
		updates: make(chan struct{}, 10),
	}
	w.EnterScrollback()
	w.ScrollUp(2)

	if got := w.GetScrollbackOffset(); got != 2 {
		t.Fatalf("offset=%d", got)
	}

	view := w.ViewLipgloss(false, termenv.TrueColor)
	lines := trimLines(view)

	// offset=2, sbLen=5 -> topAbsY=3 -> S3,S4,A0
	if lines[0] != "S3" {
		t.Fatalf("line0=%q", lines[0])
	}
	if lines[1] != "S4" {
		t.Fatalf("line1=%q", lines[1])
	}
	if lines[2] != "A0" {
		t.Fatalf("line2=%q", lines[2])
	}
}

func TestCopyModeYankSelection(t *testing.T) {
	emu := &fakeEmu{
		cols: 5,
		rows: 3,
		sb: [][]uv.Cell{
			mkCellsLine("hello", 5),
			mkCellsLine("world", 5),
		},
		screen: [][]uv.Cell{
			mkCellsLine("abcde", 5),
			mkCellsLine("fghij", 5),
			mkCellsLine("klmno", 5),
		},
		cursor: uv.Pos(0, 2),
	}

	w := &Window{
		term:    emu,
		cols:    5,
		rows:    3,
		updates: make(chan struct{}, 10),
	}

	// Enter copy mode in live view.
	w.EnterCopyMode()
	w.CopyMove(0, -4)        // move up into scrollback (absolute)
	w.CopyToggleSelect()     // start selection at scrollback line
	w.CopyMove(4, 0)         // extend to end
	text := w.CopyYankText() // yank
	if !strings.Contains(text, "hello") {
		t.Fatalf("expected yank to contain 'hello', got %q", text)
	}
}
