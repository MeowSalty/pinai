package health

import (
	"github.com/gin-gonic/gin"

	"github.com/MeowSalty/pinai/services/health"
)

// SetupHealthRoutes 配置健康状态统计相关的路由
func SetupHealthRoutes(router *gin.RouterGroup, healthService health.Service) {
	handler := NewHandler(healthService)

	healthGroup := router.Group("/health")
	healthGroup.GET("/summary", handler.GetHealthSummary)

	// 异常资源端点
	healthGroup.GET("/issues", handler.GetIssues)

	// 平台健康端点
	healthGroup.GET("/platforms", handler.GetPlatformHealthList)
	healthGroup.POST("/platforms/:platformId/enable", handler.EnablePlatform)
	healthGroup.POST("/platforms/:platformId/disable", handler.DisablePlatform)

	// 密钥健康端点
	healthGroup.GET("/keys", handler.GetAPIKeyHealthList)
	healthGroup.POST("/keys/:keyId/enable", handler.EnableAPIKey)
	healthGroup.POST("/keys/:keyId/disable", handler.DisableAPIKey)

	// 模型健康端点
	healthGroup.GET("/models", handler.GetModelHealthList)
	healthGroup.POST("/models/:modelId/enable", handler.EnableModel)
	healthGroup.POST("/models/:modelId/disable", handler.DisableModel)
}
