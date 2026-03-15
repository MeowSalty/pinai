package proxy

import (
	"log/slog"

	"github.com/gin-gonic/gin"
)

// SetupProxyRoutes 配置代理相关路由。
func SetupProxyRoutes(router *gin.RouterGroup, apiToken string, userAgent string, logger *slog.Logger) {
	_ = apiToken
	if logger == nil {
		logger = slog.Default()
	}

	handler := New(userAgent, logger.WithGroup("proxy"))
	router.POST("", handler.Proxy)
}
