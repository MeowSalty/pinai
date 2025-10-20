package anthropic

import (
	"github.com/MeowSalty/pinai/services"
	"github.com/gofiber/fiber/v2"
)

// SetupAnthropicRoutes 注册 Anthropic 兼容路由。
func SetupAnthropicRoutes(router fiber.Router, aiGatewayService services.PortalService) {
	// 创建 Handler 实例
	handler := New(aiGatewayService)

	router.Get("/models", ListModels)
	router.Post("/messages", handler.Messages)
}