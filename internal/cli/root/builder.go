package root

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/urfave/cli/v3"

	"github.com/regenrek/peakypanes/internal/cli/output"
	"github.com/regenrek/peakypanes/internal/cli/spec"
)

// BuildApp constructs a CLI app from the spec and registry.
func BuildApp(specDoc *spec.Spec, deps Dependencies, reg *Registry) (*cli.Command, error) {
	if specDoc == nil {
		return nil, fmt.Errorf("spec is nil")
	}
	if reg == nil {
		return nil, fmt.Errorf("registry is nil")
	}
	if err := reg.EnsureHandlers(specDoc); err != nil {
		return nil, err
	}
	app := &cli.Command{
		Name:        specDoc.App.Name,
		Usage:       specDoc.App.Summary,
		Description: specDoc.App.Summary,
		Commands:    []*cli.Command{},
		Writer:      deps.Stdout,
		ErrWriter:   deps.Stderr,
	}
	var runEnvCleanup func()
	app.Before = func(ctx context.Context, cmd *cli.Command) (context.Context, error) {
		if cmd != nil && cmd.Bool("version") {
			out := deps.Stdout
			if out == nil {
				out = io.Discard
			}
			_, _ = fmt.Fprintf(out, "%s %s\n", specDoc.App.Name, deps.Version)
			return ctx, cli.Exit("", 0)
		}
		cleanup, err := applyRunEnvFromFlags(cmd, deps.Version)
		if err != nil {
			return ctx, err
		}
		runEnvCleanup = cleanup
		return ctx, nil
	}
	app.After = func(ctx context.Context, cmd *cli.Command) error {
		if runEnvCleanup != nil {
			runEnvCleanup()
			runEnvCleanup = nil
		}
		return nil
	}
	globalFlags, err := buildFlags(specDoc.GlobalFlags)
	if err != nil {
		return nil, err
	}
	app.Flags = globalFlags
	for _, cmdSpec := range specDoc.Commands {
		cmd, err := buildCommand(cmdSpec, deps, reg)
		if err != nil {
			return nil, err
		}
		app.Commands = append(app.Commands, cmd)
	}
	app.Action = func(ctx context.Context, c *cli.Command) error {
		return runDefaultCommand(specDoc, deps, reg, ctx)
	}
	return app, nil
}

func buildCommand(cmdSpec spec.Command, deps Dependencies, reg *Registry) (*cli.Command, error) {
	cmd := &cli.Command{
		Name:        cmdSpec.Name,
		Aliases:     cmdSpec.Aliases,
		Usage:       cmdSpec.Summary,
		Description: cmdSpec.Description,
		Hidden:      cmdSpec.Hidden,
	}
	flags, err := buildFlags(cmdSpec.Flags)
	if err != nil {
		return nil, fmt.Errorf("flags for %s: %w", cmdSpec.ID, err)
	}
	cmd.Flags = flags
	cmd.ArgsUsage = argsUsage(cmdSpec.Args)
	cmd.Arguments = buildArguments(cmdSpec.Args)
	for _, child := range cmdSpec.Subcommands {
		sub, err := buildCommand(child, deps, reg)
		if err != nil {
			return nil, err
		}
		cmd.Commands = append(cmd.Commands, sub)
	}
	if handler, ok := reg.HandlerFor(cmdSpec.ID); ok {
		cmd.Action = func(ctx context.Context, cliCmd *cli.Command) error {
			return runHandler(ctx, cliCmd, cmdSpec, deps, handler)
		}
	}
	return cmd, nil
}

func runDefaultCommand(specDoc *spec.Spec, deps Dependencies, reg *Registry, ctx context.Context) error {
	if specDoc == nil {
		return fmt.Errorf("spec is nil")
	}
	defaultCmd := strings.TrimSpace(specDoc.App.DefaultCommand)
	if defaultCmd == "" {
		return nil
	}
	cmdSpec := specDoc.FindByID(defaultCmd)
	if cmdSpec == nil {
		return fmt.Errorf("default command %q not found", defaultCmd)
	}
	handler, ok := reg.HandlerFor(cmdSpec.ID)
	if !ok {
		return fmt.Errorf("default command handler missing: %s", cmdSpec.ID)
	}
	cliCmd := &cli.Command{Name: cmdSpec.Name}
	return runHandler(ctx, cliCmd, *cmdSpec, deps, handler)
}

func runHandler(ctx context.Context, cliCmd *cli.Command, cmdSpec spec.Command, deps Dependencies, handler Handler) error {
	if handler == nil {
		return nil
	}
	args := []string{}
	if cliCmd != nil {
		if parsed := cliCmd.Args(); parsed != nil {
			args = parsed.Slice()
		}
	}
	commandCtx := CommandContext{
		Context: ctx,
		Args:    args,
		Spec:    cmdSpec,
		Cmd:     cliCmd,
		Deps:    deps,
		JSON:    cliCmd.Bool("json"),
		Out:     deps.Stdout,
		ErrOut:  deps.Stderr,
		Stdin:   deps.Stdin,
	}
	if err := validateArgs(cmdSpec, cliCmd); err != nil {
		return err
	}
	if err := validateConstraints(cmdSpec, cliCmd); err != nil {
		return err
	}
	if commandCtx.JSON && (cmdSpec.JSON == nil || !cmdSpec.JSON.Supported) {
		return fmt.Errorf("command %s does not support --json", cmdSpec.Name)
	}
	if err := confirmIfNeeded(commandCtx, cliCmd); err != nil {
		return err
	}
	start := time.Now()
	if err := handler(commandCtx); err != nil {
		if !commandCtx.JSON {
			return err
		}
		meta := output.WithDuration(output.NewMeta(cmdSpec.ID, deps.Version), start)
		_ = output.WriteError(commandCtx.Out, meta, "command_failed", err.Error(), nil)
		return cli.Exit("", 1)
	}
	return nil
}

func argsUsage(args []spec.Arg) string {
	if len(args) == 0 {
		return ""
	}
	parts := make([]string, 0, len(args))
	for _, arg := range args {
		name := strings.ToUpper(arg.Name)
		if arg.Variadic {
			name += "..."
		}
		if arg.Required {
			parts = append(parts, name)
		} else {
			parts = append(parts, fmt.Sprintf("[%s]", name))
		}
	}
	return strings.Join(parts, " ")
}

func confirmIfNeeded(ctx CommandContext, cliCmd *cli.Command) error {
	if !ctx.Spec.SideEffects && !ctx.Spec.Confirm {
		return nil
	}
	if cliCmd.Bool("yes") {
		return nil
	}
	message := fmt.Sprintf("Confirm %s", ctx.Spec.ID)
	ok, err := PromptConfirm(ctx.Stdin, ctx.ErrOut, message)
	if err != nil {
		return err
	}
	if !ok {
		return cli.Exit("", 1)
	}
	return nil
}
