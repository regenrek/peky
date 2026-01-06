package agent

import (
	"context"
	"testing"
)

func TestMessageFactories(t *testing.T) {
	user := NewUserMessage("hello")
	if user.Role != RoleUser || user.Text != "hello" {
		t.Fatalf("unexpected user message: %#v", user)
	}
	call := ToolCall{ID: "1", Name: "peky"}
	assistant := NewAssistantMessage("ok", []ToolCall{call})
	if assistant.Role != RoleAssistant || assistant.Text != "ok" || len(assistant.ToolCalls) != 1 {
		t.Fatalf("unexpected assistant message: %#v", assistant)
	}
	result := NewToolResultMessage(ToolResult{ToolCallID: "1", ToolName: "peky", Content: "done"})
	if result.Role != RoleTool || result.ToolResult == nil || result.ToolResult.Content != "done" {
		t.Fatalf("unexpected tool result message: %#v", result)
	}
}

func TestRunPromptRequiresExecutor(t *testing.T) {
	_, _, err := RunPrompt(context.TODO(), Config{}, nil, "hi", "", "", nil)
	if err == nil {
		t.Fatalf("expected error for nil executor")
	}
}
