package sessiond

import (
	"context"
	"encoding/gob"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestClientRegistration(t *testing.T) {
	d := &Daemon{clients: make(map[uint64]*clientConn)}
	c1, c2 := net.Pipe()
	defer func() {
		_ = c1.Close()
		_ = c2.Close()
	}()

	client := d.newClient(c1)
	d.registerClient(client)
	if len(d.clients) != 1 {
		t.Fatalf("expected client registered")
	}
	d.removeClient(client)
	if len(d.clients) != 0 {
		t.Fatalf("expected client removed")
	}
}

func TestReadLoopHandlesHello(t *testing.T) {
	d := &Daemon{version: "v1", clients: make(map[uint64]*clientConn)}
	c1, c2 := net.Pipe()
	defer func() { _ = c2.Close() }()

	client := d.newClient(c1)
	d.registerClient(client)
	d.wg.Add(1)
	go d.readLoop(client)

	payload, err := encodePayload(HelloRequest{Version: "v1"})
	if err != nil {
		t.Fatalf("encodePayload: %v", err)
	}
	enc := gob.NewEncoder(c2)
	if err := enc.Encode(Envelope{Kind: EnvelopeRequest, Op: OpHello, ID: 1, Payload: payload}); err != nil {
		t.Fatalf("encode request: %v", err)
	}

	resp := <-client.respCh
	if resp.Kind != EnvelopeResponse || resp.Op != OpHello {
		t.Fatalf("unexpected response: %#v", resp)
	}

	_ = c2.Close()
	d.wg.Wait()
	if len(d.clients) != 0 {
		t.Fatalf("expected client removed after read loop exit")
	}
}

func TestWriteLoopSendsEnvelope(t *testing.T) {
	d := &Daemon{ctx: context.Background()}
	c1, c2 := net.Pipe()
	defer func() {
		_ = c1.Close()
		_ = c2.Close()
	}()

	client := d.newClient(c1)
	d.wg.Add(1)
	go d.writeLoop(client)

	payload, err := encodePayload(HelloResponse{Version: "v1"})
	if err != nil {
		t.Fatalf("encodePayload: %v", err)
	}
	client.respCh <- Envelope{Kind: EnvelopeResponse, Op: OpHello, ID: 1, Payload: payload}

	dec := gob.NewDecoder(c2)
	var got Envelope
	if err := dec.Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.Op != OpHello || got.ID != 1 {
		t.Fatalf("unexpected envelope: %#v", got)
	}

	close(client.done)
	d.wg.Wait()
}

func TestEnsureDaemonRunningWithActiveDaemon(t *testing.T) {
	dir, err := os.MkdirTemp("/tmp", "ppd-daemon-")
	if err != nil {
		t.Fatalf("MkdirTemp: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	socketPath := filepath.Join(dir, "sessiond.sock")
	pidPath := filepath.Join(dir, "sessiond.pid")

	d, err := NewDaemon(DaemonConfig{SocketPath: socketPath, PidPath: pidPath, Version: "test"})
	if err != nil {
		t.Fatalf("NewDaemon: %v", err)
	}
	d.manager = nil
	if err := d.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() { _ = d.Stop() })

	t.Setenv(socketEnv, socketPath)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := EnsureDaemonRunning(ctx, "test"); err != nil {
		t.Fatalf("EnsureDaemonRunning: %v", err)
	}
	client, err := ConnectDefault(ctx, "test")
	if err != nil {
		t.Fatalf("ConnectDefault: %v", err)
	}
	_ = client.Close()
}
