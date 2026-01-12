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
		index, ok := m.projectPickerIndexAt(msg.X, msg.Y)
		if !ok {
			return m, nil
		}
		m.projectPicker.Select(index)
		return m.updateProjectPicker(tea.KeyMsg{Type: tea.KeyEnter})
	default:
		return m, nil
	}
}

func (m *Model) projectPickerIndexAt(x, y int) (int, bool) {
	if m == nil {
		return 0, false
	}
	listX0, listY0, listX1, listY1, ok := m.projectPickerHitArea()
	if !ok || x < listX0 || x >= listX1 || y < listY0 || y >= listY1 {
		return 0, false
	}
	itemsY0 := m.projectPickerItemsTopY(listY0)
	return m.projectPickerIndexAtY(y, itemsY0)
}

func (m *Model) projectPickerHitArea() (x0, y0, x1, y1 int, ok bool) {
	if m == nil {
		return 0, 0, 0, 0, false
	}
	listW := m.projectPicker.Width()
	listH := m.projectPicker.Height()
	if listW <= 0 || listH <= 0 {
		return 0, 0, 0, 0, false
	}

	// Project picker renders as appStyle.Render(list.View()) with the same padding.
	x0 = theme.App.GetPaddingLeft()
	y0 = theme.App.GetPaddingTop()
	x1 = x0 + listW
	y1 = y0 + listH

	if m.width > 0 {
		maxX1 := m.width - theme.App.GetPaddingRight()
		if maxX1 > x0 {
			x1 = minInt(x1, maxX1)
		}
	}
	if m.height > 0 {
		maxY1 := m.height - theme.App.GetPaddingBottom()
		if maxY1 > y0 {
			y1 = minInt(y1, maxY1)
		}
	}
	return x0, y0, x1, y1, true
}

func (m *Model) projectPickerItemsTopY(listY0 int) int {
	if m == nil {
		return listY0
	}
	titleHeight := 0
	if m.projectPicker.ShowTitle() || (m.projectPicker.ShowFilter() && m.projectPicker.FilteringEnabled()) {
		titleHeight = sectionHeight(1, m.projectPicker.Styles.TitleBar)
	}
	statusHeight := 0
	if m.projectPicker.ShowStatusBar() {
		statusHeight = sectionHeight(1, m.projectPicker.Styles.StatusBar)
	}
	return listY0 + titleHeight + statusHeight
}

func (m *Model) projectPickerIndexAtY(y, itemsY0 int) (int, bool) {
	if m == nil {
		return 0, false
	}
	if y < itemsY0 {
		return 0, false
	}
	itemHeight, rowHeight := picker.ProjectPickerRowMetrics()
	if itemHeight < 1 {
		itemHeight = 1
	}
	rel := y - itemsY0
	row := rel / rowHeight
	lineInRow := rel % rowHeight
	if lineInRow >= itemHeight {
		return 0, false
	}

	items := m.projectPicker.VisibleItems()
	if len(items) == 0 {
		return 0, false
	}
	itemsOnPage := m.projectPicker.Paginator.ItemsOnPage(len(items))
	if row < 0 || row >= itemsOnPage {
		return 0, false
	}

	index := m.projectPicker.Paginator.Page*m.projectPicker.Paginator.PerPage + row
	if index < 0 || index >= len(items) {
		return 0, false
	}
	return index, true
}

func sectionHeight(base int, style lipgloss.Style) int {
	if base < 0 {
		base = 0
	}
	return base + style.GetPaddingTop() + style.GetPaddingBottom()
}
