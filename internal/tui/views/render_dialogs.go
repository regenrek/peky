package views

import (
	"fmt"
	"strings"

	"github.com/regenrek/peakypanes/internal/tui/theme"
)

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
	return m.renderConfirmDialog("⚠️  Close Session?", body.String(), []dialogChoice{
		{Key: "y", Label: "confirm"},
		{Key: "n", Label: "cancel"},
	})
}

func (m Model) viewConfirmQuit() string {
	var body strings.Builder
	if m.ConfirmQuit.RunningPanes > 0 {
		body.WriteString(theme.DialogLabel.Render("Running panes: "))
		body.WriteString(theme.DialogValue.Render(fmt.Sprintf("%d", m.ConfirmQuit.RunningPanes)))
		body.WriteString("\n\n")
	}
	body.WriteString(theme.DialogNote.Render("Quit now? Sessions stay running unless you stop the daemon."))
	body.WriteString("\n")
	body.WriteString(theme.DialogNote.Render("Press k to stop the daemon and kill all panes."))
	return m.renderConfirmDialog("Quit PeakyPanes?", body.String(), []dialogChoice{
		{Key: "y", Label: "quit (keep sessions)"},
		{Key: "k", Label: "quit & stop daemon"},
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
		{Key: "k", Label: "close sessions"},
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
		{Key: "k", Label: "close sessions"},
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
	content := dialogContent(
		dialogTitleStyle.Render(title),
		body,
		renderDialogChoices(choices),
	)
	return m.renderDialog(dialogSpec{Content: content})
}

func (m Model) viewRename() string {
	title := "Rename Session"
	if m.Rename.IsPane {
		title = "Rename Pane"
	}

	var details strings.Builder
	if m.Rename.IsPane {
		if strings.TrimSpace(m.Rename.Session) != "" {
			details.WriteString(theme.DialogLabel.Render("Session: "))
			details.WriteString(theme.DialogValue.Render(m.Rename.Session))
			details.WriteString("\n")
		}
		paneLabel := strings.TrimSpace(m.Rename.Pane)
		if paneLabel == "" && strings.TrimSpace(m.Rename.PaneIndex) != "" {
			paneLabel = fmt.Sprintf("pane %s", strings.TrimSpace(m.Rename.PaneIndex))
		}
		if paneLabel != "" {
			details.WriteString(theme.DialogLabel.Render("Pane: "))
			details.WriteString(theme.DialogValue.Render(paneLabel))
			details.WriteString("\n")
		}
	} else if strings.TrimSpace(m.Rename.Session) != "" {
		details.WriteString(theme.DialogLabel.Render("Session: "))
		details.WriteString(theme.DialogValue.Render(m.Rename.Session))
		details.WriteString("\n")
	}

	inputWidth := 40
	if m.Width > 0 {
		inputWidth = clamp(m.Width-30, 20, 60)
	}
	m.Rename.Input.Width = inputWidth
	inputLine := theme.DialogLabel.Render("New name: ") + m.Rename.Input.View()
	choices := renderDialogChoices([]dialogChoice{
		{Key: "enter", Label: "confirm"},
		{Key: "esc", Label: "cancel"},
	})

	content := dialogContent(
		dialogTitleStyle.Render(title),
		strings.TrimRight(details.String(), "\n"),
		inputLine,
		choices,
	)
	return m.renderDialog(dialogSpec{Content: content})
}

func (m Model) viewProjectRootSetup() string {
	inputWidth := 60
	if m.Width > 0 {
		inputWidth = clamp(m.Width-30, 24, 80)
	}
	m.ProjectRootInput.Width = inputWidth
	note := theme.DialogNote.Render("Comma-separated list of folders to scan for git projects.")
	inputLine := theme.DialogLabel.Render("Roots: ") + m.ProjectRootInput.View()
	choices := renderDialogChoices([]dialogChoice{
		{Key: "enter", Label: "save"},
		{Key: "esc", Label: "cancel"},
	})

	content := dialogContent(
		dialogTitleStyle.Render("Project Roots"),
		note,
		inputLine,
		choices,
	)
	return m.renderDialog(dialogSpec{Content: content})
}

func (m Model) viewAuthDialog() string {
	inputWidth := 50
	if m.Width > 0 {
		inputWidth = clamp(m.Width-30, 24, 80)
	}
	m.AuthDialog.Input.Width = inputWidth
	title := strings.TrimSpace(m.AuthDialog.Title)
	if title == "" {
		title = "OAuth"
	}
	body := strings.TrimSpace(m.AuthDialog.Body)
	if body == "" {
		body = "Follow the instructions below to continue."
	}
	footer := strings.TrimSpace(m.AuthDialog.Footer)
	if footer == "" {
		footer = "enter confirm • esc cancel"
	}
	inputLine := theme.DialogLabel.Render("Paste code or token: ") + m.AuthDialog.Input.View()
	choices := renderDialogChoices([]dialogChoice{
		{Key: "enter", Label: "confirm"},
		{Key: "esc", Label: "cancel"},
	})
	content := dialogContent(
		dialogTitleStyle.Render(title),
		body,
		inputLine,
		choices,
		theme.DialogNote.Render(footer),
	)
	return m.renderDialog(dialogSpec{Content: content})
}
