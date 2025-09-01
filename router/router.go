package router

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
)

// SetupRoutes 配置 API 路由
func SetupRoutes(web *fiber.App) {
	web.Use(cors.New())
	webAPI := web.Group("/api")

	webAPI.Get("/ping", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"message": "pong",
		})
	})
}
