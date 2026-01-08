package app

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/regenrek/peakypanes/internal/sessiond"
	"github.com/regenrek/peakypanes/internal/tui/mouse"
)

func TestAllowFallbackRequestRespectsMinInterval(t *testing.T) {
	m := newTestModelLite()
	m.settings.Performance.PaneViews.FallbackMinInterval = time.Second

	now := time.Unix(100, 0)
	if !m.allowFallbackRequest("p1", now) {
		t.Fatalf("expected first request allowed")
	}
	if m.allowFallbackRequest("p1", now.Add(500*time.Millisecond)) {
		t.Fatalf("expected throttled request denied")
	}
	if !m.allowFallbackRequest("p1", now.Add(2*time.Second)) {
		t.Fatalf("expected request allowed after interval")
	}
}

func TestNextPaneViewPumpBackoffClamps(t *testing.T) {
	perf := PaneViewPerformance{PumpBaseDelay: 10 * time.Millisecond, PumpMaxDelay: 50 * time.Millisecond}
	if got := nextPaneViewPumpBackoff(0, perf); got != 10*time.Millisecond {
		t.Fatalf("got=%s", got)
	}
	if got := nextPaneViewPumpBackoff(30*time.Millisecond, perf); got != 50*time.Millisecond {
		t.Fatalf("got=%s", got)
	}
}

func TestHandlePaneViewPumpResetsBackoffWhenNoPending(t *testing.T) {
	m := newTestModelLite()
	m.paneViewPumpBackoff = time.Second
	m.paneViewQueuedIDs = nil
	if cmd := m.handlePaneViewPump(paneViewPumpMsg{Reason: "test"}); cmd != nil {
		t.Fatalf("expected nil cmd")
	}
	if m.paneViewPumpBackoff != 0 {
		t.Fatalf("backoff=%s want=0", m.paneViewPumpBackoff)
	}
}

func TestHandlePaneViewPumpDelaysWhenInFlight(t *testing.T) {
	m := newTestModelLite()
	m.paneViewQueuedIDs = map[string]struct{}{"p1": {}}
	m.settings.Performance.PaneViews.MaxInFlightBatches = 1
	m.settings.Performance.PaneViews.MaxBatch = 10
	m.paneViewInFlight = 1

	cmd := m.handlePaneViewPump(paneViewPumpMsg{Reason: "test"})
	if cmd == nil {
		t.Fatalf("expected cmd")
	}
	if !m.paneViewPumpScheduled {
		t.Fatalf("expected pump scheduled")
	}
	_ = cmd
}

func TestPaneViewPumpMaxReqsUsesRemainingBatches(t *testing.T) {
	m := newTestModelLite()
	m.settings.Performance.PaneViews.MaxInFlightBatches = 3
	m.settings.Performance.PaneViews.MaxBatch = 4
	m.paneViewInFlight = 2
	if got := m.paneViewPumpMaxReqs(); got != 4 {
		t.Fatalf("got=%d want=4", got)
	}
	m.paneViewInFlight = 0
	if got := m.paneViewPumpMaxReqs(); got != 12 {
		t.Fatalf("got=%d want=12", got)
	}
}

func TestPaneSizeForFallbackUsesRecordedSize(t *testing.T) {
	m := newTestModelLite()
	cols, rows := m.paneSizeForFallback("p1")
	if cols <= 0 || rows <= 0 {
		t.Fatalf("expected fallback size")
	}
	m.recordPaneSize("p1", 80, 24)
	cols, rows = m.paneSizeForFallback("p1")
	if cols != 80 || rows != 24 {
		t.Fatalf("cols=%d rows=%d", cols, rows)
	}
}

func TestCombinePaneViewRefreshPrefersNonNil(t *testing.T) {
	m := newTestModelLite()
	cmdA := func() tea.Msg { return nil }
	cmdB := func() tea.Msg { return nil }
	if got := m.combinePaneViewRefresh(nil, cmdB); got == nil {
		t.Fatalf("expected cmd")
	}
	if got := m.combinePaneViewRefresh(cmdA, nil); got == nil {
		t.Fatalf("expected cmd")
	}
	cmd := m.combinePaneViewRefresh(cmdA, cmdB)
	if cmd == nil {
		t.Fatalf("expected batch cmd")
	}
}

func TestQueuePaneViewForIDUsesHitMap(t *testing.T) {
	m := newTestModelLite()
	hits := map[string]mouse.PaneHit{"p1": {PaneID: "p1"}}
	if refresh := m.queuePaneViewForID("p1", hits); refresh {
		t.Fatalf("expected no refresh when hit present")
	}
	if _, ok := m.paneViewQueuedIDs["p1"]; !ok {
		t.Fatalf("expected pane queued")
	}
	if age := m.paneViewQueuedAge("p1"); age <= 0 {
		t.Fatalf("expected queued age > 0, got %s", age)
	}
}

func TestChunkPaneViewRequestsSplits(t *testing.T) {
	reqs := []sessiond.PaneViewRequest{
		{PaneID: "p1"}, {PaneID: "p2"}, {PaneID: "p3"}, {PaneID: "p4"}, {PaneID: "p5"},
	}
	chunks := chunkPaneViewRequests(reqs, 2)
	if len(chunks) != 3 {
		t.Fatalf("chunks=%d want=3", len(chunks))
	}
	if len(chunks[0]) != 2 || len(chunks[1]) != 2 || len(chunks[2]) != 1 {
		t.Fatalf("lens=%d,%d,%d", len(chunks[0]), len(chunks[1]), len(chunks[2]))
	}
}
