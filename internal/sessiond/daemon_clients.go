package sessiond

import (
	"errors"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

type clientConn struct {
	id      uint64
	conn    net.Conn
	respCh  chan outboundEnvelope
	eventCh chan outboundEnvelope
	done    chan struct{}

	paneViews       *paneViewScheduler
	paneViewCacheMu sync.Mutex
	paneViewCache   map[paneViewCacheKey]cachedPaneView

	closed atomic.Bool
}

type outboundEnvelope struct {
	env     Envelope
	timeout time.Duration
}

func (d *Daemon) newClient(conn net.Conn) *clientConn {
	id := d.clientSeq.Add(1)
	return &clientConn{
		id:            id,
		conn:          conn,
		respCh:        make(chan outboundEnvelope, 64),
		eventCh:       make(chan outboundEnvelope, 128),
		done:          make(chan struct{}),
		paneViews:     newPaneViewScheduler(),
		paneViewCache: make(map[paneViewCacheKey]cachedPaneView),
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
	if client == nil {
		return
	}
	if client.closed.Swap(true) {
		return
	}
	if client.paneViews != nil {
		client.paneViews.close()
	}
	close(client.done)
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
