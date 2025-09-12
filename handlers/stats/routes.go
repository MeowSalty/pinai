package stats

import (
	"github.com/gofiber/fiber/v2"

	"github.com/MeowSalty/pinai/services"
)

// SetupStatsRoutes 配置统计相关的路由
func SetupStatsRoutes(router fiber.Router, statsService services.StatsServiceInterface) {
	handler := NewStatsHandler(statsService)

	statsGroup := router.Group("/stats")
	statsGroup.Get("/overview", handler.GetOverview)
	statsGroup.Get("/requests", handler.ListRequestStats)
}
