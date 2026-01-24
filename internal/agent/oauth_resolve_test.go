package agent

import (
	"encoding/json"
	"testing"
	"time"
)

func TestNormalizeOAuthProvider(t *testing.T) {
	if _, err := normalizeOAuthProvider(" "); err == nil {
		t.Fatalf("expected error for empty provider")
	}
	out, err := normalizeOAuthProvider(" OpenAI ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "openai" {
		t.Fatalf("normalized=%q", out)
	}
}

func TestOAuthAPIKeyForProvider(t *testing.T) {
	key, err := oauthAPIKeyForProvider(string(ProviderGoogleGeminiCLI), oauthCredentials{AccessToken: "tok", ProjectID: "proj"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var payload map[string]string
	if err := json.Unmarshal([]byte(key), &payload); err != nil {
		t.Fatalf("payload parse error: %v", err)
	}
	if payload["token"] != "tok" || payload["projectId"] != "proj" {
		t.Fatalf("payload=%#v", payload)
	}
	if _, err := oauthAPIKeyForProvider(string(ProviderOpenAI), oauthCredentials{}); err == nil {
		t.Fatalf("expected error for missing access token")
	}
}

func TestResolveOAuthAPIKeyNonExpired(t *testing.T) {
	now := time.Now()
	cred := oauthCredentials{AccessToken: "tok", ExpiresAtMS: now.Add(time.Hour).UnixMilli()}
	key, updated, err := resolveOAuthAPIKey(string(ProviderOpenAI), cred, now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if key != "tok" || updated != nil {
		t.Fatalf("key=%q updated=%v", key, updated)
	}
}

func TestResolveOAuthAPIKeyExpiredUnknown(t *testing.T) {
	now := time.Now()
	cred := oauthCredentials{AccessToken: "tok", RefreshToken: "refresh", ExpiresAtMS: now.Add(-time.Hour).UnixMilli()}
	if _, _, err := resolveOAuthAPIKey("unknown", cred, now); err == nil {
		t.Fatalf("expected error for unsupported provider")
	}
}

func TestMergeOAuthCredentials(t *testing.T) {
	updated := oauthCredentials{AccessToken: "new"}
	prev := oauthCredentials{ProjectID: "proj", EnterpriseURL: "ent", Email: "a@b"}
	out := mergeOAuthCredentials(updated, prev)
	if out.ProjectID != "proj" || out.EnterpriseURL != "ent" || out.Email != "a@b" {
		t.Fatalf("merged=%#v", out)
	}
}
