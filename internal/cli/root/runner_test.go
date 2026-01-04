package root

import (
	"testing"

	"github.com/regenrek/peakypanes/internal/cli/spec"
)

func TestApplyShorthandDefaultCommand(t *testing.T) {
	specDoc := minimalSpec()
	args := applyShorthand(specDoc, []string{"peakypanes"})
	if len(args) != 2 || args[1] != "dashboard" {
		t.Fatalf("expected default command, got %v", args)
	}
}

func TestApplyShorthandLayout(t *testing.T) {
	specDoc := minimalSpec()
	args := applyShorthand(specDoc, []string{"peakypanes", "auto"})
	if len(args) != 4 || args[1] != "start" || args[2] != "--layout" || args[3] != "auto" {
		t.Fatalf("unexpected shorthand args: %v", args)
	}
}

func TestApplyShorthandSkipsKnownCommand(t *testing.T) {
	specDoc := minimalSpec()
	args := applyShorthand(specDoc, []string{"peakypanes", "daemon"})
	if len(args) != 2 || args[1] != "daemon" {
		t.Fatalf("expected daemon command preserved, got %v", args)
	}
}

func TestApplyShorthandDisabled(t *testing.T) {
	specDoc := minimalSpec()
	specDoc.App.AllowLayoutShorthand = false
	args := applyShorthand(specDoc, []string{"peakypanes", "auto"})
	if len(args) != 2 || args[1] != "auto" {
		t.Fatalf("expected shorthand disabled, got %v", args)
	}
}

func minimalSpec() *spec.Spec {
	return &spec.Spec{
		App: spec.AppSpec{DefaultCommand: "dashboard", AllowLayoutShorthand: true},
		Commands: []spec.Command{
			{Name: "dashboard", Aliases: []string{"ui"}},
			{Name: "start", Aliases: []string{"open", "o"}},
			{Name: "daemon"},
		},
	}
}
