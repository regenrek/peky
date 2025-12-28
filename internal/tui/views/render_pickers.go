package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/regenrek/peakypanes/internal/tui/theme"
)

func (m Model) viewLayoutPicker() string {
	if m.Width == 0 || m.Height == 0 {
		return ""
	}
	base := appStyle.Render(theme.ListDimmed.Render(m.viewDashboardContent()))
	listW := m.LayoutPicker.Width()
	listH := m.LayoutPicker.Height()
	frameW, frameH := dialogStyle.GetFrameSize()
	overlayW := listW + frameW
	overlayH := listH + frameH
	content := lipgloss.NewStyle().Width(listW).Height(listH).Render(m.LayoutPicker.View())
	dialog := dialogStyle.Width(overlayW).Height(overlayH).Render(content)
	return overlayCenteredSized(base, dialog, m.Width, m.Height, overlayW, overlayH)
}

func (m Model) viewPaneSplitPicker() string {
	if m.Width == 0 || m.Height == 0 {
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
	return overlayCentered(base, dialog, m.Width, m.Height)
}

func (m Model) viewPaneSwapPicker() string {
	if m.Width == 0 || m.Height == 0 {
		return ""
	}
	base := appStyle.Render(theme.ListDimmed.Render(m.viewDashboardContent()))
	listW := m.PaneSwapPicker.Width()
	listH := m.PaneSwapPicker.Height()
	frameW, frameH := dialogStyle.GetFrameSize()
	overlayW := listW + frameW
	overlayH := listH + frameH
	content := lipgloss.NewStyle().Width(listW).Height(listH).Render(m.PaneSwapPicker.View())
	dialog := dialogStyle.Width(overlayW).Height(overlayH).Render(content)
	return overlayCenteredSized(base, dialog, m.Width, m.Height, overlayW, overlayH)
}

func (m Model) viewCommandPalette() string {
	if m.Width == 0 || m.Height == 0 {
		return ""
	}
	base := appStyle.Render(theme.ListDimmed.Render(m.viewDashboardContent()))
	listW := m.CommandPalette.Width()
	listH := m.CommandPalette.Height()
	frameW, frameH := dialogStyle.GetFrameSize()
	overlayW := listW + frameW
	overlayH := listH + frameH
	content := lipgloss.NewStyle().Width(listW).Height(listH).Render(m.CommandPalette.View())
	dialog := dialogStyle.Width(overlayW).Height(overlayH).Render(content)
	return overlayCenteredSized(base, dialog, m.Width, m.Height, overlayW, overlayH)
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
	left.WriteString("\nSession\n")
	left.WriteString("  enter Attach/start session (when reply empty)\n")
	left.WriteString(fmt.Sprintf("  %s New session (pick layout)\n", m.Keys.NewSession))
	left.WriteString(fmt.Sprintf("  %s Kill session\n", m.Keys.KillSession))
	left.WriteString("\nPane\n")
	left.WriteString("  type  Quick reply (terminal focus off)\n")
	left.WriteString("  enter Send quick reply\n")
	left.WriteString("  esc   Clear quick reply\n")
	left.WriteString(fmt.Sprintf("  %s Toggle terminal focus (Peaky Panes sessions)\n", m.Keys.TerminalFocus))
	left.WriteString(fmt.Sprintf("  %s Scrollback mode (Peaky Panes sessions)\n", m.Keys.Scrollback))
	left.WriteString(fmt.Sprintf("  %s Copy mode (Peaky Panes sessions)\n", m.Keys.CopyMode))
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

	dialog := dialogStyle.Render(content.String())
	return m.overlayDialog(dialog)
}
