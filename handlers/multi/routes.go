package multi

import (
	"log/slog"

	"github.com/MeowSalty/pinai/handlers/anthropic"
	"github.com/MeowSalty/pinai/handlers/openai"
	"github.com/MeowSalty/pinai/services/portal"
	"github.com/gofiber/fiber/v2"
)

// SetupMultiRoutes 注册 multi 兼容路由。
func SetupMultiRoutes(
	openaiRouter fiber.Router,
	anthropicRouter fiber.Router,
	rootRouter fiber.Router,
	portalService portal.Service,
	userAgent string,
	logger *slog.Logger,
	apiToken string,
) {
	// 创建 Handler 实例，传入 userAgent 配置
	openaiHandler := openai.New(portalService, userAgent, logger)
	anthropicHandler := anthropic.New(portalService, userAgent)

	openaiRouter.Post("/chat/completions", openaiHandler.ChatCompletions)
	openaiRouter.Post("/responses", openaiHandler.Responses)

	anthropicRouter.Post("/messages", anthropicHandler.Messages)
	rootRouter.Get("/models", SelectModels(apiToken))
}
