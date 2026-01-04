package app

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

const pekyDialogReservedLines = 2

func (m *Model) openPekyDialog(title, body, footer string) {
	if m == nil {
		return
	}
	m.pekyDialogTitle = strings.TrimSpace(title)
	m.pekyDialogFooter = strings.TrimSpace(footer)
	m.pekyViewport.SetContent(body)
	m.pekyViewport.GotoTop()
	m.setPekyDialogSize()
	if m.state != StatePekyDialog {
		m.pekyDialogPrevState = m.state
		m.state = StatePekyDialog
	}
}

func (m *Model) closePekyDialog() {
	if m == nil {
		return
	}
	if m.state != StatePekyDialog {
		return
	}
	previous := m.pekyDialogPrevState
	if previous == 0 {
		previous = StateDashboard
	}
	m.state = previous
}

func (m *Model) setPekyDialogSize() {
	if m == nil || m.width <= 0 || m.height <= 0 {
		return
	}
	frameW, frameH := dialogStyle.GetFrameSize()
	maxW := m.width - 8
	if maxW < 30 {
		maxW = m.width
	}
	innerW := maxW - frameW
	if innerW < 20 {
		innerW = clamp(m.width-frameW, 10, maxW)
	}
	maxH := m.height - 6
	if maxH < 10 {
		maxH = m.height
	}
	innerH := maxH - frameH
	if innerH < pekyDialogReservedLines+1 {
		innerH = pekyDialogReservedLines + 1
	}
	m.pekyViewport.Width = innerW
	m.pekyViewport.Height = innerH - pekyDialogReservedLines
}

func (m *Model) updatePekyDialog(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.closePekyDialog()
		return m, nil
	}
	var cmd tea.Cmd
	m.pekyViewport, cmd = m.pekyViewport.Update(msg)
	return m, cmd
}
