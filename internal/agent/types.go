package agent

import "time"

type Provider string

const (
	ProviderGoogle          Provider = "google"
	ProviderGoogleGeminiCLI Provider = "google-gemini-cli"
	ProviderGoogleAntigrav  Provider = "google-antigravity"
	ProviderAnthropic       Provider = "anthropic"
	ProviderOpenAI          Provider = "openai"
	ProviderOpenRouter      Provider = "openrouter"
	ProviderGitHubCopilot   Provider = "github-copilot"
	ProviderUnknown         Provider = ""
)

type MessageRole string

const (
	RoleUser      MessageRole = "user"
	RoleAssistant MessageRole = "assistant"
	RoleTool      MessageRole = "tool"
)

type ToolCall struct {
	ID               string
	Name             string
	Arguments        map[string]any
	ThoughtSignature string
}

type ToolResult struct {
	ToolCallID string
	ToolName   string
	Content    string
	IsError    bool
}

type Message struct {
	Role       MessageRole
	Text       string
	ToolCalls  []ToolCall
	ToolResult *ToolResult
	Timestamp  time.Time
}

type Usage struct {
	InputTokens  int
	OutputTokens int
	TotalTokens  int
}

type Result struct {
	Text       string
	Usage      Usage
	ToolCalls  []ToolCall
	Provider   Provider
	Model      string
	StopReason string
}

type ToolSpec struct {
	Name        string
	Description string
	Schema      map[string]any
}

func NewUserMessage(text string) Message {
	return Message{Role: RoleUser, Text: text, Timestamp: time.Now()}
}

func NewAssistantMessage(text string, calls []ToolCall) Message {
	return Message{Role: RoleAssistant, Text: text, ToolCalls: calls, Timestamp: time.Now()}
}

func NewToolResultMessage(result ToolResult) Message {
	return Message{Role: RoleTool, ToolResult: &result, Timestamp: time.Now()}
}
