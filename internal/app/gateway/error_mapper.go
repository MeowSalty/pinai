package gateway

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

// MapDataPlaneError 对数据面错误进行第一轮统一映射。
func (s *service) MapDataPlaneError(err error, fallbackAction string) DataPlaneError {
	if err == nil {
		return defaultDataPlaneError(fallbackAction)
	}

	lowerMsg := strings.ToLower(err.Error())
	switch {
	case errors.Is(err, context.DeadlineExceeded), strings.Contains(lowerMsg, "timeout"), strings.Contains(lowerMsg, "deadline"):
		return DataPlaneError{StatusCode: http.StatusGatewayTimeout, Message: "上游请求超时"}
	case errors.Is(err, context.Canceled):
		return DataPlaneError{StatusCode: http.StatusRequestTimeout, Message: "请求已取消"}
	case strings.Contains(lowerMsg, "429"), strings.Contains(lowerMsg, "rate limit"), strings.Contains(lowerMsg, "too many requests"), strings.Contains(lowerMsg, "quota"):
		return DataPlaneError{StatusCode: http.StatusTooManyRequests, Message: "请求过于频繁，请稍后重试"}
	case strings.Contains(lowerMsg, "401"), strings.Contains(lowerMsg, "403"), strings.Contains(lowerMsg, "unauthorized"), strings.Contains(lowerMsg, "forbidden"), strings.Contains(lowerMsg, "authentication"):
		return DataPlaneError{StatusCode: http.StatusUnauthorized, Message: "鉴权失败"}
	case strings.Contains(lowerMsg, "404"), strings.Contains(lowerMsg, "not found"):
		return DataPlaneError{StatusCode: http.StatusNotFound, Message: "请求资源不存在"}
	default:
		return DataPlaneError{StatusCode: http.StatusInternalServerError, Message: fmt.Sprintf("%s：%v", fallbackAction, err)}
	}
}
