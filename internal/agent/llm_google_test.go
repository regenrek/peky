package agent

import (
	"encoding/json"
	"testing"
)

func TestGoogleContentsIncludesToolCallWithSignature(t *testing.T) {
	msgs := []Message{
		NewUserMessage("hi"),
		NewAssistantMessage("", []ToolCall{{
			ID:               "c1",
			Name:             "peky",
			Arguments:        map[string]any{"command": "pane list"},
			ThoughtSignature: "sig",
		}}),
		NewToolResultMessage(ToolResult{
			ToolCallID: "c1",
			ToolName:   "peky",
			Content:    "ok",
		}),
	}
	contents := googleContents(msgs)
	if len(contents) != 3 {
		t.Fatalf("contents=%#v", contents)
	}
	if contents[1].Role != "model" || len(contents[1].Parts) != 1 || contents[1].Parts[0].FunctionCall == nil {
		t.Fatalf("assistant=%#v", contents[1])
	}
	if contents[2].Role != "user" || contents[2].Parts[0].FunctionResp == nil {
		t.Fatalf("tool=%#v", contents[2])
	}
}

func TestAllowGoogleToolCall(t *testing.T) {
	allowed := googleAllowedTools{ids: map[string]struct{}{"c1": {}}, allowNoID: false}
	if !allowGoogleToolCall("c1", allowed) {
		t.Fatalf("expected allowed")
	}
	if allowGoogleToolCall("", allowed) {
		t.Fatalf("expected not allowed")
	}
	allowed.allowNoID = true
	if !allowGoogleToolCall("", allowed) {
		t.Fatalf("expected allowed without id")
	}
}

func TestGoogleParseResponseToolCalls(t *testing.T) {
	raw := googleResponse{
		Candidates: []struct {
			Content struct {
				Parts []googlePart `json:"parts"`
			} `json:"content"`
			FinishReason string `json:"finishReason"`
		}{{
			Content: struct {
				Parts []googlePart `json:"parts"`
			}{
				Parts: []googlePart{
					{Text: "hi"},
					{ThoughtSignature: "sig", FunctionCall: &googleFunctionCall{Name: "peky", Args: map[string]any{"command": "pane list"}, ID: "c1"}},
				},
			},
			FinishReason: "STOP",
		}},
	}
	raw.UsageMetadata.PromptTokens = 1
	raw.UsageMetadata.CandidatesTokens = 2
	raw.UsageMetadata.TotalTokens = 3
	data, _ := json.Marshal(raw)
	got, err := googleParseResponse(data)
	if err != nil {
		t.Fatalf("googleParseResponse error: %v", err)
	}
	if got.Text != "hi" || got.StopReason != "STOP" || len(got.ToolCalls) != 1 {
		t.Fatalf("got=%#v", got)
	}
	if got.ToolCalls[0].ThoughtSignature != "sig" {
		t.Fatalf("call=%#v", got.ToolCalls[0])
	}
}
