package sessiond

import "testing"

func TestDaemonNilStartStop(t *testing.T) {
	var d *Daemon
	if err := d.Start(); err == nil {
		t.Fatalf("expected error for nil daemon start")
	}
	if err := d.Stop(); err != nil {
		t.Fatalf("expected nil stop for nil daemon")
	}
}
