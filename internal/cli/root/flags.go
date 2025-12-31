package root

import (
	"fmt"
	"strings"
	"time"

	"github.com/urfave/cli/v3"

	"github.com/regenrek/peakypanes/internal/cli/spec"
)

func buildFlags(flags []spec.Flag) ([]cli.Flag, error) {
	if len(flags) == 0 {
		return nil, nil
	}
	out := make([]cli.Flag, 0, len(flags))
	for _, flag := range flags {
		built, err := buildFlag(flag)
		if err != nil {
			return nil, err
		}
		if built != nil {
			out = append(out, built)
		}
	}
	return out, nil
}

func buildFlag(flag spec.Flag) (cli.Flag, error) {
	name := strings.TrimSpace(flag.Name)
	if name == "" {
		return nil, fmt.Errorf("flag name is required")
	}
	var sources cli.ValueSourceChain
	if env := strings.TrimSpace(flag.Env); env != "" {
		sources = cli.EnvVars(env)
	}
	switch strings.TrimSpace(flag.Type) {
	case "bool":
		return &cli.BoolFlag{
			Name:     name,
			Aliases:  flag.Aliases,
			Usage:    flag.Description,
			Required: flag.Required,
			Hidden:   flag.Hidden,
			Sources:  sources,
			Value:    boolDefault(flag.Default),
		}, nil
	case "string", "path", "enum":
		fl := &cli.StringFlag{
			Name:     name,
			Aliases:  flag.Aliases,
			Usage:    flag.Description,
			Required: flag.Required,
			Hidden:   flag.Hidden,
			Sources:  sources,
			Value:    stringDefault(flag.Default),
		}
		if len(flag.Enum) > 0 {
			fl.Validator = enumValidator(flag.Enum)
		}
		return fl, nil
	case "int":
		return &cli.IntFlag{
			Name:     name,
			Aliases:  flag.Aliases,
			Usage:    flag.Description,
			Required: flag.Required,
			Hidden:   flag.Hidden,
			Sources:  sources,
			Value:    intDefault(flag.Default),
		}, nil
	case "float":
		return &cli.FloatFlag{
			Name:     name,
			Aliases:  flag.Aliases,
			Usage:    flag.Description,
			Required: flag.Required,
			Hidden:   flag.Hidden,
			Sources:  sources,
			Value:    floatDefault(flag.Default),
		}, nil
	case "duration":
		return &cli.DurationFlag{
			Name:     name,
			Aliases:  flag.Aliases,
			Usage:    flag.Description,
			Required: flag.Required,
			Hidden:   flag.Hidden,
			Sources:  sources,
			Value:    durationDefault(flag.Default),
		}, nil
	case "string_list":
		fl := &cli.StringSliceFlag{
			Name:     name,
			Aliases:  flag.Aliases,
			Usage:    flag.Description,
			Required: flag.Required,
			Hidden:   flag.Hidden,
			Sources:  sources,
			Value:    stringSliceDefault(flag.Default),
		}
		if len(flag.Enum) > 0 {
			fl.Validator = sliceEnumValidator(flag.Enum)
		}
		return fl, nil
	default:
		return nil, fmt.Errorf("unsupported flag type %q for %s", flag.Type, name)
	}
}

func enumValidator(values []string) func(string) error {
	allowed := make(map[string]struct{}, len(values))
	for _, value := range values {
		allowed[value] = struct{}{}
	}
	return func(val string) error {
		if _, ok := allowed[val]; !ok {
			return fmt.Errorf("invalid value %q (allowed: %s)", val, strings.Join(values, ", "))
		}
		return nil
	}
}

func sliceEnumValidator(values []string) func([]string) error {
	allowed := make(map[string]struct{}, len(values))
	for _, value := range values {
		allowed[value] = struct{}{}
	}
	return func(vals []string) error {
		for _, val := range vals {
			if _, ok := allowed[val]; !ok {
				return fmt.Errorf("invalid value %q (allowed: %s)", val, strings.Join(values, ", "))
			}
		}
		return nil
	}
}

func boolDefault(value any) bool {
	if parsed, ok := value.(bool); ok {
		return parsed
	}
	return false
}

func stringDefault(value any) string {
	if parsed, ok := value.(string); ok {
		return parsed
	}
	return ""
}

func intDefault(value any) int {
	switch parsed := value.(type) {
	case int:
		return parsed
	case int64:
		return int(parsed)
	case float64:
		return int(parsed)
	default:
		return 0
	}
}

func floatDefault(value any) float64 {
	switch parsed := value.(type) {
	case float64:
		return parsed
	case float32:
		return float64(parsed)
	case int:
		return float64(parsed)
	case int64:
		return float64(parsed)
	default:
		return 0
	}
}

func durationDefault(value any) time.Duration {
	switch parsed := value.(type) {
	case time.Duration:
		return parsed
	case string:
		d, err := time.ParseDuration(parsed)
		if err == nil {
			return d
		}
	}
	return 0
}

func stringSliceDefault(value any) []string {
	switch parsed := value.(type) {
	case []string:
		return parsed
	case []any:
		out := make([]string, 0, len(parsed))
		for _, v := range parsed {
			if s, ok := v.(string); ok {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}
