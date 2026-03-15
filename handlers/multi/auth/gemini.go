package auth

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	GeminiAPIKeyHeader = "x-goog-api-key"
)

// GeminiAuth validates Gemini API keys.
type GeminiAuth struct {
	Token string
}

func (a GeminiAuth) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !a.Validate(c) {
			return
		}
		c.Next()
	}
}

func (a GeminiAuth) Validate(c *gin.Context) bool {
	apiKey := strings.TrimSpace(c.GetHeader(GeminiAPIKeyHeader))
	if apiKey == "" {
		apiKey = strings.TrimSpace(c.Query("key"))
	}
	if apiKey == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "缺少 x-goog-api-key 头或 key 查询参数",
		})
		c.Abort()
		return false
	}

	if subtle.ConstantTimeCompare([]byte(apiKey), []byte(a.Token)) != 1 {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "无效的 API key",
		})
		c.Abort()
		return false
	}

	return true
}
