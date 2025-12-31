package output

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestWriteSuccess(t *testing.T) {
	var buf bytes.Buffer
	meta := NewMeta("session.list", "test")
	data := map[string]any{"ok": true}
	if err := WriteSuccess(&buf, meta, data); err != nil {
		t.Fatalf("WriteSuccess err=%v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal(buf.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal err=%v", err)
	}
	if ok, _ := payload["ok"].(bool); !ok {
		t.Fatalf("expected ok true")
	}
}

func TestWriteError(t *testing.T) {
	var buf bytes.Buffer
	meta := NewMeta("session.list", "test")
	if err := WriteError(&buf, meta, "bad", "no", nil); err != nil {
		t.Fatalf("WriteError err=%v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal(buf.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal err=%v", err)
	}
	if ok, _ := payload["ok"].(bool); ok {
		t.Fatalf("expected ok false")
	}
}
