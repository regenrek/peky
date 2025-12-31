package sessiond

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/regenrek/peakypanes/internal/native"
)

// Client is a daemon connection used by the UI.
type Client struct {
	conn       net.Conn
	socketPath string
	version    string

	pendingMu sync.Mutex
	pending   map[uint64]chan Envelope
	nextID    atomic.Uint64

	sendMu sync.Mutex
	events chan Event

	eventMu     sync.Mutex
	eventOrder  []eventKey
	eventItems  map[eventKey]Event
	eventNotify chan struct{}
	eventOnce   sync.Once

	doneOnce  sync.Once
	done      chan struct{}
	closeOnce sync.Once

	closed atomic.Bool
}

// Dial connects to an existing daemon.
func Dial(ctx context.Context, socketPath, version string) (*Client, error) {
	conn, err := dialSocket(ctx, socketPath)
	if err != nil {
		return nil, err
	}
	client := &Client{
		conn:       conn,
		socketPath: socketPath,
		version:    version,
		pending:    make(map[uint64]chan Envelope),
	}
	client.initEvents()
	go client.readLoop()
	if err := client.hello(ctx); err != nil {
		_ = conn.Close()
		return nil, err
	}
	return client, nil
}

// Close shuts down the client connection.
func (c *Client) Close() error {
	if c == nil {
		return nil
	}
	if c.closed.Swap(true) {
		return nil
	}
	if c.conn != nil {
		_ = c.conn.Close()
	}
	c.pendingMu.Lock()
	for _, ch := range c.pending {
		close(ch)
	}
	c.pending = nil
	c.pendingMu.Unlock()
	c.closeOnce.Do(func() {
		close(c.doneCh())
	})
	return nil
}

// Events returns a channel of daemon events.
func (c *Client) Events() <-chan Event {
	if c == nil {
		return nil
	}
	return c.events
}

// Version returns the daemon version used by this client.
func (c *Client) Version() string {
	if c == nil {
		return ""
	}
	return c.version
}

// Clone opens a new client connection to the same daemon.
func (c *Client) Clone(ctx context.Context) (*Client, error) {
	if c == nil {
		return nil, errors.New("sessiond: client is nil")
	}
	socketPath := strings.TrimSpace(c.socketPath)
	if socketPath == "" {
		return nil, errors.New("sessiond: socket path unavailable")
	}
	version := strings.TrimSpace(c.version)
	if version == "" {
		return nil, errors.New("sessiond: version unavailable")
	}
	return Dial(ctx, socketPath, version)
}

func (c *Client) hello(ctx context.Context) error {
	_, err := c.call(ctx, OpHello, HelloRequest{Version: c.version}, &HelloResponse{})
	if err != nil {
		return fmt.Errorf("sessiond: hello failed: %w", err)
	}
	return nil
}

func (c *Client) readLoop() {
	c.initEvents()
	for {
		if c.conn == nil {
			return
		}
		if err := c.conn.SetReadDeadline(time.Now().Add(defaultReadTimeout)); err != nil {
			_ = c.Close()
			return
		}
		env, err := readEnvelope(c.conn)
		if err != nil {
			if isTimeout(err) {
				continue
			}
			_ = c.Close()
			return
		}
		switch env.Kind {
		case EnvelopeResponse:
			c.pendingMu.Lock()
			ch := c.pending[env.ID]
			delete(c.pending, env.ID)
			c.pendingMu.Unlock()
			if ch != nil {
				ch <- env
				close(ch)
			}
		case EnvelopeEvent:
			var evt Event
			if err := decodePayload(env.Payload, &evt); err != nil {
				continue
			}
			c.enqueueEvent(evt)
		}
	}
}

type eventKey struct {
	Type   EventType
	PaneID string
	ID     string
}

func eventKeyFor(evt Event) eventKey {
	if evt.ID != "" {
		return eventKey{Type: evt.Type, ID: evt.ID}
	}
	return eventKey{Type: evt.Type, PaneID: evt.PaneID}
}

func (c *Client) initEvents() {
	if c == nil {
		return
	}
	c.eventOnce.Do(func() {
		if c.events == nil {
			c.events = make(chan Event, 16)
		}
		if c.eventItems == nil {
			c.eventItems = make(map[eventKey]Event)
		}
		if c.eventNotify == nil {
			c.eventNotify = make(chan struct{}, 1)
		}
		go c.eventLoop()
	})
}

func (c *Client) enqueueEvent(evt Event) {
	if c == nil {
		return
	}
	key := eventKeyFor(evt)
	c.eventMu.Lock()
	if c.eventItems == nil {
		c.eventItems = make(map[eventKey]Event)
	}
	if _, ok := c.eventItems[key]; !ok {
		c.eventOrder = append(c.eventOrder, key)
	}
	c.eventItems[key] = evt
	c.eventMu.Unlock()
	select {
	case c.eventNotify <- struct{}{}:
	default:
	}
}

func (c *Client) popEvent() (Event, bool) {
	if c == nil {
		return Event{}, false
	}
	c.eventMu.Lock()
	if len(c.eventOrder) == 0 {
		c.eventMu.Unlock()
		return Event{}, false
	}
	key := c.eventOrder[0]
	c.eventOrder = c.eventOrder[1:]
	evt := c.eventItems[key]
	delete(c.eventItems, key)
	c.eventMu.Unlock()
	return evt, true
}

func (c *Client) eventLoop() {
	done := c.doneCh()
	defer func() {
		if c.events != nil {
			close(c.events)
		}
	}()
	for {
		if evt, ok := c.popEvent(); ok {
			select {
			case c.events <- evt:
			case <-done:
				return
			}
			continue
		}
		select {
		case <-c.eventNotify:
			continue
		case <-done:
			return
		}
	}
}

func (c *Client) doneCh() chan struct{} {
	if c == nil {
		return nil
	}
	c.doneOnce.Do(func() {
		c.done = make(chan struct{})
	})
	return c.done
}

func (c *Client) call(ctx context.Context, op Op, req any, out any) (Envelope, error) {
	if c == nil {
		return Envelope{}, errors.New("sessiond: client is nil")
	}
	if c.closed.Load() {
		return Envelope{}, errors.New("sessiond: client closed")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	id := c.nextID.Add(1)
	payload, err := encodePayload(req)
	if err != nil {
		return Envelope{}, err
	}
	respCh := make(chan Envelope, 1)
	c.pendingMu.Lock()
	if c.pending == nil {
		c.pendingMu.Unlock()
		return Envelope{}, errors.New("sessiond: client closed")
	}
	c.pending[id] = respCh
	c.pendingMu.Unlock()
	if err := c.send(ctx, Envelope{Kind: EnvelopeRequest, Op: op, ID: id, Payload: payload}); err != nil {
		c.pendingMu.Lock()
		delete(c.pending, id)
		c.pendingMu.Unlock()
		return Envelope{}, err
	}
	select {
	case <-ctx.Done():
		c.pendingMu.Lock()
		delete(c.pending, id)
		c.pendingMu.Unlock()
		return Envelope{}, ctx.Err()
	case env, ok := <-respCh:
		if !ok {
			return Envelope{}, errors.New("sessiond: response channel closed")
		}
		if env.Error != "" {
			return env, errors.New(env.Error)
		}
		if out != nil {
			if err := decodePayload(env.Payload, out); err != nil {
				return env, err
			}
		}
		return env, nil
	}
}

func (c *Client) send(ctx context.Context, env Envelope) error {
	if c.conn == nil {
		return errors.New("sessiond: connection unavailable")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if c.closed.Load() {
		return errors.New("sessiond: client closed")
	}
	c.sendMu.Lock()
	defer c.sendMu.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}

	deadline := time.Now().Add(defaultWriteTimeout)
	if dl, ok := ctx.Deadline(); ok && dl.Before(deadline) {
		deadline = dl
	}
	if err := c.conn.SetWriteDeadline(deadline); err != nil {
		return err
	}
	if err := writeEnvelope(c.conn, env); err != nil {
		if ctx.Err() != nil && isTimeout(err) {
			return ctx.Err()
		}
		return err
	}
	return nil
}

// SessionNames returns all session names.
func (c *Client) SessionNames(ctx context.Context) ([]string, error) {
	var resp SessionNamesResponse
	if _, err := c.call(ctx, OpSessionNames, nil, &resp); err != nil {
		return nil, err
	}
	return resp.Names, nil
}

// Snapshot fetches session snapshots.
func (c *Client) Snapshot(ctx context.Context, previewLines int) ([]native.SessionSnapshot, uint64, error) {
	resp, err := c.SnapshotState(ctx, previewLines)
	if err != nil {
		return nil, 0, err
	}
	return resp.Sessions, resp.Version, nil
}

// SnapshotState fetches session snapshots with focus state.
func (c *Client) SnapshotState(ctx context.Context, previewLines int) (SnapshotResponse, error) {
	req := SnapshotRequest{PreviewLines: previewLines}
	var resp SnapshotResponse
	if _, err := c.call(ctx, OpSnapshot, req, &resp); err != nil {
		return SnapshotResponse{}, err
	}
	return resp, nil
}

// StartSession starts a session via the daemon.
func (c *Client) StartSession(ctx context.Context, req StartSessionRequest) (StartSessionResponse, error) {
	var resp StartSessionResponse
	if _, err := c.call(ctx, OpStartSession, req, &resp); err != nil {
		return StartSessionResponse{}, err
	}
	return resp, nil
}

// KillSession stops a session.
func (c *Client) KillSession(ctx context.Context, name string) error {
	_, err := c.call(ctx, OpKillSession, KillSessionRequest{Name: name}, nil)
	return err
}

// RenameSession renames a session.
func (c *Client) RenameSession(ctx context.Context, oldName, newName string) (RenameSessionResponse, error) {
	var resp RenameSessionResponse
	if _, err := c.call(ctx, OpRenameSession, RenameSessionRequest{OldName: oldName, NewName: newName}, &resp); err != nil {
		return RenameSessionResponse{}, err
	}
	return resp, nil
}

// RenamePane renames a pane.
func (c *Client) RenamePane(ctx context.Context, sessionName, paneIndex, newTitle string) error {
	_, err := c.call(ctx, OpRenamePane, RenamePaneRequest{SessionName: sessionName, PaneIndex: paneIndex, NewTitle: newTitle}, nil)
	return err
}

// RenamePaneByID renames a pane by pane id.
func (c *Client) RenamePaneByID(ctx context.Context, paneID, newTitle string) error {
	_, err := c.call(ctx, OpRenamePane, RenamePaneRequest{PaneID: paneID, NewTitle: newTitle}, nil)
	return err
}

// SplitPane splits a pane.
func (c *Client) SplitPane(ctx context.Context, sessionName, paneIndex string, vertical bool, percent int) (string, error) {
	var resp SplitPaneResponse
	if _, err := c.call(ctx, OpSplitPane, SplitPaneRequest{SessionName: sessionName, PaneIndex: paneIndex, Vertical: vertical, Percent: percent}, &resp); err != nil {
		return "", err
	}
	return resp.NewIndex, nil
}

// ClosePane closes a pane.
func (c *Client) ClosePane(ctx context.Context, sessionName, paneIndex string) error {
	_, err := c.call(ctx, OpClosePane, ClosePaneRequest{SessionName: sessionName, PaneIndex: paneIndex}, nil)
	return err
}

// ClosePaneByID closes a pane by pane id.
func (c *Client) ClosePaneByID(ctx context.Context, paneID string) error {
	_, err := c.call(ctx, OpClosePane, ClosePaneRequest{PaneID: paneID}, nil)
	return err
}

// SwapPanes swaps two panes in a session.
func (c *Client) SwapPanes(ctx context.Context, sessionName, paneA, paneB string) error {
	_, err := c.call(ctx, OpSwapPanes, SwapPanesRequest{SessionName: sessionName, PaneA: paneA, PaneB: paneB}, nil)
	return err
}

// SendInput forwards raw input to a pane.
func (c *Client) SendInput(ctx context.Context, paneID string, input []byte) error {
	_, err := c.call(ctx, OpSendInput, SendInputRequest{PaneID: paneID, Input: input}, nil)
	return err
}

// SendInputAction forwards raw input and records an action entry.
func (c *Client) SendInputAction(ctx context.Context, paneID string, input []byte, action, summary string) error {
	req := SendInputRequest{PaneID: paneID, Input: input, RecordAction: true, Action: action, Summary: summary}
	_, err := c.call(ctx, OpSendInput, req, nil)
	return err
}

// SendInputScope forwards raw input to a scope and returns per-pane results.
func (c *Client) SendInputScope(ctx context.Context, scope string, input []byte) (SendInputResponse, error) {
	var resp SendInputResponse
	req := SendInputRequest{Scope: scope, Input: input}
	if _, err := c.call(ctx, OpSendInput, req, &resp); err != nil {
		return SendInputResponse{}, err
	}
	return resp, nil
}

// SendInputScopeAction forwards raw input to a scope and records action entries.
func (c *Client) SendInputScopeAction(ctx context.Context, scope string, input []byte, action, summary string) (SendInputResponse, error) {
	var resp SendInputResponse
	req := SendInputRequest{Scope: scope, Input: input, RecordAction: true, Action: action, Summary: summary}
	if _, err := c.call(ctx, OpSendInput, req, &resp); err != nil {
		return SendInputResponse{}, err
	}
	return resp, nil
}

// SendMouse forwards a mouse event to a pane.
func (c *Client) SendMouse(ctx context.Context, paneID string, event MouseEventPayload) error {
	_, err := c.call(ctx, OpSendMouse, SendMouseRequest{PaneID: paneID, Event: event}, nil)
	return err
}

// ResizePane resizes a pane PTY.
func (c *Client) ResizePane(ctx context.Context, paneID string, cols, rows int) error {
	_, err := c.call(ctx, OpResizePane, ResizePaneRequest{PaneID: paneID, Cols: cols, Rows: rows}, nil)
	return err
}

// GetPaneView requests a rendered pane view.
func (c *Client) GetPaneView(ctx context.Context, req PaneViewRequest) (PaneViewResponse, error) {
	var resp PaneViewResponse
	if _, err := c.call(ctx, OpPaneView, req, &resp); err != nil {
		return PaneViewResponse{}, err
	}
	return resp, nil
}

// PaneOutput fetches output lines for a pane.
func (c *Client) PaneOutput(ctx context.Context, req PaneOutputRequest) (PaneOutputResponse, error) {
	var resp PaneOutputResponse
	if _, err := c.call(ctx, OpPaneOutput, req, &resp); err != nil {
		return PaneOutputResponse{}, err
	}
	return resp, nil
}

// PaneSnapshot fetches scrollback snapshot for a pane.
func (c *Client) PaneSnapshot(ctx context.Context, paneID string, rows int) (PaneSnapshotResponse, error) {
	var resp PaneSnapshotResponse
	req := PaneSnapshotRequest{PaneID: paneID, Rows: rows}
	if _, err := c.call(ctx, OpPaneSnapshot, req, &resp); err != nil {
		return PaneSnapshotResponse{}, err
	}
	return resp, nil
}

// PaneHistory returns pane action history.
func (c *Client) PaneHistory(ctx context.Context, req PaneHistoryRequest) (PaneHistoryResponse, error) {
	var resp PaneHistoryResponse
	if _, err := c.call(ctx, OpPaneHistory, req, &resp); err != nil {
		return PaneHistoryResponse{}, err
	}
	return resp, nil
}

// PaneWait waits for output match.
func (c *Client) PaneWait(ctx context.Context, req PaneWaitRequest) (PaneWaitResponse, error) {
	var resp PaneWaitResponse
	if _, err := c.call(ctx, OpPaneWait, req, &resp); err != nil {
		return PaneWaitResponse{}, err
	}
	return resp, nil
}

// PaneTags returns tags for a pane.
func (c *Client) PaneTags(ctx context.Context, paneID string) ([]string, error) {
	req := PaneTagRequest{PaneID: paneID}
	var resp PaneTagListResponse
	if _, err := c.call(ctx, OpPaneTagList, req, &resp); err != nil {
		return nil, err
	}
	return resp.Tags, nil
}

// AddPaneTags adds tags to a pane.
func (c *Client) AddPaneTags(ctx context.Context, paneID string, tags []string) ([]string, error) {
	req := PaneTagRequest{PaneID: paneID, Tags: tags}
	var resp PaneTagListResponse
	if _, err := c.call(ctx, OpPaneTagAdd, req, &resp); err != nil {
		return nil, err
	}
	return resp.Tags, nil
}

// RemovePaneTags removes tags from a pane.
func (c *Client) RemovePaneTags(ctx context.Context, paneID string, tags []string) ([]string, error) {
	req := PaneTagRequest{PaneID: paneID, Tags: tags}
	var resp PaneTagListResponse
	if _, err := c.call(ctx, OpPaneTagRemove, req, &resp); err != nil {
		return nil, err
	}
	return resp.Tags, nil
}

// FocusSession focuses a session in the daemon.
func (c *Client) FocusSession(ctx context.Context, name string) error {
	_, err := c.call(ctx, OpSessionFocus, FocusSessionRequest{Name: name}, nil)
	return err
}

// FocusPane focuses a pane in the daemon.
func (c *Client) FocusPane(ctx context.Context, paneID string) error {
	_, err := c.call(ctx, OpPaneFocus, PaneFocusRequest{PaneID: paneID}, nil)
	return err
}

// SignalPane sends a signal to a pane process.
func (c *Client) SignalPane(ctx context.Context, paneID, signal string) error {
	_, err := c.call(ctx, OpPaneSignal, PaneSignalRequest{PaneID: paneID, Signal: signal}, nil)
	return err
}

// RelayCreate creates a relay.
func (c *Client) RelayCreate(ctx context.Context, cfg RelayConfig) (RelayInfo, error) {
	var resp RelayCreateResponse
	if _, err := c.call(ctx, OpRelayCreate, RelayCreateRequest{Config: cfg}, &resp); err != nil {
		return RelayInfo{}, err
	}
	return resp.Relay, nil
}

// RelayList lists relays.
func (c *Client) RelayList(ctx context.Context) ([]RelayInfo, error) {
	var resp RelayListResponse
	if _, err := c.call(ctx, OpRelayList, nil, &resp); err != nil {
		return nil, err
	}
	return resp.Relays, nil
}

// RelayStop stops a relay.
func (c *Client) RelayStop(ctx context.Context, id string) error {
	_, err := c.call(ctx, OpRelayStop, RelayStopRequest{ID: id}, nil)
	return err
}

// RelayStopAll stops all relays.
func (c *Client) RelayStopAll(ctx context.Context) error {
	_, err := c.call(ctx, OpRelayStopAll, nil, nil)
	return err
}

// EventsReplay returns recent events.
func (c *Client) EventsReplay(ctx context.Context, req EventsReplayRequest) (EventsReplayResponse, error) {
	var resp EventsReplayResponse
	if _, err := c.call(ctx, OpEventsReplay, req, &resp); err != nil {
		return EventsReplayResponse{}, err
	}
	return resp, nil
}

// TerminalAction executes a terminal action on a pane.
func (c *Client) TerminalAction(ctx context.Context, req TerminalActionRequest) (TerminalActionResponse, error) {
	var resp TerminalActionResponse
	if _, err := c.call(ctx, OpTerminalAction, req, &resp); err != nil {
		return TerminalActionResponse{}, err
	}
	return resp, nil
}

// HandleTerminalKey lets the daemon decide if a key should be handled by scrollback/copy mode.
func (c *Client) HandleTerminalKey(ctx context.Context, req TerminalKeyRequest) (TerminalKeyResponse, error) {
	var resp TerminalKeyResponse
	if _, err := c.call(ctx, OpHandleKey, req, &resp); err != nil {
		return TerminalKeyResponse{}, err
	}
	return resp, nil
}

func dialSocket(ctx context.Context, socketPath string) (net.Conn, error) {
	d := net.Dialer{Timeout: 2 * time.Second}
	if ctx == nil {
		ctx = context.Background()
	}
	conn, err := d.DialContext(ctx, "unix", socketPath)
	if err != nil {
		return nil, fmt.Errorf("sessiond: dial %s: %w", socketPath, err)
	}
	return conn, nil
}

func probeDaemon(ctx context.Context, socketPath, version string) error {
	client, err := Dial(ctx, socketPath, version)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || isTimeout(err) {
			return fmt.Errorf("%w: %v", ErrDaemonProbeTimeout, err)
		}
		return err
	}
	return client.Close()
}
