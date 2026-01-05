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
	providerID, err := normalizeOAuthProvider(provider)
	if err != nil {
		return "", nil, err
	}
	if cred.AccessToken == "" && cred.RefreshToken == "" {
		return "", nil, errors.New("oauth credentials missing")
	}
	if !cred.expired(now.UnixMilli()) {
		key, err := oauthAPIKeyForProvider(providerID, cred)
		return key, nil, err
	}
	updated, err := refreshOAuthCredentials(providerID, cred)
	if err != nil {
		return "", nil, err
	}
	updated = mergeOAuthCredentials(updated, cred)
	apiKey, err := oauthAPIKeyForProvider(providerID, updated)
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

func normalizeOAuthProvider(provider string) (string, error) {
	normalized := string(Provider(strings.ToLower(strings.TrimSpace(provider))))
	if normalized == "" {
		return "", errors.New("provider is required")
	}
	return normalized, nil
}

func refreshOAuthCredentials(provider string, cred oauthCredentials) (oauthCredentials, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	switch Provider(provider) {
	case ProviderAnthropic:
		return anthropicRefresh(ctx, cred.RefreshToken)
	case ProviderGitHubCopilot:
		return copilotRefreshToken(ctx, cred.RefreshToken, cred.EnterpriseURL)
	case ProviderGoogleGeminiCLI:
		if cred.ProjectID == "" {
			return oauthCredentials{}, errors.New("google gemini cli projectId missing")
		}
		return geminiCLIRefresh(ctx, cred.RefreshToken, cred.ProjectID)
	case ProviderGoogleAntigrav:
		if cred.ProjectID == "" {
			return oauthCredentials{}, errors.New("antigravity projectId missing")
		}
		return antigravityRefresh(ctx, cred.RefreshToken, cred.ProjectID)
	default:
		return oauthCredentials{}, fmt.Errorf("unsupported oauth provider %q", provider)
	}
}

func mergeOAuthCredentials(updated, prev oauthCredentials) oauthCredentials {
	if updated.ProjectID == "" {
		updated.ProjectID = prev.ProjectID
	}
	if updated.EnterpriseURL == "" {
		updated.EnterpriseURL = prev.EnterpriseURL
	}
	if updated.Email == "" {
		updated.Email = prev.Email
	}
	return updated
}
