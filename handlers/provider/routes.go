package provider

import (
	"github.com/MeowSalty/pinai/services/provider"

	"github.com/gofiber/fiber/v2"
)

// SetupProviderRoutes 配置 LLM 供应方管理相关的 API 路由
func SetupProviderRoutes(router fiber.Router, llmService provider.Service) {
	// 创建单一的 Provider Handler 实例
	handler := NewHandler(llmService)

	// 供应方 (Providers) 相关路由
	router.Post("/providers", handler.CreateProvider)
	router.Delete("/provider/:id", handler.DeleteProvider)

	// 平台 (Platforms) 相关路由
	router.Get("/platforms", handler.GetPlatforms)
	router.Get("/platform/:id", handler.GetPlatform)
	router.Put("/platform/:id", handler.UpdatePlatform)

	// 模型 (Models) 相关路由 (嵌套在平台下)
	router.Post("/platform/:platformId/models", handler.AddModelToPlatform)
	router.Get("/platform/:platformId/models", handler.GetModelsByPlatform)
	router.Put("/platform/:platformId/models/:modelId", handler.UpdateModel)
	router.Delete("/platform/:platformId/models/:modelId", handler.DeleteModel)

	// 密钥 (Keys) 相关路由 (嵌套在平台下)
	router.Post("/platform/:platformId/keys", handler.AddKeyToPlatform)
	router.Get("/platform/:platformId/keys", handler.GetKeysByPlatform)
	router.Put("/platform/:platformId/keys/:keyId", handler.UpdateKey)
	router.Delete("/platform/:platformId/keys/:keyId", handler.DeleteKey)
}
