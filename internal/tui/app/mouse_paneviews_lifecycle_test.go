package app

import (
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/regenrek/peakypanes/internal/sessiond"
	"github.com/regenrek/peakypanes/internal/tui/mouse"
)

func TestMouseHelpersAndForwarding(t *testing.T) {
	m := newTestModelLite()
	hit := mouse.PaneHit{
		PaneID: "p1",
		Selection: mouse.Selection{
			Project: "Alpha",
			Session: "alpha-1",
			Pane:    "1",
		},
		Content: mouse.Rect{X: 0, Y: 0, W: 10, H: 5},
	}
	if !m.hitIsSelected(hit) {
		t.Fatalf("expected hit to be selected")
	}

	changed := m.applySelectionFromHit(mouse.Selection{Project: "Beta", Session: "beta-1", Pane: "1"})
	if !changed || m.selection.Project != "Beta" {
		t.Fatalf("expected selection change")
	}

	m.client = &sessiond.Client{}
	m.terminalFocus = true
	m.paneMouseMotion["p4"] = true
	if !m.allowMouseMotion() {
		t.Fatalf("expected mouse motion allowed")
	}

	cmd := m.forwardMouseEvent(hit, tea.MouseMsg{X: 1, Y: 1, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft})
	if cmd == nil {
		t.Fatalf("expected forward mouse cmd")
	}

	if _, ok := mousePayloadFromTea(tea.MouseMsg{Action: tea.MouseAction(99)}, 1, 1); ok {
		t.Fatalf("expected invalid mouse action")
	}
	if !isWheelButton(tea.MouseButtonWheelUp) {
		t.Fatalf("expected wheel button")
	}
}

func TestPaneViewHelpers(t *testing.T) {
	m := newTestModelLite()
	hit := mouse.PaneHit{PaneID: "p1", Content: mouse.Rect{X: 0, Y: 0, W: 12, H: 6}}

	req := m.paneViewRequestForHit(hit)
	if req == nil || req.Mode != sessiond.PaneViewANSI || req.ShowCursor {
		t.Fatalf("expected ANSI pane view request")
	}

	m.client = &sessiond.Client{}
	m.terminalFocus = true
	req = m.paneViewRequestForHit(hit)
	if req == nil || req.Mode != sessiond.PaneViewLipgloss || !req.ShowCursor {
		t.Fatalf("expected lipgloss pane view request")
	}

	key := paneViewKeyFrom(sessiond.PaneViewResponse{PaneID: "p1", Cols: 12, Rows: 6, Mode: sessiond.PaneViewANSI})
	m.paneViews[key] = "view"
	if got := m.paneView("p1", 12, 6, sessiond.PaneViewANSI, false); got != "view" {
		t.Fatalf("expected pane view lookup, got %q", got)
	}

	if cmd := m.refreshPaneViewFor(""); cmd != nil {
		t.Fatalf("expected nil cmd for empty pane id")
	}
}

func TestRefreshPaneViewForHit(t *testing.T) {
	m := newTestModelLite()
	m.client = &sessiond.Client{}

	hits := m.paneHits()
	if len(hits) == 0 {
		t.Fatalf("expected pane hits")
	}
	var paneID string
	for _, hit := range hits {
		if !hit.Content.Empty() && hit.PaneID != "" {
			paneID = hit.PaneID
			break
		}
	}
	if paneID == "" {
		t.Fatalf("expected pane hit with content")
	}
	if cmd := m.refreshPaneViewFor(paneID); cmd == nil {
		t.Fatalf("expected pane view cmd")
	}
}

func TestRefreshCmdAndDaemonEvent(t *testing.T) {
	m := newTestModelLite()
	cfgPath := filepath.Join(t.TempDir(), "config.yml")
	if err := os.WriteFile(cfgPath, []byte("dashboard:\n  preview_mode: grid\n"), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	m.configPath = cfgPath
	m.client = nil

	cmd := m.refreshCmd()
	msg := cmd()
	if _, ok := msg.(dashboardSnapshotMsg); !ok {
		t.Fatalf("expected dashboardSnapshotMsg")
	}

	m.refreshInFlight = 0
	_ = m.handleDaemonEvent(daemonEventMsg{Event: sessiond.Event{Type: sessiond.EventSessionChanged}})
	if m.refreshInFlight == 0 {
		t.Fatalf("expected refresh started")
	}
}
