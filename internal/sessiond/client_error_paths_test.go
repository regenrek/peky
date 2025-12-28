package sessiond

import (
	"context"
	"encoding/gob"
	"testing"
	"time"
)

func TestClientNilAndSendErrors(t *testing.T) {
	var c *Client
	if _, err := c.call(context.Background(), OpHello, nil, nil); err == nil {
		t.Fatalf("expected error for nil client")
	}
	if c.Events() != nil {
		t.Fatalf("expected nil events for nil client")
	}

	c = &Client{}
	if err := c.send(Envelope{Kind: EnvelopeRequest}); err == nil {
		t.Fatalf("expected send error without conn")
	}
}

func TestClientCallErrorResponse(t *testing.T) {
	client, server := newTestClient(t)
	errCh := make(chan error, 1)

	go func() {
		dec := gob.NewDecoder(server)
		enc := gob.NewEncoder(server)
		var env Envelope
		if err := dec.Decode(&env); err != nil {
			errCh <- err
			return
		}
		resp := Envelope{Kind: EnvelopeResponse, Op: env.Op, ID: env.ID, Error: "boom"}
		if err := enc.Encode(resp); err != nil {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	_, err := client.call(context.Background(), OpHello, HelloRequest{Version: "v"}, &HelloResponse{})
	if err == nil || err.Error() != "boom" {
		t.Fatalf("expected error response, got %v", err)
	}
	if err := <-errCh; err != nil {
		t.Fatalf("server error: %v", err)
	}
}

func TestClientCallNilContext(t *testing.T) {
	client, server := newTestClient(t)
	errCh := make(chan error, 1)

	go func() {
		dec := gob.NewDecoder(server)
		enc := gob.NewEncoder(server)
		var env Envelope
		if err := dec.Decode(&env); err != nil {
			errCh <- err
			return
		}
		payload, err := encodePayload(HelloResponse{Version: "v1"})
		if err != nil {
			errCh <- err
			return
		}
		resp := Envelope{Kind: EnvelopeResponse, Op: env.Op, ID: env.ID, Payload: payload}
		if err := enc.Encode(resp); err != nil {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	var resp HelloResponse
	if _, err := client.call(context.TODO(), OpHello, HelloRequest{Version: "v1"}, &resp); err != nil {
		t.Fatalf("call ctx: %v", err)
	}
	if resp.Version != "v1" {
		t.Fatalf("unexpected hello response")
	}
	if err := <-errCh; err != nil {
		t.Fatalf("server error: %v", err)
	}
}

func TestDialMissingSocket(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if _, err := Dial(ctx, "/tmp/does-not-exist.sock", "v1"); err == nil {
		t.Fatalf("expected dial error")
	}
}

func TestClientSendSuccess(t *testing.T) {
	client, server := newTestClient(t)
	done := make(chan error, 1)
	go func() {
		dec := gob.NewDecoder(server)
		var env Envelope
		done <- dec.Decode(&env)
	}()
	if err := client.send(Envelope{Kind: EnvelopeRequest, Op: OpHello, ID: 1}); err != nil {
		t.Fatalf("send: %v", err)
	}
	if err := <-done; err != nil {
		t.Fatalf("server decode: %v", err)
	}
}

func TestClientHelloError(t *testing.T) {
	client, server := newTestClient(t)
	errCh := make(chan error, 1)

	go func() {
		dec := gob.NewDecoder(server)
		enc := gob.NewEncoder(server)
		var env Envelope
		if err := dec.Decode(&env); err != nil {
			errCh <- err
			return
		}
		resp := Envelope{Kind: EnvelopeResponse, Op: env.Op, ID: env.ID, Error: "bad hello"}
		if err := enc.Encode(resp); err != nil {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	if err := client.hello(context.Background()); err == nil {
		t.Fatalf("expected hello error")
	}
	if err := <-errCh; err != nil {
		t.Fatalf("server error: %v", err)
	}
}
