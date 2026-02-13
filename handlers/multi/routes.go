package multi

import (
	"log/slog"

	"github.com/MeowSalty/pinai/handlers/multi/auth"
	"github.com/MeowSalty/pinai/handlers/multi/native"
	"github.com/MeowSalty/pinai/services/portal"
	"github.com/gofiber/fiber/v2"
)

// SetupMultiRoutes 注册 multi 兼容路由。
func SetupMultiRoutes(
	rootRouter fiber.Router,
	portalService portal.Service,
	userAgent string,
	logger *slog.Logger,
	apiToken string,
) {
	// 配置子路由
	nativeRouter := rootRouter.Group("/native")
	v1Router := rootRouter.Group("/v1")
	v1betaRouter := rootRouter.Group("/v1beta")

	// 创建认证策略注册表
	authRegistry := auth.NewRegistry(apiToken)

	rootRouter.Use(auth.NewProviderMiddleware(authRegistry, apiToken))
	nativeRouter.Use(auth.NewProviderMiddleware(authRegistry, apiToken))

	// 创建 Handler 实例，传入 userAgent 配置
	handler := New(portalService, userAgent, logger)

	// 注册 OpenAI 兼容路由
	v1Router.Post("/chat/completions", handler.ChatCompletions)
	v1Router.Post("/responses", handler.Responses)

	// 注册 Anthropic 兼容路由
	v1Router.Post("/messages", handler.Messages)

	// 注册 Gemini 兼容路由
	v1betaRouter.Post("/models/:model<[^:]+>:generateContent", handler.GeminiGenerateContent)
	v1betaRouter.Post("/models/:model<[^:]+>:streamGenerateContent", handler.GeminiStreamGenerateContent)

	// 模型列表
	v1Router.Get("/models", handler.SelectModels())
	v1betaRouter.Get("/models", handler.SelectGeminiModels())

	// 原生请求
	native.SetupNativeRoutes(nativeRouter, portalService, userAgent, logger)
}
