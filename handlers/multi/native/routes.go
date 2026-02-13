package native

import (
	"log/slog"

	"github.com/MeowSalty/pinai/services/portal"
	"github.com/gofiber/fiber/v2"
)

// SetupNativeRoutes 设置多合一原生路由。
func SetupNativeRoutes(
	rootRouter fiber.Router,
	portalService portal.Service,
	userAgent string,
	logger *slog.Logger,
) {
	// 配置子路由
	v1Router := rootRouter.Group("/v1")
	v1betaRouter := rootRouter.Group("/v1beta")

	handler := New(portalService, userAgent, logger)

	// 注册 OpenAI 原生路由
	v1Router.Post("/chat/completions", handler.OpenAIChatCompletions)
	v1Router.Post("/responses", handler.OpenAIResponses)

	// 注册 Anthropic 原生路由
	v1Router.Post("/messages", handler.AnthropicMessages)

	// 注册 Gemini 原生路由
	v1betaRouter.Post("/models/:model<[^:]+>:generateContent", handler.GeminiGenerateContent)
	v1betaRouter.Post("/models/:model<[^:]+>:streamGenerateContent", handler.GeminiStreamGenerateContent)

	// 模型列表
	v1Router.Get("/models", SelectModels())
	v1betaRouter.Get("/models", SelectGeminiModels())
}
