package app

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/regenrek/peakypanes/internal/tui/theme"
)

// ===== Pane pickers =====

func (m *Model) openPaneSplitPicker() {
	session := m.selectedSession()
	if session == nil {
		m.setToast("No session selected", toastWarning)
		return
	}
	if session.Status == StatusStopped {
		m.setToast("Session not running", toastWarning)
		return
	}
	if m.selectedPane() == nil {
		m.setToast("No pane selected", toastWarning)
		return
	}
	m.setState(StatePaneSplitPicker)
}

func (m *Model) updatePaneSplitPicker(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q":
		m.setState(StateDashboard)
		return m, nil
	case "r":
		m.setState(StateDashboard)
		return m, m.addPaneSplit(false)
	case "d":
		m.setState(StateDashboard)
		return m, m.addPaneSplit(true)
	}
	return m, nil
}

func (m *Model) setupPaneSwapPicker() {
	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.
		Foreground(theme.TextPrimary).
		BorderLeftForeground(theme.AccentAlt)
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.
		Foreground(theme.TextSecondary).
		BorderLeftForeground(theme.AccentAlt)

	l := list.New(nil, delegate, 0, 0)
	l.Title = "🔁 Swap Pane"
	l.Styles.Title = theme.TitleAlt
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	l.SetStatusBarItemName("pane", "panes")
	m.paneSwapPicker = l
}

func (m *Model) openPaneSwapPicker() {
	session := m.selectedSession()
	if session == nil {
		m.setToast("No session selected", toastWarning)
		return
	}
	if session.Status == StatusStopped {
		m.setToast("Session not running", toastWarning)
		return
	}
	source := m.selectedPane()
	if source == nil {
		m.setToast("No pane selected", toastWarning)
		return
	}
	if len(session.Panes) < 2 {
		m.setToast("Not enough panes to swap", toastInfo)
		return
	}
	var items []list.Item
	for _, pane := range session.Panes {
		if pane.Index == source.Index {
			continue
		}
		title := strings.TrimSpace(pane.Title)
		if title == "" {
			title = strings.TrimSpace(pane.Command)
		}
		if title == "" {
			title = fmt.Sprintf("pane %s", pane.Index)
		}
		label := fmt.Sprintf("pane %s — %s", pane.Index, title)
		desc := strings.TrimSpace(pane.Command)
		if desc == "" {
			desc = "swap target"
		}
		items = append(items, PaneSwapChoice{
			Label:     label,
			Desc:      desc,
			PaneIndex: pane.Index,
		})
	}
	if len(items) == 0 {
		m.setToast("No pane selected", toastWarning)
		return
	}
	m.swapSourceSession = session.Name
	m.swapSourcePane = source.Index
	m.swapSourcePaneID = source.ID
	m.paneSwapPicker.SetItems(items)
	m.setPaneSwapPickerSize()
	m.setState(StatePaneSwapPicker)
}

func (m *Model) setPaneSwapPickerSize() {
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
	desiredW := clamp(availableW, 46, 100)
	desiredH := clamp(availableH, 12, 24)
	listW := desiredW - hFrame
	listH := desiredH - vFrame
	if listW < 20 {
		listW = clamp(m.width-hFrame, 20, m.width)
	}
	if listH < 6 {
		listH = clamp(m.height-vFrame, 6, m.height)
	}
	m.paneSwapPicker.SetSize(listW, listH)
}

func (m *Model) updatePaneSwapPicker(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.paneSwapPicker.FilterState() == list.Filtering {
		var cmd tea.Cmd
		m.paneSwapPicker, cmd = m.paneSwapPicker.Update(msg)
		return m, cmd
	}

	switch msg.String() {
	case "esc", "q":
		m.setState(StateDashboard)
		return m, nil
	case "enter":
		if item, ok := m.paneSwapPicker.SelectedItem().(PaneSwapChoice); ok {
			m.setState(StateDashboard)
			return m, m.swapPaneWith(item)
		}
		m.setState(StateDashboard)
		return m, nil
	}

	var cmd tea.Cmd
	m.paneSwapPicker, cmd = m.paneSwapPicker.Update(msg)
	return m, cmd
}
