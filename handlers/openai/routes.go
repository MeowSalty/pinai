package openai

import (
	"log/slog"

	"github.com/MeowSalty/pinai/services"
	"github.com/gofiber/fiber/v2"
)

// RegisterOpenAIRoutes registers the OpenAI compatible routes.
func RegisterOpenAIRoutes(router fiber.Router, aiGatewayService services.AIGatewayService, logger *slog.Logger) {
	// 创建 Handler 实例
	handler := NewOpenAIHandler(aiGatewayService, logger)

	router.Get("/models", ListModels)
	router.Post("/chat/completions", handler.ChatCompletions)
}
