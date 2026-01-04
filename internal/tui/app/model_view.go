package app

import "github.com/regenrek/peakypanes/internal/tui/views"

func (m *Model) View() string {
	return views.Render(m.viewModel())
}
