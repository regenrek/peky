package tool

import "strings"

// NormalizeName canonicalizes tool identifiers.
func NormalizeName(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return ""
	}
	value = strings.TrimSuffix(value, ".exe")
	if at := strings.Index(value, "@"); at >= 0 {
		value = value[:at]
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	return value
}

func normalizeList(values []string) []string {
	out := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = NormalizeName(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}
