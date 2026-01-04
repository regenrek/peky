package native

import (
	"context"
	"testing"

	"github.com/regenrek/peakypanes/internal/terminal"
)

func TestSplitPaneWithStubWindow(t *testing.T) {
	origNewWindow := newWindow
	defer func() { newWindow = origNewWindow }()
	newWindow = func(opts terminal.Options) (*terminal.Window, error) {
		return &terminal.Window{}, nil
	}

	m := newTestManager(t)
	session := &Session{Name: "sess", Path: t.TempDir()}
	pane := &Pane{ID: "p-1", Index: "0", Width: LayoutBaseSize, Height: LayoutBaseSize}
	session.Panes = []*Pane{pane}
	m.sessions[session.Name] = session
	m.panes[pane.ID] = pane

	newIndex, err := m.SplitPane(context.Background(), session.Name, pane.Index, true, 50)
	if err != nil {
		t.Fatalf("SplitPane() error: %v", err)
	}
	if newIndex == "" || len(session.Panes) != 2 {
		t.Fatalf("expected new pane added, got index=%q panes=%d", newIndex, len(session.Panes))
	}
	if !session.Panes[1].Active {
		t.Fatalf("expected new pane active")
	}
}

func TestClosePanesNoWindow(t *testing.T) {
	m := newTestManager(t)
	panes := []*Pane{
		{window: nil},
	}
	m.closePanes(panes)
}

func TestSessionLookup(t *testing.T) {
	m := newTestManager(t)
	m.sessions["demo"] = &Session{Name: "demo"}
	if m.Session("demo") == nil {
		t.Fatalf("Session() should return stored session")
	}
	if m.Session("missing") != nil {
		t.Fatalf("Session() should return nil for missing session")
	}
}

func TestCheckContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := checkContext(ctx); err == nil {
		t.Fatalf("expected context error")
	}
}
