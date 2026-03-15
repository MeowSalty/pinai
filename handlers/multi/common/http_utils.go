package common

import (
	"strings"

	"github.com/gin-gonic/gin"
)

// skipHeaders 是需要跳过的 HTTP 头部集合（小写）
var skipHeaders = map[string]struct{}{
	"connection": {}, "keep-alive": {}, "upgrade": {},
	"proxy-connection":  {},
	"transfer-encoding": {}, "te": {}, "trailer": {},
	"authorization": {}, "x-api-key": {}, "api-key": {},
	"content-type": {}, "content-length": {}, "accept": {},
	"accept-encoding": {}, "host": {}, "user-agent": {},
	"x-forwarded-for": {}, "x-real-ip": {}, "x-forwarded-host": {},
}

// ApplyHTTPHeaders 将 HTTP 请求头透传到 req.Headers。
//
// 从 HTTP 请求头中提取需要透传的头部写入。
func ApplyHTTPHeaders(headers map[string]string, configuredUA string, passthrough bool, c *gin.Context) {
	if headers == nil {
		return
	}

	// 1. 透传 HTTP 请求头
	if passthrough {
		for key, values := range c.Request.Header {
			k := strings.ToLower(key)
			if _, skip := skipHeaders[k]; skip {
				continue
			}
			if len(values) > 0 {
				headers[key] = values[0]
			}
		}
	}

	// 2. 处理 User-Agent（保持现有逻辑）
	switch configuredUA {
	case "":
		if ua := c.GetHeader("User-Agent"); ua != "" {
			headers["User-Agent"] = ua
		}
	case "default":
		// 使用 Go net/http 默认值，不设置 headers 中的 User-Agent
	default:
		headers["User-Agent"] = configuredUA
	}
}

// SetBaseSSEHeaders 设置两种模式都通用的 SSE 响应头。
func SetBaseSSEHeaders(c *gin.Context) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")
}
