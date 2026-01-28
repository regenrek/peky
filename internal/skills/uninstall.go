package skills

import (
	"errors"
	"fmt"
	"path/filepath"
)

type UninstallOptions struct {
	Targets      []Target
	SkillIDs     []string
	DestOverride string
}

type UninstallRecord struct {
	SkillID string
	Target  Target
	Path    string
	Status  string
	Message string
}

type UninstallResult struct {
	Records []UninstallRecord
}

func Uninstall(bundle Bundle, opts UninstallOptions) (UninstallResult, error) {
	if len(opts.Targets) == 0 {
		return UninstallResult{}, errors.New("targets are required")
	}
	if opts.DestOverride != "" && len(opts.Targets) != 1 {
		return UninstallResult{}, errors.New("dest override requires exactly one target")
	}
	skills, err := filterSkills(bundle.Skills, opts.SkillIDs)
	if err != nil {
		return UninstallResult{}, err
	}
	records := make([]UninstallRecord, 0, len(skills)*len(opts.Targets))
	for _, skill := range skills {
		for _, target := range opts.Targets {
			if !skillSupportsTarget(skill, target) {
				records = append(records, UninstallRecord{
					SkillID: skill.ID,
					Target:  target,
					Status:  "skipped",
					Message: "not available for target",
				})
				continue
			}
			root, err := resolveTargetRoot(target, opts.DestOverride)
			if err != nil {
				return UninstallResult{}, err
			}
			destDir := filepath.Join(root, skill.ID)
			exists, err := pathExists(destDir)
			if err != nil {
				return UninstallResult{}, err
			}
			if !exists {
				records = append(records, UninstallRecord{
					SkillID: skill.ID,
					Target:  target,
					Path:    destDir,
					Status:  "skipped",
					Message: "not installed",
				})
				continue
			}
			if err := trashPath(destDir); err != nil {
				return UninstallResult{}, fmt.Errorf("trash %s: %w", destDir, err)
			}
			records = append(records, UninstallRecord{
				SkillID: skill.ID,
				Target:  target,
				Path:    destDir,
				Status:  "removed",
			})
		}
	}
	return UninstallResult{Records: records}, nil
}
