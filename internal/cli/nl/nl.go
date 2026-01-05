package nl

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/kballard/go-shellquote"

	"github.com/regenrek/peakypanes/internal/cli/catalog"
	"github.com/regenrek/peakypanes/internal/cli/output"
	"github.com/regenrek/peakypanes/internal/cli/root"
	"github.com/regenrek/peakypanes/internal/cli/spec"
	"github.com/regenrek/peakypanes/internal/identity"
)

// Register registers natural language handlers.
func Register(reg *root.Registry) {
	reg.Register("nl.plan", runPlan)
	reg.Register("nl.run", runRun)
}

func runPlan(ctx root.CommandContext) error {
	start := time.Now()
	meta := output.NewMeta("nl.plan", ctx.Deps.Version)
	prompt, err := readPrompt(ctx)
	if err != nil {
		return err
	}
	plan, err := buildPlan(prompt)
	if err != nil {
		return err
	}
	if ctx.JSON {
		meta = output.WithDuration(meta, start)
		return output.WriteSuccess(ctx.Out, meta, plan)
	}
	if _, err := fmt.Fprintf(ctx.Out, "Plan %s: %s\n", plan.PlanID, plan.Rationale); err != nil {
		return err
	}
	for _, cmd := range plan.Commands {
		if _, err := fmt.Fprintf(ctx.Out, "- %s %s\n", cmd.ID, cmd.Command); err != nil {
			return err
		}
	}
	return nil
}

func runRun(ctx root.CommandContext) error {
	start := time.Now()
	meta := output.NewMeta("nl.run", ctx.Deps.Version)
	prompt, err := readPrompt(ctx)
	if err != nil {
		return err
	}
	plan, err := buildPlan(prompt)
	if err != nil {
		return err
	}
	if len(plan.Commands) == 0 {
		return fmt.Errorf("no commands planned")
	}
	if hasNLCommand(plan.Commands) {
		return fmt.Errorf("nl commands cannot be executed via nl.run")
	}
	if ctx.Cmd.Bool("confirm") || len(plan.RequiresConfirmations) > 0 {
		ok, err := root.PromptConfirm(ctx.Stdin, ctx.ErrOut, "Execute planned commands?")
		if err != nil {
			return err
		}
		if !ok {
			return nil
		}
	}
	results, err := executePlan(ctx, plan)
	if err != nil {
		return err
	}
	if ctx.JSON {
		meta = output.WithDuration(meta, start)
		return output.WriteSuccess(ctx.Out, meta, results)
	}
	if _, err := fmt.Fprintf(ctx.Out, "Execution %s completed\n", results.ExecutionID); err != nil {
		return err
	}
	return nil
}

func readPrompt(ctx root.CommandContext) (string, error) {
	if ctx.Cmd.Bool("stdin") {
		data, err := io.ReadAll(ctx.Stdin)
		if err != nil {
			return "", err
		}
		prompt := strings.TrimSpace(string(data))
		if prompt == "" {
			return "", fmt.Errorf("prompt is required")
		}
		return prompt, nil
	}
	args := ctx.Args
	if len(args) == 0 {
		return "", fmt.Errorf("prompt is required")
	}
	prompt := strings.TrimSpace(strings.Join(args, " "))
	if prompt == "" {
		return "", fmt.Errorf("prompt is required")
	}
	return prompt, nil
}

func buildPlan(prompt string) (output.NLPlan, error) {
	specDoc, err := spec.LoadDefault()
	if err != nil {
		return output.NLPlan{}, err
	}
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		return output.NLPlan{}, fmt.Errorf("prompt is required")
	}
	planID := fmt.Sprintf("plan-%d", time.Now().UnixNano())
	if strings.HasPrefix(prompt, "/") {
		cmd, rationale, err := planFromSlash(specDoc, prompt)
		if err == nil {
			return finalizePlan(planID, rationale, []output.NLPlannedCommand{cmd}), nil
		}
	}
	if tokens, ok := shellTokens(prompt); ok {
		if cmd, rationale, err := planFromTokens(specDoc, tokens); err == nil {
			return finalizePlan(planID, rationale, []output.NLPlannedCommand{cmd}), nil
		}
	}
	if cmd, rationale, err := planFromRules(specDoc, prompt); err == nil {
		return finalizePlan(planID, rationale, []output.NLPlannedCommand{cmd}), nil
	}
	return output.NLPlan{}, fmt.Errorf("unable to map prompt to a command")
}

func finalizePlan(planID, rationale string, cmds []output.NLPlannedCommand) output.NLPlan {
	requires := make([]string, 0)
	for _, cmd := range cmds {
		if cmd.RequiresConfirm || cmd.SideEffects {
			requires = append(requires, cmd.ID)
		}
	}
	return output.NLPlan{
		PlanID:                planID,
		Rationale:             rationale,
		Commands:              cmds,
		RequiresConfirmations: requires,
	}
}

func planFromTokens(specDoc *spec.Spec, tokens []string) (output.NLPlannedCommand, string, error) {
	cmdSpec, consumed := matchCommand(specDoc.Commands, tokens)
	if cmdSpec == nil {
		return output.NLPlannedCommand{}, "", fmt.Errorf("unknown command")
	}
	flags, args := parseFlags(*cmdSpec, tokens[consumed:])
	cmd := buildPlannedCommand(*cmdSpec, tokens[:consumed], args, flags)
	return cmd, "Parsed CLI command", nil
}

func planFromSlash(specDoc *spec.Spec, prompt string) (output.NLPlannedCommand, string, error) {
	prompt = strings.TrimPrefix(strings.TrimSpace(prompt), "/")
	if prompt == "" {
		return output.NLPlannedCommand{}, "", fmt.Errorf("empty slash command")
	}
	parts, err := shellquote.Split(prompt)
	if err != nil {
		return output.NLPlannedCommand{}, "", err
	}
	if len(parts) == 0 {
		return output.NLPlannedCommand{}, "", fmt.Errorf("empty slash command")
	}
	name := strings.ToLower(parts[0])
	for _, shortcut := range specDoc.SlashShortcuts {
		if strings.ToLower(shortcut.Name) != name {
			continue
		}
		cmdTokens := strings.Fields(shortcut.Command)
		cmdSpec, consumed := matchCommand(specDoc.Commands, cmdTokens)
		if cmdSpec == nil || consumed != len(cmdTokens) {
			return output.NLPlannedCommand{}, "", fmt.Errorf("invalid shortcut command")
		}
		flags := map[string]any{}
		for key, val := range shortcut.Flags {
			flags[key] = val
		}
		args := []string{}
		if len(parts) > 1 {
			switch cmdSpec.ID {
			case "pane.send":
				flags["text"] = strings.Join(parts[1:], " ")
			case "pane.run":
				flags["command"] = strings.Join(parts[1:], " ")
			default:
				args = append(args, parts[1:]...)
			}
		}
		cmd := buildPlannedCommand(*cmdSpec, cmdTokens, args, flags)
		return cmd, "Matched slash shortcut", nil
	}
	return output.NLPlannedCommand{}, "", fmt.Errorf("unknown slash command")
}

func planFromRules(specDoc *spec.Spec, prompt string) (output.NLPlannedCommand, string, error) {
	lower := strings.ToLower(prompt)
	if strings.Contains(lower, "list sessions") {
		return buildSimple(specDoc, "session.list", []string{"session", "list"}, nil, nil), "Matched rule: list sessions", nil
	}
	if strings.Contains(lower, "list panes") {
		return buildSimple(specDoc, "pane.list", []string{"pane", "list"}, nil, nil), "Matched rule: list panes", nil
	}
	if strings.Contains(lower, "open dashboard") || strings.Contains(lower, "show dashboard") {
		return buildSimple(specDoc, "dashboard", []string{"dashboard"}, nil, nil), "Matched rule: open dashboard", nil
	}
	if strings.Contains(lower, "restart daemon") {
		return buildSimple(specDoc, "daemon.restart", []string{"daemon", "restart"}, nil, nil), "Matched rule: restart daemon", nil
	}
	if cmd, ok := matchSessionRename(specDoc, prompt); ok {
		return cmd, "Matched rule: rename session", nil
	}
	if cmd, ok := matchSessionClose(specDoc, prompt); ok {
		return cmd, "Matched rule: close session", nil
	}
	if cmd, ok := matchSessionStart(specDoc, prompt); ok {
		return cmd, "Matched rule: start session", nil
	}
	if cmd, ok := matchPaneSend(specDoc, prompt); ok {
		return cmd, "Matched rule: send to pane", nil
	}
	if cmd, ok := matchPaneRun(specDoc, prompt); ok {
		return cmd, "Matched rule: run in pane", nil
	}
	return output.NLPlannedCommand{}, "", fmt.Errorf("no rule matched")
}

func matchSessionRename(specDoc *spec.Spec, prompt string) (output.NLPlannedCommand, bool) {
	lower := strings.ToLower(prompt)
	if !strings.Contains(lower, "rename session") {
		return output.NLPlannedCommand{}, false
	}
	parts := strings.Fields(prompt)
	oldIdx := indexOf(parts, "session")
	if oldIdx < 0 || oldIdx+1 >= len(parts) {
		return output.NLPlannedCommand{}, false
	}
	oldName := parts[oldIdx+1]
	newIdx := indexOf(parts, "to")
	if newIdx < 0 || newIdx+1 >= len(parts) {
		return output.NLPlannedCommand{}, false
	}
	newName := parts[newIdx+1]
	flags := map[string]any{"old": oldName, "new": newName}
	return buildSimple(specDoc, "session.rename", []string{"session", "rename"}, nil, flags), true
}

func matchSessionClose(specDoc *spec.Spec, prompt string) (output.NLPlannedCommand, bool) {
	lower := strings.ToLower(prompt)
	if !strings.Contains(lower, "close session") && !strings.Contains(lower, "kill session") {
		return output.NLPlannedCommand{}, false
	}
	parts := strings.Fields(prompt)
	idx := indexOf(parts, "session")
	if idx < 0 || idx+1 >= len(parts) {
		return output.NLPlannedCommand{}, false
	}
	name := parts[idx+1]
	flags := map[string]any{"name": name}
	return buildSimple(specDoc, "session.close", []string{"session", "close"}, nil, flags), true
}

func matchSessionStart(specDoc *spec.Spec, prompt string) (output.NLPlannedCommand, bool) {
	lower := strings.ToLower(prompt)
	if !strings.Contains(lower, "start session") {
		return output.NLPlannedCommand{}, false
	}
	parts := strings.Fields(prompt)
	idx := indexOf(parts, "session")
	if idx < 0 || idx+1 >= len(parts) {
		return output.NLPlannedCommand{}, false
	}
	name := parts[idx+1]
	flags := map[string]any{"name": name}
	return buildSimple(specDoc, "session.start", []string{"session", "start"}, nil, flags), true
}

func matchPaneSend(specDoc *spec.Spec, prompt string) (output.NLPlannedCommand, bool) {
	lower := strings.ToLower(prompt)
	if !strings.Contains(lower, "send") || !strings.Contains(lower, "pane") {
		return output.NLPlannedCommand{}, false
	}
	parts := strings.Fields(prompt)
	paneIdx := indexOf(parts, "pane")
	if paneIdx < 0 || paneIdx+1 >= len(parts) {
		return output.NLPlannedCommand{}, false
	}
	paneID := parts[paneIdx+1]
	textIdx := indexOf(parts, "send")
	if textIdx < 0 || textIdx+1 >= len(parts) {
		return output.NLPlannedCommand{}, false
	}
	text := strings.Join(parts[textIdx+1:paneIdx], " ")
	flags := map[string]any{"pane-id": paneID, "text": text}
	return buildSimple(specDoc, "pane.send", []string{"pane", "send"}, nil, flags), true
}

func matchPaneRun(specDoc *spec.Spec, prompt string) (output.NLPlannedCommand, bool) {
	lower := strings.ToLower(prompt)
	if !strings.Contains(lower, "run") || !strings.Contains(lower, "pane") {
		return output.NLPlannedCommand{}, false
	}
	parts := strings.Fields(prompt)
	paneIdx := indexOf(parts, "pane")
	if paneIdx < 0 || paneIdx+1 >= len(parts) {
		return output.NLPlannedCommand{}, false
	}
	paneID := parts[paneIdx+1]
	cmdIdx := indexOf(parts, "run")
	if cmdIdx < 0 || cmdIdx+1 >= len(parts) {
		return output.NLPlannedCommand{}, false
	}
	command := strings.Join(parts[cmdIdx+1:paneIdx], " ")
	flags := map[string]any{"pane-id": paneID, "command": command}
	return buildSimple(specDoc, "pane.run", []string{"pane", "run"}, nil, flags), true
}

func buildSimple(specDoc *spec.Spec, id string, commandTokens []string, args []string, flags map[string]any) output.NLPlannedCommand {
	cmdSpec := specDoc.FindByID(id)
	if cmdSpec == nil {
		return output.NLPlannedCommand{ID: id, Command: strings.Join(commandTokens, " ")}
	}
	return buildPlannedCommand(*cmdSpec, commandTokens, args, flags)
}

func buildPlannedCommand(cmdSpec spec.Command, commandTokens []string, args []string, flags map[string]any) output.NLPlannedCommand {
	if flags == nil {
		flags = map[string]any{}
	}
	command := strings.Join(commandTokens, " ")
	return output.NLPlannedCommand{
		ID:              cmdSpec.ID,
		Command:         command,
		Args:            args,
		Flags:           flags,
		Summary:         cmdSpec.Summary,
		SideEffects:     cmdSpec.SideEffects,
		RequiresConfirm: cmdSpec.Confirm,
	}
}

func shellTokens(prompt string) ([]string, bool) {
	tokens, err := shellquote.Split(prompt)
	if err != nil || len(tokens) == 0 {
		return nil, false
	}
	if identity.IsCLICommandToken(tokens[0]) {
		return tokens[1:], true
	}
	if strings.HasPrefix(tokens[0], "-") {
		return nil, false
	}
	return tokens, true
}

func matchCommand(commands []spec.Command, tokens []string) (*spec.Command, int) {
	if len(tokens) == 0 {
		return nil, 0
	}
	for _, cmd := range commands {
		if !tokenMatches(cmd, tokens[0]) {
			continue
		}
		if len(tokens) > 1 && len(cmd.Subcommands) > 0 {
			if sub, consumed := matchCommand(cmd.Subcommands, tokens[1:]); sub != nil {
				return sub, consumed + 1
			}
		}
		return &cmd, 1
	}
	return nil, 0
}

func tokenMatches(cmd spec.Command, token string) bool {
	token = strings.ToLower(token)
	if strings.ToLower(cmd.Name) == token {
		return true
	}
	for _, alias := range cmd.Aliases {
		if strings.ToLower(alias) == token {
			return true
		}
	}
	return false
}

func parseFlags(cmd spec.Command, tokens []string) (map[string]any, []string) {
	flags := map[string]any{}
	lookup := flagLookup(cmd)
	args := []string{}
	for i := 0; i < len(tokens); i++ {
		token := tokens[i]
		if strings.HasPrefix(token, "--") {
			name, val, hasVal := strings.Cut(strings.TrimPrefix(token, "--"), "=")
			flag, ok := lookup[name]
			if !ok {
				args = append(args, token)
				continue
			}
			if flag.Type == "bool" {
				flags[flag.Name] = true
				continue
			}
			if !hasVal {
				if i+1 < len(tokens) {
					val = tokens[i+1]
					i++
				}
			}
			assignFlag(flags, flag.Name, val, flag.Repeatable)
			continue
		}
		if strings.HasPrefix(token, "-") {
			name := strings.TrimPrefix(token, "-")
			flag, ok := lookup[name]
			if !ok {
				args = append(args, token)
				continue
			}
			if flag.Type == "bool" {
				flags[flag.Name] = true
				continue
			}
			val := ""
			if i+1 < len(tokens) {
				val = tokens[i+1]
				i++
			}
			assignFlag(flags, flag.Name, val, flag.Repeatable)
			continue
		}
		args = append(args, token)
	}
	return flags, args
}

func assignFlag(flags map[string]any, name, value string, repeatable bool) {
	if repeatable {
		current, ok := flags[name]
		if !ok {
			flags[name] = []string{value}
			return
		}
		slice, ok := current.([]string)
		if !ok {
			flags[name] = []string{value}
			return
		}
		flags[name] = append(slice, value)
		return
	}
	flags[name] = value
}

func flagLookup(cmd spec.Command) map[string]spec.Flag {
	lookup := make(map[string]spec.Flag)
	for _, flag := range cmd.Flags {
		lookup[flag.Name] = flag
		for _, alias := range flag.Aliases {
			lookup[alias] = flag
		}
	}
	return lookup
}

func indexOf(parts []string, match string) int {
	match = strings.ToLower(match)
	for i, part := range parts {
		if strings.ToLower(part) == match {
			return i
		}
	}
	return -1
}

func hasNLCommand(cmds []output.NLPlannedCommand) bool {
	for _, cmd := range cmds {
		if strings.HasPrefix(cmd.ID, "nl.") {
			return true
		}
	}
	return false
}

func executePlan(ctx root.CommandContext, plan output.NLPlan) (output.NLExecution, error) {
	planID := plan.PlanID
	if planID == "" {
		planID = fmt.Sprintf("plan-%d", time.Now().UnixNano())
	}
	execID := fmt.Sprintf("exec-%d", time.Now().UnixNano())
	exec := output.NLExecution{ExecutionID: execID, PlanID: planID, Steps: nil}
	specDoc, err := spec.LoadDefault()
	if err != nil {
		return exec, err
	}
	reg := root.NewRegistry()
	catalog.RegisterAll(reg)
	for _, cmd := range plan.Commands {
		step := output.NLExecutionStep{ID: cmd.ID, Command: cmd.Command, Status: "pending"}
		step.StartedAt = time.Now().UTC()
		buf := &bytes.Buffer{}
		deps := root.Dependencies{
			Version: ctx.Deps.Version,
			AppName: identity.CLIName,
			Stdout:  buf,
			Stderr:  ctx.ErrOut,
			Stdin:   ctx.Stdin,
			Connect: ctx.Deps.Connect,
		}
		cmdRunner, err := root.NewRunner(specDoc, deps, reg)
		if err != nil {
			return exec, err
		}
		args := buildArgs(cmd)
		if supportsJSON(cmd.ID) {
			args = insertGlobalFlags(args, "--json", "--yes")
		}
		err = cmdRunner.Run(ctx.Context, args)
		step.FinishedAt = time.Now().UTC()
		if err != nil {
			step.Status = "failed"
			step.Result = parseActionResult(buf)
		} else {
			step.Status = "ok"
			step.Result = parseActionResult(buf)
		}
		exec.Steps = append(exec.Steps, step)
	}
	return exec, nil
}

func buildArgs(cmd output.NLPlannedCommand) []string {
	args := []string{identity.CLIName}
	args = append(args, strings.Fields(cmd.Command)...)
	for key, val := range cmd.Flags {
		switch typed := val.(type) {
		case bool:
			if typed {
				args = append(args, "--"+key)
			}
		case []string:
			for _, item := range typed {
				args = append(args, "--"+key, item)
			}
		default:
			args = append(args, "--"+key, fmt.Sprint(typed))
		}
	}
	args = append(args, cmd.Args...)
	return args
}

func insertGlobalFlags(args []string, flags ...string) []string {
	if len(args) == 0 {
		return flags
	}
	out := make([]string, 0, len(args)+len(flags))
	out = append(out, args[0])
	out = append(out, flags...)
	out = append(out, args[1:]...)
	return out
}

func supportsJSON(cmdID string) bool {
	specDoc, err := spec.LoadDefault()
	if err != nil {
		return false
	}
	cmdSpec := specDoc.FindByID(cmdID)
	if cmdSpec == nil || cmdSpec.JSON == nil {
		return false
	}
	return cmdSpec.JSON.Supported
}

func parseActionResult(buf *bytes.Buffer) *output.ActionResult {
	if buf == nil || buf.Len() == 0 {
		return nil
	}
	dec := json.NewDecoder(bufio.NewReader(bytes.NewReader(buf.Bytes())))
	var envelope struct {
		Ok   bool            `json:"ok"`
		Data json.RawMessage `json:"data"`
	}
	if err := dec.Decode(&envelope); err != nil {
		return nil
	}
	if len(envelope.Data) == 0 {
		return nil
	}
	var result output.ActionResult
	if err := json.Unmarshal(envelope.Data, &result); err != nil {
		return nil
	}
	if result.Action == "" {
		return nil
	}
	return &result
}
