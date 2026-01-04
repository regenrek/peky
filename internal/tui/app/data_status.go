package app

import (
	"fmt"
	"strings"
	"time"

	"github.com/regenrek/peakypanes/internal/tui/agent"
	"github.com/regenrek/peakypanes/internal/tui/ansi"
)

func paneStatusFromAgent(status agent.Status) PaneStatus {
	switch status {
	case agent.StatusRunning:
		return PaneStatusRunning
	case agent.StatusDone:
		return PaneStatusDone
	case agent.StatusError:
		return PaneStatusError
	case agent.StatusIdle:
		return PaneStatusIdle
	default:
		return PaneStatusIdle
	}
}

func classifyAgentStatus(paneID string, settings DashboardConfig, now time.Time) (PaneStatus, bool) {
	cfg := agent.DetectionConfig{
		Codex:  settings.AgentDetection.Codex,
		Claude: settings.AgentDetection.Claude,
	}
	status, ok := agent.ClassifyState(paneID, cfg, now)
	if !ok {
		return PaneStatusIdle, false
	}
	return paneStatusFromAgent(status), true
}

func classifyPane(pane PaneItem, lines []string, settings DashboardConfig, now time.Time) PaneStatus {
	if pane.Disconnected {
		return PaneStatusDisconnected
	}
	if status, ok := paneStatusForDead(pane); ok {
		return status
	}
	if pane.RestoreFailed {
		return PaneStatusError
	}
	if status, ok := classifyPaneFromAgent(pane, lines, settings, now); ok {
		return status
	}
	if status, ok := classifyPaneFromLines(lines, settings.StatusMatcher); ok {
		return status
	}
	if paneIdle(pane, settings, now) {
		return PaneStatusIdle
	}
	return PaneStatusRunning
}

func paneStatusForDead(pane PaneItem) (PaneStatus, bool) {
	if !pane.Dead {
		return PaneStatusIdle, false
	}
	if pane.DeadStatus != 0 {
		return PaneStatusError, true
	}
	return PaneStatusDone, true
}

func classifyPaneFromAgent(pane PaneItem, lines []string, settings DashboardConfig, now time.Time) (PaneStatus, bool) {
	status, ok := classifyAgentStatus(pane.ID, settings, now)
	if !ok {
		return PaneStatusIdle, false
	}
	if status != PaneStatusIdle {
		return status, true
	}
	if matchesRunning(lines, settings.StatusMatcher) {
		return PaneStatusRunning, true
	}
	return status, true
}

func classifyPaneFromLines(lines []string, matcher statusMatcher) (PaneStatus, bool) {
	joined := stripJoinedLines(lines)
	if joined == "" {
		return PaneStatusIdle, false
	}
	if matcher.error != nil && matcher.error.MatchString(joined) {
		return PaneStatusError, true
	}
	if matcher.success != nil && matcher.success.MatchString(joined) {
		return PaneStatusDone, true
	}
	if matcher.running != nil && matcher.running.MatchString(joined) {
		return PaneStatusRunning, true
	}
	return PaneStatusIdle, false
}

func matchesRunning(lines []string, matcher statusMatcher) bool {
	joined := stripJoinedLines(lines)
	if joined == "" || matcher.running == nil {
		return false
	}
	return matcher.running.MatchString(joined)
}

func stripJoinedLines(lines []string) string {
	return ansi.Strip(strings.Join(lines, "\n"))
}

func paneIdle(pane PaneItem, settings DashboardConfig, now time.Time) bool {
	return !pane.LastActive.IsZero() && now.Sub(pane.LastActive) > settings.IdleThreshold
}

func paneSummaryLine(pane PaneItem, maxPreview int) string {
	preview := pane.Preview
	if maxPreview > 0 && len(preview) > maxPreview {
		preview = preview[len(preview)-maxPreview:]
	}
	if line := ansi.LastNonEmpty(preview); line != "" {
		return line
	}
	if line := strings.TrimSpace(pane.Title); line != "" {
		return line
	}
	if line := strings.TrimSpace(pane.Command); line != "" {
		return line
	}
	if strings.TrimSpace(pane.Index) != "" {
		return fmt.Sprintf("pane %s", strings.TrimSpace(pane.Index))
	}
	return ""
}
