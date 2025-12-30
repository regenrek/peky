package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/regenrek/peakypanes/internal/tui/theme"
)

func (m Model) viewLayoutPicker() string {
	listW := m.LayoutPicker.Width()
	listH := m.LayoutPicker.Height()
	content := lipgloss.NewStyle().Width(listW).Height(listH).Render(m.LayoutPicker.View())
	return m.renderDialog(dialogSpec{
		Content:         content,
		Size:            dialogSizeForContent(listW, listH),
		RequireViewport: true,
	})
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
const debugMenuHeading = "Debug"

func (m Model) viewCommandPalette() string {
	return m.renderDialogList(m.CommandPalette.View(), m.CommandPalette.Width(), m.CommandPalette.Height(), commandPaletteHeading)
}

func (m Model) viewSettingsMenu() string {
	return m.renderDialogList(m.SettingsMenu.View(), m.SettingsMenu.Width(), m.SettingsMenu.Height(), settingsMenuHeading)
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
	left.WriteString(fmt.Sprintf("  %s Kill session\n", m.Keys.KillSession))
	left.WriteString("\nPane\n")
	left.WriteString("  type  Send to panes (terminal focus off)\n")
	left.WriteString("  enter Send input\n")
	left.WriteString("  esc   Clear input\n")
	left.WriteString("  up/down Input history\n")
	left.WriteString("  /     Slash commands (↑/↓ select, tab complete)\n")
	left.WriteString(fmt.Sprintf("  %s Toggle terminal focus (Peaky Panes sessions)\n", m.Keys.TerminalFocus))
	left.WriteString(fmt.Sprintf("  %s Scrollback mode (Peaky Panes sessions)\n", m.Keys.Scrollback))
	left.WriteString(fmt.Sprintf("  %s Copy mode (Peaky Panes sessions)\n", m.Keys.CopyMode))
	left.WriteString("  mouse Wheel scrollback (shift=1, ctrl=page)\n")
	left.WriteString("  mouse Drag select (terminal focus)\n")
	left.WriteString("  type  Send input to focused pane\n")

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
	content.WriteString(theme.HelpTitle.Render("Peaky Panes — Help"))
	content.WriteString("\n")
	content.WriteString(columns)

	return m.renderDialog(dialogSpec{Content: content.String()})
}
