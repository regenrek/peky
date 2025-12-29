package sessiond

import (
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSendEnvelopeAndCloseClient(t *testing.T) {
	client := &clientConn{
		respCh:  make(chan outboundEnvelope, 1),
		eventCh: make(chan outboundEnvelope, 1),
		done:    make(chan struct{}),
	}
	env := Envelope{Kind: EnvelopeResponse, Op: OpHello, ID: 1}
	if err := sendEnvelope(client, env, 50*time.Millisecond); err != nil {
		t.Fatalf("sendEnvelope: %v", err)
	}
	select {
	case got := <-client.respCh:
		if got.env.Op != OpHello || got.env.ID != 1 {
			t.Fatalf("unexpected envelope: %#v", got.env)
		}
	default:
		t.Fatalf("expected envelope to be sent")
	}

	close(client.done)
	if err := sendEnvelope(client, env, 50*time.Millisecond); err == nil {
		t.Fatalf("expected error for closed client")
	}

	c1, c2 := net.Pipe()
	defer func() { _ = c2.Close() }()
	client.conn = c1
	client.respCh = make(chan outboundEnvelope, 1)
	client.eventCh = make(chan outboundEnvelope, 1)
	client.done = make(chan struct{})
	closeClient(client)
	select {
	case <-client.done:
	default:
		t.Fatalf("expected done closed")
	}
}

func TestBroadcast(t *testing.T) {
	d := &Daemon{clients: make(map[uint64]*clientConn)}
	clientA := &clientConn{eventCh: make(chan outboundEnvelope, 1), done: make(chan struct{})}
	clientB := &clientConn{eventCh: make(chan outboundEnvelope, 1), done: make(chan struct{})}
	d.clients[1] = clientA
	d.clients[2] = clientB

	d.broadcast(Event{Type: EventSessionChanged, Session: "demo"})

	for _, client := range []*clientConn{clientA, clientB} {
		select {
		case out := <-client.eventCh:
			var evt Event
			if err := decodePayload(out.env.Payload, &evt); err != nil {
				t.Fatalf("decode payload: %v", err)
			}
			if evt.Type != EventSessionChanged || evt.Session != "demo" {
				t.Fatalf("unexpected event payload: %#v", evt)
			}
		default:
			t.Fatalf("expected broadcast envelope")
		}
	}
}

func TestWritePidFileAndRemoveStaleSocket(t *testing.T) {
	dir := t.TempDir()
	socketPath := filepath.Join(dir, "sessiond.sock")
	pidPath := filepath.Join(dir, "sessiond.pid")

	d := &Daemon{socketPath: socketPath, pidPath: pidPath, version: "v1"}
	if err := d.writePidFile(); err != nil {
		t.Fatalf("writePidFile: %v", err)
	}
	if _, err := os.Stat(pidPath); err != nil {
		t.Fatalf("expected pid file: %v", err)
	}

	if err := os.WriteFile(socketPath, []byte("stale"), 0o600); err != nil {
		t.Fatalf("write stale socket: %v", err)
	}
	if err := d.removeStaleSocket(); err != nil {
		t.Fatalf("removeStaleSocket: %v", err)
	}
	if _, err := os.Stat(socketPath); !os.IsNotExist(err) {
		t.Fatalf("expected socket removed, got %v", err)
	}
}

func TestEnsureSocketDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested")
	socketPath := filepath.Join(dir, "sessiond.sock")
	if err := ensureSocketDir(socketPath); err != nil {
		t.Fatalf("ensureSocketDir: %v", err)
	}
	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("expected dir to exist: %v", err)
	}
}

func TestDaemonStartStop(t *testing.T) {
	dir := t.TempDir()
	socketPath := filepath.Join(dir, "sessiond.sock")
	pidPath := filepath.Join(dir, "sessiond.pid")

	d, err := NewDaemon(DaemonConfig{SocketPath: socketPath, PidPath: pidPath, Version: "test"})
	if err != nil {
		t.Fatalf("NewDaemon: %v", err)
	}

	if err := d.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}

	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	_ = conn.Close()

	if err := d.Stop(); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	if _, err := os.Stat(socketPath); !os.IsNotExist(err) {
		t.Fatalf("expected socket removed, got %v", err)
	}
	if _, err := os.Stat(pidPath); !os.IsNotExist(err) {
		t.Fatalf("expected pid removed, got %v", err)
	}
}
