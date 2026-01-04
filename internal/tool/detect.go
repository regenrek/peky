package tool

import (
	"bytes"
	"strings"

	"github.com/kballard/go-shellquote"
)

// ResolveTool determines the best tool match from pane metadata.
func (r *Registry) ResolveTool(info PaneInfo) string {
	if r == nil {
		return ""
	}
	if explicit := r.Normalize(info.Tool); explicit != "" {
		if r.Allowed(explicit) {
			return explicit
		}
		return ""
	}
	if !r.enabled {
		return ""
	}
	if tool := r.DetectFromCommand(info.StartCommand); tool != "" {
		return tool
	}
	if tool := r.DetectFromCommand(info.Command); tool != "" {
		return tool
	}
	if tool := r.DetectFromTitle(info.Title); tool != "" {
		return tool
	}
	return ""
}

// DetectFromTitle attempts to infer a tool name from a pane title.
func (r *Registry) DetectFromTitle(title string) string {
	if r == nil || !r.enabled {
		return ""
	}
	title = strings.TrimSpace(title)
	if title == "" {
		return ""
	}
	for _, name := range r.order {
		def := r.defs[name]
		for _, re := range def.TitleRegex {
			if re.MatchString(title) {
				if r.Allowed(def.Name) {
					return def.Name
				}
				return ""
			}
		}
	}
	fields := strings.Fields(strings.ToLower(title))
	if len(fields) == 0 {
		return ""
	}
	first := strings.TrimSuffix(fields[0], ":")
	name := r.Normalize(first)
	if name != "" && r.Allowed(name) {
		return name
	}
	return ""
}

// DetectFromCommand attempts to infer a tool name from a command line.
func (r *Registry) DetectFromCommand(command string) string {
	if r == nil || !r.enabled {
		return ""
	}
	command = strings.TrimSpace(command)
	if command == "" {
		return ""
	}
	if exe := commandExecutable(command); exe != "" {
		name := r.Normalize(baseNameAnySeparator(exe))
		if name != "" && r.Allowed(name) {
			return name
		}
	}
	for _, name := range r.order {
		def := r.defs[name]
		for _, re := range def.CommandRegex {
			if re.MatchString(command) {
				if r.Allowed(def.Name) {
					return def.Name
				}
				return ""
			}
		}
	}
	return ""
}

// DetectFromInput checks a single-line input for a tool command.
func (r *Registry) DetectFromInput(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	if strings.ContainsAny(text, "\r\n") {
		return ""
	}
	return r.DetectFromCommand(text)
}

// DetectFromInputBytes inspects a bounded prefix of input for a tool command.
func (r *Registry) DetectFromInputBytes(input []byte, limit int) string {
	if len(input) == 0 {
		return ""
	}
	if limit <= 0 {
		return ""
	}
	if limit > len(input) {
		limit = len(input)
	}
	trimmed := bytes.TrimSpace(input[:limit])
	if len(trimmed) == 0 {
		return ""
	}
	if bytes.ContainsAny(trimmed, "\r\n") {
		return ""
	}
	return r.DetectFromCommand(string(trimmed))
}

func commandExecutable(command string) string {
	args, err := shellquote.Split(command)
	if err != nil || len(args) == 0 {
		return ""
	}
	return executableArg(args)
}

func executableArg(args []string) string {
	if len(args) == 0 {
		return ""
	}
	first := strings.ToLower(strings.TrimSpace(args[0]))
	if first == "" {
		return ""
	}
	if isWrapperCommand(first) {
		for _, arg := range args[1:] {
			arg = strings.TrimSpace(arg)
			if arg == "" {
				continue
			}
			if first == "env" && isEnvAssignment(arg) {
				continue
			}
			if strings.HasPrefix(arg, "-") {
				continue
			}
			return arg
		}
		return ""
	}
	return args[0]
}

func isWrapperCommand(cmd string) bool {
	switch cmd {
	case "env", "npx", "pnpm", "yarn", "npm", "bunx", "bun", "sudo":
		return true
	default:
		return false
	}
}

func isEnvAssignment(arg string) bool {
	if strings.HasPrefix(arg, "-") {
		return false
	}
	return strings.Contains(arg, "=")
}

func baseNameAnySeparator(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	path = strings.TrimRight(path, "/\\")
	if path == "" {
		return ""
	}
	idx := strings.LastIndexAny(path, "/\\")
	if idx < 0 {
		return path
	}
	return path[idx+1:]
}
