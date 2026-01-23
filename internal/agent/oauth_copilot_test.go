package agent

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestCopilotBaseURL(t *testing.T) {
	if got := copilotBaseURL("proxy-ep=proxy.example.com", ""); got != "https://api.example.com" {
		t.Fatalf("proxy url=%q", got)
	}
	if got := copilotBaseURL("", "corp.example"); got != "https://copilot-api.corp.example" {
		t.Fatalf("enterprise url=%q", got)
	}
	if got := copilotBaseURL("", ""); got != "https://api.individual.githubcopilot.com" {
		t.Fatalf("default url=%q", got)
	}
}

func TestCopilotRefreshToken(t *testing.T) {
	prev := authHTTPClient
	authHTTPClient = &http.Client{Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		body := `{"token":"abc","expires_at":123}`
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body))}, nil
	})}
	t.Cleanup(func() { authHTTPClient = prev })

	cred, err := copilotRefreshToken(context.Background(), "refresh", "")
	if err != nil {
		t.Fatalf("copilotRefreshToken error: %v", err)
	}
	if cred.AccessToken != "abc" || cred.RefreshToken != "refresh" || cred.ExpiresAtMS == 0 {
		t.Fatalf("cred=%#v", cred)
	}
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
