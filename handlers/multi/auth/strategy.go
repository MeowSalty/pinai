package auth

import "github.com/gin-gonic/gin"

// Strategy defines a unified authentication interface.
type Strategy interface {
	Middleware() gin.HandlerFunc
	Validate(c *gin.Context) bool // false = 已写入错误响应并 Abort
}
