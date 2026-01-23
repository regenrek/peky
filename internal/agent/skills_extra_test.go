package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSplitFrontmatterExtra(t *testing.T) {
	content := "---\nname: Test\n---\nBody\nLine\n"
	front, body := splitFrontmatter(content)
	if !strings.Contains(front, "name: Test") {
		t.Fatalf("front=%q", front)
	}
	if strings.TrimSpace(body) != "Body\nLine" {
		t.Fatalf("body=%q", body)
	}
	front, body = splitFrontmatter("No frontmatter")
	if front != "" || body != "No frontmatter" {
		t.Fatalf("front=%q body=%q", front, body)
	}
}

func TestBuildSkillsPrompt(t *testing.T) {
	prompt := buildSkillsPrompt([]skill{
		{Name: "Alpha", Description: "Desc", Body: "Do X", Path: "/tmp/alpha/SKILL.md"},
		{Name: "", Description: "", Body: "", Path: "/tmp/beta/SKILL.md"},
	})
	if !strings.Contains(prompt, "## Skills") || !strings.Contains(prompt, "Alpha") {
		t.Fatalf("prompt=%q", prompt)
	}
	if !strings.Contains(prompt, "Skill instructions") {
		t.Fatalf("expected default description")
	}
}

func TestLoadSkillFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "SKILL.md")
	content := "---\nname: Demo\ndescription: Test skill\n---\nBody"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write skill: %v", err)
	}
	sk, err := loadSkillFile(path)
	if err != nil {
		t.Fatalf("loadSkillFile error: %v", err)
	}
	if sk.Name != "Demo" || sk.Description != "Test skill" || sk.Body != "Body" {
		t.Fatalf("skill=%#v", sk)
	}
}
