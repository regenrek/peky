package sessiond

import (
	"context"
	"testing"

	uv "github.com/charmbracelet/ultraviolet"

	"github.com/regenrek/peakypanes/internal/layout"
	"github.com/regenrek/peakypanes/internal/native"
	"github.com/regenrek/peakypanes/internal/terminal"
)

type stubManager struct {
	snapshot  []native.OutputLine
	since     []native.OutputLine
	nextSeq   uint64
	truncated bool
	wait      bool
}

var _ sessionManager = (*stubManager)(nil)

func (s *stubManager) SessionNames() []string                                 { return nil }
func (s *stubManager) Snapshot(context.Context, int) []native.SessionSnapshot { return nil }
func (s *stubManager) Version() uint64                                        { return 0 }
func (s *stubManager) StartSession(context.Context, native.SessionSpec) (*native.Session, error) {
	return nil, nil
}
func (s *stubManager) KillSession(string) error                { return nil }
func (s *stubManager) RenameSession(string, string) error      { return nil }
func (s *stubManager) RenamePane(string, string, string) error { return nil }
func (s *stubManager) SplitPane(context.Context, string, string, bool, int) (string, error) {
	return "", nil
}
func (s *stubManager) ClosePane(context.Context, string, string) error { return nil }
func (s *stubManager) SwapPanes(string, string, string) error          { return nil }
func (s *stubManager) ResizePaneEdge(string, string, layout.ResizeEdge, int, bool, layout.SnapState) (layout.ApplyResult, error) {
	return layout.ApplyResult{}, nil
}
func (s *stubManager) ResetPaneSizes(string, string) (layout.ApplyResult, error) {
	return layout.ApplyResult{}, nil
}
func (s *stubManager) ZoomPane(string, string, bool) (layout.ApplyResult, error) {
	return layout.ApplyResult{}, nil
}
func (s *stubManager) SetPaneTool(string, string) error                { return nil }
func (s *stubManager) SendInput(context.Context, string, []byte) error { return nil }
func (s *stubManager) SendMouse(string, uv.MouseEvent, terminal.MouseRoute) error {
	return nil
}
func (s *stubManager) Window(string) paneWindow                       { return nil }
func (s *stubManager) PaneTags(string) ([]string, error)              { return nil, nil }
func (s *stubManager) AddPaneTags(string, []string) ([]string, error) { return nil, nil }
func (s *stubManager) RemovePaneTags(string, []string) ([]string, error) {
	return nil, nil
}
func (s *stubManager) OutputSnapshot(string, int) ([]native.OutputLine, error) {
	return append([]native.OutputLine(nil), s.snapshot...), nil
}
func (s *stubManager) OutputLinesSince(string, uint64) ([]native.OutputLine, uint64, bool, error) {
	return append([]native.OutputLine(nil), s.since...), s.nextSeq, s.truncated, nil
}
func (s *stubManager) WaitForOutput(context.Context, string) bool { return s.wait }
func (s *stubManager) SubscribeRawOutput(string, int) (<-chan native.OutputChunk, func(), error) {
	return nil, func() {}, nil
}
func (s *stubManager) PaneScrollbackSnapshot(string, int) (string, bool, error) {
	return "", false, nil
}
func (s *stubManager) SignalPane(string, string) error { return nil }
func (s *stubManager) Events() <-chan native.PaneEvent { return nil }
func (s *stubManager) Close()                          {}

func TestNormalizePaneOutputRequest(t *testing.T) {
	req := normalizePaneOutputRequest(PaneOutputRequest{Limit: -1})
	if req.Limit != 0 {
		t.Fatalf("expected limit=0, got %d", req.Limit)
	}
}

func TestFetchPaneOutputLinesSnapshot(t *testing.T) {
	manager := &stubManager{
		snapshot: []native.OutputLine{{Seq: 1, Text: "a"}, {Seq: 2, Text: "b"}, {Seq: 3, Text: "c"}},
	}
	lines, next, truncated, err := fetchPaneOutputLines(manager, "p1", PaneOutputRequest{SinceSeq: 0, Limit: 2, Wait: false})
	if err != nil {
		t.Fatalf("fetchPaneOutputLines error: %v", err)
	}
	if len(lines) != 2 || lines[0].Text != "b" || next != 3 || !truncated {
		t.Fatalf("lines=%v next=%d truncated=%v", lines, next, truncated)
	}
}

func TestFetchPaneOutputLinesSince(t *testing.T) {
	manager := &stubManager{
		since:   []native.OutputLine{{Seq: 4, Text: "d"}},
		nextSeq: 5,
	}
	lines, next, truncated, err := fetchPaneOutputLines(manager, "p1", PaneOutputRequest{SinceSeq: 3, Limit: 0, Wait: true})
	if err != nil {
		t.Fatalf("fetchPaneOutputLines error: %v", err)
	}
	if len(lines) != 1 || lines[0].Text != "d" || next != 5 || truncated {
		t.Fatalf("lines=%v next=%d truncated=%v", lines, next, truncated)
	}
}

func TestWaitForPaneOutput(t *testing.T) {
	manager := &stubManager{wait: true}
	if !waitForPaneOutput(manager, "p1") {
		t.Fatalf("expected wait true")
	}
	manager.wait = false
	if waitForPaneOutput(manager, "p1") {
		t.Fatalf("expected wait false")
	}
}
