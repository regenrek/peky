package update

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const npmLatestURL = "https://registry.npmjs.org/peakypanes/latest"

// RegistryClient retrieves the latest version.
type RegistryClient interface {
	LatestVersion(ctx context.Context) (string, error)
}

// NPMClient fetches versions from the npm registry.
type NPMClient struct {
	HTTPClient *http.Client
	UserAgent  string
}

type npmLatestResponse struct {
	Version string `json:"version"`
}

// LatestVersion implements RegistryClient.
func (c NPMClient) LatestVersion(ctx context.Context) (version string, err error) {
	if ctx == nil {
		ctx = context.Background()
	}
	client := c.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, npmLatestURL, nil)
	if err != nil {
		return "", fmt.Errorf("npm request: %w", err)
	}
	ua := c.UserAgent
	if ua == "" {
		ua = "peakypanes/auto-update"
	}
	req.Header.Set("User-Agent", ua)
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("npm request: %w", err)
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("npm close response: %w", cerr)
		}
	}()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("npm status %d", resp.StatusCode)
	}
	var payload npmLatestResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", fmt.Errorf("npm decode: %w", err)
	}
	if strings.TrimSpace(payload.Version) == "" {
		return "", fmt.Errorf("npm response missing version")
	}
	return payload.Version, nil
}
