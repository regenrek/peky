package app

import "github.com/regenrek/peakypanes/internal/tui/mouse"

func (m *Model) dashboardBodyRect() (mouse.Rect, bool) {
	if m.width == 0 || m.height == 0 {
		return mouse.Rect{}, false
	}
	h, v := appStyle.GetFrameSize()
	contentWidth := m.width - h
	contentHeight := m.height - v
	if contentWidth <= 10 || contentHeight <= 6 {
		return mouse.Rect{}, false
	}

	headerHeight := 1
	footerHeight := 1
	quickReplyHeight := 3
	headerGap := 1
	extraLines := headerHeight + headerGap + footerHeight + quickReplyHeight
	bodyHeight := contentHeight - extraLines
	if bodyHeight < 4 {
		headerGap = 0
		extraLines = headerHeight + headerGap + footerHeight + quickReplyHeight
		bodyHeight = contentHeight - extraLines
	}
	if bodyHeight <= 0 {
		return mouse.Rect{}, false
	}

	padTop, _, _, padLeft := appStyle.GetPadding()
	bodyX := padLeft
	bodyY := padTop + headerHeight + headerGap
	return mouse.Rect{X: bodyX, Y: bodyY, W: contentWidth, H: bodyHeight}, true
}

func (m *Model) headerRect() (mouse.Rect, bool) {
	if m.width == 0 || m.height == 0 {
		return mouse.Rect{}, false
	}
	h, v := appStyle.GetFrameSize()
	contentWidth := m.width - h
	contentHeight := m.height - v
	if contentWidth <= 10 || contentHeight <= 6 {
		return mouse.Rect{}, false
	}

	headerHeight := 1
	if headerHeight <= 0 {
		return mouse.Rect{}, false
	}

	padTop, _, _, padLeft := appStyle.GetPadding()
	return mouse.Rect{X: padLeft, Y: padTop, W: contentWidth, H: headerHeight}, true
}
