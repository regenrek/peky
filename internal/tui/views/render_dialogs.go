package views

import (
	"fmt"
	"strings"

	"github.com/regenrek/peakypanes/internal/tui/theme"
)

type dialogChoice struct {
	Key   string
	Label string
}

func (m Model) viewConfirmKill() string {
	var body strings.Builder
	if m.ConfirmKill.Session != "" {
		body.WriteString(theme.DialogLabel.Render("Session: "))
		body.WriteString(theme.DialogValue.Render(m.ConfirmKill.Session))
		body.WriteString("\n")
		if m.ConfirmKill.Project != "" {
			body.WriteString(theme.DialogLabel.Render("Project: "))
			body.WriteString(theme.DialogValue.Render(m.ConfirmKill.Project))
			body.WriteString("\n")
		}
		body.WriteString("\n")
	}
	body.WriteString(theme.DialogNote.Render("Kill the session: This won't delete your project"))
	return m.renderConfirmDialog("⚠️  Kill Session?", body.String(), []dialogChoice{
		{Key: "y", Label: "confirm"},
		{Key: "n", Label: "cancel"},
	})
}

func (m Model) viewConfirmCloseProject() string {
	var body strings.Builder
	if m.ConfirmCloseProject.Project != "" {
		body.WriteString(theme.DialogLabel.Render("Project: "))
		body.WriteString(theme.DialogValue.Render(m.ConfirmCloseProject.Project))
		body.WriteString("\n")
		body.WriteString(theme.DialogLabel.Render("Running sessions: "))
		body.WriteString(theme.DialogValue.Render(fmt.Sprintf("%d", m.ConfirmCloseProject.RunningSessions)))
		body.WriteString("\n\n")
	}
	body.WriteString(theme.DialogNote.Render("Close hides the project from tabs; sessions stay running."))
	body.WriteString("\n")
	body.WriteString(theme.DialogNote.Render("Press k to kill running sessions instead."))
	return m.renderConfirmDialog("⚠️  Close Project?", body.String(), []dialogChoice{
		{Key: "y", Label: "close"},
		{Key: "k", Label: "kill sessions"},
		{Key: "n", Label: "cancel"},
	})
}

func (m Model) viewConfirmCloseAllProjects() string {
	var body strings.Builder
	body.WriteString(theme.DialogLabel.Render("Projects: "))
	body.WriteString(theme.DialogValue.Render(fmt.Sprintf("%d", m.ConfirmCloseAllProjects.ProjectCount)))
	body.WriteString("\n")
	body.WriteString(theme.DialogLabel.Render("Running sessions: "))
	body.WriteString(theme.DialogValue.Render(fmt.Sprintf("%d", m.ConfirmCloseAllProjects.RunningSessions)))
	body.WriteString("\n\n")
	body.WriteString(theme.DialogNote.Render("Close hides projects from tabs; sessions stay running."))
	body.WriteString("\n")
	body.WriteString(theme.DialogNote.Render("Press k to kill running sessions instead."))
	return m.renderConfirmDialog("⚠️  Close All Projects?", body.String(), []dialogChoice{
		{Key: "y", Label: "close all"},
		{Key: "k", Label: "kill sessions"},
		{Key: "n", Label: "cancel"},
	})
}

func (m Model) viewConfirmClosePane() string {
	var body strings.Builder
	if m.ConfirmClosePane.Title != "" {
		body.WriteString(theme.DialogLabel.Render("Pane: "))
		body.WriteString(theme.DialogValue.Render(m.ConfirmClosePane.Title))
		body.WriteString("\n")
	}
	if m.ConfirmClosePane.Session != "" {
		body.WriteString(theme.DialogLabel.Render("Session: "))
		body.WriteString(theme.DialogValue.Render(m.ConfirmClosePane.Session))
		body.WriteString("\n")
	}
	body.WriteString("\n")
	if m.ConfirmClosePane.Running {
		body.WriteString(theme.DialogNote.Render("The pane is still running. Closing it will stop the process."))
	}
	return m.renderConfirmDialog("⚠️  Close Pane?", body.String(), []dialogChoice{
		{Key: "y", Label: "close"},
		{Key: "n", Label: "cancel"},
	})
}

func (m Model) viewConfirmRestart() string {
	body := theme.DialogNote.Render("Restarting will disconnect clients. Sessions will be restored on startup.")
	return m.renderConfirmDialog("Restart Daemon?", body, []dialogChoice{
		{Key: "y", Label: "restart"},
		{Key: "n", Label: "cancel"},
	})
}

func (m Model) renderConfirmDialog(title, body string, choices []dialogChoice) string {
	var dialogContent strings.Builder
	dialogContent.WriteString(dialogTitleStyle.Render(title))
	body = strings.TrimRight(body, "\n")
	if body != "" {
		dialogContent.WriteString("\n\n")
		dialogContent.WriteString(body)
	}
	if choicesLine := renderDialogChoices(choices); choicesLine != "" {
		dialogContent.WriteString("\n\n")
		dialogContent.WriteString(choicesLine)
	}
	dialog := dialogStyle.Render(dialogContent.String())
	return m.overlayDialog(dialog)
}

func renderDialogChoices(choices []dialogChoice) string {
	if len(choices) == 0 {
		return ""
	}
	var builder strings.Builder
	for i, choice := range choices {
		if key := strings.TrimSpace(choice.Key); key != "" {
			builder.WriteString(theme.DialogChoiceKey.Render(key))
		}
		if label := strings.TrimSpace(choice.Label); label != "" {
			prefix := " "
			if strings.TrimSpace(choice.Key) == "" {
				prefix = ""
			}
			builder.WriteString(theme.DialogChoiceSep.Render(prefix + label))
		}
		if i < len(choices)-1 {
			builder.WriteString(theme.DialogChoiceSep.Render(" • "))
		}
	}
	return builder.String()
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
