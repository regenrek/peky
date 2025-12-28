package native

import (
	"context"
	"testing"

	"github.com/regenrek/peakypanes/internal/layout"
	"github.com/regenrek/peakypanes/internal/terminal"
)

func TestBuildGridPanesWithStubWindow(t *testing.T) {
	origNewWindow := newWindow
	defer func() { newWindow = origNewWindow }()

	var gotOpts []terminal.Options
	newWindow = func(opts terminal.Options) (*terminal.Window, error) {
		gotOpts = append(gotOpts, opts)
		return &terminal.Window{}, nil
	}

	m := NewManager()
	cfg := &layout.LayoutConfig{
		Grid:     "1x2",
		Commands: []string{"echo a", "echo b"},
		Titles:   []string{"one", "two"},
	}
	panes, err := m.buildGridPanes(context.Background(), "/tmp", cfg, []string{"A=B"})
	if err != nil {
		t.Fatalf("buildGridPanes() error: %v", err)
	}
	if len(panes) != 2 {
		t.Fatalf("expected 2 panes, got %d", len(panes))
	}
	if panes[0].Index != "0" || panes[1].Index != "1" {
		t.Fatalf("unexpected pane indexes: %q/%q", panes[0].Index, panes[1].Index)
	}
	if !panes[0].Active {
		t.Fatalf("first pane should be active")
	}
	if len(gotOpts) != 2 {
		t.Fatalf("expected 2 window options, got %d", len(gotOpts))
	}
	if gotOpts[0].Title != "one" || gotOpts[1].Title != "two" {
		t.Fatalf("unexpected titles: %#v", gotOpts)
	}
	if gotOpts[0].Command != "echo" || len(gotOpts[0].Args) != 1 || gotOpts[0].Args[0] != "a" {
		t.Fatalf("unexpected command for pane 0: %#v", gotOpts[0])
	}
	if gotOpts[0].Dir != "/tmp" || len(gotOpts[0].Env) != 1 || gotOpts[0].Env[0] != "A=B" {
		t.Fatalf("unexpected opts for pane 0: %#v", gotOpts[0])
	}
}

func TestBuildSplitPanesWithStubWindow(t *testing.T) {
	origNewWindow := newWindow
	defer func() { newWindow = origNewWindow }()

	newWindow = func(opts terminal.Options) (*terminal.Window, error) {
		return &terminal.Window{}, nil
	}

	m := NewManager()
	defs := []layout.PaneDef{
		{Title: "one", Cmd: "echo a"},
		{Title: "two", Cmd: "echo b", Split: "vertical", Size: "50%"},
	}
	panes, err := m.buildSplitPanes(context.Background(), "/tmp", defs, nil)
	if err != nil {
		t.Fatalf("buildSplitPanes() error: %v", err)
	}
	if len(panes) != 2 {
		t.Fatalf("expected 2 panes, got %d", len(panes))
	}
	if !panes[0].Active {
		t.Fatalf("first pane should be active")
	}
	if panes[0].Width == 0 || panes[1].Width == 0 {
		t.Fatalf("expected split geometry to be set")
	}
}

func TestCreatePaneInvalidCommand(t *testing.T) {
	origNewWindow := newWindow
	defer func() { newWindow = origNewWindow }()
	newWindow = func(opts terminal.Options) (*terminal.Window, error) {
		return &terminal.Window{}, nil
	}

	m := NewManager()
	if _, err := m.createPane(context.Background(), "/tmp", "title", "'", nil); err == nil {
		t.Fatalf("expected error for invalid command")
	}
}

func TestStartSessionWithStubWindow(t *testing.T) {
	origNewWindow := newWindow
	defer func() { newWindow = origNewWindow }()
	newWindow = func(opts terminal.Options) (*terminal.Window, error) {
		return &terminal.Window{}, nil
	}

	m := NewManager()
	cfg := &layout.LayoutConfig{
		Panes: []layout.PaneDef{
			{Title: "one", Cmd: "echo a"},
			{Title: "two", Cmd: "echo b", Split: "horizontal", Size: "50%"},
		},
	}
	session, err := m.StartSession(context.Background(), SessionSpec{
		Name:       "demo",
		Path:       t.TempDir(),
		Layout:     cfg,
		LayoutName: "custom",
	})
	if err != nil {
		t.Fatalf("StartSession() error: %v", err)
	}
	if session == nil || session.Name != "demo" || len(session.Panes) != 2 {
		t.Fatalf("unexpected session: %#v", session)
	}
	if m.Version() == 0 {
		t.Fatalf("expected version increment")
	}
	if got := drainEvents(m.Events()); got == 0 {
		t.Fatalf("expected events emitted")
	}
}
