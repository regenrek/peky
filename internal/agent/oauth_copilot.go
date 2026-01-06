package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

const (
	copilotClientID = "Iv1.b507a08c87ecfe98"
)

type copilotDeviceResponse struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
	Interval        int    `json:"interval"`
	ExpiresIn       int    `json:"expires_in"`
}

type copilotTokenSuccess struct {
	AccessToken string `json:"access_token"`
}

type copilotTokenError struct {
	Error string `json:"error"`
}

type copilotTokenResponse struct {
	Token     string `json:"token"`
	ExpiresAt int64  `json:"expires_at"`
}

func copilotNormalizeDomain(input string) (string, error) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return "", nil
	}
	if !strings.Contains(trimmed, "://") {
		trimmed = "https://" + trimmed
	}
	parsed, err := url.Parse(trimmed)
	if err != nil || parsed.Hostname() == "" {
		return "", errors.New("invalid domain")
	}
	return parsed.Hostname(), nil
}

func copilotURLs(domain string) (deviceCodeURL, accessTokenURL, copilotTokenURL string) {
	return "https://" + domain + "/login/device/code", "https://" + domain + "/login/oauth/access_token", "https://api." + domain + "/copilot_internal/v2/token"
}

func copilotStartDeviceFlow(ctx context.Context, domain string) (copilotDeviceResponse, error) {
	deviceURL, _, _ := copilotURLs(domain)
	payload := map[string]string{
		"client_id": copilotClientID,
		"scope":     "read:user",
	}
	resp, err := oauthPostJSON(ctx, deviceURL, payload, map[string]string{"Accept": "application/json", "User-Agent": "GitHubCopilotChat/0.35.0"})
	if err != nil {
		return copilotDeviceResponse{}, err
	}
	body, err := readResponseBody(resp)
	if err != nil {
		return copilotDeviceResponse{}, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return copilotDeviceResponse{}, fmt.Errorf("copilot device flow failed: %s", body)
	}
	var parsed copilotDeviceResponse
	if err := json.Unmarshal([]byte(body), &parsed); err != nil {
		return copilotDeviceResponse{}, fmt.Errorf("copilot device parse: %w", err)
	}
	if parsed.DeviceCode == "" || parsed.UserCode == "" || parsed.VerificationURI == "" {
		return copilotDeviceResponse{}, errors.New("copilot device response missing fields")
	}
	return parsed, nil
}

func copilotPollAccessToken(ctx context.Context, domain, deviceCode string, intervalSeconds, expiresIn int) (string, error) {
	_, accessURL, _ := copilotURLs(domain)
	deadline := time.Now().Add(time.Duration(expiresIn) * time.Second)
	interval := time.Duration(intervalSeconds) * time.Second
	if interval < time.Second {
		interval = time.Second
	}
	for time.Now().Before(deadline) {
		payload := map[string]string{
			"client_id":   copilotClientID,
			"device_code": deviceCode,
			"grant_type":  "urn:ietf:params:oauth:grant-type:device_code",
		}
		resp, err := oauthPostJSON(ctx, accessURL, payload, map[string]string{"Accept": "application/json", "User-Agent": "GitHubCopilotChat/0.35.0"})
		if err != nil {
			return "", err
		}
		body, err := readResponseBody(resp)
		if err != nil {
			return "", err
		}
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			var okResp copilotTokenSuccess
			if err := json.Unmarshal([]byte(body), &okResp); err == nil && okResp.AccessToken != "" {
				return okResp.AccessToken, nil
			}
			var errResp copilotTokenError
			if err := json.Unmarshal([]byte(body), &errResp); err == nil && errResp.Error != "" {
				switch errResp.Error {
				case "authorization_pending":
					time.Sleep(interval)
					continue
				case "slow_down":
					interval += 5 * time.Second
					time.Sleep(interval)
					continue
				default:
					return "", fmt.Errorf("copilot device flow failed: %s", errResp.Error)
				}
			}
		}
		return "", fmt.Errorf("copilot device flow failed: %s", body)
	}
	return "", errors.New("copilot device flow timed out")
}

func copilotRefreshToken(ctx context.Context, refreshToken, enterpriseDomain string) (oauthCredentials, error) {
	domain := enterpriseDomain
	if strings.TrimSpace(domain) == "" {
		domain = "github.com"
	}
	_, _, tokenURL := copilotURLs(domain)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, tokenURL, nil)
	if err != nil {
		return oauthCredentials{}, fmt.Errorf("copilot token request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+refreshToken)
	req.Header.Set("User-Agent", "GitHubCopilotChat/0.35.0")
	resp, err := authHTTPClient.Do(req)
	if err != nil {
		return oauthCredentials{}, err
	}
	body, err := readResponseBody(resp)
	if err != nil {
		return oauthCredentials{}, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return oauthCredentials{}, fmt.Errorf("copilot token refresh failed: %s", body)
	}
	var parsed copilotTokenResponse
	if err := json.Unmarshal([]byte(body), &parsed); err != nil {
		return oauthCredentials{}, fmt.Errorf("copilot token parse: %w", err)
	}
	if parsed.Token == "" || parsed.ExpiresAt == 0 {
		return oauthCredentials{}, errors.New("copilot token response missing fields")
	}
	return oauthCredentials{
		RefreshToken:  refreshToken,
		AccessToken:   parsed.Token,
		ExpiresAtMS:   parsed.ExpiresAt*1000 - 5*60*1000,
		EnterpriseURL: enterpriseDomain,
	}, nil
}

var copilotProxyRe = regexp.MustCompile(`proxy-ep=([^;]+)`) // #nosec G101 -- token parsing

func copilotBaseURL(token, enterpriseDomain string) string {
	if token != "" {
		if match := copilotProxyRe.FindStringSubmatch(token); len(match) == 2 {
			proxyHost := match[1]
			apiHost := strings.TrimPrefix(proxyHost, "proxy.")
			return "https://api." + apiHost
		}
	}
	if enterpriseDomain != "" {
		return "https://copilot-api." + enterpriseDomain
	}
	return "https://api.individual.githubcopilot.com"
}
