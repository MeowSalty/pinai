package router

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/MeowSalty/pinai/handlers/multi"
	"github.com/MeowSalty/pinai/services"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

type Config struct {
	EnableWeb          bool
	CORSAllowAll       bool
	WebDir             string
	ApiToken           string
	AdminToken         string
	UserAgent          string
	PassthroughHeaders bool
	ProxyEnabled       bool
}

// SetupRoutes 配置 API 路由
func SetupRoutes(web *gin.Engine, svcs *services.Services, config Config, logger *slog.Logger) error {
	if config.CORSAllowAll {
		web.Use(cors.New(createAllowAllCORSConfig()))
	} else {
		web.Use(cors.Default())
	}
	webAPI := web.Group("/api")
	openaiAPI := web.Group("/openai/v1")
	anthropicAPI := web.Group("/anthropic/v1")
	multiAPI := web.Group("/multi")

	// 为业务 API 添加统计采集中间件
	openaiAPI.Use(createStatsCollectorMiddleware())
	anthropicAPI.Use(createStatsCollectorMiddleware())
	multiAPI.Use(createStatsCollectorMiddleware())

	// 如果设置了 token，为业务 API 端点添加身份验证
	if config.ApiToken != "" {
		openaiAPI.Use(createOpenAIAuthMiddleware(config.ApiToken))
		anthropicAPI.Use(createAnthropicAuthMiddleware(config.ApiToken))
	}

	// 如果设置了管理 token，为管理 API 端点添加身份验证
	if config.AdminToken != "" {
		webAPI.Use(createOpenAIAuthMiddleware(config.AdminToken))
	}

	setupControlPlaneRoutes(webAPI, svcs, config, logger)

	multi.SetupMultiRoutes(multiAPI, svcs.PortalService, config.UserAgent, config.PassthroughHeaders, logger, config.ApiToken)

	setupFrontendRoutes(web, config)

	return nil
}

// createAllowAllCORSConfig 创建宽松跨域配置。
func createAllowAllCORSConfig() cors.Config {
	return cors.Config{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
			http.MethodDelete,
			http.MethodOptions,
			http.MethodHead,
		},
		AllowHeaders: []string{
			"Origin",
			"Content-Type",
			"Accept",
			"Authorization",
			"X-Requested-With",
			"x-api-key",
			"anthropic-version",
		},
		ExposeHeaders: []string{
			"Content-Length",
			"Content-Type",
		},
		AllowCredentials: false,
		MaxAge:           24 * time.Hour,
	}
}
