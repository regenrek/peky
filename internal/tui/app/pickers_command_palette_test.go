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

func TestCommandPaletteAddAgentChildren(t *testing.T) {
	m := newTestModelLite()

	items := m.commandPaletteItems()
	found := false
	var addAgent picker.CommandItem
	for _, item := range items {
		cmdItem, ok := item.(picker.CommandItem)
		if !ok {
			continue
		}
		if cmdItem.Label == "Add Agent" {
			addAgent = cmdItem
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected add agent entry")
	}
	if len(addAgent.Children) != 4 {
		t.Fatalf("expected 4 add agent children, got %d", len(addAgent.Children))
	}
	labels := make(map[string]struct{}, len(addAgent.Children))
	for _, child := range addAgent.Children {
		labels[child.Label] = struct{}{}
	}
	for _, label := range []string{"Codex CLI", "Claude Code", "Pi", "Opencode"} {
		if _, ok := labels[label]; !ok {
			t.Fatalf("missing add agent child %q", label)
		}
	}
}

type paletteScan struct {
	foundQuickAddPane    bool
	foundQuickClosePane  bool
	foundQuickAddSession bool
	foundAddAgent        bool
	foundSkills          bool
	foundSettings        bool
	foundDebug           bool
	foundPane            bool
	foundSession         bool
	foundProject         bool
	foundExit            bool
	paneIndex            int
	sessionIndex         int
	projectIndex         int
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
		applyPaletteLabel(&scan, cmdItem.Label)
		if cmdItem.Run != nil {
			_ = cmdItem.Run()
		}
	}
	for i, item := range items {
		cmdItem, ok := item.(picker.CommandItem)
		if !ok {
			continue
		}
		applyPaletteIndex(&scan, cmdItem.Label, i)
	}
	return scan
}

func applyPaletteLabel(scan *paletteScan, label string) {
	if scan == nil {
		return
	}
	switch label {
	case "Settings":
		scan.foundSettings = true
	case "Debug":
		scan.foundDebug = true
	case "Add Pane":
		scan.foundQuickAddPane = true
	case "Close Pane":
		scan.foundQuickClosePane = true
	case "Add Session":
		scan.foundQuickAddSession = true
	case "Add Agent":
		scan.foundAddAgent = true
	case "Skills: Install peky skills":
		scan.foundSkills = true
	case "Exit":
		scan.foundExit = true
	case "Panes":
		scan.foundPane = true
	case "Sessions":
		scan.foundSession = true
	case "Project":
		scan.foundProject = true
	}
}

func applyPaletteIndex(scan *paletteScan, label string, idx int) {
	if scan == nil {
		return
	}
	switch label {
	case "Panes":
		if scan.paneIndex == -1 {
			scan.paneIndex = idx
		}
	case "Sessions":
		if scan.sessionIndex == -1 {
			scan.sessionIndex = idx
		}
	case "Project":
		if scan.projectIndex == -1 {
			scan.projectIndex = idx
		}
	}
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
	if !scan.foundQuickAddPane || !scan.foundQuickClosePane || !scan.foundQuickAddSession {
		t.Fatalf("expected quick add/close commands")
	}
	if !scan.foundAddAgent {
		t.Fatalf("expected add agent entry")
	}
	if !scan.foundSkills {
		t.Fatalf("expected skills entry")
	}
	if !scan.foundExit {
		t.Fatalf("expected exit entry")
	}
	if !scan.foundPane || !scan.foundSession || !scan.foundProject {
		t.Fatalf("expected pane, session, and project groups")
	}
}

func assertPaletteOrder(t *testing.T, scan paletteScan) {
	t.Helper()
	if scan.paneIndex == -1 || scan.sessionIndex == -1 || scan.projectIndex == -1 {
		t.Fatalf("expected pane, session, and project positions")
	}
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
	_, updateCmd := m.updateCommandPalette(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	if updateCmd != nil {
		msg := updateCmd()
		if msg != nil {
			if batch, ok := msg.(tea.BatchMsg); ok {
				for _, part := range batch {
					m.commandPalette, _ = m.commandPalette.Update(part)
				}
			} else {
				m.commandPalette, _ = m.commandPalette.Update(msg)
			}
		}
	}
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

func TestCommandPaletteFilteringShowsChildCommands(t *testing.T) {
	m := newTestModelLite()
	m.setState(StateCommandPalette)
	m.commandPalette.SetFilterState(list.Filtering)
	cmd := m.commandPalette.SetItems(m.commandPaletteItems())
	if cmd != nil {
		msg := cmd()
		m.commandPalette, _ = m.commandPalette.Update(msg)
	}

	m.updateCommandPalette(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	items := m.commandPalette.VisibleItems()
	if len(items) == 0 {
		t.Fatalf("expected filtered items")
	}
	found := false
	for _, item := range items {
		cmdItem, ok := item.(picker.CommandItem)
		if !ok {
			continue
		}
		if strings.Contains(strings.ToLower(cmdItem.Label), "add") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected add command in filtered results")
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
