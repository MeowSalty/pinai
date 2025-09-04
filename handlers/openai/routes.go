package openai

import (
	"log/slog"

	"github.com/MeowSalty/pinai/services"
	"github.com/gofiber/fiber/v2"
)

// SetupOpenAIRoutes registers the OpenAI compatible routes.
func SetupOpenAIRoutes(router fiber.Router, aiGatewayService services.PortalService, logger *slog.Logger) {
	// 创建 Handler 实例
	handler := New(aiGatewayService, logger)

	router.Get("/models", ListModels)
	router.Post("/chat/completions", handler.ChatCompletions)
}
