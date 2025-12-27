// Package terminal provides a minimal PTY + VT (virtual terminal) wrapper
// for PeakyPanes native session manager.
package terminal

import (
	"context"
	"errors"
	"fmt"
	"image/color"
	"io"
	"os"
	"os/exec"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/charmbracelet/lipgloss"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
	xpty "github.com/charmbracelet/x/xpty"
	"github.com/regenrek/peakypanes/internal/vt"
)

// vtEmulator is the subset of the VT emulator API Window depends on.
// This makes scrollback + copy mode testable without a real PTY.
type vtEmulator interface {
	io.Reader
	io.Writer
	Close() error
	Resize(cols, rows int)
	Render() string
	CellAt(x, y int) *uv.Cell
	CursorPosition() uv.Position
	SendMouse(m uv.MouseEvent)
	SetCallbacks(vt.Callbacks)
	Height() int
	Width() int
	IsAltScreen() bool
	ScrollbackLen() int
	ScrollbackLine(index int) []uv.Cell
	ClearScrollback()
	SetScrollbackMaxLines(maxLines int)
}

// Options describes how to start a pane process.
type Options struct {
	ID    string
	Title string

	// Command is executed directly (no shell wrapping).
	// If empty, a platform-appropriate shell is used.
	Command string
	Args    []string
	Dir     string
	Env     []string

	Cols int
	Rows int
}

// Window is a single interactive terminal pane:
// PTY <-> VT emulator, plus rendering helpers.
type Window struct {
	id    string
	title atomic.Value // string

	cmd *exec.Cmd
	pty xpty.Pty

	term   vtEmulator
	termMu sync.Mutex // guards term.Write/Resize/Render/CellAt
	ptyMu  sync.Mutex // guards pty pointer swaps during close

	cols int
	rows int

	updates chan struct{} // coalesced "something changed" signal

	cancel context.CancelFunc
	wg     sync.WaitGroup

	closed atomic.Bool
	exited atomic.Bool

	exitStatus    atomic.Int64
	cursorVisible atomic.Bool
	altScreen     atomic.Bool
	mouseMode     atomic.Uint32

	writeMu sync.Mutex // serialize PTY writes from UI thread

	// Cached ANSI render (cursorless) for fast non-focused panes.
	cacheMu    sync.Mutex
	cacheDirty bool
	cacheANSI  string

	stateMu sync.Mutex
	// ScrollbackMode enables scrollback navigation (no PTY input).
	ScrollbackMode bool
	// CopyMode enables cursor + selection across scrollback+screen.
	CopyMode *CopyMode
	// ScrollbackOffset is how many lines "up" from live view we are (0 == live).
	ScrollbackOffset int

	lastUpdate atomic.Int64 // unix nanos
}

// NewWindow starts a new process attached to a PTY and backed by a VT emulator.
func NewWindow(opts Options) (*Window, error) {
	if strings.TrimSpace(opts.ID) == "" {
		return nil, fmt.Errorf("terminal: window id is required")
	}

	cols := opts.Cols
	rows := opts.Rows
	if cols <= 0 {
		cols = 80
	}
	if rows <= 0 {
		rows = 24
	}

	cmdName := strings.TrimSpace(opts.Command)
	args := opts.Args
	if cmdName == "" {
		cmdName = detectShell()
		args = nil
	}

	ctx, cancel := context.WithCancel(context.Background())

	// #nosec G204 - command is user-controlled by design in a terminal multiplexer.
	cmd := exec.CommandContext(ctx, cmdName, args...)
	if strings.TrimSpace(opts.Dir) != "" {
		cmd.Dir = opts.Dir
	}

	env := append([]string{}, os.Environ()...)
	// Prefer explicit env overrides from caller.
	if len(opts.Env) > 0 {
		env = mergeEnv(env, opts.Env)
	}

	// Set a sensible TERM for interactive apps.
	// If caller already set TERM, keep it.
	if !hasEnv(env, "TERM") {
		env = append(env, "TERM=xterm-256color")
	}
	// Many TUIs look at COLORTERM for 24-bit color.
	if !hasEnv(env, "COLORTERM") {
		env = append(env, "COLORTERM=truecolor")
	}
	env = append(env,
		"TERM_PROGRAM=PEAKYPANES",
		"PEAKYPANES_PANE_ID="+opts.ID,
	)
	cmd.Env = env

	// Platform-specific: controlling terminal, session leader, etc.
	setupPTYCommand(cmd)

	term := vt.NewEmulator(cols, rows)

	pty, err := xpty.NewPty(cols, rows)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("terminal: create pty: %w", err)
	}
	if err := pty.Start(cmd); err != nil {
		cancel()
		_ = pty.Close()
		return nil, fmt.Errorf("terminal: start process: %w", err)
	}
	_ = pty.Resize(cols, rows)

	w := &Window{
		id:         opts.ID,
		cmd:        cmd,
		pty:        pty,
		term:       term,
		cols:       cols,
		rows:       rows,
		updates:    make(chan struct{}, 1),
		cancel:     cancel,
		cacheDirty: true,
	}
	w.title.Store(opts.Title)
	w.cursorVisible.Store(true)
	w.lastUpdate.Store(time.Now().UnixNano())
	term.SetCallbacks(vt.Callbacks{
		CursorVisibility: func(visible bool) {
			w.cursorVisible.Store(visible)
			w.markDirty()
		},
		Title: func(title string) {
			w.title.Store(title)
			w.markDirty()
		},
		AltScreen: func(active bool) {
			w.altScreen.Store(active)
			w.markDirty()
		},
		EnableMode: func(mode ansi.Mode) {
			w.updateMouseMode(mode, true)
		},
		DisableMode: func(mode ansi.Mode) {
			w.updateMouseMode(mode, false)
		},
	})

	w.startIO(ctx)
	w.wg.Add(1)
	go w.waitExit(ctx)

	return w, nil
}

func (w *Window) ID() string { return w.id }

func (w *Window) Title() string {
	if v := w.title.Load(); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func (w *Window) SetTitle(title string) { w.title.Store(title) }

func (w *Window) Exited() bool { return w.exited.Load() }

func (w *Window) ExitStatus() int { return int(w.exitStatus.Load()) }

func (w *Window) PID() int {
	if w == nil || w.cmd == nil || w.cmd.Process == nil {
		return 0
	}
	return w.cmd.Process.Pid
}

func (w *Window) Cols() int { return w.cols }
func (w *Window) Rows() int { return w.rows }

// Updates returns a coalesced signal channel.
// Read from this in Bubble Tea to know when to re-render.
func (w *Window) Updates() <-chan struct{} { return w.updates }

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

// Close shuts down goroutines and releases PTY/VT resources.
func (w *Window) Close() error {
	if w == nil {
		return nil
	}
	if w.closed.Swap(true) {
		return nil
	}

	if w.cancel != nil {
		w.cancel()
	}

	// Closing PTY/VT unblocks readers.
	var pty xpty.Pty
	w.ptyMu.Lock()
	pty = w.pty
	w.pty = nil
	w.ptyMu.Unlock()
	if pty != nil {
		_ = pty.Close()
	}

	var term vtEmulator
	w.termMu.Lock()
	term = w.term
	w.term = nil
	w.termMu.Unlock()
	if term != nil {
		_ = term.Close()
	}

	w.wg.Wait()

	// Safe to close after goroutines exit.
	close(w.updates)
	return nil
}

// ViewANSI returns the VT's own ANSI-rendered screen.
// This is often the most correct rendering (attrs, reverse-video, etc).
// It is cached for speed and assumes cursor styling is handled separately.
func (w *Window) ViewANSI() string {
	if w == nil {
		return ""
	}

	w.cacheMu.Lock()
	if !w.cacheDirty && w.cacheANSI != "" {
		s := w.cacheANSI
		w.cacheMu.Unlock()
		return s
	}
	w.cacheMu.Unlock()

	w.termMu.Lock()
	term := w.term
	if term == nil {
		w.termMu.Unlock()
		return ""
	}
	s := term.Render()
	w.termMu.Unlock()

	w.cacheMu.Lock()
	w.cacheANSI = s
	w.cacheDirty = false
	w.cacheMu.Unlock()

	return s
}

// ViewLipgloss renders the VT screen by walking cells and applying lipgloss styles.
// This is useful when you need to composite the pane inside other lipgloss layouts.
func (w *Window) ViewLipgloss(showCursor bool) string {
	if w == nil {
		return ""
	}
	// Snapshot view state without holding termMu for long.
	w.stateMu.Lock()
	offset := w.ScrollbackOffset
	sbMode := w.ScrollbackMode
	cm := w.CopyMode
	w.stateMu.Unlock()

	w.termMu.Lock()
	defer w.termMu.Unlock()
	term := w.term
	if term == nil {
		return ""
	}

	// Alt screen has no scrollback/copy mode rendering.
	if term.IsAltScreen() {
		offset = 0
		sbMode = false
		cm = nil
	}

	sbLen := term.ScrollbackLen()
	if offset < 0 {
		offset = 0
	}
	if offset > sbLen {
		offset = sbLen
	}
	topAbsY := sbLen - offset
	if topAbsY < 0 {
		topAbsY = 0
	}

	cellAt := func(x, y int) *uv.Cell {
		absY := topAbsY + y
		if absY < sbLen {
			line := term.ScrollbackLine(absY)
			if line == nil || x < 0 || x >= len(line) {
				return nil
			}
			return &line[x]
		}
		screenY := absY - sbLen
		if screenY < 0 || screenY >= w.rows {
			return nil
		}
		return term.CellAt(x, screenY)
	}

	cur := term.CursorPosition()
	opts := RenderOptions{
		ShowCursor: showCursor && w.cursorVisible.Load() && offset == 0 && (cm == nil || !cm.Active),
		CursorX:    cur.X,
		CursorY:    cur.Y,
	}

	// Hide the terminal cursor in scrollback mode.
	if sbMode || offset > 0 {
		opts.ShowCursor = false
	}

	// Copy cursor + selection highlighting.
	if cm != nil && cm.Active {
		opts.ShowCursor = false
		startX, startY := cm.SelStartX, cm.SelStartAbsY
		endX, endY := cm.SelEndX, cm.SelEndAbsY
		if startY > endY || (startY == endY && startX > endX) {
			startX, endX = endX, startX
			startY, endY = endY, startY
		}
		opts.Highlight = func(x, y int) (cursor bool, selection bool) {
			absY := topAbsY + y
			cursor = (absY == cm.CursorAbsY && x == cm.CursorX)
			if cm.Selecting {
				if absY < startY || absY > endY {
					return cursor, false
				}
				if startY == endY {
					return cursor, absY == startY && x >= startX && x <= endX
				}
				if absY == startY {
					return cursor, x >= startX
				}
				if absY == endY {
					return cursor, x <= endX
				}
				return cursor, true
			}
			return cursor, false
		}
	}

	return renderCellsLipgloss(w.cols, w.rows, cellAt, opts)
}

func (w *Window) startIO(ctx context.Context) {
	// PTY -> VT (screen updates)
	w.wg.Add(1)
	go func() {
		defer w.wg.Done()

		buf := make([]byte, 32*1024)
		for {
			w.ptyMu.Lock()
			pty := w.pty
			w.ptyMu.Unlock()
			if pty == nil {
				return
			}

			n, err := pty.Read(buf)
			if n > 0 {
				// Track scrollback growth so scrollback view stays stable.
				oldSB := 0
				newSB := 0

				w.termMu.Lock()
				if w.term != nil {
					oldSB = w.term.ScrollbackLen()
					_, _ = w.term.Write(buf[:n])
					newSB = w.term.ScrollbackLen()
				}
				w.termMu.Unlock()

				if newSB > oldSB {
					w.onScrollbackGrew(newSB - oldSB)
				}
				w.markDirty()
			}
			if err != nil {
				if !errors.Is(err, io.EOF) && !w.closed.Load() {
					// Best-effort: treat as exit.
				}
				return
			}
			select {
			case <-ctx.Done():
				return
			default:
			}
		}
	}()

	// VT -> PTY (terminal query responses like DSR/DA)
	// This is critical for apps like vim/htop/ncurses that query terminal state.
	w.wg.Add(1)
	go func() {
		defer w.wg.Done()

		buf := make([]byte, 4096)
		for {
			w.termMu.Lock()
			term := w.term
			w.termMu.Unlock()
			w.ptyMu.Lock()
			pty := w.pty
			w.ptyMu.Unlock()
			if term == nil || pty == nil {
				return
			}

			n, err := term.Read(buf)
			if n > 0 {
				data := buf[:n]
				if looksLikeCPR(data) {
					w.termMu.Lock()
					pos := term.CursorPosition()
					w.termMu.Unlock()
					data = []byte(fmt.Sprintf("\x1b[%d;%dR", pos.Y+1, pos.X+1))
				}

				w.writeMu.Lock()
				_, _ = pty.Write(data)
				w.writeMu.Unlock()
			}
			if err != nil {
				if !errors.Is(err, io.EOF) && !w.closed.Load() {
					// Best-effort: treat as exit.
				}
				return
			}
			select {
			case <-ctx.Done():
				return
			default:
			}
		}
	}()
}

func (w *Window) waitExit(ctx context.Context) {
	defer w.wg.Done()
	if w.cmd == nil {
		return
	}
	_ = xpty.WaitProcess(ctx, w.cmd)
	if w.cmd.ProcessState != nil {
		w.exitStatus.Store(int64(w.cmd.ProcessState.ExitCode()))
	}
	w.exited.Store(true)
	w.markDirty()
}

func (w *Window) markDirty() {
	w.lastUpdate.Store(time.Now().UnixNano())

	w.cacheMu.Lock()
	w.cacheDirty = true
	w.cacheMu.Unlock()

	// Coalesce signals.
	select {
	case w.updates <- struct{}{}:
	default:
	}
}

// detectShell is a conservative default. In PeakyPanes, panes often run a command;
// for interactive shells, this is used when Options.Command is empty.
func detectShell() string {
	if shell := os.Getenv("SHELL"); strings.TrimSpace(shell) != "" {
		return shell
	}
	// Minimal cross-platform defaults.
	if runtimeGOOS() == "windows" {
		return "cmd.exe"
	}
	for _, s := range []string{"/bin/zsh", "/bin/bash", "/bin/fish", "/bin/sh"} {
		if _, err := os.Stat(s); err == nil {
			return s
		}
	}
	return "/bin/sh"
}

// runtimeGOOS exists to keep detectShell testable if you ever want to.
func runtimeGOOS() string { return runtime.GOOS }

// mergeEnv applies overrides by key (KEY=VALUE).
func mergeEnv(base []string, overrides []string) []string {
	out := append([]string{}, base...)
	index := map[string]int{}
	for i, kv := range out {
		if k := envKey(kv); k != "" {
			index[k] = i
		}
	}
	for _, kv := range overrides {
		k := envKey(kv)
		if k == "" {
			continue
		}
		if i, ok := index[k]; ok {
			out[i] = kv
			continue
		}
		index[k] = len(out)
		out = append(out, kv)
	}
	return out
}

func hasEnv(env []string, key string) bool {
	key = strings.ToUpper(strings.TrimSpace(key))
	if key == "" {
		return false
	}
	prefix := key + "="
	for _, kv := range env {
		if strings.HasPrefix(strings.ToUpper(kv), prefix) {
			return true
		}
	}
	return false
}

func envKey(kv string) string {
	kv = strings.TrimSpace(kv)
	if kv == "" {
		return ""
	}
	i := strings.IndexByte(kv, '=')
	if i <= 0 {
		return ""
	}
	return strings.ToUpper(strings.TrimSpace(kv[:i]))
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

//
// VT -> Lipgloss rendering
//

// RenderOptions controls VT cell rendering.
type RenderOptions struct {
	ShowCursor bool
	CursorX    int
	CursorY    int

	// Optional: override cursor/selection highlights.
	Highlight func(x, y int) (cursor bool, selection bool)
}

// renderCellsLipgloss renders a cols x rows viewport using a cellAt accessor.
func renderCellsLipgloss(cols, rows int, cellAt func(x, y int) *uv.Cell, opts RenderOptions) string {
	if cols <= 0 || rows <= 0 || cellAt == nil {
		return ""
	}

	type key struct {
		fg, bg                  string
		bold, italic, underline bool
		reverse, strike, blink  bool
		cursor                  bool
		selection               bool
	}

	styleCache := make(map[key]lipgloss.Style, 128)

	apply := func(k key, text string) string {
		if text == "" {
			return ""
		}
		if k == (key{}) {
			return text
		}
		st, ok := styleCache[k]
		if !ok {
			st = lipgloss.NewStyle()
			if k.fg != "" {
				st = st.Foreground(lipgloss.Color(k.fg))
			}
			if k.bg != "" {
				st = st.Background(lipgloss.Color(k.bg))
			}
			if k.bold {
				st = st.Bold(true)
			}
			if k.italic {
				st = st.Italic(true)
			}
			if k.underline {
				st = st.Underline(true)
			}
			if k.strike {
				st = st.Strikethrough(true)
			}
			if k.blink {
				st = st.Blink(true)
			}
			if k.reverse {
				st = st.Reverse(true)
			}
			if k.selection {
				st = st.Reverse(true)
			}
			if k.cursor {
				st = st.Reverse(true).Bold(true)
			}
			styleCache[k] = st
		}
		return st.Render(text)
	}

	var b strings.Builder
	b.Grow(cols * rows)

	for y := 0; y < rows; y++ {
		if y > 0 {
			b.WriteByte('\n')
		}

		var run strings.Builder
		var prev key
		var hasPrev bool

		flush := func() {
			if run.Len() == 0 {
				return
			}
			b.WriteString(apply(prev, run.String()))
			run.Reset()
		}

		for x := 0; x < cols; {
			cell := cellAt(x, y)
			if cell != nil && cell.Width == 0 {
				x++
				continue
			}

			ch := " "
			w := 1
			if cell != nil {
				if cell.Content != "" {
					ch = cell.Content
				}
				if cell.Width > 1 {
					w = cell.Width
				}
			}

			kc := keyFromCell(cell)

			cursor := false
			selection := false
			if opts.Highlight != nil {
				cursor, selection = opts.Highlight(x, y)
			} else if opts.ShowCursor && x == opts.CursorX && y == opts.CursorY {
				cursor = true
			}
			if cursor {
				kc.cursor = true
			}
			if selection {
				kc.selection = true
			}

			if !hasPrev {
				prev = kc
				hasPrev = true
			} else if kc != prev {
				flush()
				prev = kc
			}

			run.WriteString(ch)
			x += w
		}
		flush()
	}

	return b.String()
}

// RenderEmulatorLipgloss converts a VT emulator screen into a lipgloss-compatible string.
// It walks uv.Cells and batches runs with the same style to reduce ANSI churn.
func RenderEmulatorLipgloss(term interface {
	CellAt(x, y int) *uv.Cell
	CursorPosition() uv.Position
}, cols, rows int, opts RenderOptions) string {
	if term == nil || cols <= 0 || rows <= 0 {
		return ""
	}

	cursor := term.CursorPosition()
	return renderCellsLipgloss(cols, rows, func(x, y int) *uv.Cell {
		return term.CellAt(x, y)
	}, RenderOptions{
		ShowCursor: opts.ShowCursor,
		CursorX:    cursor.X,
		CursorY:    cursor.Y,
		Highlight:  opts.Highlight,
	})
}

func keyFromCell(cell *uv.Cell) (k struct {
	fg, bg                  string
	bold, italic, underline bool
	reverse, strike, blink  bool
	cursor                  bool
	selection               bool
}) {
	if cell == nil {
		return k
	}

	k.fg = colorToHex(cell.Style.Fg)
	k.bg = colorToHex(cell.Style.Bg)

	attrs := cell.Style.Attrs
	// Reflective feature detection keeps this resilient across uv.Attrs implementations.
	k.bold = attrsBool(attrs, "Bold")
	k.italic = attrsBool(attrs, "Italic")
	k.underline = attrsBool(attrs, "Underline")
	k.blink = attrsBool(attrs, "Blink")

	// Reverse is sometimes named Reverse or Inverse depending on implementation.
	k.reverse = attrsBool(attrs, "Reverse") || attrsBool(attrs, "Inverse")

	// Strikethrough naming varies.
	k.strike = attrsBool(attrs, "Strikethrough") || attrsBool(attrs, "Strike")

	return k
}

func attrsBool(attrs any, method string) bool {
	if attrs == nil || strings.TrimSpace(method) == "" {
		return false
	}
	v := reflect.ValueOf(attrs)

	// Try method on value.
	m := v.MethodByName(method)
	if !m.IsValid() && v.Kind() != reflect.Pointer && v.CanAddr() {
		// Try pointer receiver.
		m = v.Addr().MethodByName(method)
	}
	if !m.IsValid() {
		return false
	}
	t := m.Type()
	if t.NumIn() != 0 || t.NumOut() != 1 || t.Out(0).Kind() != reflect.Bool {
		return false
	}
	out := m.Call(nil)
	return len(out) == 1 && out[0].Bool()
}

func colorToHex(c color.Color) string {
	if c == nil {
		return ""
	}
	r, g, b, a := c.RGBA()
	if a == 0 {
		return ""
	}
	return fmt.Sprintf("#%02x%02x%02x", uint8(r>>8), uint8(g>>8), uint8(b>>8))
}
