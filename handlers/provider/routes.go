package provider

import (
	"github.com/MeowSalty/pinai/services"

	"github.com/gofiber/fiber/v2"
)

// SetupProviderRoutes 配置 LLM 供应方管理相关的 API 路由
func SetupProviderRoutes(router fiber.Router, llmService services.ProviderService) {
	// 创建 Provider Handler 实例
	handler := NewHandler(llmService)

	// 供应方 (Providers) 相关路由
	router.Post("/providers", handler.CreateProvider)
	router.Get("/providers", handler.GetProviders)
	router.Get("/providers/:id", handler.GetProvider)
	router.Put("/providers/:id", handler.UpdateProvider)
	router.Delete("/providers/:id", handler.DeleteProvider)

	// 模型 (Models) 相关路由 (嵌套在供应方下)
	router.Post("/providers/:providerId/models", handler.AddModelToProvider)
	router.Get("/providers/:providerId/models", handler.GetModelsByProvider)
	router.Put("/providers/:providerId/models/:modelId", handler.UpdateModel)
	router.Delete("/providers/:providerId/models/:modelId", handler.DeleteModel)

	// 密钥 (Keys) 相关路由 (嵌套在供应方下)
	router.Post("/providers/:providerId/keys", handler.AddKeyToProvider)
	router.Get("/providers/:providerId/keys", handler.GetKeysByProvider)
	router.Put("/providers/:providerId/keys/:keyId", handler.UpdateKey)
	router.Delete("/providers/:providerId/keys/:keyId", handler.DeleteKey)
}
