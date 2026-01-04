package identity

import "testing"

func TestResolveBinaryName(t *testing.T) {
	if got := ResolveBinaryName(nil); got != CLIName {
		t.Fatalf("expected default %q, got %q", CLIName, got)
	}
	if got := ResolveBinaryName([]string{"peky"}); got != CLIName {
		t.Fatalf("expected %q, got %q", CLIName, got)
	}
	if got := ResolveBinaryName([]string{"/usr/local/bin/peakypanes"}); got != AppSlug {
		t.Fatalf("expected %q, got %q", AppSlug, got)
	}
	if got := ResolveBinaryName([]string{"unknown"}); got != CLIName {
		t.Fatalf("expected fallback %q, got %q", CLIName, got)
	}
}

func TestNormalizeCLIName(t *testing.T) {
	cases := map[string]string{
		"":            CLIName,
		"PEKY":        CLIName,
		"peakypanes":  AppSlug,
		"pp":          CLIName,
		"unexpected":  CLIName,
		" Peakypanes": AppSlug,
	}
	for input, expected := range cases {
		if got := NormalizeCLIName(input); got != expected {
			t.Fatalf("NormalizeCLIName(%q) = %q, want %q", input, got, expected)
		}
	}
}

func TestIsCLICommandToken(t *testing.T) {
	if !IsCLICommandToken("peky") {
		t.Fatalf("expected peky to be recognized")
	}
	if !IsCLICommandToken("peakypanes") {
		t.Fatalf("expected peakypanes to be recognized")
	}
	if !IsCLICommandToken("pp") {
		t.Fatalf("expected pp to be recognized")
	}
	if IsCLICommandToken("unknown") {
		t.Fatalf("expected unknown to be rejected")
	}
}
