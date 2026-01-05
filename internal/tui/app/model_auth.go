package app

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/regenrek/peakypanes/internal/agent"
	"github.com/regenrek/peakypanes/internal/tui/picker"
)

type authMode int

type authPromptKind int

const (
	authModeAPIKey authMode = iota
	authModeOAuth
	authModeLogout
)

const (
	authPromptAPIKey authPromptKind = iota
	authPromptAnthropicCode
	authPromptCopilotDomain
)

type authFlowState struct {
	Provider      agent.Provider
	Mode          authMode
	Prompt        authPromptKind
	Verifier      string
	CopilotDomain string
	CopilotDevice agent.CopilotDeviceInfo
	Callback      *agent.CallbackServer
}

type authProviderItem struct {
	Provider agent.Provider
	Name     string
	Desc     string
}

func (a authProviderItem) Title() string       { return a.Name }
func (a authProviderItem) Description() string { return a.Desc }
func (a authProviderItem) FilterValue() string { return strings.ToLower(a.Name + " " + a.Desc) }

type authMethodItem struct {
	Mode authMode
	Name string
	Desc string
}

func (a authMethodItem) Title() string       { return a.Name }
func (a authMethodItem) Description() string { return a.Desc }
func (a authMethodItem) FilterValue() string { return strings.ToLower(a.Name + " " + a.Desc) }

func (m *Model) setupAuthPickers() {
	m.authProviderPicker = picker.NewDialogMenu()
	m.authProviderPicker.Title = "Auth Providers"
	m.authProviderPicker.SetFilteringEnabled(true)

	m.authMethodPicker = picker.NewDialogMenu()
	m.authMethodPicker.Title = "Auth Method"
	m.authMethodPicker.SetFilteringEnabled(false)
	m.authPickersReady = true
}

func (m *Model) ensureAuthPickers() {
	if !m.authPickersReady {
		m.setupAuthPickers()
	}
}

func (m *Model) openAuthProviderPicker() tea.Cmd {
	m.ensureAuthPickers()
	providers := agent.Providers()
	items := make([]list.Item, 0, len(providers))
	for _, p := range providers {
		desc := []string{}
		if p.SupportsAPIKey {
			desc = append(desc, "api key")
		}
		if p.SupportsOAuth {
			desc = append(desc, "oauth")
		}
		items = append(items, authProviderItem{
			Provider: p.ID,
			Name:     p.Name,
			Desc:     strings.Join(desc, ", "),
		})
	}
	m.authProviderPicker.SetItems(items)
	m.authProviderPicker.ResetFilter()
	m.setState(StateAuthProviderPicker)
	return nil
}

func (m *Model) openAuthMethodPicker(provider agent.Provider) {
	m.ensureAuthPickers()
	items := []list.Item{}
	manager, err := agent.NewAuthManager()
	if err != nil {
		m.setToast("Auth error: "+err.Error(), toastError)
		m.setState(StateDashboard)
		return
	}
	var info *agent.ProviderInfo
	for _, entry := range manager.ProviderList() {
		if entry.ID == provider {
			copy := entry
			info = &copy
			break
		}
	}
	if info == nil {
		m.setToast("Unknown provider", toastWarning)
		m.setState(StateDashboard)
		return
	}
	if info.SupportsAPIKey {
		items = append(items, authMethodItem{Mode: authModeAPIKey, Name: "API key", Desc: "Store an API key"})
	}
	if info.SupportsOAuth {
		items = append(items, authMethodItem{Mode: authModeOAuth, Name: "OAuth login", Desc: "Authenticate in browser"})
	}
	if manager.HasAuth(provider) {
		items = append(items, authMethodItem{Mode: authModeLogout, Name: "Logout", Desc: "Remove stored credentials"})
	}
	m.authMethodPicker.SetItems(items)
	m.setState(StateAuthMethodPicker)
}

func (m *Model) openAuthPrompt(title, note, placeholder string, kind authPromptKind) {
	input := textinput.New()
	input.Prompt = ""
	input.Placeholder = placeholder
	input.CharLimit = 200
	input.Width = 50
	input.Focus()
	m.authPromptInput = input
	m.authPromptTitle = title
	m.authPromptNote = note
	m.authFlow.Prompt = kind
	m.setState(StateAuthPrompt)
}

func (m *Model) openAuthProgress(title, body, footer string) {
	m.authProgressTitle = title
	m.authProgressBody = body
	m.authProgressFooter = footer
	m.setState(StateAuthProgress)
}

func (m *Model) updateAuthProviderPicker(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "esc" {
		m.setState(StateDashboard)
		return m, nil
	}
	if msg.String() == "enter" {
		if item, ok := m.authProviderPicker.SelectedItem().(authProviderItem); ok {
			m.authFlow = authFlowState{Provider: item.Provider}
			m.openAuthMethodPicker(item.Provider)
			return m, nil
		}
		m.setState(StateDashboard)
		return m, nil
	}
	var cmd tea.Cmd
	m.authProviderPicker, cmd = m.authProviderPicker.Update(msg)
	return m, cmd
}

func (m *Model) updateAuthMethodPicker(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		return m, m.openAuthProviderPicker()
	case "enter":
		item, ok := m.authMethodPicker.SelectedItem().(authMethodItem)
		if !ok {
			m.setState(StateDashboard)
			return m, nil
		}
		m.authFlow.Mode = item.Mode
		provider := m.authFlow.Provider
		switch item.Mode {
		case authModeAPIKey:
			m.openAuthPrompt("API Key", "Paste the API key for "+string(provider), "api-key", authPromptAPIKey)
			return m, nil
		case authModeLogout:
			return m, authRemoveCmd(provider)
		case authModeOAuth:
			return m, m.startOAuthFlow(provider)
		}
	}
	var cmd tea.Cmd
	m.authMethodPicker, cmd = m.authMethodPicker.Update(msg)
	return m, cmd
}

func (m *Model) startOAuthFlow(provider agent.Provider) tea.Cmd {
	manager, err := agent.NewAuthManager()
	if err != nil {
		m.setToast("Auth error: "+err.Error(), toastError)
		m.setState(StateDashboard)
		return nil
	}
	switch provider {
	case agent.ProviderAnthropic:
		url, verifier, err := manager.AnthropicAuthURL()
		if err != nil {
			m.setToast("Auth error: "+err.Error(), toastError)
			m.setState(StateDashboard)
			return nil
		}
		m.authFlow.Verifier = verifier
		note := "Open this URL in your browser, authorize, then paste the code#state:\n" + url
		m.openAuthPrompt("Anthropic OAuth", note, "code#state", authPromptAnthropicCode)
		return nil
	case agent.ProviderGitHubCopilot:
		m.openAuthPrompt("GitHub Copilot", "Enter enterprise domain or leave blank for github.com", "company.ghe.com", authPromptCopilotDomain)
		return nil
	case agent.ProviderGoogleGeminiCLI, agent.ProviderGoogleAntigrav:
		var url string
		var verifier string
		if provider == agent.ProviderGoogleGeminiCLI {
			url, verifier, err = manager.GeminiCLIAuthURL()
		} else {
			url, verifier, err = manager.AntigravityAuthURL()
		}
		if err != nil {
			m.setToast("Auth error: "+err.Error(), toastError)
			m.setState(StateDashboard)
			return nil
		}
		m.authFlow.Verifier = verifier
		var server *agent.CallbackServer
		if provider == agent.ProviderGoogleGeminiCLI {
			server, err = agent.StartGeminiCLICallback()
		} else {
			server, err = agent.StartAntigravityCallback()
		}
		if err != nil {
			m.setToast("Auth error: "+err.Error(), toastError)
			m.setState(StateDashboard)
			return nil
		}
		m.authFlow.Callback = server
		m.openAuthProgress("OAuth", "Open this URL and finish login:\n"+url+"\n\nWaiting for callback...", "esc cancel")
		return authWaitCallbackCmd(provider, server)
	default:
		m.setToast("OAuth not supported for provider", toastWarning)
		m.setState(StateDashboard)
		return nil
	}
}

func (m *Model) updateAuthPrompt(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.setState(StateDashboard)
		return m, nil
	case "enter":
		return m, m.handleAuthPrompt()
	}
	var cmd tea.Cmd
	m.authPromptInput, cmd = m.authPromptInput.Update(msg)
	return m, cmd
}

func (m *Model) handleAuthPrompt() tea.Cmd {
	value := strings.TrimSpace(m.authPromptInput.Value())
	provider := m.authFlow.Provider
	switch m.authFlow.Prompt {
	case authPromptAPIKey:
		if value == "" {
			m.setToast("API key required", toastWarning)
			return nil
		}
		m.openAuthProgress("Auth", "Saving API key...", "esc close")
		return authSetAPIKeyCmd(provider, value)
	case authPromptAnthropicCode:
		parts := strings.SplitN(value, "#", 2)
		if len(parts) != 2 {
			m.setToast("Expected code#state", toastWarning)
			return nil
		}
		code := strings.TrimSpace(parts[0])
		state := strings.TrimSpace(parts[1])
		if code == "" || state == "" {
			m.setToast("code#state required", toastWarning)
			return nil
		}
		m.openAuthProgress("Anthropic OAuth", "Exchanging code...", "esc close")
		return authAnthropicExchangeCmd(code, state, m.authFlow.Verifier)
	case authPromptCopilotDomain:
		m.authFlow.CopilotDomain = value
		m.openAuthProgress("GitHub Copilot", "Starting device flow...", "esc close")
		return authCopilotStartCmd(value)
	default:
		return nil
	}
}

func (m *Model) updateAuthProgress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "esc" {
		if m.authFlow.Callback != nil {
			_ = m.authFlow.Callback.Close()
			m.authFlow.Callback = nil
		}
		m.setState(StateDashboard)
		return m, nil
	}
	return m, nil
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
		m.setState(StateDashboard)
		return nil
	}
	m.setToast(msg.Message, toastSuccess)
	m.setState(StateDashboard)
	return nil
}

func (m *Model) handleCopilotDevice(msg authCopilotDeviceMsg) tea.Cmd {
	if msg.Err != nil {
		m.setToast("Auth failed: "+msg.Err.Error(), toastError)
		m.setState(StateDashboard)
		return nil
	}
	m.authFlow.CopilotDevice = msg.Device
	body := fmt.Sprintf("Open %s and enter code: %s\n\nWaiting for authorization...", msg.Device.VerificationURI, msg.Device.UserCode)
	m.openAuthProgress("GitHub Copilot", body, "esc cancel")
	return authCopilotCompleteCmd(msg.Device, msg.Domain)
}

func (m *Model) handleAuthCallback(msg authCallbackMsg) tea.Cmd {
	if msg.Err != nil {
		m.setToast("Auth failed: "+msg.Err.Error(), toastError)
		m.setState(StateDashboard)
		return nil
	}
	if msg.State != m.authFlow.Verifier {
		m.setToast("OAuth state mismatch", toastError)
		m.setState(StateDashboard)
		return nil
	}
	m.openAuthProgress("OAuth", "Exchanging code...", "esc close")
	return authGeminiExchangeCmd(msg.Provider, msg.Code, m.authFlow.Verifier)
}
