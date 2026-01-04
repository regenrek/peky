package logging

import (
	"regexp"
	"strings"
)

var (
	sensitiveFlagPattern = regexp.MustCompile(`(?i)(--(?:token|access-token|api-key|apikey|secret|password|passwd|authorization|auth|cookie|session|client-secret|bearer))(=|\s+)(\S+)`)
	sensitiveEnvPattern  = regexp.MustCompile(`(?i)\b([A-Z0-9_]*?(?:TOKEN|SECRET|PASSWORD|PASS|API_KEY|APIKEY|AUTH|AUTHORIZATION|BEARER|COOKIE|SESSION|CLIENT_SECRET)[A-Z0-9_]*)=([^\s]+)`)
	authBearerPattern    = regexp.MustCompile(`(?i)\bAuthorization:\s*Bearer\s+[^\s"'` + "`" + `]+`)
	authHeaderPattern    = regexp.MustCompile(`(?i)\bAuthorization[:=]\s*[^\s"'` + "`" + `]+`)
	bearerPattern        = regexp.MustCompile(`(?i)\bBearer\s+[^\s]+`)
)

// SanitizeCommand redacts common sensitive tokens in command strings.
func SanitizeCommand(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	out := value
	out = sensitiveFlagPattern.ReplaceAllString(out, "$1$2<redacted>")
	out = sensitiveEnvPattern.ReplaceAllString(out, "$1=<redacted>")
	out = authBearerPattern.ReplaceAllString(out, "Authorization: Bearer <redacted>")
	out = authHeaderPattern.ReplaceAllString(out, "Authorization:<redacted>")
	out = bearerPattern.ReplaceAllString(out, "Bearer <redacted>")
	return out
}
