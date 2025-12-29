package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/regenrek/peakypanes/internal/tui/icons"
	"github.com/regenrek/peakypanes/internal/tui/logo"
	"github.com/regenrek/peakypanes/internal/tui/theme"
)

func (m Model) viewDashboard() string {
	return appStyle.Render(m.viewDashboardContent())
}

func (m Model) viewDashboardContent() string {
	if m.Width == 0 || m.Height == 0 {
		return ""
	}
	h, v := appStyle.GetFrameSize()
	contentWidth := m.Width - h
	contentHeight := m.Height - v
	if contentWidth <= 10 || contentHeight <= 6 {
		return "Terminal too small"
	}

	showThumbs := m.ShowThumbnails
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
		headerGap = 0
		extraLines = headerHeight + headerGap + footerHeight + quickReplyHeight
		bodyHeight = contentHeight - extraLines
	}

	body := m.viewBody(contentWidth, bodyHeight)
	sections := []string{header}
	if headerGap > 0 {
		sections = append(sections, fitLine("", contentWidth))
	}
	sections = append(sections, body, quickReply)
	if showThumbs {
		sections = append(sections, m.viewThumbnails(contentWidth))
	}
	sections = append(sections, footer)

	return lipgloss.JoinVertical(lipgloss.Top, sections...)
}

func (m Model) viewHeader(width int) string {
	left := m.HeaderLine
	if width <= 0 {
		return left
	}
	leftWidth := lipgloss.Width(left)
	available := width - leftWidth - 1
	if available < 1 {
		return fitLine(left, width)
	}
	rightPlain := logo.SmallRender(available)
	if strings.TrimSpace(rightPlain) == "" {
		return fitLine(left, width)
	}
	right := theme.LogoStyle.Render(rightPlain)
	rightWidth := lipgloss.Width(right)
	spaces := width - leftWidth - rightWidth
	if spaces < 1 {
		return fitLine(left, width)
	}
	return left + strings.Repeat(" ", spaces) + right
}

func (m Model) viewBody(width, height int) string {
	if height <= 0 {
		return ""
	}
	if m.Tab == tabDashboard {
		return m.viewDashboardGrid(width, height)
	}
	return m.viewProjectBody(width, height)
}

func (m Model) viewProjectBody(width, height int) string {
	base := width / 3
	leftWidth := clamp(base-(width/30), 22, 36)
	if leftWidth > width-10 {
		leftWidth = width / 2
	}
	rightWidth := width - leftWidth - 1
	if rightWidth < 10 {
		leftWidth = clamp(width/2, 12, width-10)
		rightWidth = width - leftWidth - 1
	}
	left := m.viewSidebar(leftWidth, height)
	right := m.viewPreview(rightWidth, height)
	return lipgloss.JoinHorizontal(lipgloss.Top, left, right)
}

func (m Model) viewDashboardGrid(width, height int) string {
	if width <= 0 || height <= 0 {
		return ""
	}
	columns := m.DashboardColumns
	if len(columns) == 0 {
		if strings.TrimSpace(m.FilterInput.Value()) != "" {
			return padLines("No panes match the current filter.", width, height)
		}
		return m.viewSplash(width, height)
	}
	totalPanes := 0
	for _, column := range columns {
		totalPanes += len(column.Panes)
	}
	if totalPanes == 0 {
		if strings.TrimSpace(m.FilterInput.Value()) != "" {
			return padLines("No panes match the current filter.", width, height)
		}
		return m.viewSplash(width, height)
	}
	ctx := dashboardRenderContext{
		selectionSession: m.SelectionSession,
		selectionPane:    m.SelectionPane,
		previewLines:     m.DashboardPreviewLines,
		terminalFocus:    m.TerminalFocus,
		renderer:         m.renderDashboardPaneTileLive,
	}
	return renderDashboardColumnsWithRenderer(columns, width, height, m.DashboardSelectedProject, ctx)
}

func (m Model) viewSplash(width, height int) string {
	if width <= 0 || height <= 0 {
		return ""
	}
	if strings.TrimSpace(m.SplashInfo) == "" && width < 4 {
		return padLines(m.EmptyStateMessage, width, height)
	}
	logoText := logo.Render(width, false)
	if width < logo.FullWidth() || height < logo.FullHeight()+2 {
		logoText = logo.SmallRender(width)
	}
	logoLines := strings.Split(logoText, "\n")
	blockWidth := 0
	for _, line := range logoLines {
		if lineWidth := lipgloss.Width(line); lineWidth > blockWidth {
			blockWidth = lineWidth
		}
	}
	lines := make([]string, 0, len(logoLines)+2)
	for _, line := range logoLines {
		padded := padRight(line, blockWidth)
		lines = append(lines, centerLine(theme.LogoStyle.Render(padded), width))
	}
	info := strings.TrimSpace(m.SplashInfo)
	if info != "" {
		lines = append(lines, "")
		lines = append(lines, centerLine(theme.DialogNote.Render(info), width))
	}
	if len(lines) == 0 {
		return padLines(m.EmptyStateMessage, width, height)
	}
	contentHeight := len(lines)
	if contentHeight >= height {
		return padLines(strings.Join(lines, "\n"), width, height)
	}
	padTop := (height - contentHeight) / 2
	if padTop < 1 {
		padTop = 1
	}
	blank := padRight("", width)
	out := make([]string, 0, height)
	for i := 0; i < padTop; i++ {
		out = append(out, blank)
	}
	for _, line := range lines {
		out = append(out, padRight(line, width))
	}
	for len(out) < height {
		out = append(out, blank)
	}
	return strings.Join(out, "\n")
}

func (m Model) viewSidebar(width, height int) string {
	if m.SidebarProject == nil {
		return padLines(sidebarEmptyLine("No projects", width), width, height)
	}
	if len(m.SidebarSessions) == 0 {
		return padLines(sidebarEmptyLine("No sessions", width), width, height)
	}

	builder := strings.Builder{}
	builder.WriteString(m.renderSidebarSessions(width))
	if m.shouldRenderFilterLine() {
		builder.WriteString("\n")
		filterLine := fmt.Sprintf("Filter: %s", m.FilterInput.View())
		builder.WriteString(fitLine(filterLine, width))
		builder.WriteString("\n")
	}

	return padLines(builder.String(), width, height)
}

func sidebarEmptyLine(message string, width int) string {
	return fitLine(theme.StatusWarning.Render("  "+message), width)
}

func (m Model) renderSidebarSessions(width int) string {
	iconSet := icons.Active()
	iconSize := icons.ActiveSize()
	var builder strings.Builder
	last := len(m.SidebarSessions) - 1
	for i, s := range m.SidebarSessions {
		builder.WriteString(m.renderSidebarSessionBlock(s, width, iconSet, iconSize, i == last))
	}
	return builder.String()
}

func (m Model) renderSidebarSessionBlock(s Session, width int, iconSet icons.IconSet, iconSize icons.Size, isLast bool) string {
	var builder strings.Builder
	builder.WriteString(m.renderSidebarSessionLine(s, width, iconSet, iconSize))
	if s.PaneCount <= 0 {
		return appendSidebarGap(builder.String(), isLast)
	}
	if !m.sessionExpanded(s.Name) {
		return appendSidebarGap(builder.String(), isLast)
	}
	builder.WriteString(m.renderSidebarPaneLines(s, width, iconSet, iconSize))
	return appendSidebarGap(builder.String(), isLast)
}

func (m Model) renderSidebarSessionLine(s Session, width int, iconSet icons.IconSet, iconSize icons.Size) string {
	isSelected := s.Name == m.SelectionSession
	marker := " "
	if isSelected {
		marker = theme.SidebarCaret.Render(iconSet.Caret.BySize(iconSize))
	}
	nameStyle := sidebarSessionStyle(s, isSelected)
	name := nameStyle.Render(s.Name)
	count := theme.SidebarMeta.Render(fmt.Sprintf("(%d)", s.PaneCount))
	line := fmt.Sprintf("%s %s %s", marker, name, count)
	return fitLine(line, width) + "\n"
}

func sidebarSessionStyle(s Session, selected bool) lipgloss.Style {
	if s.Status == sessionStopped {
		return theme.SidebarSessionStopped
	}
	if selected {
		return theme.SidebarSessionSelected
	}
	return theme.SidebarSession
}

func (m Model) renderSidebarPaneLines(s Session, width int, iconSet icons.IconSet, iconSize icons.Size) string {
	selectedPane := m.selectedPaneForSession(s)
	var builder strings.Builder
	for _, p := range s.Panes {
		isSelected := selectedPane != "" && p.Index == selectedPane
		paneMarker := " "
		if isSelected {
			paneMarker = theme.SidebarPaneMarker.Render(iconSet.PaneDot.BySize(iconSize))
		}
		paneLabelStyle := theme.SidebarPane
		if isSelected {
			paneLabelStyle = theme.SidebarPaneSelected
		}
		line := fmt.Sprintf("%s %s", paneMarker, paneLabelStyle.Render(paneLabel(p)))
		builder.WriteString(fitLine(line, width))
		builder.WriteString("\n")
	}
	return builder.String()
}

func (m Model) selectedPaneForSession(s Session) string {
	if s.Name != m.SelectionSession {
		return ""
	}
	if m.SelectionPane != "" {
		return m.SelectionPane
	}
	if s.ActivePane != "" {
		return s.ActivePane
	}
	if len(s.Panes) > 0 {
		return s.Panes[0].Index
	}
	return ""
}

func appendSidebarGap(content string, isLast bool) string {
	if isLast {
		return content
	}
	return content + "\n"
}

func (m Model) shouldRenderFilterLine() bool {
	return m.FilterActive || strings.TrimSpace(m.FilterInput.Value()) != ""
}

func (m Model) sessionExpanded(name string) bool {
	if m.ExpandedSessions == nil {
		return true
	}
	expanded, ok := m.ExpandedSessions[name]
	if !ok {
		return true
	}
	return expanded
}

func (m Model) viewPreview(width, height int) string {
	if width <= 0 || height <= 0 {
		return ""
	}
	session := m.PreviewSession
	if session == nil {
		return padLines(m.EmptyStateMessage, width, height)
	}
	panes := session.Panes

	lines := []string{}
	gridHeight := height
	if gridHeight < 1 {
		gridHeight = 1
	}
	gridWidth := width
	grid := renderPanePreviewWithRenderer(panes, gridWidth, gridHeight, panePreviewContext{
		mode:          m.PreviewMode,
		compact:       m.PreviewCompact,
		targetPane:    m.SelectionPane,
		terminalFocus: m.TerminalFocus,
		renderer:      m.renderPaneTileLive,
	})
	lines = append(lines, grid)

	return padLines(strings.Join(lines, "\n"), width, height)
}

func (m Model) viewThumbnails(width int) string {
	sessions := collectRunningSessions(m.Projects)
	if len(sessions) == 0 {
		return padLines("No running sessions", width, 3)
	}

	boxes := []string{}
	boxWidth := 16
	maxBoxes := width / (boxWidth + 1)
	if maxBoxes < 1 {
		maxBoxes = 1
	}
	if len(sessions) > maxBoxes {
		sessions = sessions[:maxBoxes]
	}

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(theme.Border).
		Width(boxWidth).
		Height(2).
		Padding(0, 1)

	for _, s := range sessions {
		badge := renderBadge(sessionBadgeStatus(s))
		name := fitLine(fmt.Sprintf("%s %s", badge, s.Name), boxWidth-2)
		line := s.ThumbnailLine
		if line == "" {
			line = "idle"
		}
		content := fmt.Sprintf("%s\n%s", name, truncateLine(line, boxWidth-2))
		boxes = append(boxes, boxStyle.Render(content))
	}

	row := lipgloss.JoinHorizontal(lipgloss.Top, boxes...)
	return padLines(row, width, 3)
}

func (m Model) viewFooter(width int) string {
	projectKeys := m.Keys.ProjectKeys
	sessionKeys := m.Keys.SessionKeys
	paneKeys := m.Keys.PaneKeys
	sessionLabel := "session/pane"
	paneLabel := "pane"
	if m.Tab == tabDashboard {
		sessionLabel = "pane"
		paneLabel = "project"
	}
	modeHint := ""
	modeHintRendered := ""
	if m.SupportsTerminalFocus {
		label := "terminal"
		if m.TerminalFocus {
			label = "terminal on"
		}
		modeHint = fmt.Sprintf(" · %s %s", strings.ToLower(m.Keys.TerminalFocus), label)
		if m.TerminalFocus {
			modeHintRendered = theme.TerminalFocusHint.Render(modeHint)
		} else {
			modeHintRendered = theme.ListDimmed.Render(modeHint)
		}
	}
	base := fmt.Sprintf(
		"%s ←/→ project · %s ↑/↓ %s · %s %s · %s commands · %s help · %s quit",
		projectKeys,
		sessionKeys,
		sessionLabel,
		paneKeys,
		paneLabel,
		m.Keys.CommandPalette,
		m.Keys.Help,
		m.Keys.Quit,
	)
	base = theme.ListDimmed.Render(base) + modeHintRendered
	toast := m.Toast
	if toast == "" {
		return fitLine(base, width)
	}
	line := fmt.Sprintf("%s  %s", base, toast)
	return fitLine(line, width)
}

func (m Model) viewQuickReply(width int) string {
	if width <= 0 {
		return ""
	}
	barWidth := width
	contentWidth := barWidth - 4
	if contentWidth < 10 {
		contentWidth = 10
	}
	maxWidth := contentWidth - 6
	if maxWidth < 0 {
		maxWidth = 0
	}
	minWidth := 20
	if minWidth > maxWidth {
		minWidth = maxWidth
	}
	inputWidth := clamp(contentWidth-18, minWidth, maxWidth)
	m.QuickReplyInput.Width = inputWidth

	hintText := "enter send • esc clear"
	if m.SupportsTerminalFocus {
		toggle := m.Keys.TerminalFocus
		if m.TerminalFocus {
			hintText = fmt.Sprintf("%s quick reply", toggle)
		} else {
			hintText = fmt.Sprintf("%s terminal focus • %s", toggle, hintText)
		}
	}

	base := lipgloss.NewStyle().
		Foreground(theme.TextPrimary).
		Background(theme.QuickReplyBg)
	accent := base.Foreground(theme.QuickReplyAcc).Render("▌ ")
	label := base.Bold(true).Background(theme.QuickReplyTag).Render(" Quick Reply ")
	hint := base.Foreground(theme.TextDim).Italic(true).Render(" " + hintText)

	line := accent + label + m.QuickReplyInput.View() + hint
	line = ansi.Truncate(line, contentWidth, "")
	visible := lipgloss.Width(line)
	if visible < contentWidth {
		line += base.Render(strings.Repeat(" ", contentWidth-visible))
	}

	pad := base.Render(strings.Repeat(" ", 2))
	blank := base.Render(strings.Repeat(" ", contentWidth))
	lines := []string{
		pad + blank + pad,
		pad + line + pad,
		pad + blank + pad,
	}
	return strings.Join(lines, "\n")
}
