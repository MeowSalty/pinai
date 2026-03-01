package router

import (
	"crypto/subtle"
	"log/slog"
	"strings"

	"github.com/MeowSalty/pinai/handlers/anthropic"
	"github.com/MeowSalty/pinai/handlers/health"
	"github.com/MeowSalty/pinai/handlers/multi"
	"github.com/MeowSalty/pinai/handlers/openai"
	"github.com/MeowSalty/pinai/handlers/provider"
	"github.com/MeowSalty/pinai/handlers/proxy"
	"github.com/MeowSalty/pinai/handlers/stats"
	"github.com/MeowSalty/pinai/services"
	statsService "github.com/MeowSalty/pinai/services/stats"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
)

type Config struct {
	EnableWeb          bool
	WebDir             string
	ApiToken           string
	AdminToken         string
	UserAgent          string
	PassthroughHeaders bool
}

// SetupRoutes 配置 API 路由
func SetupRoutes(web *fiber.App, svcs *services.Services, config Config, logger *slog.Logger) error {
	web.Use(cors.New())
	webAPI := web.Group("/api")
	proxyAPI := webAPI.Group("/proxy")
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
		proxyAPI.Use(createOpenAIAuthMiddleware(config.AdminToken))
	}

	webAPI.Get("/ping", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"message": "pong",
		})
	})

	openai.SetupOpenAIRoutes(openaiAPI, svcs.PortalService, config.UserAgent, logger)
	anthropic.SetupAnthropicRoutes(anthropicAPI, svcs.PortalService, config.UserAgent)

	multi.SetupMultiRoutes(multiAPI, svcs.PortalService, config.UserAgent, config.PassthroughHeaders, logger, config.ApiToken)

	proxy.SetupProxyRoutes(proxyAPI, config.ApiToken, config.UserAgent, logger)
	provider.SetupProviderRoutes(webAPI, svcs.ProviderService, svcs.HealthService)
	stats.SetupStatsRoutes(webAPI, svcs.StatsService)
	health.SetupHealthRoutes(webAPI, svcs.HealthService)

	// 如果启用了前端支持，则设置前端路由
	if config.EnableWeb {
		// 静态文件服务
		web.Static("/", config.WebDir)
		// 添加一个兜底路由，将未匹配的路径都指向 index.html 以支持 SPA
		web.Get("*", func(c *fiber.Ctx) error {
			return c.SendFile(config.WebDir + "/index.html")
		})
	}

	return nil
}

// createOpenAIAuthMiddleware 创建 OpenAI API 身份验证中间件
func createOpenAIAuthMiddleware(validToken string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// 获取 Authorization 头
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "缺少 Authorization 头",
			})
		}

		// 验证 Bearer token 格式
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Authorization 头格式无效，应为：Bearer <token>",
			})
		}

		// 验证 token
		token := parts[1]
		if subtle.ConstantTimeCompare([]byte(token), []byte(validToken)) != 1 {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "无效的 API token",
			})
		}

		// token 验证通过，继续处理请求
		return c.Next()
	}
}

// createAnthropicAuthMiddleware 创建 Anthropic API 身份验证中间件
func createAnthropicAuthMiddleware(validToken string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// 获取 x-api-key 头
		apiKey := c.Get("x-api-key")
		if apiKey == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"type": "error",
				"error": fiber.Map{
					"type":    "authentication_error",
					"message": "缺少 x-api-key 头",
				},
			})
		}

		// 验证 API key
		if subtle.ConstantTimeCompare([]byte(apiKey), []byte(validToken)) != 1 {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"type": "error",
				"error": fiber.Map{
					"type":    "authentication_error",
					"message": "无效的 API key",
				},
			})
		}

		// API key 验证通过，继续处理请求
		return c.Next()
	}
}

// createStatsCollectorMiddleware 创建统计数据采集中间件
//
// 该中间件用于采集业务接口的请求数据和活动连接数
//
// 注意：
//   - 对于非流式响应，在请求完成后自动减少连接数
//   - 对于流式响应（SSE），连接数由流式处理器在流结束时减少，以确保统计准确性
func createStatsCollectorMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		collector := statsService.GetCollector()

		// 记录请求
		collector.RecordRequest()

		// 增加活动连接数
		collector.IncrementConnection()

		// 对于非流式响应，请求完成后减少活动连接数
		// 流式响应会在 SetBodyStreamWriter 的 WithStreamTracking 包装器中处理
		defer func() {
			// 检查是否为流式响应（通过响应头判断）
			contentType := string(c.Response().Header.Peek("Content-Type"))
			if contentType != "text/event-stream" {
				// 非流式响应，在这里减少连接数
				collector.DecrementConnection()
			}
			// 流式响应的连接数将在流结束时由 WithStreamTracking 减少
		}()

		return c.Next()
	}
}
