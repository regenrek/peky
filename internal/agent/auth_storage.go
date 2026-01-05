package agent

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type authCredentialType string

const (
	authCredentialAPIKey authCredentialType = "api_key"
	authCredentialOAuth  authCredentialType = "oauth"
)

type authCredential struct {
	Type         authCredentialType `json:"type"`
	Key          string             `json:"key,omitempty"`
	RefreshToken string             `json:"refresh,omitempty"`
	AccessToken  string             `json:"access,omitempty"`
	ExpiresAtMS  int64              `json:"expires,omitempty"`
	Enterprise   string             `json:"enterpriseUrl,omitempty"`
	ProjectID    string             `json:"projectId,omitempty"`
	Email        string             `json:"email,omitempty"`
}

type authStore struct {
	path     string
	mu       sync.Mutex
	data     map[string]authCredential
	override map[string]string
}

func newAuthStore(path string) (*authStore, error) {
	store := &authStore{
		path:     path,
		data:     make(map[string]authCredential),
		override: make(map[string]string),
	}
	if err := store.reload(); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *authStore) reload() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			s.data = make(map[string]authCredential)
			return nil
		}
		return fmt.Errorf("read auth store: %w", err)
	}
	var payload map[string]authCredential
	if err := json.Unmarshal(data, &payload); err != nil {
		return fmt.Errorf("parse auth store: %w", err)
	}
	s.data = payload
	return nil
}

func (s *authStore) save() error {
	if s == nil {
		return errors.New("auth store is nil")
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return fmt.Errorf("create auth dir: %w", err)
	}
	payload, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal auth store: %w", err)
	}
	if err := os.WriteFile(s.path, payload, 0o600); err != nil {
		return fmt.Errorf("write auth store: %w", err)
	}
	return nil
}

func (s *authStore) setAPIKey(provider, key string) error {
	provider = strings.TrimSpace(provider)
	key = strings.TrimSpace(key)
	if provider == "" || key == "" {
		return errors.New("provider and key are required")
	}
	s.mu.Lock()
	s.data[provider] = authCredential{Type: authCredentialAPIKey, Key: key}
	s.mu.Unlock()
	return s.save()
}

func (s *authStore) setOAuth(provider string, cred oauthCredentials) error {
	provider = strings.TrimSpace(provider)
	if provider == "" {
		return errors.New("provider is required")
	}
	s.mu.Lock()
	s.data[provider] = authCredential{
		Type:         authCredentialOAuth,
		RefreshToken: cred.RefreshToken,
		AccessToken:  cred.AccessToken,
		ExpiresAtMS:  cred.ExpiresAtMS,
		Enterprise:   cred.EnterpriseURL,
		ProjectID:    cred.ProjectID,
		Email:        cred.Email,
	}
	s.mu.Unlock()
	return s.save()
}

func (s *authStore) remove(provider string) error {
	provider = strings.TrimSpace(provider)
	if provider == "" {
		return errors.New("provider is required")
	}
	s.mu.Lock()
	delete(s.data, provider)
	s.mu.Unlock()
	return s.save()
}

func (s *authStore) hasAuth(provider string) bool {
	provider = strings.TrimSpace(provider)
	if provider == "" {
		return false
	}
	s.mu.Lock()
	_, ok := s.data[provider]
	s.mu.Unlock()
	if ok {
		return true
	}
	if envKey := envAPIKey(provider); envKey != "" {
		return true
	}
	return false
}

func (s *authStore) getAPIKey(provider string, now time.Time) (string, *oauthCredentials, error) {
	provider = strings.TrimSpace(provider)
	if provider == "" {
		return "", nil, errors.New("provider is required")
	}
	s.mu.Lock()
	if key, ok := s.override[provider]; ok && strings.TrimSpace(key) != "" {
		s.mu.Unlock()
		return key, nil, nil
	}
	cred, ok := s.data[provider]
	s.mu.Unlock()
	if ok {
		switch cred.Type {
		case authCredentialAPIKey:
			if strings.TrimSpace(cred.Key) != "" {
				return cred.Key, nil, nil
			}
		case authCredentialOAuth:
			oauth := oauthCredentials{
				RefreshToken:  cred.RefreshToken,
				AccessToken:   cred.AccessToken,
				ExpiresAtMS:   cred.ExpiresAtMS,
				EnterpriseURL: cred.Enterprise,
				ProjectID:     cred.ProjectID,
				Email:         cred.Email,
			}
			apiKey, updated, err := resolveOAuthAPIKey(provider, oauth, now)
			if err != nil {
				return "", nil, err
			}
			if updated != nil {
				if err := s.setOAuth(provider, *updated); err != nil {
					return apiKey, updated, err
				}
			}
			if apiKey != "" {
				return apiKey, updated, nil
			}
		}
	}
	if envKey := envAPIKey(provider); envKey != "" {
		return envKey, nil, nil
	}
	return "", nil, errors.New("no auth configured")
}
