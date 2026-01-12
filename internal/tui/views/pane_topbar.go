package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/regenrek/peakypanes/internal/tui/icons"
	"github.com/regenrek/peakypanes/internal/tui/theme"
)

func renderPaneTopbar(pane Pane, width int, spinner string) string {
	if width <= 0 {
		return ""
	}
	if pane.ID == "" {
		return fitLine("", width)
	}

	left := strings.TrimSpace(pane.Cwd)
	if left == "" {
		left = "-"
	}

	suffix := paneTopbarSuffix(pane, spinner)
	if suffix == "" {
		return fitLine(truncateTileLine(left, width), width)
	}
	suffixW := lipgloss.Width(suffix)
	if suffixW >= width {
		return fitLine(truncateTileLine(suffix, width), width)
	}
	leftW := width - suffixW - 1
	if leftW <= 0 {
		return fitLine(truncateTileLine(suffix, width), width)
	}
	leftTrunc := truncateTileLine(left, leftW)
	leftFit := fitLine(leftTrunc, leftW)
	return leftFit + " " + suffix
}

func paneTopbarSuffix(pane Pane, spinner string) string {
	parts := []string{}

	if git := paneTopbarGit(pane); git != "" {
		parts = append(parts, git)
	}
	if agent := paneTopbarAgent(pane, spinner); agent != "" {
		parts = append(parts, agent)
	}
	if len(parts) == 0 {
		return ""
	}
	sep := theme.ListDimmed.Render("│")
	return strings.Join(parts, " "+sep+" ")
}

func paneTopbarGit(pane Pane) string {
	branch := strings.TrimSpace(pane.GitBranch)
	if branch == "" {
		return ""
	}
	if pane.GitDirty {
		branch += "*"
	}
	text := fmt.Sprintf("⎇ %s", branch)
	if pane.GitWorktree {
		text += " ⧉"
	}
	return text
}

func paneTopbarAgent(pane Pane, spinner string) string {
	tool := strings.TrimSpace(pane.Tool)
	if !isAgentTool(tool) {
		return ""
	}
	icon := "AI"
	if pane.AgentState == "running" {
		if strings.TrimSpace(spinner) == "" {
			spinner = icons.Active().Spinner[0]
		}
		return fmt.Sprintf("%s %s %s", icon, tool, spinner)
	}
	if pane.AgentUnread {
		dot := icons.Active().PaneDot.BySize(icons.ActiveSize())
		return fmt.Sprintf("%s %s %s", icon, tool, theme.StatusMessage.Render(dot))
	}
	return theme.ListDimmed.Render(fmt.Sprintf("%s %s", icon, tool))
}

func isAgentTool(tool string) bool {
	switch strings.ToLower(strings.TrimSpace(tool)) {
	case "codex", "claude", "opencode", "pi":
		return true
	default:
		return false
	}
}
