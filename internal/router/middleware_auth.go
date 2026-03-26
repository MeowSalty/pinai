package router

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// createOpenAIAuthMiddleware 创建 OpenAI API 身份验证中间件
func createOpenAIAuthMiddleware(validToken string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取 Authorization 头
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "缺少 Authorization 头",
			})
			c.Abort()
			return
		}

		// 验证 Bearer token 格式
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Authorization 头格式无效，应为：Bearer <token>",
			})
			c.Abort()
			return
		}

		// 验证 token
		token := parts[1]
		if subtle.ConstantTimeCompare([]byte(token), []byte(validToken)) != 1 {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "无效的 API token",
			})
			c.Abort()
			return
		}

		// token 验证通过，继续处理请求
		c.Next()
	}
}

// createAnthropicAuthMiddleware 创建 Anthropic API 身份验证中间件
func createAnthropicAuthMiddleware(validToken string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取 x-api-key 头
		apiKey := c.GetHeader("x-api-key")
		if apiKey == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"type": "error",
				"error": gin.H{
					"type":    "authentication_error",
					"message": "缺少 x-api-key 头",
				},
			})
			c.Abort()
			return
		}

		// 验证 API key
		if subtle.ConstantTimeCompare([]byte(apiKey), []byte(validToken)) != 1 {
			c.JSON(http.StatusUnauthorized, gin.H{
				"type": "error",
				"error": gin.H{
					"type":    "authentication_error",
					"message": "无效的 API key",
				},
			})
			c.Abort()
			return
		}

		// API key 验证通过，继续处理请求
		c.Next()
	}
}
