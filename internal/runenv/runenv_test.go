package runenv

import (
	"testing"
	"time"
)

func TestStartSessionTimeoutDefault(t *testing.T) {
	t.Setenv(StartSessionTimeoutEnv, "")
	if got := StartSessionTimeout(); got != 5*time.Second {
		t.Fatalf("expected default timeout 5s, got %v", got)
	}
}

func TestStartSessionTimeoutDuration(t *testing.T) {
	t.Setenv(StartSessionTimeoutEnv, "12s")
	if got := StartSessionTimeout(); got != 12*time.Second {
		t.Fatalf("expected 12s, got %v", got)
	}
}

func TestStartSessionTimeoutSecondsNumber(t *testing.T) {
	t.Setenv(StartSessionTimeoutEnv, "9")
	if got := StartSessionTimeout(); got != 9*time.Second {
		t.Fatalf("expected 9s, got %v", got)
	}
}

func TestStartSessionTimeoutInvalid(t *testing.T) {
	t.Setenv(StartSessionTimeoutEnv, "nope")
	if got := StartSessionTimeout(); got != 5*time.Second {
		t.Fatalf("expected default timeout on invalid value, got %v", got)
	}
}

func TestStartSessionTimeoutNonPositive(t *testing.T) {
	t.Setenv(StartSessionTimeoutEnv, "-3")
	if got := StartSessionTimeout(); got != 5*time.Second {
		t.Fatalf("expected default timeout on non-positive value, got %v", got)
	}
	t.Setenv(StartSessionTimeoutEnv, "0s")
	if got := StartSessionTimeout(); got != 5*time.Second {
		t.Fatalf("expected default timeout on zero duration, got %v", got)
	}
}

func TestDataDirEnv(t *testing.T) {
	t.Setenv(DataDirEnv, "/tmp/pp-data")
	if got := DataDir(); got != "/tmp/pp-data" {
		t.Fatalf("DataDir() = %q, want %q", got, "/tmp/pp-data")
	}
}

func TestDataDirEmpty(t *testing.T) {
	t.Setenv(DataDirEnv, "")
	if got := DataDir(); got != "" {
		t.Fatalf("DataDir() = %q, want empty", got)
	}
}
