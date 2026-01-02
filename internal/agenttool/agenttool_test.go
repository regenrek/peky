package agenttool

import "testing"

func TestNormalize(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		want Tool
	}{
		{name: "empty", in: "", want: ""},
		{name: "codex", in: "codex", want: ToolCodex},
		{name: "codex_version", in: "codex@latest", want: ToolCodex},
		{name: "codex_exe", in: "codex.exe", want: ToolCodex},
		{name: "claude", in: "claude", want: ToolClaude},
		{name: "claude_version", in: "claude@latest", want: ToolClaude},
		{name: "claude_exe", in: "claude.exe", want: ToolClaude},
		{name: "unknown", in: "gemini", want: ""},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := Normalize(tt.in); got != tt.want {
				t.Fatalf("Normalize(%q) = %q want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestDetectFromTitle(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		title string
		want  Tool
	}{
		{name: "empty", title: "", want: ""},
		{name: "codex", title: "codex", want: ToolCodex},
		{name: "codex_prefix", title: "codex: editor", want: ToolCodex},
		{name: "claude", title: "claude", want: ToolClaude},
		{name: "claude_prefix", title: "claude: chat", want: ToolClaude},
		{name: "unknown", title: "editor", want: ""},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := DetectFromTitle(tt.title); got != tt.want {
				t.Fatalf("DetectFromTitle(%q) = %q want %q", tt.title, got, tt.want)
			}
		})
	}
}

func TestDetectFromCommand(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		cmd  string
		want Tool
	}{
		{name: "empty", cmd: "", want: ""},
		{name: "codex", cmd: "codex", want: ToolCodex},
		{name: "codex_subcommand", cmd: "codex tui", want: ToolCodex},
		{name: "codex_env", cmd: "env FOO=1 codex", want: ToolCodex},
		{name: "codex_npx", cmd: "npx codex", want: ToolCodex},
		{name: "codex_scoped", cmd: "npx @openai/codex", want: ToolCodex},
		{name: "codex_versioned", cmd: "npx codex@latest", want: ToolCodex},
		{name: "codex_quoted", cmd: "'codex' tui", want: ToolCodex},
		{name: "codex_path", cmd: "/usr/local/bin/codex tui", want: ToolCodex},
		{name: "codex_exe", cmd: "codex.exe tui", want: ToolCodex},
		{name: "claude", cmd: "claude", want: ToolClaude},
		{name: "claude_subcommand", cmd: "claude chat", want: ToolClaude},
		{name: "claude_env", cmd: "env FOO=1 claude", want: ToolClaude},
		{name: "claude_npx", cmd: "npx claude", want: ToolClaude},
		{name: "claude_scoped", cmd: "npx @anthropic-ai/claude", want: ToolClaude},
		{name: "claude_versioned", cmd: "npx claude@latest", want: ToolClaude},
		{name: "claude_quoted", cmd: "'claude' chat", want: ToolClaude},
		{name: "claude_path", cmd: "/usr/local/bin/claude chat", want: ToolClaude},
		{name: "claude_exe", cmd: "claude.exe chat", want: ToolClaude},
		{name: "unknown", cmd: "echo codex", want: ""},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := DetectFromCommand(tt.cmd); got != tt.want {
				t.Fatalf("DetectFromCommand(%q) = %q want %q", tt.cmd, got, tt.want)
			}
		})
	}
}
