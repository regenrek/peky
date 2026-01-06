package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

type googleClient struct {
	cfg providerConfig
}

type googlePart struct {
	Text             string                  `json:"text,omitempty"`
	ThoughtSignature string                  `json:"thoughtSignature,omitempty"`
	FunctionCall     *googleFunctionCall     `json:"functionCall,omitempty"`
	FunctionResp     *googleFunctionResponse `json:"functionResponse,omitempty"`
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
	payload := googleRequest{
		Model:    c.cfg.Model,
		Contents: googleContents(req.Messages),
	}
	if strings.TrimSpace(req.SystemPrompt) != "" {
		payload.SystemInstruction = &googleContent{Parts: []googlePart{{Text: req.SystemPrompt}}}
	}
	if len(req.Tools) > 0 {
		payload.Tools = []googleTools{{FunctionDeclarations: googleToolDecls(req.Tools)}}
		payload.ToolConfig = map[string]any{"functionCallingConfig": map[string]any{"mode": "AUTO"}}
	}
	data, err := googleDoRequest(ctx, c.cfg, payload)
	if err != nil {
		return llmResponse{}, err
	}
	return googleParseResponse(data)
}

func googleContents(messages []Message) []googleContent {
	allowed := collectGoogleToolIDs(messages)
	out := make([]googleContent, 0, len(messages))
	for _, msg := range messages {
		if content, ok := googleContentForMessage(msg, allowed); ok {
			out = append(out, content)
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

type googleAllowedTools struct {
	ids       map[string]struct{}
	allowNoID bool
}

func googleDoRequest(ctx context.Context, cfg providerConfig, payload googleRequest) ([]byte, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("google request encode: %w", err)
	}
	url := strings.TrimRight(cfg.BaseURL, "/") + "/v1beta/models/" + cfg.Model + ":generateContent?key=" + cfg.APIKey
	reqHTTP, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("google request: %w", err)
	}
	reqHTTP.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(reqHTTP)
	if err != nil {
		return nil, err
	}
	return readHTTPResponse(resp, "google")
}

func googleParseResponse(data []byte) (llmResponse, error) {
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
	mergeGoogleParts(&result, candidate.Content.Parts)
	if result.Text == "" && len(result.ToolCalls) == 0 {
		result.Text = "(no response)"
	}
	return result, nil
}

func collectGoogleToolIDs(messages []Message) googleAllowedTools {
	allowed := googleAllowedTools{ids: map[string]struct{}{}}
	for _, msg := range messages {
		if msg.Role != RoleAssistant {
			continue
		}
		for _, call := range msg.ToolCalls {
			if strings.TrimSpace(call.ThoughtSignature) == "" {
				continue
			}
			if strings.TrimSpace(call.ID) == "" {
				allowed.allowNoID = true
				continue
			}
			allowed.ids[call.ID] = struct{}{}
		}
	}
	return allowed
}

func googleContentForMessage(msg Message, allowed googleAllowedTools) (googleContent, bool) {
	switch msg.Role {
	case RoleUser:
		return googleContent{Role: "user", Parts: []googlePart{{Text: msg.Text}}}, true
	case RoleAssistant:
		return googleAssistantContent(msg)
	case RoleTool:
		return googleToolContent(msg, allowed)
	default:
		return googleContent{}, false
	}
}

func googleAssistantContent(msg Message) (googleContent, bool) {
	parts := []googlePart{}
	if strings.TrimSpace(msg.Text) != "" {
		parts = append(parts, googlePart{Text: msg.Text})
	}
	for _, call := range msg.ToolCalls {
		if strings.TrimSpace(call.ThoughtSignature) == "" {
			continue
		}
		parts = append(parts, googlePart{
			ThoughtSignature: call.ThoughtSignature,
			FunctionCall: &googleFunctionCall{
				Name: call.Name,
				Args: call.Arguments,
				ID:   call.ID,
			},
		})
	}
	if len(parts) == 0 {
		return googleContent{}, false
	}
	return googleContent{Role: "model", Parts: parts}, true
}

func googleToolContent(msg Message, allowed googleAllowedTools) (googleContent, bool) {
	if msg.ToolResult == nil {
		return googleContent{}, false
	}
	if !allowGoogleToolCall(msg.ToolResult.ToolCallID, allowed) {
		return googleContent{}, false
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
	return googleContent{Role: "user", Parts: []googlePart{{FunctionResp: &resp}}}, true
}

func allowGoogleToolCall(callID string, allowed googleAllowedTools) bool {
	if callID != "" {
		_, ok := allowed.ids[callID]
		return ok
	}
	return allowed.allowNoID
}

func mergeGoogleParts(result *llmResponse, parts []googlePart) {
	for _, part := range parts {
		if part.Text != "" {
			result.Text += part.Text
		}
		if part.FunctionCall != nil {
			result.ToolCalls = append(result.ToolCalls, ToolCall{
				ID:               part.FunctionCall.ID,
				Name:             part.FunctionCall.Name,
				Arguments:        part.FunctionCall.Args,
				ThoughtSignature: part.ThoughtSignature,
			})
		}
	}
}
