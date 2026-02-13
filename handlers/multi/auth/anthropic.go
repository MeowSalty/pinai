package auth

import (
	"crypto/subtle"

	"github.com/gofiber/fiber/v2"
)

const (
	AnthropicVersionHeader = "anthropic-version"
	AnthropicAPIKeyHeader  = "x-api-key"
)

// AnthropicAuth validates x-api-key header.
type AnthropicAuth struct {
	Token string
}

func (a AnthropicAuth) Middleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		if err := a.Validate(c); err != nil {
			return err
		}
		return c.Next()
	}
}

func (a AnthropicAuth) Validate(c *fiber.Ctx) error {
	apiKey := c.Get(AnthropicAPIKeyHeader)
	if apiKey == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"type": "error",
			"error": fiber.Map{
				"type":    "authentication_error",
				"message": "缺少 x-api-key 头",
			},
		})
	}

	if subtle.ConstantTimeCompare([]byte(apiKey), []byte(a.Token)) != 1 {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"type": "error",
			"error": fiber.Map{
				"type":    "authentication_error",
				"message": "无效的 API key",
			},
		})
	}

	return nil
}
