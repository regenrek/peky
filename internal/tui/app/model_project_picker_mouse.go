package app

import (
	"github.com/charmbracelet/lipgloss"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/regenrek/peakypanes/internal/tui/picker"
	"github.com/regenrek/peakypanes/internal/tui/theme"
)

func (m *Model) updateProjectPickerMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	if m == nil {
		return nil, nil
	}

	if msg.Action != tea.MouseActionPress {
		return m, nil
	}

	switch msg.Button {
	case tea.MouseButtonWheelUp:
		m.projectPicker.CursorUp()
		return m, nil
	case tea.MouseButtonWheelDown:
		m.projectPicker.CursorDown()
		return m, nil
	case tea.MouseButtonLeft:
		changed := m.selectProjectPickerIndexAt(msg.X, msg.Y)
		if !changed {
			return m, nil
		}
		return m, nil
	default:
		return m, nil
	}
}

func (m *Model) selectProjectPickerIndexAt(x, y int) bool {
	if m == nil {
		return false
	}

	// Project picker renders as appStyle.Render(list.View()) with the same padding.
	frameW, frameH := theme.App.GetFrameSize()
	if frameW < 0 || frameH < 0 {
		return false
	}
	padLeft := theme.App.GetPaddingLeft()
	padTop := theme.App.GetPaddingTop()

	listW := m.projectPicker.Width()
	listH := m.projectPicker.Height()
	if listW <= 0 || listH <= 0 {
		return false
	}

	listX0 := padLeft
	listY0 := padTop
	listX1 := listX0 + listW
	listY1 := listY0 + listH

	// Guard against stale sizing: if list is larger than the app viewport,
	// clamp hit-testing to the app frame.
	if m.width > 0 {
		listX1 = minInt(listX1, m.width-frameW+padLeft)
	}
	if m.height > 0 {
		listY1 = minInt(listY1, m.height-frameH+padTop)
	}

	if x < listX0 || x >= listX1 || y < listY0 || y >= listY1 {
		return false
	}

	titleHeight := 0
	if m.projectPicker.ShowTitle() || (m.projectPicker.ShowFilter() && m.projectPicker.FilteringEnabled()) {
		titleHeight = sectionHeight(1, m.projectPicker.Styles.TitleBar)
	}
	statusHeight := 0
	if m.projectPicker.ShowStatusBar() {
		statusHeight = sectionHeight(1, m.projectPicker.Styles.StatusBar)
	}

	itemsY0 := listY0 + titleHeight + statusHeight
	if y < itemsY0 {
		return false
	}

	itemHeight, rowHeight := picker.ProjectPickerRowMetrics()
	if itemHeight < 1 {
		itemHeight = 1
	}
	row := (y - itemsY0) / rowHeight
	lineInRow := (y - itemsY0) % rowHeight
	if lineInRow >= itemHeight {
		return false
	}

	items := m.projectPicker.VisibleItems()
	itemsOnPage := m.projectPicker.Paginator.ItemsOnPage(len(items))
	if row < 0 || row >= itemsOnPage {
		return false
	}

	index := m.projectPicker.Paginator.Page*m.projectPicker.Paginator.PerPage + row
	if index < 0 || index >= len(items) {
		return false
	}

	before := m.projectPicker.Index()
	m.projectPicker.Select(index)
	return m.projectPicker.Index() != before
}

func sectionHeight(base int, style lipgloss.Style) int {
	if base < 0 {
		base = 0
	}
	return base + style.GetPaddingTop() + style.GetPaddingBottom()
}
