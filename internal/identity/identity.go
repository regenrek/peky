package identity

import (
	"strings"
)

const (
	BrandName = "PeakyPanes"
	AppSlug   = "peakypanes"
	CLIName   = "peky"
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
