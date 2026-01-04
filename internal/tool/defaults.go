package tool

import (
	"regexp"
	"time"
)

// DefaultProfile returns the fallback input profile.
func DefaultProfile() Profile {
	return Profile{Submit: []byte{'\n'}}
}

func defaultDefinitions() []Definition {
	codexProfile := Profile{
		BracketedPaste: true,
		Submit:         []byte{'\r'},
		SubmitDelay:    30 * time.Millisecond,
	}
	claudeProfile := Profile{
		Submit:      []byte{'\r'},
		SubmitDelay: 30 * time.Millisecond,
	}
	return []Definition{
		{
			Name:         "codex",
			Aliases:      []string{"openai-codex"},
			CommandNames: []string{"codex"},
			TitleRegex:   []*regexp.Regexp{regexp.MustCompile(`(?i)\bcodex\b`)},
			Profile:      codexProfile,
		},
		{
			Name:         "claude",
			Aliases:      []string{"claude-code"},
			CommandNames: []string{"claude", "claude-code"},
			TitleRegex:   []*regexp.Regexp{regexp.MustCompile(`(?i)\bclaude\b`)},
			Profile:      claudeProfile,
		},
		{
			Name:         "lazygit",
			CommandNames: []string{"lazygit"},
			TitleRegex:   []*regexp.Regexp{regexp.MustCompile(`(?i)\blazygit\b`)},
		},
		{
			Name:         "gh-dash",
			CommandNames: []string{"gh-dash"},
			CommandRegex: []*regexp.Regexp{regexp.MustCompile(`(?i)\bgh\s+dash\b`)},
			TitleRegex:   []*regexp.Regexp{regexp.MustCompile(`(?i)\bgh[- ]dash\b`)},
		},
		{
			Name:         "k9s",
			CommandNames: []string{"k9s"},
			TitleRegex:   []*regexp.Regexp{regexp.MustCompile(`(?i)\bk9s\b`)},
		},
		{
			Name:         "btop",
			CommandNames: []string{"btop", "btop++"},
			TitleRegex:   []*regexp.Regexp{regexp.MustCompile(`(?i)\bbtop\b`)},
		},
		{
			Name:         "htop",
			CommandNames: []string{"htop"},
			TitleRegex:   []*regexp.Regexp{regexp.MustCompile(`(?i)\bhtop\b`)},
		},
		{
			Name:         "nvtop",
			CommandNames: []string{"nvtop"},
			TitleRegex:   []*regexp.Regexp{regexp.MustCompile(`(?i)\bnvtop\b`)},
		},
		{
			Name:         "ncdu",
			CommandNames: []string{"ncdu"},
			TitleRegex:   []*regexp.Regexp{regexp.MustCompile(`(?i)\bncdu\b`)},
		},
		{
			Name:         "tig",
			CommandNames: []string{"tig"},
			TitleRegex:   []*regexp.Regexp{regexp.MustCompile(`(?i)\btig\b`)},
		},
		{
			Name:         "gitui",
			CommandNames: []string{"gitui"},
			TitleRegex:   []*regexp.Regexp{regexp.MustCompile(`(?i)\bgitui\b`)},
		},
		{
			Name:         "ranger",
			CommandNames: []string{"ranger"},
			TitleRegex:   []*regexp.Regexp{regexp.MustCompile(`(?i)\branger\b`)},
		},
		{
			Name:         "yazi",
			CommandNames: []string{"yazi"},
			TitleRegex:   []*regexp.Regexp{regexp.MustCompile(`(?i)\byazi\b`)},
		},
		{
			Name:         "nnn",
			CommandNames: []string{"nnn"},
			TitleRegex:   []*regexp.Regexp{regexp.MustCompile(`(?i)\bnnn\b`)},
		},
		{
			Name:         "lf",
			CommandNames: []string{"lf"},
			TitleRegex:   []*regexp.Regexp{regexp.MustCompile(`(?i)\blf\b`)},
		},
	}
}
