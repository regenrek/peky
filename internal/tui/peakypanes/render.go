package peakypanes

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/cellbuf"
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
	divider := m.dividerLine(contentWidth)
	footer := m.viewFooter(contentWidth)

	extraLines := 3 // header + divider + footer
	if showThumbs {
		extraLines += thumbHeight + 2 // divider + thumbnails + divider
	}
	bodyHeight := contentHeight - extraLines
	if bodyHeight < 4 {
		showThumbs = false
		thumbHeight = 0
		extraLines = 3
		bodyHeight = contentHeight - extraLines
	}

	body := m.viewBody(contentWidth, bodyHeight)
	sections := []string{header, divider, body}

	if showThumbs {
		sections = append(sections, divider, m.viewThumbnails(contentWidth))
	}
	sections = append(sections, divider, footer)

	return lipgloss.JoinVertical(lipgloss.Top, sections...)
}

func (m Model) viewHeader(width int) string {
	logo := "🎩 Peaky Panes"
	parts := []string{logo, "Projects:"}

	if len(m.data.Projects) == 0 {
		parts = append(parts, theme.TabInactive.Render("none"))
	} else {
		for _, p := range m.data.Projects {
			if p.Name == m.selection.Project {
				parts = append(parts, theme.TabActive.Render(p.Name))
			} else {
				parts = append(parts, theme.TabInactive.Render(p.Name))
			}
		}
	}
	parts = append(parts, theme.TabAdd.Render("+ New"))
	line := strings.Join(parts, " ")
	return fitLine(line, width)
}

func (m Model) viewBody(width, height int) string {
	if height <= 0 {
		return ""
	}
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
	sepLine := "│"
	if height > 1 {
		sepLine = strings.Repeat("│\n", height-1) + "│"
	}
	sep := theme.ListDimmed.Render(sepLine)
	return lipgloss.JoinHorizontal(lipgloss.Top, left, sep, right)
}

func (m Model) viewSidebar(width, height int) string {
	builder := strings.Builder{}
	title := theme.SectionTitle.Render("Sessions")
	builder.WriteString(fitLine(title, width))
	builder.WriteString("\n")

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

	for _, s := range sessions {
		marker := " "
		if s.Name == m.selection.Session {
			marker = "▸"
		}
		line := fmt.Sprintf("%s %s %s (%d)", marker, statusIcon(s.Status), s.Name, s.WindowCount)
		if s.Name == m.selection.Session {
			line = theme.ListSelectedTitle.Render(line)
		} else if s.Status == StatusStopped {
			line = theme.ListDimmed.Render(line)
		}
		builder.WriteString(fitLine(line, width))
		builder.WriteString("\n")
		if s.Name == m.selection.Session {
			if s.WindowCount > 0 {
				builder.WriteString(fitLine(sessionWindowToggleLine(m.expandedSessions[s.Name]), width))
				builder.WriteString("\n")
			}
			if m.expandedSessions[s.Name] && s.WindowCount > 0 {
				for _, w := range s.Windows {
					marker := " "
					if w.Index == m.selection.Window {
						marker = "▸"
					}
					wline := fmt.Sprintf("  %s %s", marker, w.Name)
					if w.Index == m.selection.Window {
						wline = theme.ListSelectedDesc.Render(wline)
					}
					builder.WriteString(fitLine(wline, width))
					builder.WriteString("\n")
				}
			}
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

func (m Model) viewPreview(width, height int) string {
	if width <= 0 || height <= 0 {
		return ""
	}
	session := m.selectedSession()
	if session == nil {
		return padLines(emptyStateMessage(), width, height)
	}
	windowName := ""
	paneCount := 0
	layoutName := ""
	path := ""
	statusBadge := ""
	var panes []PaneItem
	if session != nil {
		layoutName = session.LayoutName
		path = session.Path
		statusBadge = renderBadge(sessionBadgeStatus(*session))
		if window := selectedWindow(session, m.selection.Window); window != nil {
			windowName = window.Name
			panes = window.Panes
			paneCount = len(window.Panes)
		}
	}

	title := fmt.Sprintf("Pane Preview (%s)  • refresh: %ds", windowNameOrDash(windowName), int(m.settings.RefreshInterval.Seconds()))
	if windowName == "" {
		title = "Pane Preview"
	}
	lines := []string{fitLine(title, width)}

	gridHeight := height - 4
	if gridHeight < 4 {
		gridHeight = 4
	}
	gridWidth := width
	grid := renderPanePreview(panes, gridWidth, gridHeight, m.settings.PreviewMode, m.settings.PreviewCompact)
	lines = append(lines, grid)

	windowBar := m.viewWindowBar(width)
	lines = append(lines, windowBar)

	infoLine := fmt.Sprintf("Path: %s  •  Layout: %s", pathOrDash(path), layoutOrDash(layoutName))
	muxName := m.muxClient.Type().String()
	mode := "outside " + muxName
	if m.insideMux {
		mode = "inside " + muxName
	}
	statusLine := fmt.Sprintf("Status: %s  •  Panes: %d  •  %s", statusBadge, paneCount, mode)
	lines = append(lines, fitLine(infoLine, width))
	lines = append(lines, fitLine(statusLine, width))

	return padLines(strings.Join(lines, "\n"), width, height)
}

func (m Model) viewWindowBar(width int) string {
	session := m.selectedSession()
	if session == nil || len(session.Windows) == 0 {
		return fitLine("[ no windows ]", width)
	}
	parts := make([]string, 0, len(session.Windows))
	for _, w := range session.Windows {
		label := w.Name
		if strings.TrimSpace(label) == "" {
			label = w.Index
		}
		if w.Index == m.selection.Window {
			parts = append(parts, theme.TabActive.Render(label))
		} else {
			parts = append(parts, theme.TabInactive.Render(label))
		}
	}
	line := strings.Join(parts, " ")
	return fitLine(line, width)
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
	base := "←/→ project   ↑/↓ session   ⇧↑/⇧↓ window   ^p commands   t new term   ? help"
	toast := m.toastText()
	if toast == "" {
		return fitLine(base, width)
	}
	line := fmt.Sprintf("%s   %s", base, toast)
	return fitLine(line, width)
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

	dialogContent.WriteString(theme.DialogNote.Render("This will kill all running sessions in the project."))
	dialogContent.WriteString("\n\n")

	dialogContent.WriteString(theme.DialogChoiceKey.Render("y"))
	dialogContent.WriteString(theme.DialogChoiceSep.Render(" confirm • "))
	dialogContent.WriteString(theme.DialogChoiceKey.Render("n"))
	dialogContent.WriteString(theme.DialogChoiceSep.Render(" cancel"))

	dialog := dialogStyle.Render(dialogContent.String())
	return m.overlayDialog(dialog)
}

func (m Model) viewRename() string {
	var dialogContent strings.Builder

	title := "Rename Session"
	if m.state == StateRenameWindow {
		title = "Rename Window"
	}
	dialogContent.WriteString(dialogTitleStyle.Render(title))
	dialogContent.WriteString("\n\n")

	if m.state == StateRenameWindow {
		if strings.TrimSpace(m.renameSession) != "" {
			dialogContent.WriteString(theme.DialogLabel.Render("Session: "))
			dialogContent.WriteString(theme.DialogValue.Render(m.renameSession))
			dialogContent.WriteString("\n")
		}
		if strings.TrimSpace(m.renameWindow) != "" {
			dialogContent.WriteString(theme.DialogLabel.Render("Window: "))
			dialogContent.WriteString(theme.DialogValue.Render(m.renameWindow))
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
	left.WriteString("  ←/→   Switch projects\n")
	left.WriteString("  ↑/↓   Switch sessions\n")
	left.WriteString("  ⇧↑/⇧↓ Switch windows\n")
	left.WriteString("\nProject\n")
	left.WriteString("  o     Open project picker\n")
	left.WriteString("  c     Close project\n")
	left.WriteString("\nSession\n")
	left.WriteString("  enter Attach/start session\n")
	left.WriteString("  n     New session (pick layout)\n")
	left.WriteString("  t     Open in new terminal window\n")
	left.WriteString("  K     Kill session\n")

	var right strings.Builder
	right.WriteString("Window\n")
	right.WriteString("  space Toggle window list\n")
	right.WriteString("\nTmux\n")
	right.WriteString("  prefix+g Open dashboard popup\n")
	right.WriteString("\nOther\n")
	right.WriteString("  r     Refresh\n")
	right.WriteString("  e     Edit config\n")
	right.WriteString("  ^p    Command palette\n")
	right.WriteString("  /     Filter sessions\n")
	right.WriteString("  ?     Close help\n")
	right.WriteString("  q     Quit\n")

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

func renderPanePreview(panes []PaneItem, width, height int, mode string, compact bool) string {
	if mode == "layout" {
		return renderPaneLayout(panes, width, height)
	}
	return renderPaneTiles(panes, width, height, compact)
}

func renderPaneLayout(panes []PaneItem, width, height int) string {
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

func renderPaneTiles(panes []PaneItem, width, height int, compact bool) string {
	if width <= 0 || height <= 0 {
		return ""
	}
	if len(panes) == 0 {
		return padLines("No panes", width, height)
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
	gap := 1
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
			tiles = append(tiles, renderPaneTile(panes[idx], tileWidth, rowHeight, compact))
		}
		row := lipgloss.JoinHorizontal(lipgloss.Top, tiles...)
		renderedRows = append(renderedRows, row)
		if r < rows-1 {
			renderedRows = append(renderedRows, strings.Repeat(" ", width))
		}
	}
	return padLines(strings.Join(renderedRows, "\n"), width, height)
}

func renderPaneTile(pane PaneItem, width, height int, compact bool) string {
	title := pane.Title
	if pane.Active {
		title = "▶ " + title
	}
	header := fmt.Sprintf("%s %s", renderBadge(pane.Status), title)
	lines := []string{header}
	if strings.TrimSpace(pane.Command) != "" {
		lines = append(lines, pane.Command)
	}
	previewSource := pane.Preview
	if compact {
		previewSource = compactPreviewLines(previewSource)
	}
	previewLines := tailLines(previewSource, height-3)
	lines = append(lines, previewLines...)

	content := strings.Join(lines, "\n")
	style := lipgloss.NewStyle().
		Width(width).
		Height(height).
		Border(lipgloss.NormalBorder()).
		BorderForeground(theme.Border).
		Padding(0, 1)
	if pane.Active {
		style = style.BorderForeground(theme.BorderFocused)
	}
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

func sessionWindowToggleLine(expanded bool) string {
	if expanded {
		return "  [▾] windows"
	}
	return "  [▸] windows"
}

func windowNameOrDash(name string) string {
	if strings.TrimSpace(name) == "" {
		return "-"
	}
	return name
}

func pathOrDash(path string) string {
	if strings.TrimSpace(path) == "" {
		return "-"
	}
	return shortenPath(path)
}

func layoutOrDash(layout string) string {
	if strings.TrimSpace(layout) == "" {
		return "-"
	}
	return layout
}

func emptyStateMessage() string {
	return strings.Join([]string{
		"No sessions found.",
		"",
		"Tips:",
		"  • Run 'peakypanes init' to create a global config",
		"  • Run 'peakypanes start' in a project directory",
		"  • Press 'o' to scan for git projects (set dashboard.project_roots)",
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

func tailLines(lines []string, max int) []string {
	if max <= 0 {
		return nil
	}
	if len(lines) <= max {
		return lines
	}
	return lines[len(lines)-max:]
}

func compactPreviewLines(lines []string) []string {
	if len(lines) == 0 {
		return nil
	}
	compact := make([]string, 0, len(lines))
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
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

func selectedWindow(session *SessionItem, windowIndex string) *WindowItem {
	if session == nil {
		return nil
	}
	for i := range session.Windows {
		if session.Windows[i].Index == windowIndex {
			return &session.Windows[i]
		}
	}
	if len(session.Windows) > 0 {
		return &session.Windows[0]
	}
	return nil
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

func (m Model) dividerLine(width int) string {
	if width <= 0 {
		return ""
	}
	return theme.ListDimmed.Render(strings.Repeat("─", width))
}
