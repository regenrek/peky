// Package terminal provides a minimal PTY + VT (virtual terminal) wrapper
// for PeakyPanes native session manager.
package terminal

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
	xpty "github.com/charmbracelet/x/xpty"
	"github.com/kballard/go-shellquote"
	"github.com/regenrek/peakypanes/internal/limits"
	"github.com/regenrek/peakypanes/internal/termframe"
	"github.com/regenrek/peakypanes/internal/vt"
)

const (
	frameRenderDebounce    = 50 * time.Millisecond
	frameRenderMaxInterval = 250 * time.Millisecond
	frameDemandDefaultTTL  = 2 * time.Second
)

// vtEmulator is the subset of the VT emulator API Window depends on.
// This makes scrollback + copy mode testable without a real PTY.
type vtEmulator interface {
	io.Reader
	io.Writer
	Close() error
	Resize(cols, rows int)
	CellAt(x, y int) *uv.Cell
	CursorPosition() uv.Position
	SendMouse(m uv.MouseEvent)
	SetCallbacks(vt.Callbacks)
	Height() int
	Width() int
	IsAltScreen() bool
	Cwd() string
	ScrollbackLen() int
	CopyScrollbackRow(index int, dst []uv.Cell) bool
	ClearScrollback()
	SetScrollbackMaxBytes(maxBytes int64)
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

	// ScrollbackMaxBytes sets the scrollback byte budget for this pane.
	// 0 uses limits.TerminalScrollbackMaxBytesDefault.
	// Negative disables scrollback.
	ScrollbackMaxBytes int64

	// OnToast is called for terminal-originated toast messages.
	OnToast func(message string)
	// OnFirstRead is called once when the pane receives its first output.
	OnFirstRead func()

	// OnOutput receives raw output bytes from the pane.
	// The payload is only valid until the callback returns; copy it if you retain it.
	OnOutput func(payload []byte)
}

// Window is a single interactive terminal pane:
// PTY <-> VT emulator, plus rendering helpers.
type Window struct {
	id    string
	title atomic.Value // string
	cwd   atomic.Value // string

	createdAt time.Time
	// Stage timing probes (unix nanos).
	ptyCreatedAt     atomic.Int64
	processStartedAt atomic.Int64
	ioStartedAt      atomic.Int64

	cmd *exec.Cmd
	pty xpty.Pty

	term   vtEmulator
	termMu sync.Mutex // guards term.Write/Resize/Render/CellAt
	ptyMu  sync.Mutex // guards pty pointer swaps during close
	// scrollbackScratch is reused for scrollback row inspection (guarded by termMu).
	scrollbackScratch []uv.Cell

	cols int
	rows int

	updates chan struct{} // coalesced "something changed" signal

	cancel context.CancelFunc
	wg     sync.WaitGroup

	closed            atomic.Bool
	exited            atomic.Bool
	inputClosed       atomic.Bool
	inputClosedReason atomic.Int32

	exitStatus    atomic.Int64
	cursorVisible atomic.Bool
	altScreen     atomic.Bool
	mouseMode     atomic.Uint32

	writeMu sync.Mutex // serialize PTY writes from UI thread
	writeCh chan writeRequest

	// Cached frame render for fast pane previews.
	cacheMu    sync.Mutex
	cacheDirty bool
	cacheFrame termframe.Frame
	cacheSeq   uint64

	cacheCols      int
	cacheRows      int
	cacheAltScreen bool

	renderCh chan struct{}

	stateMu sync.Mutex
	// ScrollbackMode enables scrollback navigation (no PTY input).
	ScrollbackMode bool
	// CopyMode enables cursor + selection across scrollback+screen.
	CopyMode *CopyMode
	// ScrollbackOffset is how many lines "up" from live view we are (0 == live).
	ScrollbackOffset int

	// Mouse selection drag state (guarded by stateMu).
	mouseSel mouseSelectionState

	// mouseNow is used for mouse multi-click detection. Defaults to time.Now.
	// This is not guarded by stateMu because it should only be set at construction/tests.
	mouseNow func() time.Time

	toastFn     func(string)
	onFirstRead func()
	outputFn    func([]byte)

	lastUpdate atomic.Int64 // unix nanos

	updateSeq atomic.Uint64

	bytesIn               atomic.Uint64
	bytesOut              atomic.Uint64
	firstReadAt           atomic.Int64
	firstWriteAt          atomic.Int64
	firstReadAfterWriteAt atomic.Int64
	lastWriteAt           atomic.Int64
	firstUpdateAt         atomic.Int64

	frameDemandUntil atomic.Int64
}

// SetScrollbackMaxBytes updates the scrollback byte budget for the underlying VT.
// This is safe to call while the pane is running; it takes effect immediately.
func (w *Window) SetScrollbackMaxBytes(maxBytes int64) {
	if w == nil {
		return
	}
	w.termMu.Lock()
	defer w.termMu.Unlock()
	if w.term != nil {
		w.term.SetScrollbackMaxBytes(maxBytes)
	}
}

// NewWindow starts a new process attached to a PTY and backed by a VT emulator.
func NewWindow(opts Options) (*Window, error) {
	if strings.TrimSpace(opts.ID) == "" {
		return nil, fmt.Errorf("terminal: window id is required")
	}
	startAt := time.Now()

	cols := opts.Cols
	rows := opts.Rows
	if cols <= 0 {
		cols = 80
	}
	if rows <= 0 {
		rows = 24
	}
	cols, rows = limits.Clamp(cols, rows)

	cmdName := strings.TrimSpace(opts.Command)
	args := opts.Args
	if cmdName == "" {
		cmdName, args = detectShellCommand()
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
	scrollbackMax := opts.ScrollbackMaxBytes
	if scrollbackMax == 0 {
		scrollbackMax = limits.TerminalScrollbackMaxBytesDefault
	}
	term.SetScrollbackMaxBytes(scrollbackMax)

	pty, err := xpty.NewPty(cols, rows)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("terminal: create pty: %w", err)
	}
	ptyCreatedAt := time.Now()
	if err := pty.Start(cmd); err != nil {
		cancel()
		_ = pty.Close()
		return nil, fmt.Errorf("terminal: start process: %w", err)
	}
	processStartedAt := time.Now()
	_ = pty.Resize(cols, rows)

	w := &Window{
		id:          opts.ID,
		cmd:         cmd,
		pty:         pty,
		term:        term,
		cols:        cols,
		rows:        rows,
		createdAt:   startAt,
		updates:     make(chan struct{}, 1),
		renderCh:    make(chan struct{}, 1),
		writeCh:     make(chan writeRequest, ptyWriteQueueSize),
		cancel:      cancel,
		cacheDirty:  true,
		toastFn:     opts.OnToast,
		onFirstRead: opts.OnFirstRead,
		outputFn:    opts.OnOutput,
	}
	w.ptyCreatedAt.Store(ptyCreatedAt.UnixNano())
	w.processStartedAt.Store(processStartedAt.UnixNano())
	w.title.Store(opts.Title)
	w.cwd.Store(strings.TrimSpace(opts.Dir))
	w.cursorVisible.Store(true)
	w.lastUpdate.Store(time.Now().UnixNano())
	term.SetCallbacks(vt.Callbacks{
		CursorVisibility: func(visible bool) {
			w.cursorVisible.Store(visible)
			w.markDirty()
		},
		CursorPosition: func(old, new uv.Position) {
			_ = old
			_ = new
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
		WorkingDirectory: func(path string) {
			w.cwd.Store(path)
		},
	})

	w.startIO(ctx)
	w.startFrameRenderer(ctx)
	w.wg.Add(1)
	go w.waitExit(ctx)

	return w, nil
}

func (w *Window) startFrameRenderer(ctx context.Context) {
	w.wg.Add(1)
	go func() {
		defer w.wg.Done()

		var timer *time.Timer
		pending := false
		var lastRender time.Time

		stopTimer := func() {
			if timer == nil {
				return
			}
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			timer = nil
		}

		for {
			var timerCh <-chan time.Time
			if timer != nil {
				timerCh = timer.C
			}
			select {
			case <-ctx.Done():
				stopTimer()
				return
			case <-w.renderCh:
				pending = true
				if !lastRender.IsZero() && time.Since(lastRender) >= frameRenderMaxInterval {
					w.refreshFrameCache()
					lastRender = time.Now()
					pending = false
					stopTimer()
					continue
				}
				if timer == nil {
					timer = time.NewTimer(frameRenderDebounce)
				} else {
					if !timer.Stop() {
						select {
						case <-timer.C:
						default:
						}
					}
					timer.Reset(frameRenderDebounce)
				}
			case <-timerCh:
				if pending {
					w.refreshFrameCache()
					lastRender = time.Now()
					pending = false
				}
				stopTimer()
			}
		}
	}()
}

// RequestFrameRender schedules a background frame render for cached previews.
func (w *Window) RequestFrameRender() {
	if w == nil || w.closed.Load() {
		return
	}
	select {
	case w.renderCh <- struct{}{}:
	default:
	}
}

func (w *Window) ID() string { return w.id }

func (w *Window) UpdateSeq() uint64 {
	if w == nil {
		return 0
	}
	return w.updateSeq.Load()
}

// FrameCacheSeq returns the UpdateSeq value that produced the cached frame.
// If the cache is dirty, it returns 0 so callers fall back to UpdateSeq.
func (w *Window) FrameCacheSeq() uint64 {
	if w == nil {
		return 0
	}
	w.cacheMu.Lock()
	dirty := w.cacheDirty
	seq := w.cacheSeq
	w.cacheMu.Unlock()
	if dirty {
		return 0
	}
	return seq
}

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

// Dead reports whether the pane can no longer accept input.
func (w *Window) Dead() bool {
	if w == nil {
		return true
	}
	return w.closed.Load() || w.exited.Load() || w.inputClosed.Load()
}

func (w *Window) PID() int {
	if w == nil || w.cmd == nil || w.cmd.Process == nil {
		return 0
	}
	return w.cmd.Process.Pid
}

func (w *Window) Cwd() string {
	if w == nil {
		return ""
	}
	if pid := w.PID(); pid > 0 {
		if cwd, ok := cachedPIDCwd(pid); ok && strings.TrimSpace(cwd) != "" {
			w.cwd.Store(cwd)
			return cwd
		}
	}
	if v := w.cwd.Load(); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func (w *Window) Cols() int { return w.cols }
func (w *Window) Rows() int { return w.rows }

func (w *Window) BytesIn() uint64 {
	if w == nil {
		return 0
	}
	return w.bytesIn.Load()
}

func (w *Window) BytesOut() uint64 {
	if w == nil {
		return 0
	}
	return w.bytesOut.Load()
}

// Updates returns a coalesced signal channel.
// Read from this in Bubble Tea to know when to re-render.
func (w *Window) Updates() <-chan struct{} { return w.updates }

func (w *Window) CreatedAt() time.Time {
	if w == nil {
		return time.Time{}
	}
	return w.createdAt
}

func (w *Window) PtyCreatedAt() time.Time {
	if w == nil {
		return time.Time{}
	}
	return timeFromUnixNano(w.ptyCreatedAt.Load())
}

func (w *Window) ProcessStartedAt() time.Time {
	if w == nil {
		return time.Time{}
	}
	return timeFromUnixNano(w.processStartedAt.Load())
}

func (w *Window) IOStartedAt() time.Time {
	if w == nil {
		return time.Time{}
	}
	return timeFromUnixNano(w.ioStartedAt.Load())
}

func (w *Window) FirstReadAt() time.Time {
	if w == nil {
		return time.Time{}
	}
	return timeFromUnixNano(w.firstReadAt.Load())
}

func (w *Window) FirstWriteAt() time.Time {
	if w == nil {
		return time.Time{}
	}
	return timeFromUnixNano(w.firstWriteAt.Load())
}

func (w *Window) FirstReadAfterWriteAt() time.Time {
	if w == nil {
		return time.Time{}
	}
	return timeFromUnixNano(w.firstReadAfterWriteAt.Load())
}

func (w *Window) FirstUpdateAt() time.Time {
	if w == nil {
		return time.Time{}
	}
	return timeFromUnixNano(w.firstUpdateAt.Load())
}

func timeFromUnixNano(value int64) time.Time {
	if value == 0 {
		return time.Time{}
	}
	return time.Unix(0, value)
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
	w.markInputClosed(PaneClosedProcessExited)
	w.markDirty()
}

func (w *Window) markInputClosed(reason PaneClosedReason) {
	if w == nil {
		return
	}
	if reason != PaneClosedUnknown {
		w.inputClosedReason.Store(int32(reason))
	}
	if w.inputClosed.Swap(true) {
		return
	}
	w.detachPTY()
}

func (w *Window) inputClosedReasonValue() PaneClosedReason {
	if w == nil || !w.inputClosed.Load() {
		return PaneClosedUnknown
	}
	if reason := w.inputClosedReason.Load(); reason != 0 {
		return PaneClosedReason(reason)
	}
	return PaneClosedUnknown
}

func (w *Window) detachPTY() {
	var pty xpty.Pty
	w.ptyMu.Lock()
	pty = w.pty
	w.pty = nil
	w.ptyMu.Unlock()
	if pty == nil {
		return
	}
	w.writeMu.Lock()
	_ = pty.Close()
	w.writeMu.Unlock()
}

func (w *Window) markDirty() {
	w.updateSeq.Add(1)

	now := time.Now().UnixNano()
	w.lastUpdate.Store(now)
	w.firstUpdateAt.CompareAndSwap(0, now)

	w.cacheMu.Lock()
	w.cacheDirty = true
	w.cacheMu.Unlock()
	if w.frameDemandActive() {
		w.RequestFrameRender()
	}

	// Coalesce signals.
	select {
	case w.updates <- struct{}{}:
	default:
	}
}

// TouchFrameDemand keeps the frame cache renderer active for a short window.
func (w *Window) TouchFrameDemand(ttl time.Duration) {
	if w == nil || w.closed.Load() {
		return
	}
	if ttl <= 0 {
		ttl = frameDemandDefaultTTL
	}
	until := time.Now().Add(ttl).UnixNano()
	for {
		cur := w.frameDemandUntil.Load()
		if until <= cur {
			return
		}
		if w.frameDemandUntil.CompareAndSwap(cur, until) {
			return
		}
	}
}

func (w *Window) frameDemandActive() bool {
	if w == nil {
		return false
	}
	until := w.frameDemandUntil.Load()
	return until > 0 && time.Now().UnixNano() <= until
}

func (w *Window) notifyToast(message string) {
	if w == nil || strings.TrimSpace(message) == "" {
		return
	}
	if w.toastFn == nil {
		return
	}
	w.toastFn(message)
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

func detectShellCommand() (string, []string) {
	if raw := strings.TrimSpace(os.Getenv("PEAKYPANES_DEFAULT_SHELL_CMD")); raw != "" {
		parts, err := shellquote.Split(raw)
		if err == nil && len(parts) > 0 {
			return parts[0], parts[1:]
		}
	}
	return detectShell(), nil
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
