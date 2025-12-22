package zellijctl

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureBridgePluginWritesAndMatches(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmp, "xdg"))

	path, err := EnsureBridgePlugin()
	if err != nil {
		t.Fatalf("EnsureBridgePlugin: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat bridge: %v", err)
	}
	if info.Size() == 0 {
		t.Fatalf("bridge file is empty")
	}

	match, err := fileMatchesHash(path, bridgeWasm)
	if err != nil {
		t.Fatalf("fileMatchesHash: %v", err)
	}
	if !match {
		t.Fatalf("expected bridge file to match embedded hash")
	}

	path2, err := EnsureBridgePlugin()
	if err != nil {
		t.Fatalf("EnsureBridgePlugin second: %v", err)
	}
	if path2 != path {
		t.Fatalf("expected same bridge path, got %q and %q", path, path2)
	}
}
