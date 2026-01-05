package agent

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

const defaultMaxSteps = 10

var baseSystemPrompt = strings.TrimSpace(`You are Peky, the PeakyPanes agent.
Use the "peky" tool to run CLI commands; pass commands without the leading "peky".
Always run "--help" to discover commands and flags.
Examples: "pane add --count 3", "pane close --pane-id p-8", "session start --name work --panes 3".
Keep replies extremely short so they fit in a toast; omit extra detail.
Prefer a single tool call; after a successful tool result, respond and stop.
Use pane run for commands; do not use pane send.
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
	runID := fmt.Sprintf("peky-%d", time.Now().UnixNano())
	trace, closeTrace, _ := newTraceLogger(norm.TracePath)
	if closeTrace != nil {
		defer func() { _ = closeTrace() }()
	}
	if trace != nil {
		trace.log(traceEvent{
			Time:     nowRFC3339(),
			RunID:    runID,
			Event:    "run_start",
			Provider: string(norm.Provider),
			Model:    norm.Model,
			Prompt:   truncateField(prompt),
			Context:  truncateField(contextHint),
			Meta: map[string]any{
				"max_steps": defaultMaxSteps,
				"history":   len(history),
			},
		})
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
			if trace != nil {
				trace.log(traceEvent{
					Time:     nowRFC3339(),
					RunID:    runID,
					Event:    "llm_error",
					Step:     step,
					Provider: string(norm.Provider),
					Model:    norm.Model,
					Error:    err.Error(),
				})
			}
			return Result{}, history, err
		}
		filteredCalls := filterToolCalls(norm.Provider, resp.ToolCalls)
		last = Result{
			Text:       strings.TrimSpace(resp.Text),
			Usage:      resp.Usage,
			ToolCalls:  filteredCalls,
			Provider:   norm.Provider,
			Model:      norm.Model,
			StopReason: resp.StopReason,
		}
		if trace != nil {
			trace.log(traceEvent{
				Time:       nowRFC3339(),
				RunID:      runID,
				Event:      "llm_response",
				Step:       step,
				Provider:   string(norm.Provider),
				Model:      norm.Model,
				Text:       truncateField(resp.Text),
				StopReason: resp.StopReason,
				Usage:      resp.Usage,
				Meta: map[string]any{
					"tool_calls": len(filteredCalls),
				},
			})
		}
		messages = append(messages, NewAssistantMessage(resp.Text, filteredCalls))
		if len(filteredCalls) == 0 {
			if last.Text == "" {
				last.Text = "(no response)"
			}
			if trace != nil {
				trace.log(traceEvent{
					Time:     nowRFC3339(),
					RunID:    runID,
					Event:    "run_end",
					Step:     step,
					Provider: string(norm.Provider),
					Model:    norm.Model,
					Text:     truncateField(last.Text),
					Usage:    resp.Usage,
				})
			}
			return last, messages, nil
		}
		for _, call := range filteredCalls {
			if trace != nil {
				trace.log(traceEvent{
					Time:     nowRFC3339(),
					RunID:    runID,
					Event:    "tool_call",
					Step:     step,
					Provider: string(norm.Provider),
					Model:    norm.Model,
					ToolCall: &call,
				})
			}
			result, err := execTool(ctx, call)
			if err != nil {
				result.IsError = true
				if result.Content == "" {
					result.Content = err.Error()
				}
			}
			if trace != nil {
				captured := result
				captured.Content = truncateField(captured.Content)
				trace.log(traceEvent{
					Time:       nowRFC3339(),
					RunID:      runID,
					Event:      "tool_result",
					Step:       step,
					Provider:   string(norm.Provider),
					Model:      norm.Model,
					ToolResult: &captured,
				})
			}
			messages = append(messages, NewToolResultMessage(result))
		}
	}
	err = fmt.Errorf("max steps reached (%d)", defaultMaxSteps)
	if trace != nil {
		trace.log(traceEvent{
			Time:     nowRFC3339(),
			RunID:    runID,
			Event:    "run_end",
			Step:     defaultMaxSteps,
			Provider: string(norm.Provider),
			Model:    norm.Model,
			Error:    err.Error(),
			Usage:    last.Usage,
		})
	}
	return last, messages, err
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
