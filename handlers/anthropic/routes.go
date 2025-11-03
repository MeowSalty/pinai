package anthropic

import (
	"github.com/MeowSalty/pinai/services/portal"
	"github.com/gofiber/fiber/v2"
)

// SetupAnthropicRoutes 注册 Anthropic 兼容路由。
func SetupAnthropicRoutes(router fiber.Router, portalService portal.Service, userAgent string) {
	// 创建 Handler 实例，传入 userAgent 配置
	handler := New(portalService, userAgent)

	router.Get("/models", ListModels)
	router.Post("/messages", handler.Messages)
}
