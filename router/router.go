package router

import (
	"crypto/subtle"
	"log/slog"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/MeowSalty/pinai/handlers/health"
	"github.com/MeowSalty/pinai/handlers/multi"
	"github.com/MeowSalty/pinai/handlers/provider"
	"github.com/MeowSalty/pinai/handlers/proxy"
	"github.com/MeowSalty/pinai/handlers/stats"
	"github.com/MeowSalty/pinai/services"
	statsService "github.com/MeowSalty/pinai/services/stats"
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

	multi.SetupMultiRoutes(multiAPI, svcs.PortalService, config.UserAgent, config.PassthroughHeaders, logger, config.ApiToken)

	provider.SetupProviderRoutes(webAPI, svcs.ProviderService, svcs.HealthService, svcs.HealthStorage)
	stats.SetupStatsRoutes(webAPI, svcs.StatsService, logger)
	health.SetupHealthRoutes(webAPI, svcs.HealthService, logger)

	// 如果启用了前端支持，则设置前端路由
	if config.EnableWeb {
		indexFilePath := filepath.Join(config.WebDir, "index.html")

		// 根路径返回 index.html
		web.GET("/", func(c *gin.Context) {
			c.File(indexFilePath)
		})

		// 添加兜底路由：优先返回静态文件，不存在时回退 index.html 以支持 SPA
		web.NoRoute(func(c *gin.Context) {
			requestPath := normalizeRequestPath(c.Request.URL.Path)

			// 保留 API 路径的 404 行为，避免被前端兜底路由吞掉
			if isReservedAPIPath(requestPath) {
				c.JSON(http.StatusNotFound, gin.H{
					"error": "接口不存在",
				})
				return
			}

			// 仅对浏览器常见的 GET/HEAD 请求做前端路由回退
			if c.Request.Method != http.MethodGet && c.Request.Method != http.MethodHead {
				c.Status(http.StatusNotFound)
				return
			}

			if requestPath != "/" {
				if filePath, ok := resolveFrontendFilePath(config.WebDir, requestPath); ok {
					fileInfo, err := os.Stat(filePath)
					if err == nil && !fileInfo.IsDir() {
						c.File(filePath)
						return
					}
				}
			}

			c.File(indexFilePath)
		})
	}

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

// normalizeRequestPath 规范化请求路径，确保结果始终以 / 开头。
func normalizeRequestPath(rawPath string) string {
	cleanedPath := path.Clean(rawPath)
	if cleanedPath == "." {
		return "/"
	}

	if !strings.HasPrefix(cleanedPath, "/") {
		return "/" + cleanedPath
	}

	return cleanedPath
}

// resolveFrontendFilePath 根据请求路径解析前端静态文件绝对路径。
func resolveFrontendFilePath(webDir, requestPath string) (string, bool) {
	if requestPath == "/" {
		return "", false
	}

	relativePath := strings.TrimPrefix(requestPath, "/")
	resolvedPath := filepath.Join(webDir, filepath.FromSlash(relativePath))

	if !isPathWithinBase(webDir, resolvedPath) {
		return "", false
	}

	return resolvedPath, true
}

// isPathWithinBase 判断目标路径是否在基础目录内，避免路径穿越。
func isPathWithinBase(basePath, targetPath string) bool {
	baseAbsPath, err := filepath.Abs(basePath)
	if err != nil {
		return false
	}

	targetAbsPath, err := filepath.Abs(targetPath)
	if err != nil {
		return false
	}

	relativePath, err := filepath.Rel(baseAbsPath, targetAbsPath)
	if err != nil {
		return false
	}

	if relativePath == "." {
		return true
	}

	return relativePath != ".." && !strings.HasPrefix(relativePath, ".."+string(filepath.Separator))
}

// isReservedAPIPath 判断路径是否属于后端 API 前缀。
func isReservedAPIPath(requestPath string) bool {
	reservedPrefixes := []string{"/api", "/openai/v1", "/anthropic/v1", "/multi"}
	for _, prefix := range reservedPrefixes {
		if requestPath == prefix || strings.HasPrefix(requestPath, prefix+"/") {
			return true
		}
	}

	return false
}

// createOpenAIAuthMiddleware 创建 OpenAI API 身份验证中间件
func createOpenAIAuthMiddleware(validToken string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取 Authorization 头
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "缺少 Authorization 头",
			})
			c.Abort()
			return
		}

		// 验证 Bearer token 格式
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Authorization 头格式无效，应为：Bearer <token>",
			})
			c.Abort()
			return
		}

		// 验证 token
		token := parts[1]
		if subtle.ConstantTimeCompare([]byte(token), []byte(validToken)) != 1 {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "无效的 API token",
			})
			c.Abort()
			return
		}

		// token 验证通过，继续处理请求
		c.Next()
	}
}

// createAnthropicAuthMiddleware 创建 Anthropic API 身份验证中间件
func createAnthropicAuthMiddleware(validToken string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取 x-api-key 头
		apiKey := c.GetHeader("x-api-key")
		if apiKey == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"type": "error",
				"error": gin.H{
					"type":    "authentication_error",
					"message": "缺少 x-api-key 头",
				},
			})
			c.Abort()
			return
		}

		// 验证 API key
		if subtle.ConstantTimeCompare([]byte(apiKey), []byte(validToken)) != 1 {
			c.JSON(http.StatusUnauthorized, gin.H{
				"type": "error",
				"error": gin.H{
					"type":    "authentication_error",
					"message": "无效的 API key",
				},
			})
			c.Abort()
			return
		}

		// API key 验证通过，继续处理请求
		c.Next()
	}
}

// createStatsCollectorMiddleware 创建统计数据采集中间件
//
// 该中间件用于采集业务接口的请求数据和活动连接数
//
// 注意：
//   - 对于非流式响应，在请求完成后自动减少连接数
//   - 对于流式响应（SSE），连接数由流式处理器在流结束时通过 defer collector.DecrementConnection() 减少
func createStatsCollectorMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		collector := statsService.GetCollector()

		// 记录请求
		collector.RecordRequest()

		// 增加活动连接数
		collector.IncrementConnection()

		// 对于非流式响应，请求完成后减少活动连接数
		// 流式响应会在流式 handler 中通过 defer collector.DecrementConnection() 处理
		defer func() {
			// 检查是否为流式响应（通过响应头判断）
			contentType := c.Writer.Header().Get("Content-Type")
			if contentType != "text/event-stream" {
				// 非流式响应，在这里减少连接数
				collector.DecrementConnection()
			}
			// 流式响应的连接数将在流结束时由 handler 中的 defer 减少
		}()

		c.Next()
	}
}
