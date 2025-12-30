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

	header := m.viewHeader(contentWidth)
	footer := m.viewFooter(contentWidth)
	quickReply := m.viewQuickReply(contentWidth)
	quickReplyHeight := lipgloss.Height(quickReply)

	headerHeight := lipgloss.Height(header)
	footerHeight := lipgloss.Height(footer)
	headerGap := 1
	extraLines := headerHeight + headerGap + footerHeight + quickReplyHeight
	bodyHeight := contentHeight - extraLines
	if bodyHeight < 4 {
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
	sections = append(sections, footer)
	view := lipgloss.JoinVertical(lipgloss.Top, sections...)
	return m.overlaySlashMenu(view, contentWidth, contentHeight, headerHeight, headerGap, bodyHeight)
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
	if m.SidebarHidden {
		return m.viewPreview(width, height)
	}
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
	lines := m.buildSplashLines(width, height)
	return renderSplashLines(lines, width, height, m.EmptyStateMessage)
}

func (m Model) buildSplashLines(width, height int) []string {
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
	return lines
}

func renderSplashLines(lines []string, width, height int, emptyMessage string) string {
	if len(lines) == 0 {
		return padLines(emptyMessage, width, height)
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

	hintText := "enter send • esc clear • up/down history/select • / commands • tab complete"
	if m.SupportsTerminalFocus {
		toggle := m.Keys.TerminalFocus
		if m.TerminalFocus {
			hintText = fmt.Sprintf("%s input mode", toggle)
		} else {
			hintText = fmt.Sprintf("%s terminal focus • %s", toggle, hintText)
		}
	}

	base := lipgloss.NewStyle().
		Foreground(theme.TextPrimary).
		Background(theme.QuickReplyBg)
	accent := base.Foreground(theme.QuickReplyAcc).Render("▌ ")
	hint := base.Foreground(theme.TextDim).Italic(true).Render(" " + hintText)
	inputWidth := contentWidth - lipgloss.Width(accent) - lipgloss.Width(hint)
	if inputWidth < 10 {
		inputWidth = 10
	}
	if inputWidth > contentWidth {
		inputWidth = contentWidth
	}
	m.QuickReplyInput.Width = inputWidth

	line := accent + m.QuickReplyInput.View() + hint
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
	}
	lines = append(lines, pad+blank+pad)
	return strings.Join(lines, "\n")
}

func (m Model) overlaySlashMenu(base string, width, height, headerHeight, headerGap, bodyHeight int) string {
	if len(m.SlashSuggestions) == 0 || width <= 0 || height <= 0 {
		return base
	}
	menuX := 2
	menuWidth := width - 4
	if menuWidth < 10 {
		menuWidth = width
		menuX = 0
	}
	availableHeight := headerHeight + headerGap + bodyHeight
	menu := renderSlashMenu(m.SlashSuggestions, m.SlashSelected, menuWidth, availableHeight)
	if strings.TrimSpace(menu) == "" {
		return base
	}
	menuHeight := lipgloss.Height(menu)
	if menuHeight <= 0 {
		return base
	}
	menuY := availableHeight - menuHeight
	if menuY < 0 {
		menuY = 0
	}
	return overlayAt(base, menu, width, height, menuX, menuY)
}

func renderSlashMenu(suggestions []SlashSuggestion, selectedIdx, width, maxHeight int) string {
	if len(suggestions) == 0 || width <= 0 || maxHeight <= 0 {
		return ""
	}
	visible, selectedVisible := slashMenuVisible(suggestions, selectedIdx, maxHeight)
	if len(visible) == 0 {
		return ""
	}
	commandWidth := slashMenuCommandWidth(visible)
	if commandWidth == 0 {
		return ""
	}
	return renderSlashMenuLines(visible, selectedVisible, width, slashMenuStyles(), commandWidth)
}

type slashMenuStyleSet struct {
	base              lipgloss.Style
	normal            lipgloss.Style
	highlight         lipgloss.Style
	desc              lipgloss.Style
	selectedBase      lipgloss.Style
	selectedNormal    lipgloss.Style
	selectedHighlight lipgloss.Style
	selectedDesc      lipgloss.Style
}

type slashMenuLineStyle struct {
	base      lipgloss.Style
	normal    lipgloss.Style
	highlight lipgloss.Style
	desc      lipgloss.Style
}

func slashMenuStyles() slashMenuStyleSet {
	base := lipgloss.NewStyle().
		Background(theme.Highlight).
		Foreground(theme.TextPrimary)
	selectedBase := lipgloss.NewStyle().
		Background(theme.Accent).
		Foreground(theme.TextPrimary)
	return slashMenuStyleSet{
		base:              base,
		normal:            base.Foreground(theme.TextSecondary),
		highlight:         base.Foreground(theme.TextPrimary),
		desc:              base.Foreground(theme.TextDim),
		selectedBase:      selectedBase,
		selectedNormal:    selectedBase.Foreground(theme.TextPrimary),
		selectedHighlight: selectedBase.Foreground(theme.TextPrimary).Bold(true),
		selectedDesc:      selectedBase.Foreground(theme.TextPrimary),
	}
}

func (s slashMenuStyleSet) lineStyle(selected bool) slashMenuLineStyle {
	if selected {
		return slashMenuLineStyle{
			base:      s.selectedBase,
			normal:    s.selectedNormal,
			highlight: s.selectedHighlight,
			desc:      s.selectedDesc,
		}
	}
	return slashMenuLineStyle{
		base:      s.base,
		normal:    s.normal,
		highlight: s.highlight,
		desc:      s.desc,
	}
}

func slashMenuVisible(suggestions []SlashSuggestion, selectedIdx, maxHeight int) ([]SlashSuggestion, int) {
	if maxHeight <= 2 {
		return nil, -1
	}
	maxSuggestions := 10
	if maxSuggestions > len(suggestions) {
		maxSuggestions = len(suggestions)
	}
	if maxSuggestions > maxHeight-2 {
		maxSuggestions = maxHeight - 2
	}
	if maxSuggestions <= 0 {
		return nil, -1
	}
	if selectedIdx < 0 || selectedIdx >= len(suggestions) {
		selectedIdx = 0
	}
	start := 0
	if selectedIdx >= maxSuggestions {
		start = selectedIdx - (maxSuggestions - 1)
	}
	if start+maxSuggestions > len(suggestions) {
		start = len(suggestions) - maxSuggestions
	}
	if start < 0 {
		start = 0
	}
	end := start + maxSuggestions
	return suggestions[start:end], selectedIdx - start
}

func slashMenuCommandWidth(visible []SlashSuggestion) int {
	commandWidth := 0
	for i := 0; i < len(visible); i++ {
		length := lipgloss.Width(visible[i].Text)
		if length > commandWidth {
			commandWidth = length
		}
	}
	return commandWidth
}

func renderSlashMenuLines(
	visible []SlashSuggestion,
	selectedIdx int,
	width int,
	styles slashMenuStyleSet,
	commandWidth int,
) string {
	contentWidth := slashMenuContentWidth(width)
	lines := make([]string, 0, len(visible)+2)
	lines = append(lines, styles.base.Render(strings.Repeat(" ", width)))
	for i := 0; i < len(visible); i++ {
		line := renderSlashMenuLine(visible[i], styles.lineStyle(i == selectedIdx), commandWidth, width, contentWidth)
		lines = append(lines, line)
	}
	lines = append(lines, styles.base.Render(strings.Repeat(" ", width)))
	return strings.Join(lines, "\n")
}

func renderSlashMenuLine(
	suggestion SlashSuggestion,
	styles slashMenuLineStyle,
	commandWidth int,
	width int,
	contentWidth int,
) string {
	rendered := renderSlashSuggestion(suggestion, styles.normal, styles.highlight)
	rendered = padSlashCommand(rendered, suggestion.Text, commandWidth, styles.base)
	rendered = appendSlashDesc(rendered, suggestion.Desc, styles.base, styles.desc)
	return fitSlashMenuLine(rendered, styles.base, width, contentWidth)
}

func slashMenuContentWidth(width int) int {
	contentWidth := width - 2
	if contentWidth < 1 {
		contentWidth = width
	}
	return contentWidth
}

func padSlashCommand(rendered, command string, commandWidth int, style lipgloss.Style) string {
	cmdPad := commandWidth - lipgloss.Width(command)
	if cmdPad <= 0 {
		return rendered
	}
	return rendered + style.Render(strings.Repeat(" ", cmdPad))
}

func appendSlashDesc(rendered, desc string, baseStyle, descStyle lipgloss.Style) string {
	desc = strings.TrimSpace(desc)
	if desc == "" {
		return rendered
	}
	return rendered + baseStyle.Render("  ") + descStyle.Render(desc)
}

func fitSlashMenuLine(rendered string, baseStyle lipgloss.Style, width, contentWidth int) string {
	rendered = ansi.Truncate(rendered, contentWidth, "")
	visible := lipgloss.Width(rendered)
	if visible < contentWidth {
		rendered += baseStyle.Render(strings.Repeat(" ", contentWidth-visible))
	}
	if contentWidth < width {
		rendered = baseStyle.Render(" ") + rendered + baseStyle.Render(" ")
	}
	return rendered
}

func renderSlashSuggestion(suggestion SlashSuggestion, normal, highlight lipgloss.Style) string {
	text := suggestion.Text
	if text == "" {
		return ""
	}
	matchLen := suggestion.MatchLen
	if matchLen < 0 {
		matchLen = 0
	}
	if matchLen > len(text) {
		matchLen = len(text)
	}
	if matchLen == 0 {
		return normal.Render(text)
	}
	prefix := text[:matchLen]
	rest := text[matchLen:]
	return highlight.Render(prefix) + normal.Render(rest)
}
