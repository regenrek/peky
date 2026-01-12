package agent

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/regenrek/peakypanes/internal/identity"
)

type geminiCLIClient struct {
	cfg providerConfig
}

type geminiCLICreds struct {
	Token     string
	ProjectID string
}

type geminiCLIRequest struct {
	Project string `json:"project"`
	Model   string `json:"model"`
	Request struct {
		Contents          []googleContent `json:"contents"`
		SystemInstruction *googleContent  `json:"systemInstruction,omitempty"`
		GenerationConfig  map[string]any  `json:"generationConfig,omitempty"`
		Tools             []googleTools   `json:"tools,omitempty"`
		ToolConfig        map[string]any  `json:"toolConfig,omitempty"`
	} `json:"request"`
	UserAgent string `json:"userAgent,omitempty"`
	RequestID string `json:"requestId,omitempty"`
}

type geminiCLIChunk struct {
	Response *struct {
		Candidates []struct {
			Content struct {
				Parts []googlePart `json:"parts"`
			} `json:"content"`
			FinishReason string `json:"finishReason"`
		} `json:"candidates"`
		UsageMetadata struct {
			PromptTokens     int `json:"promptTokenCount"`
			CandidatesTokens int `json:"candidatesTokenCount"`
			ThoughtsTokens   int `json:"thoughtsTokenCount"`
			TotalTokens      int `json:"totalTokenCount"`
			CachedTokens     int `json:"cachedContentTokenCount"`
		} `json:"usageMetadata"`
	} `json:"response"`
}

func newGeminiCLIClient(cfg providerConfig) *geminiCLIClient {
	return &geminiCLIClient{cfg: cfg}
}

func (c *geminiCLIClient) Generate(ctx context.Context, req llmRequest) (llmResponse, error) {
	creds, err := parseGeminiCLICreds(c.cfg.APIKey)
	if err != nil {
		return llmResponse{}, err
	}
	request := geminiCLIRequest{
		Project:   creds.ProjectID,
		Model:     c.cfg.Model,
		UserAgent: identity.AppSlug,
		RequestID: fmt.Sprintf("%s-%d", identity.AppSlug, time.Now().UnixNano()),
	}
	request.Request.Contents = googleContents(req.Messages)
	if strings.TrimSpace(req.SystemPrompt) != "" {
		request.Request.SystemInstruction = &googleContent{Parts: []googlePart{{Text: req.SystemPrompt}}}
	}
	if len(req.Tools) > 0 {
		request.Request.Tools = []googleTools{{FunctionDeclarations: googleToolDecls(req.Tools)}}
		request.Request.ToolConfig = map[string]any{"functionCallingConfig": map[string]any{"mode": "AUTO"}}
	}
	body, err := json.Marshal(request)
	if err != nil {
		return llmResponse{}, fmt.Errorf("gemini cli request encode: %w", err)
	}
	endpoint := strings.TrimRight(c.cfg.BaseURL, "/") + "/v1internal:streamGenerateContent?alt=sse"
	resp, err := geminiCLIStreamRequest(ctx, endpoint, creds.Token, body)
	if err != nil {
		return llmResponse{}, err
	}
	return geminiCLIParseStream(resp)
}

func parseGeminiCLICreds(apiKey string) (geminiCLICreds, error) {
	if strings.TrimSpace(apiKey) == "" {
		return geminiCLICreds{}, errors.New("missing API key")
	}
	var payloadCred struct {
		Token     string `json:"token"`
		ProjectID string `json:"projectId"`
	}
	if err := json.Unmarshal([]byte(apiKey), &payloadCred); err != nil {
		return geminiCLICreds{}, errors.New("invalid oauth credentials")
	}
	if payloadCred.Token == "" || payloadCred.ProjectID == "" {
		return geminiCLICreds{}, errors.New("missing oauth token or projectId")
	}
	return geminiCLICreds{Token: payloadCred.Token, ProjectID: payloadCred.ProjectID}, nil
}

func geminiCLIStreamRequest(ctx context.Context, endpoint, token string, body []byte) (*http.Response, error) {
	reqHTTP, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("gemini cli request: %w", err)
	}
	for k, v := range geminiCLIHeaders(endpoint, token) {
		reqHTTP.Header.Set(k, v)
	}
	return http.DefaultClient.Do(reqHTTP)
}

func geminiCLIHeaders(endpoint, token string) map[string]string {
	headers := map[string]string{
		"Authorization":   "Bearer " + token,
		"Content-Type":    "application/json",
		"Accept":          "text/event-stream",
		"Client-Metadata": `{"ideType":"IDE_UNSPECIFIED","platform":"PLATFORM_UNSPECIFIED","pluginType":"GEMINI"}`,
	}
	if strings.Contains(endpoint, "sandbox.googleapis.com") {
		headers["User-Agent"] = "antigravity/1.11.5 darwin/arm64"
		headers["X-Goog-Api-Client"] = "google-cloud-sdk vscode_cloudshelleditor/0.1"
	} else {
		headers["User-Agent"] = "google-cloud-sdk vscode_cloudshelleditor/0.1"
		headers["X-Goog-Api-Client"] = "gl-node/22.17.0"
	}
	return headers
}

func geminiCLIParseStream(resp *http.Response) (llmResponse, error) {
	if resp.Body == nil {
		return llmResponse{}, errors.New("gemini cli empty response")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return llmResponse{}, geminiCLIReadError(resp)
	}
	result, scanErr := geminiCLIReadStream(resp.Body)
	closeErr := resp.Body.Close()
	if scanErr != nil {
		if closeErr != nil {
			return llmResponse{}, errors.Join(scanErr, fmt.Errorf("gemini cli response close: %w", closeErr))
		}
		return llmResponse{}, scanErr
	}
	if closeErr != nil {
		return llmResponse{}, fmt.Errorf("gemini cli response close: %w", closeErr)
	}
	if result.Text == "" && len(result.ToolCalls) == 0 {
		result.Text = "(no response)"
	}
	return result, nil
}

func geminiCLIReadStream(body io.Reader) (llmResponse, error) {
	result := llmResponse{}
	scanner := bufio.NewScanner(body)
	for scanner.Scan() {
		payload := geminiCLIExtractPayload(scanner.Text())
		if payload == "" {
			continue
		}
		var chunk geminiCLIChunk
		if err := json.Unmarshal([]byte(payload), &chunk); err != nil {
			continue
		}
		if chunk.Response == nil || len(chunk.Response.Candidates) == 0 {
			continue
		}
		updateGeminiCLIResult(&result, chunk)
	}
	return result, scanner.Err()
}

func geminiCLIExtractPayload(line string) string {
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "data:") {
		return ""
	}
	payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
	if payload == "" {
		return ""
	}
	return payload
}

func updateGeminiCLIResult(result *llmResponse, chunk geminiCLIChunk) {
	candidate := chunk.Response.Candidates[0]
	result.StopReason = candidate.FinishReason
	for _, part := range candidate.Content.Parts {
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
	if chunk.Response.UsageMetadata.TotalTokens > 0 {
		result.Usage = Usage{
			InputTokens:  chunk.Response.UsageMetadata.PromptTokens - chunk.Response.UsageMetadata.CachedTokens,
			OutputTokens: chunk.Response.UsageMetadata.CandidatesTokens + chunk.Response.UsageMetadata.ThoughtsTokens,
			TotalTokens:  chunk.Response.UsageMetadata.TotalTokens,
		}
	}
}

func geminiCLIReadError(resp *http.Response) error {
	data, readErr := io.ReadAll(resp.Body)
	closeErr := resp.Body.Close()
	if readErr != nil {
		err := fmt.Errorf("gemini cli response read: %w", readErr)
		if closeErr != nil {
			return errors.Join(err, fmt.Errorf("gemini cli response close: %w", closeErr))
		}
		return err
	}
	err := fmt.Errorf("gemini cli error: %s", string(data))
	if closeErr != nil {
		return errors.Join(err, fmt.Errorf("gemini cli response close: %w", closeErr))
	}
	return err
}
