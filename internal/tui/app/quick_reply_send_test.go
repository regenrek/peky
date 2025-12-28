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

func TestQuickReplyTextBytes_NonCodexUsesPlainText(t *testing.T) {
	t.Parallel()

	pane := PaneItem{StartCommand: "claude"}
	got := quickReplyTextBytes(pane, "hello")
	want := []byte("hello")
	if !bytes.Equal(got, want) {
		t.Fatalf("quickReplyTextBytes() = %q want %q", string(got), string(want))
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
		{name: "quoted", cmd: "'codex' tui", want: true},
		{name: "path", cmd: "/usr/local/bin/codex tui", want: true},
		{name: "exe", cmd: "codex.exe tui", want: true},
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

