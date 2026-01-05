package agent

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/regenrek/peakypanes/internal/runenv"
)

type ProviderInfo struct {
	ID             Provider
	Name           string
	SupportsAPIKey bool
	SupportsOAuth  bool
}

func Providers() []ProviderInfo {
	return []ProviderInfo{
		{ID: ProviderGoogle, Name: "Google Gemini (API key)", SupportsAPIKey: true},
		{ID: ProviderGoogleGeminiCLI, Name: "Gemini CLI (OAuth)", SupportsOAuth: true},
		{ID: ProviderGoogleAntigrav, Name: "Antigravity (OAuth)", SupportsOAuth: true},
		{ID: ProviderAnthropic, Name: "Anthropic (Claude)", SupportsAPIKey: true, SupportsOAuth: true},
		{ID: ProviderOpenAI, Name: "OpenAI", SupportsAPIKey: true},
		{ID: ProviderOpenRouter, Name: "OpenRouter", SupportsAPIKey: true},
		{ID: ProviderGitHubCopilot, Name: "GitHub Copilot (OAuth)", SupportsOAuth: true},
	}
}

func DefaultAgentDir() (string, error) {
	if dir := runenv.ConfigDir(); dir != "" {
		return filepath.Join(dir, "agent"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("user home: %w", err)
	}
	return filepath.Join(home, ".config", "peakypanes", "agent"), nil
}

func DefaultAuthPath() (string, error) {
	dir, err := DefaultAgentDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "auth.json"), nil
}

func DefaultSkillsDir() (string, error) {
	if dir := runenv.ConfigDir(); dir != "" {
		return filepath.Join(dir, "skills"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("user home: %w", err)
	}
	return filepath.Join(home, ".config", "peakypanes", "skills"), nil
}
