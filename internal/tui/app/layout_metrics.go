package app

import (
	"strings"

	"github.com/regenrek/peakypanes/internal/tui/mouse"
	"github.com/regenrek/peakypanes/internal/tui/viewlayout"
)

type dashboardLayout struct {
	contentWidth  int
	contentHeight int
	padTop        int
	padLeft       int

	headerHeight     int
	headerGap        int
	bodyHeight       int
	quickReplyHeight int
	pekyPromptHeight int
	footerHeight     int
}

func (m *Model) dashboardLayoutInternal(logCtx string) (dashboardLayout, bool) {
	log := logCtx != ""
	if m.width == 0 || m.height == 0 {
		if log {
			m.logPaneViewSkipGlobal("window_zero", logCtx)
		}
		return dashboardLayout{}, false
	}
	h, v := appStyle.GetFrameSize()
	contentWidth := m.width - h
	contentHeight := m.height - v
	hasPekyPrompt := strings.TrimSpace(m.pekyPromptLine) != ""
	hasUpdateBanner := false
	if _, _, ok := m.updateBannerInfo(); ok {
		hasUpdateBanner = true
	}
	layout, ok := viewlayout.Dashboard(contentWidth, contentHeight, hasPekyPrompt, m.quickReplyEnabled(), hasUpdateBanner)
	if !ok {
		if log {
			m.logPaneViewSkipGlobal("content_too_small", logCtx)
		}
		return dashboardLayout{}, false
	}

	padTop, _, _, padLeft := appStyle.GetPadding()
	return dashboardLayout{
		contentWidth:     contentWidth,
		contentHeight:    contentHeight,
		padTop:           padTop,
		padLeft:          padLeft,
		headerHeight:     layout.HeaderHeight,
		headerGap:        layout.HeaderGap,
		bodyHeight:       layout.BodyHeight,
		quickReplyHeight: layout.QuickReplyHeight,
		pekyPromptHeight: layout.PekyPromptHeight,
		footerHeight:     layout.FooterHeight,
	}, true
}

func (m *Model) dashboardBodyRect() (mouse.Rect, bool) {
	layout, ok := m.dashboardLayoutInternal("dashboardBodyRect")
	if !ok {
		return mouse.Rect{}, false
	}

	bodyX := layout.padLeft
	bodyY := layout.padTop + layout.headerHeight + layout.headerGap
	return mouse.Rect{X: bodyX, Y: bodyY, W: layout.contentWidth, H: layout.bodyHeight}, true
}

func (m *Model) headerRect() (mouse.Rect, bool) {
	layout, ok := m.dashboardLayoutInternal("headerRect")
	if !ok {
		return mouse.Rect{}, false
	}
	return mouse.Rect{X: layout.padLeft, Y: layout.padTop, W: layout.contentWidth, H: layout.headerHeight}, true
}

func (m *Model) quickReplyRect() (mouse.Rect, bool) {
	if !m.quickReplyEnabled() {
		return mouse.Rect{}, false
	}
	layout, ok := m.dashboardLayoutInternal("quickReplyRect")
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

func (m *Model) footerRect() (mouse.Rect, bool) {
	layout, ok := m.dashboardLayoutInternal("footerRect")
	if !ok {
		return mouse.Rect{}, false
	}
	bodyY := layout.padTop + layout.headerHeight + layout.headerGap
	quickReplyY := bodyY + layout.bodyHeight
	footerY := quickReplyY + layout.quickReplyHeight + layout.pekyPromptHeight
	return mouse.Rect{
		X: layout.padLeft,
		Y: footerY,
		W: layout.contentWidth,
		H: layout.footerHeight,
	}, true
}

func (m *Model) serverStatusRect() (mouse.Rect, bool) {
	footer, ok := m.footerRect()
	if !ok || footer.W <= 0 || footer.H <= 0 {
		return mouse.Rect{}, false
	}
	const slot = 8
	if footer.W < slot {
		return mouse.Rect{X: footer.X, Y: footer.Y, W: footer.W, H: 1}, true
	}
	return mouse.Rect{
		X: footer.X + footer.W - slot,
		Y: footer.Y,
		W: slot,
		H: 1,
	}, true
}
