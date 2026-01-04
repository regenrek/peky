package sessiond

import (
	"context"
	"testing"
	"time"

	uv "github.com/charmbracelet/ultraviolet"

	"github.com/regenrek/peakypanes/internal/native"
	"github.com/regenrek/peakypanes/internal/terminal"
)

type scopeSendManager struct {
	snapshot []native.SessionSnapshot
	sendErr  map[string]error
	blockOn  map[string]chan struct{}
}

func (m *scopeSendManager) SessionNames() []string { return nil }
func (m *scopeSendManager) Snapshot(ctx context.Context, previewLines int) []native.SessionSnapshot {
	return m.snapshot
}
func (m *scopeSendManager) Version() uint64 { return 0 }
func (m *scopeSendManager) StartSession(context.Context, native.SessionSpec) (*native.Session, error) {
	return nil, nil
}
func (m *scopeSendManager) RestoreSession(context.Context, native.SessionRestoreSpec) (*native.Session, error) {
	return nil, nil
}
func (m *scopeSendManager) KillSession(string) error                { return nil }
func (m *scopeSendManager) RenameSession(string, string) error      { return nil }
func (m *scopeSendManager) RenamePane(string, string, string) error { return nil }
func (m *scopeSendManager) SplitPane(context.Context, string, string, bool, int) (string, error) {
	return "", nil
}
func (m *scopeSendManager) ClosePane(context.Context, string, string) error { return nil }
func (m *scopeSendManager) SwapPanes(string, string, string) error          { return nil }
func (m *scopeSendManager) SetPaneTool(string, string) error                { return nil }
func (m *scopeSendManager) SendInput(ctx context.Context, paneID string, input []byte) error {
	if ch, ok := m.blockOn[paneID]; ok {
		if ctx == nil {
			<-ch
		} else {
			select {
			case <-ch:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}
	if err, ok := m.sendErr[paneID]; ok {
		return err
	}
	return nil
}
func (m *scopeSendManager) SendMouse(string, uv.MouseEvent, terminal.MouseRoute) error { return nil }
func (m *scopeSendManager) Window(string) paneWindow                                   { return nil }
func (m *scopeSendManager) PaneTags(string) ([]string, error)                          { return nil, nil }
func (m *scopeSendManager) AddPaneTags(string, []string) ([]string, error)             { return nil, nil }
func (m *scopeSendManager) RemovePaneTags(string, []string) ([]string, error)          { return nil, nil }
func (m *scopeSendManager) OutputSnapshot(string, int) ([]native.OutputLine, error) {
	return nil, nil
}
func (m *scopeSendManager) OutputLinesSince(string, uint64) ([]native.OutputLine, uint64, bool, error) {
	return nil, 0, false, nil
}
func (m *scopeSendManager) WaitForOutput(context.Context, string) bool { return false }
func (m *scopeSendManager) SubscribeRawOutput(string, int) (<-chan native.OutputChunk, func(), error) {
	return nil, func() {}, nil
}
func (m *scopeSendManager) PaneScrollbackSnapshot(string, int) (string, bool, error) {
	return "", false, nil
}
func (m *scopeSendManager) SignalPane(string, string) error { return nil }
func (m *scopeSendManager) Events() <-chan native.PaneEvent { return nil }
func (m *scopeSendManager) Close()                          {}

func TestHandleSendInputScope(t *testing.T) {
	mgr := &scopeSendManager{
		snapshot: []native.SessionSnapshot{{
			Name:  "s1",
			Path:  "/proj",
			Panes: []native.PaneSnapshot{{ID: "p1"}, {ID: "p2"}},
		}},
		sendErr: map[string]error{"p2": errTestSend},
	}
	d := &Daemon{
		manager:    mgr,
		actionLogs: make(map[string]*actionLog),
	}
	d.focusedSession = "s1"
	payload, _ := encodePayload(SendInputRequest{Scope: "session", Input: []byte("hi"), RecordAction: true})
	data, err := d.handleSendInput(payload)
	if err != nil {
		t.Fatalf("handleSendInput: %v", err)
	}
	var resp SendInputResponse
	if err := decodePayload(data, &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(resp.Results))
	}
	if resp.Results[0].PaneID != "p1" || resp.Results[1].PaneID != "p2" {
		t.Fatalf("unexpected result order: %+v", resp.Results)
	}
	if resp.Results[0].Status != "ok" {
		t.Fatalf("expected p1 ok, got %q", resp.Results[0].Status)
	}
	if resp.Results[1].Status != "failed" {
		t.Fatalf("expected p2 failed, got %q", resp.Results[1].Status)
	}
	if len(d.paneHistory("p1", 10, time.Time{})) == 0 {
		t.Fatalf("expected history for p1")
	}
	if len(d.paneHistory("p2", 10, time.Time{})) == 0 {
		t.Fatalf("expected history for p2")
	}
}

func TestHandleSendInputScopeUnknown(t *testing.T) {
	mgr := &scopeSendManager{snapshot: []native.SessionSnapshot{{Name: "s1"}}}
	d := &Daemon{manager: mgr}
	payload, _ := encodePayload(SendInputRequest{Scope: "bogus", Input: []byte("hi")})
	if _, err := d.handleSendInput(payload); err == nil {
		t.Fatalf("expected error for unknown scope")
	}
}

func TestHandleSendInputScopeTimeout(t *testing.T) {
	origTimeout := scopeSendTimeout
	scopeSendTimeout = 20 * time.Millisecond
	t.Cleanup(func() {
		scopeSendTimeout = origTimeout
	})

	block := make(chan struct{})
	mgr := &scopeSendManager{
		snapshot: []native.SessionSnapshot{{
			Name:  "s1",
			Path:  "/proj",
			Panes: []native.PaneSnapshot{{ID: "p1"}, {ID: "p2"}},
		}},
		blockOn: map[string]chan struct{}{"p2": block},
	}
	d := &Daemon{
		manager:    mgr,
		actionLogs: make(map[string]*actionLog),
	}
	d.focusedSession = "s1"
	payload, _ := encodePayload(SendInputRequest{Scope: "session", Input: []byte("hi"), RecordAction: true})
	data, err := d.handleSendInput(payload)
	close(block)
	if err != nil {
		t.Fatalf("handleSendInput: %v", err)
	}
	var resp SendInputResponse
	if err := decodePayload(data, &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(resp.Results))
	}
	foundTimeout := false
	for _, res := range resp.Results {
		if res.PaneID == "p2" {
			if res.Status != "timeout" {
				t.Fatalf("expected timeout for p2, got %q", res.Status)
			}
			foundTimeout = true
		}
	}
	if !foundTimeout {
		t.Fatalf("expected timeout result for p2")
	}
}

var errTestSend = &sendError{msg: "send failed"}

type sendError struct{ msg string }

func (e *sendError) Error() string { return e.msg }
