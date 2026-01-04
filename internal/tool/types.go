package tool

import (
	"regexp"
	"time"
)

// Profile describes how input should be sent to a tool.
type Profile struct {
	BracketedPaste bool
	Submit         []byte
	SubmitDelay    time.Duration
	CombineSubmit  bool
}

// Definition describes a detectable tool and its input profile.
type Definition struct {
	Name         string
	Aliases      []string
	CommandNames []string
	CommandRegex []*regexp.Regexp
	TitleRegex   []*regexp.Regexp
	Profile      Profile
}

// PaneInfo holds pane metadata used for tool detection.
type PaneInfo struct {
	Tool         string
	Command      string
	StartCommand string
	Title        string
}

// Registry stores tool definitions and detection policy.
type Registry struct {
	enabled        bool
	defs           map[string]Definition
	aliases        map[string]string
	allow          map[string]bool
	order          []string
	defaultProfile Profile
}

// RegistryOptions configure registry behavior.
type RegistryOptions struct {
	Enabled        bool
	Allow          map[string]bool
	DefaultProfile Profile
}
