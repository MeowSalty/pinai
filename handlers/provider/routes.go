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
	router.POST("/platforms", handler.CreatePlatform)
	router.GET("/platforms", handler.GetPlatforms)
	router.GET("/platforms/:id", handler.GetPlatform)
	router.PUT("/platforms/:id", handler.UpdatePlatform)
	router.DELETE("/platforms/:id", handler.DeletePlatform)

	// 平台健康状态管理路由
	router.POST("/platforms/:id/health/enable", handler.EnablePlatformHealth)
	router.POST("/platforms/:id/health/disable", handler.DisablePlatformHealth)

	// 模型 (Models) 相关路由 (嵌套在平台下)
	router.POST("/platforms/:platformId/models", handler.AddModelToPlatform)
	router.POST("/platforms/:platformId/models/batch", handler.BatchAddModelsToPlatform)
	router.GET("/platforms/:platformId/models", handler.GetModelsByPlatform)
	router.PUT("/platforms/:platformId/models/batch", handler.BatchUpdateModels)
	router.PUT("/platforms/:platformId/models/:modelId", handler.UpdateModel)
	router.DELETE("/platforms/:platformId/models/batch", handler.BatchDeleteModels)
	router.DELETE("/platforms/:platformId/models/:modelId", handler.DeleteModel)

	// 模型健康状态管理路由
	router.POST("/platforms/:platformId/models/:modelId/health/enable", handler.EnableModelHealth)
	router.POST("/platforms/:platformId/models/:modelId/health/disable", handler.DisableModelHealth)

	// 密钥 (Keys) 相关路由 (嵌套在平台下)
	router.POST("/platforms/:platformId/keys", handler.AddKeyToPlatform)
	router.GET("/platforms/:platformId/keys", handler.GetKeysByPlatform)
	router.PUT("/platforms/:platformId/keys/:keyId", handler.UpdateKey)
	router.DELETE("/platforms/:platformId/keys/:keyId", handler.DeleteKey)

	// 密钥健康状态管理路由
	router.POST("/platforms/:platformId/keys/:keyId/health/enable", handler.EnableKeyHealth)
	router.POST("/platforms/:platformId/keys/:keyId/health/disable", handler.DisableKeyHealth)

	// 端点 (Endpoints) 相关路由 (嵌套在平台下)
	router.POST("/platforms/:platformId/endpoints", handler.AddEndpointToPlatform)
	router.POST("/platforms/:platformId/endpoints/batch", handler.BatchAddEndpointsToPlatform)
	router.GET("/platforms/:platformId/endpoints", handler.GetEndpointsByPlatform)
	router.GET("/platforms/:platformId/endpoints/:endpointId", handler.GetEndpoint)
	router.PUT("/platforms/:platformId/endpoints/batch", handler.BatchUpdateEndpoints)
	router.PUT("/platforms/:platformId/endpoints/:endpointId", handler.UpdateEndpoint)
	router.DELETE("/platforms/:platformId/endpoints/:endpointId", handler.DeleteEndpoint)
}
