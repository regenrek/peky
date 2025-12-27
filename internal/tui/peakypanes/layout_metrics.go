package peakypanes

import "github.com/charmbracelet/lipgloss"

func (m *Model) dashboardBodyRect() (rect, bool) {
	if m.width == 0 || m.height == 0 {
		return rect{}, false
	}
	h, v := appStyle.GetFrameSize()
	contentWidth := m.width - h
	contentHeight := m.height - v
	if contentWidth <= 10 || contentHeight <= 6 {
		return rect{}, false
	}

	showThumbs := m.settings.ShowThumbnails
	thumbHeight := 0
	if showThumbs {
		thumbHeight = 3
	}

	header := m.viewHeader(contentWidth)
	footer := m.viewFooter(contentWidth)
	quickReply := m.viewQuickReply(contentWidth)
	quickReplyHeight := lipgloss.Height(quickReply)

	headerHeight := lipgloss.Height(header)
	footerHeight := lipgloss.Height(footer)
	headerGap := 1
	extraLines := headerHeight + headerGap + footerHeight + quickReplyHeight
	if showThumbs {
		extraLines += thumbHeight
	}
	bodyHeight := contentHeight - extraLines
	if bodyHeight < 4 {
		showThumbs = false
		thumbHeight = 0
		headerGap = 0
		extraLines = headerHeight + headerGap + footerHeight + quickReplyHeight
		bodyHeight = contentHeight - extraLines
	}
	if bodyHeight <= 0 {
		return rect{}, false
	}

	padTop, _, _, padLeft := appStyle.GetPadding()
	bodyX := padLeft
	bodyY := padTop + headerHeight + headerGap
	return rect{X: bodyX, Y: bodyY, W: contentWidth, H: bodyHeight}, true
}
