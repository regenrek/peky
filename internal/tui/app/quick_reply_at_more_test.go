package app

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestAtInputContextFor_Group(t *testing.T) {
	input := "echo @panes hi"
	cursor := len([]rune("echo @panes"))
	ctx := atInputContextFor(input, cursor)
	if !ctx.Active {
		t.Fatalf("expected active")
	}
	if ctx.Query != "panes" {
		t.Fatalf("Query=%q, want %q", ctx.Query, "panes")
	}
	if ctx.Group != atGroupPanes {
		t.Fatalf("Group=%v, want panes", ctx.Group)
	}
	if ctx.Start != len([]rune("echo ")) || ctx.End != len([]rune("echo @panes")) {
		t.Fatalf("range=%d..%d", ctx.Start, ctx.End)
	}
}

func TestAtGroupFromQuery(t *testing.T) {
	if atGroupFromQuery("pane") != atGroupPanes {
		t.Fatalf("expected panes")
	}
	if atGroupFromQuery("sessions") != atGroupSessions {
		t.Fatalf("expected sessions")
	}
	if atGroupFromQuery("files") != atGroupNone {
		t.Fatalf("expected none")
	}
}

func TestParseAtFileQuery(t *testing.T) {
	dir, q := parseAtFileQuery("")
	if dir != "" || q != "" {
		t.Fatalf("expected empty")
	}
	dir, q = parseAtFileQuery("foo")
	if dir != "" || q != "foo" {
		t.Fatalf("got dir=%q q=%q", dir, q)
	}
	dir, q = parseAtFileQuery("foo/bar")
	if dir != "foo" || q != "bar" {
		t.Fatalf("got dir=%q q=%q", dir, q)
	}
	dir, q = parseAtFileQuery("foo\\bar")
	if dir != "foo" || q != "bar" {
		t.Fatalf("got dir=%q q=%q", dir, q)
	}
	dir, q = parseAtFileQuery("/abs/path")
	if dir != "" || q != "" {
		t.Fatalf("expected abs rejected, got dir=%q q=%q", dir, q)
	}
}

func TestResolveAtFileRoot(t *testing.T) {
	root := t.TempDir()

	gotRoot, rel, err := resolveAtFileRoot(root, "")
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if gotRoot == "" || rel != "" {
		t.Fatalf("got root=%q rel=%q", gotRoot, rel)
	}

	gotRoot, rel, err = resolveAtFileRoot(root, ".")
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if gotRoot == "" || rel != "" {
		t.Fatalf("got root=%q rel=%q", gotRoot, rel)
	}

	gotRoot, rel, err = resolveAtFileRoot(root, "a/../b")
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if !strings.HasSuffix(gotRoot, string(filepath.Separator)+"b") || rel != "b" {
		t.Fatalf("got root=%q rel=%q", gotRoot, rel)
	}

	if _, _, err := resolveAtFileRoot(root, "../x"); err == nil {
		t.Fatalf("expected error for ..")
	}
	if _, _, err := resolveAtFileRoot(root, "/x"); err == nil {
		t.Fatalf("expected error for abs")
	}
}

func TestAtSuggestionEntries_FuzzyRootGroups(t *testing.T) {
	m := newTestModelLite()
	m.quickReplyInput.SetValue("@ses")
	m.quickReplyInput.SetCursor(len([]rune("@ses")))

	ctx := atInputContextFor(m.quickReplyInput.Value(), m.quickReplyInput.Position())
	if !ctx.Active || ctx.Group != atGroupNone || ctx.FileQuery != "ses" {
		t.Fatalf("unexpected ctx: %+v", ctx)
	}

	entries := m.atSuggestionEntries(ctx)
	if len(entries) == 0 {
		t.Fatalf("expected entries")
	}
	for _, e := range entries {
		if !strings.HasPrefix(e.Text, "@ses") {
			t.Fatalf("unexpected entry: %q", e.Text)
		}
	}
}

func TestApplyAtCompletion_PaneGroupAddsSpace(t *testing.T) {
	m := newTestModelLite()
	m.quickReplyInput.SetValue("@pane")
	m.quickReplyInput.SetCursor(len([]rune("@pane")))
	m.quickReplyMenuIndex = 0

	if !m.applyAtCompletion() {
		t.Fatalf("expected completion applied")
	}
	if got := m.quickReplyInput.Value(); got != "@allpanes " {
		t.Fatalf("value=%q, want %q", got, "@allpanes ")
	}
}
