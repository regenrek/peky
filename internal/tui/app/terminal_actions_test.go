package app

import (
	"context"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/regenrek/peakypanes/internal/sessiond"
)

func TestTerminalControlHelpers(t *testing.T) {
	if !isTerminalControlKey("esc") {
		t.Fatalf("expected esc to be control key")
	}
	if isTerminalControlKey("z") {
		t.Fatalf("expected z to be non-control key")
	}
	if !shouldHandleTerminalKey("z", true, false) {
		t.Fatalf("expected scroll toggle to force handling")
	}
}

func TestToastFromTerminalResponse(t *testing.T) {
	msg := toastFromTerminalResponse(sessiond.TerminalKeyResponse{Toast: "ok", ToastKind: sessiond.ToastSuccess})
	if _, ok := msg.(SuccessMsg); !ok {
		t.Fatalf("expected success msg")
	}
	msg = toastFromTerminalResponse(sessiond.TerminalKeyResponse{Toast: "warn", ToastKind: sessiond.ToastWarning})
	if _, ok := msg.(WarningMsg); !ok {
		t.Fatalf("expected warning msg")
	}
	msg = toastFromTerminalResponse(sessiond.TerminalKeyResponse{Toast: "info", ToastKind: sessiond.ToastInfo})
	if _, ok := msg.(InfoMsg); !ok {
		t.Fatalf("expected info msg")
	}
}

func TestHandleTerminalKeyCmdNilClient(t *testing.T) {
	m := newTestModelLite()
	m.client = nil
	if cmd := m.handleTerminalKeyCmd(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}); cmd != nil {
		t.Fatalf("expected nil cmd when client missing")
	}
}

func TestSendPaneInputCmdErrors(t *testing.T) {
	m := newTestModelLite()
	m.client = nil
	cmd := m.sendPaneInputCmd([]byte("hi"), "ctx")
	if cmd == nil {
		t.Fatalf("expected error cmd when client missing")
	}
	if msg := cmd(); msg == nil {
		t.Fatalf("expected error msg")
	}

	m = newTestModelLite()
	m.data = DashboardData{}
	cmd = m.sendPaneInputCmd([]byte("hi"), "ctx")
	if msg := cmd(); msg == nil {
		t.Fatalf("expected warning msg")
	}
}

func TestHandleTerminalKeyResponseHandled(t *testing.T) {
	m := newTestModelLite()
	m.client = nil
	msg := m.handleTerminalKeyResponse(context.Background(), "pane", []byte("x"), sessiond.TerminalKeyResponse{
		Handled:   true,
		Toast:     "ok",
		ToastKind: sessiond.ToastSuccess,
	})
	if _, ok := msg.(SuccessMsg); !ok {
		t.Fatalf("expected success msg for handled response")
	}
}
