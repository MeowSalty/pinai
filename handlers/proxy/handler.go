package proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	defaultTimeoutMS     = 30000
	maxResponseBodyBytes = 20 << 20
)

var hopByHopHeaders = map[string]struct{}{
	"connection":          {},
	"proxy-authenticate":  {},
	"proxy-authorization": {},
	"te":                  {},
	"trailer":             {},
	"transfer-encoding":   {},
	"upgrade":             {},
	"keep-alive":          {},
}

// Handler 负责处理代理请求。
type Handler struct {
	userAgent string
	logger    *slog.Logger
}

// ProxyRequest 表示 /api/proxy 的请求体。
type ProxyRequest struct {
	URL       string            `json:"url"`
	Method    string            `json:"method"`
	Headers   map[string]string `json:"headers"`
	Body      json.RawMessage   `json:"body"`
	TimeoutMS int               `json:"timeout_ms"`
}

// New 创建代理处理器实例。
func New(userAgent string, logger *slog.Logger) *Handler {
	return &Handler{
		userAgent: userAgent,
		logger:    logger,
	}
}

// Proxy 处理后端代理请求并透传上游响应。
func (h *Handler) Proxy(c *gin.Context) {
	var req ProxyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("无效的请求格式：%v", err),
		})
		return
	}

	if strings.TrimSpace(req.URL) == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "缺少 url 参数",
		})
		return
	}

	method := strings.ToUpper(strings.TrimSpace(req.Method))
	if method == "" {
		method = http.MethodGet
	}

	ctx := c.Request.Context()
	if ctx == nil {
		ctx = context.Background()
	}

	var bodyReader io.Reader
	if len(req.Body) > 0 {
		bodyReader = bytes.NewReader(req.Body)
	}

	upstreamReq, err := http.NewRequestWithContext(ctx, method, req.URL, bodyReader)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("创建上游请求失败：%v", err),
		})
		return
	}

	for key, value := range req.Headers {
		if strings.TrimSpace(key) == "" {
			continue
		}
		upstreamReq.Header.Set(key, value)
	}

	if len(req.Body) > 0 && upstreamReq.Header.Get("Content-Type") == "" {
		upstreamReq.Header.Set("Content-Type", "application/json")
	}

	if upstreamReq.Header.Get("User-Agent") == "" && h.userAgent != "" {
		upstreamReq.Header.Set("User-Agent", h.userAgent)
	}

	timeoutMS := req.TimeoutMS
	if timeoutMS <= 0 {
		timeoutMS = defaultTimeoutMS
	}

	logger := h.logger.With("method", method, "url", req.URL)
	if method != http.MethodGet && method != http.MethodHead {
		logger.Info("代理请求审计")
	}

	client := &http.Client{Timeout: time.Duration(timeoutMS) * time.Millisecond}
	upstreamResp, err := client.Do(upstreamReq)
	if err != nil {
		logger.Error("上游请求失败", slog.Any("error", err))
		c.JSON(http.StatusBadGateway, gin.H{
			"error": fmt.Sprintf("上游请求失败：%v", err),
		})
		return
	}
	defer upstreamResp.Body.Close()

	limitedBody := io.LimitReader(upstreamResp.Body, maxResponseBodyBytes+1)
	bodyBytes, err := io.ReadAll(limitedBody)
	if err != nil {
		logger.Error("读取上游响应失败", slog.Any("error", err))
		c.JSON(http.StatusBadGateway, gin.H{
			"error": fmt.Sprintf("读取上游响应失败：%v", err),
		})
		return
	}

	if len(bodyBytes) > maxResponseBodyBytes {
		logger.Warn("上游响应过大", slog.Int("size", len(bodyBytes)))
		c.JSON(http.StatusBadGateway, gin.H{
			"error": "上游响应过大，已超过限制",
		})
		return
	}

	for key, values := range upstreamResp.Header {
		if isHopByHopHeader(key) {
			continue
		}
		for _, value := range values {
			c.Writer.Header().Add(key, value)
		}
	}

	contentType := upstreamResp.Header.Get("Content-Type")
	c.Data(upstreamResp.StatusCode, contentType, bodyBytes)
}

func isHopByHopHeader(key string) bool {
	_, exists := hopByHopHeaders[strings.ToLower(key)]
	return exists
}
