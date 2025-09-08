package router

import (
	"github.com/MeowSalty/pinai/handlers/health"
	"github.com/MeowSalty/pinai/handlers/openai"
	"github.com/MeowSalty/pinai/handlers/provider"
	"github.com/MeowSalty/pinai/services"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
)

// SetupRoutes 配置 API 路由
func SetupRoutes(web *fiber.App, svcs *services.Services) error {
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
	return nil
}
