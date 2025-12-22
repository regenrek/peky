package peakypanes

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const defaultAgentStateTTL = 30 * time.Second

const (
	agentToolCodex  = "codex"
	agentToolClaude = "claude"
)

type agentState struct {
	State           string `json:"state"`
	Tool            string `json:"tool"`
	UpdatedAtUnixMS int64  `json:"updated_at_unix_ms"`
	PaneID          string `json:"pane_id"`
}

func agentStateDir() string {
	if dir := strings.TrimSpace(os.Getenv("PEAKYPANES_AGENT_STATE_DIR")); dir != "" {
		return dir
	}
	runtimeDir := strings.TrimSpace(os.Getenv("XDG_RUNTIME_DIR"))
	if runtimeDir == "" {
		runtimeDir = os.TempDir()
	}
	return filepath.Join(runtimeDir, "peakypanes", "agent-state")
}

func agentStatePath(paneID string) string {
	paneID = strings.TrimSpace(paneID)
	if paneID == "" {
		return ""
	}
	return filepath.Join(agentStateDir(), paneID+".json")
}

func readAgentState(paneID string) (agentState, error) {
	var state agentState
	path := agentStatePath(paneID)
	if path == "" {
		return state, errors.New("empty pane id")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return state, err
	}
	if err := json.Unmarshal(data, &state); err != nil {
		return state, err
	}
	return state, nil
}

func (s agentState) updatedAt() time.Time {
	if s.UpdatedAtUnixMS <= 0 {
		return time.Time{}
	}
	return time.Unix(0, s.UpdatedAtUnixMS*int64(time.Millisecond))
}

func agentStatusFromState(state string) (PaneStatus, bool) {
	switch strings.ToLower(strings.TrimSpace(state)) {
	case "running", "in_progress", "in-progress":
		return PaneStatusRunning, true
	case "idle", "waiting", "paused":
		return PaneStatusIdle, true
	case "done", "completed", "success":
		return PaneStatusDone, true
	case "error", "failed", "failure":
		return PaneStatusError, true
	default:
		return PaneStatusIdle, false
	}
}

func agentDetectionAllowed(tool string, cfg AgentDetectionConfig) bool {
	switch strings.ToLower(strings.TrimSpace(tool)) {
	case agentToolCodex:
		return cfg.Codex
	case agentToolClaude:
		return cfg.Claude
	default:
		return false
	}
}

func classifyAgentState(pane PaneItem, cfg DashboardConfig, now time.Time) (PaneStatus, bool) {
	if !cfg.AgentDetection.Codex && !cfg.AgentDetection.Claude {
		return PaneStatusIdle, false
	}
	if pane.ID == "" {
		return PaneStatusIdle, false
	}
	state, err := readAgentState(pane.ID)
	if err != nil {
		return PaneStatusIdle, false
	}
	if state.PaneID != "" && state.PaneID != pane.ID {
		return PaneStatusIdle, false
	}
	if !agentDetectionAllowed(state.Tool, cfg.AgentDetection) {
		return PaneStatusIdle, false
	}
	updatedAt := state.updatedAt()
	if updatedAt.IsZero() || now.Sub(updatedAt) > defaultAgentStateTTL {
		return PaneStatusIdle, false
	}
	status, ok := agentStatusFromState(state.State)
	if !ok {
		return PaneStatusIdle, false
	}
	return status, true
}
