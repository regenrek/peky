package skills

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Manifest struct {
	Version int            `json:"version"`
	Skills  []ManifestItem `json:"skills"`
}

type ManifestItem struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Targets     []Target `json:"targets"`
	Path        string   `json:"path"`
}

// Skill describes a bundled skill with resolved source path.
type Skill struct {
	ID          string
	Name        string
	Description string
	Targets     []Target
	SourceDir   string
}

func loadManifest(root string) (Manifest, error) {
	path := filepath.Join(root, manifestFilename)
	data, err := os.ReadFile(path)
	if err != nil {
		return Manifest{}, fmt.Errorf("read manifest: %w", err)
	}
	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return Manifest{}, fmt.Errorf("parse manifest: %w", err)
	}
	if err := manifest.validate(); err != nil {
		return Manifest{}, err
	}
	return manifest, nil
}

func (m Manifest) validate() error {
	if m.Version <= 0 {
		return errors.New("manifest version must be positive")
	}
	if len(m.Skills) == 0 {
		return errors.New("manifest has no skills")
	}
	seen := make(map[string]struct{})
	for _, item := range m.Skills {
		id := strings.TrimSpace(item.ID)
		if id == "" {
			return errors.New("skill id is required")
		}
		if _, ok := seen[id]; ok {
			return fmt.Errorf("duplicate skill id %q", id)
		}
		seen[id] = struct{}{}
		if strings.TrimSpace(item.Path) == "" {
			return fmt.Errorf("skill %q path is required", id)
		}
		if len(item.Targets) == 0 {
			return fmt.Errorf("skill %q targets are required", id)
		}
		if _, err := cleanRelPath(item.Path); err != nil {
			return fmt.Errorf("skill %q invalid path: %w", id, err)
		}
		for _, target := range item.Targets {
			if _, ok := targetFromString(string(target)); !ok {
				return fmt.Errorf("skill %q has unknown target %q", id, target)
			}
		}
	}
	return nil
}

func (m Manifest) resolveSkills(root string) ([]Skill, error) {
	out := make([]Skill, 0, len(m.Skills))
	for _, item := range m.Skills {
		rel, err := cleanRelPath(item.Path)
		if err != nil {
			return nil, fmt.Errorf("skill %q path: %w", item.ID, err)
		}
		source := filepath.Join(root, rel)
		skill := Skill{
			ID:          strings.TrimSpace(item.ID),
			Name:        strings.TrimSpace(item.Name),
			Description: strings.TrimSpace(item.Description),
			Targets:     normalizeTargets(item.Targets),
			SourceDir:   source,
		}
		if err := skill.validate(); err != nil {
			return nil, fmt.Errorf("skill %q: %w", item.ID, err)
		}
		out = append(out, skill)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func (s Skill) validate() error {
	if s.ID == "" {
		return errors.New("id is required")
	}
	if s.SourceDir == "" {
		return errors.New("source dir is required")
	}
	info, err := os.Stat(s.SourceDir)
	if err != nil {
		return fmt.Errorf("stat source dir: %w", err)
	}
	if !info.IsDir() {
		return errors.New("source path is not a directory")
	}
	return nil
}

func normalizeTargets(targets []Target) []Target {
	seen := make(map[Target]struct{})
	out := make([]Target, 0, len(targets))
	for _, target := range targets {
		if _, ok := seen[target]; ok {
			continue
		}
		seen[target] = struct{}{}
		out = append(out, target)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

func cleanRelPath(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", errors.New("path is empty")
	}
	if filepath.IsAbs(value) {
		return "", errors.New("path must be relative")
	}
	clean := filepath.Clean(value)
	if clean == "." || strings.HasPrefix(clean, "..") || strings.Contains(clean, ".."+string(filepath.Separator)) {
		return "", errors.New("path must not traverse parent directories")
	}
	return clean, nil
}
