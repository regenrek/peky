package sessiond

import (
	"context"
	"encoding/gob"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	uv "github.com/charmbracelet/ultraviolet"

	"github.com/regenrek/peakypanes/internal/layout"
	"github.com/regenrek/peakypanes/internal/native"
)

const (
	defaultReadTimeout  = 2 * time.Minute
	defaultWriteTimeout = 5 * time.Second
	defaultOpTimeout    = 5 * time.Second
)

// DaemonConfig configures a session daemon instance.
type DaemonConfig struct {
	Version       string
	SocketPath    string
	PidPath       string
	HandleSignals bool
}

// Daemon owns persistent sessions and serves clients over a local socket.
type Daemon struct {
	manager    *native.Manager
	listener   net.Listener
	socketPath string
	pidPath    string
	version    string

	ctx    context.Context
	cancel context.CancelFunc

	clients   map[uint64]*clientConn
	clientsMu sync.RWMutex
	clientSeq atomic.Uint64

	closing atomic.Bool
	wg      sync.WaitGroup
}

type clientConn struct {
	id     uint64
	conn   net.Conn
	enc    *gob.Encoder
	dec    *gob.Decoder
	sendCh chan Envelope
	done   chan struct{}
}

// NewDaemon creates a daemon instance.
func NewDaemon(cfg DaemonConfig) (*Daemon, error) {
	socketPath := cfg.SocketPath
	if socketPath == "" {
		path, err := DefaultSocketPath()
		if err != nil {
			return nil, err
		}
		socketPath = path
	}
	pidPath := cfg.PidPath
	if pidPath == "" {
		path, err := DefaultPidPath()
		if err != nil {
			return nil, err
		}
		pidPath = path
	}
	ctx, cancel := context.WithCancel(context.Background())
	d := &Daemon{
		manager:    native.NewManager(),
		socketPath: socketPath,
		pidPath:    pidPath,
		version:    cfg.Version,
		ctx:        ctx,
		cancel:     cancel,
		clients:    make(map[uint64]*clientConn),
	}
	if cfg.HandleSignals {
		d.handleSignals()
	}
	return d, nil
}

// Start begins listening for client connections.
func (d *Daemon) Start() error {
	if d == nil {
		return errors.New("sessiond: daemon is nil")
	}
	if err := ensureSocketDir(d.socketPath); err != nil {
		return err
	}
	if err := d.removeStaleSocket(); err != nil {
		return err
	}
	listener, err := net.Listen("unix", d.socketPath)
	if err != nil {
		return fmt.Errorf("sessiond: listen on %s: %w", d.socketPath, err)
	}
	d.listener = listener
	if err := os.Chmod(d.socketPath, 0o700); err != nil {
		_ = listener.Close()
		return fmt.Errorf("sessiond: chmod socket: %w", err)
	}
	if err := d.writePidFile(); err != nil {
		_ = listener.Close()
		return err
	}

	d.wg.Add(2)
	go d.acceptLoop()
	go d.eventLoop()

	log.Printf("sessiond: daemon listening on %s", d.socketPath)
	return nil
}

// Run starts the daemon and blocks until it is stopped.
func (d *Daemon) Run() error {
	if err := d.Start(); err != nil {
		return err
	}
	<-d.ctx.Done()
	return d.shutdown()
}

// Stop signals the daemon to shut down.
func (d *Daemon) Stop() error {
	if d == nil {
		return nil
	}
	if d.closing.Swap(true) {
		return nil
	}
	d.cancel()
	return d.shutdown()
}

func (d *Daemon) shutdown() error {
	if d.listener != nil {
		_ = d.listener.Close()
	}

	d.clientsMu.Lock()
	for _, client := range d.clients {
		closeClient(client)
	}
	d.clients = make(map[uint64]*clientConn)
	d.clientsMu.Unlock()

	if d.manager != nil {
		d.manager.Close()
	}

	d.wg.Wait()

	_ = os.Remove(d.socketPath)
	_ = os.Remove(d.pidPath)
	return nil
}

func (d *Daemon) acceptLoop() {
	defer d.wg.Done()
	for {
		conn, err := d.listener.Accept()
		if err != nil {
			if d.closing.Load() {
				return
			}
			continue
		}
		client := d.newClient(conn)
		d.registerClient(client)
		d.wg.Add(1)
		go d.readLoop(client)
		d.wg.Add(1)
		go d.writeLoop(client)
	}
}

func (d *Daemon) eventLoop() {
	defer d.wg.Done()
	if d.manager == nil {
		return
	}
	for event := range d.manager.Events() {
		d.broadcast(Event{Type: EventPaneUpdated, PaneID: event.PaneID})
	}
}

func (d *Daemon) newClient(conn net.Conn) *clientConn {
	id := d.clientSeq.Add(1)
	return &clientConn{
		id:     id,
		conn:   conn,
		enc:    gob.NewEncoder(conn),
		dec:    gob.NewDecoder(conn),
		sendCh: make(chan Envelope, 64),
		done:   make(chan struct{}),
	}
}

func (d *Daemon) registerClient(client *clientConn) {
	d.clientsMu.Lock()
	d.clients[client.id] = client
	d.clientsMu.Unlock()
}

func (d *Daemon) removeClient(client *clientConn) {
	d.clientsMu.Lock()
	delete(d.clients, client.id)
	d.clientsMu.Unlock()
}

func (d *Daemon) readLoop(client *clientConn) {
	defer d.wg.Done()
	defer func() {
		d.removeClient(client)
		closeClient(client)
	}()
	for {
		if err := client.conn.SetReadDeadline(time.Now().Add(defaultReadTimeout)); err != nil {
			return
		}
		var env Envelope
		if err := client.dec.Decode(&env); err != nil {
			if isTimeout(err) {
				continue
			}
			return
		}
		if env.Kind != EnvelopeRequest {
			continue
		}
		resp := d.handleRequest(env)
		if err := sendEnvelope(client, resp); err != nil {
			return
		}
	}
}

func (d *Daemon) writeLoop(client *clientConn) {
	defer d.wg.Done()
	for {
		select {
		case env, ok := <-client.sendCh:
			if !ok {
				return
			}
			if err := client.conn.SetWriteDeadline(time.Now().Add(defaultWriteTimeout)); err != nil {
				return
			}
			if err := client.enc.Encode(env); err != nil {
				return
			}
		case <-client.done:
			return
		case <-d.ctx.Done():
			return
		}
	}
}

func sendEnvelope(client *clientConn, env Envelope) error {
	select {
	case client.sendCh <- env:
		return nil
	case <-client.done:
		return errors.New("sessiond: client closed")
	case <-time.After(defaultWriteTimeout):
		return errors.New("sessiond: client send timeout")
	}
}

func closeClient(client *clientConn) {
	select {
	case <-client.done:
		return
	default:
		close(client.done)
	}
	if client.conn != nil {
		_ = client.conn.Close()
	}
	close(client.sendCh)
}

func (d *Daemon) handleRequest(env Envelope) Envelope {
	resp := Envelope{
		Kind: EnvelopeResponse,
		Op:   env.Op,
		ID:   env.ID,
	}
	payload, err := d.handleRequestPayload(env)
	if err != nil {
		resp.Error = err.Error()
		return resp
	}
	resp.Payload = payload
	return resp
}

func (d *Daemon) handleRequestPayload(env Envelope) ([]byte, error) {
	switch env.Op {
	case OpHello:
		var req HelloRequest
		if err := decodePayload(env.Payload, &req); err != nil {
			return nil, err
		}
		resp := HelloResponse{Version: d.version, PID: os.Getpid()}
		return encodePayload(resp)
	case OpSessionNames:
		if d.manager == nil {
			return nil, errors.New("sessiond: manager unavailable")
		}
		return encodePayload(SessionNamesResponse{Names: d.manager.SessionNames()})
	case OpSnapshot:
		var req SnapshotRequest
		if err := decodePayload(env.Payload, &req); err != nil {
			return nil, err
		}
		sessions := d.manager.Snapshot(req.PreviewLines)
		resp := SnapshotResponse{Version: d.manager.Version(), Sessions: sessions}
		return encodePayload(resp)
	case OpStartSession:
		var req StartSessionRequest
		if err := decodePayload(env.Payload, &req); err != nil {
			return nil, err
		}
		resp, err := d.startSession(req)
		if err != nil {
			return nil, err
		}
		d.broadcast(Event{Type: EventSessionChanged, Session: resp.Name})
		return encodePayload(resp)
	case OpKillSession:
		var req KillSessionRequest
		if err := decodePayload(env.Payload, &req); err != nil {
			return nil, err
		}
		name, err := validateSessionName(req.Name)
		if err != nil {
			return nil, err
		}
		if err := d.manager.KillSession(name); err != nil {
			return nil, err
		}
		d.broadcast(Event{Type: EventSessionChanged, Session: name})
		return nil, nil
	case OpRenameSession:
		var req RenameSessionRequest
		if err := decodePayload(env.Payload, &req); err != nil {
			return nil, err
		}
		oldName, err := validateSessionName(req.OldName)
		if err != nil {
			return nil, err
		}
		newName, err := validateSessionName(req.NewName)
		if err != nil {
			return nil, err
		}
		if err := d.manager.RenameSession(oldName, newName); err != nil {
			return nil, err
		}
		d.broadcast(Event{Type: EventSessionChanged, Session: newName})
		return encodePayload(RenameSessionResponse{NewName: newName})
	case OpRenamePane:
		var req RenamePaneRequest
		if err := decodePayload(env.Payload, &req); err != nil {
			return nil, err
		}
		sessionName, err := validateSessionName(req.SessionName)
		if err != nil {
			return nil, err
		}
		paneIndex, err := validatePaneIndex(req.PaneIndex)
		if err != nil {
			return nil, err
		}
		newTitle := strings.TrimSpace(req.NewTitle)
		if newTitle == "" {
			return nil, errors.New("sessiond: pane title is required")
		}
		if err := d.manager.RenamePane(sessionName, paneIndex, newTitle); err != nil {
			return nil, err
		}
		return nil, nil
	case OpSplitPane:
		var req SplitPaneRequest
		if err := decodePayload(env.Payload, &req); err != nil {
			return nil, err
		}
		sessionName, err := validateSessionName(req.SessionName)
		if err != nil {
			return nil, err
		}
		paneIndex, err := validatePaneIndex(req.PaneIndex)
		if err != nil {
			return nil, err
		}
		ctx, cancel := context.WithTimeout(context.Background(), defaultOpTimeout)
		defer cancel()
		newIndex, err := d.manager.SplitPane(ctx, sessionName, paneIndex, req.Vertical, req.Percent)
		if err != nil {
			return nil, err
		}
		return encodePayload(SplitPaneResponse{NewIndex: newIndex})
	case OpClosePane:
		var req ClosePaneRequest
		if err := decodePayload(env.Payload, &req); err != nil {
			return nil, err
		}
		sessionName, err := validateSessionName(req.SessionName)
		if err != nil {
			return nil, err
		}
		paneIndex, err := validatePaneIndex(req.PaneIndex)
		if err != nil {
			return nil, err
		}
		ctx, cancel := context.WithTimeout(context.Background(), defaultOpTimeout)
		defer cancel()
		if err := d.manager.ClosePane(ctx, sessionName, paneIndex); err != nil {
			return nil, err
		}
		return nil, nil
	case OpSwapPanes:
		var req SwapPanesRequest
		if err := decodePayload(env.Payload, &req); err != nil {
			return nil, err
		}
		sessionName, err := validateSessionName(req.SessionName)
		if err != nil {
			return nil, err
		}
		paneA, err := validatePaneIndex(req.PaneA)
		if err != nil {
			return nil, err
		}
		paneB, err := validatePaneIndex(req.PaneB)
		if err != nil {
			return nil, err
		}
		if err := d.manager.SwapPanes(sessionName, paneA, paneB); err != nil {
			return nil, err
		}
		return nil, nil
	case OpSendInput:
		var req SendInputRequest
		if err := decodePayload(env.Payload, &req); err != nil {
			return nil, err
		}
		paneID := strings.TrimSpace(req.PaneID)
		if paneID == "" {
			return nil, errors.New("sessiond: pane id is required")
		}
		if err := d.manager.SendInput(paneID, req.Input); err != nil {
			return nil, err
		}
		return nil, nil
	case OpSendMouse:
		var req SendMouseRequest
		if err := decodePayload(env.Payload, &req); err != nil {
			return nil, err
		}
		paneID := strings.TrimSpace(req.PaneID)
		if paneID == "" {
			return nil, errors.New("sessiond: pane id is required")
		}
		event, ok := mousePayloadToEvent(req.Event)
		if !ok {
			return nil, nil
		}
		if err := d.manager.SendMouse(paneID, event); err != nil {
			return nil, err
		}
		return nil, nil
	case OpResizePane:
		var req ResizePaneRequest
		if err := decodePayload(env.Payload, &req); err != nil {
			return nil, err
		}
		paneID := strings.TrimSpace(req.PaneID)
		if paneID == "" {
			return nil, errors.New("sessiond: pane id is required")
		}
		win := d.manager.Window(paneID)
		if win == nil {
			return nil, fmt.Errorf("sessiond: pane %q not found", paneID)
		}
		cols := req.Cols
		rows := req.Rows
		if cols < 1 {
			cols = 1
		}
		if rows < 1 {
			rows = 1
		}
		if err := win.Resize(cols, rows); err != nil {
			return nil, err
		}
		return nil, nil
	case OpPaneView:
		var req PaneViewRequest
		if err := decodePayload(env.Payload, &req); err != nil {
			return nil, err
		}
		paneID := strings.TrimSpace(req.PaneID)
		if paneID == "" {
			return nil, errors.New("sessiond: pane id is required")
		}
		win := d.manager.Window(paneID)
		if win == nil {
			return nil, fmt.Errorf("sessiond: pane %q not found", paneID)
		}
		cols := req.Cols
		rows := req.Rows
		if cols < 1 {
			cols = 1
		}
		if rows < 1 {
			rows = 1
		}
		_ = win.Resize(cols, rows)
		view := ""
		switch req.Mode {
		case PaneViewLipgloss:
			view = win.ViewLipgloss(req.ShowCursor)
		default:
			view = win.ViewANSI()
		}
		hasMouse := win.HasMouseMode()
		allowMotion := win.AllowsMouseMotion()
		resp := PaneViewResponse{
			PaneID:      paneID,
			Cols:        cols,
			Rows:        rows,
			Mode:        req.Mode,
			ShowCursor:  req.ShowCursor,
			View:        view,
			HasMouse:    hasMouse,
			AllowMotion: allowMotion,
		}
		return encodePayload(resp)
	case OpTerminalAction:
		var req TerminalActionRequest
		if err := decodePayload(env.Payload, &req); err != nil {
			return nil, err
		}
		resp, err := d.terminalAction(req)
		if err != nil {
			return nil, err
		}
		return encodePayload(resp)
	case OpHandleKey:
		var req TerminalKeyRequest
		if err := decodePayload(env.Payload, &req); err != nil {
			return nil, err
		}
		resp, err := d.handleTerminalKey(req)
		if err != nil {
			return nil, err
		}
		return encodePayload(resp)
	default:
		return nil, fmt.Errorf("sessiond: unknown op %q", env.Op)
	}
}

func (d *Daemon) startSession(req StartSessionRequest) (StartSessionResponse, error) {
	if d.manager == nil {
		return StartSessionResponse{}, errors.New("sessiond: manager unavailable")
	}
	path, err := validatePath(req.Path)
	if err != nil {
		return StartSessionResponse{}, err
	}
	nameOverride, err := validateOptionalSessionName(req.Name)
	if err != nil {
		return StartSessionResponse{}, err
	}
	loader, err := layout.NewLoader()
	if err != nil {
		return StartSessionResponse{}, err
	}
	loader.SetProjectDir(path)
	if err := loader.LoadAll(); err != nil {
		return StartSessionResponse{}, err
	}
	sessionName := layout.ResolveSessionName(path, nameOverride, loader.GetProjectConfig())
	sessionName = strings.TrimSpace(sessionName)
	if sessionName == "" {
		return StartSessionResponse{}, errors.New("sessiond: session name is required")
	}
	if _, err := validateSessionName(sessionName); err != nil {
		return StartSessionResponse{}, err
	}

	layoutName := strings.TrimSpace(req.LayoutName)
	var selectedLayout *layout.LayoutConfig
	if layoutName != "" {
		selectedLayout, _, err = loader.GetLayout(layoutName)
		if err != nil {
			return StartSessionResponse{}, err
		}
	} else if loader.HasProjectConfig() {
		selectedLayout = loader.GetProjectLayout()
		if selectedLayout == nil {
			selectedLayout, _, _ = loader.GetLayout("dev-3")
		}
	} else {
		selectedLayout, _, _ = loader.GetLayout("dev-3")
	}
	if selectedLayout == nil {
		return StartSessionResponse{}, errors.New("sessiond: no layout found")
	}

	projectName := filepath.Base(path)
	var projectVars map[string]string
	if loader.GetProjectConfig() != nil {
		projectVars = loader.GetProjectConfig().Vars
	}
	expanded := layout.ExpandLayoutVars(selectedLayout, projectVars, path, projectName)

	ctx, cancel := context.WithTimeout(context.Background(), defaultOpTimeout)
	defer cancel()
	_, err = d.manager.StartSession(ctx, native.SessionSpec{
		Name:       sessionName,
		Path:       path,
		Layout:     expanded,
		LayoutName: selectedLayout.Name,
	})
	if err != nil {
		return StartSessionResponse{}, err
	}
	return StartSessionResponse{Name: sessionName, Path: path, LayoutName: selectedLayout.Name}, nil
}

func (d *Daemon) terminalAction(req TerminalActionRequest) (TerminalActionResponse, error) {
	paneID := strings.TrimSpace(req.PaneID)
	if paneID == "" {
		return TerminalActionResponse{}, errors.New("sessiond: pane id is required")
	}
	win := d.manager.Window(paneID)
	if win == nil {
		return TerminalActionResponse{}, fmt.Errorf("sessiond: pane %q not found", paneID)
	}
	switch req.Action {
	case TerminalEnterScrollback:
		win.EnterScrollback()
	case TerminalExitScrollback:
		win.ExitScrollback()
	case TerminalScrollUp:
		win.ScrollUp(req.Lines)
	case TerminalScrollDown:
		win.ScrollDown(req.Lines)
	case TerminalPageUp:
		win.PageUp()
	case TerminalPageDown:
		win.PageDown()
	case TerminalScrollTop:
		win.ScrollToTop()
	case TerminalScrollBottom:
		win.ScrollToBottom()
	case TerminalEnterCopyMode:
		win.EnterCopyMode()
	case TerminalExitCopyMode:
		win.ExitCopyMode()
	case TerminalCopyMove:
		win.CopyMove(req.DeltaX, req.DeltaY)
	case TerminalCopyPageUp:
		win.CopyPageUp()
	case TerminalCopyPageDown:
		win.CopyPageDown()
	case TerminalCopyToggleSelect:
		win.CopyToggleSelect()
	case TerminalCopyYank:
		text := win.CopyYankText()
		return TerminalActionResponse{PaneID: paneID, Text: text}, nil
	default:
		return TerminalActionResponse{}, errors.New("sessiond: unknown terminal action")
	}
	return TerminalActionResponse{PaneID: paneID}, nil
}

func (d *Daemon) handleTerminalKey(req TerminalKeyRequest) (TerminalKeyResponse, error) {
	paneID := strings.TrimSpace(req.PaneID)
	if paneID == "" {
		return TerminalKeyResponse{}, errors.New("sessiond: pane id is required")
	}
	win := d.manager.Window(paneID)
	if win == nil {
		return TerminalKeyResponse{}, fmt.Errorf("sessiond: pane %q not found", paneID)
	}
	key := req.Key

	if win.IsAltScreen() {
		if win.CopyModeActive() {
			win.ExitCopyMode()
		}
		if win.ScrollbackModeActive() || win.GetScrollbackOffset() > 0 {
			win.ExitScrollback()
		}
		return TerminalKeyResponse{Handled: false}, nil
	}

	if win.CopyModeActive() {
		switch key {
		case "esc", "q":
			win.ExitCopyMode()
			return TerminalKeyResponse{Handled: true, Toast: "Copy mode exited", ToastKind: ToastInfo}, nil
		case "up", "k":
			win.CopyMove(0, -1)
			return TerminalKeyResponse{Handled: true}, nil
		case "down", "j":
			win.CopyMove(0, 1)
			return TerminalKeyResponse{Handled: true}, nil
		case "left", "h":
			win.CopyMove(-1, 0)
			return TerminalKeyResponse{Handled: true}, nil
		case "right", "l":
			win.CopyMove(1, 0)
			return TerminalKeyResponse{Handled: true}, nil
		case "pgup":
			win.CopyPageUp()
			return TerminalKeyResponse{Handled: true}, nil
		case "pgdown":
			win.CopyPageDown()
			return TerminalKeyResponse{Handled: true}, nil
		case "v":
			win.CopyToggleSelect()
			return TerminalKeyResponse{Handled: true, Toast: "Selection toggled (v) | Yank (y) | Exit (esc/q)", ToastKind: ToastInfo}, nil
		case "y":
			text := win.CopyYankText()
			win.ExitCopyMode()
			if text == "" {
				return TerminalKeyResponse{Handled: true, Toast: "Nothing to yank", ToastKind: ToastWarning}, nil
			}
			return TerminalKeyResponse{Handled: true, Toast: "Yanked to clipboard", ToastKind: ToastSuccess, YankText: text}, nil
		default:
			return TerminalKeyResponse{Handled: true}, nil
		}
	}

	if win.ScrollbackModeActive() || win.GetScrollbackOffset() > 0 {
		if req.CopyToggle {
			win.EnterCopyMode()
			return TerminalKeyResponse{Handled: true, Toast: "Copy mode: hjkl/arrows | v select | y yank | esc/q exit", ToastKind: ToastInfo}, nil
		}
		if req.ScrollbackToggle {
			win.PageUp()
			return TerminalKeyResponse{Handled: true}, nil
		}
		switch key {
		case "esc", "q":
			win.ExitScrollback()
			return TerminalKeyResponse{Handled: true, Toast: "Scrollback exited", ToastKind: ToastInfo}, nil
		case "up", "k":
			win.ScrollUp(1)
			return TerminalKeyResponse{Handled: true}, nil
		case "down", "j":
			win.ScrollDown(1)
			return TerminalKeyResponse{Handled: true}, nil
		case "pgup":
			win.PageUp()
			return TerminalKeyResponse{Handled: true}, nil
		case "pgdown":
			win.PageDown()
			return TerminalKeyResponse{Handled: true}, nil
		case "home", "g":
			win.ScrollToTop()
			return TerminalKeyResponse{Handled: true}, nil
		case "end", "G":
			win.ScrollToBottom()
			return TerminalKeyResponse{Handled: true}, nil
		default:
			return TerminalKeyResponse{Handled: true}, nil
		}
	}

	if req.ScrollbackToggle {
		win.EnterScrollback()
		win.PageUp()
		return TerminalKeyResponse{Handled: true, Toast: "Scrollback: up/down/pgup/pgdown | Copy (f8) | Exit (esc/q)", ToastKind: ToastInfo}, nil
	}
	if req.CopyToggle {
		win.EnterCopyMode()
		return TerminalKeyResponse{Handled: true, Toast: "Copy mode: hjkl/arrows | v select | y yank | esc/q exit", ToastKind: ToastInfo}, nil
	}

	return TerminalKeyResponse{Handled: false}, nil
}

func (d *Daemon) broadcast(event Event) {
	d.clientsMu.RLock()
	defer d.clientsMu.RUnlock()
	if len(d.clients) == 0 {
		return
	}
	payload, err := encodePayload(event)
	if err != nil {
		return
	}
	env := Envelope{Kind: EnvelopeEvent, Event: event.Type, Payload: payload}
	for _, client := range d.clients {
		select {
		case client.sendCh <- env:
		default:
		}
	}
}

func (d *Daemon) writePidFile() error {
	if d.pidPath == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(d.pidPath), 0o755); err != nil {
		return fmt.Errorf("sessiond: create pid dir: %w", err)
	}
	pid := strconv.Itoa(os.Getpid())
	if err := os.WriteFile(d.pidPath, []byte(pid), 0o600); err != nil {
		return fmt.Errorf("sessiond: write pid file: %w", err)
	}
	return nil
}

func (d *Daemon) removeStaleSocket() error {
	if _, err := os.Stat(d.socketPath); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("sessiond: stat socket: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := probeDaemon(ctx, d.socketPath, d.version); err == nil {
		return fmt.Errorf("sessiond: daemon already running on %s", d.socketPath)
	}
	if err := os.Remove(d.socketPath); err != nil {
		return fmt.Errorf("sessiond: remove stale socket: %w", err)
	}
	return nil
}

func ensureSocketDir(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("sessiond: create socket dir: %w", err)
	}
	return nil
}

func isTimeout(err error) bool {
	if err == nil {
		return false
	}
	if ne, ok := err.(net.Error); ok {
		return ne.Timeout()
	}
	return false
}

func mousePayloadToEvent(payload MouseEventPayload) (uv.MouseEvent, bool) {
	if payload.X < 0 || payload.Y < 0 {
		return nil, false
	}
	mod := uv.KeyMod(0)
	if payload.Shift {
		mod |= uv.ModShift
	}
	if payload.Alt {
		mod |= uv.ModAlt
	}
	if payload.Ctrl {
		mod |= uv.ModCtrl
	}
	mouse := uv.Mouse{X: payload.X, Y: payload.Y, Button: uv.MouseButton(payload.Button), Mod: mod}
	if payload.Wheel {
		return uv.MouseWheelEvent(mouse), true
	}
	switch payload.Action {
	case MouseActionPress:
		return uv.MouseClickEvent(mouse), true
	case MouseActionRelease:
		return uv.MouseReleaseEvent(mouse), true
	case MouseActionMotion:
		return uv.MouseMotionEvent(mouse), true
	default:
		return nil, false
	}
}
