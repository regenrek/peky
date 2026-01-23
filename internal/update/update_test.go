package update

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

func TestCompareVersions(t *testing.T) {
	cmp, err := CompareVersions("1.2.3", "1.2.4")
	if err != nil {
		t.Fatalf("CompareVersions error: %v", err)
	}
	if cmp >= 0 {
		t.Fatalf("expected current < latest")
	}

	cmp, err = CompareVersions("v1.2.3", "1.2.3")
	if err != nil {
		t.Fatalf("CompareVersions error: %v", err)
	}
	if cmp != 0 {
		t.Fatalf("expected versions equal")
	}
}

func TestIsDevelopmentVersion(t *testing.T) {
	cases := []struct {
		value string
		want  bool
	}{
		{"dev", true},
		{"unknown", true},
		{"v0.1.0-0.20251231235959-06c807842604", true},
		{"1.2.3", false},
	}
	for _, c := range cases {
		if got := IsDevelopmentVersion(c.value); got != c.want {
			t.Fatalf("IsDevelopmentVersion(%q) = %v, want %v", c.value, got, c.want)
		}
	}
}

func TestStateUpdateAvailableAndSkip(t *testing.T) {
	state := State{CurrentVersion: "1.0.0", LatestVersion: "1.1.0"}
	if !state.UpdateAvailable() {
		t.Fatalf("expected update available")
	}
	state.SkippedVersion = "1.1.0"
	if !state.IsSkipped() {
		t.Fatalf("expected update skipped")
	}
}

func TestPolicyShouldPromptCooldown(t *testing.T) {
	policy := DefaultPolicy()
	now := time.Now()
	state := State{CurrentVersion: "1.0.0", LatestVersion: "1.1.0"}
	state.MarkPrompted(now)
	if policy.ShouldPrompt(state, now.Add(1*time.Hour)) {
		t.Fatalf("expected cooldown to suppress prompt")
	}
	if !policy.ShouldPrompt(state, now.Add(73*time.Hour)) {
		t.Fatalf("expected prompt after cooldown")
	}
}

func TestFileStoreSaveLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "update-state.json")
	store := FileStore{Path: path}
	state := State{
		CurrentVersion:   "1.0.0",
		LatestVersion:    "1.2.0",
		SkippedVersion:   "",
		LastPromptUnixMs: 123,
		LastCheckUnixMs:  456,
		Channel:          ChannelNPMGlobal,
	}
	if err := store.Save(context.Background(), state); err != nil {
		t.Fatalf("Save error: %v", err)
	}
	loaded, err := store.Load(context.Background())
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	if loaded.CurrentVersion != state.CurrentVersion || loaded.LatestVersion != state.LatestVersion {
		t.Fatalf("loaded state mismatch: %#v", loaded)
	}
	if loaded.Channel != state.Channel {
		t.Fatalf("expected channel %q, got %q", state.Channel, loaded.Channel)
	}
}
