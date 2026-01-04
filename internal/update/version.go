package update

import (
	"regexp"
	"strings"

	"github.com/Masterminds/semver/v3"
)

var goInstallRegexp = regexp.MustCompile(`^v?\d+\.\d+\.\d+-\d+\.\d{14}-[0-9a-f]{12}$`)

// NormalizeVersion trims whitespace and a leading "v".
func NormalizeVersion(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	return strings.TrimPrefix(trimmed, "v")
}

// IsDevelopmentVersion reports whether the version should suppress update prompts.
func IsDevelopmentVersion(raw string) bool {
	value := strings.TrimSpace(raw)
	if value == "" {
		return true
	}
	lower := strings.ToLower(value)
	switch lower {
	case "dev", "devel", "unknown":
		return true
	}
	if strings.Contains(lower, "dirty") {
		return true
	}
	return goInstallRegexp.MatchString(value)
}

func parseSemver(raw string) (*semver.Version, error) {
	normalized := NormalizeVersion(raw)
	if normalized == "" {
		return nil, semver.ErrInvalidSemVer
	}
	return semver.NewVersion(normalized)
}

// CompareVersions compares two semantic versions.
func CompareVersions(current, latest string) (int, error) {
	cur, err := parseSemver(current)
	if err != nil {
		return 0, err
	}
	lat, err := parseSemver(latest)
	if err != nil {
		return 0, err
	}
	return cur.Compare(lat), nil
}
