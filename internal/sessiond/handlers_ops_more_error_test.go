package sessiond

import "testing"

func TestHandleSessionNamesNoManager(t *testing.T) {
	d := &Daemon{}
	if _, err := d.handleSessionNames(); err == nil {
		t.Fatalf("expected error without manager")
	}
}

func TestHandleSnapshotDecodeError(t *testing.T) {
	d := &Daemon{}
	if _, err := d.handleSnapshot([]byte("bad")); err == nil {
		t.Fatalf("expected decode error")
	}
}
