package skills

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type InstallMode string

const (
	InstallCopy    InstallMode = "copy"
	InstallSymlink InstallMode = "symlink"
)

type InstallOptions struct {
	Targets      []Target
	SkillIDs     []string
	Mode         InstallMode
	DestOverride string
	Overwrite    bool
}

type InstallRecord struct {
	SkillID string
	Target  Target
	Path    string
	Status  string
	Message string
}

type InstallResult struct {
	Records []InstallRecord
}

func (o *InstallOptions) validate() error {
	if len(o.Targets) == 0 {
		return errors.New("targets are required")
	}
	if o.Mode == "" {
		o.Mode = InstallCopy
	}
	switch o.Mode {
	case InstallCopy, InstallSymlink:
	default:
		return fmt.Errorf("unsupported install mode %q", o.Mode)
	}
	if o.DestOverride != "" && len(o.Targets) != 1 {
		return errors.New("dest override requires exactly one target")
	}
	return nil
}

// Install installs the requested skills into each target root.
func Install(bundle Bundle, opts InstallOptions) (InstallResult, error) {
	if err := opts.validate(); err != nil {
		return InstallResult{}, err
	}
	skills, err := filterSkills(bundle.Skills, opts.SkillIDs)
	if err != nil {
		return InstallResult{}, err
	}
	records := make([]InstallRecord, 0, len(skills)*len(opts.Targets))
	for _, skill := range skills {
		for _, target := range opts.Targets {
			if !skillSupportsTarget(skill, target) {
				records = append(records, InstallRecord{
					SkillID: skill.ID,
					Target:  target,
					Status:  "skipped",
					Message: "not available for target",
				})
				continue
			}
			destRoot, err := resolveTargetRoot(target, opts.DestOverride)
			if err != nil {
				return InstallResult{}, err
			}
			destDir := filepath.Join(destRoot, skill.ID)
			if exists, err := pathExists(destDir); err != nil {
				return InstallResult{}, err
			} else if exists {
				if !opts.Overwrite {
					records = append(records, InstallRecord{
						SkillID: skill.ID,
						Target:  target,
						Path:    destDir,
						Status:  "skipped",
						Message: "already installed",
					})
					continue
				}
				if err := trashPath(destDir); err != nil {
					return InstallResult{}, fmt.Errorf("trash %s: %w", destDir, err)
				}
			}
			if err := os.MkdirAll(destRoot, 0o755); err != nil {
				return InstallResult{}, fmt.Errorf("create target dir: %w", err)
			}
			if err := installSkillDir(skill.SourceDir, destDir, opts.Mode); err != nil {
				return InstallResult{}, err
			}
			records = append(records, InstallRecord{
				SkillID: skill.ID,
				Target:  target,
				Path:    destDir,
				Status:  "installed",
			})
		}
	}
	return InstallResult{Records: records}, nil
}

func installSkillDir(srcDir, destDir string, mode InstallMode) error {
	switch mode {
	case InstallCopy:
		return copyDir(srcDir, destDir)
	case InstallSymlink:
		if err := os.Symlink(srcDir, destDir); err != nil {
			return fmt.Errorf("symlink %s: %w", destDir, err)
		}
		return nil
	default:
		return fmt.Errorf("unsupported install mode %q", mode)
	}
}

func resolveTargetRoot(target Target, override string) (string, error) {
	if override != "" {
		return override, nil
	}
	return TargetRoot(target)
}

func filterSkills(skills []Skill, ids []string) ([]Skill, error) {
	if len(ids) == 0 {
		return skills, nil
	}
	wantAll := false
	want := make(map[string]struct{})
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if id == "all" {
			wantAll = true
			break
		}
		want[id] = struct{}{}
	}
	if wantAll {
		return skills, nil
	}
	if len(want) == 0 {
		return nil, errors.New("skill is required")
	}
	out := make([]Skill, 0, len(want))
	for _, skill := range skills {
		if _, ok := want[skill.ID]; ok {
			out = append(out, skill)
		}
	}
	if len(out) == 0 {
		return nil, errors.New("no matching skills")
	}
	return out, nil
}

func skillSupportsTarget(skill Skill, target Target) bool {
	for _, t := range skill.Targets {
		if t == target {
			return true
		}
	}
	return false
}
