package catalog

import "testing"

import "github.com/regenrek/peakypanes/internal/cli/root"

func TestRegisterAllRegistersHandlers(t *testing.T) {
	reg := root.NewRegistry()
	RegisterAll(reg)

	want := []string{
		"dashboard",
		"start",
		"daemon.restart",
		"init",
		"layouts.list",
		"layouts.export",
		"clone",
		"debug.paths",
		"session.list",
		"pane.list",
		"relay.list",
		"events.watch",
		"context.pack",
		"workspace.list",
		"version",
		"help",
	}
	for _, id := range want {
		if _, ok := reg.HandlerFor(id); !ok {
			t.Fatalf("missing handler %q", id)
		}
	}
	if _, ok := reg.HandlerFor("nl.plan"); ok {
		t.Fatalf("nl should not be registered")
	}
}
