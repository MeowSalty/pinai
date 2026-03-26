package router

import (
	"log/slog"
	"net/http"

	"github.com/MeowSalty/pinai/internal/handler/control/health"
	"github.com/MeowSalty/pinai/internal/handler/control/provider"
	"github.com/MeowSalty/pinai/internal/handler/control/proxy"
	"github.com/MeowSalty/pinai/internal/handler/control/stats"
	"github.com/MeowSalty/pinai/services"
	"github.com/gin-gonic/gin"
)

// ControlConfig 定义控制面路由所需最小配置。
type ControlConfig struct {
	ApiToken     string
	AdminToken   string
	UserAgent    string
	ProxyEnabled bool
}

// SetupControlPlaneRoutes 装配控制面路由与管理接口。
func SetupControlPlaneRoutes(webAPI *gin.RouterGroup, svcs *services.Services, config ControlConfig, logger *slog.Logger) {
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

	provider.SetupProviderRoutes(webAPI, svcs.ProviderService)
	stats.SetupStatsRoutes(webAPI, svcs.StatsService, logger)
	health.SetupHealthRoutes(webAPI, svcs.HealthService, logger)
}
