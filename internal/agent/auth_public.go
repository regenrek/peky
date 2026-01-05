package agent

import (
	"context"
	"errors"
	"strings"
	"time"
)

type AuthManager struct {
	store *authStore
}

type AuthStatus struct {
	Provider Provider
	HasAuth  bool
	Method   string
}

type CopilotDeviceInfo struct {
	DeviceCode      string
	UserCode        string
	VerificationURI string
	Interval        int
	ExpiresIn       int
}

func NewAuthManager() (*AuthManager, error) {
	path, err := DefaultAuthPath()
	if err != nil {
		return nil, err
	}
	store, err := newAuthStore(path)
	if err != nil {
		return nil, err
	}
	return &AuthManager{store: store}, nil
}

func (a *AuthManager) HasAuth(provider Provider) bool {
	if a == nil || a.store == nil {
		return false
	}
	return a.store.hasAuth(string(provider))
}

func (a *AuthManager) Status(provider Provider) AuthStatus {
	status := AuthStatus{Provider: provider}
	if a == nil || a.store == nil {
		return status
	}
	status.HasAuth = a.store.hasAuth(string(provider))
	if !status.HasAuth {
		return status
	}
	return status
}

func (a *AuthManager) SetAPIKey(provider Provider, key string) error {
	if a == nil || a.store == nil {
		return errors.New("auth store unavailable")
	}
	return a.store.setAPIKey(string(provider), key)
}

func (a *AuthManager) Remove(provider Provider) error {
	if a == nil || a.store == nil {
		return errors.New("auth store unavailable")
	}
	return a.store.remove(string(provider))
}

func (a *AuthManager) ProviderList() []ProviderInfo {
	return Providers()
}

func (a *AuthManager) AnthropicAuthURL() (string, string, error) {
	return anthropicAuthURLWithPKCE()
}

func (a *AuthManager) AnthropicExchange(ctx context.Context, code, state, verifier string) error {
	if a == nil || a.store == nil {
		return errors.New("auth store unavailable")
	}
	cred, err := anthropicExchange(ctx, code, state, verifier)
	if err != nil {
		return err
	}
	return a.store.setOAuth(string(ProviderAnthropic), cred)
}

func (a *AuthManager) CopilotStart(ctx context.Context, domain string) (CopilotDeviceInfo, error) {
	domain = strings.TrimSpace(domain)
	if domain == "" {
		domain = "github.com"
	}
	resp, err := copilotStartDeviceFlow(ctx, domain)
	if err != nil {
		return CopilotDeviceInfo{}, err
	}
	return CopilotDeviceInfo(resp), nil
}

func (a *AuthManager) CopilotComplete(ctx context.Context, device CopilotDeviceInfo, domain string) error {
	if a == nil || a.store == nil {
		return errors.New("auth store unavailable")
	}
	if domain == "" {
		domain = "github.com"
	}
	accessToken, err := copilotPollAccessToken(ctx, domain, device.DeviceCode, device.Interval, device.ExpiresIn)
	if err != nil {
		return err
	}
	cred, err := copilotRefreshToken(ctx, accessToken, domain)
	if err != nil {
		return err
	}
	return a.store.setOAuth(string(ProviderGitHubCopilot), cred)
}

func (a *AuthManager) GeminiCLIAuthURL() (string, string, error) {
	return geminiCLIAuthURL()
}

func (a *AuthManager) GeminiCLIExchange(ctx context.Context, code, verifier string) error {
	if a == nil || a.store == nil {
		return errors.New("auth store unavailable")
	}
	cred, err := geminiCLIExchange(ctx, code, verifier)
	if err != nil {
		return err
	}
	return a.store.setOAuth(string(ProviderGoogleGeminiCLI), cred)
}

func (a *AuthManager) AntigravityAuthURL() (string, string, error) {
	return antigravityAuthURL()
}

func (a *AuthManager) AntigravityExchange(ctx context.Context, code, verifier string) error {
	if a == nil || a.store == nil {
		return errors.New("auth store unavailable")
	}
	cred, err := antigravityExchange(ctx, code, verifier)
	if err != nil {
		return err
	}
	return a.store.setOAuth(string(ProviderGoogleAntigrav), cred)
}

func (a *AuthManager) Touch(provider Provider) error {
	if a == nil || a.store == nil {
		return errors.New("auth store unavailable")
	}
	_, _, err := a.store.getAPIKey(string(provider), time.Now())
	return err
}
