package agent

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

// Status represents an agent state classification.
type Status int

const (
	StatusIdle Status = iota
	StatusRunning
	StatusDone
	StatusError
)

// DetectionConfig controls which tools are eligible for detection.
type DetectionConfig struct {
	Codex  bool
	Claude bool
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

func agentStatusFromState(state string) (Status, bool) {
	switch strings.ToLower(strings.TrimSpace(state)) {
	case "running", "in_progress", "in-progress":
		return StatusRunning, true
	case "idle", "waiting", "paused":
		return StatusIdle, true
	case "done", "completed", "success":
		return StatusDone, true
	case "error", "failed", "failure":
		return StatusError, true
	default:
		return StatusIdle, false
	}
}

func agentDetectionAllowed(tool string, cfg DetectionConfig) bool {
	switch strings.ToLower(strings.TrimSpace(tool)) {
	case agentToolCodex:
		return cfg.Codex
	case agentToolClaude:
		return cfg.Claude
	default:
		return false
	}
}

// ClassifyState reads the agent state file for the pane and classifies it.
func ClassifyState(paneID string, cfg DetectionConfig, now time.Time) (Status, bool) {
	if !cfg.Codex && !cfg.Claude {
		return StatusIdle, false
	}
	if strings.TrimSpace(paneID) == "" {
		return StatusIdle, false
	}
	state, err := readAgentState(paneID)
	if err != nil {
		return StatusIdle, false
	}
	if state.PaneID != "" && state.PaneID != paneID {
		return StatusIdle, false
	}
	if !agentDetectionAllowed(state.Tool, cfg) {
		return StatusIdle, false
	}
	updatedAt := state.updatedAt()
	if updatedAt.IsZero() || now.Sub(updatedAt) > defaultAgentStateTTL {
		return StatusIdle, false
	}
	status, ok := agentStatusFromState(state.State)
	if !ok {
		return StatusIdle, false
	}
	return status, true
}
