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
}
