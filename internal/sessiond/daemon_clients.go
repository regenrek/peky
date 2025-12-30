package sessiond

import (
	"errors"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

type clientConn struct {
	id     uint64
	conn   net.Conn
	respCh chan outboundEnvelope
	done   chan struct{}

	paneViews       *paneViewScheduler
	paneViewCacheMu sync.Mutex
	paneViewCache   map[paneViewCacheKey]cachedPaneView

	eventMu     sync.Mutex
	eventOrder  []eventKey
	eventItems  map[eventKey]outboundEnvelope
	eventNotify chan struct{}

	closed atomic.Bool
}

type outboundEnvelope struct {
	env     Envelope
	timeout time.Duration
}

func (d *Daemon) newClient(conn net.Conn) *clientConn {
	id := d.clientSeq.Add(1)
	client := &clientConn{
		id:            id,
		conn:          conn,
		respCh:        make(chan outboundEnvelope, 64),
		done:          make(chan struct{}),
		paneViews:     newPaneViewScheduler(),
		paneViewCache: make(map[paneViewCacheKey]cachedPaneView),
	}
	client.initEventQueue()
	return client
}

func (c *clientConn) initEventQueue() {
	if c == nil {
		return
	}
	if c.eventNotify == nil {
		c.eventNotify = make(chan struct{}, 1)
	}
	if c.eventItems == nil {
		c.eventItems = make(map[eventKey]outboundEnvelope)
	}
}

func (c *clientConn) enqueueEvent(key eventKey, env outboundEnvelope) {
	if c == nil {
		return
	}
	c.initEventQueue()
	c.eventMu.Lock()
	if _, ok := c.eventItems[key]; !ok {
		c.eventOrder = append(c.eventOrder, key)
	}
	c.eventItems[key] = env
	c.eventMu.Unlock()
	select {
	case c.eventNotify <- struct{}{}:
	default:
	}
}

func (c *clientConn) popEvent() (outboundEnvelope, bool) {
	if c == nil {
		return outboundEnvelope{}, false
	}
	c.eventMu.Lock()
	if len(c.eventOrder) == 0 {
		c.eventMu.Unlock()
		return outboundEnvelope{}, false
	}
	key := c.eventOrder[0]
	c.eventOrder = c.eventOrder[1:]
	env := c.eventItems[key]
	delete(c.eventItems, key)
	c.eventMu.Unlock()
	return env, true
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
