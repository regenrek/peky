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

const anthropicVersion = "2023-06-01"

type anthropicClient struct {
	cfg providerConfig
}

type anthropicContent struct {
	Type       string         `json:"type"`
	Text       string         `json:"text,omitempty"`
	ToolUseID  string         `json:"id,omitempty"`
	Name       string         `json:"name,omitempty"`
	Input      map[string]any `json:"input,omitempty"`
	ToolResult string         `json:"content,omitempty"`
	IsError    bool           `json:"is_error,omitempty"`
}

type anthropicMessage struct {
	Role    string             `json:"role"`
	Content []anthropicContent `json:"content"`
}

type anthropicTool struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	InputSchema map[string]any `json:"input_schema"`
}

type anthropicRequest struct {
	Model     string             `json:"model"`
	MaxTokens int                `json:"max_tokens"`
	System    string             `json:"system,omitempty"`
	Messages  []anthropicMessage `json:"messages"`
	Tools     []anthropicTool    `json:"tools,omitempty"`
}

type anthropicResponse struct {
	Content []anthropicContent `json:"content"`
	Stop    string             `json:"stop_reason"`
	Usage   struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

func newAnthropicClient(cfg providerConfig) *anthropicClient {
	return &anthropicClient{cfg: cfg}
}

func (c *anthropicClient) Generate(ctx context.Context, req llmRequest) (llmResponse, error) {
	if strings.TrimSpace(c.cfg.APIKey) == "" {
		return llmResponse{}, errors.New("missing API key")
	}
	messages, err := anthropicMessages(req)
	if err != nil {
		return llmResponse{}, err
	}
	payload := anthropicRequest{
		Model:     c.cfg.Model,
		MaxTokens: 2048,
		System:    strings.TrimSpace(req.SystemPrompt),
		Messages:  messages,
	}
	if len(req.Tools) > 0 {
		payload.Tools = anthropicTools(req.Tools)
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return llmResponse{}, fmt.Errorf("anthropic request encode: %w", err)
	}
	url := strings.TrimRight(c.cfg.BaseURL, "/") + "/v1/messages"
	reqHTTP, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return llmResponse{}, fmt.Errorf("anthropic request: %w", err)
	}
	reqHTTP.Header.Set("Content-Type", "application/json")
	reqHTTP.Header.Set("x-api-key", c.cfg.APIKey)
	reqHTTP.Header.Set("anthropic-version", anthropicVersion)
	resp, err := http.DefaultClient.Do(reqHTTP)
	if err != nil {
		return llmResponse{}, err
	}
	data, readErr := io.ReadAll(resp.Body)
	closeErr := resp.Body.Close()
	if readErr != nil {
		err := fmt.Errorf("anthropic response read: %w", readErr)
		if closeErr != nil {
			return llmResponse{}, errors.Join(err, fmt.Errorf("anthropic response close: %w", closeErr))
		}
		return llmResponse{}, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		err := fmt.Errorf("anthropic error: %s", string(data))
		if closeErr != nil {
			return llmResponse{}, errors.Join(err, fmt.Errorf("anthropic response close: %w", closeErr))
		}
		return llmResponse{}, err
	}
	if closeErr != nil {
		return llmResponse{}, fmt.Errorf("anthropic response close: %w", closeErr)
	}
	var parsed anthropicResponse
	if err := json.Unmarshal(data, &parsed); err != nil {
		return llmResponse{}, fmt.Errorf("anthropic response parse: %w", err)
	}
	result := llmResponse{
		StopReason: parsed.Stop,
		Usage: Usage{
			InputTokens:  parsed.Usage.InputTokens,
			OutputTokens: parsed.Usage.OutputTokens,
			TotalTokens:  parsed.Usage.InputTokens + parsed.Usage.OutputTokens,
		},
	}
	for _, block := range parsed.Content {
		switch block.Type {
		case "text":
			result.Text += block.Text
		case "tool_use":
			result.ToolCalls = append(result.ToolCalls, ToolCall{
				ID:        block.ToolUseID,
				Name:      block.Name,
				Arguments: block.Input,
			})
		}
	}
	if result.Text == "" && len(result.ToolCalls) == 0 {
		result.Text = "(no response)"
	}
	return result, nil
}

func anthropicMessages(req llmRequest) ([]anthropicMessage, error) {
	out := make([]anthropicMessage, 0, len(req.Messages))
	for _, msg := range req.Messages {
		switch msg.Role {
		case RoleUser:
			out = append(out, anthropicMessage{Role: "user", Content: []anthropicContent{{Type: "text", Text: msg.Text}}})
		case RoleAssistant:
			content := []anthropicContent{}
			if strings.TrimSpace(msg.Text) != "" {
				content = append(content, anthropicContent{Type: "text", Text: msg.Text})
			}
			for _, call := range msg.ToolCalls {
				content = append(content, anthropicContent{Type: "tool_use", ToolUseID: call.ID, Name: call.Name, Input: call.Arguments})
			}
			if len(content) > 0 {
				out = append(out, anthropicMessage{Role: "assistant", Content: content})
			}
		case RoleTool:
			if msg.ToolResult == nil {
				continue
			}
			content := anthropicContent{Type: "tool_result", ToolUseID: msg.ToolResult.ToolCallID, ToolResult: msg.ToolResult.Content, IsError: msg.ToolResult.IsError}
			out = append(out, anthropicMessage{Role: "user", Content: []anthropicContent{content}})
		}
	}
	return out, nil
}

func anthropicTools(specs []ToolSpec) []anthropicTool {
	tools := make([]anthropicTool, 0, len(specs))
	for _, tool := range specs {
		tools = append(tools, anthropicTool{
			Name:        tool.Name,
			Description: tool.Description,
			InputSchema: tool.Schema,
		})
	}
	return tools
}
