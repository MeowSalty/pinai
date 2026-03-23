package router

import (
	"log/slog"
	"net/http"
	"time"

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

	// 如果设置了管理 token，为管理 API 端点添加身份验证
	if config.AdminToken != "" {
		webAPI.Use(createOpenAIAuthMiddleware(config.AdminToken))
	}

	setupControlPlaneRoutes(webAPI, svcs, config, logger)
	setupDataPlaneRoutes(web, svcs, config, logger)

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
