package sessiond

import (
	"strings"
	"time"

	"github.com/regenrek/peakypanes/internal/native"
)

func (d *Daemon) acceptLoop() {
	defer d.wg.Done()
	listener := d.listenerValue()
	if listener == nil {
		return
	}
	for {
		if d.closing.Load() {
			return
		}
		select {
		case <-d.ctx.Done():
			return
		default:
		}
		conn, err := listener.Accept()
		if err != nil {
			if d.closing.Load() {
				return
			}
			continue
		}
		if d.closing.Load() {
			_ = conn.Close()
			return
		}
		d.spawnMu.Lock()
		if d.closing.Load() {
			d.spawnMu.Unlock()
			_ = conn.Close()
			return
		}
		client := d.newClient(conn)
		d.registerClient(client)
		d.startPaneViewWorkers(client)
		d.wg.Add(1)
		go d.readLoop(client)
		d.wg.Add(1)
		go d.writeLoop(client)
		d.spawnMu.Unlock()
	}
}

func (d *Daemon) eventLoop() {
	defer d.wg.Done()
	if d.manager == nil {
		return
	}
	for event := range d.manager.Events() {
		eventType := event.Type
		if eventType == 0 {
			eventType = native.PaneEventUpdated
		}
		switch eventType {
		case native.PaneEventToast:
			if strings.TrimSpace(event.Toast) == "" {
				continue
			}
			d.broadcast(Event{Type: EventToast, PaneID: event.PaneID, Toast: event.Toast, ToastKind: ToastSuccess})
		default:
			d.broadcast(Event{Type: EventPaneUpdated, PaneID: event.PaneID, PaneUpdateSeq: event.Seq})
		}
	}
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
		if env.Op == OpPaneView && client.paneViews != nil {
			var req PaneViewRequest
			if err := decodePayload(env.Payload, &req); err == nil {
				if paneID, err := requirePaneID(req.PaneID); err == nil {
					client.paneViews.enqueue(paneID, env, req)
					continue
				}
			}
		}
		resp := d.handleRequest(env)
		timeout := d.responseTimeout(env)
		if err := sendEnvelope(client, resp, timeout); err != nil {
			return
		}
	}
}

func (d *Daemon) responseTimeout(env Envelope) time.Duration {
	return defaultWriteTimeout
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

		if out, ok := client.popEvent(); ok {
			if err := d.writeEnvelopeWithTimeout(client, out.env, out.timeout); err != nil {
				d.shutdownClientConn(client)
				return
			}
			continue
		}

		select {
		case out := <-client.respCh:
			if err := d.writeEnvelopeWithTimeout(client, out.env, out.timeout); err != nil {
				d.shutdownClientConn(client)
				return
			}
		case <-client.eventNotify:
			continue
		case <-client.done:
			return
		case <-d.ctx.Done():
			return
		}
	}
}

func (d *Daemon) broadcast(event Event) {
	event = d.recordEvent(event)
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
		client.enqueueEvent(eventKeyFor(event), outboundEnvelope{env: env, timeout: defaultWriteTimeout})
	}
}
