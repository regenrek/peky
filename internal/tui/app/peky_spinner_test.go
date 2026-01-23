package app

import "testing"

func TestPekySpinnerFrame(t *testing.T) {
	t.Setenv("PEKY_ICON_SET", "ascii")
	m := &Model{pekySpinnerIndex: 0}
	if got := m.pekySpinnerFrame(); got != "-" {
		t.Fatalf("frame = %q", got)
	}
}

func TestHandlePekySpinnerTick(t *testing.T) {
	t.Setenv("PEKY_ICON_SET", "ascii")
	m := &Model{pekyBusy: true}
	if cmd := m.handlePekySpinnerTick(); cmd == nil {
		t.Fatalf("expected tick cmd")
	}
	if m.pekySpinnerIndex != 1 {
		t.Fatalf("spinner index = %d", m.pekySpinnerIndex)
	}
	m.pekyBusy = false
	if cmd := m.handlePekySpinnerTick(); cmd != nil {
		t.Fatalf("expected nil cmd when idle")
	}
}
