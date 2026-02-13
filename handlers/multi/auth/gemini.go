package auth

import (
	"crypto/subtle"
	"strings"

	"github.com/gofiber/fiber/v2"
)

const (
	GeminiAPIKeyHeader = "x-goog-api-key"
)

// GeminiAuth validates Gemini API keys.
type GeminiAuth struct {
	Token string
}

func (a GeminiAuth) Middleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		if err := a.Validate(c); err != nil {
			return err
		}
		return c.Next()
	}
}

func (a GeminiAuth) Validate(c *fiber.Ctx) error {
	apiKey := strings.TrimSpace(c.Get(GeminiAPIKeyHeader))
	if apiKey == "" {
		apiKey = strings.TrimSpace(c.Query("key"))
	}
	if apiKey == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "缺少 x-goog-api-key 头或 key 查询参数",
		})
	}

	if subtle.ConstantTimeCompare([]byte(apiKey), []byte(a.Token)) != 1 {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "无效的 API key",
		})
	}

	return nil
}
