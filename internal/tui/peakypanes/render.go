package peakypanes

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/cellbuf"
	"github.com/regenrek/peakypanes/internal/pathutil"
	"github.com/regenrek/peakypanes/internal/tui/icons"
	"github.com/regenrek/peakypanes/internal/tui/theme"
)

func (m Model) viewDashboard() string {
	return appStyle.Render(m.viewDashboardContent())
}

func (m Model) viewDashboardContent() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}
	h, v := appStyle.GetFrameSize()
	contentWidth := m.width - h
	contentHeight := m.height - v
	if contentWidth <= 10 || contentHeight <= 6 {
		return "Terminal too small"
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
	line := headerLine(m.headerParts())
	return fitLine(line, width)
}

func (m Model) viewBody(width, height int) string {
	if height <= 0 {
		return ""
	}
	if m.tab == TabDashboard {
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
	columns := collectDashboardColumns(m.data.Projects)
	if len(columns) == 0 {
		return padLines(m.emptyStateMessage(), width, height)
	}
	columns = m.filteredDashboardColumns(columns)
	totalPanes := 0
	for _, column := range columns {
		totalPanes += len(column.Panes)
	}
	if totalPanes == 0 {
		if strings.TrimSpace(m.filterInput.Value()) != "" {
			return padLines("No panes match the current filter.", width, height)
		}
		return padLines(m.emptyStateMessage(), width, height)
	}
	selectedProject := m.dashboardSelectedProject(columns)
	previewLines := dashboardPreviewLines(m.settings)
	return renderDashboardColumnsWithRenderer(columns, width, height, selectedProject, m.selection, previewLines, m.renderDashboardPaneTileLive)
}

func (m Model) viewSidebar(width, height int) string {
	builder := strings.Builder{}

	project := m.selectedProject()
	if project == nil {
		builder.WriteString(fitLine(theme.StatusWarning.Render("  No projects"), width))
		return padLines(builder.String(), width, height)
	}

	sessions := m.filteredSessions(project.Sessions)
	if len(sessions) == 0 {
		builder.WriteString(fitLine(theme.StatusWarning.Render("  No sessions"), width))
		return padLines(builder.String(), width, height)
	}

	iconSet := icons.Active()
	iconSize := icons.ActiveSize()
	for i, s := range sessions {
		isSelectedSession := s.Name == m.selection.Session
		marker := " "
		if isSelectedSession {
			marker = theme.SidebarCaret.Render(iconSet.Caret.BySize(iconSize))
		}
		nameStyle := theme.SidebarSession
		if isSelectedSession {
			nameStyle = theme.SidebarSessionSelected
		}
		if s.Status == StatusStopped {
			nameStyle = theme.SidebarSessionStopped
		}
		name := nameStyle.Render(s.Name)
		count := theme.SidebarMeta.Render(fmt.Sprintf("(%d)", s.PaneCount))
		line := fmt.Sprintf("%s %s %s", marker, name, count)
		builder.WriteString(fitLine(line, width))
		builder.WriteString("\n")
		if s.PaneCount <= 0 {
			continue
		}
		expanded := m.sessionExpanded(s.Name)
		if !expanded {
			if i < len(sessions)-1 {
				builder.WriteString("\n")
			}
			continue
		}
		selectedPane := ""
		if isSelectedSession {
			selectedPane = m.selection.Pane
			if selectedPane == "" {
				selectedPane = activePaneIndex(s.Panes)
			}
		}
		for _, p := range s.Panes {
			isSelectedPane := isSelectedSession && selectedPane != "" && p.Index == selectedPane
			paneMarker := " "
			if isSelectedPane {
				paneMarker = theme.SidebarPaneMarker.Render(iconSet.PaneDot.BySize(iconSize))
			}
			paneLabelStyle := theme.SidebarPane
			if isSelectedPane {
				paneLabelStyle = theme.SidebarPaneSelected
			}
			pline := fmt.Sprintf("%s %s", paneMarker, paneLabelStyle.Render(paneLabel(p)))
			builder.WriteString(fitLine(pline, width))
			builder.WriteString("\n")
		}
		if i < len(sessions)-1 {
			builder.WriteString("\n")
		}
	}

	if m.filterActive || strings.TrimSpace(m.filterInput.Value()) != "" {
		builder.WriteString("\n")
		filterLine := fmt.Sprintf("Filter: %s", m.filterInput.View())
		builder.WriteString(fitLine(filterLine, width))
		builder.WriteString("\n")
	}

	return padLines(builder.String(), width, height)
}

func (m Model) sessionExpanded(name string) bool {
	if m.expandedSessions == nil {
		return true
	}
	expanded, ok := m.expandedSessions[name]
	if !ok {
		return true
	}
	return expanded
}

func (m Model) viewPreview(width, height int) string {
	if width <= 0 || height <= 0 {
		return ""
	}
	session := m.selectedSession()
	if session == nil {
		return padLines(m.emptyStateMessage(), width, height)
	}
	panes := session.Panes

	lines := []string{}
	gridHeight := height
	if gridHeight < 1 {
		gridHeight = 1
	}
	gridWidth := width
	grid := renderPanePreviewWithRenderer(panes, gridWidth, gridHeight, m.settings.PreviewMode, m.settings.PreviewCompact, m.selection.Pane, m.renderPaneTileLive)
	lines = append(lines, grid)

	return padLines(strings.Join(lines, "\n"), width, height)
}

func (m Model) viewThumbnails(width int) string {
	sessions := collectRunningSessions(m.data.Projects)
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
		line := s.Thumbnail.Line
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
	projectKeys := joinKeyLabels(m.keys.projectLeft, m.keys.projectRight)
	sessionKeys := joinKeyLabels(m.keys.sessionUp, m.keys.sessionDown)
	paneKeys := joinKeyLabels(m.keys.paneNext, m.keys.panePrev)
	sessionLabel := "session/pane"
	paneLabel := "pane"
	if m.tab == TabDashboard {
		sessionLabel = "pane"
		paneLabel = "project"
	}
	modeHint := ""
	if m.supportsTerminalFocus() {
		label := "terminal"
		if m.terminalFocus {
			label = "terminal on"
		}
		modeHint = fmt.Sprintf(" · %s %s", keyLabel(m.keys.terminalFocus), label)
	}
	base := fmt.Sprintf(
		"%s ←/→ project · %s ↑/↓ %s · %s %s · %s commands · %s help · %s quit%s",
		projectKeys,
		sessionKeys,
		sessionLabel,
		paneKeys,
		paneLabel,
		keyLabel(m.keys.commandPalette),
		keyLabel(m.keys.help),
		keyLabel(m.keys.quit),
		modeHint,
	)
	base = theme.ListDimmed.Render(base)
	toast := m.toastText()
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
	m.quickReplyInput.Width = inputWidth

	hintText := "enter send • esc clear"
	if m.supportsTerminalFocus() {
		toggle := keyLabel(m.keys.terminalFocus)
		if m.terminalFocus {
			hintText = fmt.Sprintf("%s quick reply", toggle)
		} else {
			hintText = fmt.Sprintf("%s terminal focus • %s", toggle, hintText)
		}
	}

	base := lipgloss.NewStyle().
		Foreground(theme.TextPrimary).
		Background(theme.QuickReplyBg)
	accent := base.Copy().
		Foreground(theme.QuickReplyAcc).
		Render("▌ ")
	label := base.Copy().
		Bold(true).
		Background(theme.QuickReplyTag).
		Render(" Quick Reply ")
	hint := base.Copy().
		Foreground(theme.TextDim).
		Italic(true).
		Render(" " + hintText)

	line := accent + label + m.quickReplyInput.View() + hint
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

func (m Model) viewConfirmKill() string {
	var dialogContent strings.Builder

	dialogContent.WriteString(dialogTitleStyle.Render("⚠️  Kill Session?"))
	dialogContent.WriteString("\n\n")
	if m.confirmSession != "" {
		dialogContent.WriteString(theme.DialogLabel.Render("Session: "))
		dialogContent.WriteString(theme.DialogValue.Render(m.confirmSession))
		dialogContent.WriteString("\n")
		if m.confirmProject != "" {
			dialogContent.WriteString(theme.DialogLabel.Render("Project: "))
			dialogContent.WriteString(theme.DialogValue.Render(m.confirmProject))
			dialogContent.WriteString("\n")
		}
		dialogContent.WriteString("\n")
	}

	dialogContent.WriteString(theme.DialogNote.Render("Kill the session: This won't delete your project"))
	dialogContent.WriteString("\n\n")

	dialogContent.WriteString(theme.DialogChoiceKey.Render("y"))
	dialogContent.WriteString(theme.DialogChoiceSep.Render(" confirm • "))
	dialogContent.WriteString(theme.DialogChoiceKey.Render("n"))
	dialogContent.WriteString(theme.DialogChoiceSep.Render(" cancel"))

	dialog := dialogStyle.Render(dialogContent.String())
	return m.overlayDialog(dialog)
}

func (m Model) viewConfirmCloseProject() string {
	var dialogContent strings.Builder

	dialogContent.WriteString(dialogTitleStyle.Render("⚠️  Close Project?"))
	dialogContent.WriteString("\n\n")
	if m.confirmClose != "" {
		dialogContent.WriteString(theme.DialogLabel.Render("Project: "))
		dialogContent.WriteString(theme.DialogValue.Render(m.confirmClose))
		dialogContent.WriteString("\n")
		if project := findProject(m.data.Projects, m.confirmClose); project != nil {
			running := 0
			for _, s := range project.Sessions {
				if s.Status != StatusStopped {
					running++
				}
			}
			dialogContent.WriteString(theme.DialogLabel.Render("Running sessions: "))
			dialogContent.WriteString(theme.DialogValue.Render(fmt.Sprintf("%d", running)))
			dialogContent.WriteString("\n")
		}
		dialogContent.WriteString("\n")
	}

	dialogContent.WriteString(theme.DialogNote.Render("Close hides the project from tabs; sessions stay running."))
	dialogContent.WriteString("\n")
	dialogContent.WriteString(theme.DialogNote.Render("Press k to kill running sessions instead."))
	dialogContent.WriteString("\n\n")

	dialogContent.WriteString(theme.DialogChoiceKey.Render("y"))
	dialogContent.WriteString(theme.DialogChoiceSep.Render(" close • "))
	dialogContent.WriteString(theme.DialogChoiceKey.Render("k"))
	dialogContent.WriteString(theme.DialogChoiceSep.Render(" kill sessions • "))
	dialogContent.WriteString(theme.DialogChoiceKey.Render("n"))
	dialogContent.WriteString(theme.DialogChoiceSep.Render(" cancel"))

	dialog := dialogStyle.Render(dialogContent.String())
	return m.overlayDialog(dialog)
}

func (m Model) viewConfirmClosePane() string {
	var dialogContent strings.Builder

	dialogContent.WriteString(dialogTitleStyle.Render("⚠️  Close Pane?"))
	dialogContent.WriteString("\n\n")
	if m.confirmPaneTitle != "" {
		dialogContent.WriteString(theme.DialogLabel.Render("Pane: "))
		dialogContent.WriteString(theme.DialogValue.Render(m.confirmPaneTitle))
		dialogContent.WriteString("\n")
	}
	if m.confirmPaneSession != "" {
		dialogContent.WriteString(theme.DialogLabel.Render("Session: "))
		dialogContent.WriteString(theme.DialogValue.Render(m.confirmPaneSession))
		dialogContent.WriteString("\n")
	}
	dialogContent.WriteString("\n")

	if m.confirmPaneRunning {
		dialogContent.WriteString(theme.DialogNote.Render("The pane is still running. Closing it will stop the process."))
		dialogContent.WriteString("\n\n")
	}

	dialogContent.WriteString(theme.DialogChoiceKey.Render("y"))
	dialogContent.WriteString(theme.DialogChoiceSep.Render(" close • "))
	dialogContent.WriteString(theme.DialogChoiceKey.Render("n"))
	dialogContent.WriteString(theme.DialogChoiceSep.Render(" cancel"))

	dialog := dialogStyle.Render(dialogContent.String())
	return m.overlayDialog(dialog)
}

func (m Model) viewRename() string {
	var dialogContent strings.Builder

	title := "Rename Session"
	switch m.state {
	case StateRenamePane:
		title = "Rename Pane"
	}
	dialogContent.WriteString(dialogTitleStyle.Render(title))
	dialogContent.WriteString("\n\n")

	if m.state == StateRenamePane {
		if strings.TrimSpace(m.renameSession) != "" {
			dialogContent.WriteString(theme.DialogLabel.Render("Session: "))
			dialogContent.WriteString(theme.DialogValue.Render(m.renameSession))
			dialogContent.WriteString("\n")
		}
		paneLabel := strings.TrimSpace(m.renamePane)
		if paneLabel == "" && strings.TrimSpace(m.renamePaneIndex) != "" {
			paneLabel = fmt.Sprintf("pane %s", strings.TrimSpace(m.renamePaneIndex))
		}
		if paneLabel != "" {
			dialogContent.WriteString(theme.DialogLabel.Render("Pane: "))
			dialogContent.WriteString(theme.DialogValue.Render(paneLabel))
			dialogContent.WriteString("\n")
		}
		dialogContent.WriteString("\n")
	} else if strings.TrimSpace(m.renameSession) != "" {
		dialogContent.WriteString(theme.DialogLabel.Render("Session: "))
		dialogContent.WriteString(theme.DialogValue.Render(m.renameSession))
		dialogContent.WriteString("\n\n")
	}

	inputWidth := 40
	if m.width > 0 {
		inputWidth = clamp(m.width-30, 20, 60)
	}
	m.renameInput.Width = inputWidth
	dialogContent.WriteString(theme.DialogLabel.Render("New name: "))
	dialogContent.WriteString(m.renameInput.View())
	dialogContent.WriteString("\n\n")

	dialogContent.WriteString(theme.DialogChoiceKey.Render("enter"))
	dialogContent.WriteString(theme.DialogChoiceSep.Render(" confirm • "))
	dialogContent.WriteString(theme.DialogChoiceKey.Render("esc"))
	dialogContent.WriteString(theme.DialogChoiceSep.Render(" cancel"))

	dialog := dialogStyle.Render(dialogContent.String())
	return m.overlayDialog(dialog)
}

func (m Model) viewProjectRootSetup() string {
	var dialogContent strings.Builder

	dialogContent.WriteString(dialogTitleStyle.Render("Project Roots"))
	dialogContent.WriteString("\n\n")
	dialogContent.WriteString(theme.DialogNote.Render("Comma-separated list of folders to scan for git projects."))
	dialogContent.WriteString("\n\n")

	inputWidth := 60
	if m.width > 0 {
		inputWidth = clamp(m.width-30, 24, 80)
	}
	m.projectRootInput.Width = inputWidth
	dialogContent.WriteString(theme.DialogLabel.Render("Roots: "))
	dialogContent.WriteString(m.projectRootInput.View())
	dialogContent.WriteString("\n\n")

	dialogContent.WriteString(theme.DialogChoiceKey.Render("enter"))
	dialogContent.WriteString(theme.DialogChoiceSep.Render(" save • "))
	dialogContent.WriteString(theme.DialogChoiceKey.Render("esc"))
	dialogContent.WriteString(theme.DialogChoiceSep.Render(" cancel"))

	dialog := dialogStyle.Render(dialogContent.String())
	return m.overlayDialog(dialog)
}

func (m Model) overlayDialog(dialog string) string {
	if m.width == 0 || m.height == 0 {
		return appStyle.Render(dialog)
	}
	base := appStyle.Render(theme.ListDimmed.Render(m.viewDashboardContent()))
	return overlayCentered(base, dialog, m.width, m.height)
}

func (m Model) viewLayoutPicker() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}
	base := appStyle.Render(theme.ListDimmed.Render(m.viewDashboardContent()))
	listW := m.layoutPicker.Width()
	listH := m.layoutPicker.Height()
	frameW, frameH := dialogStyle.GetFrameSize()
	overlayW := listW + frameW
	overlayH := listH + frameH
	content := lipgloss.NewStyle().Width(listW).Height(listH).Render(m.layoutPicker.View())
	dialog := dialogStyle.Width(overlayW).Height(overlayH).Render(content)
	return overlayCenteredSized(base, dialog, m.width, m.height, overlayW, overlayH)
}

func (m Model) viewPaneSplitPicker() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}
	base := appStyle.Render(theme.ListDimmed.Render(m.viewDashboardContent()))
	var dialogContent strings.Builder
	dialogContent.WriteString(dialogTitleStyle.Render("➕ Add Pane"))
	dialogContent.WriteString("\n\n")
	dialogContent.WriteString(theme.DialogNote.Render("Choose split direction"))
	dialogContent.WriteString("\n\n")
	dialogContent.WriteString(theme.DialogChoiceKey.Render("r"))
	dialogContent.WriteString(theme.DialogChoiceSep.Render(" split right • "))
	dialogContent.WriteString(theme.DialogChoiceKey.Render("d"))
	dialogContent.WriteString(theme.DialogChoiceSep.Render(" split down • "))
	dialogContent.WriteString(theme.DialogChoiceKey.Render("esc"))
	dialogContent.WriteString(theme.DialogChoiceSep.Render(" cancel"))

	dialog := dialogStyle.Render(dialogContent.String())
	return overlayCentered(base, dialog, m.width, m.height)
}

func (m Model) viewPaneSwapPicker() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}
	base := appStyle.Render(theme.ListDimmed.Render(m.viewDashboardContent()))
	listW := m.paneSwapPicker.Width()
	listH := m.paneSwapPicker.Height()
	frameW, frameH := dialogStyle.GetFrameSize()
	overlayW := listW + frameW
	overlayH := listH + frameH
	content := lipgloss.NewStyle().Width(listW).Height(listH).Render(m.paneSwapPicker.View())
	dialog := dialogStyle.Width(overlayW).Height(overlayH).Render(content)
	return overlayCenteredSized(base, dialog, m.width, m.height, overlayW, overlayH)
}

func (m Model) viewCommandPalette() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}
	base := appStyle.Render(theme.ListDimmed.Render(m.viewDashboardContent()))
	listW := m.commandPalette.Width()
	listH := m.commandPalette.Height()
	frameW, frameH := dialogStyle.GetFrameSize()
	overlayW := listW + frameW
	overlayH := listH + frameH
	content := lipgloss.NewStyle().Width(listW).Height(listH).Render(m.commandPalette.View())
	dialog := dialogStyle.Width(overlayW).Height(overlayH).Render(content)
	return overlayCenteredSized(base, dialog, m.width, m.height, overlayW, overlayH)
}

func (m Model) viewHelp() string {
	var left strings.Builder
	left.WriteString("Navigation\n")
	left.WriteString(fmt.Sprintf("  %s Switch projects\n", joinKeyLabels(m.keys.projectLeft, m.keys.projectRight)))
	left.WriteString(fmt.Sprintf("  %s Switch sessions/panes (project view)\n", joinKeyLabels(m.keys.sessionUp, m.keys.sessionDown)))
	left.WriteString(fmt.Sprintf("  %s Switch sessions only (project view)\n", joinKeyLabels(m.keys.sessionOnlyUp, m.keys.sessionOnlyDown)))
	left.WriteString(fmt.Sprintf("  %s Switch panes (project view)\n", joinKeyLabels(m.keys.paneNext, m.keys.panePrev)))
	left.WriteString(fmt.Sprintf("  %s Switch panes (dashboard)\n", joinKeyLabels(m.keys.sessionUp, m.keys.sessionDown)))
	left.WriteString(fmt.Sprintf("  %s Switch project column (dashboard)\n", joinKeyLabels(m.keys.paneNext, m.keys.panePrev)))
	left.WriteString("\nProject\n")
	left.WriteString(fmt.Sprintf("  %s Open project picker\n", keyLabel(m.keys.openProject)))
	left.WriteString(fmt.Sprintf("  %s Close project\n", keyLabel(m.keys.closeProject)))
	left.WriteString("\nSession\n")
	left.WriteString("  enter Attach/start session (when reply empty)\n")
	left.WriteString(fmt.Sprintf("  %s New session (pick layout)\n", keyLabel(m.keys.newSession)))
	left.WriteString(fmt.Sprintf("  %s Kill session\n", keyLabel(m.keys.kill)))
	left.WriteString("\nPane\n")
	left.WriteString("  type  Quick reply (terminal focus off)\n")
	left.WriteString("  enter Send quick reply\n")
	left.WriteString("  esc   Clear quick reply\n")
	left.WriteString(fmt.Sprintf("  %s Toggle terminal focus (Peaky Panes sessions)\n", keyLabel(m.keys.terminalFocus)))
	left.WriteString(fmt.Sprintf("  %s Scrollback mode (Peaky Panes sessions)\n", keyLabel(m.keys.scrollback)))
	left.WriteString(fmt.Sprintf("  %s Copy mode (Peaky Panes sessions)\n", keyLabel(m.keys.copyMode)))
	left.WriteString("  type  Send input to focused pane\n")

	var right strings.Builder
	right.WriteString("Pane List\n")
	right.WriteString(fmt.Sprintf("  %s Toggle pane list\n", keyLabel(m.keys.togglePanes)))
	right.WriteString("\nOther\n")
	right.WriteString(fmt.Sprintf("  %s Refresh\n", keyLabel(m.keys.refresh)))
	right.WriteString(fmt.Sprintf("  %s Edit config\n", keyLabel(m.keys.editConfig)))
	right.WriteString(fmt.Sprintf("  %s Command palette\n", keyLabel(m.keys.commandPalette)))
	right.WriteString(fmt.Sprintf("  %s Filter sessions\n", keyLabel(m.keys.filter)))
	right.WriteString(fmt.Sprintf("  %s Close help\n", keyLabel(m.keys.help)))
	right.WriteString(fmt.Sprintf("  %s Quit\n", keyLabel(m.keys.quit)))

	colWidth := 36
	if m.width > 0 {
		frameW, _ := dialogStyle.GetFrameSize()
		avail := m.width - frameW - 6
		if avail > 0 {
			candidate := (avail / 2) - 1
			if candidate > 20 {
				colWidth = candidate
			}
		}
	}
	colStyle := lipgloss.NewStyle().Width(colWidth)
	columns := lipgloss.JoinHorizontal(lipgloss.Top, colStyle.Render(left.String()), "  ", colStyle.Render(right.String()))

	var content strings.Builder
	content.WriteString(theme.HelpTitle.Render("Peaky Panes — Help"))
	content.WriteString("\n")
	content.WriteString(columns)

	dialog := dialogStyle.Render(content.String())
	return m.overlayDialog(dialog)
}

// ===== Rendering helpers =====

type canvas struct {
	w     int
	h     int
	cells []rune
}

func newCanvas(w, h int) *canvas {
	c := &canvas{w: w, h: h, cells: make([]rune, w*h)}
	for i := range c.cells {
		c.cells[i] = ' '
	}
	return c
}

func (c *canvas) set(x, y int, r rune) {
	if x < 0 || y < 0 || x >= c.w || y >= c.h {
		return
	}
	c.cells[y*c.w+x] = r
}

func (c *canvas) drawBox(x, y, w, h int) {
	if w < 2 || h < 2 {
		return
	}
	x2 := x + w - 1
	y2 := y + h - 1
	for ix := x + 1; ix < x2; ix++ {
		c.set(ix, y, '-')
		c.set(ix, y2, '-')
	}
	for iy := y + 1; iy < y2; iy++ {
		c.set(x, iy, '|')
		c.set(x2, iy, '|')
	}
	c.set(x, y, '+')
	c.set(x2, y, '+')
	c.set(x, y2, '+')
	c.set(x2, y2, '+')
}

func (c *canvas) write(x, y int, text string, max int) {
	if y < 0 || y >= c.h || max <= 0 {
		return
	}
	trimmed := truncateLine(text, max)
	for i, r := range []rune(trimmed) {
		if x+i >= c.w {
			break
		}
		c.set(x+i, y, r)
	}
}

func (c *canvas) String() string {
	lines := make([]string, c.h)
	for y := 0; y < c.h; y++ {
		lines[y] = string(c.cells[y*c.w : (y+1)*c.w])
	}
	return strings.Join(lines, "\n")
}

type paneTileRenderer func(pane PaneItem, width, height int, compact bool, target bool, borders tileBorders) string

func renderPanePreview(panes []PaneItem, width, height int, mode string, compact bool, targetPane string) string {
	return renderPanePreviewWithRenderer(panes, width, height, mode, compact, targetPane, nil)
}

func renderPanePreviewWithRenderer(panes []PaneItem, width, height int, mode string, compact bool, targetPane string, renderer paneTileRenderer) string {
	if mode == "layout" {
		return renderPaneLayout(panes, width, height, targetPane)
	}
	return renderPaneTilesWithRenderer(panes, width, height, compact, targetPane, renderer)
}

func renderPaneLayout(panes []PaneItem, width, height int, targetPane string) string {
	if width <= 0 || height <= 0 {
		return ""
	}
	if len(panes) == 0 {
		return padLines("No panes", width, height)
	}
	c := newCanvas(width, height)

	maxW, maxH := paneBounds(panes)
	if maxW == 0 || maxH == 0 {
		return padLines("No panes", width, height)
	}

	for _, pane := range panes {
		x1, y1, w, h := scalePane(pane, maxW, maxH, width, height)
		if w < 2 || h < 2 {
			continue
		}
		c.drawBox(x1, y1, w, h)
		title := pane.Title
		if pane.Index == targetPane {
			title = "TARGET " + title
		}
		if pane.Active {
			title = "▶ " + title
		}
		c.write(x1+1, y1+1, title, w-2)
		c.write(x1+1, y1+2, pane.Command, w-2)
		status := lastNonEmpty(pane.Preview)
		if status == "" {
			status = "idle"
		}
		c.write(x1+1, y1+3, status, w-2)
	}

	return c.String()
}

const (
	borderLevelDefault = iota
	borderLevelActive
	borderLevelTarget
)

func borderLevelForPane(pane PaneItem, targetPane string) int {
	if pane.Index == targetPane {
		return borderLevelTarget
	}
	if pane.Active {
		return borderLevelActive
	}
	return borderLevelDefault
}

func borderColorFor(level int) lipgloss.TerminalColor {
	switch level {
	case borderLevelTarget:
		return theme.BorderTarget
	case borderLevelActive:
		return theme.BorderFocused
	default:
		return theme.Border
	}
}

func maxBorderLevel(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func renderPaneTiles(panes []PaneItem, width, height int, compact bool, targetPane string) string {
	return renderPaneTilesWithRenderer(panes, width, height, compact, targetPane, renderPaneTile)
}

func renderPaneTilesWithRenderer(panes []PaneItem, width, height int, compact bool, targetPane string, renderer paneTileRenderer) string {
	if width <= 0 || height <= 0 {
		return ""
	}
	if len(panes) == 0 {
		return padLines("No panes", width, height)
	}
	if renderer == nil {
		renderer = renderPaneTile
	}

	cols := 3
	if width < 70 {
		cols = 2
	}
	if width < 42 {
		cols = 1
	}
	if len(panes) < cols {
		cols = len(panes)
	}
	if cols <= 0 {
		cols = 1
	}

	rows := (len(panes) + cols - 1) / cols
	gap := 0
	availableHeight := height - gap*(rows-1)
	if availableHeight < rows {
		availableHeight = rows
	}
	baseHeight := availableHeight / rows
	extraHeight := availableHeight % rows
	if baseHeight < 4 {
		baseHeight = 4
		extraHeight = 0
	}
	tileWidth := (width - gap*(cols-1)) / cols
	if tileWidth < 14 {
		tileWidth = 14
	}

	paneLevels := make([]int, len(panes))
	for i, pane := range panes {
		paneLevels[i] = borderLevelForPane(pane, targetPane)
	}

	var renderedRows []string
	for r := 0; r < rows; r++ {
		rowHeight := baseHeight
		if r == rows-1 {
			rowHeight += extraHeight
		}
		var tiles []string
		for c := 0; c < cols; c++ {
			idx := r*cols + c
			if idx >= len(panes) {
				tiles = append(tiles, padLines("", tileWidth, rowHeight))
				continue
			}
			level := paneLevels[idx]
			rightLevel := borderLevelDefault
			if c < cols-1 {
				neighbor := idx + 1
				if neighbor < len(panes) {
					rightLevel = paneLevels[neighbor]
				}
			}
			bottomLevel := borderLevelDefault
			if r < rows-1 {
				neighbor := idx + cols
				if neighbor < len(panes) {
					bottomLevel = paneLevels[neighbor]
				}
			}
			colors := tileBorderColors{
				top:    borderColorFor(level),
				left:   borderColorFor(level),
				right:  borderColorFor(maxBorderLevel(level, rightLevel)),
				bottom: borderColorFor(maxBorderLevel(level, bottomLevel)),
			}
			borders := tileBorders{
				top:    r == 0,
				left:   c == 0,
				right:  true,
				bottom: true,
				colors: colors,
			}
			tiles = append(tiles, renderer(panes[idx], tileWidth, rowHeight, compact, panes[idx].Index == targetPane, borders))
		}
		row := lipgloss.JoinHorizontal(lipgloss.Top, tiles...)
		renderedRows = append(renderedRows, row)
	}
	return padLines(strings.Join(renderedRows, "\n"), width, height)
}

func (m Model) dashboardSelectedProject(columns []DashboardProjectColumn) string {
	if len(columns) == 0 {
		return ""
	}
	if m.selection.Project != "" {
		for _, column := range columns {
			if column.ProjectName == m.selection.Project {
				return column.ProjectName
			}
		}
	}
	if m.selection.Session != "" {
		for _, column := range columns {
			for _, pane := range column.Panes {
				if pane.SessionName == m.selection.Session {
					return column.ProjectName
				}
			}
		}
	}
	return columns[0].ProjectName
}

type dashboardPaneRenderer func(pane DashboardPane, width, height, previewLines int, selected bool, iconSet icons.IconSet, iconSize icons.Size) string

func renderDashboardColumnsWithRenderer(columns []DashboardProjectColumn, width, height int, selectedProject string, selection selectionState, previewLines int, renderer dashboardPaneRenderer) string {
	if width <= 0 || height <= 0 {
		return ""
	}
	if len(columns) == 0 {
		return padLines("No projects", width, height)
	}
	gap := 2
	minColWidth := 24
	maxCols := (width + gap) / (minColWidth + gap)
	if maxCols < 1 {
		maxCols = 1
	}
	selectedIndex := 0
	for i, column := range columns {
		if column.ProjectName == selectedProject {
			selectedIndex = i
			break
		}
	}
	if len(columns) > maxCols {
		start := selectedIndex - maxCols/2
		if start < 0 {
			start = 0
		}
		if start+maxCols > len(columns) {
			start = len(columns) - maxCols
		}
		columns = columns[start : start+maxCols]
		selectedIndex = selectedIndex - start
	}
	colWidth := width
	if len(columns) > 1 {
		colWidth = (width - gap*(len(columns)-1)) / len(columns)
	}
	if colWidth < 1 {
		colWidth = 1
	}

	iconSet := icons.Active()
	iconSize := icons.ActiveSize()
	parts := make([]string, 0, len(columns)*2)
	for i, column := range columns {
		if i > 0 {
			parts = append(parts, strings.Repeat(" ", gap))
		}
		selected := i == selectedIndex
		parts = append(parts, renderDashboardColumn(column, colWidth, height, selected, selection, previewLines, iconSet, iconSize, renderer))
	}
	return padLines(lipgloss.JoinHorizontal(lipgloss.Top, parts...), width, height)
}

func renderDashboardColumn(column DashboardProjectColumn, width, height int, selected bool, selection selectionState, previewLines int, iconSet icons.IconSet, iconSize icons.Size, renderer dashboardPaneRenderer) string {
	if width <= 0 || height <= 0 {
		return ""
	}
	name := strings.TrimSpace(column.ProjectName)
	if name == "" {
		name = "project"
	}
	path := pathOrDash(column.ProjectPath)
	headerStyle := theme.TabInactive
	if selected {
		headerStyle = theme.TabActive
	}
	headerLines := []string{
		fitLine(headerStyle.Render(name), width),
		fitLine(theme.SidebarMeta.Render(path), width),
		strings.Repeat("─", width),
	}
	headerHeight := len(headerLines)
	bodyHeight := height - headerHeight
	if bodyHeight <= 0 {
		return padLines(strings.Join(headerLines, "\n"), width, height)
	}

	if len(column.Panes) == 0 {
		body := padLines("No running panes", width, bodyHeight)
		return strings.Join(append(headerLines, body), "\n")
	}

	blockHeight := dashboardPaneBlockHeight(previewLines)
	if blockHeight > bodyHeight {
		blockHeight = bodyHeight
	}
	if blockHeight < 3 {
		blockHeight = bodyHeight
	}
	visibleBlocks := bodyHeight / blockHeight
	if visibleBlocks < 1 {
		visibleBlocks = 1
	}

	selectedIndex := -1
	if selected {
		selectedIndex = dashboardPaneIndex(column.Panes, selection)
	}
	start := 0
	if selected && selectedIndex >= 0 && selectedIndex >= visibleBlocks {
		start = selectedIndex - visibleBlocks + 1
	}
	if start < 0 {
		start = 0
	}
	if start > len(column.Panes)-1 {
		start = len(column.Panes) - 1
	}
	end := start + visibleBlocks
	if end > len(column.Panes) {
		end = len(column.Panes)
	}
	if end < start {
		end = start
	}

	blocks := make([]string, 0, visibleBlocks)
	for i := start; i < end; i++ {
		selectedPane := selected && i == selectedIndex
		if renderer == nil {
			renderer = renderDashboardPaneTile
		}
		blocks = append(blocks, renderer(column.Panes[i], width, blockHeight, previewLines, selectedPane, iconSet, iconSize))
	}
	body := padLines(strings.Join(blocks, "\n"), width, bodyHeight)
	return strings.Join(append(headerLines, body), "\n")
}

func dashboardPaneBlockHeight(previewLines int) int {
	if previewLines < 0 {
		previewLines = 0
	}
	return previewLines + 4
}

func renderDashboardPaneTile(pane DashboardPane, width, height, previewLines int, selected bool, iconSet icons.IconSet, iconSize icons.Size) string {
	if width <= 0 || height <= 0 {
		return ""
	}
	if previewLines < 0 {
		previewLines = 0
	}
	borderColor := theme.Border
	if selected {
		borderColor = theme.BorderTarget
	}
	style := lipgloss.NewStyle().
		Width(width).
		Height(height).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(borderColor).
		Padding(0, 1)

	frameW, frameH := style.GetFrameSize()
	contentWidth := width - frameW
	if contentWidth < 1 {
		contentWidth = 1
	}
	contentHeight := height - frameH
	if contentHeight < 1 {
		contentHeight = 1
	}
	availablePreview := previewLines
	if contentHeight-2 < availablePreview {
		availablePreview = contentHeight - 2
	}
	if availablePreview < 0 {
		availablePreview = 0
	}

	marker := " "
	if selected {
		marker = theme.SidebarCaret.Render(iconSet.Caret.BySize(iconSize))
	}
	label := fmt.Sprintf("%s / %s", pane.SessionName, paneLabel(pane.Pane))
	header := fmt.Sprintf("%s %s %s", marker, renderBadge(pane.Pane.Status), label)
	if selected {
		header = theme.SidebarSessionSelected.Render(header)
	}
	header = truncateTileLine(header, contentWidth)

	detail := strings.TrimSpace(pane.Pane.Command)
	if detail == "" {
		detail = strings.TrimSpace(pane.Pane.Title)
	}
	if detail == "" {
		detail = "-"
	}
	detail = "cmd: " + detail
	if selected {
		detail = theme.SidebarPaneSelected.Render(detail)
	} else {
		detail = theme.SidebarPane.Render(detail)
	}
	detailLine := truncateTileLine(detail, contentWidth)

	lines := []string{header, detailLine}
	preview := tailLines(pane.Pane.Preview, availablePreview)
	for len(preview) < availablePreview {
		preview = append(preview, "")
	}
	for i := 0; i < availablePreview; i++ {
		lines = append(lines, truncateTileLine(preview[i], contentWidth))
	}
	if len(lines) > contentHeight {
		lines = lines[:contentHeight]
	}
	return style.Render(strings.Join(lines, "\n"))
}

func (m Model) renderDashboardPaneTileLive(pane DashboardPane, width, height, previewLines int, selected bool, iconSet icons.IconSet, iconSize icons.Size) string {
	if width <= 0 || height <= 0 {
		return ""
	}
	if previewLines < 0 {
		previewLines = 0
	}
	borderColor := theme.Border
	if selected {
		borderColor = theme.BorderTarget
	}
	style := lipgloss.NewStyle().
		Width(width).
		Height(height).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(borderColor).
		Padding(0, 1)

	frameW, frameH := style.GetFrameSize()
	contentWidth := width - frameW
	if contentWidth < 1 {
		contentWidth = 1
	}
	contentHeight := height - frameH
	if contentHeight < 1 {
		contentHeight = 1
	}
	availablePreview := previewLines
	if contentHeight-2 < availablePreview {
		availablePreview = contentHeight - 2
	}
	if availablePreview < 0 {
		availablePreview = 0
	}

	marker := " "
	if selected {
		marker = theme.SidebarCaret.Render(iconSet.Caret.BySize(iconSize))
	}
	label := fmt.Sprintf("%s / %s", pane.SessionName, paneLabel(pane.Pane))
	header := fmt.Sprintf("%s %s %s", marker, renderBadge(pane.Pane.Status), label)
	if selected {
		header = theme.SidebarSessionSelected.Render(header)
	}
	header = truncateTileLine(header, contentWidth)

	detail := strings.TrimSpace(pane.Pane.Command)
	if detail == "" {
		detail = strings.TrimSpace(pane.Pane.Title)
	}
	if detail == "" {
		detail = "-"
	}
	detail = "cmd: " + detail
	if selected {
		detail = theme.SidebarPaneSelected.Render(detail)
	} else {
		detail = theme.SidebarPane.Render(detail)
	}
	detailLine := truncateTileLine(detail, contentWidth)

	lines := []string{header, detailLine}
	if availablePreview > 0 {
		var live string
		if m.native != nil {
			if win := m.native.Window(pane.Pane.ID); win != nil {
				_ = win.Resize(contentWidth, availablePreview)
				if selected && m.terminalFocus {
					live = win.ViewLipgloss(true)
				} else {
					live = win.ViewANSI()
				}
			}
		}

		if live != "" {
			live = padLines(live, contentWidth, availablePreview)
			lines = append(lines, strings.Split(live, "\n")...)
		} else {
			preview := pane.Pane.Preview
			if len(preview) == 0 {
				if summary := paneSummaryLine(pane.Pane, 0); summary != "" {
					preview = []string{summary}
				}
			}
			preview = tailLines(preview, availablePreview)
			for len(preview) < availablePreview {
				preview = append(preview, "")
			}
			for i := 0; i < availablePreview; i++ {
				lines = append(lines, truncateTileLine(preview[i], contentWidth))
			}
		}
	}
	if len(lines) > contentHeight {
		lines = lines[:contentHeight]
	}
	return style.Render(strings.Join(lines, "\n"))
}

type tileBorderColors struct {
	top    lipgloss.TerminalColor
	right  lipgloss.TerminalColor
	bottom lipgloss.TerminalColor
	left   lipgloss.TerminalColor
}

type tileBorders struct {
	top    bool
	right  bool
	bottom bool
	left   bool
	colors tileBorderColors
}

func renderPaneTile(pane PaneItem, width, height int, compact bool, target bool, borders tileBorders) string {
	title := pane.Title
	if target {
		title = "TARGET " + title
	}
	if pane.Active {
		title = "▶ " + title
	}

	style := lipgloss.NewStyle().
		Width(width).
		Height(height).
		BorderStyle(lipgloss.NormalBorder()).
		BorderTop(borders.top).
		BorderRight(borders.right).
		BorderBottom(borders.bottom).
		BorderLeft(borders.left).
		BorderForeground(theme.Border).
		BorderTopForeground(borders.colors.top).
		BorderRightForeground(borders.colors.right).
		BorderBottomForeground(borders.colors.bottom).
		BorderLeftForeground(borders.colors.left).
		Padding(0, 1)

	frameW, frameH := style.GetFrameSize()
	contentWidth := width - frameW
	if contentWidth < 1 {
		contentWidth = 1
	}
	innerHeight := height - frameH
	if innerHeight < 1 {
		innerHeight = 1
	}

	header := fmt.Sprintf("%s %s", renderBadge(pane.Status), title)
	lines := []string{truncateTileLine(header, contentWidth)}
	if strings.TrimSpace(pane.Command) != "" {
		lines = append(lines, truncateTileLine(pane.Command, contentWidth))
	}

	previewSource := pane.Preview
	if compact {
		previewSource = compactPreviewLines(previewSource)
	}
	previewSource = trimTrailingBlankLines(previewSource)

	maxPreview := innerHeight - len(lines)
	if maxPreview < 0 {
		maxPreview = 0
	}
	previewLines := tailLines(previewSource, maxPreview)
	lines = append(lines, truncateTileLines(previewLines, contentWidth)...)

	content := strings.Join(lines, "\n")
	return style.Render(content)
}

func (m Model) renderPaneTileLive(pane PaneItem, width, height int, compact bool, target bool, borders tileBorders) string {
	title := pane.Title
	if target {
		title = "TARGET " + title
	}
	if pane.Active {
		title = "▶ " + title
	}

	style := lipgloss.NewStyle().
		Width(width).
		Height(height).
		BorderStyle(lipgloss.NormalBorder()).
		BorderTop(borders.top).
		BorderRight(borders.right).
		BorderBottom(borders.bottom).
		BorderLeft(borders.left).
		BorderForeground(theme.Border).
		BorderTopForeground(borders.colors.top).
		BorderRightForeground(borders.colors.right).
		BorderBottomForeground(borders.colors.bottom).
		BorderLeftForeground(borders.colors.left).
		Padding(0, 1)

	frameW, frameH := style.GetFrameSize()
	contentWidth := width - frameW
	if contentWidth < 1 {
		contentWidth = 1
	}
	innerHeight := height - frameH
	if innerHeight < 1 {
		innerHeight = 1
	}

	header := fmt.Sprintf("%s %s", renderBadge(pane.Status), title)
	lines := []string{truncateTileLine(header, contentWidth)}
	if strings.TrimSpace(pane.Command) != "" {
		lines = append(lines, truncateTileLine(pane.Command, contentWidth))
	}

	maxPreview := innerHeight - len(lines)
	if maxPreview < 0 {
		maxPreview = 0
	}
	if maxPreview > 0 {
		var live string
		if m.native != nil {
			if win := m.native.Window(pane.ID); win != nil {
				_ = win.Resize(contentWidth, maxPreview)
				if target && m.terminalFocus {
					live = win.ViewLipgloss(true)
				} else {
					live = win.ViewANSI()
				}
			}
		}

		if live != "" {
			live = padLines(live, contentWidth, maxPreview)
			lines = append(lines, strings.Split(live, "\n")...)
		} else {
			previewSource := pane.Preview
			if len(previewSource) == 0 {
				if summary := paneSummaryLine(pane, 0); summary != "" {
					previewSource = []string{summary}
				}
			}
			if compact {
				previewSource = compactPreviewLines(previewSource)
			}
			previewSource = trimTrailingBlankLines(previewSource)
			previewLines := tailLines(previewSource, maxPreview)
			lines = append(lines, truncateTileLines(previewLines, contentWidth)...)
		}
	}

	content := padLines(strings.Join(lines, "\n"), contentWidth, innerHeight)
	return style.Render(content)
}

func paneBounds(panes []PaneItem) (int, int) {
	maxW := 0
	maxH := 0
	for _, p := range panes {
		if p.Left+p.Width > maxW {
			maxW = p.Left + p.Width
		}
		if p.Top+p.Height > maxH {
			maxH = p.Top + p.Height
		}
	}
	return maxW, maxH
}

func scalePane(p PaneItem, totalW, totalH, width, height int) (int, int, int, int) {
	x1 := int(float64(p.Left) / float64(totalW) * float64(width))
	y1 := int(float64(p.Top) / float64(totalH) * float64(height))
	x2 := int(float64(p.Left+p.Width) / float64(totalW) * float64(width))
	y2 := int(float64(p.Top+p.Height) / float64(totalH) * float64(height))
	w := x2 - x1
	h := y2 - y1
	if w < 2 {
		w = 2
	}
	if h < 2 {
		h = 2
	}
	if x1+w > width {
		w = width - x1
	}
	if y1+h > height {
		h = height - y1
	}
	return x1, y1, w, h
}

func paneLabel(pane PaneItem) string {
	label := strings.TrimSpace(pane.Title)
	if label == "" {
		label = strings.TrimSpace(pane.Command)
	}
	if label == "" {
		return fmt.Sprintf("pane %s", pane.Index)
	}
	if strings.TrimSpace(pane.Index) == "" {
		return label
	}
	return fmt.Sprintf("%s %s", pane.Index, label)
}

func pathOrDash(path string) string {
	if strings.TrimSpace(path) == "" {
		return "-"
	}
	return pathutil.ShortenUser(path)
}

func layoutOrDash(layout string) string {
	if strings.TrimSpace(layout) == "" {
		return "-"
	}
	return layout
}

func (m Model) emptyStateMessage() string {
	openKey := keyLabel(m.keys.openProject)
	if strings.TrimSpace(openKey) == "" {
		openKey = "ctrl+o"
	}
	return strings.Join([]string{
		"No sessions found.",
		"",
		"Tips:",
		"  • Run 'peakypanes init' to create a global config",
		"  • Run 'peakypanes start' in a project directory",
		fmt.Sprintf("  • Press %s to open a project (set dashboard.project_roots)", openKey),
	}, "\n")
}

func renderBadge(status PaneStatus) string {
	switch status {
	case PaneStatusDone:
		return theme.StatusBadgeDone.Render("done")
	case PaneStatusError:
		return theme.StatusBadgeError.Render("error")
	case PaneStatusRunning:
		return theme.StatusBadgeRunning.Render("running")
	default:
		return theme.StatusBadgeIdle.Render("idle")
	}
}

func truncateTileLine(text string, width int) string {
	if width <= 0 {
		return ""
	}
	return ansi.Truncate(text, width, "…")
}

func truncateTileLines(lines []string, width int) []string {
	if len(lines) == 0 {
		return nil
	}
	out := make([]string, len(lines))
	for i, line := range lines {
		out[i] = truncateTileLine(line, width)
	}
	return out
}

func tailLines(lines []string, max int) []string {
	if max <= 0 {
		return nil
	}
	if len(lines) <= max {
		return lines
	}
	return lines[len(lines)-max:]
}

func trimTrailingBlankLines(lines []string) []string {
	end := len(lines)
	for end > 0 {
		if !isBlankANSI(lines[end-1]) {
			break
		}
		end--
	}
	return lines[:end]
}

func compactPreviewLines(lines []string) []string {
	if len(lines) == 0 {
		return nil
	}
	compact := make([]string, 0, len(lines))
	for _, line := range lines {
		if isBlankANSI(line) {
			continue
		}
		compact = append(compact, line)
	}
	return compact
}

func collectRunningSessions(projects []ProjectGroup) []SessionItem {
	var sessions []SessionItem
	for _, p := range projects {
		for _, s := range p.Sessions {
			if s.Status == StatusStopped {
				continue
			}
			sessions = append(sessions, s)
		}
	}
	return sessions
}

func sessionBadgeStatus(session SessionItem) PaneStatus {
	if session.Thumbnail.Line != "" {
		return session.Thumbnail.Status
	}
	if session.Status == StatusRunning || session.Status == StatusCurrent {
		return PaneStatusRunning
	}
	if session.Status == StatusStopped {
		return PaneStatusIdle
	}
	return session.Thumbnail.Status
}

func truncateLine(text string, width int) string {
	if width <= 0 {
		return ""
	}
	if runeWidth(text) <= width {
		return text
	}
	if width <= 1 {
		return "…"
	}
	trim := truncateRunes(text, width-1)
	return trim + "…"
}

func fitLine(text string, width int) string {
	if width <= 0 {
		return ""
	}
	truncated := ansi.Truncate(text, width, "")
	padding := width - lipgloss.Width(truncated)
	if padding < 0 {
		padding = 0
	}
	return truncated + strings.Repeat(" ", padding)
}

func truncateRunes(text string, width int) string {
	if width <= 0 {
		return ""
	}
	count := 0
	for i := range text {
		if count >= width {
			return text[:i]
		}
		count++
	}
	return text
}

func runeWidth(text string) int {
	return utf8.RuneCountInString(text)
}

func padLines(text string, width, height int) string {
	lines := strings.Split(text, "\n")
	if len(lines) > height {
		lines = lines[:height]
	}
	for i, line := range lines {
		lines[i] = padRight(line, width)
	}
	for len(lines) < height {
		lines = append(lines, padRight("", width))
	}
	return strings.Join(lines, "\n")
}

func padRight(text string, width int) string {
	if width <= 0 {
		return ""
	}
	return fitLine(text, width)
}

func overlayCentered(base, overlay string, width, height int) string {
	overlayW := lipgloss.Width(overlay)
	overlayH := lipgloss.Height(overlay)
	return overlayCenteredSized(base, overlay, width, height, overlayW, overlayH)
}

func overlayCenteredSized(base, overlay string, width, height, overlayW, overlayH int) string {
	if width <= 0 || height <= 0 {
		return base
	}
	base = padLines(base, width, height)
	baseBuf := cellbuf.NewBuffer(width, height)
	cellbuf.SetContent(baseBuf, base)

	if overlayW > width {
		overlayW = width
	}
	if overlayH > height {
		overlayH = height
	}
	if overlayW <= 0 || overlayH <= 0 {
		return renderBufferLines(baseBuf)
	}
	x := (width - overlayW) / 2
	y := (height - overlayH) / 2
	rect := cellbuf.Rect(x, y, overlayW, overlayH)

	bgLine := lipgloss.NewStyle().Background(theme.Background).Render(strings.Repeat(" ", overlayW))
	bgBlock := strings.Repeat(bgLine+"\n", overlayH-1) + bgLine
	cellbuf.SetContentRect(baseBuf, bgBlock, rect)
	cellbuf.SetContentRect(baseBuf, overlay, rect)

	return renderBufferLines(baseBuf)
}

func clamp(value, min, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func renderBufferLines(buf *cellbuf.Buffer) string {
	height := buf.Bounds().Dy()
	lines := make([]string, height)
	for y := 0; y < height; y++ {
		_, line := cellbuf.RenderLine(buf, y)
		lines[y] = line
	}
	return strings.Join(lines, "\n")
}
