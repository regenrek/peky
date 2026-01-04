package tool

import "fmt"

// DefaultRegistry returns a registry with built-in tool definitions.
func DefaultRegistry() (*Registry, error) {
	reg, err := NewRegistry(defaultDefinitions(), RegistryOptions{Enabled: true, DefaultProfile: DefaultProfile()})
	if err != nil {
		return nil, fmt.Errorf("tool: default registry init failed: %w", err)
	}
	return reg, nil
}

// NewRegistry builds a registry from tool definitions.
func NewRegistry(defs []Definition, opts RegistryOptions) (*Registry, error) {
	defaultProfile := opts.DefaultProfile
	if defaultProfile.Submit == nil {
		defaultProfile = DefaultProfile()
	}
	reg := &Registry{
		enabled:        opts.Enabled,
		defs:           make(map[string]Definition, len(defs)),
		aliases:        make(map[string]string, len(defs)*2),
		allow:          normalizeAllow(opts.Allow),
		defaultProfile: defaultProfile,
	}
	if len(defs) == 0 {
		return reg, nil
	}
	for _, def := range defs {
		normalized, err := normalizeDefinition(def)
		if err != nil {
			return nil, err
		}
		reg.defs[normalized.Name] = normalized
		reg.order = append(reg.order, normalized.Name)
		reg.registerAliases(normalized.Name, normalized.Aliases)
		reg.registerAliases(normalized.Name, normalized.CommandNames)
		reg.registerAliases(normalized.Name, []string{normalized.Name})
	}
	return reg, nil
}

func normalizeDefinition(def Definition) (Definition, error) {
	name := NormalizeName(def.Name)
	if name == "" {
		return Definition{}, fmt.Errorf("tool: definition name is required")
	}
	def.Name = name
	def.Aliases = normalizeList(def.Aliases)
	def.CommandNames = normalizeList(def.CommandNames)
	if def.Profile.Submit == nil {
		def.Profile.Submit = append([]byte(nil), DefaultProfile().Submit...)
	}
	return def, nil
}

func normalizeAllow(allow map[string]bool) map[string]bool {
	if len(allow) == 0 {
		return nil
	}
	out := make(map[string]bool, len(allow))
	for key, value := range allow {
		key = NormalizeName(key)
		if key == "" {
			continue
		}
		out[key] = value
	}
	return out
}

func (r *Registry) registerAliases(name string, values []string) {
	if r == nil || name == "" {
		return
	}
	for _, value := range values {
		value = NormalizeName(value)
		if value == "" {
			continue
		}
		r.aliases[value] = name
	}
}

// Normalize returns the canonical tool name when known.
func (r *Registry) Normalize(value string) string {
	value = NormalizeName(value)
	if value == "" {
		return ""
	}
	if r == nil {
		return value
	}
	if canonical, ok := r.aliases[value]; ok {
		return canonical
	}
	return ""
}

// Allowed reports whether a tool is enabled by policy.
func (r *Registry) Allowed(name string) bool {
	name = NormalizeName(name)
	if name == "" {
		return false
	}
	if r == nil {
		return true
	}
	if r.allow == nil {
		return true
	}
	allowed, ok := r.allow[name]
	if !ok {
		return true
	}
	return allowed
}

// Enabled reports whether automatic detection is enabled.
func (r *Registry) Enabled() bool {
	if r == nil {
		return false
	}
	return r.enabled
}

// Profile returns the input profile for a tool or the default profile.
func (r *Registry) Profile(name string) Profile {
	if r == nil {
		return DefaultProfile()
	}
	name = r.Normalize(name)
	if name == "" || !r.Allowed(name) {
		return r.defaultProfile
	}
	def, ok := r.defs[name]
	if !ok {
		return r.defaultProfile
	}
	return def.Profile
}
