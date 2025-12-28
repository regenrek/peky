package views

import (
	"fmt"
	"strings"

	"github.com/regenrek/peakypanes/internal/tui/theme"
)

func (m Model) viewConfirmKill() string {
	var dialogContent strings.Builder

	dialogContent.WriteString(dialogTitleStyle.Render("⚠️  Kill Session?"))
	dialogContent.WriteString("\n\n")
	if m.ConfirmKill.Session != "" {
		dialogContent.WriteString(theme.DialogLabel.Render("Session: "))
		dialogContent.WriteString(theme.DialogValue.Render(m.ConfirmKill.Session))
		dialogContent.WriteString("\n")
		if m.ConfirmKill.Project != "" {
			dialogContent.WriteString(theme.DialogLabel.Render("Project: "))
			dialogContent.WriteString(theme.DialogValue.Render(m.ConfirmKill.Project))
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
	if m.ConfirmCloseProject.Project != "" {
		dialogContent.WriteString(theme.DialogLabel.Render("Project: "))
		dialogContent.WriteString(theme.DialogValue.Render(m.ConfirmCloseProject.Project))
		dialogContent.WriteString("\n")
		dialogContent.WriteString(theme.DialogLabel.Render("Running sessions: "))
		dialogContent.WriteString(theme.DialogValue.Render(fmt.Sprintf("%d", m.ConfirmCloseProject.RunningSessions)))
		dialogContent.WriteString("\n")
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
	if m.ConfirmClosePane.Title != "" {
		dialogContent.WriteString(theme.DialogLabel.Render("Pane: "))
		dialogContent.WriteString(theme.DialogValue.Render(m.ConfirmClosePane.Title))
		dialogContent.WriteString("\n")
	}
	if m.ConfirmClosePane.Session != "" {
		dialogContent.WriteString(theme.DialogLabel.Render("Session: "))
		dialogContent.WriteString(theme.DialogValue.Render(m.ConfirmClosePane.Session))
		dialogContent.WriteString("\n")
	}
	dialogContent.WriteString("\n")

	if m.ConfirmClosePane.Running {
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
	if m.Rename.IsPane {
		title = "Rename Pane"
	}
	dialogContent.WriteString(dialogTitleStyle.Render(title))
	dialogContent.WriteString("\n\n")

	if m.Rename.IsPane {
		if strings.TrimSpace(m.Rename.Session) != "" {
			dialogContent.WriteString(theme.DialogLabel.Render("Session: "))
			dialogContent.WriteString(theme.DialogValue.Render(m.Rename.Session))
			dialogContent.WriteString("\n")
		}
		paneLabel := strings.TrimSpace(m.Rename.Pane)
		if paneLabel == "" && strings.TrimSpace(m.Rename.PaneIndex) != "" {
			paneLabel = fmt.Sprintf("pane %s", strings.TrimSpace(m.Rename.PaneIndex))
		}
		if paneLabel != "" {
			dialogContent.WriteString(theme.DialogLabel.Render("Pane: "))
			dialogContent.WriteString(theme.DialogValue.Render(paneLabel))
			dialogContent.WriteString("\n")
		}
		dialogContent.WriteString("\n")
	} else if strings.TrimSpace(m.Rename.Session) != "" {
		dialogContent.WriteString(theme.DialogLabel.Render("Session: "))
		dialogContent.WriteString(theme.DialogValue.Render(m.Rename.Session))
		dialogContent.WriteString("\n\n")
	}

	inputWidth := 40
	if m.Width > 0 {
		inputWidth = clamp(m.Width-30, 20, 60)
	}
	m.Rename.Input.Width = inputWidth
	dialogContent.WriteString(theme.DialogLabel.Render("New name: "))
	dialogContent.WriteString(m.Rename.Input.View())
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
	if m.Width > 0 {
		inputWidth = clamp(m.Width-30, 24, 80)
	}
	m.ProjectRootInput.Width = inputWidth
	dialogContent.WriteString(theme.DialogLabel.Render("Roots: "))
	dialogContent.WriteString(m.ProjectRootInput.View())
	dialogContent.WriteString("\n\n")

	dialogContent.WriteString(theme.DialogChoiceKey.Render("enter"))
	dialogContent.WriteString(theme.DialogChoiceSep.Render(" save • "))
	dialogContent.WriteString(theme.DialogChoiceKey.Render("esc"))
	dialogContent.WriteString(theme.DialogChoiceSep.Render(" cancel"))

	dialog := dialogStyle.Render(dialogContent.String())
	return m.overlayDialog(dialog)
}

func (m Model) overlayDialog(dialog string) string {
	if m.Width == 0 || m.Height == 0 {
		return appStyle.Render(dialog)
	}
	base := appStyle.Render(theme.ListDimmed.Render(m.viewDashboardContent()))
	return overlayCentered(base, dialog, m.Width, m.Height)
}
