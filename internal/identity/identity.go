package identity

import (
	"path/filepath"
	"strings"
)

const (
	BrandName = "PeakyPanes"
	AppSlug   = "peakypanes"
	CLIName   = "peky"
)

var (
	CLIAliases   = []string{AppSlug}
	InputAliases = []string{"pp"}
)

func ResolveBinaryName(args []string) string {
	if len(args) == 0 {
		return CLIName
	}
	base := strings.ToLower(filepath.Base(strings.TrimSpace(args[0])))
	if base == "" {
		return CLIName
	}
	if base == CLIName {
		return CLIName
	}
	for _, alias := range CLIAliases {
		if base == alias {
			return alias
		}
	}
	return CLIName
}

func NormalizeCLIName(name string) string {
	trimmed := strings.ToLower(strings.TrimSpace(name))
	if trimmed == "" {
		return CLIName
	}
	if trimmed == CLIName {
		return CLIName
	}
	for _, alias := range CLIAliases {
		if trimmed == alias {
			return alias
		}
	}
	return CLIName
}

func IsCLICommandToken(token string) bool {
	trimmed := strings.ToLower(strings.TrimSpace(token))
	if trimmed == "" {
		return false
	}
	if trimmed == CLIName {
		return true
	}
	for _, alias := range CLIAliases {
		if trimmed == alias {
			return true
		}
	}
	for _, alias := range InputAliases {
		if trimmed == alias {
			return true
		}
	}
	return false
}
