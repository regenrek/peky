//go:build unix

package terminal

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/muesli/termenv"
)

func TestWindowScrollbackCopySmoke(t *testing.T) {
	w := newCatWindow(t, 20, 3)

	for i := 0; i < 9; i++ {
		line := fmt.Sprintf("line%02d\n", i)
		if err := w.SendInput([]byte(line)); err != nil {
			t.Fatalf("send input: %v", err)
		}
	}

	waitFor(t, 2*time.Second, "scrollback populated", func() bool {
		return w.ScrollbackLen() >= 4
	})

	w.ScrollToTop()
	view := w.ViewLipgloss(false, termenv.TrueColor)
	lines := trimLinesSmoke(view)
	if !containsLineSmoke(lines, "line00") {
		t.Fatalf("expected scrollback to show line00, got %q", lines)
	}

	w.EnterCopyMode()
	w.CopyMove(0, -999)
	w.CopyToggleSelect()
	w.CopyMove(6, 0)
	text := w.CopyYankText()
	if !strings.Contains(text, "line00") {
		t.Fatalf("expected yank to contain line00, got %q", text)
	}
}

func TestWindowScrollbackAnchorsOnNewOutput(t *testing.T) {
	w := newCatWindow(t, 20, 3)

	for i := 0; i < 6; i++ {
		line := fmt.Sprintf("line%02d\n", i)
		if err := w.SendInput([]byte(line)); err != nil {
			t.Fatalf("send input: %v", err)
		}
	}

	waitFor(t, 4*time.Second, "scrollback populated", func() bool {
		return w.ScrollbackLen() >= 2
	})

	w.ScrollUp(2)
	oldOffset := w.GetScrollbackOffset()
	oldSB := w.ScrollbackLen()
	if oldOffset == 0 || oldSB == 0 {
		t.Fatalf("expected scrollback state, offset=%d sb=%d", oldOffset, oldSB)
	}

	if err := w.SendInput([]byte("line99\n")); err != nil {
		t.Fatalf("send input: %v", err)
	}

	waitFor(t, 4*time.Second, "scrollback grows", func() bool {
		return w.ScrollbackLen() > oldSB
	})

	newSB := w.ScrollbackLen()
	expected := oldOffset + (newSB - oldSB)
	waitFor(t, 4*time.Second, "offset anchored", func() bool {
		return w.GetScrollbackOffset() == expected
	})
}

func newCatWindow(t *testing.T, cols, rows int) *Window {
	w, err := NewWindow(Options{
		ID:      "test-smoke",
		Command: "sh",
		Args:    []string{"-c", "stty -echo; cat"},
		Cols:    cols,
		Rows:    rows,
	})
	if err != nil {
		t.Fatalf("new window: %v", err)
	}
	t.Cleanup(func() {
		_ = w.Close()
	})
	return w
}

func waitFor(t *testing.T, timeout time.Duration, label string, fn func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if fn() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timeout waiting for %s", label)
}

func trimLinesSmoke(s string) []string {
	raw := strings.Split(s, "\n")
	out := make([]string, 0, len(raw))
	for _, ln := range raw {
		out = append(out, strings.TrimRight(ln, " "))
	}
	return out
}

func containsLineSmoke(lines []string, target string) bool {
	for _, ln := range lines {
		if ln == target {
			return true
		}
	}
	return false
}
