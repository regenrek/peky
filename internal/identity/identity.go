package identity

import (
	"strings"
)

const (
	BrandName = "PeakyPanes"
	// AppSlug is the canonical identifier for user-facing and on-disk state.
	// It intentionally matches the only supported CLI binary name.
	AppSlug = "peky"
	CLIName = "peky"

	ProjectConfigFileYML  = ".peky.yml"
	ProjectConfigFileYAML = ".peky.yaml"

	GlobalConfigFile = "config.yml"
	GlobalLayoutsDir = "layouts"
)

var (
	InputAliases = []string{"pp"}
)

func IsCLICommandToken(token string) bool {
	trimmed := strings.ToLower(strings.TrimSpace(token))
	if trimmed == "" {
		return false
	}
	if trimmed == CLIName {
		return true
	}
	for _, alias := range InputAliases {
		if trimmed == alias {
			return true
		}
	}
	return false
}
