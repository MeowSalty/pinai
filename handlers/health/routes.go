package health

import (
	"github.com/gofiber/fiber/v2"

	"github.com/MeowSalty/pinai/services/health"
)

// SetupHealthRoutes 配置健康状态统计相关的路由
func SetupHealthRoutes(router fiber.Router, healthService health.Service) {
	handler := NewHandler(healthService)

	healthGroup := router.Group("/health")
	healthGroup.Get("/summary", handler.GetHealthSummary)

	// 平台健康端点
	healthGroup.Get("/platforms", handler.GetPlatformHealthList)
	healthGroup.Post("/platforms/:platformId/enable", handler.EnablePlatform)
	healthGroup.Post("/platforms/:platformId/disable", handler.DisablePlatform)

	// 密钥健康端点
	healthGroup.Get("/keys", handler.GetAPIKeyHealthList)
	healthGroup.Post("/keys/:keyId/enable", handler.EnableAPIKey)
	healthGroup.Post("/keys/:keyId/disable", handler.DisableAPIKey)

	// 模型健康端点
	healthGroup.Get("/models", handler.GetModelHealthList)
	healthGroup.Post("/models/:modelId/enable", handler.EnableModel)
	healthGroup.Post("/models/:modelId/disable", handler.DisableModel)
}
