package limits

import "testing"

func TestScrollbackMaxBytesPerPane(t *testing.T) {
	if got, want := ScrollbackMaxBytesPerPane(0), TerminalScrollbackMaxBytesDefault; got != want {
		t.Fatalf("active=0 got %d want %d", got, want)
	}

	if got, want := ScrollbackMaxBytesPerPane(1), TerminalScrollbackMaxBytesDefault; got != want {
		t.Fatalf("active=1 got %d want %d", got, want)
	}

	if got := ScrollbackMaxBytesPerPane(2); got <= 0 {
		t.Fatalf("active=2 got %d want > 0", got)
	}

	// Large pane counts should reduce per-pane budget to respect total cap.
	got := ScrollbackMaxBytesPerPane(100)
	if got >= TerminalScrollbackMaxBytesDefault {
		t.Fatalf("active=100 got %d want < %d", got, TerminalScrollbackMaxBytesDefault)
	}

	// Extreme pane counts should disable scrollback instead of returning 0 (0 means default).
	if got := ScrollbackMaxBytesPerPane(int(TerminalScrollbackTotalMaxBytesDefault) + 1); got != -1 {
		t.Fatalf("extreme active got %d want -1", got)
	}
}
