package views

import (
	"strings"

	"github.com/regenrek/peakypanes/internal/tui/theme"
)

func (m Model) viewPekyDialog() string {
	title := strings.TrimSpace(m.PekyDialogTitle)
	if title == "" {
		title = "Peky"
	}
	footer := strings.TrimSpace(m.PekyDialogFooter)
	if footer == "" {
		footer = "esc close • ↑/↓ scroll"
	}
	body := m.PekyDialogViewport.View()
	content := strings.Join([]string{
		dialogTitleStyle.Render(title),
		body,
		theme.DialogNote.Render(footer),
	}, "\n")
	spec := dialogSpec{Content: content}
	if m.PekyDialogIsError && m.PekyDialogViewport.Width > 0 {
		frameW, _ := dialogStyle.GetFrameSize()
		spec.Size.Width = m.PekyDialogViewport.Width + frameW
	}
	return m.renderDialog(spec)
}
