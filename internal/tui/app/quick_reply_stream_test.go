package app

import (
	"bytes"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/regenrek/peakypanes/internal/layout"
)

func TestQuickReplyStreamBatchesKeysUntilFlush(t *testing.T) {
	m := newTestModelLite()
	m.config = &layout.Config{QuickReply: layout.QuickReplyConfig{StreamToPane: true}}
	m.quickReplyMode = quickReplyModePane
	m.quickReplyInput.SetValue("")
	m.quickReplyInput.Focus()

	_, _ = m.updateQuickReply(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	gen := m.quickReplyStreamGen
	if gen == 0 || !m.quickReplyStreamFlush {
		t.Fatalf("expected stream flush scheduled gen=%d flush=%v", gen, m.quickReplyStreamFlush)
	}
	if string(m.quickReplyStreamBuf) != "a" {
		t.Fatalf("buf=%q", string(m.quickReplyStreamBuf))
	}

	_, _ = m.updateQuickReply(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("b")})
	if m.quickReplyStreamGen != gen {
		t.Fatalf("expected same gen, got %d want %d", m.quickReplyStreamGen, gen)
	}
	if string(m.quickReplyStreamBuf) != "ab" {
		t.Fatalf("buf=%q", string(m.quickReplyStreamBuf))
	}

	cmd := m.handleQuickReplyStreamFlush(quickReplyStreamFlushMsg{Gen: gen, PaneID: m.selectedPaneID()})
	if cmd == nil {
		t.Fatalf("expected cmd")
	}
	if _, ok := cmd().(ErrorMsg); !ok {
		t.Fatalf("expected ErrorMsg when client nil")
	}
	if m.quickReplyStreamFlush || len(m.quickReplyStreamBuf) != 0 {
		t.Fatalf("expected buffer cleared after flush")
	}
}

func TestQuickReplyStreamDoesNotSendSlashStart(t *testing.T) {
	m := newTestModelLite()
	m.config = &layout.Config{QuickReply: layout.QuickReplyConfig{StreamToPane: true}}
	m.quickReplyMode = quickReplyModePane
	m.quickReplyInput.SetValue("")
	m.quickReplyInput.Focus()

	_, _ = m.updateQuickReply(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	if len(m.quickReplyStreamBuf) != 0 || m.quickReplyStreamFlush {
		t.Fatalf("expected no streaming for slash start")
	}
}

func TestQuickReplyStreamEnterClearsInputWithoutDoubleSend(t *testing.T) {
	m := newTestModelLite()
	m.config = &layout.Config{QuickReply: layout.QuickReplyConfig{StreamToPane: true}}
	m.quickReplyMode = quickReplyModePane
	m.quickReplyInput.SetValue("")
	m.quickReplyInput.Focus()

	_, _ = m.updateQuickReply(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("h")})
	_, _ = m.updateQuickReply(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("i")})

	_, cmd := m.updateQuickReply(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatalf("expected cmd")
	}
	if _, ok := cmd().(ErrorMsg); !ok {
		t.Fatalf("expected ErrorMsg when client nil")
	}
	if got := m.quickReplyInput.Value(); got != "" {
		t.Fatalf("expected input cleared, got %q", got)
	}
	if len(m.quickReplyHistory) == 0 || m.quickReplyHistory[len(m.quickReplyHistory)-1] != "hi" {
		t.Fatalf("expected history to include %q", "hi")
	}
}

func TestQuickReplyStreamPaneSwitchDropsBufferedBytes(t *testing.T) {
	m := newTestModelLite()
	m.config = &layout.Config{QuickReply: layout.QuickReplyConfig{StreamToPane: true}}
	m.quickReplyMode = quickReplyModePane
	m.quickReplyInput.SetValue("")
	m.quickReplyInput.Focus()

	_, _ = m.updateQuickReply(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
	gen := m.quickReplyStreamGen
	if len(m.quickReplyStreamBuf) == 0 {
		t.Fatalf("expected buffered bytes")
	}

	sel := m.selection
	sel.Pane = "2"
	m.applySelection(sel)

	if len(m.quickReplyStreamBuf) != 0 || m.quickReplyStreamFlush {
		t.Fatalf("expected buffer dropped on pane switch")
	}

	cmd := m.handleQuickReplyStreamFlush(quickReplyStreamFlushMsg{Gen: gen, PaneID: "p1"})
	if cmd != nil {
		t.Fatalf("expected old flush ignored")
	}
}

func TestQuickReplyStreamNormalizesSpaceKey(t *testing.T) {
	m := newTestModelLite()
	m.config = &layout.Config{QuickReply: layout.QuickReplyConfig{StreamToPane: true}}
	m.quickReplyMode = quickReplyModePane
	m.quickReplyInput.SetValue("")
	m.quickReplyInput.Focus()

	_, _ = m.updateQuickReply(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("h")})
	_, _ = m.updateQuickReply(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("i")})
	_, _ = m.updateQuickReply(tea.KeyMsg{Type: tea.KeySpace})
	_, _ = m.updateQuickReply(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})

	if got := m.quickReplyInput.Value(); got != "hi x" {
		t.Fatalf("expected input %q, got %q", "hi x", got)
	}
	if got := string(m.quickReplyStreamBuf); got != "hi x" {
		t.Fatalf("expected stream buf %q, got %q", "hi x", got)
	}
}

func TestQuickReplyStreamStreamsCursorKeys(t *testing.T) {
	m := newTestModelLite()
	m.config = &layout.Config{QuickReply: layout.QuickReplyConfig{StreamToPane: true}}
	m.quickReplyMode = quickReplyModePane
	m.quickReplyInput.SetValue("")
	m.quickReplyInput.Focus()

	_, _ = m.updateQuickReply(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	_, _ = m.updateQuickReply(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("b")})
	_, _ = m.updateQuickReply(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("c")})
	_, _ = m.updateQuickReply(tea.KeyMsg{Type: tea.KeyLeft})
	_, _ = m.updateQuickReply(tea.KeyMsg{Type: tea.KeyLeft})
	_, _ = m.updateQuickReply(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("X")})

	if got := m.quickReplyInput.Value(); got != "aXbc" {
		t.Fatalf("expected input %q, got %q", "aXbc", got)
	}
	want := append([]byte("abc"), []byte("\x1b[D\x1b[D")...)
	want = append(want, 'X')
	if !bytes.Equal(m.quickReplyStreamBuf, want) {
		t.Fatalf("unexpected stream buf %q, want %q", string(m.quickReplyStreamBuf), string(want))
	}
}
