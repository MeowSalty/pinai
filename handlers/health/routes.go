package health

import (
	"github.com/gofiber/fiber/v2"

	"github.com/MeowSalty/pinai/services"
)

func SetupHealthRoutes(router fiber.Router) {
	handler := NewHealthHandler(services.NewHealthService())

	healthGroup := router.Group("/health")
	healthGroup.Get("/:resourceType/:id", handler.GetResourceHealth)
	healthGroup.Get("/platforms", handler.GetPlatformsHealthOverview)
	healthGroup.Get("/models", handler.GetModelsHealthOverview)
	healthGroup.Get("/platforms/:id/resources", handler.GetPlatformResourcesHealth)
}