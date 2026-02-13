package auth

import (
	"crypto/subtle"
	"strings"

	"github.com/gofiber/fiber/v2"
)

// OpenAIAuth validates Authorization: Bearer <token> header.
type OpenAIAuth struct {
	Token string
}

func (a OpenAIAuth) Middleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		if err := a.Validate(c); err != nil {
			return err
		}
		return c.Next()
	}
}

func (a OpenAIAuth) Validate(c *fiber.Ctx) error {
	authHeader := c.Get("Authorization")
	if authHeader == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "缺少 Authorization 头",
		})
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Authorization 头格式无效，应为：Bearer <token>",
		})
	}

	token := parts[1]
	if subtle.ConstantTimeCompare([]byte(token), []byte(a.Token)) != 1 {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "无效的 API token",
		})
	}

	return nil
}
