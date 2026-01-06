package viewlayout

const (
	dashboardHeaderHeight     = 1
	dashboardFooterHeight     = 1
	dashboardQuickReplyHeight = 3
	dashboardMinBodyHeight    = 4
	dashboardMinContentWidth  = 10
	dashboardMinContentHeight = 6
)

type DashboardLayout struct {
	ContentWidth     int
	ContentHeight    int
	HeaderHeight     int
	HeaderGap        int
	BodyHeight       int
	QuickReplyHeight int
	PekyPromptHeight int
	FooterHeight     int
}

func DashboardOk(contentWidth, contentHeight int) bool {
	return contentWidth > dashboardMinContentWidth && contentHeight > dashboardMinContentHeight
}

func Dashboard(contentWidth, contentHeight int, hasPekyPrompt bool) (DashboardLayout, bool) {
	if !DashboardOk(contentWidth, contentHeight) {
		return DashboardLayout{}, false
	}
	headerHeight := dashboardHeaderHeight
	footerHeight := dashboardFooterHeight
	quickReplyHeight := dashboardQuickReplyHeight
	pekyPromptHeight := 0
	if hasPekyPrompt {
		pekyPromptHeight = 1
	}
	headerGap := 1
	extraLines := headerHeight + headerGap + footerHeight + quickReplyHeight + pekyPromptHeight
	bodyHeight := contentHeight - extraLines
	if bodyHeight < dashboardMinBodyHeight {
		headerGap = 0
		extraLines = headerHeight + headerGap + footerHeight + quickReplyHeight + pekyPromptHeight
		bodyHeight = contentHeight - extraLines
	}
	if bodyHeight <= 0 {
		return DashboardLayout{}, false
	}
	return DashboardLayout{
		ContentWidth:     contentWidth,
		ContentHeight:    contentHeight,
		HeaderHeight:     headerHeight,
		HeaderGap:        headerGap,
		BodyHeight:       bodyHeight,
		QuickReplyHeight: quickReplyHeight,
		PekyPromptHeight: pekyPromptHeight,
		FooterHeight:     footerHeight,
	}, true
}
