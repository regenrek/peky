package sessiond

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/regenrek/peakypanes/internal/native"
)

const (
	defaultReadTimeout  = 2 * time.Minute
	defaultWriteTimeout = 5 * time.Second
	defaultOpTimeout    = 5 * time.Second
	snapshotResponsePad = 200 * time.Millisecond
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
	manager    sessionManager
	listener   net.Listener
	listenerMu sync.RWMutex
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
	id      uint64
	conn    net.Conn
	respCh  chan outboundEnvelope
	eventCh chan outboundEnvelope
	done    chan struct{}
}

type outboundEnvelope struct {
	env     Envelope
	timeout time.Duration
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
		manager:    wrapManager(native.NewManager()),
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
	d.setListener(listener)
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
	if listener := d.clearListener(); listener != nil {
		_ = listener.Close()
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
	listener := d.listenerValue()
	if listener == nil {
		return
	}
	for {
		conn, err := listener.Accept()
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

func (d *Daemon) setListener(listener net.Listener) {
	d.listenerMu.Lock()
	d.listener = listener
	d.listenerMu.Unlock()
}

func (d *Daemon) listenerValue() net.Listener {
	d.listenerMu.RLock()
	listener := d.listener
	d.listenerMu.RUnlock()
	return listener
}

func (d *Daemon) clearListener() net.Listener {
	d.listenerMu.Lock()
	listener := d.listener
	d.listener = nil
	d.listenerMu.Unlock()
	return listener
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
		id:      id,
		conn:    conn,
		respCh:  make(chan outboundEnvelope, 64),
		eventCh: make(chan outboundEnvelope, 128),
		done:    make(chan struct{}),
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
	defer d.shutdownClientConn(client)
	for {
		if err := client.conn.SetReadDeadline(time.Now().Add(defaultReadTimeout)); err != nil {
			return
		}
		env, err := readEnvelope(client.conn)
		if err != nil {
			if isTimeout(err) {
				continue
			}
			return
		}
		if env.Kind != EnvelopeRequest {
			continue
		}
		resp := d.handleRequest(env)
		timeout := d.responseTimeout(env)
		if err := sendEnvelope(client, resp, timeout); err != nil {
			return
		}
	}
}

func (d *Daemon) responseTimeout(env Envelope) time.Duration {
	if env.Op != OpSnapshot {
		return defaultWriteTimeout
	}
	var req SnapshotRequest
	if err := decodePayload(env.Payload, &req); err != nil {
		return defaultWriteTimeout
	}
	if req.MaxDurationMS <= 0 {
		return defaultWriteTimeout
	}
	timeout := time.Duration(req.MaxDurationMS) * time.Millisecond
	if timeout < 0 {
		return defaultWriteTimeout
	}
	return timeout + snapshotResponsePad
}

func (d *Daemon) writeLoop(client *clientConn) {
	defer d.wg.Done()
	for {
		select {
		case <-client.done:
			return
		case <-d.ctx.Done():
			return
		default:
		}

		select {
		case out := <-client.respCh:
			if err := d.writeEnvelopeWithTimeout(client, out.env, out.timeout); err != nil {
				d.shutdownClientConn(client)
				return
			}
			continue
		default:
		}

		select {
		case out := <-client.respCh:
			if err := d.writeEnvelopeWithTimeout(client, out.env, out.timeout); err != nil {
				d.shutdownClientConn(client)
				return
			}
		case out := <-client.eventCh:
			if err := d.writeEnvelopeWithTimeout(client, out.env, out.timeout); err != nil {
				d.shutdownClientConn(client)
				return
			}
		case <-client.done:
			return
		case <-d.ctx.Done():
			return
		}
	}
}

func sendEnvelope(client *clientConn, env Envelope, timeout time.Duration) error {
	if timeout <= 0 {
		timeout = defaultWriteTimeout
	}
	select {
	case <-client.done:
		return errors.New("sessiond: client closed")
	default:
	}
	select {
	case client.respCh <- outboundEnvelope{env: env, timeout: timeout}:
		return nil
	case <-client.done:
		return errors.New("sessiond: client closed")
	case <-time.After(timeout):
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
}

func (d *Daemon) shutdownClientConn(client *clientConn) {
	if client == nil {
		return
	}
	d.removeClient(client)
	closeClient(client)
}

func (d *Daemon) writeEnvelopeWithTimeout(client *clientConn, env Envelope, timeout time.Duration) error {
	if client == nil {
		return errors.New("sessiond: client unavailable")
	}
	if timeout <= 0 {
		timeout = defaultWriteTimeout
	}
	if err := client.conn.SetWriteDeadline(time.Now().Add(timeout)); err != nil {
		return err
	}
	return writeEnvelope(client.conn, env)
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
		case <-client.done:
			continue
		default:
		}
		select {
		case client.eventCh <- outboundEnvelope{env: env, timeout: defaultWriteTimeout}:
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
	} else if errors.Is(err, ErrDaemonProbeTimeout) {
		return err
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
