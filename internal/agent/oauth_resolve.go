package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

func resolveOAuthAPIKey(provider string, cred oauthCredentials, now time.Time) (string, *oauthCredentials, error) {
	provider = string(Provider(strings.ToLower(strings.TrimSpace(provider))))
	if provider == "" {
		return "", nil, errors.New("provider is required")
	}
	if cred.AccessToken == "" && cred.RefreshToken == "" {
		return "", nil, errors.New("oauth credentials missing")
	}
	if !cred.expired(now.UnixMilli()) {
		key, err := oauthAPIKeyForProvider(provider, cred)
		return key, nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	var updated oauthCredentials
	var err error
	switch Provider(provider) {
	case ProviderAnthropic:
		updated, err = anthropicRefresh(ctx, cred.RefreshToken)
	case ProviderGitHubCopilot:
		updated, err = copilotRefreshToken(ctx, cred.RefreshToken, cred.EnterpriseURL)
	case ProviderGoogleGeminiCLI:
		if cred.ProjectID == "" {
			return "", nil, errors.New("google gemini cli projectId missing")
		}
		updated, err = geminiCLIRefresh(ctx, cred.RefreshToken, cred.ProjectID)
	case ProviderGoogleAntigrav:
		if cred.ProjectID == "" {
			return "", nil, errors.New("antigravity projectId missing")
		}
		updated, err = antigravityRefresh(ctx, cred.RefreshToken, cred.ProjectID)
	default:
		return "", nil, fmt.Errorf("unsupported oauth provider %q", provider)
	}
	if err != nil {
		return "", nil, err
	}
	if updated.ProjectID == "" {
		updated.ProjectID = cred.ProjectID
	}
	if updated.EnterpriseURL == "" {
		updated.EnterpriseURL = cred.EnterpriseURL
	}
	if updated.Email == "" {
		updated.Email = cred.Email
	}
	apiKey, err := oauthAPIKeyForProvider(provider, updated)
	if err != nil {
		return "", nil, err
	}
	return apiKey, &updated, nil
}

func oauthAPIKeyForProvider(provider string, cred oauthCredentials) (string, error) {
	switch Provider(provider) {
	case ProviderGoogleGeminiCLI, ProviderGoogleAntigrav:
		if cred.AccessToken == "" || cred.ProjectID == "" {
			return "", errors.New("missing oauth token or projectId")
		}
		payload := map[string]string{
			"token":     cred.AccessToken,
			"projectId": cred.ProjectID,
		}
		blob, err := json.Marshal(payload)
		if err != nil {
			return "", fmt.Errorf("encode oauth payload: %w", err)
		}
		return string(blob), nil
	default:
		if cred.AccessToken == "" {
			return "", errors.New("missing oauth access token")
		}
		return cred.AccessToken, nil
	}
}
