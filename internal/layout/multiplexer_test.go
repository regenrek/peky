package layout

import "testing"

func TestNormalizeMultiplexer(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"empty", "", ""},
		{"spaces", "   ", ""},
		{"tmux", "tmux", MultiplexerTmux},
		{"tmux case", "TMUX", MultiplexerTmux},
		{"native", "native", MultiplexerNative},
		{"unknown", "zellij", MultiplexerNative},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NormalizeMultiplexer(tt.input); got != tt.want {
				t.Fatalf("NormalizeMultiplexer(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestResolveMultiplexer(t *testing.T) {
	base := &Config{Multiplexer: "tmux"}
	project := &ProjectConfig{Multiplexer: "native"}
	local := &ProjectLocalConfig{Multiplexer: "tmux"}

	if got := ResolveMultiplexer(nil, nil, nil); got != MultiplexerNative {
		t.Fatalf("ResolveMultiplexer(nil) = %q, want %q", got, MultiplexerNative)
	}

	if got := ResolveMultiplexer(nil, nil, base); got != MultiplexerTmux {
		t.Fatalf("ResolveMultiplexer(global tmux) = %q, want %q", got, MultiplexerTmux)
	}

	if got := ResolveMultiplexer(nil, project, base); got != MultiplexerNative {
		t.Fatalf("ResolveMultiplexer(project native) = %q, want %q", got, MultiplexerNative)
	}

	if got := ResolveMultiplexer(local, project, base); got != MultiplexerTmux {
		t.Fatalf("ResolveMultiplexer(local tmux) = %q, want %q", got, MultiplexerTmux)
	}

	localUnknown := &ProjectLocalConfig{Multiplexer: "unknown"}
	if got := ResolveMultiplexer(localUnknown, project, base); got != MultiplexerNative {
		t.Fatalf("ResolveMultiplexer(local unknown) = %q, want %q", got, MultiplexerNative)
	}
}
