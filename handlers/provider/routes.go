package provider

import (
	"github.com/MeowSalty/pinai/services/health"
	"github.com/MeowSalty/pinai/services/provider"

	"github.com/gin-gonic/gin"
)

// SetupProviderRoutes 配置 LLM 供应方管理相关的 API 路由
func SetupProviderRoutes(router *gin.RouterGroup, llmService provider.Service, healthService health.Service) {
	// 创建单一的 Provider Handler 实例
	handler := NewHandler(llmService, healthService)

	// 平台 (Platforms) 相关路由
	platforms := router.Group("/platforms")
	platforms.POST("", handler.CreatePlatform)
	platforms.GET("", handler.GetPlatforms)

	platform := platforms.Group("/:platformId")
	platform.GET("", handler.GetPlatform)
	platform.PUT("", handler.UpdatePlatform)
	platform.DELETE("", handler.DeletePlatform)

	// 平台健康状态管理路由
	platform.PATCH("/health", handler.UpdatePlatformHealth)

	// 模型 (Models) 相关路由 (嵌套在平台下)
	models := platform.Group("/models")
	models.POST("", handler.AddModelToPlatform)
	models.POST("/batch", handler.BatchAddModelsToPlatform)
	models.GET("", handler.GetModelsByPlatform)
	models.PUT("/batch", handler.BatchUpdateModels)
	models.PUT("/:modelId", handler.UpdateModel)
	models.DELETE("/batch", handler.BatchDeleteModels)
	models.DELETE("/:modelId", handler.DeleteModel)

	// 模型健康状态管理路由
	models.PATCH("/:modelId/health", handler.UpdateModelHealth)

	// 密钥 (Keys) 相关路由 (嵌套在平台下)
	keys := platform.Group("/keys")
	keys.POST("", handler.AddKeyToPlatform)
	keys.GET("", handler.GetKeysByPlatform)
	keys.PUT("/:keyId", handler.UpdateKey)
	keys.DELETE("/:keyId", handler.DeleteKey)

	// 密钥健康状态管理路由
	keys.PATCH("/:keyId/health", handler.UpdateKeyHealth)

	// 端点 (Endpoints) 相关路由 (嵌套在平台下)
	endpoints := platform.Group("/endpoints")
	endpoints.POST("", handler.AddEndpointToPlatform)
	endpoints.POST("/batch", handler.BatchAddEndpointsToPlatform)
	endpoints.GET("", handler.GetEndpointsByPlatform)
	endpoints.GET("/:endpointId", handler.GetEndpoint)
	endpoints.PUT("/batch", handler.BatchUpdateEndpoints)
	endpoints.PUT("/:endpointId", handler.UpdateEndpoint)
	endpoints.DELETE("/:endpointId", handler.DeleteEndpoint)
}
