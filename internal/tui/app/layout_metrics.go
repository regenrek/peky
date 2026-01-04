package app

import "github.com/regenrek/peakypanes/internal/tui/mouse"

type dashboardLayout struct {
	contentWidth  int
	contentHeight int
	padTop        int
	padLeft       int

	headerHeight     int
	headerGap        int
	bodyHeight       int
	quickReplyHeight int
	footerHeight     int
}

func (m *Model) dashboardLayout() (dashboardLayout, bool) {
	if m.width == 0 || m.height == 0 {
		m.logPaneViewSkipGlobal("window_zero", "dashboardLayout")
		return dashboardLayout{}, false
	}
	h, v := appStyle.GetFrameSize()
	contentWidth := m.width - h
	contentHeight := m.height - v
	if contentWidth <= 10 || contentHeight <= 6 {
		m.logPaneViewSkipGlobal("content_too_small", "dashboardLayout")
		return dashboardLayout{}, false
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
		m.logPaneViewSkipGlobal("body_height_invalid", "dashboardLayout")
		return dashboardLayout{}, false
	}

	padTop, _, _, padLeft := appStyle.GetPadding()
	return dashboardLayout{
		contentWidth:     contentWidth,
		contentHeight:    contentHeight,
		padTop:           padTop,
		padLeft:          padLeft,
		headerHeight:     headerHeight,
		headerGap:        headerGap,
		bodyHeight:       bodyHeight,
		quickReplyHeight: quickReplyHeight,
		footerHeight:     footerHeight,
	}, true
}

func (m *Model) dashboardBodyRect() (mouse.Rect, bool) {
	layout, ok := m.dashboardLayout()
	if !ok {
		return mouse.Rect{}, false
	}

	bodyX := layout.padLeft
	bodyY := layout.padTop + layout.headerHeight + layout.headerGap
	return mouse.Rect{X: bodyX, Y: bodyY, W: layout.contentWidth, H: layout.bodyHeight}, true
}

func (m *Model) headerRect() (mouse.Rect, bool) {
	if m.width == 0 || m.height == 0 {
		m.logPaneViewSkipGlobal("window_zero", "headerRect")
		return mouse.Rect{}, false
	}
	h, v := appStyle.GetFrameSize()
	contentWidth := m.width - h
	contentHeight := m.height - v
	if contentWidth <= 10 || contentHeight <= 6 {
		m.logPaneViewSkipGlobal("content_too_small", "headerRect")
		return mouse.Rect{}, false
	}

	headerHeight := 1
	if headerHeight <= 0 {
		m.logPaneViewSkipGlobal("header_height_invalid", "headerRect")
		return mouse.Rect{}, false
	}

	padTop, _, _, padLeft := appStyle.GetPadding()
	return mouse.Rect{X: padLeft, Y: padTop, W: contentWidth, H: headerHeight}, true
}

func (m *Model) quickReplyRect() (mouse.Rect, bool) {
	layout, ok := m.dashboardLayout()
	if !ok {
		return mouse.Rect{}, false
	}

	bodyY := layout.padTop + layout.headerHeight + layout.headerGap
	quickReplyY := bodyY + layout.bodyHeight
	return mouse.Rect{
		X: layout.padLeft,
		Y: quickReplyY,
		W: layout.contentWidth,
		H: layout.quickReplyHeight,
	}, true
}
