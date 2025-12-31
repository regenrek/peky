package output

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

type errWriter struct{}

func (errWriter) Write([]byte) (int, error) {
	return 0, errors.New("write failed")
}

func TestNewMetaAndStreamMeta(t *testing.T) {
	meta := NewMeta("cmd.test", "1.2.3")
	if meta.Command != "cmd.test" {
		t.Fatalf("expected command set, got %q", meta.Command)
	}
	if meta.SchemaVersion != SchemaVersion {
		t.Fatalf("expected schema version %q, got %q", SchemaVersion, meta.SchemaVersion)
	}
	if meta.Version != "1.2.3" {
		t.Fatalf("expected version set, got %q", meta.Version)
	}
	stream := NewStreamMeta("cmd.test", "1.2.3", 7, true)
	if !stream.Stream || stream.Seq != 7 || !stream.EOF {
		t.Fatalf("expected stream meta flags set, got stream=%v seq=%d eof=%v", stream.Stream, stream.Seq, stream.EOF)
	}
	start := time.Now().Add(-2 * time.Second)
	withDuration := WithDuration(meta, start)
	if withDuration.DurationMS <= 0 {
		t.Fatalf("expected duration > 0, got %f", withDuration.DurationMS)
	}
}

func TestWriteErrorDefaults(t *testing.T) {
	buf := &bytes.Buffer{}
	meta := NewMeta("cmd.test", "1.2.3")
	if err := WriteError(buf, meta, "", "", nil); err != nil {
		t.Fatalf("WriteError: %v", err)
	}
	var env ErrorEnvelope
	if err := json.Unmarshal(buf.Bytes(), &env); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if env.Ok {
		t.Fatalf("expected ok=false")
	}
	if env.Error.Code != "unknown" || env.Error.Message != "unknown error" {
		t.Fatalf("unexpected defaults: %+v", env.Error)
	}
}

func TestWriteSuccessErrorPropagation(t *testing.T) {
	meta := NewMeta("cmd.test", "1.2.3")
	err := WriteSuccess(errWriter{}, meta, map[string]string{"ok": "true"})
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "encode json") {
		t.Fatalf("expected encode json error, got %v", err)
	}
}
