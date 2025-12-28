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
	m.config = &layout.Config{
		Dashboard: layout.DashboardConfig{
			HiddenProjects: []layout.HiddenProjectConfig{{Name: "Hidden", Path: "/hidden"}},
		},
	}

	items := m.commandPaletteItems()
	if len(items) == 0 {
		t.Fatalf("expected command palette items")
	}
	foundHidden := false
	for _, item := range items {
		cmdItem, ok := item.(picker.CommandItem)
		if !ok {
			continue
		}
		if strings.Contains(cmdItem.Label, "Reopen") {
			foundHidden = true
		}
		if cmdItem.Run != nil {
			_ = cmdItem.Run()
		}
	}
	if !foundHidden {
		t.Fatalf("expected reopen hidden project entry")
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
