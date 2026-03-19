package health

import (
	"log/slog"

	"github.com/gin-gonic/gin"

	"github.com/MeowSalty/pinai/services/health"
)

// SetupHealthRoutes 配置健康状态统计相关的路由
func SetupHealthRoutes(router *gin.RouterGroup, healthService health.Service, logger *slog.Logger) {
	if logger == nil {
		logger = slog.Default()
	}

	handler := NewHandler(healthService, logger.WithGroup("health_handler"))

	healthGroup := router.Group("/health")
	healthGroup.GET("/summary", handler.GetHealthSummary)

	// 异常资源端点
	healthGroup.GET("/issues", handler.GetIssues)

	// 平台健康端点
	healthGroup.GET("/platforms", handler.GetPlatformHealthList)

	// 密钥健康端点
	healthGroup.GET("/keys", handler.GetAPIKeyHealthList)

	// 模型健康端点
	healthGroup.GET("/models", handler.GetModelHealthList)
}
