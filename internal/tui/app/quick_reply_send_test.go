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

func TestQuickReplyTextBytes_CodexUsesToolHint(t *testing.T) {
	t.Parallel()

	pane := PaneItem{Tool: "codex"}
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
	want := []byte("\x1b[200~hello\x1b[201~")
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
