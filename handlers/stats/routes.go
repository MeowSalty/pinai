package stats

import (
	"github.com/gin-gonic/gin"

	"github.com/MeowSalty/pinai/services/stats"
)

// SetupStatsRoutes 配置统计相关的路由
func SetupStatsRoutes(router *gin.RouterGroup, statsService stats.Service) {
	handler := NewStatsHandler(statsService)

	statsGroup := router.Group("/stats")
	statsGroup.GET("/dashboard", handler.GetDashboard)
	statsGroup.GET("/overview", handler.GetOverview)
	statsGroup.GET("/requests", handler.ListRequestLogs)
	statsGroup.GET("/realtime", handler.GetRealtime)
	statsGroup.GET("/models/call-rank", handler.GetModelCallRank)
	statsGroup.GET("/platforms/call-rank", handler.GetPlatformCallRank)
	statsGroup.GET("/models/usage-rank", handler.GetModelUsageRank)
	statsGroup.GET("/platforms/usage-rank", handler.GetPlatformUsageRank)
}
