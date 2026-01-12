package agent

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/regenrek/peakypanes/internal/identity"
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

// PaneState is a validated, TTL-checked snapshot of the agent state file.
type PaneState struct {
	PaneID     string
	Tool       string
	Status     Status
	UpdatedAt  time.Time
	RawState   string
	RawTool    string
	SourcePath string
}

func agentStateDir() string {
	if dir := strings.TrimSpace(os.Getenv("PEKY_AGENT_STATE_DIR")); dir != "" {
		return dir
	}
	runtimeDir := strings.TrimSpace(os.Getenv("XDG_RUNTIME_DIR"))
	if runtimeDir == "" {
		// Keep in sync with scripts/agent-state/* defaults.
		// On macOS, os.TempDir() is often not /tmp, which breaks cross-process coordination.
		runtimeDir = "/tmp"
	}
	return filepath.Join(runtimeDir, identity.AppSlug, "agent-state")
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
	state, ok := ReadPaneState(paneID, cfg, now)
	if !ok {
		return StatusIdle, false
	}
	return state.Status, true
}

// ReadPaneState reads the agent state file for the pane and returns the validated snapshot.
func ReadPaneState(paneID string, cfg DetectionConfig, now time.Time) (PaneState, bool) {
	if !cfg.Codex && !cfg.Claude {
		return PaneState{}, false
	}
	if strings.TrimSpace(paneID) == "" {
		return PaneState{}, false
	}
	raw, err := readAgentState(paneID)
	if err != nil {
		return PaneState{}, false
	}
	if raw.PaneID != "" && raw.PaneID != paneID {
		return PaneState{}, false
	}
	if !agentDetectionAllowed(raw.Tool, cfg) {
		return PaneState{}, false
	}
	updatedAt := raw.updatedAt()
	if updatedAt.IsZero() || now.Sub(updatedAt) > defaultAgentStateTTL {
		return PaneState{}, false
	}
	status, ok := agentStatusFromState(raw.State)
	if !ok {
		return PaneState{}, false
	}
	return PaneState{
		PaneID:     paneID,
		Tool:       strings.ToLower(strings.TrimSpace(raw.Tool)),
		Status:     status,
		UpdatedAt:  updatedAt,
		RawState:   raw.State,
		RawTool:    raw.Tool,
		SourcePath: agentStatePath(paneID),
	}, true
}
