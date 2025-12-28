package app

import (
	"errors"
	"testing"
)

func TestErrorMsgFormatting(t *testing.T) {
	err := errors.New("boom")
	msg := NewErrorMsg(err, "load")
	if msg.Error() != "load: boom" {
		t.Fatalf("ErrorMsg.Error() = %q", msg.Error())
	}
	plain := ErrorMsg{Err: err}
	if plain.Error() != "boom" {
		t.Fatalf("ErrorMsg.Error() = %q", plain.Error())
	}
}

func TestStatusFormatHelpers(t *testing.T) {
	if FormatStatusError(errors.New("fail")) == "" {
		t.Fatalf("FormatStatusError returned empty")
	}
	if FormatStatusSuccess("ok") == "" {
		t.Fatalf("FormatStatusSuccess returned empty")
	}
	if FormatStatusWarning("warn") == "" {
		t.Fatalf("FormatStatusWarning returned empty")
	}
	if FormatStatusInfo("info") == "" {
		t.Fatalf("FormatStatusInfo returned empty")
	}
}

func TestMessageCommands(t *testing.T) {
	if msg := NewErrorCmd(errors.New("boom"), "ctx")(); msg.(ErrorMsg).Context != "ctx" {
		t.Fatalf("NewErrorCmd() = %#v", msg)
	}
	if msg := NewSuccessCmd("ok")(); msg.(SuccessMsg).Message != "ok" {
		t.Fatalf("NewSuccessCmd() = %#v", msg)
	}
	if msg := NewWarningCmd("warn")(); msg.(WarningMsg).Message != "warn" {
		t.Fatalf("NewWarningCmd() = %#v", msg)
	}
}
