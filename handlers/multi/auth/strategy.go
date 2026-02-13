package auth

import "github.com/gofiber/fiber/v2"

// Strategy defines a unified authentication interface.
type Strategy interface {
	Middleware() fiber.Handler
	Validate(c *fiber.Ctx) error
}
