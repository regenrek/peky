package tool

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/regenrek/peakypanes/internal/layout"
)

// RegistryFromConfig builds a registry using layout tool detection settings.
func RegistryFromConfig(cfg layout.ToolDetectionConfig) (*Registry, error) {
	enabled := true
	if cfg.Enabled != nil {
		enabled = *cfg.Enabled
	}
	defs := defaultDefinitions()
	defsByName := make(map[string]Definition, len(defs))
	order := make([]string, 0, len(defs))
	for _, def := range defs {
		name := NormalizeName(def.Name)
		defsByName[name] = def
		order = append(order, name)
	}
	for _, custom := range cfg.Tools {
		name := NormalizeName(custom.Name)
		if name == "" {
			return nil, fmt.Errorf("tool_detection.tools: name is required")
		}
		base, ok := defsByName[name]
		if !ok {
			base = Definition{Name: name, Profile: DefaultProfile()}
			order = append(order, name)
		}
		merged, err := mergeToolDefinition(base, custom)
		if err != nil {
			return nil, err
		}
		defsByName[name] = merged
	}
	for name, input := range cfg.Profiles {
		canonical := NormalizeName(name)
		if canonical == "" {
			continue
		}
		base, ok := defsByName[canonical]
		if !ok {
			base = Definition{Name: canonical, Profile: DefaultProfile()}
			order = append(order, canonical)
		}
		base.Profile = applyInputConfig(base.Profile, input)
		defsByName[canonical] = base
	}
	resolved := make([]Definition, 0, len(defsByName))
	for _, name := range order {
		if def, ok := defsByName[name]; ok {
			resolved = append(resolved, def)
		}
	}
	return NewRegistry(resolved, RegistryOptions{
		Enabled:        enabled,
		Allow:          cfg.Allow,
		DefaultProfile: DefaultProfile(),
	})
}

func mergeToolDefinition(base Definition, cfg layout.ToolDefinitionConfig) (Definition, error) {
	if cfg.Name != "" {
		base.Name = NormalizeName(cfg.Name)
	}
	if len(cfg.Aliases) > 0 {
		base.Aliases = append(base.Aliases, cfg.Aliases...)
	}
	if len(cfg.CommandNames) > 0 {
		base.CommandNames = append(base.CommandNames, cfg.CommandNames...)
	}
	if len(cfg.CommandRegex) > 0 {
		compiled, err := compileRegexList(cfg.CommandRegex)
		if err != nil {
			return Definition{}, fmt.Errorf("tool_detection.tools[%s].command_regex: %w", cfg.Name, err)
		}
		base.CommandRegex = append(base.CommandRegex, compiled...)
	}
	if len(cfg.TitleRegex) > 0 {
		compiled, err := compileRegexList(cfg.TitleRegex)
		if err != nil {
			return Definition{}, fmt.Errorf("tool_detection.tools[%s].title_regex: %w", cfg.Name, err)
		}
		base.TitleRegex = append(base.TitleRegex, compiled...)
	}
	base.Profile = applyInputConfig(base.Profile, cfg.Input)
	return base, nil
}

func compileRegexList(values []string) ([]*regexp.Regexp, error) {
	out := make([]*regexp.Regexp, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		re, err := regexp.Compile(value)
		if err != nil {
			return nil, err
		}
		out = append(out, re)
	}
	return out, nil
}

func applyInputConfig(profile Profile, cfg layout.ToolInputConfig) Profile {
	if cfg.BracketedPaste != nil {
		profile.BracketedPaste = *cfg.BracketedPaste
	}
	if cfg.Submit != nil {
		profile.Submit = []byte(*cfg.Submit)
	}
	if cfg.SubmitDelayMS != nil {
		profile.SubmitDelay = time.Duration(*cfg.SubmitDelayMS) * time.Millisecond
		if profile.SubmitDelay < 0 {
			profile.SubmitDelay = 0
		}
	}
	if cfg.CombineSubmit != nil {
		profile.CombineSubmit = *cfg.CombineSubmit
	}
	return profile
}
