package identity

import "testing"

func TestIsCLICommandToken(t *testing.T) {
	if !IsCLICommandToken("peky") {
		t.Fatalf("expected peky to be recognized")
	}
	if !IsCLICommandToken("pp") {
		t.Fatalf("expected pp to be recognized")
	}
	if IsCLICommandToken("unknown") {
		t.Fatalf("expected unknown to be rejected")
	}
}
