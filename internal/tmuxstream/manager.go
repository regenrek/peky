package tmuxstream

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/charmbracelet/x/vt"

	"github.com/regenrek/peakypanes/internal/terminal"
)

type Tmux interface {
	PipePane(ctx context.Context, paneID string, shellCommand string) error
	PanePipe(ctx context.Context, paneID string) (cmd string, pid int, err error)
	CapturePaneLines(ctx context.Context, target string, lines int) ([]string, error)
}

type PaneSpec struct {
	PaneID string
	Cols   int
	Rows   int
}

type ViewOptions struct {
	Height     int
	ShowCursor bool
}

type Options struct {
	Executable string
	IdleTTL    int64
	MinRedraw  int64
}

type Manager struct {
	tmux Tmux

	exe   string
	token string

	socketDir  string
	socketPath string
	ln         net.Listener

	events chan struct{}

	mu    sync.Mutex
	panes map[string]*paneState

	closed atomic.Bool

	lastNotify atomic.Int64
	idleTTL    time.Duration
	minRedraw  time.Duration
}

func New(tmux Tmux, opts Options) (*Manager, error) {
	if tmux == nil {
		return nil, errors.New("tmuxstream: tmux is required")
	}

	exe := strings.TrimSpace(opts.Executable)
	if exe == "" {
		var err error
		exe, err = os.Executable()
		if err != nil {
			return nil, fmt.Errorf("tmuxstream: os.Executable: %w", err)
		}
	}

	idle := time.Duration(opts.IdleTTL)
	if idle <= 0 {
		idle = 2 * time.Second
	}
	min := time.Duration(opts.MinRedraw)
	if min <= 0 {
		min = 33 * time.Millisecond
	}

	dir, err := os.MkdirTemp("", "peakypanes-tmuxstream-*")
	if err != nil {
		return nil, fmt.Errorf("tmuxstream: mkdirtemp: %w", err)
	}

	sock := filepath.Join(dir, "stream.sock")
	ln, err := net.Listen("unix", sock)
	if err != nil {
		_ = os.RemoveAll(dir)
		return nil, fmt.Errorf("tmuxstream: listen unix: %w", err)
	}
	_ = os.Chmod(sock, 0o600)

	token := randomToken(16)

	m := &Manager{
		tmux:       tmux,
		exe:        exe,
		token:      token,
		socketDir:  dir,
		socketPath: sock,
		ln:         ln,
		events:     make(chan struct{}, 1),
		panes:      make(map[string]*paneState),
		idleTTL:    idle,
		minRedraw:  min,
	}

	go m.acceptLoop()
	return m, nil
}

func (m *Manager) Events() <-chan struct{} { return m.events }

func (m *Manager) Close(ctx context.Context) error {
	if m == nil {
		return nil
	}
	if m.closed.Swap(true) {
		return nil
	}

	if m.ln != nil {
		_ = m.ln.Close()
	}

	m.mu.Lock()
	ids := make([]string, 0, len(m.panes))
	for id := range m.panes {
		ids = append(ids, id)
	}
	m.mu.Unlock()

	var firstErr error
	for _, id := range ids {
		if err := m.stopPane(ctx, id, true); err != nil && firstErr == nil {
			firstErr = err
		}
	}

	close(m.events)

	if m.socketDir != "" {
		_ = os.RemoveAll(m.socketDir)
	}

	return firstErr
}

func (m *Manager) SetDesired(ctx context.Context, panes []PaneSpec) error {
	if m == nil || m.closed.Load() {
		return nil
	}

	now := time.Now()
	desired := make(map[string]PaneSpec, len(panes))
	for _, p := range panes {
		id := strings.TrimSpace(p.PaneID)
		if id == "" {
			continue
		}
		desired[id] = p
	}

	var toStart []PaneSpec
	var toResize []PaneSpec
	var toStop []string

	m.mu.Lock()
	for id, spec := range desired {
		if ps := m.panes[id]; ps == nil {
			toStart = append(toStart, spec)
		} else {
			ps.touch(now)
			if spec.Cols > 0 && spec.Rows > 0 && (spec.Cols != ps.cols || spec.Rows != ps.rows) {
				toResize = append(toResize, spec)
			}
		}
	}

	for id, ps := range m.panes {
		if _, ok := desired[id]; ok {
			continue
		}
		if now.Sub(ps.lastTouch) > m.idleTTL {
			toStop = append(toStop, id)
		}
	}
	m.mu.Unlock()

	var firstErr error

	for _, spec := range toStart {
		if err := m.startPane(ctx, spec); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	for _, spec := range toResize {
		if err := m.resyncPane(ctx, spec); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	for _, id := range toStop {
		if err := m.stopPane(ctx, id, false); err != nil && firstErr == nil {
			firstErr = err
		}
	}

	return firstErr
}

func (m *Manager) View(paneID string, opts ViewOptions) (string, bool) {
	if m == nil {
		return "", false
	}
	paneID = strings.TrimSpace(paneID)
	if paneID == "" {
		return "", false
	}

	m.mu.Lock()
	ps := m.panes[paneID]
	m.mu.Unlock()
	if ps == nil {
		return "", false
	}

	return ps.view(opts)
}

func (m *Manager) acceptLoop() {
	for {
		conn, err := m.ln.Accept()
		if err != nil {
			if m.closed.Load() {
				return
			}
			continue
		}
		go m.handleConn(conn)
	}
}

func (m *Manager) handleConn(conn net.Conn) {
	defer func() { _ = conn.Close() }()

	r := bufio.NewReader(conn)
	line, err := r.ReadString('\n')
	if err != nil {
		return
	}
	line = strings.TrimSpace(line)
	parts := strings.Fields(line)
	if len(parts) != 3 || parts[0] != "PP1" {
		return
	}
	if parts[1] != m.token {
		return
	}
	paneID := parts[2]

	m.mu.Lock()
	ps := m.panes[paneID]
	m.mu.Unlock()
	if ps == nil {
		return
	}

	ps.attachStream(conn, r, m.notify)
}

func (m *Manager) notify() {
	if m == nil {
		return
	}
	now := time.Now().UnixNano()
	last := m.lastNotify.Load()
	if last > 0 && time.Duration(now-last) < m.minRedraw {
		return
	}
	if !m.lastNotify.CompareAndSwap(last, now) {
		return
	}
	select {
	case m.events <- struct{}{}:
	default:
	}
}

func randomToken(n int) string {
	if n <= 0 {
		n = 16
	}
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

type paneState struct {
	id string

	cols int
	rows int

	termMu sync.Mutex
	term   *vt.Emulator

	cacheMu    sync.Mutex
	cacheDirty bool
	cacheANSI  string
	cacheCols  int
	cacheRows  int
	lastTouch  time.Time

	pipeCommandStr string
	pipeByUs       bool

	streamMu sync.Mutex
	stream   net.Conn
}

func (ps *paneState) touch(t time.Time) { ps.lastTouch = t }

func (ps *paneState) resetTerm(cols, rows int) {
	if cols <= 0 {
		cols = 80
	}
	if rows <= 0 {
		rows = 24
	}
	ps.cols, ps.rows = cols, rows
	ps.termMu.Lock()
	ps.term = vt.NewEmulator(cols, rows)
	_, _ = ps.term.Write([]byte("\x1b[2J\x1b[H"))
	ps.termMu.Unlock()
	ps.markDirty()
}

func (ps *paneState) applySnapshot(lines []string) {
	ps.termMu.Lock()
	if ps.term == nil {
		ps.termMu.Unlock()
		return
	}
	_, _ = ps.term.Write([]byte("\x1b[2J\x1b[H"))
	for _, line := range lines {
		if line == "" {
			_, _ = ps.term.Write([]byte("\r\n"))
			continue
		}
		_, _ = ps.term.Write([]byte(line))
		_, _ = ps.term.Write([]byte("\r\n"))
	}
	ps.termMu.Unlock()
	ps.markDirty()
}

func (ps *paneState) markDirty() {
	ps.cacheMu.Lock()
	ps.cacheDirty = true
	ps.cacheMu.Unlock()
}

func (ps *paneState) attachStream(conn net.Conn, r *bufio.Reader, notify func()) {
	ps.streamMu.Lock()
	if ps.stream != nil {
		_ = ps.stream.Close()
	}
	ps.stream = conn
	ps.streamMu.Unlock()

	go func() {
		buf := make([]byte, 32*1024)
		for {
			n, err := r.Read(buf)
			if n > 0 {
				ps.termMu.Lock()
				if ps.term != nil {
					_, _ = ps.term.Write(buf[:n])
				}
				ps.termMu.Unlock()
				ps.markDirty()
				if notify != nil {
					notify()
				}
			}
			if err != nil {
				return
			}
		}
	}()
}

func (ps *paneState) view(opts ViewOptions) (string, bool) {
	h := opts.Height
	if h <= 0 {
		h = ps.rows
		if h <= 0 {
			h = 24
		}
	}

	var rendered string
	if opts.ShowCursor {
		ps.termMu.Lock()
		term := ps.term
		cols, rows := ps.cols, ps.rows
		if term == nil {
			ps.termMu.Unlock()
			return "", false
		}
		rendered = terminal.RenderEmulatorLipgloss(term, cols, rows, terminal.RenderOptions{ShowCursor: true})
		ps.termMu.Unlock()
	} else {
		ps.cacheMu.Lock()
		dirty := ps.cacheDirty || ps.cacheCols != ps.cols || ps.cacheRows != ps.rows
		cached := ps.cacheANSI
		ps.cacheMu.Unlock()

		if !dirty && cached != "" {
			rendered = cached
		} else {
			ps.termMu.Lock()
			term := ps.term
			if term != nil {
				rendered = term.Render()
			}
			ps.termMu.Unlock()

			ps.cacheMu.Lock()
			ps.cacheANSI = rendered
			ps.cacheDirty = false
			ps.cacheCols = ps.cols
			ps.cacheRows = ps.rows
			ps.cacheMu.Unlock()
		}
	}

	lines := strings.Split(rendered, "\n")
	if len(lines) > h {
		lines = lines[len(lines)-h:]
	}
	return strings.Join(lines, "\n"), true
}
