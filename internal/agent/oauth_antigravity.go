package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
)

const (
	antigravityClientID     = "1071006060591-tmhssin2h21lcre235vtolojh4g403ep.apps.googleusercontent.com"
	antigravityClientSecret = "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"
	antigravityRedirect     = "http://localhost:51121/oauth-callback"
	antigravityTokenURL     = "https://oauth2.googleapis.com/token"
	antigravityAuthorizeURL = "https://accounts.google.com/o/oauth2/v2/auth"
	antigravityDefaultProj  = "rising-fact-p41fc"
)

var antigravityScopes = []string{
	"https://www.googleapis.com/auth/cloud-platform",
	"https://www.googleapis.com/auth/userinfo.email",
	"https://www.googleapis.com/auth/userinfo.profile",
	"https://www.googleapis.com/auth/cclog",
	"https://www.googleapis.com/auth/experimentsandconfigs",
}

func antigravityAuthURL() (string, string, error) {
	verifier, challenge, err := oauthPKCE()
	if err != nil {
		return "", "", err
	}
	params := url.Values{}
	params.Set("client_id", antigravityClientID)
	params.Set("response_type", "code")
	params.Set("redirect_uri", antigravityRedirect)
	params.Set("scope", strings.Join(antigravityScopes, " "))
	params.Set("code_challenge", challenge)
	params.Set("code_challenge_method", "S256")
	params.Set("state", verifier)
	params.Set("access_type", "offline")
	params.Set("prompt", "consent")
	return antigravityAuthorizeURL + "?" + params.Encode(), verifier, nil
}

func antigravityExchange(ctx context.Context, code, verifier string) (oauthCredentials, error) {
	values := url.Values{}
	values.Set("client_id", antigravityClientID)
	values.Set("client_secret", antigravityClientSecret)
	values.Set("code", code)
	values.Set("grant_type", "authorization_code")
	values.Set("redirect_uri", antigravityRedirect)
	values.Set("code_verifier", verifier)
	resp, err := oauthPostForm(ctx, antigravityTokenURL, values)
	if err != nil {
		return oauthCredentials{}, err
	}
	body, err := readResponseBody(resp)
	if err != nil {
		return oauthCredentials{}, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return oauthCredentials{}, fmt.Errorf("antigravity token exchange failed: %s", body)
	}
	var parsed geminiCLITokenResponse
	if err := json.Unmarshal([]byte(body), &parsed); err != nil {
		return oauthCredentials{}, fmt.Errorf("antigravity token parse: %w", err)
	}
	if parsed.AccessToken == "" || parsed.RefreshToken == "" {
		return oauthCredentials{}, errors.New("antigravity token response missing fields")
	}
	email, _ := googleUserEmail(ctx, parsed.AccessToken)
	projectID, err := antigravityDiscoverProject(ctx, parsed.AccessToken)
	if err != nil {
		return oauthCredentials{}, err
	}
	return oauthCredentials{
		AccessToken:  parsed.AccessToken,
		RefreshToken: parsed.RefreshToken,
		ExpiresAtMS:  oauthExpiry(parsed.ExpiresIn),
		ProjectID:    projectID,
		Email:        email,
	}, nil
}

func antigravityRefresh(ctx context.Context, refreshToken, projectID string) (oauthCredentials, error) {
	values := url.Values{}
	values.Set("client_id", antigravityClientID)
	values.Set("client_secret", antigravityClientSecret)
	values.Set("refresh_token", refreshToken)
	values.Set("grant_type", "refresh_token")
	resp, err := oauthPostForm(ctx, antigravityTokenURL, values)
	if err != nil {
		return oauthCredentials{}, err
	}
	body, err := readResponseBody(resp)
	if err != nil {
		return oauthCredentials{}, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return oauthCredentials{}, fmt.Errorf("antigravity refresh failed: %s", body)
	}
	var parsed geminiCLITokenResponse
	if err := json.Unmarshal([]byte(body), &parsed); err != nil {
		return oauthCredentials{}, fmt.Errorf("antigravity refresh parse: %w", err)
	}
	if parsed.AccessToken == "" {
		return oauthCredentials{}, errors.New("antigravity refresh missing access token")
	}
	return oauthCredentials{
		AccessToken:  parsed.AccessToken,
		RefreshToken: refreshToken,
		ExpiresAtMS:  oauthExpiry(parsed.ExpiresIn),
		ProjectID:    projectID,
	}, nil
}

func antigravityDiscoverProject(ctx context.Context, accessToken string) (string, error) {
	headers := map[string]string{
		"Authorization":     "Bearer " + accessToken,
		"Content-Type":      "application/json",
		"User-Agent":        "antigravity/1.11.5 darwin/arm64",
		"X-Goog-Api-Client": "google-cloud-sdk vscode_cloudshelleditor/0.1",
		"Client-Metadata":   `{"ideType":"IDE_UNSPECIFIED","platform":"PLATFORM_UNSPECIFIED","pluginType":"GEMINI"}`,
	}
	payload := map[string]any{
		"metadata": map[string]string{
			"ideType":    "IDE_UNSPECIFIED",
			"platform":   "PLATFORM_UNSPECIFIED",
			"pluginType": "GEMINI",
		},
	}
	endpoints := []string{"https://cloudcode-pa.googleapis.com", "https://daily-cloudcode-pa.sandbox.googleapis.com"}
	for _, endpoint := range endpoints {
		resp, err := oauthPostJSON(ctx, endpoint+"/v1internal:loadCodeAssist", payload, headers)
		if err != nil {
			continue
		}
		body, err := readResponseBody(resp)
		if err != nil {
			continue
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			continue
		}
		var parsed geminiCLILoadResponse
		if err := json.Unmarshal([]byte(body), &parsed); err == nil {
			if id := extractProjectID(parsed.CloudAIProject); id != "" {
				return id, nil
			}
		}
	}
	return antigravityDefaultProj, nil
}

func StartAntigravityCallback() (*CallbackServer, error) {
	return startCallbackServer("127.0.0.1:51121", "/oauth-callback")
}

func StartGeminiCLICallback() (*CallbackServer, error) {
	return startCallbackServer("127.0.0.1:8085", "/oauth2callback")
}
