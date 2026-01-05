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

type googleClient struct {
	cfg providerConfig
}

type googlePart struct {
	Text         string                  `json:"text,omitempty"`
	FunctionCall *googleFunctionCall     `json:"functionCall,omitempty"`
	FunctionResp *googleFunctionResponse `json:"functionResponse,omitempty"`
}

type googleFunctionCall struct {
	Name string         `json:"name"`
	Args map[string]any `json:"args"`
	ID   string         `json:"id,omitempty"`
}

type googleFunctionResponse struct {
	Name     string         `json:"name"`
	ID       string         `json:"id,omitempty"`
	Response map[string]any `json:"response"`
}

type googleContent struct {
	Role  string       `json:"role"`
	Parts []googlePart `json:"parts"`
}

type googleToolDecl struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Parameters  map[string]any `json:"parameters"`
}

type googleTools struct {
	FunctionDeclarations []googleToolDecl `json:"functionDeclarations"`
}

type googleRequest struct {
	Model             string          `json:"model"`
	Contents          []googleContent `json:"contents"`
	SystemInstruction *googleContent  `json:"systemInstruction,omitempty"`
	Tools             []googleTools   `json:"tools,omitempty"`
	ToolConfig        map[string]any  `json:"toolConfig,omitempty"`
}

type googleResponse struct {
	Candidates []struct {
		Content struct {
			Parts []googlePart `json:"parts"`
		} `json:"content"`
		FinishReason string `json:"finishReason"`
	} `json:"candidates"`
	UsageMetadata struct {
		PromptTokens     int `json:"promptTokenCount"`
		CandidatesTokens int `json:"candidatesTokenCount"`
		TotalTokens      int `json:"totalTokenCount"`
	} `json:"usageMetadata"`
}

func newGoogleClient(cfg providerConfig) *googleClient {
	return &googleClient{cfg: cfg}
}

func (c *googleClient) Generate(ctx context.Context, req llmRequest) (llmResponse, error) {
	if strings.TrimSpace(c.cfg.APIKey) == "" {
		return llmResponse{}, errors.New("missing API key")
	}
	contents := googleContents(req.Messages)
	payload := googleRequest{
		Model:    c.cfg.Model,
		Contents: contents,
	}
	if strings.TrimSpace(req.SystemPrompt) != "" {
		payload.SystemInstruction = &googleContent{Parts: []googlePart{{Text: req.SystemPrompt}}}
	}
	if len(req.Tools) > 0 {
		payload.Tools = []googleTools{{FunctionDeclarations: googleToolDecls(req.Tools)}}
		payload.ToolConfig = map[string]any{"functionCallingConfig": map[string]any{"mode": "AUTO"}}
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return llmResponse{}, fmt.Errorf("google request encode: %w", err)
	}
	url := strings.TrimRight(c.cfg.BaseURL, "/") + "/v1beta/models/" + c.cfg.Model + ":generateContent?key=" + c.cfg.APIKey
	reqHTTP, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return llmResponse{}, fmt.Errorf("google request: %w", err)
	}
	reqHTTP.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(reqHTTP)
	if err != nil {
		return llmResponse{}, err
	}
	data, readErr := io.ReadAll(resp.Body)
	closeErr := resp.Body.Close()
	if readErr != nil {
		err := fmt.Errorf("google response read: %w", readErr)
		if closeErr != nil {
			return llmResponse{}, errors.Join(err, fmt.Errorf("google response close: %w", closeErr))
		}
		return llmResponse{}, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		err := fmt.Errorf("google error: %s", string(data))
		if closeErr != nil {
			return llmResponse{}, errors.Join(err, fmt.Errorf("google response close: %w", closeErr))
		}
		return llmResponse{}, err
	}
	if closeErr != nil {
		return llmResponse{}, fmt.Errorf("google response close: %w", closeErr)
	}
	var parsed googleResponse
	if err := json.Unmarshal(data, &parsed); err != nil {
		return llmResponse{}, fmt.Errorf("google response parse: %w", err)
	}
	if len(parsed.Candidates) == 0 {
		return llmResponse{}, errors.New("google response missing candidates")
	}
	candidate := parsed.Candidates[0]
	result := llmResponse{
		StopReason: candidate.FinishReason,
		Usage: Usage{
			InputTokens:  parsed.UsageMetadata.PromptTokens,
			OutputTokens: parsed.UsageMetadata.CandidatesTokens,
			TotalTokens:  parsed.UsageMetadata.TotalTokens,
		},
	}
	for _, part := range candidate.Content.Parts {
		if part.Text != "" {
			result.Text += part.Text
		}
		if part.FunctionCall != nil {
			result.ToolCalls = append(result.ToolCalls, ToolCall{
				ID:        part.FunctionCall.ID,
				Name:      part.FunctionCall.Name,
				Arguments: part.FunctionCall.Args,
			})
		}
	}
	if result.Text == "" && len(result.ToolCalls) == 0 {
		result.Text = "(no response)"
	}
	return result, nil
}

func googleContents(messages []Message) []googleContent {
	out := make([]googleContent, 0, len(messages))
	for _, msg := range messages {
		switch msg.Role {
		case RoleUser:
			out = append(out, googleContent{Role: "user", Parts: []googlePart{{Text: msg.Text}}})
		case RoleAssistant:
			parts := []googlePart{}
			if strings.TrimSpace(msg.Text) != "" {
				parts = append(parts, googlePart{Text: msg.Text})
			}
			for _, call := range msg.ToolCalls {
				parts = append(parts, googlePart{FunctionCall: &googleFunctionCall{Name: call.Name, Args: call.Arguments, ID: call.ID}})
			}
			if len(parts) > 0 {
				out = append(out, googleContent{Role: "model", Parts: parts})
			}
		case RoleTool:
			if msg.ToolResult == nil {
				continue
			}
			resp := googleFunctionResponse{
				Name: msg.ToolResult.ToolName,
				ID:   msg.ToolResult.ToolCallID,
				Response: map[string]any{
					"output": msg.ToolResult.Content,
				},
			}
			if msg.ToolResult.IsError {
				resp.Response = map[string]any{"error": msg.ToolResult.Content}
			}
			out = append(out, googleContent{Role: "user", Parts: []googlePart{{FunctionResp: &resp}}})
		}
	}
	return out
}

func googleToolDecls(specs []ToolSpec) []googleToolDecl {
	decls := make([]googleToolDecl, 0, len(specs))
	for _, tool := range specs {
		decls = append(decls, googleToolDecl{
			Name:        tool.Name,
			Description: tool.Description,
			Parameters:  tool.Schema,
		})
	}
	return decls
}
