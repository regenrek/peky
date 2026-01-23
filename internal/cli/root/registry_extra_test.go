package root

import (
	"testing"

	"github.com/regenrek/peakypanes/internal/cli/spec"
)

func TestRegistryEnsureHandlers(t *testing.T) {
	specDoc := &spec.Spec{
		Commands: []spec.Command{{ID: "cmd", Name: "cmd"}},
	}
	reg := NewRegistry()
	if err := reg.EnsureHandlers(specDoc); err == nil {
		t.Fatalf("expected missing handler error")
	}
	reg.Register("cmd", func(ctx CommandContext) error { return nil })
	if err := reg.EnsureHandlers(specDoc); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRegisterNoop(t *testing.T) {
	var reg *Registry
	reg.Register("id", func(ctx CommandContext) error { return nil })
	empty := NewRegistry()
	empty.Register("", func(ctx CommandContext) error { return nil })
	if _, ok := empty.HandlerFor(""); ok {
		t.Fatalf("expected no handler for empty id")
	}
}
