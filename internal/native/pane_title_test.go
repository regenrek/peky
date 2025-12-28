package native

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestExtractPathFromTitle(t *testing.T) {
	cases := []struct {
		name  string
		title string
		want  string
		ok    bool
	}{
		{name: "absolute", title: "/Users/me/projects/aidex", want: "/Users/me/projects/aidex", ok: true},
		{name: "host", title: "user@host:~/projects/cli/aidex", want: "~/projects/cli/aidex", ok: true},
		{name: "dash", title: "shell - /tmp/app", want: "/tmp/app", ok: true},
		{name: "non-path", title: "codex", want: "", ok: false},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := extractPathFromTitle(tt.title)
			if ok != tt.ok {
				t.Fatalf("extractPathFromTitle(%q) ok=%v want %v", tt.title, ok, tt.ok)
			}
			if got != tt.want {
				t.Fatalf("extractPathFromTitle(%q)=%q want %q", tt.title, got, tt.want)
			}
		})
	}
}

func TestShortenPathTitle(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("home directory unavailable")
	}

	sessionPath := filepath.Join(home, "projects", "cli", "aidex")
	cases := []struct {
		name    string
		title   string
		session string
		want    string
	}{
		{name: "session root", title: filepath.Join(home, "projects", "cli", "aidex"), session: sessionPath, want: "aidex"},
		{name: "session subdir", title: filepath.Join(home, "projects", "cli", "aidex", "src"), session: sessionPath, want: "aidex:src"},
		{name: "session deep", title: filepath.Join(home, "projects", "cli", "aidex", "services", "api"), session: sessionPath, want: "aidex:services/api"},
		{name: "outside session", title: filepath.Join(home, "projects", "other", "repo"), session: sessionPath, want: "~/other/repo"},
		{name: "tilde path", title: "~/projects/cli/aidex", session: sessionPath, want: "aidex"},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			got := shortenPathTitle(tt.title, tt.session)
			if got != tt.want {
				t.Fatalf("shortenPathTitle(%q, %q)=%q want %q", tt.title, tt.session, got, tt.want)
			}
		})
	}
}

func TestResolvePaneTitle(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("home directory unavailable")
	}
	root := filepath.Join(home, "projects", "cli", "aidex")
	windowPath := filepath.Join(root, "src")

	title, kind := resolvePaneTitle(root, "server", "codex", "0")
	if title != "codex" || kind != paneTitleWindow {
		t.Fatalf("window title override = %q kind %v", title, kind)
	}

	title, kind = resolvePaneTitle(root, "server", fmt.Sprintf("user@host:%s", windowPath), "0")
	if title != "server" || kind != paneTitleExplicit {
		t.Fatalf("explicit title with path window = %q kind %v", title, kind)
	}

	title, kind = resolvePaneTitle(root, "", fmt.Sprintf("user@host:%s", windowPath), "1")
	if title != "aidex:src" || kind != paneTitlePath {
		t.Fatalf("path window title = %q kind %v", title, kind)
	}

	title, kind = resolvePaneTitle(root, "", "", "2")
	if title != "pane 2" || kind != paneTitleFallback {
		t.Fatalf("fallback title = %q kind %v", title, kind)
	}
}

func TestDedupePaneTitles(t *testing.T) {
	paneA := &Pane{Index: "0"}
	paneB := &Pane{Index: "1"}
	paneC := &Pane{Index: "2"}

	entries := []paneTitleEntry{
		{pane: paneA, title: "aidex", kind: paneTitlePath},
		{pane: paneB, title: "aidex", kind: paneTitlePath},
		{pane: paneC, title: "server", kind: paneTitleExplicit},
	}
	result := dedupePaneTitles(entries)
	if result[paneA] != "aidex" || result[paneB] != "aidex #2" {
		t.Fatalf("dedupe path titles = %#v", result)
	}
	if result[paneC] != "server" {
		t.Fatalf("dedupe explicit title = %#v", result[paneC])
	}

	entries = []paneTitleEntry{
		{pane: paneA, title: "shell", kind: paneTitleExplicit},
		{pane: paneB, title: "shell", kind: paneTitlePath},
	}
	result = dedupePaneTitles(entries)
	if result[paneA] != "shell" || result[paneB] != "shell #2" {
		t.Fatalf("dedupe mixed titles = %#v", result)
	}
}
