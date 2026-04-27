package stats

import (
	"log/slog"

	"github.com/gin-gonic/gin"

	"github.com/MeowSalty/pinai/internal/app/stats"
)

// SetupStatsRoutes 配置统计相关的路由
func SetupStatsRoutes(router *gin.RouterGroup, statsService stats.Service, logger *slog.Logger) {
	if logger == nil {
		logger = slog.Default()
	}

	handler := NewStatsHandler(statsService, logger.WithGroup("handlers"))

	statsGroup := router.Group("/stats")
	statsGroup.GET("/dashboard", handler.GetDashboard)
	statsGroup.GET("/model-status", handler.GetModelStatus)
	statsGroup.GET("/requests", handler.ListRequestLogs)
	statsGroup.GET("/realtime", handler.GetRealtime)
}
