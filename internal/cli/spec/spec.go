package spec

import (
	"bytes"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"gopkg.in/yaml.v3"
)

//go:embed commands.yaml commands.schema.json
var embeddedFS embed.FS

// Spec is the canonical CLI command specification.
type Spec struct {
	Version        int             `yaml:"version"`
	App            AppSpec         `yaml:"app"`
	GlobalFlags    []Flag          `yaml:"global_flags"`
	SlashShortcuts []SlashShortcut `yaml:"slash_shortcuts"`
	Commands       []Command       `yaml:"commands"`
}

// AppSpec configures the top-level CLI app.
type AppSpec struct {
	Name                 string `yaml:"name"`
	Summary              string `yaml:"summary"`
	DefaultCommand       string `yaml:"default_command"`
	AllowLayoutShorthand bool   `yaml:"allow_layout_shorthand"`
}

// Flag describes a CLI flag.
type Flag struct {
	Name        string   `yaml:"name"`
	Aliases     []string `yaml:"aliases"`
	Type        string   `yaml:"type"`
	Required    bool     `yaml:"required"`
	Default     any      `yaml:"default"`
	Enum        []string `yaml:"enum"`
	Repeatable  bool     `yaml:"repeatable"`
	Description string   `yaml:"description"`
	Env         string   `yaml:"env"`
	Hidden      bool     `yaml:"hidden"`
}

// Arg describes a positional argument.
type Arg struct {
	Name        string   `yaml:"name"`
	Type        string   `yaml:"type"`
	Required    bool     `yaml:"required"`
	Variadic    bool     `yaml:"variadic"`
	Enum        []string `yaml:"enum"`
	Description string   `yaml:"description"`
}

// Constraint describes argument/flag validation rules.
type Constraint struct {
	Type   string   `yaml:"type"`
	Fields []string `yaml:"fields"`
}

// JSONSpec declares JSON output capability.
type JSONSpec struct {
	Supported bool   `yaml:"supported"`
	SchemaRef string `yaml:"schema_ref"`
	Stream    bool   `yaml:"stream"`
}

// SlashSpec controls slash command exposure.
type SlashSpec struct {
	Enabled bool     `yaml:"enabled"`
	Aliases []string `yaml:"aliases"`
}

// SlashShortcut maps a short slash command to a CLI command.
type SlashShortcut struct {
	Name    string         `yaml:"name"`
	Summary string         `yaml:"summary"`
	Command string         `yaml:"command"`
	Flags   map[string]any `yaml:"flags"`
}

// Command describes a CLI command and its subcommands.
type Command struct {
	Name        string       `yaml:"name"`
	ID          string       `yaml:"id"`
	Summary     string       `yaml:"summary"`
	Description string       `yaml:"description"`
	Aliases     []string     `yaml:"aliases"`
	Flags       []Flag       `yaml:"flags"`
	Args        []Arg        `yaml:"args"`
	Constraints []Constraint `yaml:"constraints"`
	SideEffects bool         `yaml:"side_effects"`
	Confirm     bool         `yaml:"confirm"`
	JSON        *JSONSpec    `yaml:"json"`
	Slash       *SlashSpec   `yaml:"slash"`
	Hidden      bool         `yaml:"hidden"`
	Subcommands []Command    `yaml:"subcommands"`
}

// LoadDefault loads the embedded spec and validates it.
func LoadDefault() (*Spec, error) {
	data, err := embeddedFS.ReadFile("commands.yaml")
	if err != nil {
		return nil, fmt.Errorf("read embedded spec: %w", err)
	}
	return Parse(data)
}

// Parse loads a spec from YAML bytes and validates it against the embedded schema.
func Parse(data []byte) (*Spec, error) {
	if len(bytes.TrimSpace(data)) == 0 {
		return nil, errors.New("spec is empty")
	}
	spec := &Spec{}
	if err := yaml.Unmarshal(data, spec); err != nil {
		return nil, fmt.Errorf("parse spec yaml: %w", err)
	}
	if err := Validate(data); err != nil {
		return nil, err
	}
	return spec, nil
}

// Validate checks the YAML spec against the embedded JSON schema.
func Validate(data []byte) error {
	if len(bytes.TrimSpace(data)) == 0 {
		return errors.New("spec is empty")
	}
	schemaBytes, err := embeddedFS.ReadFile("commands.schema.json")
	if err != nil {
		return fmt.Errorf("read embedded schema: %w", err)
	}
	var schemaDoc any
	if err := json.Unmarshal(schemaBytes, &schemaDoc); err != nil {
		return fmt.Errorf("parse schema json: %w", err)
	}
	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource("commands.schema.json", schemaDoc); err != nil {
		return fmt.Errorf("load schema: %w", err)
	}
	schema, err := compiler.Compile("commands.schema.json")
	if err != nil {
		return fmt.Errorf("compile schema: %w", err)
	}
	payload, err := yamlToJSON(data)
	if err != nil {
		return fmt.Errorf("serialize spec: %w", err)
	}
	var payloadDoc any
	if err := json.Unmarshal(payload, &payloadDoc); err != nil {
		return fmt.Errorf("parse spec json: %w", err)
	}
	if err := schema.Validate(payloadDoc); err != nil {
		return fmt.Errorf("spec schema validation: %w", err)
	}
	return nil
}

func yamlToJSON(data []byte) ([]byte, error) {
	var raw any
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse yaml: %w", err)
	}
	normalized, err := normalizeYAML(raw)
	if err != nil {
		return nil, err
	}
	payload, err := json.Marshal(normalized)
	if err != nil {
		return nil, fmt.Errorf("marshal json: %w", err)
	}
	return payload, nil
}

func normalizeYAML(value any) (any, error) {
	switch typed := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(typed))
		for key, val := range typed {
			normalized, err := normalizeYAML(val)
			if err != nil {
				return nil, err
			}
			out[key] = normalized
		}
		return out, nil
	case map[any]any:
		out := make(map[string]any, len(typed))
		for key, val := range typed {
			strKey, ok := key.(string)
			if !ok {
				return nil, fmt.Errorf("invalid yaml map key: %T", key)
			}
			normalized, err := normalizeYAML(val)
			if err != nil {
				return nil, err
			}
			out[strKey] = normalized
		}
		return out, nil
	case []any:
		out := make([]any, len(typed))
		for i, val := range typed {
			normalized, err := normalizeYAML(val)
			if err != nil {
				return nil, err
			}
			out[i] = normalized
		}
		return out, nil
	default:
		return value, nil
	}
}

// AllCommands returns a flat list of commands including subcommands.
func (s *Spec) AllCommands() []Command {
	if s == nil {
		return nil
	}
	var out []Command
	for _, cmd := range s.Commands {
		appendCommands(&out, cmd)
	}
	return out
}

func appendCommands(out *[]Command, cmd Command) {
	*out = append(*out, cmd)
	for _, sub := range cmd.Subcommands {
		appendCommands(out, sub)
	}
}

// FindByID returns the command with the matching ID.
func (s *Spec) FindByID(id string) *Command {
	id = strings.TrimSpace(id)
	if id == "" || s == nil {
		return nil
	}
	for _, cmd := range s.AllCommands() {
		if cmd.ID == id {
			copy := cmd
			return &copy
		}
	}
	return nil
}

// SlashCommands returns commands eligible for slash input.
func (s *Spec) SlashCommands() []Command {
	if s == nil {
		return nil
	}
	var out []Command
	for _, cmd := range s.AllCommands() {
		slash := cmd.Slash
		if slash != nil && !slash.Enabled {
			continue
		}
		out = append(out, cmd)
	}
	return out
}
