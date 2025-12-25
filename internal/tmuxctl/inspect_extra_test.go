package tmuxctl

import (
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestParsePanesFullVariants(t *testing.T) {
	now := time.Now().Unix()
	lines := []string{
		strings.Join([]string{
			"%1", "0", "1", "", "bash", "bash",
			"0", "1", "80", "24", "0", "0", "0",
		}, "\t"),
		strings.Join([]string{
			"%2", "1", "0", "Title", "zsh", "zsh",
			"1234", "5", "6", "70", "20", "1", "2", strconv.FormatInt(now, 10),
		}, "\t"),
	}
	out := strings.Join(lines, "\n")
	panes := parsePanesFull(out)
	if len(panes) != 2 {
		t.Fatalf("parsePanesFull() = %#v", panes)
	}
	if panes[0].Title != "bash" || panes[0].Command != "bash" {
		t.Fatalf("parsePanesFull() title fallback = %#v", panes[0])
	}
	if panes[1].PID != 1234 || !panes[1].Dead {
		t.Fatalf("parsePanesFull() = %#v", panes[1])
	}
	if panes[1].LastActive.IsZero() {
		t.Fatalf("parsePanesFull() lastActive missing")
	}
}

func TestParseInt64(t *testing.T) {
	if got := parseInt64("bad"); got != 0 {
		t.Fatalf("parseInt64(bad) = %d", got)
	}
	if got := parseInt64("42"); got != 42 {
		t.Fatalf("parseInt64(42) = %d", got)
	}
}
