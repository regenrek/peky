package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type openaiClient struct {
	cfg providerConfig
}

type openaiMessage struct {
	Role       string           `json:"role"`
	Content    string           `json:"content,omitempty"`
	ToolCalls  []openaiToolCall `json:"tool_calls,omitempty"`
	ToolCallID string           `json:"tool_call_id,omitempty"`
	Name       string           `json:"name,omitempty"`
}

type openaiToolCall struct {
	ID       string            `json:"id"`
	Type     string            `json:"type"`
	Function openaiToolCallDef `json:"function"`
}

type openaiToolCallDef struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type openaiTool struct {
	Type     string          `json:"type"`
	Function openaiToolEntry `json:"function"`
}

type openaiToolEntry struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Parameters  map[string]any `json:"parameters"`
}

type openaiRequest struct {
	Model      string          `json:"model"`
	Messages   []openaiMessage `json:"messages"`
	Tools      []openaiTool    `json:"tools,omitempty"`
	ToolChoice any             `json:"tool_choice,omitempty"`
}

type openaiResponse struct {
	Choices []struct {
		Message openaiMessage `json:"message"`
		Finish  string        `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

func newOpenAIClient(cfg providerConfig) *openaiClient {
	return &openaiClient{cfg: cfg}
}

func (c *openaiClient) Generate(ctx context.Context, req llmRequest) (llmResponse, error) {
	if strings.TrimSpace(c.cfg.APIKey) == "" {
		return llmResponse{}, errors.New("missing API key")
	}
	messages, err := openaiMessages(req)
	if err != nil {
		return llmResponse{}, err
	}
	payload := openaiRequest{
		Model:    c.cfg.Model,
		Messages: messages,
	}
	if len(req.Tools) > 0 {
		payload.Tools = openaiTools(req.Tools)
		payload.ToolChoice = "auto"
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return llmResponse{}, fmt.Errorf("openai request encode: %w", err)
	}
	url := strings.TrimRight(c.cfg.BaseURL, "/") + "/chat/completions"
	reqHTTP, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return llmResponse{}, fmt.Errorf("openai request: %w", err)
	}
	reqHTTP.Header.Set("Content-Type", "application/json")
	reqHTTP.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)
	client := http.DefaultClient
	resp, err := client.Do(reqHTTP)
	if err != nil {
		return llmResponse{}, err
	}
	data, readErr := io.ReadAll(resp.Body)
	closeErr := resp.Body.Close()
	if readErr != nil {
		err := fmt.Errorf("openai response read: %w", readErr)
		if closeErr != nil {
			return llmResponse{}, errors.Join(err, fmt.Errorf("openai response close: %w", closeErr))
		}
		return llmResponse{}, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		err := fmt.Errorf("openai error: %s", string(data))
		if closeErr != nil {
			return llmResponse{}, errors.Join(err, fmt.Errorf("openai response close: %w", closeErr))
		}
		return llmResponse{}, err
	}
	if closeErr != nil {
		return llmResponse{}, fmt.Errorf("openai response close: %w", closeErr)
	}
	var parsed openaiResponse
	if err := json.Unmarshal(data, &parsed); err != nil {
		return llmResponse{}, fmt.Errorf("openai response parse: %w", err)
	}
	if len(parsed.Choices) == 0 {
		return llmResponse{}, errors.New("openai response missing choices")
	}
	msg := parsed.Choices[0].Message
	result := llmResponse{
		Text:       msg.Content,
		StopReason: parsed.Choices[0].Finish,
		Usage: Usage{
			InputTokens:  parsed.Usage.PromptTokens,
			OutputTokens: parsed.Usage.CompletionTokens,
			TotalTokens:  parsed.Usage.TotalTokens,
		},
	}
	if len(msg.ToolCalls) > 0 {
		result.ToolCalls = make([]ToolCall, 0, len(msg.ToolCalls))
		for _, call := range msg.ToolCalls {
			args := map[string]any{}
			if call.Function.Arguments != "" {
				_ = json.Unmarshal([]byte(call.Function.Arguments), &args)
			}
			result.ToolCalls = append(result.ToolCalls, ToolCall{
				ID:        call.ID,
				Name:      call.Function.Name,
				Arguments: args,
			})
		}
	}
	if result.Text == "" && len(result.ToolCalls) == 0 {
		result.Text = "(no response)"
	}
	return result, nil
}

func openaiMessages(req llmRequest) ([]openaiMessage, error) {
	out := make([]openaiMessage, 0, len(req.Messages)+1)
	if strings.TrimSpace(req.SystemPrompt) != "" {
		out = append(out, openaiMessage{Role: "system", Content: req.SystemPrompt})
	}
	for _, msg := range req.Messages {
		switch msg.Role {
		case RoleUser:
			out = append(out, openaiMessage{Role: "user", Content: msg.Text})
		case RoleAssistant:
			entry := openaiMessage{Role: "assistant", Content: msg.Text}
			if len(msg.ToolCalls) > 0 {
				entry.ToolCalls = make([]openaiToolCall, 0, len(msg.ToolCalls))
				for _, call := range msg.ToolCalls {
					payload, err := json.Marshal(call.Arguments)
					if err != nil {
						return nil, fmt.Errorf("openai tool args: %w", err)
					}
					entry.ToolCalls = append(entry.ToolCalls, openaiToolCall{
						ID:   call.ID,
						Type: "function",
						Function: openaiToolCallDef{
							Name:      call.Name,
							Arguments: string(payload),
						},
					})
				}
			}
			out = append(out, entry)
		case RoleTool:
			if msg.ToolResult == nil {
				continue
			}
			out = append(out, openaiMessage{
				Role:       "tool",
				Content:    msg.ToolResult.Content,
				ToolCallID: msg.ToolResult.ToolCallID,
				Name:       msg.ToolResult.ToolName,
			})
		}
	}
	return out, nil
}

func openaiTools(specs []ToolSpec) []openaiTool {
	tools := make([]openaiTool, 0, len(specs))
	for _, tool := range specs {
		tools = append(tools, openaiTool{
			Type: "function",
			Function: openaiToolEntry{
				Name:        tool.Name,
				Description: tool.Description,
				Parameters:  tool.Schema,
			},
		})
	}
	return tools
}
