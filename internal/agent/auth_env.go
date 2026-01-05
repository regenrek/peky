package agent

import "os"

func envAPIKey(provider string) string {
	switch Provider(provider) {
	case ProviderGoogle:
		if value := os.Getenv("GEMINI_API_KEY"); value != "" {
			return value
		}
		return os.Getenv("GOOGLE_API_KEY")
	case ProviderAnthropic:
		return os.Getenv("ANTHROPIC_API_KEY")
	case ProviderOpenAI:
		return os.Getenv("OPENAI_API_KEY")
	case ProviderOpenRouter:
		return os.Getenv("OPENROUTER_API_KEY")
	default:
		return ""
	}
}
