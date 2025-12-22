package peakypanes

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestAgentStatePathAndRead(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PEAKYPANES_AGENT_STATE_DIR", dir)

	path := agentStatePath("pane-1")
	if path != filepath.Join(dir, "pane-1.json") {
		t.Fatalf("agentStatePath() = %q", path)
	}

	state := agentState{State: "running", Tool: "codex", UpdatedAtUnixMS: time.Now().UnixMilli(), PaneID: "pane-1"}
	data, err := json.Marshal(state)
	if err != nil {
		t.Fatalf("Marshal() error: %v", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("WriteFile() error: %v", err)
	}

	loaded, err := readAgentState("pane-1")
	if err != nil {
		t.Fatalf("readAgentState() error: %v", err)
	}
	if loaded.State != "running" || loaded.Tool != "codex" || loaded.PaneID != "pane-1" {
		t.Fatalf("readAgentState() = %#v", loaded)
	}
}

func TestAgentStatusFromState(t *testing.T) {
	cases := map[string]PaneStatus{
		"running":   PaneStatusRunning,
		"idle":      PaneStatusIdle,
		"done":      PaneStatusDone,
		"error":     PaneStatusError,
		"completed": PaneStatusDone,
	}
	for input, want := range cases {
		got, ok := agentStatusFromState(input)
		if !ok || got != want {
			t.Fatalf("agentStatusFromState(%q) = %v,%v", input, got, ok)
		}
	}
	if _, ok := agentStatusFromState("unknown"); ok {
		t.Fatalf("agentStatusFromState(unknown) should be false")
	}
}

func TestAgentDetectionAllowed(t *testing.T) {
	cfg := AgentDetectionConfig{Codex: true, Claude: false}
	if !agentDetectionAllowed("codex", cfg) {
		t.Fatalf("agentDetectionAllowed(codex) should be true")
	}
	if agentDetectionAllowed("claude", cfg) {
		t.Fatalf("agentDetectionAllowed(claude) should be false")
	}
}

func TestClassifyAgentState(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PEAKYPANES_AGENT_STATE_DIR", dir)

	paneID := "pane-9"
	state := agentState{State: "running", Tool: "codex", UpdatedAtUnixMS: time.Now().UnixMilli(), PaneID: paneID}
	data, err := json.Marshal(state)
	if err != nil {
		t.Fatalf("Marshal() error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, paneID+".json"), data, 0o644); err != nil {
		t.Fatalf("WriteFile() error: %v", err)
	}

	cfg := DashboardConfig{AgentDetection: AgentDetectionConfig{Codex: true}}
	status, ok := classifyAgentState(PaneItem{ID: paneID}, cfg, time.Now())
	if !ok || status != PaneStatusRunning {
		t.Fatalf("classifyAgentState() = %v,%v", status, ok)
	}

	stale := agentState{State: "running", Tool: "codex", UpdatedAtUnixMS: time.Now().Add(-time.Hour).UnixMilli(), PaneID: paneID}
	staleData, _ := json.Marshal(stale)
	if err := os.WriteFile(filepath.Join(dir, paneID+".json"), staleData, 0o644); err != nil {
		t.Fatalf("WriteFile() error: %v", err)
	}
	_, ok = classifyAgentState(PaneItem{ID: paneID}, cfg, time.Now())
	if ok {
		t.Fatalf("classifyAgentState() should be false for stale state")
	}
}
