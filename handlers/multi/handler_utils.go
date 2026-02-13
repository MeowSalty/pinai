package multi

import (
	"bufio"
	"encoding/json"
	"fmt"

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

// setSSEHeaders 设置 SSE 响应头
func setSSEHeaders(c *fiber.Ctx) {
	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("Access-Control-Allow-Origin", "*")
}

// sendStreamError 发送流式错误响应
func sendStreamError(w *bufio.Writer, errType, message, code string) {
	errorEvent := map[string]any{
		"error": map[string]any{
			"type":    errType,
			"message": message,
			"code":    code,
		},
	}
	if jsonBytes, err := json.Marshal(errorEvent); err == nil {
		fmt.Fprintf(w, "data: %s\n\n", string(jsonBytes))
		w.Flush()
	}
}
