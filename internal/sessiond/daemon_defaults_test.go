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
	if d.restore != nil {
		t.Fatalf("expected restore service disabled by default")
	}
	_ = d.Stop()
}

func TestNewDaemonPprofAddr(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("HOME", dir)

	addr := "127.0.0.1:6061"
	d, err := NewDaemon(DaemonConfig{Version: "test", PprofAddr: addr})
	if err != nil {
		t.Fatalf("NewDaemon: %v", err)
	}
	if d.pprofAddr != addr {
		t.Fatalf("pprof addr = %q", d.pprofAddr)
	}
	_ = d.Stop()
}
