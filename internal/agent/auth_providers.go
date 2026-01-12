package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/regenrek/peakypanes/internal/identity"
	"github.com/regenrek/peakypanes/internal/runenv"
)

type ProviderInfo struct {
	ID             Provider
	Name           string
	Aliases        []string
	SupportsAPIKey bool
	SupportsOAuth  bool
	DefaultModel   string
	Models         []string
}

func Providers() []ProviderInfo {
	return []ProviderInfo{
		{
			ID:             ProviderGoogle,
			Name:           "Google",
			Aliases:        []string{"google", "gemini", "google-gemini"},
			SupportsAPIKey: true,
			DefaultModel:   "gemini-3-flash",
			Models:         []string{"gemini-3-flash", "gemini-3-pro", "gemini-2.5-flash", "gemini-2.5-pro"},
		},
		{
			ID:            ProviderGoogleGeminiCLI,
			Name:          "Gemini CLI",
			Aliases:       []string{"gemini-cli", "google-gemini-cli"},
			SupportsOAuth: true,
			DefaultModel:  "gemini-3-flash",
			Models:        []string{"gemini-3-flash", "gemini-3-pro", "gemini-2.5-flash", "gemini-2.5-pro"},
		},
		{
			ID:            ProviderGoogleAntigrav,
			Name:          "Antigravity",
			Aliases:       []string{"antigravity", "google-antigravity"},
			SupportsOAuth: true,
			DefaultModel:  "gemini-3-flash",
			Models:        []string{"gemini-3-flash", "gemini-3-pro", "gemini-2.5-flash", "gemini-2.5-pro"},
		},
		{
			ID:             ProviderAnthropic,
			Name:           "Claude - Anthropic",
			Aliases:        []string{"anthropic", "claude"},
			SupportsAPIKey: true,
			SupportsOAuth:  true,
			DefaultModel:   "claude-3-5-sonnet",
			Models:         []string{"claude-3-5-sonnet", "claude-3-5-haiku", "claude-3-opus"},
		},
		{
			ID:             ProviderOpenAI,
			Name:           "OpenAI",
			Aliases:        []string{"openai"},
			SupportsAPIKey: true,
			DefaultModel:   "gpt-4o-mini",
			Models:         []string{"gpt-4o-mini", "gpt-4o"},
		},
		{
			ID:             ProviderOpenRouter,
			Name:           "OpenRouter",
			Aliases:        []string{"openrouter"},
			SupportsAPIKey: true,
			DefaultModel:   "openrouter/auto",
			Models:         []string{"openrouter/auto"},
		},
		{
			ID:            ProviderGitHubCopilot,
			Name:          "GitHub Copilot",
			Aliases:       []string{"copilot", "github-copilot", "gh-copilot"},
			SupportsOAuth: true,
			DefaultModel:  "gpt-4o-mini",
			Models:        []string{"gpt-4o-mini", "gpt-4o"},
		},
	}
}

func DefaultAgentDir() (string, error) {
	if dir := runenv.ConfigDir(); dir != "" {
		return filepath.Join(dir, "agent"), nil
	}
	cfgDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("user config dir: %w", err)
	}
	return filepath.Join(cfgDir, identity.AppSlug, "agent"), nil
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
	cfgDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("user config dir: %w", err)
	}
	return filepath.Join(cfgDir, identity.AppSlug, "skills"), nil
}

func FindProviderInfo(input string) (ProviderInfo, bool) {
	normalized := strings.ToLower(strings.TrimSpace(input))
	if normalized == "" {
		return ProviderInfo{}, false
	}
	for _, entry := range Providers() {
		if normalized == strings.ToLower(string(entry.ID)) {
			return entry, true
		}
		if normalized == strings.ToLower(entry.Name) {
			return entry, true
		}
		for _, alias := range entry.Aliases {
			if normalized == strings.ToLower(strings.TrimSpace(alias)) {
				return entry, true
			}
		}
	}
	return ProviderInfo{}, false
}

func ProviderLabel(provider Provider) string {
	for _, entry := range Providers() {
		if entry.ID == provider {
			return entry.Name
		}
	}
	return ""
}

func ProviderModels(provider Provider) []string {
	for _, entry := range Providers() {
		if entry.ID == provider {
			return append([]string(nil), entry.Models...)
		}
	}
	return nil
}

func ProviderDefaultModel(provider Provider) string {
	for _, entry := range Providers() {
		if entry.ID == provider {
			return entry.DefaultModel
		}
	}
	return ""
}

func ProviderHasModel(provider Provider, model string) bool {
	model = strings.TrimSpace(model)
	if model == "" {
		return false
	}
	models := ProviderModels(provider)
	if len(models) == 0 {
		return true
	}
	for _, entry := range models {
		if strings.TrimSpace(entry) == model {
			return true
		}
	}
	return false
}
