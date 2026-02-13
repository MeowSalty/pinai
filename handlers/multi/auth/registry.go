package auth

// Provider names.
const (
	ProviderOpenAI    = "openai"
	ProviderAnthropic = "anthropic"
	ProviderGemini    = "gemini"
)

// Registry maps provider name to auth strategy.
type Registry map[string]Strategy

// NewRegistry constructs a registry with all providers.
func NewRegistry(apiToken string) Registry {
	return Registry{
		ProviderOpenAI:    OpenAIAuth{Token: apiToken},
		ProviderAnthropic: AnthropicAuth{Token: apiToken},
		ProviderGemini:    GeminiAuth{Token: apiToken},
	}
}
