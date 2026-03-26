package router

import (
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// setupCORS 按配置注册跨域中间件。
func setupCORS(web *gin.Engine, config Config) {
	if config.CORSAllowAll {
		web.Use(cors.New(createAllowAllCORSConfig()))
		return
	}

	web.Use(cors.Default())
}

// setupAPIRootGroup 创建 API 根分组并按需注册管理鉴权。
func setupAPIRootGroup(web *gin.Engine, config Config) *gin.RouterGroup {
	webAPI := web.Group("/api")

	// 如果设置了管理 token，为管理 API 端点添加身份验证。
	if config.AdminToken != "" {
		webAPI.Use(createOpenAIAuthMiddleware(config.AdminToken))
	}

	return webAPI
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
