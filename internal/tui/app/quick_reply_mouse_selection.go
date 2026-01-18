package app

import (
	"strings"

	"github.com/mattn/go-runewidth"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/regenrek/peakypanes/internal/tui/mouse"
)

type quickReplyMouseSelection struct {
	active bool
	start  int // rune index (inclusive)
	end    int // rune index (exclusive-ish; can equal start)
}

func (s *quickReplyMouseSelection) clear() {
	if s == nil {
		return
	}
	s.active = false
	s.start = 0
	s.end = 0
}

func (m *Model) quickReplyInputBounds() (mouse.Rect, bool) {
	rect, ok := m.quickReplyRect()
	if !ok || rect.Empty() {
		return mouse.Rect{}, false
	}
	if rect.H < 3 {
		return mouse.Rect{}, false
	}

	contentWidth := rect.W - 4 // matches views.Model.viewQuickReply
	if contentWidth < 10 {
		contentWidth = 10
	}

	const leftPad = 2
	const accentWidth = 2 // "▌ "
	labelWidth := 0
	if mode := strings.TrimSpace(m.quickReplyModeLabel()); mode != "" {
		labelWidth = runewidth.StringWidth(mode) + 1 // trailing space
	}
	inputWidth := contentWidth - accentWidth - labelWidth
	if inputWidth < 10 {
		inputWidth = 10
	}
	if inputWidth > contentWidth {
		inputWidth = contentWidth
	}

	return mouse.Rect{
		X: rect.X + leftPad + accentWidth + labelWidth,
		Y: rect.Y + 1,
		W: inputWidth,
		H: 1,
	}, true
}

func (m *Model) runeIndexAtQuickReplyX(x int) (int, bool) {
	if m == nil {
		return 0, false
	}
	inputRect, ok := m.quickReplyInputBounds()
	if !ok || inputRect.Empty() {
		return 0, false
	}
	col := x - inputRect.X
	if col <= 0 {
		return 0, true
	}
	value := m.quickReplyInput.Value()
	if value == "" {
		return 0, true
	}

	width := 0
	idx := 0
	for _, r := range value {
		w := runewidth.RuneWidth(r)
		if w < 0 {
			w = 0
		}
		if width+w > col {
			return idx, true
		}
		width += w
		idx++
	}
	return idx, true
}

func normalizeRuneRange(start, end, max int) (int, int) {
	if start < 0 {
		start = 0
	}
	if end < 0 {
		end = 0
	}
	if start > max {
		start = max
	}
	if end > max {
		end = max
	}
	if start > end {
		start, end = end, start
	}
	return start, end
}

func (m *Model) handleQuickReplyMouse(msg tea.MouseMsg) (tea.Cmd, bool) {
	if m == nil {
		return nil, false
	}
	if !m.quickReplyEnabled() {
		return nil, false
	}

	rect, ok := m.quickReplyRect()
	if !ok || rect.Empty() {
		return nil, false
	}
	if !rect.Contains(msg.X, msg.Y) {
		return nil, false
	}

	if msg.Button != tea.MouseButtonLeft && msg.Action != tea.MouseActionMotion && msg.Action != tea.MouseActionRelease {
		return nil, false
	}

	switch msg.Action {
	case tea.MouseActionPress:
		return m.handleQuickReplyMousePress(msg)

	case tea.MouseActionMotion:
		return m.handleQuickReplyMouseMotion(msg)

	case tea.MouseActionRelease:
		return m.handleQuickReplyMouseRelease(msg)

	default:
		return nil, true
	}
}

func (m *Model) handleQuickReplyMousePress(msg tea.MouseMsg) (tea.Cmd, bool) {
	if m == nil {
		return nil, false
	}
	if msg.Button != tea.MouseButtonLeft {
		return nil, false
	}

	cmd := m.setHardRaw(false)
	if m.filterActive {
		m.filterActive = false
		m.filterInput.Blur()
	}
	m.quickReplyInput.Focus()

	idx, ok := m.runeIndexAtQuickReplyX(msg.X)
	if !ok {
		return cmd, true
	}
	m.quickReplyMouseSel.active = true
	m.quickReplyMouseSel.start = idx
	m.quickReplyMouseSel.end = idx
	return cmd, true
}

func (m *Model) handleQuickReplyMouseMotion(msg tea.MouseMsg) (tea.Cmd, bool) {
	if m == nil {
		return nil, false
	}
	if !m.quickReplyMouseSel.active {
		return nil, true
	}

	idx, ok := m.runeIndexAtQuickReplyX(msg.X)
	if !ok {
		return nil, true
	}
	m.quickReplyMouseSel.end = idx
	return nil, true
}

func (m *Model) handleQuickReplyMouseRelease(_ tea.MouseMsg) (tea.Cmd, bool) {
	if m == nil {
		return nil, false
	}
	if !m.quickReplyMouseSel.active {
		return nil, true
	}

	value := m.quickReplyInput.Value()
	runes := []rune(value)
	start, end := normalizeRuneRange(m.quickReplyMouseSel.start, m.quickReplyMouseSel.end, len(runes))
	m.quickReplyMouseSel.clear()
	if start == end || len(runes) == 0 {
		return nil, true
	}

	text := string(runes[start:end])
	if strings.TrimSpace(text) == "" {
		return nil, true
	}
	if err := writeClipboard(text); err != nil {
		m.setToast("Copy failed", toastWarning)
		return nil, true
	}
	m.setToast("Copied selection", toastSuccess)
	return nil, true
}
