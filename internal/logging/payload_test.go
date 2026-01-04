package logging

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/regenrek/peakypanes/internal/limits"
)

func TestPayloadAttrRedactsByDefault(t *testing.T) {
	payload := []byte("secret")
	attr := PayloadAttr("payload", payload)
	got := attr.Value.String()
	if !strings.Contains(got, "redacted(") {
		t.Fatalf("expected redacted payload, got %q", got)
	}
	if strings.Contains(got, "secret") {
		t.Fatalf("expected payload to be redacted, got %q", got)
	}
}

func TestRedactedPayloadHashUsesPrefix(t *testing.T) {
	limit := limits.PayloadInspectLimit
	if limit <= 0 {
		t.Fatalf("expected payload inspect limit > 0")
	}
	base := strings.Repeat("a", limit)
	payload1 := []byte(base + "SECRET_ONE")
	payload2 := []byte(base + "SECRET_TWO")

	got1 := redactedPayloadString(payload1)
	got2 := redactedPayloadString(payload2)

	if got1 != got2 {
		t.Fatalf("expected same hash for same prefix, got %q vs %q", got1, got2)
	}
	if !strings.Contains(got1, fmt.Sprintf("len=%d", len(payload1))) {
		t.Fatalf("expected full length in redaction, got %q", got1)
	}
	if strings.Contains(got1, "SECRET") {
		t.Fatalf("expected secrets to be redacted, got %q", got1)
	}
}

func TestPayloadAttrIncludesPreviewWhenEnabled(t *testing.T) {
	sink := string(SinkNone)
	include := true
	cfg := Config{Sink: &sink, IncludePayloads: &include}
	closeFn, err := Init(context.Background(), cfg, InitOptions{App: "test", Mode: ModeCLI})
	if err != nil {
		t.Fatalf("Init() error: %v", err)
	}
	t.Cleanup(func() {
		disable := false
		_, _ = Init(context.Background(), Config{Sink: &sink, IncludePayloads: &disable}, InitOptions{App: "test", Mode: ModeCLI})
		if closeFn != nil {
			_ = closeFn()
		}
	})

	payload := []byte("hello")
	attr := PayloadAttr("payload", payload)
	got := attr.Value.String()
	want := fmt.Sprintf("%q", payload)
	if got != want {
		t.Fatalf("expected preview %q, got %q", want, got)
	}
}
