package vt

import (
	"image/color"
	"io"
	"strings"
	"testing"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
	"github.com/regenrek/peakypanes/internal/limits"
)

func TestBufferInsertDelete(t *testing.T) {
	s := NewScreen(3, 3)
	setLine(s, 0, "abc")
	setLine(s, 1, "def")
	setLine(s, 2, "ghi")

	s.setCursor(0, 1, false)
	if ok := s.InsertLine(1); !ok {
		t.Fatalf("expected InsertLine to succeed")
	}
	if got := lineText(s, 0); got != "abc" {
		t.Fatalf("line0 = %q", got)
	}
	if got := lineText(s, 1); got != "   " {
		t.Fatalf("line1 = %q", got)
	}
	if got := lineText(s, 2); got != "def" {
		t.Fatalf("line2 = %q", got)
	}

	s = NewScreen(3, 3)
	setLine(s, 0, "abc")
	setLine(s, 1, "def")
	setLine(s, 2, "ghi")
	s.setCursor(0, 1, false)
	if ok := s.DeleteLine(1); !ok {
		t.Fatalf("expected DeleteLine to succeed")
	}
	if got := lineText(s, 1); got != "ghi" {
		t.Fatalf("delete line: %q", got)
	}
	if got := lineText(s, 2); got != "   " {
		t.Fatalf("delete line tail: %q", got)
	}

	s = NewScreen(3, 1)
	setLine(s, 0, "abc")
	s.setCursor(1, 0, false)
	s.InsertCell(1)
	if got := lineText(s, 0); got != "a b" {
		t.Fatalf("insert cell: %q", got)
	}
	s.DeleteCell(1)
	if got := lineText(s, 0); got != "ab " {
		t.Fatalf("delete cell: %q", got)
	}
}

func TestScrollbackBasics(t *testing.T) {
	sb := NewScrollback(limits.TerminalScrollbackMaxBytesDefault)
	sb.SetWrapWidth(3)
	sb.PushLineWithWrap(mkLine("one", 3), false)
	sb.PushLineWithWrap(mkLine("two", 3), false)
	if sb.Len() != 2 {
		t.Fatalf("Len = %d", sb.Len())
	}
	row := make([]uv.Cell, 3)
	if !sb.CopyRow(0, row) {
		t.Fatalf("expected CopyRow(0) to succeed")
	}
	if got := cellsToText(row); !strings.Contains(got, "one") {
		t.Fatalf("row0 = %q", got)
	}
	if !sb.CopyRow(1, row) {
		t.Fatalf("expected CopyRow(1) to succeed")
	}
	if got := cellsToText(row); !strings.Contains(got, "two") {
		t.Fatalf("row1 = %q", got)
	}
	sb.SetMaxBytes(-1)
	if got := sb.Len(); got != 0 {
		t.Fatalf("disabled Len = %d", got)
	}
}

func TestScreenBasics(t *testing.T) {
	s := NewScreen(3, 2)
	cell := &uv.Cell{Content: "X", Width: 1}
	s.SetCell(0, 0, cell)
	if got := s.CellAt(0, 0).Content; got != "X" {
		t.Fatalf("CellAt = %q", got)
	}
	s.Fill(&uv.Cell{Content: "Z", Width: 1})
	if got := s.CellAt(1, 1).Content; got != "Z" {
		t.Fatalf("Fill cell = %q", got)
	}
	s.Reset()
	if x, y := s.CursorPosition(); x != 0 || y != 0 {
		t.Fatalf("cursor reset = %d,%d", x, y)
	}
	_ = s.Cursor()

	s.scrollback.SetWrapWidth(3)
	s.scrollback.PushLineWithWrap(mkLine("hi", 3), false)
	if s.ScrollbackLen() != 1 {
		t.Fatalf("scrollback len = %d", s.ScrollbackLen())
	}
	row := make([]uv.Cell, 3)
	if !s.CopyScrollbackRow(0, row) {
		t.Fatalf("expected scrollback row")
	}
	s.SetScrollbackMaxBytes(-1)
	s.ClearScrollback()
	if s.ScrollbackLen() != 0 {
		t.Fatalf("scrollback not cleared")
	}

	visibleCalls := 0
	styleCalls := 0
	s.cb = &Callbacks{
		CursorVisibility: func(bool) { visibleCalls++ },
		CursorStyle:      func(CursorStyle, bool) { styleCalls++ },
	}
	s.HideCursor()
	s.ShowCursor()
	s.setCursorStyle(CursorBar, false)
	if visibleCalls == 0 || styleCalls == 0 {
		t.Fatalf("expected cursor callbacks")
	}

	setLine(s, 0, "abc")
	s.cur = Cursor{Position: uv.Pos(1, 0)}
	s.InsertCell(1)
	if got := s.CellAt(1, 0).Content; got != " " {
		t.Fatalf("InsertCell expected blank, got %q", got)
	}
}

func TestEmulatorBasics(t *testing.T) {
	emu := NewEmulator(3, 2)
	emu.SetCell(0, 0, &uv.Cell{Content: "A", Width: 1})
	if got := emu.CellAt(0, 0).Content; got != "A" {
		t.Fatalf("CellAt = %q", got)
	}
	if !strings.Contains(emu.Render(), "A") {
		t.Fatalf("Render missing cell")
	}
	if !strings.Contains(emu.String(), "A") {
		t.Fatalf("String missing cell")
	}
	if emu.Bounds().Dx() != 3 {
		t.Fatalf("Bounds width = %d", emu.Bounds().Dx())
	}
	if emu.InputPipe() == nil {
		t.Fatalf("expected input pipe")
	}
	emu.resetTabStops()
	if emu.WidthMethod() == nil {
		t.Fatalf("expected width method")
	}
}

func TestEmulatorScrollback(t *testing.T) {
	emu := NewEmulator(3, 2)
	emu.scrs[0].scrollback.SetWrapWidth(3)
	emu.scrs[0].scrollback.PushLineWithWrap(mkLine("sb", 3), false)
	if emu.ScrollbackLen() != 1 {
		t.Fatalf("scrollback len = %d", emu.ScrollbackLen())
	}
	row := make([]uv.Cell, 3)
	if !emu.CopyScrollbackRow(0, row) {
		t.Fatalf("expected scrollback row")
	}
	emu.ClearScrollback()
	if emu.ScrollbackLen() != 0 {
		t.Fatalf("scrollback not cleared")
	}
	emu.SetScrollbackMaxBytes(limits.TerminalScrollbackMaxBytesDefault)
	_ = emu.IsAltScreen()
}

func TestEmulatorColors(t *testing.T) {
	emu := NewEmulator(3, 2)
	cbCalls := 0
	emu.SetCallbacks(Callbacks{
		ForegroundColor: func(color.Color) { cbCalls++ },
		BackgroundColor: func(color.Color) { cbCalls++ },
		CursorColor:     func(color.Color) { cbCalls++ },
	})
	fg := color.RGBA{R: 10, G: 20, B: 30, A: 255}
	emu.SetForegroundColor(fg)
	if emu.ForegroundColor() == nil {
		t.Fatalf("expected foreground color")
	}
	bg := color.RGBA{R: 1, G: 2, B: 3, A: 255}
	emu.SetBackgroundColor(bg)
	if emu.BackgroundColor() == nil {
		t.Fatalf("expected background color")
	}
	cur := color.RGBA{R: 9, G: 9, B: 9, A: 255}
	emu.SetCursorColor(cur)
	if emu.CursorColor() == nil {
		t.Fatalf("expected cursor color")
	}
	emu.SetDefaultForegroundColor(nil)
	emu.SetDefaultBackgroundColor(nil)
	emu.SetDefaultCursorColor(nil)
	emu.SetIndexedColor(5, fg)
	if emu.IndexedColor(5) == nil {
		t.Fatalf("expected indexed color")
	}
	_ = emu.IndexedColor(-1)
	if cbCalls == 0 {
		t.Fatalf("expected callbacks to fire")
	}
}

func TestEmulatorInputAndClose(t *testing.T) {
	emu := NewEmulator(3, 2)
	data := readAfter(t, emu, func() {
		emu.SendText("hi")
	})
	if data != "hi" {
		t.Fatalf("SendText = %q", data)
	}

	data = readAfter(t, emu, func() {
		emu.SendKeys(KeyPressEvent(uv.Key{Code: KeyEnter}))
	})
	if data != "\r" {
		t.Fatalf("SendKeys = %q", data)
	}

	emu.setMode(ansi.ModeBracketedPaste, ansi.ModeSet)
	data = readN(t, emu, len(ansi.BracketedPasteStart)+len("clip")+len(ansi.BracketedPasteEnd), func() {
		emu.Paste("clip")
	})
	if !strings.Contains(data, "clip") || !strings.Contains(data, ansi.BracketedPasteStart) {
		t.Fatalf("Paste = %q", data)
	}

	emu.setMode(ansi.ModeMouseNormal, ansi.ModeSet)
	data = readAfter(t, emu, func() {
		emu.SendMouse(MouseClick(uv.Mouse{X: 1, Y: 1, Button: MouseLeft}))
	})
	if data == "" {
		t.Fatalf("expected mouse data")
	}

	if _, err := emu.WriteString("noop"); err != nil {
		t.Fatalf("WriteString: %v", err)
	}
	if err := emu.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	buf := make([]byte, 1)
	if _, err := emu.Read(buf); err != io.EOF {
		t.Fatalf("expected EOF after close, got %v", err)
	}
}

func TestSafeEmulator(t *testing.T) {
	se := NewSafeEmulator(2, 2)
	se.SetCell(0, 0, &uv.Cell{Content: "A", Width: 1})
	if se.CellAt(0, 0).Content != "A" {
		t.Fatalf("CellAt missing")
	}
	se.Resize(3, 2)
	if se.Width() != 3 || se.Height() != 2 {
		t.Fatalf("Resize failed")
	}
	se.SetForegroundColor(color.White)
	if se.ForegroundColor() == nil {
		t.Fatalf("ForegroundColor missing")
	}
	se.SetBackgroundColor(color.Black)
	se.SetCursorColor(color.White)
	if se.BackgroundColor() == nil || se.CursorColor() == nil {
		t.Fatalf("expected background/cursor color")
	}
	se.SetIndexedColor(1, color.White)
	_ = se.IndexedColor(1)
	se.Render()
	se.CursorPosition()

	screen := &fakeScreen{buf: uv.NewBuffer(3, 2)}
	se.Draw(screen, screen.Bounds())

	if _, err := se.Write([]byte("noop")); err != nil {
		t.Fatalf("Write: %v", err)
	}

	data := readAfterSafe(t, se, func() {
		se.SendText("ok")
	})
	if data != "ok" {
		t.Fatalf("SafeEmulator SendText = %q", data)
	}

	data = readAfterSafe(t, se, func() {
		se.SendKey(KeyPressEvent(uv.Key{Code: KeyEnter}))
	})
	if data != "\r" {
		t.Fatalf("SafeEmulator SendKey = %q", data)
	}

	emu := se.Emulator
	emu.setMode(ansi.ModeMouseNormal, ansi.ModeSet)
	data = readAfterSafe(t, se, func() {
		se.SendMouse(MouseClick(uv.Mouse{X: 1, Y: 1, Button: MouseLeft}))
	})
	if data == "" {
		t.Fatalf("SafeEmulator SendMouse empty")
	}

	emu.setMode(ansi.ModeBracketedPaste, ansi.ModeSet)
	data = readNSafe(t, se, len(ansi.BracketedPasteStart)+len("paste")+len(ansi.BracketedPasteEnd), func() {
		se.Paste("paste")
	})
	if !strings.Contains(data, "paste") {
		t.Fatalf("SafeEmulator Paste = %q", data)
	}
}

func TestDamageBounds(t *testing.T) {
	d := CellDamage{X: 1, Y: 2, Width: 3}
	if b := d.Bounds(); b.Dx() != 3 || b.Dy() != 1 {
		t.Fatalf("cell damage bounds = %#v", b)
	}
	rd := RectDamage(uv.Rect(2, 3, 4, 5))
	if b := rd.Bounds(); b.Dx() != 4 || b.Dy() != 5 {
		t.Fatalf("rect damage bounds = %#v", b)
	}
	if rd.X() != 2 || rd.Y() != 3 || rd.Width() != 4 || rd.Height() != 5 {
		t.Fatalf("rect damage = %#v", rd)
	}
	sd := ScreenDamage{Width: 4, Height: 3}
	if b := sd.Bounds(); b.Dx() != 4 || b.Dy() != 3 {
		t.Fatalf("screen damage bounds = %#v", b)
	}
}

func readAfter(t *testing.T, emu *Emulator, write func()) string {
	t.Helper()
	ch := make(chan string, 1)
	go func() {
		buf := make([]byte, 64)
		n, _ := emu.Read(buf)
		ch <- string(buf[:n])
	}()
	write()
	return <-ch
}

func readN(t *testing.T, emu *Emulator, want int, write func()) string {
	t.Helper()
	ch := make(chan string, 1)
	go func() {
		buf := make([]byte, 64)
		out := make([]byte, 0, want)
		for len(out) < want {
			n, _ := emu.Read(buf)
			out = append(out, buf[:n]...)
		}
		ch <- string(out)
	}()
	write()
	return <-ch
}

func readAfterSafe(t *testing.T, se *SafeEmulator, write func()) string {
	t.Helper()
	ch := make(chan string, 1)
	go func() {
		buf := make([]byte, 64)
		n, _ := se.Read(buf)
		ch <- string(buf[:n])
	}()
	write()
	return <-ch
}

func readNSafe(t *testing.T, se *SafeEmulator, want int, write func()) string {
	t.Helper()
	ch := make(chan string, 1)
	go func() {
		buf := make([]byte, 64)
		out := make([]byte, 0, want)
		for len(out) < want {
			n, _ := se.Read(buf)
			out = append(out, buf[:n]...)
		}
		ch <- string(out)
	}()
	write()
	return <-ch
}

type fakeScreen struct {
	buf *uv.Buffer
}

func (s *fakeScreen) Bounds() uv.Rectangle { return s.buf.Bounds() }
func (s *fakeScreen) CellAt(x, y int) *uv.Cell {
	return s.buf.CellAt(x, y)
}
func (s *fakeScreen) SetCell(x, y int, c *uv.Cell) {
	s.buf.SetCell(x, y, c)
}
func (s *fakeScreen) WidthMethod() uv.WidthMethod { return ansi.WcWidth }

func setLine(s *Screen, y int, text string) {
	if s == nil || y < 0 || y >= s.Height() {
		return
	}
	for x := 0; x < s.Width(); x++ {
		s.SetCell(x, y, nil)
	}
	runes := []rune(text)
	for i := 0; i < len(runes) && i < s.Width(); i++ {
		c := uv.Cell{Content: string(runes[i]), Width: 1}
		s.SetCell(i, y, &c)
	}
}

func lineText(s *Screen, y int) string {
	if s == nil || y < 0 || y >= s.Height() {
		return ""
	}
	var b strings.Builder
	for x := 0; x < s.Width(); x++ {
		cell := s.CellAt(x, y)
		if cell == nil || cell.Width == 0 {
			continue
		}
		if cell.Content == "" {
			b.WriteByte(' ')
		} else {
			b.WriteString(cell.Content)
		}
	}
	return b.String()
}
