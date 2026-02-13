package native

import (
	"github.com/gofiber/fiber/v2"
)

// applyUserAgent 应用 User-Agent 头部
func applyUserAgent(headers map[string]string, configuredUA string, c *fiber.Ctx) {
	if headers == nil {
		return
	}

	switch configuredUA {
	case "":
		// 透传客户端 User-Agent
		if ua := c.Get("User-Agent"); ua != "" {
			headers["User-Agent"] = ua
		}
	case "default":
		// 使用 fasthttp 默认值，不设置 headers 中的 User-Agent
	default:
		// 使用配置的 User-Agent
		headers["User-Agent"] = configuredUA
	}
}

// setSSEHeaders 设置服务器发送事件 (SSE) 的头部信息
func setSSEHeaders(c *fiber.Ctx) {
	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("Transfer-Encoding", "chunked")
}
