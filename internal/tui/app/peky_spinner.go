package app

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/regenrek/peakypanes/internal/tui/icons"
)

const pekySpinnerInterval = 120 * time.Millisecond

func (m *Model) pekySpinnerTickCmd() tea.Cmd {
	return tea.Tick(pekySpinnerInterval, func(time.Time) tea.Msg {
		return pekySpinnerTickMsg{}
	})
}

func (m *Model) handlePekySpinnerTick() tea.Cmd {
	if m == nil || !m.pekyBusy {
		return nil
	}
	frames := icons.Active().Spinner
	if len(frames) == 0 {
		return nil
	}
	m.pekySpinnerIndex = (m.pekySpinnerIndex + 1) % len(frames)
	return m.pekySpinnerTickCmd()
}

func (m *Model) pekySpinnerFrame() string {
	if m == nil {
		return ""
	}
	frames := icons.Active().Spinner
	if len(frames) == 0 {
		return ""
	}
	idx := m.pekySpinnerIndex % len(frames)
	return frames[idx]
}
