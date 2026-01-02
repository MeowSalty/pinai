package router

import (
	"crypto/subtle"
	"strings"

	"github.com/MeowSalty/pinai/handlers/anthropic"
	"github.com/MeowSalty/pinai/handlers/openai"
	"github.com/MeowSalty/pinai/handlers/provider"
	"github.com/MeowSalty/pinai/handlers/stats"
	"github.com/MeowSalty/pinai/services"
	statsService "github.com/MeowSalty/pinai/services/stats"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
)

// SetupRoutes 配置 API 路由
func SetupRoutes(web *fiber.App, svcs *services.Services, enableWeb bool, webDir string, apiToken string, adminToken string, userAgent string) error {
	web.Use(cors.New())
	webAPI := web.Group("/api")
	openaiAPI := web.Group("/openai/v1")
	anthropicAPI := web.Group("/anthropic/v1")

	// 为业务 API 添加统计采集中间件
	openaiAPI.Use(createStatsCollectorMiddleware())
	anthropicAPI.Use(createStatsCollectorMiddleware())

	// 如果设置了 token，为业务 API 端点添加身份验证
	if apiToken != "" {
		openaiAPI.Use(createOpenAIAuthMiddleware(apiToken))
		anthropicAPI.Use(createAnthropicAuthMiddleware(apiToken))
	}

	// 如果设置了管理 token，为管理 API 端点添加身份验证
	if adminToken != "" {
		webAPI.Use(createOpenAIAuthMiddleware(adminToken))
	}

	webAPI.Get("/ping", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"message": "pong",
		})
	})

	openai.SetupOpenAIRoutes(openaiAPI, svcs.PortalService, userAgent)
	anthropic.SetupAnthropicRoutes(anthropicAPI, svcs.PortalService, userAgent)
	provider.SetupProviderRoutes(webAPI, svcs.ProviderService, svcs.HealthService)
	stats.SetupStatsRoutes(webAPI, svcs.StatsService)

	// 如果启用了前端支持，则设置前端路由
	if enableWeb {
		// 静态文件服务
		web.Static("/", webDir)
		// 添加一个兜底路由，将未匹配的路径都指向 index.html 以支持 SPA
		web.Get("*", func(c *fiber.Ctx) error {
			return c.SendFile(webDir + "/index.html")
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
