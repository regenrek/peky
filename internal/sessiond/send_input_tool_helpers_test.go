package sessiond

import (
	"testing"
	"time"

	"github.com/regenrek/peakypanes/internal/layout"
	"github.com/regenrek/peakypanes/internal/native"
	"github.com/regenrek/peakypanes/internal/tool"
)

func defaultToolRegistry(t *testing.T) *tool.Registry {
	t.Helper()
	reg, err := tool.DefaultRegistry()
	if err != nil {
		t.Fatalf("DefaultRegistry: %v", err)
	}
	return reg
}

func TestNormalizeToolFilterUnknown(t *testing.T) {
	reg := defaultToolRegistry(t)
	if _, err := normalizeToolFilter(reg, "unknown"); err == nil {
		t.Fatalf("expected error for unknown tool")
	}
}

func TestNormalizeToolFilterAllowed(t *testing.T) {
	reg := defaultToolRegistry(t)
	if got, err := normalizeToolFilter(reg, "codex"); err != nil || got != "codex" {
		t.Fatalf("normalizeToolFilter(codex) = %q err=%v", got, err)
	}
}

func TestNormalizeToolFilterDisallowed(t *testing.T) {
	reg, err := tool.RegistryFromConfig(toolConfigWithAllow(map[string]bool{"codex": false}))
	if err != nil {
		t.Fatalf("RegistryFromConfig: %v", err)
	}
	if _, err := normalizeToolFilter(reg, "codex"); err == nil {
		t.Fatalf("expected error for disallowed tool")
	}
}

func TestResolveSendInputToolTargetErrors(t *testing.T) {
	if _, err := resolveSendInputToolTarget(SendInputToolRequest{}); err == nil {
		t.Fatalf("expected error for missing target")
	}
	if _, err := resolveSendInputToolTarget(SendInputToolRequest{PaneID: "p", Scope: "all"}); err == nil {
		t.Fatalf("expected error for mixed target")
	}
}

func TestBuildToolSendPlanRaw(t *testing.T) {
	reg := defaultToolRegistry(t)
	info := tool.PaneInfo{Tool: "codex", StartCommand: "codex"}
	req := SendInputToolRequest{Input: []byte("hello"), Submit: true, Raw: true}
	plan, ok := buildToolSendPlan(reg, info, req, "")
	if !ok {
		t.Fatalf("expected plan")
	}
	if got := string(plan.Payload); got != "hello" {
		t.Fatalf("payload = %q", got)
	}
	if got := string(plan.Submit); got != "\n" {
		t.Fatalf("submit = %q", got)
	}
}

func TestBuildToolSendPlanSubmitDelayOverride(t *testing.T) {
	reg := defaultToolRegistry(t)
	info := tool.PaneInfo{StartCommand: "claude"}
	delay := 150
	req := SendInputToolRequest{Input: []byte("hello"), Submit: true, SubmitDelayMS: &delay}
	plan, ok := buildToolSendPlan(reg, info, req, "")
	if !ok {
		t.Fatalf("expected plan")
	}
	if plan.SubmitDelay != 150*time.Millisecond {
		t.Fatalf("submit delay = %v", plan.SubmitDelay)
	}
}

func TestBuildToolSendPlanFilterMismatch(t *testing.T) {
	reg := defaultToolRegistry(t)
	info := tool.PaneInfo{Tool: "codex", StartCommand: "codex"}
	req := SendInputToolRequest{Input: []byte("hello")}
	if _, ok := buildToolSendPlan(reg, info, req, "claude"); ok {
		t.Fatalf("expected filter mismatch")
	}
}

func TestBuildToolSendPlanDisallowedFallsBack(t *testing.T) {
	reg, err := tool.RegistryFromConfig(toolConfigWithAllow(map[string]bool{"codex": false}))
	if err != nil {
		t.Fatalf("RegistryFromConfig: %v", err)
	}
	info := tool.PaneInfo{Tool: "codex", StartCommand: "codex"}
	req := SendInputToolRequest{Input: []byte("hello"), Submit: true}
	plan, ok := buildToolSendPlan(reg, info, req, "")
	if !ok {
		t.Fatalf("expected plan")
	}
	if string(plan.Submit) != "\n" {
		t.Fatalf("expected default submit, got %q", string(plan.Submit))
	}
	if plan.Combine {
		t.Fatalf("expected default profile (no combine)")
	}
}

func toolConfigWithAllow(allow map[string]bool) layout.ToolDetectionConfig {
	return layout.ToolDetectionConfig{Allow: allow}
}

func TestBuildToolSendPlanNegativeDelay(t *testing.T) {
	reg := defaultToolRegistry(t)
	info := tool.PaneInfo{StartCommand: "codex"}
	delay := -10
	req := SendInputToolRequest{Input: []byte("hello"), Submit: true, SubmitDelayMS: &delay}
	plan, ok := buildToolSendPlan(reg, info, req, "")
	if !ok {
		t.Fatalf("expected plan")
	}
	if plan.SubmitDelay != 0 {
		t.Fatalf("submit delay = %v", plan.SubmitDelay)
	}
}

func TestLookupPaneInfoNotFound(t *testing.T) {
	manager := &fakeManager{
		snapshot: []native.SessionSnapshot{
			{Name: "s1", Panes: []native.PaneSnapshot{{ID: "p-1"}}},
		},
	}
	d := &Daemon{manager: manager}
	if _, err := d.lookupPaneInfo(manager, "missing"); err == nil {
		t.Fatalf("expected error for missing pane")
	}
}

func TestSnapshotPaneInfoSkipsEmptyID(t *testing.T) {
	sessions := []native.SessionSnapshot{
		{Name: "s1", Panes: []native.PaneSnapshot{{ID: ""}, {ID: "p-1"}}},
	}
	info := snapshotPaneInfo(sessions)
	if len(info) != 1 {
		t.Fatalf("expected 1 pane, got %d", len(info))
	}
	if _, ok := info["p-1"]; !ok {
		t.Fatalf("expected p-1 in map")
	}
}

func TestHandleSendInputToolNoSessions(t *testing.T) {
	manager := &fakeManager{}
	d := &Daemon{manager: manager}
	payload, err := encodePayload(SendInputToolRequest{Scope: "all", Input: []byte("hello")})
	if err != nil {
		t.Fatalf("encodePayload: %v", err)
	}
	if _, err := d.handleSendInputTool(payload); err == nil {
		t.Fatalf("expected error for no sessions")
	}
}
