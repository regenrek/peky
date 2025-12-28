package picker

import (
	"strings"
	"testing"
)

func TestProjectItem(t *testing.T) {
	item := ProjectItem{Name: "repo", Path: "/tmp/repo", DisplayPath: "/tmp/repo"}
	if item.Title() != "📁 repo" {
		t.Fatalf("ProjectItem.Title() = %q", item.Title())
	}
	if item.Description() == "" {
		t.Fatalf("ProjectItem.Description() empty")
	}
	if item.FilterValue() != "repo" {
		t.Fatalf("ProjectItem.FilterValue() = %q", item.FilterValue())
	}
}

func TestLayoutChoice(t *testing.T) {
	item := LayoutChoice{Label: "dev", Desc: "layout", LayoutName: "dev"}
	if item.Title() != "dev" {
		t.Fatalf("LayoutChoice.Title() = %q", item.Title())
	}
	if item.Description() == "" {
		t.Fatalf("LayoutChoice.Description() empty")
	}
	if item.FilterValue() != "dev" {
		t.Fatalf("LayoutChoice.FilterValue() = %q", item.FilterValue())
	}
}

func TestCommandItem(t *testing.T) {
	item := CommandItem{Label: "Cmd", Desc: "desc"}
	if !strings.Contains(item.FilterValue(), "cmd") {
		t.Fatalf("CommandItem.FilterValue() = %q", item.FilterValue())
	}
}
