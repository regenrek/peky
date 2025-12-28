package app

import (
	"testing"

	"github.com/regenrek/peakypanes/internal/sessiond"
)

func TestSetAutoStartAndWaitDaemonEvent(t *testing.T) {
	m := newTestModelLite()
	spec := AutoStartSpec{Session: "demo", Path: "/tmp/demo", Layout: "dev", Focus: true}
	m.SetAutoStart(spec)
	if m.autoStart == nil || m.autoStart.Session != "demo" {
		t.Fatalf("expected auto start set")
	}

	if cmd := waitDaemonEvent(nil); cmd != nil {
		t.Fatalf("expected nil cmd for nil client")
	}

	if cmd := waitDaemonEvent(&sessiond.Client{}); cmd == nil {
		t.Fatalf("expected cmd for client")
	}
}
