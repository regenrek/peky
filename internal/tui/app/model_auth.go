package app

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/regenrek/peakypanes/internal/agent"
	"github.com/regenrek/peakypanes/internal/workspace"
)

type authFlowState struct {
	Provider agent.Provider
	Verifier string
	Callback *agent.CallbackServer
}

type slashCommandInput struct {
	Command       string
	Args          []string
	TrailingSpace bool
}

func parseSlashCommandInput(input string) (slashCommandInput, bool) {
	trimmed := strings.TrimSpace(input)
	if !strings.HasPrefix(trimmed, "/") {
		return slashCommandInput{}, false
	}
	parts := strings.Fields(trimmed)
	if len(parts) == 0 {
		return slashCommandInput{}, false
	}
	cmd := strings.TrimPrefix(parts[0], "/")
	if cmd == "" {
		return slashCommandInput{}, false
	}
	args := []string{}
	if len(parts) > 1 {
		args = parts[1:]
	}
	trailing := false
	if len(input) > 0 {
		last := input[len(input)-1]
		trailing = last == ' ' || last == '\t'
	}
	return slashCommandInput{
		Command:       strings.ToLower(cmd),
		Args:          args,
		TrailingSpace: trailing,
	}, true
}

func (m *Model) authMenuState() quickReplyMenu {
	cmd, ok := parseSlashCommandInput(m.quickReplyInput.Value())
	if !ok || cmd.Command != "auth" {
		return quickReplyMenu{}
	}
	if len(cmd.Args) == 0 {
		if !cmd.TrailingSpace {
			return quickReplyMenu{}
		}
		return authProviderMenu("")
	}
	if len(cmd.Args) == 1 {
		prefix := strings.ToLower(cmd.Args[0])
		if cmd.TrailingSpace {
			info, ok := agent.FindProviderInfo(cmd.Args[0])
			if !ok {
				return quickReplyMenu{}
			}
			return m.authMethodMenu(info, "")
		}
		return authProviderMenu(prefix)
	}
	info, ok := agent.FindProviderInfo(cmd.Args[0])
	if !ok {
		return quickReplyMenu{}
	}
	if len(cmd.Args) == 2 && !cmd.TrailingSpace {
		return m.authMethodMenu(info, strings.ToLower(cmd.Args[1]))
	}
	return quickReplyMenu{}
}

func (m *Model) modelMenuState() quickReplyMenu {
	cmd, ok := parseSlashCommandInput(m.quickReplyInput.Value())
	if !ok || cmd.Command != "model" {
		return quickReplyMenu{}
	}
	if len(cmd.Args) == 0 {
		if !cmd.TrailingSpace {
			return quickReplyMenu{}
		}
		return m.modelMenu("")
	}
	if len(cmd.Args) == 1 && !cmd.TrailingSpace {
		return m.modelMenu(strings.ToLower(cmd.Args[0]))
	}
	return quickReplyMenu{}
}

func authProviderMenu(prefix string) quickReplyMenu {
	entries := agent.Providers()
	suggestions := make([]quickReplySuggestion, 0, len(entries))
	for _, entry := range entries {
		token := string(entry.ID)
		label := entry.Name
		desc := authProviderDesc(entry)
		if prefix != "" {
			if !strings.HasPrefix(strings.ToLower(token), prefix) && !strings.HasPrefix(strings.ToLower(label), prefix) {
				continue
			}
		}
		matchLen := 0
		if strings.HasPrefix(strings.ToLower(label), prefix) {
			matchLen = len(prefix)
		}
		suggestions = append(suggestions, quickReplySuggestion{
			Text:     label,
			Value:    token,
			MatchLen: matchLen,
			Desc:     desc,
		})
	}
	if len(suggestions) == 0 {
		return quickReplyMenu{}
	}
	return quickReplyMenu{
		kind:        quickReplyMenuAuthProvider,
		prefix:      prefix,
		suggestions: suggestions,
	}
}

func (m *Model) authMethodMenu(info agent.ProviderInfo, prefix string) quickReplyMenu {
	suggestions := make([]quickReplySuggestion, 0, 3)
	if info.SupportsAPIKey {
		suggestions = append(suggestions, authMethodSuggestion("api-key", "Use an API key", prefix))
	}
	if info.SupportsOAuth {
		suggestions = append(suggestions, authMethodSuggestion("oauth", "Login via OAuth", prefix))
	}
	if m.providerHasAuth(info.ID) {
		suggestions = append(suggestions, authMethodSuggestion("logout", "Remove stored credentials", prefix))
	}
	out := make([]quickReplySuggestion, 0, len(suggestions))
	for _, suggestion := range suggestions {
		if suggestion.Text == "" {
			continue
		}
		out = append(out, suggestion)
	}
	if len(out) == 0 {
		return quickReplyMenu{}
	}
	return quickReplyMenu{
		kind:        quickReplyMenuAuthMethod,
		prefix:      prefix,
		suggestions: out,
	}
}

func authMethodSuggestion(value, desc, prefix string) quickReplySuggestion {
	label := value
	if prefix != "" && !strings.HasPrefix(strings.ToLower(label), prefix) {
		return quickReplySuggestion{}
	}
	matchLen := 0
	if strings.HasPrefix(strings.ToLower(label), prefix) {
		matchLen = len(prefix)
	}
	return quickReplySuggestion{
		Text:     label,
		Value:    value,
		MatchLen: matchLen,
		Desc:     desc,
	}
}

func (m *Model) modelMenu(prefix string) quickReplyMenu {
	cfg := m.pekyConfig().Agent
	provider := agent.Provider(strings.ToLower(strings.TrimSpace(cfg.Provider)))
	models := agent.ProviderModels(provider)
	current := strings.TrimSpace(cfg.Model)
	if current != "" && !stringSliceContains(models, current) {
		models = append([]string{current}, models...)
	}
	suggestions := make([]quickReplySuggestion, 0, len(models))
	for _, model := range models {
		if prefix != "" && !strings.HasPrefix(strings.ToLower(model), prefix) {
			continue
		}
		matchLen := 0
		if strings.HasPrefix(strings.ToLower(model), prefix) {
			matchLen = len(prefix)
		}
		suggestions = append(suggestions, quickReplySuggestion{
			Text:     model,
			Value:    model,
			MatchLen: matchLen,
			Desc:     "Model",
		})
	}
	if len(suggestions) == 0 {
		return quickReplyMenu{}
	}
	return quickReplyMenu{
		kind:        quickReplyMenuModel,
		prefix:      prefix,
		suggestions: suggestions,
	}
}

func (m *Model) applyAuthProviderCompletion() bool {
	selection, ok := m.selectedQuickReplySuggestion()
	if !ok {
		return false
	}
	value := suggestionValue(selection)
	if value == "" {
		return false
	}
	m.quickReplyInput.SetValue("/auth " + value + " ")
	m.quickReplyInput.CursorEnd()
	return true
}

func (m *Model) applyAuthMethodCompletion() bool {
	selection, ok := m.selectedQuickReplySuggestion()
	if !ok {
		return false
	}
	value := suggestionValue(selection)
	if value == "" {
		return false
	}
	cmd, ok := parseSlashCommandInput(m.quickReplyInput.Value())
	if !ok || cmd.Command != "auth" || len(cmd.Args) == 0 {
		return false
	}
	provider := cmd.Args[0]
	m.quickReplyInput.SetValue("/auth " + provider + " " + value + " ")
	m.quickReplyInput.CursorEnd()
	return true
}

func (m *Model) applyModelCompletion() bool {
	selection, ok := m.selectedQuickReplySuggestion()
	if !ok {
		return false
	}
	value := suggestionValue(selection)
	if value == "" {
		return false
	}
	m.quickReplyInput.SetValue("/model " + value + " ")
	m.quickReplyInput.CursorEnd()
	return true
}

func (m *Model) selectedQuickReplySuggestion() (quickReplySuggestion, bool) {
	menu := m.quickReplyMenuState()
	if len(menu.suggestions) == 0 {
		return quickReplySuggestion{}, false
	}
	if m.quickReplyMenuIndex < 0 || m.quickReplyMenuIndex >= len(menu.suggestions) {
		return quickReplySuggestion{}, false
	}
	return menu.suggestions[m.quickReplyMenuIndex], true
}

func suggestionValue(entry quickReplySuggestion) string {
	if strings.TrimSpace(entry.Value) != "" {
		return entry.Value
	}
	return entry.Text
}

func (m *Model) handleAuthSlashCommand(input string) quickReplyCommandOutcome {
	cmd, ok := parseSlashCommandInput(input)
	if !ok || cmd.Command != "auth" {
		return quickReplyCommandOutcome{}
	}
	if len(cmd.Args) == 0 {
		return quickReplyCommandOutcome{
			Cmd:     m.prefillQuickReplyInput("/auth "),
			Handled: true,
		}
	}
	info, ok := agent.FindProviderInfo(cmd.Args[0])
	if !ok {
		return quickReplyCommandOutcome{
			Cmd:     NewWarningCmd("Unknown provider"),
			Handled: true,
		}
	}
	providerToken := string(info.ID)
	setProvider := m.setAgentProvider(info.ID)
	if len(cmd.Args) == 1 {
		if info.SupportsAPIKey && !info.SupportsOAuth {
			return quickReplyCommandOutcome{
				Cmd:     tea.Batch(setProvider, m.prefillQuickReplyInput("/auth "+providerToken+" api-key ")),
				Handled: true,
			}
		}
		if info.SupportsOAuth && !info.SupportsAPIKey {
			return quickReplyCommandOutcome{
				Cmd:        tea.Batch(setProvider, m.startOAuthFlow(info.ID, "")),
				Handled:    true,
				ClearInput: true,
			}
		}
		return quickReplyCommandOutcome{
			Cmd:     tea.Batch(setProvider, m.prefillQuickReplyInput("/auth "+providerToken+" ")),
			Handled: true,
		}
	}
	method := strings.ToLower(cmd.Args[1])
	switch method {
	case "api-key", "apikey", "key":
		if len(cmd.Args) < 3 {
			return quickReplyCommandOutcome{
				Cmd:     tea.Batch(setProvider, m.prefillQuickReplyInput("/auth "+providerToken+" api-key ")),
				Handled: true,
			}
		}
		key := strings.TrimSpace(strings.Join(cmd.Args[2:], " "))
		if key == "" {
			return quickReplyCommandOutcome{
				Cmd:     NewWarningCmd("API key required"),
				Handled: true,
			}
		}
		return quickReplyCommandOutcome{
			Cmd:        tea.Batch(setProvider, authSetAPIKeyCmd(info.ID, key)),
			Handled:    true,
			ClearInput: true,
		}
	case "oauth":
		if info.ID == agent.ProviderAnthropic {
			if len(cmd.Args) < 3 {
				return quickReplyCommandOutcome{
					Cmd:     tea.Batch(setProvider, m.startAnthropicOAuth()),
					Handled: true,
				}
			}
			code, state, ok := splitAuthCode(cmd.Args[2])
			if !ok {
				return quickReplyCommandOutcome{
					Cmd:     NewWarningCmd("Expected code#state"),
					Handled: true,
				}
			}
			if strings.TrimSpace(m.authFlow.Verifier) == "" {
				return quickReplyCommandOutcome{
					Cmd:     NewWarningCmd("Run /auth anthropic oauth first"),
					Handled: true,
				}
			}
			return quickReplyCommandOutcome{
				Cmd:        tea.Batch(setProvider, authAnthropicExchangeCmd(code, state, m.authFlow.Verifier)),
				Handled:    true,
				ClearInput: true,
			}
		}
		domain := ""
		if len(cmd.Args) >= 3 {
			domain = strings.TrimSpace(cmd.Args[2])
		}
		return quickReplyCommandOutcome{
			Cmd:        tea.Batch(setProvider, m.startOAuthFlow(info.ID, domain)),
			Handled:    true,
			ClearInput: true,
		}
	case "logout", "remove", "signout":
		return quickReplyCommandOutcome{
			Cmd:        tea.Batch(setProvider, authRemoveCmd(info.ID)),
			Handled:    true,
			ClearInput: true,
		}
	default:
		return quickReplyCommandOutcome{
			Cmd:     NewWarningCmd("Unknown auth method"),
			Handled: true,
		}
	}
}

func (m *Model) handleModelSlashCommand(input string) quickReplyCommandOutcome {
	cmd, ok := parseSlashCommandInput(input)
	if !ok || cmd.Command != "model" {
		return quickReplyCommandOutcome{}
	}
	if len(cmd.Args) == 0 {
		return quickReplyCommandOutcome{
			Cmd:     m.prefillQuickReplyInput("/model "),
			Handled: true,
		}
	}
	model := strings.TrimSpace(strings.Join(cmd.Args, " "))
	if model == "" {
		return quickReplyCommandOutcome{
			Cmd:     NewWarningCmd("Model required"),
			Handled: true,
		}
	}
	return quickReplyCommandOutcome{
		Cmd:        m.setAgentModel(model),
		Handled:    true,
		ClearInput: true,
	}
}

func (m *Model) startAnthropicOAuth() tea.Cmd {
	manager, err := agent.NewAuthManager()
	if err != nil {
		return NewErrorCmd(err, "auth manager")
	}
	url, verifier, err := manager.AnthropicAuthURL()
	if err != nil {
		return NewErrorCmd(err, "auth url")
	}
	m.authFlow = authFlowState{Provider: agent.ProviderAnthropic, Verifier: verifier}
	m.setToast("Open URL then paste code#state: "+url, toastInfo)
	return m.prefillQuickReplyInput("/auth anthropic oauth ")
}

func (m *Model) startOAuthFlow(provider agent.Provider, domain string) tea.Cmd {
	manager, err := agent.NewAuthManager()
	if err != nil {
		return NewErrorCmd(err, "auth manager")
	}
	m.authFlow = authFlowState{Provider: provider}
	switch provider {
	case agent.ProviderGitHubCopilot:
		m.setToast("Starting Copilot device flow...", toastInfo)
		return authCopilotStartCmd(domain)
	case agent.ProviderGoogleGeminiCLI, agent.ProviderGoogleAntigrav:
		var url string
		var verifier string
		if provider == agent.ProviderGoogleGeminiCLI {
			url, verifier, err = manager.GeminiCLIAuthURL()
		} else {
			url, verifier, err = manager.AntigravityAuthURL()
		}
		if err != nil {
			return NewErrorCmd(err, "auth url")
		}
		m.authFlow.Verifier = verifier
		var server *agent.CallbackServer
		if provider == agent.ProviderGoogleGeminiCLI {
			server, err = agent.StartGeminiCLICallback()
		} else {
			server, err = agent.StartAntigravityCallback()
		}
		if err != nil {
			return NewErrorCmd(err, "oauth callback")
		}
		m.authFlow.Callback = server
		m.setToast("Open URL to finish login: "+url, toastInfo)
		return authWaitCallbackCmd(provider, server)
	default:
		return NewWarningCmd("OAuth not supported for provider")
	}
}

func splitAuthCode(raw string) (string, string, bool) {
	parts := strings.SplitN(strings.TrimSpace(raw), "#", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	code := strings.TrimSpace(parts[0])
	state := strings.TrimSpace(parts[1])
	if code == "" || state == "" {
		return "", "", false
	}
	return code, state, true
}

func authProviderDesc(info agent.ProviderInfo) string {
	parts := []string{}
	if info.SupportsAPIKey {
		parts = append(parts, "api key")
	}
	if info.SupportsOAuth {
		parts = append(parts, "oauth")
	}
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, ", ")
}

func (m *Model) providerHasAuth(provider agent.Provider) bool {
	manager, err := agent.NewAuthManager()
	if err != nil {
		return false
	}
	return manager.HasAuth(provider)
}

type authDoneMsg struct {
	Provider agent.Provider
	Err      error
	Message  string
}

type authCopilotDeviceMsg struct {
	Device agent.CopilotDeviceInfo
	Domain string
	Err    error
}

type authCallbackMsg struct {
	Provider agent.Provider
	Code     string
	State    string
	Err      error
}

func authSetAPIKeyCmd(provider agent.Provider, key string) tea.Cmd {
	return func() tea.Msg {
		manager, err := agent.NewAuthManager()
		if err == nil {
			err = manager.SetAPIKey(provider, key)
		}
		msg := "API key saved"
		if err != nil {
			msg = err.Error()
		}
		return authDoneMsg{Provider: provider, Err: err, Message: msg}
	}
}

func authRemoveCmd(provider agent.Provider) tea.Cmd {
	return func() tea.Msg {
		manager, err := agent.NewAuthManager()
		if err == nil {
			err = manager.Remove(provider)
		}
		msg := "Credentials removed"
		if err != nil {
			msg = err.Error()
		}
		return authDoneMsg{Provider: provider, Err: err, Message: msg}
	}
}

func authAnthropicExchangeCmd(code, state, verifier string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		manager, err := agent.NewAuthManager()
		if err == nil {
			err = manager.AnthropicExchange(ctx, code, state, verifier)
		}
		msg := "Anthropic connected"
		if err != nil {
			msg = err.Error()
		}
		return authDoneMsg{Provider: agent.ProviderAnthropic, Err: err, Message: msg}
	}
}

func authCopilotStartCmd(domain string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		manager, err := agent.NewAuthManager()
		if err != nil {
			return authCopilotDeviceMsg{Err: err}
		}
		device, err := manager.CopilotStart(ctx, domain)
		return authCopilotDeviceMsg{Device: device, Domain: domain, Err: err}
	}
}

func authCopilotCompleteCmd(device agent.CopilotDeviceInfo, domain string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		manager, err := agent.NewAuthManager()
		if err == nil {
			err = manager.CopilotComplete(ctx, device, domain)
		}
		msg := "Copilot connected"
		if err != nil {
			msg = err.Error()
		}
		return authDoneMsg{Provider: agent.ProviderGitHubCopilot, Err: err, Message: msg}
	}
}

func authWaitCallbackCmd(provider agent.Provider, server *agent.CallbackServer) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		code, state, err := server.Wait(ctx)
		_ = server.Close()
		return authCallbackMsg{Provider: provider, Code: code, State: state, Err: err}
	}
}

func authGeminiExchangeCmd(provider agent.Provider, code, verifier string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		manager, err := agent.NewAuthManager()
		if err == nil {
			switch provider {
			case agent.ProviderGoogleGeminiCLI:
				err = manager.GeminiCLIExchange(ctx, code, verifier)
			case agent.ProviderGoogleAntigrav:
				err = manager.AntigravityExchange(ctx, code, verifier)
			default:
				err = errors.New("unsupported provider")
			}
		}
		msg := "OAuth connected"
		if err != nil {
			msg = err.Error()
		}
		return authDoneMsg{Provider: provider, Err: err, Message: msg}
	}
}

func (m *Model) handleAuthDone(msg authDoneMsg) tea.Cmd {
	if msg.Err != nil {
		m.setToast("Auth failed: "+msg.Message, toastError)
		return nil
	}
	m.setToast(msg.Message, toastSuccess)
	return nil
}

func (m *Model) handleCopilotDevice(msg authCopilotDeviceMsg) tea.Cmd {
	if msg.Err != nil {
		m.setToast("Auth failed: "+msg.Err.Error(), toastError)
		return nil
	}
	m.authFlow = authFlowState{Provider: agent.ProviderGitHubCopilot}
	body := fmt.Sprintf("Open %s and enter code %s", msg.Device.VerificationURI, msg.Device.UserCode)
	m.setToast(body, toastInfo)
	return authCopilotCompleteCmd(msg.Device, msg.Domain)
}

func (m *Model) handleAuthCallback(msg authCallbackMsg) tea.Cmd {
	if msg.Err != nil {
		m.setToast("Auth failed: "+msg.Err.Error(), toastError)
		return nil
	}
	if msg.State != m.authFlow.Verifier {
		m.setToast("OAuth state mismatch", toastError)
		return nil
	}
	m.setToast("OAuth callback received; exchanging...", toastInfo)
	return authGeminiExchangeCmd(msg.Provider, msg.Code, m.authFlow.Verifier)
}

func (m *Model) setAgentProvider(provider agent.Provider) tea.Cmd {
	configPath, err := m.requireConfigPath()
	if err != nil {
		return NewErrorCmd(err, "agent config")
	}
	cfg, err := loadConfig(m.configPath)
	if err != nil {
		return NewErrorCmd(err, "agent config")
	}
	prevProvider := strings.ToLower(strings.TrimSpace(cfg.Agent.Provider))
	newProvider := strings.ToLower(strings.TrimSpace(string(provider)))
	cfg.Agent.Provider = newProvider
	defaultModel := agent.ProviderDefaultModel(provider)
	if cfg.Agent.Model == "" {
		if defaultModel != "" {
			cfg.Agent.Model = defaultModel
		}
	} else if prevProvider != newProvider && !agent.ProviderHasModel(provider, cfg.Agent.Model) {
		if defaultModel != "" {
			cfg.Agent.Model = defaultModel
		}
	}
	if err := workspace.SaveConfig(configPath, cfg); err != nil {
		return NewErrorCmd(err, "agent config")
	}
	m.config = cfg
	label := agent.ProviderLabel(provider)
	if label == "" {
		label = string(provider)
	}
	m.setToast("Agent provider: "+label, toastSuccess)
	return nil
}

func (m *Model) setAgentModel(model string) tea.Cmd {
	configPath, err := m.requireConfigPath()
	if err != nil {
		return NewErrorCmd(err, "agent config")
	}
	cfg, err := loadConfig(m.configPath)
	if err != nil {
		return NewErrorCmd(err, "agent config")
	}
	cfg.Agent.Model = strings.TrimSpace(model)
	if cfg.Agent.Model == "" {
		return NewWarningCmd("Model required")
	}
	if err := workspace.SaveConfig(configPath, cfg); err != nil {
		return NewErrorCmd(err, "agent config")
	}
	m.config = cfg
	m.setToast("Agent model: "+cfg.Agent.Model, toastSuccess)
	return nil
}

func stringSliceContains(values []string, target string) bool {
	target = strings.TrimSpace(target)
	for _, value := range values {
		if strings.TrimSpace(value) == target {
			return true
		}
	}
	return false
}
