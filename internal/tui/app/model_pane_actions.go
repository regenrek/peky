package app

import tea "github.com/charmbracelet/bubbletea"

func (m *Model) addPaneAuto() tea.Cmd {
	pane := m.selectedPane()
	vertical := false
	if pane != nil {
		vertical = autoSplitVertical(pane.Width, pane.Height)
	}
	return m.addPaneSplit(vertical)
}

func autoSplitVertical(width, height int) bool {
	if width <= 0 || height <= 0 {
		return false
	}
	return height > width
}
