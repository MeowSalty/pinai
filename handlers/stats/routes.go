package stats

import (
	"github.com/gofiber/fiber/v2"

	"github.com/MeowSalty/pinai/services/stats"
)

// SetupStatsRoutes 配置统计相关的路由
func SetupStatsRoutes(router fiber.Router, statsService stats.Service) {
	handler := NewStatsHandler(statsService)

	statsGroup := router.Group("/stats")
	statsGroup.Get("/overview", handler.GetOverview)
	statsGroup.Get("/requests", handler.ListRequestLogs)
	statsGroup.Get("/realtime", handler.GetRealtime)
	statsGroup.Get("/models/call-rank", handler.GetModelCallRank)
	statsGroup.Get("/platforms/call-rank", handler.GetPlatformCallRank)
	statsGroup.Get("/models/usage-rank", handler.GetModelUsageRank)
	statsGroup.Get("/platforms/usage-rank", handler.GetPlatformUsageRank)
}
