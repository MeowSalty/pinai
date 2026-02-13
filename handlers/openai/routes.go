// Deprecated: 不再维护，逐步迁移到 github.com/MeowSalty/pinai/handlers/multi 包。
package openai

import (
	"log/slog"

	"github.com/MeowSalty/pinai/services/portal"
	"github.com/gofiber/fiber/v2"
)

// SetupOpenAIRoutes registers the OpenAI compatible routes.
func SetupOpenAIRoutes(router fiber.Router, portalService portal.Service, userAgent string, logger *slog.Logger) {
	// 创建 Handler 实例，传入 userAgent 配置
	handler := New(portalService, userAgent, logger)

	router.Get("/models", ListModels)
	router.Post("/chat/completions", handler.ChatCompletions)
	router.Post("/responses", handler.Responses)
}
