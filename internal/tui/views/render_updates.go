package views

import (
	"fmt"
	"strings"

	"github.com/regenrek/peakypanes/internal/tui/theme"
)

func (m Model) viewUpdateDialog() string {
	dlg := m.UpdateDialog
	body := strings.Builder{}
	if dlg.CurrentVersion != "" {
		body.WriteString(theme.DialogLabel.Render("Current: "))
		body.WriteString(theme.DialogValue.Render(fmt.Sprintf("v%s", dlg.CurrentVersion)))
		body.WriteString("\n")
	}
	if dlg.LatestVersion != "" {
		body.WriteString(theme.DialogLabel.Render("Latest:  "))
		body.WriteString(theme.DialogValue.Render(fmt.Sprintf("v%s", dlg.LatestVersion)))
		body.WriteString("\n")
	}
	if strings.TrimSpace(dlg.Channel) != "" {
		body.WriteString(theme.DialogLabel.Render("Channel: "))
		body.WriteString(theme.DialogValue.Render(formatUpdateChannel(dlg.Channel)))
		body.WriteString("\n")
	}
	if dlg.Command != "" && dlg.Command != "Update manually" {
		body.WriteString(theme.DialogNote.Render("Command: "))
		body.WriteString(theme.DialogValue.Render(dlg.Command))
		body.WriteString("\n")
	}
	if !dlg.CanInstall {
		body.WriteString(theme.DialogNote.Render("Automatic updates are unavailable for this install."))
		body.WriteString("\n")
	}
	choices := []dialogChoice{
		{Key: "l", Label: "Later"},
		{Key: "s", Label: "Skip this version"},
		{Key: "esc", Label: "Close"},
	}
	if dlg.CanInstall {
		choices = append([]dialogChoice{{Key: "i", Label: "Install now"}}, choices...)
	}
	content := dialogContent(
		dialogTitleStyle.Render("Update Available"),
		body.String(),
		renderDialogChoices(choices),
	)
	return m.renderDialog(dialogSpec{Content: content})
}

func (m Model) viewUpdateProgress() string {
	progress := m.UpdateProgress
	bar := renderProgressBar(28, progress.Percent)
	body := strings.Builder{}
	if progress.Step != "" {
		body.WriteString(theme.DialogLabel.Render("Step: "))
		body.WriteString(theme.DialogValue.Render(progress.Step))
		body.WriteString("\n")
	}
	body.WriteString(theme.DialogValue.Render(bar))
	body.WriteString("\n")
	body.WriteString(theme.DialogNote.Render(fmt.Sprintf("%d%%", clampPercent(progress.Percent))))
	content := dialogContent(
		dialogTitleStyle.Render("Installing Update"),
		body.String(),
	)
	return m.renderDialog(dialogSpec{Content: content})
}

func (m Model) viewUpdateRestart() string {
	body := theme.DialogNote.Render("Update installed. Restart to finish.")
	choices := renderDialogChoices([]dialogChoice{
		{Key: "r", Label: "Restart now"},
		{Key: "esc", Label: "Later"},
	})
	content := dialogContent(
		dialogTitleStyle.Render("Restart Required"),
		body,
		choices,
	)
	return m.renderDialog(dialogSpec{Content: content})
}

func formatUpdateChannel(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "-"
	}
	switch trimmed {
	case "npm_global":
		return "npm (global)"
	case "npm_local":
		return "npm (local)"
	case "homebrew":
		return "homebrew"
	case "git":
		return "git"
	default:
		return trimmed
	}
}

func renderProgressBar(width int, percent int) string {
	if width < 10 {
		width = 10
	}
	p := clampPercent(percent)
	fill := width * p / 100
	if fill < 0 {
		fill = 0
	}
	if fill > width {
		fill = width
	}
	return "[" + strings.Repeat("=", fill) + strings.Repeat("-", width-fill) + "]"
}

func clampPercent(value int) int {
	if value < 0 {
		return 0
	}
	if value > 100 {
		return 100
	}
	return value
}
