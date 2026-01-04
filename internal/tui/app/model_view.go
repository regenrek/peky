package app

import "github.com/regenrek/peakypanes/internal/tui/views"

func (m *Model) View() string {
	out := views.Render(m.viewModel())
	if m != nil && m.oscPending != "" {
		return m.oscPending + out
	}
	return out
}
