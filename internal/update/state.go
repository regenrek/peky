package update

import "time"

// State captures persisted update metadata.
type State struct {
	CurrentVersion   string  `json:"current_version"`
	LatestVersion    string  `json:"latest_version,omitempty"`
	SkippedVersion   string  `json:"skipped_version,omitempty"`
	LastPromptUnixMs int64   `json:"last_prompt_unix_ms,omitempty"`
	LastCheckUnixMs  int64   `json:"last_check_unix_ms,omitempty"`
	Channel          Channel `json:"channel,omitempty"`
}

// UpdateAvailable reports whether a newer version exists.
func (s State) UpdateAvailable() bool {
	if IsDevelopmentVersion(s.CurrentVersion) {
		return false
	}
	cmp, err := CompareVersions(s.CurrentVersion, s.LatestVersion)
	if err != nil {
		return false
	}
	return cmp < 0
}

// IsSkipped reports whether the latest version is skipped.
func (s State) IsSkipped() bool {
	if s.SkippedVersion == "" || s.LatestVersion == "" {
		return false
	}
	return NormalizeVersion(s.SkippedVersion) == NormalizeVersion(s.LatestVersion)
}

func (s *State) MarkPrompted(now time.Time) {
	if s == nil {
		return
	}
	s.LastPromptUnixMs = now.UnixMilli()
}

func (s *State) MarkChecked(now time.Time) {
	if s == nil {
		return
	}
	s.LastCheckUnixMs = now.UnixMilli()
}
