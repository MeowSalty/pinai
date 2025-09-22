package router

import (
	"strings"

	"github.com/MeowSalty/pinai/handlers/health"
	"github.com/MeowSalty/pinai/handlers/openai"
	"github.com/MeowSalty/pinai/handlers/provider"
	"github.com/MeowSalty/pinai/handlers/stats"
	"github.com/MeowSalty/pinai/services"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
)

// SetupRoutes 配置 API 路由
func SetupRoutes(web *fiber.App, svcs *services.Services, enableWeb bool, webDir string, apiToken string) error {
	web.Use(cors.New())
	// 如果设置了 token，为 OpenAI 端点添加身份验证
	if apiToken != "" {
		web.Use(createOpenAIAuthMiddleware(apiToken))
	}
	webAPI := web.Group("/api")
	openaiAPI := web.Group("/openai/v1")

	webAPI.Get("/ping", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"message": "pong",
		})
	})

	health.SetupHealthRoutes(webAPI, svcs.HealthService)
	openai.SetupOpenAIRoutes(openaiAPI, svcs.AIGatewayService)
	provider.SetupProviderRoutes(webAPI, svcs.ProviderService)
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
		if token != validToken {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "无效的 API token",
			})
		}

		// token 验证通过，继续处理请求
		return c.Next()
	}
}
