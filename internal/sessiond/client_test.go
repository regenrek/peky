package sessiond

import (
	"context"
	"errors"
	"net"
	"reflect"
	"testing"
	"time"
)

func newTestClient(t *testing.T) (*Client, net.Conn) {
	t.Helper()
	clientConn, serverConn := net.Pipe()
	client := &Client{
		conn:    clientConn,
		pending: make(map[uint64]chan Envelope),
		events:  make(chan Event, 8),
	}
	go client.readLoop()
	t.Cleanup(func() {
		_ = client.Close()
		_ = serverConn.Close()
	})
	return client, serverConn
}

func TestClientSessionNames(t *testing.T) {
	client, server := newTestClient(t)
	errCh := make(chan error, 2)

	go func() {
		env, err := readEnvelope(server)
		if err != nil {
			errCh <- err
			return
		}
		payload, err := encodePayload(SessionNamesResponse{Names: []string{"a", "b"}})
		if err != nil {
			errCh <- err
			return
		}
		resp := Envelope{Kind: EnvelopeResponse, Op: env.Op, ID: env.ID, Payload: payload}
		if err := writeEnvelope(server, resp); err != nil {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	names, err := client.SessionNames(context.Background())
	if err != nil {
		t.Fatalf("SessionNames: %v", err)
	}
	if !reflect.DeepEqual(names, []string{"a", "b"}) {
		t.Fatalf("SessionNames = %#v", names)
	}
	if err := <-errCh; err != nil {
		t.Fatalf("server error: %v", err)
	}
}

func TestClientEvents(t *testing.T) {
	client, server := newTestClient(t)
	payload, err := encodePayload(Event{Type: EventSessionChanged, Session: "demo"})
	if err != nil {
		t.Fatalf("encode payload: %v", err)
	}
	if err := writeEnvelope(server, Envelope{Kind: EnvelopeEvent, Event: EventSessionChanged, Payload: payload}); err != nil {
		t.Fatalf("encode event: %v", err)
	}

	select {
	case evt := <-client.Events():
		if evt.Type != EventSessionChanged || evt.Session != "demo" {
			t.Fatalf("unexpected event: %#v", evt)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for event")
	}
}

func TestClientCallCanceled(t *testing.T) {
	client, server := newTestClient(t)
	received := make(chan struct{}, 1)
	errCh := make(chan error, 2)

	go func() {
		_, err := readEnvelope(server)
		if err != nil {
			errCh <- err
			return
		}
		received <- struct{}{}
		// no response
		errCh <- nil
	}()

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		_, err := client.call(ctx, OpHello, HelloRequest{Version: "v"}, &HelloResponse{})
		errCh <- err
	}()

	select {
	case <-received:
		cancel()
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for request")
	}

	var callErr error
	for i := 0; i < 2; i++ {
		err := <-errCh
		if err == nil {
			continue
		}
		callErr = err
	}
	if !errors.Is(callErr, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", callErr)
	}
}
