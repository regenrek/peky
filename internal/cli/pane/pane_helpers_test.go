package pane

import (
	"bytes"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/regenrek/peakypanes/internal/cli/output"
	"github.com/regenrek/peakypanes/internal/cli/root"
	"github.com/regenrek/peakypanes/internal/limits"
	"github.com/regenrek/peakypanes/internal/native"
	"github.com/regenrek/peakypanes/internal/sessiond"
)

func TestTrimTrailingNewline(t *testing.T) {
	if got := string(trimTrailingNewline([]byte("hello\n"))); got != "hello" {
		t.Fatalf("trimTrailingNewline() = %q", got)
	}
	if got := string(trimTrailingNewline([]byte("hello"))); got != "hello" {
		t.Fatalf("trimTrailingNewline(no newline) = %q", got)
	}
	if got := string(trimTrailingNewline(nil)); got != "" {
		t.Fatalf("trimTrailingNewline(nil) = %q", got)
	}
}

func TestSummarizePayload(t *testing.T) {
	if got := summarizePayload([]byte("   ")); got != "" {
		t.Fatalf("summarizePayload(blank) = %q", got)
	}
	short := "hello world"
	if got := summarizePayload([]byte(short)); got != short {
		t.Fatalf("summarizePayload(short) = %q", got)
	}
	long := strings.Repeat("a", 130)
	got := summarizePayload([]byte(long))
	if len(got) != 120 || !strings.HasSuffix(got, "...") {
		t.Fatalf("summarizePayload(long) = %q", got)
	}
	limit := limits.PayloadInspectLimit
	large := append([]byte(strings.Repeat("a", limit)), []byte("SECRET")...)
	got = summarizePayload(large)
	if strings.Contains(got, "SECRET") {
		t.Fatalf("summarizePayload(limit) leaked suffix: %q", got)
	}
}

func TestSafePath(t *testing.T) {
	if _, err := safePath(""); err == nil {
		t.Fatalf("expected error for empty path")
	}
	abs := "/tmp/peaky"
	if got, err := safePath(abs); err != nil || got != abs {
		t.Fatalf("safePath(abs) = %q, err=%v", got, err)
	}
	rel := "relative/path"
	got, err := safePath(rel)
	if err != nil {
		t.Fatalf("safePath(rel) error: %v", err)
	}
	if !strings.HasSuffix(got, rel) {
		t.Fatalf("safePath(rel) = %q", got)
	}
}

func TestMapSendResultsAndStatus(t *testing.T) {
	resp := sessiond.SendInputResponse{
		Results: []sessiond.SendInputResult{
			{PaneID: "p1", Status: "", Message: "ok"},
			{PaneID: "p2", Status: "failed", Message: "nope"},
		},
	}
	results := mapSendResults(resp)
	if len(results) != 2 {
		t.Fatalf("mapSendResults() len=%d", len(results))
	}
	if results[0].Status != "ok" {
		t.Fatalf("default status = %q", results[0].Status)
	}
	if got := actionStatus(results); got != "partial" {
		t.Fatalf("actionStatus(partial) = %q", got)
	}
	if got := actionStatus([]output.TargetResult{{Status: "ok"}}); got != "ok" {
		t.Fatalf("actionStatus(ok) = %q", got)
	}
	if got := actionStatus([]output.TargetResult{{Status: "skipped"}}); got != "ok" {
		t.Fatalf("actionStatus(skipped) = %q", got)
	}
	if got := actionStatus([]output.TargetResult{{Status: "failed"}}); got != "failed" {
		t.Fatalf("actionStatus(failed) = %q", got)
	}
}

func TestParseTimeOrDuration(t *testing.T) {
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	past, err := parseTimeOrDuration("5s", now, false)
	if err != nil || !past.Equal(now.Add(-5*time.Second)) {
		t.Fatalf("parseTimeOrDuration(past) = %v, err=%v", past, err)
	}
	future, err := parseTimeOrDuration("5s", now, true)
	if err != nil || !future.Equal(now.Add(5*time.Second)) {
		t.Fatalf("parseTimeOrDuration(future) = %v, err=%v", future, err)
	}
	ts := "2025-01-01T01:02:03Z"
	parsed, err := parseTimeOrDuration(ts, now, true)
	if err != nil || parsed.Format(time.RFC3339) != ts {
		t.Fatalf("parseTimeOrDuration(rfc3339) = %v, err=%v", parsed, err)
	}
}

func TestFilterOutputLines(t *testing.T) {
	now := time.Now()
	lines := []native.OutputLine{
		{TS: now.Add(-2 * time.Minute), Text: "alpha"},
		{TS: now.Add(-1 * time.Minute), Text: "beta"},
		{TS: now, Text: "gamma"},
	}
	since := now.Add(-90 * time.Second)
	filtered := filterOutputLines(lines, since, time.Time{}, nil)
	if len(filtered) != 2 {
		t.Fatalf("filterOutputLines(since) len=%d", len(filtered))
	}
	until := now.Add(-30 * time.Second)
	filtered = filterOutputLines(lines, time.Time{}, until, nil)
	if len(filtered) != 2 {
		t.Fatalf("filterOutputLines(until) len=%d", len(filtered))
	}
	re := regexp.MustCompile("alpha|gamma")
	filtered = filterOutputLines(lines, time.Time{}, time.Time{}, re)
	if len(filtered) != 2 {
		t.Fatalf("filterOutputLines(regex) len=%d", len(filtered))
	}
}

func TestTerminalActionParsing(t *testing.T) {
	action, err := parseTerminalAction("scroll-up")
	if err != nil || action != sessiond.TerminalScrollUp {
		t.Fatalf("parseTerminalAction(scroll-up) = %v, err=%v", action, err)
	}
	if _, err := parseTerminalAction("unknown"); err == nil {
		t.Fatalf("expected error for unknown action")
	}
	if got := normalizeTerminalAction("Scroll Up"); got != "scroll_up" {
		t.Fatalf("normalizeTerminalAction() = %q", got)
	}
}

func TestDefaultActionLines(t *testing.T) {
	if got := defaultActionLines(sessiond.TerminalScrollUp, 0); got != 1 {
		t.Fatalf("defaultActionLines(scroll up) = %d", got)
	}
	if got := defaultActionLines(sessiond.TerminalCopyYank, 0); got != 0 {
		t.Fatalf("defaultActionLines(copy yank) = %d", got)
	}
	if got := defaultActionLines(sessiond.TerminalScrollUp, 5); got != 5 {
		t.Fatalf("defaultActionLines(value) = %d", got)
	}
}

func TestBuildKeyWithMods(t *testing.T) {
	if _, err := buildKeyWithMods("", nil); err == nil {
		t.Fatalf("expected error for empty key")
	}
	if got, err := buildKeyWithMods("k", nil); err != nil || got != "k" {
		t.Fatalf("buildKeyWithMods(k) = %q, err=%v", got, err)
	}
	got, err := buildKeyWithMods("k", []string{"CTRL", "shift", "alt"})
	if err != nil || got != "ctrl+shift+alt+k" {
		t.Fatalf("buildKeyWithMods(mods) = %q, err=%v", got, err)
	}
	if _, err := buildKeyWithMods("k", []string{"weird"}); err == nil {
		t.Fatalf("expected error for unknown modifier")
	}
}

func TestParseTailRegex(t *testing.T) {
	if got, err := parseTailRegex(""); err != nil || got != nil {
		t.Fatalf("parseTailRegex(empty) err=%v", err)
	}
	if _, err := parseTailRegex("("); err == nil {
		t.Fatalf("expected regex error")
	}
	if got, err := parseTailRegex("hello"); err != nil || got == nil {
		t.Fatalf("parseTailRegex(valid) err=%v", err)
	}
}

func TestSendWarnings(t *testing.T) {
	if got := sendWarnings(true, 2*time.Second); len(got) != 1 {
		t.Fatalf("sendWarnings() = %#v", got)
	}
	if got := sendWarnings(false, 2*time.Second); got != nil {
		t.Fatalf("sendWarnings(no newline) = %#v", got)
	}
}

func TestWriteSendLikeOutput(t *testing.T) {
	var out bytes.Buffer
	ctx := root.CommandContext{Out: &out}
	meta := output.NewMeta("pane.send", "dev")
	results := []output.TargetResult{
		{Status: "ok"},
		{Status: "ok"},
	}
	if err := writeSendLikeOutput(ctx, meta, time.Now(), "pane.send", results, nil); err != nil {
		t.Fatalf("writeSendLikeOutput() error: %v", err)
	}
	if !strings.Contains(out.String(), "Sent to 2 pane(s)") {
		t.Fatalf("writeSendLikeOutput() = %q", out.String())
	}
}

func TestToastKindString(t *testing.T) {
	if got := toastKindString(sessiond.ToastInfo); got != "info" {
		t.Fatalf("toastKindString(info) = %q", got)
	}
	if got := toastKindString(sessiond.ToastSuccess); got != "success" {
		t.Fatalf("toastKindString(success) = %q", got)
	}
	if got := toastKindString(sessiond.ToastWarning); got != "warning" {
		t.Fatalf("toastKindString(warning) = %q", got)
	}
	if got := toastKindString(sessiond.ToastLevel(99)); got != "unknown" {
		t.Fatalf("toastKindString(unknown) = %q", got)
	}
}

func TestRequireAck(t *testing.T) {
	if err := requireAck(strings.NewReader("ACK\n"), bytes.NewBuffer(nil)); err != nil {
		t.Fatalf("requireAck(ACK) error: %v", err)
	}
	if err := requireAck(strings.NewReader("nope\n"), bytes.NewBuffer(nil)); err == nil {
		t.Fatalf("expected error for missing ack")
	}
}
