package router

import (
	"github.com/MeowSalty/pinai/handlers/health"
	"github.com/MeowSalty/pinai/handlers/openai"
	"github.com/MeowSalty/pinai/handlers/provider"
	"github.com/MeowSalty/pinai/handlers/stats"
	"github.com/MeowSalty/pinai/services"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
)

// SetupRoutes 配置 API 路由
func SetupRoutes(web *fiber.App, svcs *services.Services, enableWeb bool, webDir string) error {
	web.Use(cors.New())
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
