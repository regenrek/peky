package tmuxctl

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadLinkTTY(t *testing.T) {
	dir := t.TempDir()
	good := filepath.Join(dir, "tty-good")
	if err := os.Symlink("/dev/null", good); err != nil {
		t.Fatalf("Symlink() error: %v", err)
	}
	if got := readLinkTTY(good); got != "/dev/null" {
		t.Fatalf("readLinkTTY() = %q", got)
	}

	bad := filepath.Join(dir, "tty-bad")
	if err := os.Symlink("relative", bad); err != nil {
		t.Fatalf("Symlink() error: %v", err)
	}
	if got := readLinkTTY(bad); got != "" {
		t.Fatalf("readLinkTTY(bad) = %q", got)
	}
}
