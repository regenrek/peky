package agent

import "testing"

func TestToolCallKey(t *testing.T) {
	if got := toolCallKey(ToolCall{Name: "noop"}); got != "" {
		t.Fatalf("expected empty key")
	}
	key := toolCallKey(ToolCall{Name: "peky", Arguments: map[string]any{"command": "  hello   world  "}})
	if key != "peky:hello world" {
		t.Fatalf("key=%q", key)
	}
}

func TestFilterToolCalls(t *testing.T) {
	calls := []ToolCall{
		{Name: "peky", Arguments: map[string]any{"command": "one"}, ThoughtSignature: "sig"},
		{Name: "peky", Arguments: map[string]any{"command": "one"}, ThoughtSignature: "sig"},
		{Name: "peky", Arguments: map[string]any{"command": "two"}},
	}
	out := filterToolCalls(ProviderGoogle, calls)
	if len(out) != 1 {
		t.Fatalf("expected 1 call, got %d", len(out))
	}
	out = filterToolCalls(ProviderOpenAI, calls)
	if len(out) != 2 {
		t.Fatalf("expected 2 calls, got %d", len(out))
	}
}
