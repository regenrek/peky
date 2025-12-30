package app

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/regenrek/peakypanes/internal/layout"
	"github.com/regenrek/peakypanes/internal/tui/picker"
)

func TestCommandPaletteItemsAndRun(t *testing.T) {
	m := newTestModelLite()

	items := m.commandPaletteItems()
	if len(items) == 0 {
		t.Fatalf("expected command palette items")
	}
	scan := scanPaletteItems(t, items)
	assertPaletteEntries(t, scan)
	assertPaletteOrder(t, scan)
}

type paletteScan struct {
	foundSettings  bool
	foundDebug     bool
	foundPane      bool
	foundSession   bool
	foundProject   bool
	foundBroadcast bool
	paneIndex      int
	sessionIndex   int
	projectIndex   int
}

func scanPaletteItems(t *testing.T, items []list.Item) paletteScan {
	t.Helper()
	scan := paletteScan{
		paneIndex:    -1,
		sessionIndex: -1,
		projectIndex: -1,
	}
	for _, item := range items {
		cmdItem, ok := item.(picker.CommandItem)
		if !ok {
			continue
		}
		assertPaletteLabel(t, cmdItem.Label)
		if cmdItem.Label == "Settings" {
			scan.foundSettings = true
		}
		if cmdItem.Label == "Debug" {
			scan.foundDebug = true
		}
		if cmdItem.Label == "Broadcast: /all" {
			scan.foundBroadcast = true
		}
		if cmdItem.Run != nil {
			_ = cmdItem.Run()
		}
	}
	for i, item := range items {
		cmdItem, ok := item.(picker.CommandItem)
		if !ok {
			continue
		}
		if strings.HasPrefix(cmdItem.Label, "Pane:") && scan.paneIndex == -1 {
			scan.paneIndex = i
			scan.foundPane = true
		}
		if strings.HasPrefix(cmdItem.Label, "Session:") && scan.sessionIndex == -1 {
			scan.sessionIndex = i
			scan.foundSession = true
		}
		if strings.HasPrefix(cmdItem.Label, "Project:") && scan.projectIndex == -1 {
			scan.projectIndex = i
			scan.foundProject = true
		}
	}
	return scan
}

func assertPaletteLabel(t *testing.T, label string) {
	t.Helper()
	if strings.Contains(label, "Reopen") {
		t.Fatalf("unexpected reopen entry in command palette")
	}
	if strings.Contains(label, "Quick reply") {
		t.Fatalf("unexpected quick reply entry in command palette")
	}
	if strings.Contains(label, "Attach / start") {
		t.Fatalf("unexpected attach/start entry in command palette")
	}
}

func assertPaletteEntries(t *testing.T, scan paletteScan) {
	t.Helper()
	if !scan.foundSettings || !scan.foundDebug {
		t.Fatalf("expected settings and debug entries")
	}
	if !scan.foundBroadcast {
		t.Fatalf("expected broadcast entry")
	}
	if !scan.foundPane || !scan.foundSession || !scan.foundProject {
		t.Fatalf("expected pane, session, and project groups")
	}
}

func assertPaletteOrder(t *testing.T, scan paletteScan) {
	t.Helper()
	if scan.paneIndex >= scan.sessionIndex || scan.sessionIndex >= scan.projectIndex {
		t.Fatalf("expected pane items before session and project items")
	}
}

func TestUpdateCommandPaletteEnterAndEsc(t *testing.T) {
	m := newTestModelLite()
	called := false
	item := picker.CommandItem{
		Label: "Test",
		Run: func() tea.Cmd {
			called = true
			return nil
		},
	}
	m.commandPalette.SetItems([]list.Item{item})
	m.commandPalette.Select(0)
	m.setState(StateCommandPalette)

	m.updateCommandPalette(tea.KeyMsg{Type: tea.KeyEnter})
	if !called {
		t.Fatalf("expected command run")
	}
	if m.state != StateDashboard {
		t.Fatalf("expected dashboard state")
	}

	m.setState(StateCommandPalette)
	m.updateCommandPalette(tea.KeyMsg{Type: tea.KeyEsc})
	if m.state != StateDashboard {
		t.Fatalf("expected dashboard state after esc")
	}
}

func TestUpdateCommandPaletteFilteringAndQuit(t *testing.T) {
	m := newTestModelLite()
	m.setState(StateCommandPalette)
	m.commandPalette.SetFilterState(list.Filtering)
	m.updateCommandPalette(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	if m.state != StateCommandPalette {
		t.Fatalf("expected command palette to stay open while filtering")
	}

	m.commandPalette.SetFilterState(list.Unfiltered)
	m.updateCommandPalette(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if m.state != StateDashboard {
		t.Fatalf("expected dashboard state after quit")
	}

	item := picker.CommandItem{Label: "noop"}
	m.commandPalette.SetItems([]list.Item{item})
	m.commandPalette.Select(0)
	m.setState(StateCommandPalette)
	m.updateCommandPalette(tea.KeyMsg{Type: tea.KeyEnter})
	if m.state != StateDashboard {
		t.Fatalf("expected dashboard state after enter with nil run")
	}
}

func TestCommandPaletteNavigationWhileFiltering(t *testing.T) {
	m := newTestModelLite()
	m.setCommandPaletteSize()
	items := []list.Item{
		picker.CommandItem{Label: "Alpha"},
		picker.CommandItem{Label: "Beta"},
	}
	m.commandPalette.SetFilterState(list.Filtering)
	cmd := m.commandPalette.SetItems(items)
	if cmd != nil {
		msg := cmd()
		m.commandPalette, _ = m.commandPalette.Update(msg)
	}
	m.setState(StateCommandPalette)

	m.updateCommandPalette(tea.KeyMsg{Type: tea.KeyDown})
	if m.commandPalette.FilterState() != list.Filtering {
		t.Fatalf("expected filtering state to remain active")
	}
	if got := m.commandPalette.Index(); got != 1 {
		t.Fatalf("expected selection to move while filtering, got index %d", got)
	}
}

func TestCommandPaletteEnterWhileFilteringRuns(t *testing.T) {
	m := newTestModelLite()
	called := false
	item := picker.CommandItem{
		Label: "Run",
		Run: func() tea.Cmd {
			called = true
			return nil
		},
	}
	m.commandPalette.SetFilterState(list.Filtering)
	cmd := m.commandPalette.SetItems([]list.Item{item})
	if cmd != nil {
		msg := cmd()
		m.commandPalette, _ = m.commandPalette.Update(msg)
	}
	m.commandPalette.Select(0)
	m.setState(StateCommandPalette)

	m.updateCommandPalette(tea.KeyMsg{Type: tea.KeyEnter})
	if !called {
		t.Fatalf("expected enter to run selection while filtering")
	}
	if m.state != StateDashboard {
		t.Fatalf("expected dashboard state after enter while filtering")
	}
}

func TestCommandPaletteEscWhileFilteringCloses(t *testing.T) {
	m := newTestModelLite()
	m.commandPalette.SetFilterState(list.Filtering)
	m.setState(StateCommandPalette)

	m.updateCommandPalette(tea.KeyMsg{Type: tea.KeyEsc})
	if m.state != StateDashboard {
		t.Fatalf("expected dashboard state after esc while filtering")
	}
}

func TestLayoutPickerAndSummary(t *testing.T) {
	m := newTestModelLite()
	tmp := t.TempDir()
	m.data.Projects[0].Path = tmp
	m.data.Projects[0].Sessions[0].Path = tmp
	m.openLayoutPicker()
	if m.state != StateLayoutPicker {
		t.Fatalf("expected layout picker state")
	}
	m.updateLayoutPicker(tea.KeyMsg{Type: tea.KeyEsc})
	if m.state != StateDashboard {
		t.Fatalf("expected dashboard state after esc")
	}

	if got := layoutSummary(&layout.LayoutConfig{Grid: "2x1"}); !strings.Contains(got, "grid") {
		t.Fatalf("expected grid summary, got %q", got)
	}
	if got := layoutSummary(&layout.LayoutConfig{Panes: []layout.PaneDef{{}}}); !strings.Contains(got, "1 pane") {
		t.Fatalf("expected pane summary, got %q", got)
	}
}

func TestUpdateLayoutPickerEnterAndQuit(t *testing.T) {
	m := newTestModelLite()
	tmp := t.TempDir()
	m.data.Projects[0].Path = tmp
	m.data.Projects[0].Sessions[0].Path = tmp
	m.layoutPicker.SetItems([]list.Item{picker.LayoutChoice{Label: "dev", LayoutName: "dev"}})
	m.layoutPicker.Select(0)

	m.setState(StateLayoutPicker)
	m.layoutPicker.SetFilterState(list.Filtering)
	m.updateLayoutPicker(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	if m.state != StateLayoutPicker {
		t.Fatalf("expected layout picker to stay open while filtering")
	}

	m.layoutPicker.SetFilterState(list.Unfiltered)
	m.setState(StateLayoutPicker)
	m.updateLayoutPicker(tea.KeyMsg{Type: tea.KeyEnter})
	if m.state != StateDashboard {
		t.Fatalf("expected dashboard state after enter")
	}

	m.setState(StateLayoutPicker)
	m.updateLayoutPicker(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if m.state != StateDashboard {
		t.Fatalf("expected dashboard state after quit")
	}
}

func TestPanePickers(t *testing.T) {
	m := newTestModelLite()

	m.openPaneSplitPicker()
	if m.state != StatePaneSplitPicker {
		t.Fatalf("expected pane split picker state")
	}
	m.updatePaneSplitPicker(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	if m.state != StateDashboard {
		t.Fatalf("expected dashboard after split picker")
	}

	m.openPaneSwapPicker()
	if m.state != StatePaneSwapPicker {
		t.Fatalf("expected pane swap picker state")
	}
	m.updatePaneSwapPicker(tea.KeyMsg{Type: tea.KeyEnter})
	if m.state != StateDashboard {
		t.Fatalf("expected dashboard after swap picker")
	}
}
