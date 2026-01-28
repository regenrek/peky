package skills

import (
	"fmt"
	"path/filepath"
)

type StatusRecord struct {
	SkillID  string
	Target   Target
	Path     string
	Present  bool
	Matches  bool
	ErrorMsg string
}

// Status returns install status for the requested targets.
func Status(bundle Bundle, targets []Target, destOverride string) ([]StatusRecord, error) {
	if len(targets) == 0 {
		return nil, fmt.Errorf("targets are required")
	}
	if destOverride != "" && len(targets) != 1 {
		return nil, fmt.Errorf("dest override requires exactly one target")
	}
	records := make([]StatusRecord, 0, len(bundle.Skills)*len(targets))
	for _, skill := range bundle.Skills {
		sourceHash, err := hashDir(skill.SourceDir)
		if err != nil {
			return nil, fmt.Errorf("hash source %s: %w", skill.ID, err)
		}
		for _, target := range targets {
			if !skillSupportsTarget(skill, target) {
				continue
			}
			root, err := resolveTargetRoot(target, destOverride)
			if err != nil {
				return nil, err
			}
			destDir := filepath.Join(root, skill.ID)
			present, err := pathExists(destDir)
			if err != nil {
				return nil, err
			}
			record := StatusRecord{
				SkillID: skill.ID,
				Target:  target,
				Path:    destDir,
				Present: present,
			}
			if present {
				destHash, err := hashDir(destDir)
				if err != nil {
					record.ErrorMsg = err.Error()
				} else {
					record.Matches = destHash == sourceHash
				}
			}
			records = append(records, record)
		}
	}
	return records, nil
}
