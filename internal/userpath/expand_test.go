package userpath

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExpandUser(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("cannot get home dir: %v", err)
	}

	tests := []struct {
		name string
		path string
		want string
	}{
		{"empty string", "", ""},
		{"tilde only", "~", home},
		{"tilde slash path", "~/Documents", filepath.Join(home, "Documents")},
		{"tilde slash nested", "~/a/b/c", filepath.Join(home, "a/b/c")},
		{"absolute path unchanged", "/usr/local/bin", "/usr/local/bin"},
		{"relative path unchanged", "foo/bar", "foo/bar"},
		{"tilde in middle unchanged", "/home/~user", "/home/~user"},
		{"tilde no slash unchanged", "~user", "~user"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExpandUser(tt.path)
			if got != tt.want {
				t.Errorf("ExpandUser(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestShortenUser(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("cannot get home dir: %v", err)
	}

	tests := []struct {
		name string
		path string
		want string
	}{
		{"empty string", "", ""},
		{"home dir exact", home, "~"},
		{"home subpath", filepath.Join(home, "Documents"), "~/Documents"},
		{"home nested", filepath.Join(home, "a/b/c"), "~/a/b/c"},
		{"non-home path unchanged", "/usr/local/bin", "/usr/local/bin"},
		{"relative path unchanged", "foo/bar", "foo/bar"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ShortenUser(tt.path)
			if got != tt.want {
				t.Errorf("ShortenUser(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestExpandUserShortenUserRoundtrip(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("cannot get home dir: %v", err)
	}

	// Expand then shorten should return original
	paths := []string{
		"~",
		"~/foo",
		"~/foo/bar/baz",
	}
	for _, p := range paths {
		expanded := ExpandUser(p)
		shortened := ShortenUser(expanded)
		if shortened != p {
			t.Errorf("roundtrip failed: %q -> %q -> %q", p, expanded, shortened)
		}
	}

	// Shorten then expand should return original (for home paths)
	homePaths := []string{
		home,
		filepath.Join(home, "test"),
		filepath.Join(home, "a/b/c"),
	}
	for _, p := range homePaths {
		shortened := ShortenUser(p)
		expanded := ExpandUser(shortened)
		if expanded != p {
			t.Errorf("roundtrip failed: %q -> %q -> %q", p, shortened, expanded)
		}
	}
}
