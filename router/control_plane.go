package router

import (
	"log/slog"
	"net/http"

	"github.com/MeowSalty/pinai/handlers/health"
	"github.com/MeowSalty/pinai/handlers/provider"
	"github.com/MeowSalty/pinai/handlers/proxy"
	"github.com/MeowSalty/pinai/handlers/stats"
	"github.com/MeowSalty/pinai/services"
	"github.com/gin-gonic/gin"
)

// setupControlPlaneRoutes 装配控制面路由与管理接口。
func setupControlPlaneRoutes(webAPI *gin.RouterGroup, svcs *services.Services, config Config, logger *slog.Logger) {
	// 条件注册代理路由（需 ProxyEnabled=true 且 AdminToken 非空）
	if config.ProxyEnabled && config.AdminToken != "" {
		proxyAPI := webAPI.Group("/proxy")
		proxy.SetupProxyRoutes(proxyAPI, config.ApiToken, config.UserAgent, logger)
	}

	webAPI.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})

	provider.SetupProviderRoutes(webAPI, svcs.ProviderService, svcs.HealthService, svcs.HealthStorage)
	stats.SetupStatsRoutes(webAPI, svcs.StatsService, logger)
	health.SetupHealthRoutes(webAPI, svcs.HealthService, logger)
}
