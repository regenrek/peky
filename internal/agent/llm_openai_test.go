package agent

import (
	"encoding/json"
	"testing"
)

func TestOpenAIMessagesIncludesToolCallsAndResults(t *testing.T) {
	req := llmRequest{
		Model:        "m",
		SystemPrompt: "sys",
		Messages: []Message{
			NewUserMessage("hi"),
			NewAssistantMessage("ok", []ToolCall{{
				ID:        "c1",
				Name:      "peky",
				Arguments: map[string]any{"command": "pane list"},
			}}),
			NewToolResultMessage(ToolResult{
				ToolCallID: "c1",
				ToolName:   "peky",
				Content:    "{\"ok\":true}",
			}),
		},
		Tools:      []ToolSpec{pekyToolSpec()},
		ToolChoice: "auto",
	}

	msgs, err := openaiMessages(req)
	if err != nil {
		t.Fatalf("openaiMessages error: %v", err)
	}
	if len(msgs) < 3 || msgs[0].Role != "system" || msgs[1].Role != "user" {
		t.Fatalf("msgs=%#v", msgs)
	}
	if len(msgs[2].ToolCalls) != 1 || msgs[2].ToolCalls[0].Function.Name != "peky" {
		t.Fatalf("assistant=%#v", msgs[2])
	}
	if msgs[3].Role != "tool" || msgs[3].ToolCallID != "c1" {
		t.Fatalf("tool=%#v", msgs[3])
	}
}

func TestOpenAIPayloadAddsTools(t *testing.T) {
	req := llmRequest{
		Model:        "m",
		SystemPrompt: "sys",
		Messages:     []Message{NewUserMessage("hi")},
		Tools:        []ToolSpec{pekyToolSpec()},
	}
	payload, err := openaiPayload(req, "m")
	if err != nil {
		t.Fatalf("openaiPayload error: %v", err)
	}
	if payload.Model != "m" || len(payload.Tools) != 1 || payload.ToolChoice != "auto" {
		t.Fatalf("payload=%#v", payload)
	}
}

func TestOpenAIParseResponseToolCalls(t *testing.T) {
	raw := openaiResponse{
		Choices: []struct {
			Message openaiMessage `json:"message"`
			Finish  string        `json:"finish_reason"`
		}{{
			Message: openaiMessage{
				Content: "",
				ToolCalls: []openaiToolCall{{
					ID:   "c1",
					Type: "function",
					Function: openaiToolCallDef{
						Name:      "peky",
						Arguments: `{"command":"pane list"}`,
					},
				}},
			},
			Finish: "tool_calls",
		}},
	}
	raw.Usage.PromptTokens = 1
	raw.Usage.CompletionTokens = 2
	raw.Usage.TotalTokens = 3
	data, _ := json.Marshal(raw)
	got, err := openaiParseResponse(data)
	if err != nil {
		t.Fatalf("openaiParseResponse error: %v", err)
	}
	if got.StopReason != "tool_calls" || len(got.ToolCalls) != 1 {
		t.Fatalf("got=%#v", got)
	}
	if got.ToolCalls[0].Arguments["command"] != "pane list" {
		t.Fatalf("call=%#v", got.ToolCalls[0])
	}
}
