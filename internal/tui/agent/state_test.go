package agent

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

func TestAgentStatePathEmpty(t *testing.T) {
	t.Setenv("PEAKYPANES_AGENT_STATE_DIR", "")
	if path := agentStatePath(""); path != "" {
		t.Fatalf("expected empty path, got %q", path)
	}
	if _, err := readAgentState(""); err == nil {
		t.Fatalf("expected error for empty pane id")
	}
}

func TestAgentStateDirFallback(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PEAKYPANES_AGENT_STATE_DIR", "")
	t.Setenv("XDG_RUNTIME_DIR", dir)
	got := agentStateDir()
	if got != filepath.Join(dir, "peakypanes", "agent-state") {
		t.Fatalf("agentStateDir() = %q", got)
	}
}

func TestAgentStateUpdatedAt(t *testing.T) {
	state := agentState{UpdatedAtUnixMS: 0}
	if !state.updatedAt().IsZero() {
		t.Fatalf("expected zero time for unset updated_at")
	}
	now := time.Now()
	state = agentState{UpdatedAtUnixMS: now.UnixMilli()}
	if state.updatedAt().IsZero() {
		t.Fatalf("expected non-zero time")
	}
}

func TestAgentStatusFromState(t *testing.T) {
	cases := map[string]Status{
		"running":   StatusRunning,
		"idle":      StatusIdle,
		"done":      StatusDone,
		"error":     StatusError,
		"completed": StatusDone,
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
	cfg := DetectionConfig{Codex: true, Claude: false}
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
	now := time.Now()
	state := agentState{State: "running", Tool: "codex", UpdatedAtUnixMS: now.UnixMilli(), PaneID: paneID}
	data, err := json.Marshal(state)
	if err != nil {
		t.Fatalf("Marshal() error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, paneID+".json"), data, 0o644); err != nil {
		t.Fatalf("WriteFile() error: %v", err)
	}

	cfg := DetectionConfig{Codex: true}
	status, ok := ClassifyState(paneID, cfg, now)
	if !ok || status != StatusRunning {
		t.Fatalf("ClassifyState() = %v,%v", status, ok)
	}

	stale := agentState{State: "running", Tool: "codex", UpdatedAtUnixMS: now.Add(-time.Hour).UnixMilli(), PaneID: paneID}
	staleData, _ := json.Marshal(stale)
	if err := os.WriteFile(filepath.Join(dir, paneID+".json"), staleData, 0o644); err != nil {
		t.Fatalf("WriteFile() error: %v", err)
	}
	_, ok = ClassifyState(paneID, cfg, now)
	if ok {
		t.Fatalf("ClassifyState() should be false for stale state")
	}
}

func TestClassifyAgentStateIgnoresMismatches(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PEAKYPANES_AGENT_STATE_DIR", dir)

	paneID := "pane-42"
	state := agentState{State: "running", Tool: "codex", UpdatedAtUnixMS: time.Now().UnixMilli(), PaneID: "other-pane"}
	data, err := json.Marshal(state)
	if err != nil {
		t.Fatalf("Marshal() error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, paneID+".json"), data, 0o644); err != nil {
		t.Fatalf("WriteFile() error: %v", err)
	}

	cfg := DetectionConfig{Codex: true}
	if _, ok := ClassifyState(paneID, cfg, time.Now()); ok {
		t.Fatalf("ClassifyState() should ignore mismatched pane_id")
	}
}

func TestClassifyAgentStateDisabledOrUnknown(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PEAKYPANES_AGENT_STATE_DIR", dir)

	paneID := "pane-100"
	state := agentState{State: "running", Tool: "unknown", UpdatedAtUnixMS: time.Now().UnixMilli(), PaneID: paneID}
	data, err := json.Marshal(state)
	if err != nil {
		t.Fatalf("Marshal() error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, paneID+".json"), data, 0o644); err != nil {
		t.Fatalf("WriteFile() error: %v", err)
	}

	cfg := DetectionConfig{Codex: true, Claude: true}
	if _, ok := ClassifyState(paneID, cfg, time.Now()); ok {
		t.Fatalf("ClassifyState() should ignore unknown tool")
	}

	// Disable detection entirely.
	cfg = DetectionConfig{Codex: false, Claude: false}
	if _, ok := ClassifyState(paneID, cfg, time.Now()); ok {
		t.Fatalf("ClassifyState() should be false when detection disabled")
	}
}
