package agent

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var authHTTPClient = http.DefaultClient

func oauthPKCE() (verifier, challenge string, err error) {
	verifierBytes := make([]byte, 32)
	if _, err = rand.Read(verifierBytes); err != nil {
		return "", "", fmt.Errorf("pkce random: %w", err)
	}
	verifier = base64.RawURLEncoding.EncodeToString(verifierBytes)
	hash := sha256.Sum256([]byte(verifier))
	challenge = base64.RawURLEncoding.EncodeToString(hash[:])
	return verifier, challenge, nil
}

func oauthPostForm(ctx context.Context, endpoint string, values url.Values) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(values.Encode()))
	if err != nil {
		return nil, fmt.Errorf("oauth request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return authHTTPClient.Do(req)
}

func oauthPostJSON(ctx context.Context, endpoint string, payload any, headers map[string]string) (*http.Response, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("oauth json: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("oauth request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		if strings.TrimSpace(k) == "" {
			continue
		}
		req.Header.Set(k, v)
	}
	return authHTTPClient.Do(req)
}

func readResponseBody(resp *http.Response) (string, error) {
	if resp == nil {
		return "", errors.New("nil response")
	}
	data, readErr := io.ReadAll(resp.Body)
	closeErr := resp.Body.Close()
	if readErr != nil {
		err := fmt.Errorf("read response: %w", readErr)
		if closeErr != nil {
			return "", errors.Join(err, fmt.Errorf("response close: %w", closeErr))
		}
		return "", err
	}
	if closeErr != nil {
		return "", fmt.Errorf("response close: %w", closeErr)
	}
	return string(data), nil
}

func oauthExpiry(expiresInSeconds int64) int64 {
	if expiresInSeconds <= 0 {
		return time.Now().Add(55 * time.Minute).UnixMilli()
	}
	return time.Now().Add(time.Duration(expiresInSeconds)*time.Second - 5*time.Minute).UnixMilli()
}
