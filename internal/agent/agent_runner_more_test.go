package agent

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateRunPromptInput(t *testing.T) {
	if err := validateRunPromptInput("x", nil); err == nil {
		t.Fatalf("expected error for nil tool executor")
	}
	if err := validateRunPromptInput(" ", func(ctx context.Context, call ToolCall) (ToolResult, error) { return ToolResult{}, nil }); err == nil {
		t.Fatalf("expected error for empty prompt")
	}
}

func TestNormalizeRunConfigDefaults(t *testing.T) {
	if _, err := normalizeRunConfig(Config{}); err == nil {
		t.Fatalf("expected error for missing model")
	}
	cfg, err := normalizeRunConfig(Config{Provider: ProviderUnknown, Model: "  m "})
	if err != nil {
		t.Fatalf("normalizeRunConfig error: %v", err)
	}
	if cfg.Provider != ProviderGoogle || cfg.Model != "m" {
		t.Fatalf("cfg=%#v", cfg)
	}
}

func TestBuildSystemPromptAddsContextHint(t *testing.T) {
	got, err := buildSystemPrompt("", " hello ")
	if err != nil {
		t.Fatalf("buildSystemPrompt error: %v", err)
	}
	if !strings.Contains(got, "hello") {
		t.Fatalf("prompt=%q", got)
	}
}

func TestSplitFrontmatter(t *testing.T) {
	front, body := splitFrontmatter("---\nname: x\n---\n\nbody\n")
	if !strings.Contains(front, "name: x") || !strings.Contains(body, "body") {
		t.Fatalf("front=%q body=%q", front, body)
	}
}

func TestLoadSkillsAndBuildPrompt(t *testing.T) {
	dir := t.TempDir()
	skillDir := filepath.Join(dir, "demo")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	content := "---\nname: demo\ndescription: desc\n---\n\nDo X\n"
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	skills, err := loadSkills(dir)
	if err != nil || len(skills) != 1 {
		t.Fatalf("skills=%v err=%v", skills, err)
	}
	prompt := buildSkillsPrompt(skills)
	if !strings.Contains(prompt, "## Skills") || !strings.Contains(prompt, "Do X") {
		t.Fatalf("prompt=%q", prompt)
	}
}

func TestFilterToolCallsDedupAndSignature(t *testing.T) {
	calls := []ToolCall{
		{Name: "peky", ThoughtSignature: "", Arguments: map[string]any{"command": "pane list"}},
		{Name: "peky", ThoughtSignature: "sig", Arguments: map[string]any{"command": "pane   list"}},
		{Name: "peky", ThoughtSignature: "sig", Arguments: map[string]any{"command": "pane list"}},
	}
	got := filterToolCalls(ProviderGoogle, calls)
	if len(got) != 1 {
		t.Fatalf("got=%v", got)
	}
}

type stubLLMClient struct {
	resp llmResponse
	err  error
}

func (c stubLLMClient) Generate(ctx context.Context, req llmRequest) (llmResponse, error) {
	return c.resp, c.err
}

func TestRunStateRunNoToolCalls(t *testing.T) {
	state := &runState{
		norm:         Config{Provider: ProviderGoogle, Model: "m"},
		client:       stubLLMClient{resp: llmResponse{Text: " ok ", StopReason: "stop", Usage: Usage{}}},
		systemPrompt: "sys",
		tools:        []ToolSpec{pekyToolSpec()},
		execTool: func(ctx context.Context, call ToolCall) (ToolResult, error) {
			return ToolResult{}, nil
		},
		runID:    "test",
		history:  nil,
		messages: buildMessageHistory(nil, "hi"),
		last:     Result{},
		trace:    nil,
	}
	got, msgs, err := state.run(context.Background())
	if err != nil {
		t.Fatalf("run error: %v", err)
	}
	if got.Text != "ok" || got.StopReason != "stop" {
		t.Fatalf("got=%#v", got)
	}
	if len(msgs) < 2 {
		t.Fatalf("msgs=%d", len(msgs))
	}
}

func TestExecCallMarksError(t *testing.T) {
	s := &runState{
		execTool: func(ctx context.Context, call ToolCall) (ToolResult, error) {
			return ToolResult{}, errors.New("boom")
		},
	}
	out := s.execCall(context.Background(), ToolCall{Name: "peky", Arguments: map[string]any{"command": "x"}})
	if !out.IsError || out.Content != "boom" {
		t.Fatalf("out=%#v", out)
	}
}

func TestBuildProviderConfig(t *testing.T) {
	if _, err := buildProviderConfig(ProviderUnknown, "m", "k", nil); err == nil {
		t.Fatalf("expected error")
	}
	cfg, err := buildProviderConfig(ProviderOpenAI, "m", "k", nil)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if cfg.BaseURL == "" || cfg.Provider != ProviderOpenAI {
		t.Fatalf("cfg=%#v", cfg)
	}
}

func TestProviderInfoHelpers(t *testing.T) {
	if _, ok := FindProviderInfo("google"); !ok {
		t.Fatalf("expected google provider")
	}
	if ProviderLabel(ProviderGoogle) == "" {
		t.Fatalf("expected label")
	}
	if !ProviderHasModel(ProviderGoogle, ProviderDefaultModel(ProviderGoogle)) {
		t.Fatalf("expected default model present")
	}
	if ProviderHasModel(ProviderGoogle, " ") {
		t.Fatalf("blank model should be false")
	}
	unknown := ProviderModels(Provider("unknown"))
	if unknown != nil {
		t.Fatalf("expected nil")
	}
}
