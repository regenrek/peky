package app

import "testing"

import "github.com/regenrek/peakypanes/internal/sessiond"

func TestPaneViewProvider(t *testing.T) {
	m := newTestModelLite()
	if m.paneViewProvider() != nil {
		t.Fatalf("expected nil pane view provider without client")
	}

	m.client = &sessiond.Client{}
	provider := m.paneViewProvider()
	if provider == nil {
		t.Fatalf("expected pane view provider with client")
	}
	key := paneViewKey{PaneID: "p1", Cols: 10, Rows: 5, Mode: sessiond.PaneViewANSI, ShowCursor: false}
	m.paneViews[key] = "view"
	if got := provider("p1", 10, 5, false); got != "view" {
		t.Fatalf("expected pane view lookup, got %q", got)
	}
	if got := provider("", 10, 5, false); got != "" {
		t.Fatalf("expected empty result for missing id")
	}
}
