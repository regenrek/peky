package sessionrestore

import (
	"testing"
	"time"
)

func TestParseMode(t *testing.T) {
	cases := []struct {
		in      string
		want    Mode
		wantErr bool
	}{
		{"", ModeDefault, false},
		{"true", ModeEnabled, false},
		{"enabled", ModeEnabled, false},
		{"false", ModeDisabled, false},
		{"disabled", ModeDisabled, false},
		{"private", ModePrivate, false},
		{"bogus", ModeDefault, true},
	}
	for _, tc := range cases {
		got, err := ParseMode(tc.in)
		if tc.wantErr && err == nil {
			t.Fatalf("ParseMode(%q) expected error", tc.in)
		}
		if !tc.wantErr && err != nil {
			t.Fatalf("ParseMode(%q) error: %v", tc.in, err)
		}
		if got != tc.want {
			t.Fatalf("ParseMode(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

func TestModeAllowsPersistence(t *testing.T) {
	if ModeDefault.AllowsPersistence(true) != true {
		t.Fatalf("ModeDefault should follow global true")
	}
	if ModeDefault.AllowsPersistence(false) != false {
		t.Fatalf("ModeDefault should follow global false")
	}
	if ModeEnabled.AllowsPersistence(false) != true {
		t.Fatalf("ModeEnabled should force true")
	}
	if ModeDisabled.AllowsPersistence(true) != false {
		t.Fatalf("ModeDisabled should force false")
	}
	if ModePrivate.AllowsPersistence(true) != false {
		t.Fatalf("ModePrivate should be false")
	}
}

func TestModeStringAndPrivate(t *testing.T) {
	if ModeEnabled.String() != "enabled" {
		t.Fatalf("ModeEnabled.String() = %q", ModeEnabled.String())
	}
	if !ModePrivate.IsPrivate() {
		t.Fatalf("ModePrivate should be private")
	}
	if ModeDisabled.IsPrivate() {
		t.Fatalf("ModeDisabled should not be private")
	}
}

func TestConfigNormalizedDefaults(t *testing.T) {
	cfg := Config{}
	n := cfg.Normalized()
	if n.SnapshotInterval != DefaultSnapshotInterval {
		t.Fatalf("SnapshotInterval = %v, want %v", n.SnapshotInterval, DefaultSnapshotInterval)
	}
	if n.MaxDiskBytes != int64(DefaultMaxDiskMB)*1024*1024 {
		t.Fatalf("MaxDiskBytes = %d", n.MaxDiskBytes)
	}
	if n.TTLInactive != DefaultTTLInactive {
		t.Fatalf("TTLInactive = %v, want %v", n.TTLInactive, DefaultTTLInactive)
	}
}

func TestRenderPlainLines(t *testing.T) {
	lines := RenderPlainLines(4, 3, []string{"old"}, []string{"A", "B"})
	if len(lines) != 3 {
		t.Fatalf("lines = %d", len(lines))
	}
	if lines[0] != "old " {
		t.Fatalf("line0 = %q", lines[0])
	}
	if lines[1] != "A   " || lines[2] != "B   " {
		t.Fatalf("lines = %#v", lines)
	}
}

func TestConfigNormalizedPreservesValues(t *testing.T) {
	cfg := Config{
		SnapshotInterval: 5 * time.Second,
		MaxDiskBytes:     1024,
		TTLInactive:      9 * time.Hour,
	}
	n := cfg.Normalized()
	if n.SnapshotInterval != cfg.SnapshotInterval {
		t.Fatalf("SnapshotInterval = %v, want %v", n.SnapshotInterval, cfg.SnapshotInterval)
	}
	if n.MaxDiskBytes != cfg.MaxDiskBytes {
		t.Fatalf("MaxDiskBytes = %d, want %d", n.MaxDiskBytes, cfg.MaxDiskBytes)
	}
	if n.TTLInactive != cfg.TTLInactive {
		t.Fatalf("TTLInactive = %v, want %v", n.TTLInactive, cfg.TTLInactive)
	}
}
