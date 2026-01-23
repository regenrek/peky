package app

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/kballard/go-shellquote"

	"github.com/regenrek/peakypanes/internal/agent"
	"github.com/regenrek/peakypanes/internal/cli/contextpack"
	"github.com/regenrek/peakypanes/internal/cli/daemon"
	"github.com/regenrek/peakypanes/internal/cli/debug"
	"github.com/regenrek/peakypanes/internal/cli/events"
	"github.com/regenrek/peakypanes/internal/cli/help"
	"github.com/regenrek/peakypanes/internal/cli/initcfg"
	"github.com/regenrek/peakypanes/internal/cli/layouts"
	"github.com/regenrek/peakypanes/internal/cli/pane"
	"github.com/regenrek/peakypanes/internal/cli/relay"
	"github.com/regenrek/peakypanes/internal/cli/root"
	"github.com/regenrek/peakypanes/internal/cli/session"
	"github.com/regenrek/peakypanes/internal/cli/spec"
	"github.com/regenrek/peakypanes/internal/cli/version"
	"github.com/regenrek/peakypanes/internal/cli/workspace"
	"github.com/regenrek/peakypanes/internal/identity"
	"github.com/regenrek/peakypanes/internal/layout"
	"github.com/regenrek/peakypanes/internal/sessiond"
)

const (
	pekyPromptTimeout  = 2 * time.Minute
	pekyCommandTimeout = 20 * time.Second
	pekyPromptLineTTL  = 4 * time.Second
)

type pekyResultMsg struct {
	Prompt    string
	Text      string
	Usage     agent.Usage
	History   []agent.Message
	Err       error
	SetupHint string
	RunID     int64
}

type pekyToolInput struct {
	Command string `json:"command" description:"CLI command to run (omit the leading 'peky')."`
}

func (m *Model) sendPekyPrompt(text string) tea.Cmd {
	text = strings.TrimSpace(text)
	if text == "" {
		return NewInfoCmd("Enter a prompt")
	}
	if !agentFeaturesEnabled {
		return NewWarningCmd("Agent mode disabled")
	}
	if m.pekyBusy {
		return NewWarningCmd("peky is busy")
	}
	workDir, err := m.pekyWorkDir()
	if err != nil {
		return NewWarningCmd(err.Error())
	}
	cfg := m.pekyConfig().Agent
	history := append([]agent.Message(nil), m.pekyMessages...)
	contextHint := m.pekyContext()
	cliVersion := ""
	if m.client != nil {
		cliVersion = m.client.Version()
	}

	ctx, cancel := context.WithTimeout(context.Background(), pekyPromptTimeout)
	m.pekyCancel = cancel
	m.pekyRunID++
	runID := m.pekyRunID

	m.quickReplyInput.SetValue("")
	m.resetQuickReplyMenu()
	m.pekyBusy = true
	m.pekySpinnerIndex = 0
	m.pekyPromptLine = ""
	m.pekyPromptLineID++

	return tea.Batch(
		m.pekySpinnerTickCmd(),
		func() tea.Msg {
			return runPekyPrompt(ctx, runID, text, cfg, history, contextHint, workDir, cliVersion)
		},
	)
}

func runPekyPrompt(ctx context.Context, runID int64, prompt string, cfg layout.AgentConfig, history []agent.Message, contextHint, workDir, cliVersion string) tea.Msg {
	skillsDir, err := agent.DefaultSkillsDir()
	if err != nil {
		return pekyResultMsg{Prompt: prompt, Err: err, RunID: runID}
	}

	result, updated, err := agent.RunPrompt(
		ctx,
		agent.Config{
			Provider:        agent.Provider(cfg.Provider),
			Model:           cfg.Model,
			AllowedCommands: cfg.AllowedCommands,
			BlockedCommands: cfg.BlockedCommands,
			TracePath:       pekyTracePath(),
		},
		history,
		prompt,
		contextHint,
		skillsDir,
		func(ctx context.Context, call agent.ToolCall) (agent.ToolResult, error) {
			command, ok := call.Arguments["command"].(string)
			if !ok || strings.TrimSpace(command) == "" {
				return agent.ToolResult{
					ToolCallID: call.ID,
					ToolName:   call.Name,
					Content:    "missing command",
					IsError:    true,
				}, errors.New("missing command")
			}
			output, err := runPekyCommand(ctx, newPekyPolicy(cfg), pekyToolInput{Command: command}, workDir, cliVersion)
			return agent.ToolResult{
				ToolCallID: call.ID,
				ToolName:   call.Name,
				Content:    output,
				IsError:    err != nil,
			}, err
		},
	)
	if err != nil {
		return pekyResultMsg{Prompt: prompt, Err: err, SetupHint: pekySetupHint(agent.Provider(cfg.Provider)), RunID: runID}
	}
	text := strings.TrimSpace(result.Text)
	if text == "" {
		text = "(no response)"
	}
	return pekyResultMsg{
		Prompt:  prompt,
		Text:    text,
		Usage:   result.Usage,
		History: updated,
		RunID:   runID,
	}
}

func pekySetupHint(provider agent.Provider) string {
	provider = agent.Provider(strings.ToLower(strings.TrimSpace(string(provider))))
	switch provider {
	case agent.ProviderGoogle:
		return "Set GEMINI_API_KEY or GOOGLE_API_KEY."
	case agent.ProviderAnthropic:
		return "Set ANTHROPIC_API_KEY."
	case agent.ProviderOpenAI:
		return "Set OPENAI_API_KEY."
	case agent.ProviderOpenRouter:
		return "Set OPENROUTER_API_KEY."
	case agent.ProviderGitHubCopilot:
		return "Authentication required for GitHub Copilot."
	case agent.ProviderGoogleGeminiCLI:
		return "Authentication required for Gemini CLI."
	case agent.ProviderGoogleAntigrav:
		return "Authentication required for Antigravity."
	default:
		return "Configure provider credentials in your environment."
	}
}

func runPekyCommand(ctx context.Context, policy pekyPolicy, input pekyToolInput, workDir, cliVersion string) (string, error) {
	command := strings.TrimSpace(input.Command)
	if command == "" {
		return "command is required", errors.New("command is required")
	}
	if strings.ContainsAny(command, "\n\r") {
		return "command must be a single line", errors.New("command must be a single line")
	}
	tokens, err := shellquote.Split(command)
	if err != nil || len(tokens) == 0 {
		return fmt.Sprintf("invalid command: %v", err), errors.New("invalid command")
	}
	specDoc, err := loadPekySpec()
	if err != nil {
		return fmt.Sprintf("spec load failed: %v", err), err
	}
	cmdID, err := resolvePekyCommandID(specDoc, tokens)
	if err != nil {
		return err.Error(), err
	}
	if !policy.allows(cmdID) {
		return fmt.Sprintf("command %q is not allowed", cmdID), errors.New("command blocked")
	}
	args := append([]string{identity.CLIName}, tokens...)
	if !hasYesFlag(tokens) {
		args = append(args, "--yes")
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	deps := cliDeps(&stdout, &stderr, workDir, cliVersion)
	runner, err := newPekyRunner(specDoc, deps)
	if err != nil {
		return fmt.Sprintf("runner init failed: %v", err), err
	}
	ctxTimeout, cancel := context.WithTimeout(ctx, pekyCommandTimeout)
	defer cancel()
	runErr := runner.Run(ctxTimeout, args)
	output := formatPekyOutput(stdout.String(), stderr.String(), runErr)
	if runErr != nil {
		return output, runErr
	}
	return output, nil
}

func formatPekyOutput(stdout, stderr string, err error) string {
	parts := make([]string, 0, 3)
	stdout = strings.TrimSpace(stdout)
	stderr = strings.TrimSpace(stderr)
	if stdout != "" {
		parts = append(parts, stdout)
	}
	if stderr != "" {
		parts = append(parts, "stderr:\n"+stderr)
	}
	if err != nil {
		parts = append(parts, "error: "+err.Error())
	}
	if len(parts) == 0 {
		if err == nil {
			return "ok"
		}
		return err.Error()
	}
	return strings.Join(parts, "\n\n")
}

func cliDeps(stdout, stderr *bytes.Buffer, workDir, version string) root.Dependencies {
	if strings.TrimSpace(version) == "" {
		version = "dev"
	}
	return root.Dependencies{
		Version: version,
		AppName: identity.CLIName,
		WorkDir: strings.TrimSpace(workDir),
		Stdout:  stdout,
		Stderr:  stderr,
		Stdin:   strings.NewReader(""),
		Connect: sessiond.ConnectDefault,
	}
}

func (m *Model) pekyWorkDir() (string, error) {
	if m == nil {
		return "", errors.New("no selection available")
	}
	if pane := m.selectedPane(); pane != nil {
		cwd := strings.TrimSpace(pane.Cwd)
		if cwd != "" && !strings.EqualFold(cwd, "unknown") {
			return cwd, nil
		}
	}
	if session := m.selectedSession(); session != nil {
		path := strings.TrimSpace(session.Path)
		if path != "" {
			return path, nil
		}
	}
	return "", errors.New("select a pane or session with a valid path")
}

func pekyTracePath() string {
	path, err := agent.DefaultTracePath()
	if err != nil {
		return ""
	}
	return path
}

func loadPekySpec() (*spec.Spec, error) {
	specDoc, err := spec.LoadDefault()
	if err != nil {
		return nil, err
	}
	return filterPekySpec(specDoc), nil
}

func filterPekySpec(specDoc *spec.Spec) *spec.Spec {
	if specDoc == nil {
		return nil
	}
	filtered := *specDoc
	filtered.Commands = filterPekyCommands(specDoc.Commands)
	if filtered.FindByID(filtered.App.DefaultCommand) == nil {
		filtered.App.DefaultCommand = ""
	}
	return &filtered
}

func filterPekyCommands(commands []spec.Command) []spec.Command {
	if len(commands) == 0 {
		return nil
	}
	out := make([]spec.Command, 0, len(commands))
	for _, cmd := range commands {
		if isPekySkippedCommand(cmd.ID) {
			continue
		}
		copy := cmd
		copy.Subcommands = filterPekyCommands(cmd.Subcommands)
		out = append(out, copy)
	}
	return out
}

func isPekySkippedCommand(id string) bool {
	switch id {
	case "dashboard", "start", "clone", "nl", "pane.send":
		return true
	default:
		return false
	}
}

func newPekyRunner(specDoc *spec.Spec, deps root.Dependencies) (*root.Runner, error) {
	if specDoc == nil {
		return nil, errors.New("spec is nil")
	}
	reg := root.NewRegistry()
	registerPekyCommands(reg)
	return root.NewRunner(specDoc, deps, reg)
}

func registerPekyCommands(reg *root.Registry) {
	if reg == nil {
		return
	}
	daemon.Register(reg)
	initcfg.Register(reg)
	layouts.Register(reg)
	session.Register(reg)
	pane.Register(reg)
	relay.Register(reg)
	events.Register(reg)
	contextpack.Register(reg)
	debug.Register(reg)
	workspace.Register(reg)
	version.Register(reg)
	help.Register(reg)
}

func resolvePekyCommandID(specDoc *spec.Spec, tokens []string) (string, error) {
	if specDoc == nil {
		return "", errors.New("spec is nil")
	}
	cmdSpec, _ := matchCommandSpec(specDoc.Commands, tokens)
	if cmdSpec == nil {
		return "", errors.New("unknown command")
	}
	return cmdSpec.ID, nil
}

func matchCommandSpec(commands []spec.Command, tokens []string) (*spec.Command, int) {
	if len(tokens) == 0 {
		return nil, 0
	}
	head := strings.ToLower(tokens[0])
	for _, cmd := range commands {
		if matchesCommandToken(cmd, head) {
			if len(tokens) > 1 && len(cmd.Subcommands) > 0 {
				if sub, consumed := matchCommandSpec(cmd.Subcommands, tokens[1:]); sub != nil {
					return sub, consumed + 1
				}
			}
			return &cmd, 1
		}
	}
	return nil, 0
}

func matchesCommandToken(cmd spec.Command, token string) bool {
	if strings.EqualFold(cmd.Name, token) {
		return true
	}
	for _, alias := range cmd.Aliases {
		if strings.EqualFold(alias, token) {
			return true
		}
	}
	return false
}

func hasYesFlag(tokens []string) bool {
	for _, token := range tokens {
		if token == "--yes" || token == "-y" {
			return true
		}
	}
	return false
}

func (m *Model) handlePekyResult(msg pekyResultMsg) tea.Cmd {
	if msg.RunID != m.pekyRunID {
		return nil
	}
	m.pekyBusy = false
	m.pekySpinnerIndex = 0
	if m.pekyCancel != nil {
		m.pekyCancel()
		m.pekyCancel = nil
	}
	if msg.Err != nil {
		if errors.Is(msg.Err, agent.ErrAuthMissing) {
			m.setToast("peky needs authentication. Agent auth is disabled.", toastWarning)
			return nil
		}
		if errors.Is(msg.Err, context.Canceled) {
			m.setToast("peky canceled", toastInfo)
			return nil
		}
		body := strings.TrimSpace(msg.Err.Error())
		if hint := strings.TrimSpace(msg.SetupHint); hint != "" {
			body = strings.TrimSpace(body + "\n\nSetup:\n" + hint)
		}
		m.openPekyDialog("peky error", body, "esc close • ↑/↓ scroll", true)
		return nil
	}
	m.pekyMessages = append([]agent.Message(nil), msg.History...)
	m.closePekyDialog()
	m.pekyPromptLine = pekySuccessToast(msg.Text)
	m.pekyPromptLineID++
	return m.pekyPromptClearCmd(m.pekyPromptLineID)
}

func (m *Model) cancelPekyRun() {
	if !m.pekyBusy && m.pekyCancel == nil {
		return
	}
	if m.pekyCancel != nil {
		m.pekyCancel()
		m.pekyCancel = nil
	}
	m.pekyBusy = false
	m.pekySpinnerIndex = 0
	m.pekyPromptLine = ""
	m.pekyPromptLineID++
	m.pekyRunID++
	m.setToast("peky canceled", toastInfo)
}

func pekySuccessToast(text string) string {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return "Done"
	}
	trimmed = strings.ReplaceAll(trimmed, "\n", " ")
	trimmed = strings.Join(strings.Fields(trimmed), " ")
	const limit = 120
	if len(trimmed) > limit {
		const suffix = "..."
		return trimmed[:limit-len(suffix)] + suffix
	}
	return trimmed
}

func (m *Model) pekyPromptClearCmd(id int64) tea.Cmd {
	return tea.Tick(pekyPromptLineTTL, func(time.Time) tea.Msg {
		return pekyPromptClearMsg{ID: id}
	})
}

func (m *Model) handlePekyPromptClear(msg pekyPromptClearMsg) tea.Cmd {
	if msg.ID == m.pekyPromptLineID {
		m.pekyPromptLine = ""
	}
	return nil
}
