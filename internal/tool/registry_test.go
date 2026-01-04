package tool

import (
	"testing"

	"github.com/regenrek/peakypanes/internal/layout"
)

func TestNormalizeName(t *testing.T) {
	if got := NormalizeName("Codex@1"); got != "codex" {
		t.Fatalf("NormalizeName = %q", got)
	}
	if got := NormalizeName("codex.exe"); got != "codex" {
		t.Fatalf("NormalizeName(.exe) = %q", got)
	}
	if got := NormalizeName(" "); got != "" {
		t.Fatalf("NormalizeName(blank) = %q", got)
	}
}

func TestDefaultRegistryDetectsCodex(t *testing.T) {
	reg := defaultRegistry(t)
	if got := reg.DetectFromCommand("codex"); got != "codex" {
		t.Fatalf("DetectFromCommand(codex) = %q", got)
	}
	if got := reg.DetectFromCommand("env FOO=1 codex"); got != "codex" {
		t.Fatalf("DetectFromCommand(env codex) = %q", got)
	}
	if got := reg.DetectFromCommand("gh dash"); got != "gh-dash" {
		t.Fatalf("DetectFromCommand(gh dash) = %q", got)
	}
}

func TestDefaultRegistryNotEmpty(t *testing.T) {
	reg := defaultRegistry(t)
	if reg == nil || len(reg.defs) == 0 {
		t.Fatalf("expected default registry to have definitions")
	}
	if got := reg.DetectFromCommand("codex"); got == "" {
		t.Fatalf("expected codex detection")
	}
}

func TestRegistryDetectFromTitle(t *testing.T) {
	reg := defaultRegistry(t)
	if got := reg.DetectFromTitle("Codex: chat"); got != "codex" {
		t.Fatalf("DetectFromTitle = %q", got)
	}
}

func TestRegistryProfileCodex(t *testing.T) {
	reg := defaultRegistry(t)
	profile := reg.Profile("codex")
	if !profile.BracketedPaste {
		t.Fatalf("codex should use bracketed paste")
	}
	if len(profile.Submit) == 0 {
		t.Fatalf("codex submit bytes missing")
	}
}

func TestRegistryFromConfigCustomTool(t *testing.T) {
	cfg := layout.ToolDetectionConfig{
		Tools: []layout.ToolDefinitionConfig{
			{
				Name:         "demo",
				CommandNames: []string{"demo"},
				Input: layout.ToolInputConfig{
					BracketedPaste: boolPtr(true),
				},
			},
		},
		Profiles: map[string]layout.ToolInputConfig{
			"codex": {Submit: stringPtr("\n")},
		},
	}
	reg, err := RegistryFromConfig(cfg)
	if err != nil {
		t.Fatalf("RegistryFromConfig: %v", err)
	}
	if got := reg.DetectFromCommand("demo"); got != "demo" {
		t.Fatalf("custom DetectFromCommand = %q", got)
	}
	if got := reg.Profile("demo"); !got.BracketedPaste {
		t.Fatalf("custom profile not applied")
	}
	if got := reg.Profile("codex"); string(got.Submit) != "\n" {
		t.Fatalf("profile override not applied")
	}
}

func TestRegistryAllowDenyPolicy(t *testing.T) {
	cfg := layout.ToolDetectionConfig{
		Allow: map[string]bool{
			"codex": false,
		},
	}
	reg, err := RegistryFromConfig(cfg)
	if err != nil {
		t.Fatalf("RegistryFromConfig: %v", err)
	}
	if reg.Allowed("codex") {
		t.Fatalf("expected codex disallowed")
	}
	if got := reg.ResolveTool(PaneInfo{Tool: "codex"}); got != "" {
		t.Fatalf("ResolveTool(disallowed) = %q", got)
	}
	if got := reg.Profile("codex"); got.BracketedPaste || string(got.Submit) != "\n" {
		t.Fatalf("expected default profile for disallowed tool, got=%+v", got)
	}
}

func boolPtr(v bool) *bool       { return &v }
func stringPtr(v string) *string { return &v }
