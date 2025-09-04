package openai

import (
	"github.com/MeowSalty/pinai/services"
	"github.com/gofiber/fiber/v2"
)

// SetupOpenAIRoutes registers the OpenAI compatible routes.
func SetupOpenAIRoutes(router fiber.Router, aiGatewayService services.PortalService) {
	// 创建 Handler 实例
	handler := New(aiGatewayService)

	router.Get("/models", ListModels)
	router.Post("/chat/completions", handler.ChatCompletions)
}
