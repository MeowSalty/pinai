package provider

import (
	"github.com/MeowSalty/pinai/services/health"
	"github.com/MeowSalty/pinai/services/provider"

	"github.com/gofiber/fiber/v2"
)

// SetupProviderRoutes 配置 LLM 供应方管理相关的 API 路由
func SetupProviderRoutes(router fiber.Router, llmService provider.Service, healthService health.Service) {
	// 创建单一的 Provider Handler 实例
	handler := NewHandler(llmService, healthService)

	// 平台 (Platforms) 相关路由
	router.Post("/platforms", handler.CreatePlatform)
	router.Get("/platforms", handler.GetPlatforms)
	router.Get("/platforms/:id", handler.GetPlatform)
	router.Put("/platforms/:id", handler.UpdatePlatform)
	router.Delete("/platforms/:id", handler.DeletePlatform)

	// 平台健康状态管理路由
	router.Post("/platforms/:id/health/enable", handler.EnablePlatformHealth)
	router.Post("/platforms/:id/health/disable", handler.DisablePlatformHealth)

	// 模型 (Models) 相关路由 (嵌套在平台下)
	router.Post("/platforms/:platformId/models", handler.AddModelToPlatform)
	router.Post("/platforms/:platformId/models/batch", handler.BatchAddModelsToPlatform)
	router.Get("/platforms/:platformId/models", handler.GetModelsByPlatform)
	router.Put("/platforms/:platformId/models/batch", handler.BatchUpdateModels)
	router.Put("/platforms/:platformId/models/:modelId", handler.UpdateModel)
	router.Delete("/platforms/:platformId/models/batch", handler.BatchDeleteModels)
	router.Delete("/platforms/:platformId/models/:modelId", handler.DeleteModel)

	// 模型健康状态管理路由
	router.Post("/platforms/:platformId/models/:modelId/health/enable", handler.EnableModelHealth)
	router.Post("/platforms/:platformId/models/:modelId/health/disable", handler.DisableModelHealth)

	// 密钥 (Keys) 相关路由 (嵌套在平台下)
	router.Post("/platforms/:platformId/keys", handler.AddKeyToPlatform)
	router.Get("/platforms/:platformId/keys", handler.GetKeysByPlatform)
	router.Put("/platforms/:platformId/keys/:keyId", handler.UpdateKey)
	router.Delete("/platforms/:platformId/keys/:keyId", handler.DeleteKey)

	// 密钥健康状态管理路由
	router.Post("/platforms/:platformId/keys/:keyId/health/enable", handler.EnableKeyHealth)
	router.Post("/platforms/:platformId/keys/:keyId/health/disable", handler.DisableKeyHealth)
}
