package sessiond

import "testing"

func TestNewDaemonDefaults(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("HOME", dir)

	d, err := NewDaemon(DaemonConfig{Version: "test"})
	if err != nil {
		t.Fatalf("NewDaemon: %v", err)
	}
	if d.socketPath == "" || d.pidPath == "" {
		t.Fatalf("expected default paths set")
	}
	_ = d.Stop()
}
