package agent

import (
	"encoding/json"
	"testing"
)

func TestAnthropicMessages(t *testing.T) {
	req := llmRequest{
		Messages: []Message{
			{Role: RoleUser, Text: "hello"},
			{Role: RoleAssistant, Text: "ack", ToolCalls: []ToolCall{{ID: "t1", Name: "tool", Arguments: map[string]any{"x": "y"}}}},
			{Role: RoleTool, ToolResult: &ToolResult{ToolCallID: "t1", Content: "ok", IsError: true}},
			{Role: RoleTool, ToolResult: nil},
		},
	}
	msgs, err := anthropicMessages(req)
	if err != nil {
		t.Fatalf("anthropicMessages error: %v", err)
	}
	if len(msgs) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(msgs))
	}
	if msgs[0].Role != "user" || len(msgs[0].Content) != 1 || msgs[0].Content[0].Text != "hello" {
		t.Fatalf("user msg=%#v", msgs[0])
	}
	if msgs[1].Role != "assistant" || len(msgs[1].Content) != 2 {
		t.Fatalf("assistant msg=%#v", msgs[1])
	}
	if msgs[1].Content[1].Type != "tool_use" || msgs[1].Content[1].ToolUseID != "t1" {
		t.Fatalf("tool use=%#v", msgs[1].Content[1])
	}
	if msgs[2].Role != "user" || msgs[2].Content[0].Type != "tool_result" || !msgs[2].Content[0].IsError {
		t.Fatalf("tool result=%#v", msgs[2])
	}
}

func TestAnthropicParseResponse(t *testing.T) {
	payload := anthropicResponse{
		Content: []anthropicContent{
			{Type: "text", Text: "Hello"},
			{Type: "tool_use", ToolUseID: "id1", Name: "tool", Input: map[string]any{"k": "v"}},
		},
		Stop: "stop",
	}
	payload.Usage.InputTokens = 2
	payload.Usage.OutputTokens = 3
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	out, err := anthropicParseResponse(data)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if out.Text != "Hello" || out.StopReason != "stop" {
		t.Fatalf("out=%#v", out)
	}
	if len(out.ToolCalls) != 1 || out.ToolCalls[0].Name != "tool" {
		t.Fatalf("toolcalls=%#v", out.ToolCalls)
	}
	if out.Usage.TotalTokens != 5 {
		t.Fatalf("usage=%#v", out.Usage)
	}
}

func TestAnthropicParseResponseEmpty(t *testing.T) {
	data, err := json.Marshal(anthropicResponse{})
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	out, err := anthropicParseResponse(data)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if out.Text != "(no response)" {
		t.Fatalf("expected fallback text, got %q", out.Text)
	}
}

func TestAnthropicPayload(t *testing.T) {
	req := llmRequest{
		SystemPrompt: "sys",
		Messages:     []Message{{Role: RoleUser, Text: "hi"}},
		Tools:        []ToolSpec{{Name: "tool", Description: "desc", Schema: map[string]any{"type": "object"}}},
	}
	payload, err := anthropicPayload(req, "model")
	if err != nil {
		t.Fatalf("anthropicPayload error: %v", err)
	}
	if payload.System != "sys" || payload.Model != "model" {
		t.Fatalf("payload=%#v", payload)
	}
	if len(payload.Tools) != 1 || payload.Tools[0].Name != "tool" {
		t.Fatalf("tools=%#v", payload.Tools)
	}
}
