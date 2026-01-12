package agent

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

const defaultMaxSteps = 10

var baseSystemPrompt = strings.TrimSpace(`You are peky, the peky agent.
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
	if err := validateRunPromptInput(prompt, execTool); err != nil {
		return Result{}, history, err
	}
	norm, err := normalizeRunConfig(cfg)
	if err != nil {
		return Result{}, history, err
	}
	client, err := buildLLMClientForRun(norm)
	if err != nil {
		return Result{}, history, err
	}
	systemPrompt, err := buildSystemPrompt(skillsDir, contextHint)
	if err != nil {
		return Result{}, history, err
	}
	runID := fmt.Sprintf("peky-%d", time.Now().UnixNano())
	trace, closeTrace := openTraceLogger(norm.TracePath)
	if closeTrace != nil {
		defer func() { _ = closeTrace() }()
	}
	state := runState{
		norm:         norm,
		client:       client,
		systemPrompt: systemPrompt,
		tools:        []ToolSpec{pekyToolSpec()},
		execTool:     execTool,
		trace:        trace,
		runID:        runID,
		history:      history,
		messages:     buildMessageHistory(history, prompt),
	}
	state.logStart(prompt, contextHint, len(history))
	return state.run(ctx)
}

func pekyToolSpec() ToolSpec {
	return ToolSpec{
		Name:        "peky",
		Description: "Run a peky CLI command. Provide the command without the leading 'peky'.",
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

type runState struct {
	norm         Config
	client       llmClient
	systemPrompt string
	tools        []ToolSpec
	execTool     ToolExecutor
	trace        *traceLogger
	runID        string
	history      []Message
	messages     []Message
	last         Result
}

func validateRunPromptInput(prompt string, execTool ToolExecutor) error {
	if execTool == nil {
		return errors.New("tool executor is required")
	}
	if strings.TrimSpace(prompt) == "" {
		return errors.New("prompt is required")
	}
	return nil
}

func normalizeRunConfig(cfg Config) (Config, error) {
	norm := cfg.normalized()
	if norm.Provider == ProviderUnknown {
		norm.Provider = ProviderGoogle
	}
	if norm.Model == "" {
		return Config{}, errors.New("model is required")
	}
	return norm, nil
}

func buildLLMClientForRun(norm Config) (llmClient, error) {
	apiKey, oauthCred, err := loadAuthKey(norm.Provider)
	if err != nil {
		return nil, err
	}
	providerCfg, err := buildProviderConfig(norm.Provider, norm.Model, apiKey, oauthCred)
	if err != nil {
		return nil, err
	}
	return newLLMClient(providerCfg)
}

func loadAuthKey(provider Provider) (string, *oauthCredentials, error) {
	authPath, err := DefaultAuthPath()
	if err != nil {
		return "", nil, err
	}
	store, err := newAuthStore(authPath)
	if err != nil {
		return "", nil, err
	}
	return store.getAPIKey(string(provider), time.Now())
}

func buildSystemPrompt(skillsDir, contextHint string) (string, error) {
	systemPrompt := baseSystemPrompt
	skillsPrompt, err := loadSkillsPrompt(skillsDir)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(skillsPrompt) != "" {
		systemPrompt = systemPrompt + "\n\n" + skillsPrompt
	}
	if strings.TrimSpace(contextHint) != "" {
		systemPrompt = systemPrompt + "\n\n" + strings.TrimSpace(contextHint)
	}
	return systemPrompt, nil
}

func loadSkillsPrompt(skillsDir string) (string, error) {
	if strings.TrimSpace(skillsDir) == "" {
		return "", nil
	}
	loaded, err := loadSkills(skillsDir)
	if err != nil {
		return "", err
	}
	return buildSkillsPrompt(loaded), nil
}

func buildMessageHistory(history []Message, prompt string) []Message {
	messages := append([]Message(nil), history...)
	prompt = strings.TrimSpace(prompt)
	return append(messages, NewUserMessage(prompt))
}

func openTraceLogger(path string) (*traceLogger, func() error) {
	trace, closeTrace, _ := newTraceLogger(path)
	return trace, closeTrace
}

func (s *runState) run(ctx context.Context) (Result, []Message, error) {
	for step := 0; step < defaultMaxSteps; step++ {
		resp, err := s.client.Generate(ctx, llmRequest{
			Model:        s.norm.Model,
			SystemPrompt: s.systemPrompt,
			Messages:     s.messages,
			Tools:        s.tools,
			ToolChoice:   "auto",
		})
		if err != nil {
			s.logLLMError(step, err)
			return Result{}, s.history, err
		}
		filteredCalls := filterToolCalls(s.norm.Provider, resp.ToolCalls)
		s.last = buildResult(s.norm, resp, filteredCalls)
		s.logResponse(step, resp, filteredCalls)
		s.messages = append(s.messages, NewAssistantMessage(resp.Text, filteredCalls))
		if len(filteredCalls) == 0 {
			s.last.Text = ensureResultText(s.last.Text, filteredCalls)
			s.logEnd(step, s.last, nil)
			return s.last, s.messages, nil
		}
		s.appendToolResults(ctx, step, filteredCalls)
	}
	err := fmt.Errorf("max steps reached (%d)", defaultMaxSteps)
	s.logEnd(defaultMaxSteps, s.last, err)
	return s.last, s.messages, err
}

func buildResult(cfg Config, resp llmResponse, calls []ToolCall) Result {
	return Result{
		Text:       strings.TrimSpace(resp.Text),
		Usage:      resp.Usage,
		ToolCalls:  calls,
		Provider:   cfg.Provider,
		Model:      cfg.Model,
		StopReason: resp.StopReason,
	}
}

func ensureResultText(text string, calls []ToolCall) string {
	if strings.TrimSpace(text) != "" || len(calls) > 0 {
		return text
	}
	return "(no response)"
}

func (s *runState) appendToolResults(ctx context.Context, step int, calls []ToolCall) {
	for _, call := range calls {
		s.logToolCall(step, call)
		result := s.execCall(ctx, call)
		s.logToolResult(step, result)
		s.messages = append(s.messages, NewToolResultMessage(result))
	}
}

func (s *runState) execCall(ctx context.Context, call ToolCall) ToolResult {
	result, err := s.execTool(ctx, call)
	if err == nil {
		return result
	}
	result.IsError = true
	if result.Content == "" {
		result.Content = err.Error()
	}
	return result
}

func (s *runState) logStart(prompt, contextHint string, historyCount int) {
	if s.trace == nil {
		return
	}
	s.trace.log(traceEvent{
		Time:     nowRFC3339(),
		RunID:    s.runID,
		Event:    "run_start",
		Provider: string(s.norm.Provider),
		Model:    s.norm.Model,
		Prompt:   truncateField(prompt),
		Context:  truncateField(contextHint),
		Meta: map[string]any{
			"max_steps": defaultMaxSteps,
			"history":   historyCount,
		},
	})
}

func (s *runState) logLLMError(step int, err error) {
	if s.trace == nil {
		return
	}
	s.trace.log(traceEvent{
		Time:     nowRFC3339(),
		RunID:    s.runID,
		Event:    "llm_error",
		Step:     step,
		Provider: string(s.norm.Provider),
		Model:    s.norm.Model,
		Error:    err.Error(),
	})
}

func (s *runState) logResponse(step int, resp llmResponse, calls []ToolCall) {
	if s.trace == nil {
		return
	}
	s.trace.log(traceEvent{
		Time:       nowRFC3339(),
		RunID:      s.runID,
		Event:      "llm_response",
		Step:       step,
		Provider:   string(s.norm.Provider),
		Model:      s.norm.Model,
		Text:       truncateField(resp.Text),
		StopReason: resp.StopReason,
		Usage:      resp.Usage,
		Meta: map[string]any{
			"tool_calls": len(calls),
		},
	})
}

func (s *runState) logToolCall(step int, call ToolCall) {
	if s.trace == nil {
		return
	}
	s.trace.log(traceEvent{
		Time:     nowRFC3339(),
		RunID:    s.runID,
		Event:    "tool_call",
		Step:     step,
		Provider: string(s.norm.Provider),
		Model:    s.norm.Model,
		ToolCall: &call,
	})
}

func (s *runState) logToolResult(step int, result ToolResult) {
	if s.trace == nil {
		return
	}
	captured := result
	captured.Content = truncateField(captured.Content)
	s.trace.log(traceEvent{
		Time:       nowRFC3339(),
		RunID:      s.runID,
		Event:      "tool_result",
		Step:       step,
		Provider:   string(s.norm.Provider),
		Model:      s.norm.Model,
		ToolResult: &captured,
	})
}

func (s *runState) logEnd(step int, result Result, err error) {
	if s.trace == nil {
		return
	}
	event := traceEvent{
		Time:     nowRFC3339(),
		RunID:    s.runID,
		Event:    "run_end",
		Step:     step,
		Provider: string(s.norm.Provider),
		Model:    s.norm.Model,
		Text:     truncateField(result.Text),
		Usage:    result.Usage,
	}
	if err != nil {
		event.Error = err.Error()
	}
	s.trace.log(event)
}
