package app

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/regenrek/peakypanes/internal/layout"
	"github.com/regenrek/peakypanes/internal/tui/picker"
)

// ===== Layout picker =====

func (m *Model) setupLayoutPicker() {
	m.layoutPicker = picker.NewLayoutPicker()
}

func (m *Model) openLayoutPicker() {
	project := m.selectedProject()
	session := m.selectedSession()
	if project == nil || session == nil {
		m.setToast("No project selected", toastWarning)
		return
	}
	path := session.Path
	if strings.TrimSpace(path) == "" {
		path = project.Path
	}
	if strings.TrimSpace(path) == "" {
		m.setToast("No project path configured", toastWarning)
		return
	}

	choices, err := m.loadLayoutChoices(path)
	if err != nil {
		m.setToast("Layouts: "+err.Error(), toastError)
		return
	}
	m.layoutPicker.SetItems(layoutChoicesToItems(choices))
	m.setLayoutPickerSize()
	m.setState(StateLayoutPicker)
}

func (m *Model) setLayoutPickerSize() {
	if m.width <= 0 || m.height <= 0 {
		return
	}
	hFrame, vFrame := dialogStyle.GetFrameSize()
	availableW := m.width - 6
	availableH := m.height - 4
	if availableW < 30 {
		availableW = m.width
	}
	if availableH < 10 {
		availableH = m.height
	}
	desiredW := clamp(availableW, 40, 90)
	desiredH := clamp(availableH, 12, 26)
	listW := desiredW - hFrame
	listH := desiredH - vFrame
	if listW < 20 {
		listW = clamp(m.width-hFrame, 20, m.width)
	}
	if listH < 6 {
		listH = clamp(m.height-vFrame, 6, m.height)
	}
	m.layoutPicker.SetSize(listW, listH)
}

func (m *Model) updateLayoutPicker(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.layoutPicker.FilterState() == list.Filtering {
		var cmd tea.Cmd
		m.layoutPicker, cmd = m.layoutPicker.Update(msg)
		return m, cmd
	}

	switch msg.String() {
	case "esc":
		m.setState(StateDashboard)
		return m, nil
	case "enter":
		if item, ok := m.layoutPicker.SelectedItem().(picker.LayoutChoice); ok {
			m.setState(StateDashboard)
			return m, m.startNewSessionWithLayout(item.LayoutName)
		}
		m.setState(StateDashboard)
		return m, nil
	case "q":
		m.setState(StateDashboard)
		return m, nil
	}

	var cmd tea.Cmd
	m.layoutPicker, cmd = m.layoutPicker.Update(msg)
	return m, cmd
}

func (m *Model) loadLayoutChoices(projectPath string) ([]picker.LayoutChoice, error) {
	loader, err := layout.NewLoader()
	if err != nil {
		return nil, err
	}
	loader.SetProjectDir(projectPath)
	if err := loader.LoadAll(); err != nil {
		return nil, err
	}

	choices := []picker.LayoutChoice{{
		Label:      "auto (project/default)",
		Desc:       "Use .peakypanes.yml or dev-3",
		LayoutName: "",
	}}

	layouts := loader.ListLayouts()
	seen := map[string]bool{}
	for _, info := range layouts {
		name := strings.TrimSpace(info.Name)
		if info.Source == "project" && (name == "" || name == "(project)") {
			continue
		}
		if name == "" {
			continue
		}
		if seen[name] {
			continue
		}
		seen[name] = true

		label := fmt.Sprintf("%s [%s]", name, info.Source)
		desc := strings.TrimSpace(info.Description)
		if desc == "" {
			if cfg, _, err := loader.GetLayout(name); err == nil {
				desc = layoutSummary(cfg)
			}
		}
		if desc == "" {
			desc = "layout"
		}
		choices = append(choices, picker.LayoutChoice{
			Label:      label,
			Desc:       desc,
			LayoutName: name,
		})
	}
	return choices, nil
}

func layoutChoicesToItems(choices []picker.LayoutChoice) []list.Item {
	items := make([]list.Item, len(choices))
	for i, c := range choices {
		items[i] = c
	}
	return items
}

func layoutSummary(cfg *layout.LayoutConfig) string {
	if cfg == nil {
		return ""
	}
	if strings.TrimSpace(cfg.Grid) != "" {
		if grid, err := layout.Parse(cfg.Grid); err == nil {
			return fmt.Sprintf("%d panes • %s grid", grid.Panes(), grid)
		}
		return fmt.Sprintf("grid %s", cfg.Grid)
	}
	panes := len(cfg.Panes)
	if panes == 0 {
		return ""
	}
	if panes == 1 {
		return "1 pane • split layout"
	}
	return fmt.Sprintf("%d panes • split layout", panes)
}
