package auth

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// OpenAIAuth validates Authorization: Bearer <token> header.
type OpenAIAuth struct {
	Token string
}

func (a OpenAIAuth) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !a.Validate(c) {
			return
		}
		c.Next()
	}
}

func (a OpenAIAuth) Validate(c *gin.Context) bool {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "缺少 Authorization 头",
		})
		c.Abort()
		return false
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Authorization 头格式无效，应为：Bearer <token>",
		})
		c.Abort()
		return false
	}

	token := parts[1]
	if subtle.ConstantTimeCompare([]byte(token), []byte(a.Token)) != 1 {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "无效的 API token",
		})
		c.Abort()
		return false
	}

	return true
}
