package mux

import (
	"testing"

	"github.com/regenrek/peakypanes/internal/layout"
)

func TestResolveTypePrecedence(t *testing.T) {
	cfg := &layout.Config{Multiplexer: "zellij"}
	project := &layout.ProjectConfig{Multiplexer: "tmux"}
	local := &layout.ProjectLocalConfig{Multiplexer: "zellij"}

	if got := ResolveType("tmux", cfg, project, local); got != Tmux {
		t.Fatalf("cli override: expected tmux, got %s", got)
	}

	if got := ResolveType("", cfg, project, local); got != Zellij {
		t.Fatalf("local override: expected zellij, got %s", got)
	}

	if got := ResolveType("", cfg, project, nil); got != Tmux {
		t.Fatalf("project override: expected tmux, got %s", got)
	}

	if got := ResolveType("", cfg, nil, nil); got != Zellij {
		t.Fatalf("global fallback: expected zellij, got %s", got)
	}

	if got := ResolveType("", &layout.Config{}, nil, nil); got != Tmux {
		t.Fatalf("default fallback: expected tmux, got %s", got)
	}
}

func TestResolveTypeIgnoresInvalidCliValue(t *testing.T) {
	cfg := &layout.Config{Multiplexer: "tmux"}
	local := &layout.ProjectLocalConfig{Multiplexer: "zellij"}

	if got := ResolveType("not-a-mux", cfg, nil, local); got != Zellij {
		t.Fatalf("invalid cli should fall back to local: expected zellij, got %s", got)
	}
}
