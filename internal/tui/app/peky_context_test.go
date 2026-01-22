package app

import (
	"strings"
	"testing"
)

func TestPekyContext_Selected(t *testing.T) {
	m := newTestModelLite()
	got := m.pekyContext()
	if !strings.HasPrefix(got, "Context:\n") {
		t.Fatalf("expected context prefix, got %q", got)
	}
	if !strings.Contains(got, "Selected project: Alpha (/alpha)") {
		t.Fatalf("expected project line, got %q", got)
	}
	if !strings.Contains(got, "Selected session: alpha-1") {
		t.Fatalf("expected session line, got %q", got)
	}
	if !strings.Contains(got, "Selected pane: id=p1 index=1 title=one cwd=unknown") {
		t.Fatalf("expected pane line, got %q", got)
	}
	if !strings.Contains(got, "Use --pane-id p1") {
		t.Fatalf("expected pane-id hint, got %q", got)
	}
}

func TestPekyContext_NilModel(t *testing.T) {
	var m *Model
	if got := m.pekyContext(); got != "" {
		t.Fatalf("expected empty context, got %q", got)
	}
}
