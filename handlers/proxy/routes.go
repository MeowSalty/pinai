package proxy

import (
	"log/slog"

	"github.com/gofiber/fiber/v2"
)

// SetupProxyRoutes 配置代理相关路由。
func SetupProxyRoutes(router fiber.Router, apiToken string, userAgent string, logger *slog.Logger) {
	_ = apiToken
	if logger == nil {
		logger = slog.Default()
	}

	handler := New(userAgent, logger.WithGroup("proxy"))
	router.Post("", handler.Proxy)
}
