package auth

import (
	"strings"

	"github.com/gin-gonic/gin"
)

const ProviderLocalKey = "provider"

// NewProviderMiddleware validates provider-specific auth and stores provider in context.
func NewProviderMiddleware(registry Registry, apiToken string) gin.HandlerFunc {
	return func(c *gin.Context) {
		provider := ResolveProvider(c)
		c.Set(ProviderLocalKey, provider)
		if apiToken != "" {
			strategy := registry[provider]
			if strategy != nil {
				if !strategy.Validate(c) {
					return
				}
			}
		}
		c.Next()
	}
}

func ProviderFromContext(c *gin.Context) string {
	return c.GetString(ProviderLocalKey)
}

// ResolveProvider determines provider based on path, query, and headers.
func ResolveProvider(c *gin.Context) string {
	if provider := providerFromPath(c.Request.URL.Path); provider != "" {
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

func providerFromQuery(c *gin.Context) string {
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

func isAnthropicRequest(c *gin.Context) bool {
	if c.GetHeader(AnthropicAPIKeyHeader) == "" {
		return false
	}
	return c.GetHeader(AnthropicVersionHeader) != ""
}

func isGeminiRequest(c *gin.Context) bool {
	if c.GetHeader(GeminiAPIKeyHeader) != "" {
		return true
	}
	return c.Query("key") != ""
}
