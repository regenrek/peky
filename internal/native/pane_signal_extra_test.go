package native

import (
	"testing"

	"github.com/regenrek/peakypanes/internal/terminal"
)

func TestSignalFromName(t *testing.T) {
	if _, err := signalFromName(""); err != nil {
		t.Fatalf("expected default signal, err=%v", err)
	}
	if _, err := signalFromName("SIGINT"); err != nil {
		t.Fatalf("expected int signal, err=%v", err)
	}
	if _, err := signalFromName("nope"); err == nil {
		t.Fatalf("expected error for unknown signal")
	}
}

func TestSignalPaneErrors(t *testing.T) {
	if err := (*Manager)(nil).SignalPane("p1", ""); err == nil {
		t.Fatalf("expected nil manager error")
	}
	mgr := &Manager{panes: map[string]*Pane{}}
	if err := mgr.SignalPane("", ""); err == nil {
		t.Fatalf("expected empty id error")
	}
	if err := mgr.SignalPane("missing", ""); err == nil {
		t.Fatalf("expected missing pane error")
	}
	mgr.panes["p1"] = &Pane{window: &terminal.Window{}}
	if err := mgr.SignalPane("p1", ""); err == nil {
		t.Fatalf("expected missing process error")
	}
}
