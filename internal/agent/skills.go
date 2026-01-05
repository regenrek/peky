package agent

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type skill struct {
	Name        string
	Description string
	Body        string
	Path        string
}

type skillFrontmatter struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

func loadSkills(dir string) ([]skill, error) {
	if strings.TrimSpace(dir) == "" {
		return nil, nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read skills dir: %w", err)
	}
	var skills []skill
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		path := filepath.Join(dir, entry.Name(), "SKILL.md")
		info, err := entry.Info()
		if err != nil {
			return nil, fmt.Errorf("stat skill dir: %w", err)
		}
		if !info.IsDir() {
			continue
		}
		item, err := loadSkillFile(path)
		if err != nil {
			if errorsIsNotExist(err) {
				continue
			}
			return nil, err
		}
		skills = append(skills, item)
	}
	return skills, nil
}

func loadSkillFile(path string) (skill, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return skill{}, fmt.Errorf("read skill %q: %w", path, err)
	}
	front, body := splitFrontmatter(string(data))
	meta := skillFrontmatter{}
	if front != "" {
		if err := yaml.Unmarshal([]byte(front), &meta); err != nil {
			return skill{}, fmt.Errorf("parse skill frontmatter %q: %w", path, err)
		}
	}
	body = strings.TrimSpace(body)
	return skill{
		Name:        strings.TrimSpace(meta.Name),
		Description: strings.TrimSpace(meta.Description),
		Body:        body,
		Path:        path,
	}, nil
}

func splitFrontmatter(content string) (string, string) {
	trimmed := strings.TrimSpace(content)
	if !strings.HasPrefix(trimmed, "---") {
		return "", content
	}
	lines := strings.Split(trimmed, "\n")
	if len(lines) < 3 || strings.TrimSpace(lines[0]) != "---" {
		return "", content
	}
	end := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			end = i
			break
		}
	}
	if end == -1 {
		return "", content
	}
	front := strings.Join(lines[1:end], "\n")
	body := strings.Join(lines[end+1:], "\n")
	return front, body
}

func buildSkillsPrompt(skills []skill) string {
	if len(skills) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("## Skills\n")
	for _, s := range skills {
		name := s.Name
		if name == "" {
			name = filepath.Base(filepath.Dir(s.Path))
		}
		desc := s.Description
		if desc == "" {
			desc = "Skill instructions"
		}
		b.WriteString("\n### ")
		b.WriteString(name)
		b.WriteString("\n")
		b.WriteString(desc)
		b.WriteString("\n")
		if s.Body != "" {
			b.WriteString("\n")
			b.WriteString(s.Body)
			b.WriteString("\n")
		}
	}
	return strings.TrimSpace(b.String())
}

func errorsIsNotExist(err error) bool {
	return errors.Is(err, fs.ErrNotExist)
}
