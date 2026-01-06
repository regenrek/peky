package agent

import "strings"

func filterToolCalls(provider Provider, calls []ToolCall) []ToolCall {
	if len(calls) == 0 {
		return calls
	}
	out := make([]ToolCall, 0, len(calls))
	seen := map[string]struct{}{}
	requireSig := providerRequiresThoughtSignature(provider)
	for _, call := range calls {
		if requireSig && strings.TrimSpace(call.ThoughtSignature) == "" {
			continue
		}
		key := toolCallKey(call)
		if key != "" {
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
		}
		out = append(out, call)
	}
	return out
}

func providerRequiresThoughtSignature(provider Provider) bool {
	switch provider {
	case ProviderGoogle, ProviderGoogleGeminiCLI, ProviderGoogleAntigrav:
		return true
	default:
		return false
	}
}

func toolCallKey(call ToolCall) string {
	if !strings.EqualFold(call.Name, "peky") {
		return ""
	}
	command, ok := call.Arguments["command"].(string)
	if !ok {
		return ""
	}
	command = strings.Join(strings.Fields(command), " ")
	if command == "" {
		return ""
	}
	return "peky:" + command
}
