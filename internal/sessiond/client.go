package sessiond

import (
	"context"
	"encoding/gob"
	"errors"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/regenrek/peakypanes/internal/native"
)

// Client is a daemon connection used by the UI.
type Client struct {
	conn       net.Conn
	enc        *gob.Encoder
	dec        *gob.Decoder
	socketPath string
	version    string

	pendingMu sync.Mutex
	pending   map[uint64]chan Envelope
	nextID    atomic.Uint64

	sendMu sync.Mutex
	events chan Event

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
		enc:        gob.NewEncoder(conn),
		dec:        gob.NewDecoder(conn),
		socketPath: socketPath,
		version:    version,
		pending:    make(map[uint64]chan Envelope),
		events:     make(chan Event, 128),
	}
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
	return nil
}

// Events returns a channel of daemon events.
func (c *Client) Events() <-chan Event {
	if c == nil {
		return nil
	}
	return c.events
}

func (c *Client) hello(ctx context.Context) error {
	_, err := c.call(ctx, OpHello, HelloRequest{Version: c.version}, &HelloResponse{})
	if err != nil {
		return fmt.Errorf("sessiond: hello failed: %w", err)
	}
	return nil
}

func (c *Client) readLoop() {
	defer func() {
		if c.events != nil {
			close(c.events)
		}
	}()
	for {
		if c.conn == nil {
			return
		}
		var env Envelope
		if err := c.dec.Decode(&env); err != nil {
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
			select {
			case c.events <- evt:
			default:
			}
		}
	}
}

func (c *Client) call(ctx context.Context, op Op, req any, out any) (Envelope, error) {
	if c == nil {
		return Envelope{}, errors.New("sessiond: client is nil")
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
	c.pending[id] = respCh
	c.pendingMu.Unlock()
	if err := c.send(Envelope{Kind: EnvelopeRequest, Op: op, ID: id, Payload: payload}); err != nil {
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

func (c *Client) send(env Envelope) error {
	if c.conn == nil {
		return errors.New("sessiond: connection unavailable")
	}
	c.sendMu.Lock()
	defer c.sendMu.Unlock()
	if err := c.conn.SetWriteDeadline(time.Now().Add(defaultWriteTimeout)); err != nil {
		return err
	}
	if err := c.enc.Encode(env); err != nil {
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
	var resp SnapshotResponse
	if _, err := c.call(ctx, OpSnapshot, SnapshotRequest{PreviewLines: previewLines}, &resp); err != nil {
		return nil, 0, err
	}
	return resp.Sessions, resp.Version, nil
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
		return err
	}
	return client.Close()
}
