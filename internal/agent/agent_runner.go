package agent

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

const defaultMaxSteps = 6

var baseSystemPrompt = strings.TrimSpace(`You are Peky, the PeakyPanes agent.
Use the "peky" tool to run CLI commands (without the leading "peky").
Never run daemon commands or modify the daemon.
Use @file tokens as references only; do not assume file contents.
Ask a clarifying question when a target (pane/session) is ambiguous.`)

type ToolExecutor func(ctx context.Context, call ToolCall) (ToolResult, error)

func RunPrompt(
	ctx context.Context,
	cfg Config,
	history []Message,
	prompt string,
	contextHint string,
	skillsDir string,
	execTool ToolExecutor,
) (Result, []Message, error) {
	if execTool == nil {
		return Result{}, history, errors.New("tool executor is required")
	}
	if strings.TrimSpace(prompt) == "" {
		return Result{}, history, errors.New("prompt is required")
	}
	norm := cfg.normalized()
	if norm.Provider == ProviderUnknown {
		norm.Provider = ProviderGoogle
	}
	if norm.Model == "" {
		return Result{}, history, errors.New("model is required")
	}
	authPath, err := DefaultAuthPath()
	if err != nil {
		return Result{}, history, err
	}
	store, err := newAuthStore(authPath)
	if err != nil {
		return Result{}, history, err
	}
	apiKey, oauthCred, err := store.getAPIKey(string(norm.Provider), time.Now())
	if err != nil {
		return Result{}, history, err
	}
	providerCfg, err := buildProviderConfig(norm.Provider, norm.Model, apiKey, oauthCred)
	if err != nil {
		return Result{}, history, err
	}
	client, err := newLLMClient(providerCfg)
	if err != nil {
		return Result{}, history, err
	}
	skillsPrompt := ""
	if skillsDir != "" {
		loaded, err := loadSkills(skillsDir)
		if err != nil {
			return Result{}, history, err
		}
		skillsPrompt = buildSkillsPrompt(loaded)
	}
	systemPrompt := baseSystemPrompt
	if strings.TrimSpace(skillsPrompt) != "" {
		systemPrompt = systemPrompt + "\n\n" + skillsPrompt
	}
	if strings.TrimSpace(contextHint) != "" {
		systemPrompt = systemPrompt + "\n\n" + strings.TrimSpace(contextHint)
	}
	messages := append([]Message(nil), history...)
	prompt = strings.TrimSpace(prompt)
	messages = append(messages, NewUserMessage(prompt))

	tools := []ToolSpec{pekyToolSpec()}

	var last Result
	for step := 0; step < defaultMaxSteps; step++ {
		resp, err := client.Generate(ctx, llmRequest{
			Model:        norm.Model,
			SystemPrompt: systemPrompt,
			Messages:     messages,
			Tools:        tools,
			ToolChoice:   "auto",
		})
		if err != nil {
			return Result{}, history, err
		}
		last = Result{
			Text:       strings.TrimSpace(resp.Text),
			Usage:      resp.Usage,
			ToolCalls:  resp.ToolCalls,
			Provider:   norm.Provider,
			Model:      norm.Model,
			StopReason: resp.StopReason,
		}
		messages = append(messages, NewAssistantMessage(resp.Text, resp.ToolCalls))
		if len(resp.ToolCalls) == 0 {
			if last.Text == "" {
				last.Text = "(no response)"
			}
			return last, messages, nil
		}
		for _, call := range resp.ToolCalls {
			result, err := execTool(ctx, call)
			if err != nil {
				result.IsError = true
				if result.Content == "" {
					result.Content = err.Error()
				}
			}
			messages = append(messages, NewToolResultMessage(result))
		}
	}
	return last, messages, fmt.Errorf("max steps reached (%d)", defaultMaxSteps)
}

func pekyToolSpec() ToolSpec {
	return ToolSpec{
		Name:        "peky",
		Description: "Run a PeakyPanes CLI command. Provide the command without the leading 'peky'.",
		Schema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"command": map[string]any{
					"type":        "string",
					"description": "CLI command to run (omit the leading 'peky').",
				},
			},
			"required": []string{"command"},
		},
	}
}
