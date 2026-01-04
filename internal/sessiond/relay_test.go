package sessiond

import (
	"context"
	"sync"
	"testing"
	"time"

	uv "github.com/charmbracelet/ultraviolet"

	"github.com/regenrek/peakypanes/internal/native"
	"github.com/regenrek/peakypanes/internal/terminal"
)

type fakeRelayManager struct {
	rawCh      chan native.OutputChunk
	sent       map[string][][]byte
	sendErr    map[string]error
	sentCh     chan string
	cancelOnce sync.Once
}

func (m *fakeRelayManager) SessionNames() []string { return nil }
func (m *fakeRelayManager) Snapshot(context.Context, int) []native.SessionSnapshot {
	return nil
}
func (m *fakeRelayManager) Version() uint64 { return 0 }
func (m *fakeRelayManager) StartSession(context.Context, native.SessionSpec) (*native.Session, error) {
	return nil, nil
}
func (m *fakeRelayManager) KillSession(string) error                { return nil }
func (m *fakeRelayManager) RenameSession(string, string) error      { return nil }
func (m *fakeRelayManager) RenamePane(string, string, string) error { return nil }
func (m *fakeRelayManager) SplitPane(context.Context, string, string, bool, int) (string, error) {
	return "", nil
}
func (m *fakeRelayManager) ClosePane(context.Context, string, string) error { return nil }
func (m *fakeRelayManager) SwapPanes(string, string, string) error          { return nil }
func (m *fakeRelayManager) SetPaneTool(string, string) error                { return nil }
func (m *fakeRelayManager) SendInput(_ context.Context, paneID string, input []byte) error {
	if m.sent == nil {
		m.sent = make(map[string][][]byte)
	}
	m.sent[paneID] = append(m.sent[paneID], append([]byte(nil), input...))
	if m.sentCh != nil {
		select {
		case m.sentCh <- paneID:
		default:
		}
	}
	if err, ok := m.sendErr[paneID]; ok {
		return err
	}
	return nil
}
func (m *fakeRelayManager) SendMouse(string, uv.MouseEvent, terminal.MouseRoute) error {
	return nil
}
func (m *fakeRelayManager) Window(string) paneWindow                          { return nil }
func (m *fakeRelayManager) PaneTags(string) ([]string, error)                 { return nil, nil }
func (m *fakeRelayManager) AddPaneTags(string, []string) ([]string, error)    { return nil, nil }
func (m *fakeRelayManager) RemovePaneTags(string, []string) ([]string, error) { return nil, nil }
func (m *fakeRelayManager) OutputSnapshot(string, int) ([]native.OutputLine, error) {
	return nil, nil
}
func (m *fakeRelayManager) OutputLinesSince(string, uint64) ([]native.OutputLine, uint64, bool, error) {
	return nil, 0, false, nil
}
func (m *fakeRelayManager) WaitForOutput(context.Context, string) bool { return false }
func (m *fakeRelayManager) SubscribeRawOutput(string, int) (<-chan native.OutputChunk, func(), error) {
	if m.rawCh == nil {
		m.rawCh = make(chan native.OutputChunk, 4)
	}
	cancel := func() {
		m.cancelOnce.Do(func() {
			close(m.rawCh)
		})
	}
	return m.rawCh, cancel, nil
}
func (m *fakeRelayManager) PaneScrollbackSnapshot(string, int) (string, bool, error) {
	return "", false, nil
}
func (m *fakeRelayManager) SignalPane(string, string) error { return nil }
func (m *fakeRelayManager) Events() <-chan native.PaneEvent { return nil }
func (m *fakeRelayManager) Close()                          {}

func TestRelayManagerCreateListStop(t *testing.T) {
	mgr := &fakeRelayManager{rawCh: make(chan native.OutputChunk, 4), sentCh: make(chan string, 1)}
	relays := newRelayManager()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	info, err := relays.create(ctx, mgr, RelayConfig{FromPaneID: "p1", ToPaneIDs: []string{"p2"}, Mode: RelayModeRaw})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	list := relays.list()
	if len(list) != 1 {
		t.Fatalf("expected 1 relay, got %d", len(list))
	}
	if list[0].ID != info.ID {
		t.Fatalf("unexpected relay id")
	}
	mgr.rawCh <- native.OutputChunk{Data: []byte("hi"), TS: time.Now()}
	select {
	case <-mgr.sentCh:
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("timed out waiting for relay send")
	}
	if !relays.stop(info.ID) {
		t.Fatalf("expected stop true")
	}
	if len(relays.list()) != 0 {
		t.Fatalf("expected no relays after stop")
	}
	if len(mgr.sent["p2"]) == 0 {
		t.Fatalf("expected relay to send payload")
	}
}

func TestRelaySendToTargetsFiltersClosed(t *testing.T) {
	mgr := &fakeRelayManager{sendErr: map[string]error{"p1": terminal.ErrPaneClosed}}
	r := &relay{cfg: RelayConfig{ToPaneIDs: []string{"p1", "p2"}}}
	if err := r.sendToTargets(context.Background(), mgr, []byte("hi")); err != nil {
		t.Fatalf("sendToTargets: %v", err)
	}
	if len(r.cfg.ToPaneIDs) != 1 || r.cfg.ToPaneIDs[0] != "p2" {
		t.Fatalf("expected p2 remaining, got %+v", r.cfg.ToPaneIDs)
	}
}

func TestRelaySendToTargetsAllClosed(t *testing.T) {
	mgr := &fakeRelayManager{sendErr: map[string]error{"p1": terminal.ErrPaneClosed}}
	r := &relay{cfg: RelayConfig{ToPaneIDs: []string{"p1"}}}
	if err := r.sendToTargets(context.Background(), mgr, []byte("hi")); err == nil {
		t.Fatalf("expected error when all targets closed")
	}
}
