// Deprecated: 不再维护，逐步迁移到 github.com/MeowSalty/pinai/handlers/multi 包。
package anthropic

import (
	"github.com/MeowSalty/pinai/services/portal"
	"github.com/gin-gonic/gin"
)

// SetupAnthropicRoutes 注册 Anthropic 兼容路由。
func SetupAnthropicRoutes(router *gin.RouterGroup, portalService portal.Service, userAgent string) {
	// 创建 Handler 实例，传入 userAgent 配置
	handler := New(portalService, userAgent)

	router.GET("/models", ListModels)
	router.POST("/messages", handler.Messages)
}
