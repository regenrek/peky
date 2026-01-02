package agenttool

import (
	"strings"

	"github.com/kballard/go-shellquote"
)

// Tool identifies a known interactive agent tool.
type Tool string

const (
	ToolCodex  Tool = "codex"
	ToolClaude Tool = "claude"
)

var knownTools = []Tool{
	ToolCodex,
	ToolClaude,
}

// Normalize returns the canonical tool name or an empty string when unknown.
func Normalize(value string) Tool {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return ""
	}
	value = strings.TrimSuffix(value, ".exe")
	for _, tool := range knownTools {
		name := string(tool)
		if value == name || strings.HasPrefix(value, name+"@") {
			return tool
		}
	}
	return ""
}

// DetectFromTitle attempts to infer a tool name from a pane title.
func DetectFromTitle(title string) Tool {
	title = strings.TrimSpace(title)
	if title == "" {
		return ""
	}
	fields := strings.Fields(strings.ToLower(title))
	if len(fields) == 0 {
		return ""
	}
	first := strings.TrimSuffix(fields[0], ":")
	return Normalize(first)
}

// DetectFromCommand attempts to infer a tool name from a launch command.
func DetectFromCommand(command string) Tool {
	command = strings.TrimSpace(command)
	if command == "" {
		return ""
	}
	args, err := shellquote.Split(command)
	if err != nil || len(args) == 0 {
		return ""
	}
	exe := executableArg(args)
	if exe == "" {
		return ""
	}
	return DetectFromArg(exe)
}

// DetectFromArg attempts to infer a tool name from an argument or path.
func DetectFromArg(arg string) Tool {
	base := strings.ToLower(strings.TrimSpace(baseNameAnySeparator(arg)))
	if base == "" {
		return ""
	}
	base = strings.TrimSuffix(base, ".exe")
	return Normalize(base)
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
