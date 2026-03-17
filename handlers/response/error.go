package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// ErrorResponse 统一错误响应结构。
type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

const (
	CodeBadRequest    = "bad_request"
	CodeNotFound      = "not_found"
	CodeInternalError = "internal_error"
)

// Error 输出统一错误响应。
func Error(c *gin.Context, status int, code, message string) {
	c.JSON(status, ErrorResponse{
		Code:    code,
		Message: message,
	})
}

// ErrorWithDetails 输出带 details 的统一错误响应。
func ErrorWithDetails(c *gin.Context, status int, code, message string, details any) {
	c.JSON(status, ErrorResponse{
		Code:    code,
		Message: message,
		Details: details,
	})
}

// BadRequest 输出 400 错误。
func BadRequest(c *gin.Context, message string) {
	Error(c, http.StatusBadRequest, CodeBadRequest, message)
}

// NotFound 输出 404 错误。
func NotFound(c *gin.Context, message string) {
	Error(c, http.StatusNotFound, CodeNotFound, message)
}

// InternalError 输出 500 错误。
func InternalError(c *gin.Context, message string) {
	Error(c, http.StatusInternalServerError, CodeInternalError, message)
}
