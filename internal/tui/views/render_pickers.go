package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/regenrek/peakypanes/internal/tui/theme"
)

func (m Model) viewLayoutPicker() string {
	return m.renderDialogList(m.LayoutPicker.View(), m.LayoutPicker.Width(), m.LayoutPicker.Height(), "New Session Layout")
}

func (m Model) viewPaneSplitPicker() string {
	choices := renderDialogChoices([]dialogChoice{
		{Key: "r", Label: "split right"},
		{Key: "d", Label: "split down"},
		{Key: "esc", Label: "cancel"},
	})
	content := dialogContent(
		dialogTitleStyle.Render("➕ Add Pane"),
		theme.DialogNote.Render("Choose split direction"),
		choices,
	)
	return m.renderDialog(dialogSpec{Content: content, RequireViewport: true})
}

func (m Model) viewPaneSwapPicker() string {
	listW := m.PaneSwapPicker.Width()
	listH := m.PaneSwapPicker.Height()
	content := lipgloss.NewStyle().Width(listW).Height(listH).Render(m.PaneSwapPicker.View())
	return m.renderDialog(dialogSpec{
		Content:         content,
		Size:            dialogSizeForContent(listW, listH),
		RequireViewport: true,
	})
}

const commandPaletteHeading = "⌘ Command Palette"
const settingsMenuHeading = "Settings"
const performanceMenuHeading = "Performance"
const debugMenuHeading = "Debug"

func (m Model) viewCommandPalette() string {
	return m.renderDialogList(m.CommandPalette.View(), m.CommandPalette.Width(), m.CommandPalette.Height(), commandPaletteHeading)
}

func (m Model) viewSettingsMenu() string {
	return m.renderDialogList(m.SettingsMenu.View(), m.SettingsMenu.Width(), m.SettingsMenu.Height(), settingsMenuHeading)
}

func (m Model) viewPerformanceMenu() string {
	return m.renderDialogListWithHelp(
		m.PerformanceMenu.View(),
		m.PerformanceMenu.Width(),
		m.PerformanceMenu.Height(),
		performanceMenuHeading,
		m.DialogHelp,
	)
}

func (m Model) viewDebugMenu() string {
	return m.renderDialogList(m.DebugMenu.View(), m.DebugMenu.Width(), m.DebugMenu.Height(), debugMenuHeading)
}

func (m Model) renderDialogList(listView string, listW, listH int, heading string) string {
	header := theme.HelpTitle.Render(heading)
	headerW := lipgloss.Width(header)
	headerH := lipgloss.Height(header)
	body := lipgloss.NewStyle().
		Width(listW).
		Height(listH).
		Background(theme.Background).
		Render(listView)
	content := lipgloss.JoinVertical(lipgloss.Left, header, body)
	contentW := listW
	if headerW > contentW {
		contentW = headerW
	}
	contentH := listH + headerH
	return m.renderDialog(dialogSpec{
		Content:         content,
		Size:            dialogSizeForContentWithStyle(dialogStyleCompact, contentW, contentH),
		RequireViewport: true,
		Style:           dialogStyleCompact,
		UseStyle:        true,
	})
}

func (m Model) renderDialogListWithHelp(listView string, listW, listH int, heading string, help DialogHelp) string {
	header := theme.HelpTitle.Render(heading)
	headerW := lipgloss.Width(header)
	headerH := lipgloss.Height(header)
	body := lipgloss.NewStyle().
		Width(listW).
		Height(listH).
		Background(theme.Background).
		Render(listView)

	contentW := listW
	if headerW > contentW {
		contentW = headerW
	}
	contentH := listH + headerH

	if strings.TrimSpace(help.Line) != "" {
		helpLines := buildHelpLines(help, listW, 3)
		spacer := lipgloss.NewStyle().
			Width(listW).
			Background(theme.Background).
			Render("")
		helpBlocks := make([]string, 0, 2+len(helpLines))
		helpBlocks = append(helpBlocks, spacer)
		helpBlocks = append(helpBlocks, helpLines...)
		helpBlocks = append(helpBlocks, spacer)
		content := lipgloss.JoinVertical(lipgloss.Left, append([]string{header, body}, helpBlocks...)...)
		contentH += 2 + len(helpLines)
		dialog := m.renderDialog(dialogSpec{
			Content:         content,
			Size:            dialogSizeForContentWithStyle(dialogStyleCompact, contentW, contentH),
			RequireViewport: true,
			Style:           dialogStyleCompact,
			UseStyle:        true,
		})
		return dialog
	}

	content := lipgloss.JoinVertical(lipgloss.Left, header, body)
	return m.renderDialog(dialogSpec{
		Content:         content,
		Size:            dialogSizeForContentWithStyle(dialogStyleCompact, contentW, contentH),
		RequireViewport: true,
		Style:           dialogStyleCompact,
		UseStyle:        true,
	})
}

func buildHelpLines(help DialogHelp, listW int, maxLines int) []string {
	if maxLines < 1 {
		return nil
	}
	helpWidth := listW - 1
	if helpWidth < 1 {
		helpWidth = 1
	}
	text := strings.TrimSpace(help.Line)
	if help.Open && strings.TrimSpace(help.Body) != "" {
		text = help.Body
	}
	lines := wrapHelpText(text, helpWidth)
	lines = clampHelpLines(lines, helpWidth, maxLines)
	styled := make([]string, 0, maxLines)
	for _, line := range lines {
		styled = append(styled, lipgloss.NewStyle().
			Width(listW).
			Padding(0, 0, 0, 1).
			Background(theme.Background).
			Render(line))
	}
	for len(styled) < maxLines {
		styled = append(styled, lipgloss.NewStyle().
			Width(listW).
			Padding(0, 0, 0, 1).
			Background(theme.Background).
			Render(""))
	}
	return styled
}

func wrapHelpText(text string, width int) []string {
	if width < 1 {
		return []string{""}
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return []string{}
	}
	paragraphs := strings.Split(text, "\n")
	lines := make([]string, 0, len(paragraphs))
	for _, paragraph := range paragraphs {
		paragraph = strings.TrimSpace(paragraph)
		if paragraph == "" {
			lines = append(lines, "")
			continue
		}
		words := strings.Fields(paragraph)
		line := ""
		for _, word := range words {
			if line == "" {
				line = word
				continue
			}
			if lipgloss.Width(line)+1+lipgloss.Width(word) <= width {
				line += " " + word
				continue
			}
			lines = append(lines, truncateHelpLine(line, width))
			line = word
		}
		if line != "" {
			lines = append(lines, truncateHelpLine(line, width))
		}
	}
	return lines
}

func clampHelpLines(lines []string, width int, maxLines int) []string {
	if len(lines) == 0 {
		return make([]string, 0, maxLines)
	}
	if len(lines) <= maxLines {
		return lines
	}
	out := append([]string{}, lines[:maxLines]...)
	out[maxLines-1] = ansi.Truncate(strings.TrimSpace(out[maxLines-1]), width, "…")
	return out
}

func truncateHelpLine(line string, width int) string {
	if width < 1 {
		return ""
	}
	return ansi.Truncate(strings.TrimSpace(line), width, "")
}

func (m Model) viewHelp() string {
	var left strings.Builder
	left.WriteString("Navigation\n")
	left.WriteString(fmt.Sprintf("  %s Switch projects\n", m.Keys.ProjectKeys))
	left.WriteString(fmt.Sprintf("  %s Switch sessions/panes (project view)\n", m.Keys.SessionKeys))
	left.WriteString(fmt.Sprintf("  %s Switch sessions only (project view)\n", m.Keys.SessionOnlyKeys))
	left.WriteString(fmt.Sprintf("  %s Switch panes (project view)\n", m.Keys.PaneKeys))
	left.WriteString(fmt.Sprintf("  %s Switch panes (dashboard)\n", m.Keys.SessionKeys))
	left.WriteString(fmt.Sprintf("  %s Switch project column (dashboard)\n", m.Keys.PaneKeys))
	left.WriteString("\nProject\n")
	left.WriteString(fmt.Sprintf("  %s Open project picker\n", m.Keys.OpenProject))
	left.WriteString(fmt.Sprintf("  %s Close project\n", m.Keys.CloseProject))
	left.WriteString(fmt.Sprintf("  %s Toggle sidebar\n", m.Keys.ToggleSidebar))
	left.WriteString("\nSession\n")
	left.WriteString("  enter Attach/start session (when reply empty)\n")
	left.WriteString(fmt.Sprintf("  %s New session (pick layout)\n", m.Keys.NewSession))
	left.WriteString(fmt.Sprintf("  %s Close session\n", m.Keys.KillSession))
	left.WriteString("\nPane\n")
	left.WriteString("  type  Send input to selected pane (default)\n")
	left.WriteString(fmt.Sprintf("  %s Toggle last pane\n", m.Keys.ToggleLastPane))
	left.WriteString(fmt.Sprintf("  %s Focus action line\n", m.Keys.FocusAction))
	left.WriteString(fmt.Sprintf("  %s Toggle HARD RAW\n", m.Keys.HardRaw))
	if m.Keys.ResizeMode != "" {
		left.WriteString(fmt.Sprintf("  %s Resize mode\n", m.Keys.ResizeMode))
	}
	left.WriteString(fmt.Sprintf("  %s Scrollback mode (peky sessions)\n", m.Keys.Scrollback))
	left.WriteString(fmt.Sprintf("  %s Copy mode (peky sessions)\n", m.Keys.CopyMode))
	left.WriteString("  mouse Wheel scrollback (shift=1, ctrl=page)\n")
	left.WriteString("  mouse Drag select (HARD RAW)\n")

	var right strings.Builder
	right.WriteString("Pane List\n")
	right.WriteString(fmt.Sprintf("  %s Toggle pane list\n", m.Keys.TogglePanes))
	right.WriteString("\nOther\n")
	right.WriteString(fmt.Sprintf("  %s Refresh\n", m.Keys.Refresh))
	right.WriteString(fmt.Sprintf("  %s Edit config\n", m.Keys.EditConfig))
	right.WriteString(fmt.Sprintf("  %s Command palette\n", m.Keys.CommandPalette))
	right.WriteString(fmt.Sprintf("  %s Filter sessions\n", m.Keys.Filter))
	right.WriteString(fmt.Sprintf("  %s Close help\n", m.Keys.Help))
	right.WriteString(fmt.Sprintf("  %s Quit\n", m.Keys.Quit))

	colWidth := 36
	if m.Width > 0 {
		frameW, _ := dialogStyle.GetFrameSize()
		avail := m.Width - frameW - 6
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
	content.WriteString(theme.HelpTitle.Render("peky — Help"))
	content.WriteString("\n")
	content.WriteString(columns)

	return m.renderDialog(dialogSpec{Content: content.String()})
}
