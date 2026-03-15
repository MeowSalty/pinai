package auth

import (
	"crypto/subtle"
	"net/http"

	"github.com/gin-gonic/gin"
)

const (
	AnthropicVersionHeader = "anthropic-version"
	AnthropicAPIKeyHeader  = "x-api-key"
)

// AnthropicAuth validates x-api-key header.
type AnthropicAuth struct {
	Token string
}

func (a AnthropicAuth) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !a.Validate(c) {
			return
		}
		c.Next()
	}
}

func (a AnthropicAuth) Validate(c *gin.Context) bool {
	apiKey := c.GetHeader(AnthropicAPIKeyHeader)
	if apiKey == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"type": "error",
			"error": gin.H{
				"type":    "authentication_error",
				"message": "缺少 x-api-key 头",
			},
		})
		c.Abort()
		return false
	}

	if subtle.ConstantTimeCompare([]byte(apiKey), []byte(a.Token)) != 1 {
		c.JSON(http.StatusUnauthorized, gin.H{
			"type": "error",
			"error": gin.H{
				"type":    "authentication_error",
				"message": "无效的 API key",
			},
		})
		c.Abort()
		return false
	}

	return true
}
