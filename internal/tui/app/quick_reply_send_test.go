package app

import (
	"bytes"
	"testing"
)

func TestQuickReplyTextBytes_CodexUsesBracketedPaste(t *testing.T) {
	t.Parallel()

	pane := PaneItem{StartCommand: "codex tui"}
	got := quickReplyTextBytes(pane, "hello")
	want := []byte("\x1b[200~hello\x1b[201~")
	if !bytes.Equal(got, want) {
		t.Fatalf("quickReplyTextBytes() = %q want %q", string(got), string(want))
	}
}

func TestQuickReplyTextBytes_CodexUsesBracketedPasteFromTitle(t *testing.T) {
	t.Parallel()

	pane := PaneItem{Title: "codex"}
	got := quickReplyTextBytes(pane, "hello")
	want := []byte("\x1b[200~hello\x1b[201~")
	if !bytes.Equal(got, want) {
		t.Fatalf("quickReplyTextBytes() = %q want %q", string(got), string(want))
	}
}

func TestQuickReplyInputBytes_CodexAppendsSubmit(t *testing.T) {
	t.Parallel()

	pane := PaneItem{StartCommand: "codex tui"}
	got := quickReplyInputBytes(pane, "hello")
	want := []byte("\x1b[200~hello\x1b[201~\r")
	if !bytes.Equal(got, want) {
		t.Fatalf("quickReplyInputBytes() = %q want %q", string(got), string(want))
	}
}

func TestQuickReplyTextBytes_NonCodexUsesPlainText(t *testing.T) {
	t.Parallel()

	pane := PaneItem{StartCommand: "claude"}
	got := quickReplyTextBytes(pane, "hello")
	want := []byte("hello")
	if !bytes.Equal(got, want) {
		t.Fatalf("quickReplyTextBytes() = %q want %q", string(got), string(want))
	}
}

func TestQuickReplyInputBytes_NonCodexSkipsSubmit(t *testing.T) {
	t.Parallel()

	pane := PaneItem{StartCommand: "bash"}
	got := quickReplyInputBytes(pane, "hello")
	want := []byte("hello")
	if !bytes.Equal(got, want) {
		t.Fatalf("quickReplyInputBytes() = %q want %q", string(got), string(want))
	}
}

func TestQuickReplyInputBytes_ClaudeSkipsCombinedSubmit(t *testing.T) {
	t.Parallel()

	pane := PaneItem{StartCommand: "claude"}
	got := quickReplyInputBytes(pane, "hello")
	want := []byte("hello")
	if !bytes.Equal(got, want) {
		t.Fatalf("quickReplyInputBytes() = %q want %q", string(got), string(want))
	}
}

func TestQuickReplySubmitBytes_ClaudeUsesEnter(t *testing.T) {
	t.Parallel()

	pane := PaneItem{StartCommand: "claude"}
	got := quickReplySubmitBytes(pane)
	want := []byte("\r")
	if !bytes.Equal(got, want) {
		t.Fatalf("quickReplySubmitBytes() = %q want %q", string(got), string(want))
	}
}

func TestQuickReplySubmitDelay_Claude(t *testing.T) {
	t.Parallel()

	pane := PaneItem{StartCommand: "claude"}
	got := quickReplySubmitDelay(pane)
	if got <= 0 {
		t.Fatalf("quickReplySubmitDelay() = %v want > 0", got)
	}
}

func TestCommandIsCodex(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		cmd  string
		want bool
	}{
		{name: "empty", cmd: "", want: false},
		{name: "plain", cmd: "codex", want: true},
		{name: "subcommand", cmd: "codex tui", want: true},
		{name: "env wrapper", cmd: "env FOO=1 codex", want: true},
		{name: "npx", cmd: "npx codex", want: true},
		{name: "scoped", cmd: "npx @openai/codex", want: true},
		{name: "versioned", cmd: "npx codex@latest", want: true},
		{name: "quoted", cmd: "'codex' tui", want: true},
		{name: "path", cmd: "/usr/local/bin/codex tui", want: true},
		{name: "exe", cmd: "codex.exe tui", want: true},
		{name: "hyphenated", cmd: "codex-lup", want: false},
		{name: "prefix_only", cmd: "codexsomething tui", want: false},
		{name: "unbalanced_quote", cmd: "'codex tui", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := commandIsCodex(tt.cmd); got != tt.want {
				t.Fatalf("commandIsCodex(%q) = %v want %v", tt.cmd, got, tt.want)
			}
		})
	}
}

func TestCommandIsClaude(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		cmd  string
		want bool
	}{
		{name: "empty", cmd: "", want: false},
		{name: "plain", cmd: "claude", want: true},
		{name: "subcommand", cmd: "claude chat", want: true},
		{name: "env wrapper", cmd: "env FOO=1 claude", want: true},
		{name: "npx", cmd: "npx claude", want: true},
		{name: "scoped", cmd: "npx @anthropic-ai/claude", want: true},
		{name: "versioned", cmd: "npx claude@latest", want: true},
		{name: "quoted", cmd: "'claude' chat", want: true},
		{name: "path", cmd: "/usr/local/bin/claude chat", want: true},
		{name: "exe", cmd: "claude.exe chat", want: true},
		{name: "hyphenated", cmd: "claude-code", want: false},
		{name: "prefix_only", cmd: "claudesomething chat", want: false},
		{name: "unbalanced_quote", cmd: "'claude chat", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := commandIsClaude(tt.cmd); got != tt.want {
				t.Fatalf("commandIsClaude(%q) = %v want %v", tt.cmd, got, tt.want)
			}
		})
	}
}
