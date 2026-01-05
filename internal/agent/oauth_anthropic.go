package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
)

const (
	anthropicClientID = "9d1c250a-e61b-44d9-88ed-5944d1962f5e"
	anthropicAuthURL  = "https://claude.ai/oauth/authorize"
	anthropicTokenURL = "https://console.anthropic.com/v1/oauth/token"
	anthropicRedirect = "https://console.anthropic.com/oauth/code/callback"
	anthropicScopes   = "org:create_api_key user:profile user:inference"
)

type anthropicTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
}

func anthropicAuthURLWithPKCE() (string, string, error) {
	verifier, challenge, err := oauthPKCE()
	if err != nil {
		return "", "", err
	}
	params := url.Values{}
	params.Set("code", "true")
	params.Set("client_id", anthropicClientID)
	params.Set("response_type", "code")
	params.Set("redirect_uri", anthropicRedirect)
	params.Set("scope", anthropicScopes)
	params.Set("code_challenge", challenge)
	params.Set("code_challenge_method", "S256")
	params.Set("state", verifier)
	return anthropicAuthURL + "?" + params.Encode(), verifier, nil
}

func anthropicExchange(ctx context.Context, code, state, verifier string) (oauthCredentials, error) {
	payload := map[string]string{
		"grant_type":    "authorization_code",
		"client_id":     anthropicClientID,
		"code":          code,
		"state":         state,
		"redirect_uri":  anthropicRedirect,
		"code_verifier": verifier,
	}
	resp, err := oauthPostJSON(ctx, anthropicTokenURL, payload, nil)
	if err != nil {
		return oauthCredentials{}, err
	}
	body, err := readResponseBody(resp)
	if err != nil {
		return oauthCredentials{}, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return oauthCredentials{}, fmt.Errorf("anthropic token exchange failed: %s", body)
	}
	var parsed anthropicTokenResponse
	if err := json.Unmarshal([]byte(body), &parsed); err != nil {
		return oauthCredentials{}, fmt.Errorf("anthropic token parse: %w", err)
	}
	if parsed.AccessToken == "" || parsed.RefreshToken == "" {
		return oauthCredentials{}, fmt.Errorf("anthropic token response missing fields")
	}
	return oauthCredentials{
		AccessToken:  parsed.AccessToken,
		RefreshToken: parsed.RefreshToken,
		ExpiresAtMS:  oauthExpiry(parsed.ExpiresIn),
	}, nil
}

func anthropicRefresh(ctx context.Context, refreshToken string) (oauthCredentials, error) {
	payload := map[string]string{
		"grant_type":    "refresh_token",
		"client_id":     anthropicClientID,
		"refresh_token": refreshToken,
	}
	resp, err := oauthPostJSON(ctx, anthropicTokenURL, payload, nil)
	if err != nil {
		return oauthCredentials{}, err
	}
	body, err := readResponseBody(resp)
	if err != nil {
		return oauthCredentials{}, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return oauthCredentials{}, fmt.Errorf("anthropic refresh failed: %s", body)
	}
	var parsed anthropicTokenResponse
	if err := json.Unmarshal([]byte(body), &parsed); err != nil {
		return oauthCredentials{}, fmt.Errorf("anthropic refresh parse: %w", err)
	}
	if parsed.AccessToken == "" || parsed.RefreshToken == "" {
		return oauthCredentials{}, fmt.Errorf("anthropic refresh missing fields")
	}
	return oauthCredentials{
		AccessToken:  parsed.AccessToken,
		RefreshToken: parsed.RefreshToken,
		ExpiresAtMS:  oauthExpiry(parsed.ExpiresIn),
	}, nil
}
