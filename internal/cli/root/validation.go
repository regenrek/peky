package root

import (
	"fmt"
	"strings"

	"github.com/urfave/cli/v3"

	"github.com/regenrek/peakypanes/internal/cli/spec"
)

func validateArgs(cmdSpec spec.Command, cmd *cli.Command) error {
	if len(cmdSpec.Args) == 0 {
		return nil
	}
	for _, argSpec := range cmdSpec.Args {
		if !argSpec.Required {
			continue
		}
		name := strings.TrimSpace(argSpec.Name)
		if name == "" {
			continue
		}
		if argSpec.Variadic {
			values := cmd.StringArgs(name)
			if len(values) == 0 {
				return fmt.Errorf("missing argument %q", argSpec.Name)
			}
			continue
		}
		if strings.TrimSpace(cmd.StringArg(name)) == "" {
			return fmt.Errorf("missing argument %q", argSpec.Name)
		}
	}
	return nil
}

func validateConstraints(cmdSpec spec.Command, cmd *cli.Command) error {
	for _, constraint := range cmdSpec.Constraints {
		if err := validateConstraint(constraint, cmdSpec, cmd); err != nil {
			return err
		}
	}
	return nil
}

func validateConstraint(constraint spec.Constraint, cmdSpec spec.Command, cmd *cli.Command) error {
	fields := constraint.Fields
	if len(fields) == 0 {
		return nil
	}
	presentCount := 0
	present := make(map[string]bool, len(fields))
	for _, field := range fields {
		ok := fieldPresent(field, cmdSpec, cmd)
		present[field] = ok
		if ok {
			presentCount++
		}
	}
	switch strings.TrimSpace(constraint.Type) {
	case "exactly_one":
		if presentCount != 1 {
			return fmt.Errorf("exactly one of %s is required", strings.Join(fields, ", "))
		}
	case "at_least_one":
		if presentCount == 0 {
			return fmt.Errorf("at least one of %s is required", strings.Join(fields, ", "))
		}
	case "requires":
		if present[fields[0]] && !allPresent(fields[1:], present) {
			return fmt.Errorf("%s requires %s", fields[0], strings.Join(fields[1:], ", "))
		}
	case "excludes":
		if presentCount > 1 {
			return fmt.Errorf("only one of %s may be set", strings.Join(fields, ", "))
		}
	}
	return nil
}

func allPresent(fields []string, present map[string]bool) bool {
	for _, field := range fields {
		if !present[field] {
			return false
		}
	}
	return true
}

func fieldPresent(field string, cmdSpec spec.Command, cmd *cli.Command) bool {
	field = strings.TrimSpace(field)
	if field == "" {
		return false
	}
	for _, arg := range cmdSpec.Args {
		if arg.Name == field {
			if arg.Variadic {
				values := cmd.StringArgs(field)
				for _, value := range values {
					if strings.TrimSpace(value) != "" {
						return true
					}
				}
				return false
			}
			return strings.TrimSpace(cmd.StringArg(field)) != ""
		}
	}
	return cmd.IsSet(field)
}
