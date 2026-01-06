package agent

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type stubTransport struct {
	handler func(*http.Request) (*http.Response, error)
}

func (s stubTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return s.handler(req)
}

func withStubClient(t *testing.T, handler func(*http.Request) (*http.Response, error)) func() {
	prev := authHTTPClient
	authHTTPClient = &http.Client{Transport: stubTransport{handler: handler}}
	return func() { authHTTPClient = prev }
}

func TestDefaultPathsWithConfigDir(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PEAKYPANES_CONFIG_DIR", dir)
	authPath, err := DefaultAuthPath()
	if err != nil {
		t.Fatalf("DefaultAuthPath error: %v", err)
	}
	if want := filepath.Join(dir, "agent", "auth.json"); authPath != want {
		t.Fatalf("auth path=%q want %q", authPath, want)
	}
	skillsDir, err := DefaultSkillsDir()
	if err != nil {
		t.Fatalf("DefaultSkillsDir error: %v", err)
	}
	if want := filepath.Join(dir, "skills"); skillsDir != want {
		t.Fatalf("skills dir=%q want %q", skillsDir, want)
	}
}

func TestProvidersList(t *testing.T) {
	providers := Providers()
	if len(providers) == 0 {
		t.Fatalf("expected providers")
	}
	found := false
	for _, p := range providers {
		if p.ID == ProviderGoogle {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected google provider")
	}
}

func TestAuthManagerSetRemoveAPIKey(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PEAKYPANES_CONFIG_DIR", dir)
	manager, err := NewAuthManager()
	if err != nil {
		t.Fatalf("NewAuthManager error: %v", err)
	}
	if manager.HasAuth(ProviderOpenAI) {
		t.Fatalf("expected no auth initially")
	}
	if err := manager.SetAPIKey(ProviderOpenAI, "sk-test"); err != nil {
		t.Fatalf("SetAPIKey error: %v", err)
	}
	if !manager.HasAuth(ProviderOpenAI) {
		t.Fatalf("expected auth after set")
	}
	if err := manager.Remove(ProviderOpenAI); err != nil {
		t.Fatalf("Remove error: %v", err)
	}
	if manager.HasAuth(ProviderOpenAI) {
		t.Fatalf("expected auth removed")
	}
}

func TestAnthropicAuthURL(t *testing.T) {
	manager, err := NewAuthManager()
	if err != nil {
		t.Fatalf("NewAuthManager error: %v", err)
	}
	url, verifier, err := manager.AnthropicAuthURL()
	if err != nil {
		t.Fatalf("AnthropicAuthURL error: %v", err)
	}
	if verifier == "" || url == "" {
		t.Fatalf("expected url and verifier")
	}
	if !bytes.Contains([]byte(url), []byte("client_id=")) {
		t.Fatalf("expected client_id in url")
	}
}

func TestCopilotStartError(t *testing.T) {
	restore := withStubClient(t, func(req *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 500, Body: io.NopCloser(bytes.NewBufferString("boom"))}, nil
	})
	defer restore()
	manager, err := NewAuthManager()
	if err != nil {
		t.Fatalf("NewAuthManager error: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, err = manager.CopilotStart(ctx, "github.com")
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestCopilotCompleteError(t *testing.T) {
	restore := withStubClient(t, func(req *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 500, Body: io.NopCloser(bytes.NewBufferString("boom"))}, nil
	})
	defer restore()
	manager, err := NewAuthManager()
	if err != nil {
		t.Fatalf("NewAuthManager error: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	err = manager.CopilotComplete(ctx, CopilotDeviceInfo{DeviceCode: "code", Interval: 1, ExpiresIn: 1}, "github.com")
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestGeminiCLIExchangeError(t *testing.T) {
	restore := withStubClient(t, func(req *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 500, Body: io.NopCloser(bytes.NewBufferString("boom"))}, nil
	})
	defer restore()
	manager, err := NewAuthManager()
	if err != nil {
		t.Fatalf("NewAuthManager error: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := manager.GeminiCLIExchange(ctx, "code", "verifier"); err == nil {
		t.Fatalf("expected error")
	}
}

func TestAntigravityExchangeError(t *testing.T) {
	restore := withStubClient(t, func(req *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 500, Body: io.NopCloser(bytes.NewBufferString("boom"))}, nil
	})
	defer restore()
	manager, err := NewAuthManager()
	if err != nil {
		t.Fatalf("NewAuthManager error: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := manager.AntigravityExchange(ctx, "code", "verifier"); err == nil {
		t.Fatalf("expected error")
	}
}

func TestCallbackServers(t *testing.T) {
	server, err := StartGeminiCLICallback()
	if err != nil {
		t.Fatalf("StartGeminiCLICallback error: %v", err)
	}
	defer func() { _ = server.Close() }()
	go func() {
		_, _ = http.Get("http://127.0.0.1:8085/oauth2callback?code=abc&state=xyz")
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	code, state, err := server.Wait(ctx)
	if err != nil {
		t.Fatalf("callback wait error: %v", err)
	}
	if code != "abc" || state != "xyz" {
		t.Fatalf("unexpected callback data: %s %s", code, state)
	}

	server2, err := StartAntigravityCallback()
	if err != nil {
		t.Fatalf("StartAntigravityCallback error: %v", err)
	}
	defer func() { _ = server2.Close() }()
	go func() {
		_, _ = http.Get("http://127.0.0.1:51121/oauth-callback?code=abc&state=xyz")
	}()
	ctx2, cancel2 := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel2()
	code2, state2, err := server2.Wait(ctx2)
	if err != nil {
		t.Fatalf("callback wait error: %v", err)
	}
	if code2 != "abc" || state2 != "xyz" {
		t.Fatalf("unexpected callback data: %s %s", code2, state2)
	}
}

func TestTouchMissingAuth(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PEAKYPANES_CONFIG_DIR", dir)
	manager, err := NewAuthManager()
	if err != nil {
		t.Fatalf("NewAuthManager error: %v", err)
	}
	if err := manager.Touch(ProviderOpenAI); err == nil {
		t.Fatalf("expected error")
	}
}

func TestAuthManagerUsesConfigDir(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PEAKYPANES_CONFIG_DIR", dir)
	manager, err := NewAuthManager()
	if err != nil {
		t.Fatalf("NewAuthManager error: %v", err)
	}
	if err := manager.SetAPIKey(ProviderOpenRouter, "key"); err != nil {
		t.Fatalf("SetAPIKey error: %v", err)
	}
	path := filepath.Join(dir, "agent", "auth.json")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected auth.json: %v", err)
	}
}

func TestAuthManagerStatus(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PEAKYPANES_CONFIG_DIR", dir)
	manager, err := NewAuthManager()
	if err != nil {
		t.Fatalf("NewAuthManager error: %v", err)
	}
	status := manager.Status(ProviderOpenAI)
	if status.HasAuth {
		t.Fatalf("expected no auth")
	}
	if err := manager.SetAPIKey(ProviderOpenAI, "sk"); err != nil {
		t.Fatalf("SetAPIKey error: %v", err)
	}
	status = manager.Status(ProviderOpenAI)
	if !status.HasAuth {
		t.Fatalf("expected auth")
	}
}

func TestGeminiAuthURL(t *testing.T) {
	manager, err := NewAuthManager()
	if err != nil {
		t.Fatalf("NewAuthManager error: %v", err)
	}
	url, verifier, err := manager.GeminiCLIAuthURL()
	if err != nil {
		t.Fatalf("GeminiCLIAuthURL error: %v", err)
	}
	if url == "" || verifier == "" {
		t.Fatalf("expected url and verifier")
	}
}

func TestAntigravityAuthURL(t *testing.T) {
	manager, err := NewAuthManager()
	if err != nil {
		t.Fatalf("NewAuthManager error: %v", err)
	}
	url, verifier, err := manager.AntigravityAuthURL()
	if err != nil {
		t.Fatalf("AntigravityAuthURL error: %v", err)
	}
	if url == "" || verifier == "" {
		t.Fatalf("expected url and verifier")
	}
	if !bytes.Contains([]byte(url), []byte("client_id=")) {
		t.Fatalf("expected client_id in url")
	}
}

func TestCopilotNormalizeDomain(t *testing.T) {
	domain, err := copilotNormalizeDomain("company.ghe.com")
	if err != nil {
		t.Fatalf("normalize error: %v", err)
	}
	if domain != "company.ghe.com" {
		t.Fatalf("expected domain, got %q", domain)
	}
	if _, err := copilotNormalizeDomain("::bad"); err == nil {
		t.Fatalf("expected error for bad domain")
	}
}

func TestProviderListIncludesOAuth(t *testing.T) {
	providers := Providers()
	foundOAuth := false
	for _, p := range providers {
		if p.SupportsOAuth {
			foundOAuth = true
			break
		}
	}
	if !foundOAuth {
		t.Fatalf("expected oauth provider")
	}
}

func TestAuthManagerProviderList(t *testing.T) {
	manager, err := NewAuthManager()
	if err != nil {
		t.Fatalf("NewAuthManager error: %v", err)
	}
	list := manager.ProviderList()
	if len(list) == 0 {
		t.Fatalf("expected provider list")
	}
}

func TestAuthManagerCopilotStartDomainDefault(t *testing.T) {
	restore := withStubClient(t, func(req *http.Request) (*http.Response, error) {
		if req.URL.Host == "github.com" && strings.HasSuffix(req.URL.Path, "/login/device/code") {
			body := `{"device_code":"dev","user_code":"USER","verification_uri":"https://github.com/login/device","interval":1,"expires_in":10}`
			return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewBufferString(body))}, nil
		}
		return &http.Response{StatusCode: 500, Body: io.NopCloser(bytes.NewBufferString("boom"))}, nil
	})
	defer restore()
	manager, err := NewAuthManager()
	if err != nil {
		t.Fatalf("NewAuthManager error: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	device, err := manager.CopilotStart(ctx, "")
	if err != nil {
		t.Fatalf("CopilotStart error: %v", err)
	}
	if device.DeviceCode == "" {
		t.Fatalf("expected device code")
	}
}
