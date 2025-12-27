package peakypanes

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type MouseMotionFilter struct {
	throttle time.Duration
	lastAt   time.Time
	lastX    int
	lastY    int
}

func NewMouseMotionFilter() *MouseMotionFilter {
	return &MouseMotionFilter{
		throttle: 16 * time.Millisecond,
		lastX:    -1,
		lastY:    -1,
	}
}

func (f *MouseMotionFilter) Filter(model tea.Model, msg tea.Msg) tea.Msg {
	if f == nil {
		return msg
	}
	mouse, ok := msg.(tea.MouseMsg)
	if !ok {
		return msg
	}
	if mouse.Action != tea.MouseActionMotion {
		return msg
	}
	m, ok := model.(*Model)
	if !ok || !m.allowMouseMotion() {
		f.reset()
		return nil
	}
	if mouse.X == f.lastX && mouse.Y == f.lastY {
		return nil
	}
	now := time.Now()
	if !f.lastAt.IsZero() && now.Sub(f.lastAt) < f.throttle {
		return nil
	}
	f.lastAt = now
	f.lastX = mouse.X
	f.lastY = mouse.Y
	return msg
}

func (f *MouseMotionFilter) reset() {
	f.lastAt = time.Time{}
	f.lastX = -1
	f.lastY = -1
}
