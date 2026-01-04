package logging

import (
	"strings"
	"testing"
)

func TestSanitizeCommandRedactsSensitiveTokens(t *testing.T) {
	input := `TOKEN=abc123 codex --token=def456 --api-key ghi789 -H "Authorization: Bearer xyz999" Authorization=foo000 Bearer bar111`
	out := SanitizeCommand(input)
	for _, secret := range []string{"abc123", "def456", "ghi789", "xyz999", "foo000", "bar111"} {
		if strings.Contains(out, secret) {
			t.Fatalf("expected %q to be redacted in %q", secret, out)
		}
	}
	if !strings.Contains(out, "codex") {
		t.Fatalf("expected command to retain codex, got %q", out)
	}
}
