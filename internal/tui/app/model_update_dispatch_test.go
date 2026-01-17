package app

import (
	"errors"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	uv "github.com/charmbracelet/ultraviolet"

	tuiinput "github.com/regenrek/peakypanes/internal/tui/input"
)

func TestUpdate_CancelsPekyRunOnEsc(t *testing.T) {
	m := newTestModelLite()
	m.pekyBusy = true
	canceled := false
	m.pekyCancel = func() { canceled = true }

	_, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if !canceled {
		t.Fatalf("expected peky cancel called")
	}
	if m.pekyBusy {
		t.Fatalf("expected pekyBusy false")
	}
	if m.toast.Text == "" {
		t.Fatalf("expected toast set")
	}
}

func TestUpdate_CancelsPekyRunOnEsc_InputKeyMsg(t *testing.T) {
	m := newTestModelLite()
	m.pekyBusy = true
	canceled := false
	m.pekyCancel = func() { canceled = true }

	_, _ = m.Update(tuiinput.KeyMsg{Key: uv.Key{Code: uv.KeyEscape}})
	if !canceled {
		t.Fatalf("expected peky cancel called")
	}
	if m.pekyBusy {
		t.Fatalf("expected pekyBusy false")
	}
}

func TestUpdate_GlobalBindings_InputKeyMsg(t *testing.T) {
	m := newTestModelLite()
	m.setState(StateDashboard)

	_, _ = m.Update(tuiinput.KeyMsg{Key: uv.Key{Code: '?'}})
	if m.state != StateHelp {
		t.Fatalf("expected help state, got %v", m.state)
	}
}

func TestUpdate_WindowSizeDispatch(t *testing.T) {
	m := newTestModelLite()
	_, _ = m.Update(tea.WindowSizeMsg{Width: 120, Height: 44})
	if m.width != 120 || m.height != 44 {
		t.Fatalf("size=%dx%d, want 120x44", m.width, m.height)
	}
}

func TestUpdate_PassivePickerUpdate(t *testing.T) {
	m := newTestModelLite()
	m.setState(StateProjectPicker)

	_, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	if m.state != StateProjectPicker {
		t.Fatalf("expected state to remain project picker")
	}
}

func TestUpdate_PassiveFilterUpdate(t *testing.T) {
	m := newTestModelLite()
	m.filterActive = true
	m.filterInput.Focus()

	_, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	if m.filterInput.Value() != "a" {
		t.Fatalf("filter=%q, want %q", m.filterInput.Value(), "a")
	}
}

func TestUpdate_ToastUpdateHandlers(t *testing.T) {
	m := newTestModelLite()

	_, _ = m.Update(SuccessMsg{Message: "ok"})
	if m.toast.Text == "" {
		t.Fatalf("expected toast set")
	}
	_, _ = m.Update(WarningMsg{Message: "warn"})
	if m.toast.Text == "" {
		t.Fatalf("expected toast set")
	}
	_, _ = m.Update(InfoMsg{Message: "info"})
	if m.toast.Text == "" {
		t.Fatalf("expected toast set")
	}
	_, _ = m.Update(ErrorMsg{Err: errors.New("boom"), Context: "test"})
	if m.toast.Text == "" {
		t.Fatalf("expected toast set")
	}
}

func TestUpdate_RefreshTick_DisconnectedSchedulesReconnect(t *testing.T) {
	m := newTestModelLite()
	m.daemonDisconnected = true

	_, _ = m.Update(refreshTickMsg{})
	if !m.reconnectInFlight {
		t.Fatalf("expected reconnect in flight")
	}
	if m.reconnectBackoff == 0 {
		t.Fatalf("expected reconnect backoff set")
	}
}

func TestUpdate_DaemonReconnectMsg_ErrorKeepsRetrying(t *testing.T) {
	m := newTestModelLite()
	m.daemonDisconnected = true

	_, _ = m.Update(daemonReconnectMsg{Err: errors.New("boom")})
	if !m.reconnectInFlight {
		t.Fatalf("expected reconnect in flight")
	}
	if !m.daemonDisconnected {
		t.Fatalf("expected still disconnected")
	}
}
