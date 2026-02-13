package auth

import (
	"strings"

	"github.com/gofiber/fiber/v2"
)

const ProviderLocalKey = "provider"

// NewProviderMiddleware validates provider-specific auth and stores provider in context.
func NewProviderMiddleware(registry Registry, apiToken string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		provider := ResolveProvider(c)
		c.Locals(ProviderLocalKey, provider)
		if apiToken != "" {
			strategy := registry[provider]
			if strategy != nil {
				if err := strategy.Validate(c); err != nil {
					return err
				}
			}
		}
		return c.Next()
	}
}

func ProviderFromContext(c *fiber.Ctx) string {
	if value := c.Locals(ProviderLocalKey); value != nil {
		if provider, ok := value.(string); ok {
			return provider
		}
	}
	return ""
}

// ResolveProvider determines provider based on path, query, and headers.
func ResolveProvider(c *fiber.Ctx) string {
	if provider := providerFromPath(c.Path()); provider != "" {
		return provider
	}
	if provider := providerFromQuery(c); provider != "" {
		return provider
	}
	if isGeminiRequest(c) {
		return ProviderGemini
	}
	if isAnthropicRequest(c) {
		return ProviderAnthropic
	}
	return ProviderOpenAI
}

func providerFromQuery(c *fiber.Ctx) string {
	provider := strings.ToLower(strings.TrimSpace(c.Query("provider")))
	switch provider {
	case ProviderOpenAI, ProviderAnthropic, ProviderGemini:
		return provider
	default:
		return ""
	}
}

func providerFromPath(path string) string {
	switch {
	case strings.HasSuffix(path, "/v1beta/models"):
		return ProviderGemini
	case strings.HasSuffix(path, "/generateContent"), strings.HasSuffix(path, "/streamGenerateContent"):
		return ProviderGemini
	case strings.HasSuffix(path, "/messages"), strings.HasSuffix(path, "/messages/stream"):
		return ProviderAnthropic
	case strings.HasSuffix(path, "/chat/completions"), strings.HasSuffix(path, "/chat/completions/stream"):
		return ProviderOpenAI
	case strings.HasSuffix(path, "/responses"), strings.HasSuffix(path, "/responses/stream"):
		return ProviderOpenAI
	default:
		return ""
	}
}

func isAnthropicRequest(c *fiber.Ctx) bool {
	if c.Get(AnthropicAPIKeyHeader) == "" {
		return false
	}
	return c.Get(AnthropicVersionHeader) != ""
}

func isGeminiRequest(c *fiber.Ctx) bool {
	if c.Get(GeminiAPIKeyHeader) != "" {
		return true
	}
	return c.Query("key") != ""
}
