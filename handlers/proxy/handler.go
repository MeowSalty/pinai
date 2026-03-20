package proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
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
	start := time.Now()
	logger := h.newRequestLogger(c)

	auditTargetID := ""
	writeAuditLog := func(result string, statusCode int, errorType string, method string) {
		if method == http.MethodGet || method == http.MethodHead {
			return
		}

		auditLogger := logger.With(
			"target_type", "proxy_request",
			"target_id", auditTargetID,
		)

		attrs := []any{
			"result", result,
			"status_code", statusCode,
			"latency_ms", time.Since(start).Milliseconds(),
		}
		if errorType != "" {
			attrs = append(attrs, "error_type", errorType)
		}

		auditLogger.Info("代理请求审计", attrs...)
	}

	var req ProxyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Warn("请求参数校验失败",
			"error", err,
			"error_type", "validation_error",
			"content_type", c.ContentType(),
			"latency_ms", time.Since(start).Milliseconds(),
		)
		writeAuditLog("failed", http.StatusBadRequest, "validation_error", c.Request.Method)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("无效的请求格式：%v", err),
		})
		return
	}

	if strings.TrimSpace(req.URL) == "" {
		logger.Warn("请求参数校验失败",
			"error", errors.New("缺少 url 参数"),
			"error_type", "validation_error",
			"content_type", c.ContentType(),
			"latency_ms", time.Since(start).Milliseconds(),
		)
		writeAuditLog("failed", http.StatusBadRequest, "validation_error", c.Request.Method)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "缺少 url 参数",
		})
		return
	}
	auditTargetID = summarizeTargetID(req.URL)

	if err := validateURLScheme(req.URL); err != nil {
		logger.Warn("请求参数校验失败",
			"error", err,
			"error_type", "validation_error",
			"target_id", auditTargetID,
			"latency_ms", time.Since(start).Milliseconds(),
		)
		writeAuditLog("failed", http.StatusBadRequest, "validation_error", c.Request.Method)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("不允许的 URL：%v", err),
		})
		return
	}

	method := strings.ToUpper(strings.TrimSpace(req.Method))
	if method == "" {
		method = http.MethodGet
	}
	logger = logger.With("upstream_method", method)

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
		logger.Warn("创建上游请求失败",
			"error", err,
			"error_type", "validation_error",
			"target_id", auditTargetID,
			"latency_ms", time.Since(start).Milliseconds(),
		)
		writeAuditLog("failed", http.StatusBadRequest, "validation_error", method)
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

	targetHost, targetScheme := splitTarget(req.URL)
	logger = logger.With(
		"target_host", targetHost,
		"scheme", targetScheme,
		"timeout_ms", timeoutMS,
		"body_size", len(req.Body),
	)

	client := newSafeClient(time.Duration(timeoutMS) * time.Millisecond)
	upstreamResp, err := client.Do(upstreamReq)
	if err != nil {
		logger.Error("上游请求失败",
			"error", err,
			"error_type", "upstream_error",
			"latency_ms", time.Since(start).Milliseconds(),
		)
		writeAuditLog("failed", http.StatusBadGateway, "upstream_error", method)
		c.JSON(http.StatusBadGateway, gin.H{
			"error": fmt.Sprintf("上游请求失败：%v", err),
		})
		return
	}
	defer upstreamResp.Body.Close()

	limitedBody := io.LimitReader(upstreamResp.Body, maxResponseBodyBytes+1)
	bodyBytes, err := io.ReadAll(limitedBody)
	if err != nil {
		logger.Error("读取上游响应失败",
			"error", err,
			"error_type", "read_response_error",
			"status_code", upstreamResp.StatusCode,
			"latency_ms", time.Since(start).Milliseconds(),
		)
		writeAuditLog("failed", http.StatusBadGateway, "read_response_error", method)
		c.JSON(http.StatusBadGateway, gin.H{
			"error": fmt.Sprintf("读取上游响应失败：%v", err),
		})
		return
	}

	if len(bodyBytes) > maxResponseBodyBytes {
		logger.Warn("上游响应过大",
			"error_type", "response_too_large",
			"status_code", upstreamResp.StatusCode,
			"response_size", len(bodyBytes),
			"latency_ms", time.Since(start).Milliseconds(),
		)
		writeAuditLog("failed", http.StatusBadGateway, "response_too_large", method)
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
	logger.Debug("代理请求处理成功",
		"status_code", upstreamResp.StatusCode,
		"latency_ms", time.Since(start).Milliseconds(),
	)
	writeAuditLog("success", upstreamResp.StatusCode, "", method)
	c.Data(upstreamResp.StatusCode, contentType, bodyBytes)
}

func (h *Handler) newRequestLogger(c *gin.Context) *slog.Logger {
	if c == nil || c.Request == nil {
		return h.logger.With(
			"operation", "proxy",
			"path", "",
			"method", "",
			"request_id", "",
			"client_ip", "",
		)
	}

	return h.logger.With(
		"operation", "proxy",
		"path", c.Request.URL.Path,
		"method", c.Request.Method,
		"request_id", requestIDFromContext(c),
		"client_ip", c.ClientIP(),
	)
}

func requestIDFromContext(c *gin.Context) string {
	if c == nil {
		return ""
	}

	if v, ok := c.Get("request_id"); ok {
		if s, ok := v.(string); ok {
			return strings.TrimSpace(s)
		}
	}

	headers := []string{"X-Request-ID", "X-Correlation-ID", "Request-Id"}
	for _, key := range headers {
		if value := strings.TrimSpace(c.GetHeader(key)); value != "" {
			return value
		}
	}

	return ""
}

func summarizeTargetID(rawURL string) string {
	u, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return ""
	}

	if u.Scheme == "" || u.Host == "" {
		return ""
	}

	return u.Scheme + "://" + u.Host
}

func splitTarget(rawURL string) (string, string) {
	u, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return "", ""
	}

	return u.Host, u.Scheme
}

func isHopByHopHeader(key string) bool {
	_, exists := hopByHopHeaders[strings.ToLower(key)]
	return exists
}
