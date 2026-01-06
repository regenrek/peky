package app

import (
	"time"

	"github.com/regenrek/peakypanes/internal/tui/theme"
)

// ===== Toast messages =====

type toastLevel int

const (
	toastInfo toastLevel = iota
	toastSuccess
	toastWarning
	toastError
)

type toastMessage struct {
	Text  string
	Level toastLevel
	Until time.Time
}

func (m *Model) setToast(text string, level toastLevel) {
	m.toast = toastMessage{Text: singleLine(text), Level: level, Until: time.Now().Add(3 * time.Second)}
}

func (m *Model) toastText() string {
	if m.toast.Text == "" || time.Now().After(m.toast.Until) {
		return ""
	}
	switch m.toast.Level {
	case toastSuccess:
		return theme.StatusMessage.Render(m.toast.Text)
	case toastWarning:
		return theme.StatusWarning.Render(m.toast.Text)
	case toastError:
		return theme.StatusError.Render(m.toast.Text)
	default:
		return theme.StatusMessage.Render(m.toast.Text)
	}
}
