package agent

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

type llmRequest struct {
	Model        string
	SystemPrompt string
	Messages     []Message
	Tools        []ToolSpec
	ToolChoice   string
}

type llmResponse struct {
	Text       string
	ToolCalls  []ToolCall
	Usage      Usage
	StopReason string
}

type llmClient interface {
	Generate(ctx context.Context, req llmRequest) (llmResponse, error)
}

type providerConfig struct {
	Provider Provider
	Model    string
	APIKey   string
	BaseURL  string
	Headers  map[string]string
}

func buildProviderConfig(provider Provider, model string, apiKey string, cred *oauthCredentials) (providerConfig, error) {
	model = strings.TrimSpace(model)
	if model == "" {
		return providerConfig{}, errors.New("model is required")
	}
	cfg := providerConfig{Provider: provider, Model: model, APIKey: apiKey}
	switch provider {
	case ProviderOpenAI:
		cfg.BaseURL = "https://api.openai.com/v1"
	case ProviderOpenRouter:
		cfg.BaseURL = "https://openrouter.ai/api/v1"
	case ProviderGitHubCopilot:
		enterprise := ""
		if cred != nil {
			enterprise = cred.EnterpriseURL
		}
		cfg.BaseURL = copilotBaseURL(apiKey, enterprise)
	case ProviderAnthropic:
		cfg.BaseURL = "https://api.anthropic.com"
	case ProviderGoogle:
		cfg.BaseURL = "https://generativelanguage.googleapis.com"
	case ProviderGoogleGeminiCLI:
		cfg.BaseURL = "https://cloudcode-pa.googleapis.com"
	case ProviderGoogleAntigrav:
		cfg.BaseURL = "https://daily-cloudcode-pa.sandbox.googleapis.com"
	default:
		return providerConfig{}, fmt.Errorf("unsupported provider %q", provider)
	}
	return cfg, nil
}

func newLLMClient(cfg providerConfig) (llmClient, error) {
	switch cfg.Provider {
	case ProviderOpenAI, ProviderOpenRouter, ProviderGitHubCopilot:
		return newOpenAIClient(cfg), nil
	case ProviderAnthropic:
		return newAnthropicClient(cfg), nil
	case ProviderGoogle:
		return newGoogleClient(cfg), nil
	case ProviderGoogleGeminiCLI, ProviderGoogleAntigrav:
		return newGeminiCLIClient(cfg), nil
	default:
		return nil, fmt.Errorf("unsupported provider %q", cfg.Provider)
	}
}
