package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	geminiCLIClientID     = "681255809395-oo8ft2oprdnp9e3aqf6av3hmdib135j.apps.googleusercontent.com"
	geminiCLIClientSecret = "GOCSPX-4uHgMPm-1o7Sk-gvV6Cu5clXFsl"
	geminiCLIRedirect     = "http://localhost:8085/oauth2callback"
	geminiCLITokenURL     = "https://oauth2.googleapis.com/token"
	geminiCLIAuthorizeURL = "https://accounts.google.com/o/oauth2/v2/auth"
	geminiCLICodeAssist   = "https://cloudcode-pa.googleapis.com"
)

var geminiCLIScopes = []string{
	"https://www.googleapis.com/auth/cloud-platform",
	"https://www.googleapis.com/auth/userinfo.email",
	"https://www.googleapis.com/auth/userinfo.profile",
}

type geminiCLITokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
}

type geminiCLILoadResponse struct {
	CloudAIProject any `json:"cloudaicompanionProject"`
	AllowedTiers   []struct {
		ID        string `json:"id"`
		IsDefault bool   `json:"isDefault"`
	} `json:"allowedTiers"`
}

type geminiCLIOnboardResponse struct {
	Done     bool `json:"done"`
	Response struct {
		Project struct {
			ID string `json:"id"`
		} `json:"cloudaicompanionProject"`
	} `json:"response"`
}

func geminiCLIAuthURL() (string, string, error) {
	verifier, challenge, err := oauthPKCE()
	if err != nil {
		return "", "", err
	}
	params := url.Values{}
	params.Set("client_id", strings.ReplaceAll(geminiCLIClientID, " ", ""))
	params.Set("response_type", "code")
	params.Set("redirect_uri", geminiCLIRedirect)
	params.Set("scope", strings.Join(geminiCLIScopes, " "))
	params.Set("code_challenge", challenge)
	params.Set("code_challenge_method", "S256")
	params.Set("state", verifier)
	params.Set("access_type", "offline")
	params.Set("prompt", "consent")
	return geminiCLIAuthorizeURL + "?" + params.Encode(), verifier, nil
}

func geminiCLIExchange(ctx context.Context, code, verifier string) (oauthCredentials, error) {
	values := url.Values{}
	values.Set("client_id", strings.ReplaceAll(geminiCLIClientID, " ", ""))
	values.Set("client_secret", geminiCLIClientSecret)
	values.Set("code", code)
	values.Set("grant_type", "authorization_code")
	values.Set("redirect_uri", geminiCLIRedirect)
	values.Set("code_verifier", verifier)
	resp, err := oauthPostForm(ctx, geminiCLITokenURL, values)
	if err != nil {
		return oauthCredentials{}, err
	}
	body, err := readResponseBody(resp)
	if err != nil {
		return oauthCredentials{}, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return oauthCredentials{}, fmt.Errorf("gemini cli token exchange failed: %s", body)
	}
	var parsed geminiCLITokenResponse
	if err := json.Unmarshal([]byte(body), &parsed); err != nil {
		return oauthCredentials{}, fmt.Errorf("gemini cli token parse: %w", err)
	}
	if parsed.AccessToken == "" || parsed.RefreshToken == "" {
		return oauthCredentials{}, errors.New("gemini cli token response missing fields")
	}
	email, _ := googleUserEmail(ctx, parsed.AccessToken)
	projectID, err := geminiCLIDiscoverProject(ctx, parsed.AccessToken)
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

func geminiCLIRefresh(ctx context.Context, refreshToken, projectID string) (oauthCredentials, error) {
	values := url.Values{}
	values.Set("client_id", strings.ReplaceAll(geminiCLIClientID, " ", ""))
	values.Set("client_secret", geminiCLIClientSecret)
	values.Set("refresh_token", refreshToken)
	values.Set("grant_type", "refresh_token")
	resp, err := oauthPostForm(ctx, geminiCLITokenURL, values)
	if err != nil {
		return oauthCredentials{}, err
	}
	body, err := readResponseBody(resp)
	if err != nil {
		return oauthCredentials{}, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return oauthCredentials{}, fmt.Errorf("gemini cli refresh failed: %s", body)
	}
	var parsed geminiCLITokenResponse
	if err := json.Unmarshal([]byte(body), &parsed); err != nil {
		return oauthCredentials{}, fmt.Errorf("gemini cli refresh parse: %w", err)
	}
	if parsed.AccessToken == "" {
		return oauthCredentials{}, errors.New("gemini cli refresh missing access token")
	}
	return oauthCredentials{
		AccessToken:  parsed.AccessToken,
		RefreshToken: refreshToken,
		ExpiresAtMS:  oauthExpiry(parsed.ExpiresIn),
		ProjectID:    projectID,
	}, nil
}

func geminiCLIDiscoverProject(ctx context.Context, accessToken string) (string, error) {
	payload := map[string]any{
		"metadata": map[string]string{
			"ideType":    "IDE_UNSPECIFIED",
			"platform":   "PLATFORM_UNSPECIFIED",
			"pluginType": "GEMINI",
		},
	}
	resp, err := oauthPostJSON(ctx, geminiCLICodeAssist+"/v1internal:loadCodeAssist", payload, map[string]string{
		"Authorization":     "Bearer " + accessToken,
		"Content-Type":      "application/json",
		"User-Agent":        "google-api-nodejs-client/9.15.1",
		"X-Goog-Api-Client": "gl-node/22.17.0",
	})
	if err != nil {
		return "", err
	}
	body, err := readResponseBody(resp)
	if err != nil {
		return "", err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("code assist load failed: %s", body)
	}
	var parsed geminiCLILoadResponse
	if err := json.Unmarshal([]byte(body), &parsed); err != nil {
		return "", fmt.Errorf("code assist parse: %w", err)
	}
	if id := extractProjectID(parsed.CloudAIProject); id != "" {
		return id, nil
	}
	// onboard
	tierID := "FREE"
	for _, tier := range parsed.AllowedTiers {
		if tier.IsDefault {
			tierID = tier.ID
			break
		}
		if tierID == "FREE" && tier.ID != "" {
			tierID = tier.ID
		}
	}
	for attempt := 0; attempt < 10; attempt++ {
		resp, err := oauthPostJSON(ctx, geminiCLICodeAssist+"/v1internal:onboardUser", map[string]any{
			"tierId": tierID,
			"metadata": map[string]string{
				"ideType":    "IDE_UNSPECIFIED",
				"platform":   "PLATFORM_UNSPECIFIED",
				"pluginType": "GEMINI",
			},
		}, map[string]string{
			"Authorization":     "Bearer " + accessToken,
			"Content-Type":      "application/json",
			"User-Agent":        "google-api-nodejs-client/9.15.1",
			"X-Goog-Api-Client": "gl-node/22.17.0",
		})
		if err != nil {
			return "", err
		}
		body, err := readResponseBody(resp)
		if err != nil {
			return "", err
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			continue
		}
		var onboard geminiCLIOnboardResponse
		if err := json.Unmarshal([]byte(body), &onboard); err == nil {
			if onboard.Done && onboard.Response.Project.ID != "" {
				return onboard.Response.Project.ID, nil
			}
		}
		if attempt < 9 {
			time.Sleep(3 * time.Second)
		}
	}
	return "", errors.New("code assist project provisioning failed")
}

func extractProjectID(raw any) string {
	switch v := raw.(type) {
	case string:
		return strings.TrimSpace(v)
	case map[string]any:
		if id, ok := v["id"].(string); ok {
			return strings.TrimSpace(id)
		}
	}
	return ""
}

func googleUserEmail(ctx context.Context, accessToken string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://www.googleapis.com/oauth2/v1/userinfo?alt=json", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	resp, err := authHTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	body, err := readResponseBody(resp)
	if err != nil {
		return "", err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", nil
	}
	var parsed struct {
		Email string `json:"email"`
	}
	if err := json.Unmarshal([]byte(body), &parsed); err != nil {
		return "", nil
	}
	return strings.TrimSpace(parsed.Email), nil
}
