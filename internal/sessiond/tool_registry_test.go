package sessiond

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/regenrek/peakypanes/internal/runenv"
)

func TestLoadToolRegistryConfigPathDirectory(t *testing.T) {
	configDir := filepath.Join(t.TempDir(), "cfg")
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}
	path := filepath.Join(configDir, "config.yml")
	if err := os.MkdirAll(path, 0o700); err != nil {
		t.Fatalf("mkdir config path: %v", err)
	}
	if err := os.Setenv(runenv.ConfigDirEnv, configDir); err != nil {
		t.Fatalf("setenv: %v", err)
	}
	defer os.Unsetenv(runenv.ConfigDirEnv)

	_, err := loadToolRegistry()
	if err == nil {
		t.Fatalf("expected error for directory config path")
	}
	if !strings.Contains(err.Error(), "is a directory") {
		t.Fatalf("unexpected error: %v", err)
	}
}
