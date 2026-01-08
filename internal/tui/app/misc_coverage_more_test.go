package app

import "testing"

func TestReconnectDaemonRequiresVersion(t *testing.T) {
	msg := reconnectDaemon("")
	if msg.Err == nil {
		t.Fatalf("expected error")
	}
}

func TestOpenBrowserURLRejectsEmpty(t *testing.T) {
	if err := openBrowserURL(" "); err == nil {
		t.Fatalf("expected error")
	}
}
