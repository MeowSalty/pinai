package openai

import (
	"github.com/MeowSalty/pinai/services/portal"
	"github.com/gofiber/fiber/v2"
)

// SetupOpenAIRoutes registers the OpenAI compatible routes.
func SetupOpenAIRoutes(router fiber.Router, portalService portal.Service, userAgent string) {
	// 创建 Handler 实例，传入 userAgent 配置
	handler := New(portalService, userAgent)

	router.Get("/models", ListModels)
	router.Post("/chat/completions", handler.ChatCompletions)
	router.Post("/responses", handler.Responses)
}
