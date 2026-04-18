package common

import (
	"context"
	"log/slog"
	"strings"

	"github.com/gin-gonic/gin"
)

// RequestLogContext 承载数据面请求的标准化日志上下文字段。
//
// 该结构体统一了从 Handler 到 Gateway 的日志关联字段，
// 通过 context.Context 透传，避免在各层重复手动拼接日志属性。
// 后续子任务可逐步将各 handler 中的手动 logger.With(...) 替换为该结构。
//
// 字段说明：
//   - RequestID:  请求唯一标识（来自 X-Request-ID / X-Correlation-ID / gin.Context 的 "request_id"）
//   - Path:       请求路径
//   - Method:     HTTP 方法
//   - Provider:   目标供应商（如 "openai"、"anthropic"、"gemini"）
//   - APIStyle:   API 风格（"compat" 或 "native"）
//   - RequestName: 请求名称（如 "chat_completions"、"messages"）
//   - Model:      请求使用的模型名称（可能在解析请求体后才填充）
//   - ClientIP:   客户端 IP 地址
//   - UserAgent:  客户端 User-Agent
//   - Extra:      预留的附加字段，用于流式场景等需要额外上下文的场景
type RequestLogContext struct {
	RequestID   string
	Path        string
	Method      string
	Provider    string
	APIStyle    string
	RequestName string
	Model       string
	ClientIP    string
	UserAgent   string
	Extra       map[string]string
}

type contextKey struct{}

// logCtxKey 是 context.Context 中存储 RequestLogContext 的键。
var logCtxKey contextKey

// NewRequestLogContext 从 gin.Context 提取并构建标准化的日志上下文。
//
// 该函数从 HTTP 请求中提取公共字段（path、method、client_ip、user_agent、request_id），
// 并与调用方提供的业务字段（provider、apiStyle、requestName）合并。
// model 字段通常在解析请求体后通过 WithModel 设置。
func NewRequestLogContext(c *gin.Context, provider, apiStyle, requestName string) RequestLogContext {
	var lc RequestLogContext

	if c != nil && c.Request != nil {
		lc.Path = c.Request.URL.Path
		lc.Method = c.Request.Method
		lc.ClientIP = c.ClientIP()
		lc.UserAgent = c.Request.UserAgent()
		lc.RequestID = requestIDFromGinContext(c)
	}

	lc.Provider = provider
	lc.APIStyle = apiStyle
	lc.RequestName = requestName

	return lc
}

// WithModel 返回一份设置了 Model 字段的副本。
//
// 用于在解析请求体获得模型名称后，补充日志上下文。
func (lc RequestLogContext) WithModel(model string) RequestLogContext {
	lc.Model = model
	return lc
}

// WithExtra 返回一份合并了附加字段的副本。
//
// 已有的 Extra 字段会被保留，传入的键值对会覆盖同名键。
// 该方法为流式场景等需要额外上下文的场景预留扩展能力。
func (lc RequestLogContext) WithExtra(kvs map[string]string) RequestLogContext {
	// 创建新 map 以避免修改原始结构体的 Extra 引用
	newExtra := make(map[string]string, len(lc.Extra)+len(kvs))
	for k, v := range lc.Extra {
		newExtra[k] = v
	}
	for k, v := range kvs {
		newExtra[k] = v
	}
	lc.Extra = newExtra
	return lc
}

// WithContext 将 RequestLogContext 写入 context.Context。
//
// Handler 层调用此方法将日志上下文附加到请求的 context 中，
// 后续 Gateway 层可通过 FromContext 读取。
func (lc RequestLogContext) WithContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, logCtxKey, lc)
}

// FromContext 从 context.Context 中读取 RequestLogContext。
//
// 如果 context 中未存储日志上下文，返回 false。
// Gateway 层使用此方法获取 Handler 透传的日志字段。
func FromContext(ctx context.Context) (RequestLogContext, bool) {
	if ctx == nil {
		return RequestLogContext{}, false
	}
	lc, ok := ctx.Value(logCtxKey).(RequestLogContext)
	return lc, ok
}

// SlogAttrs 将 RequestLogContext 转换为 slog 属性切片。
//
// 仅输出非空字段，保证日志输出简洁。
// 返回的切片可直接传递给 slog.Logger.With() 或日志消息。
func (lc RequestLogContext) SlogAttrs() []any {
	attrs := make([]any, 0, 18)
	// 预估容量：最多 9 个基础字段 × 2（键值对）+ Extra 条目

	if lc.RequestID != "" {
		attrs = append(attrs, "request_id", lc.RequestID)
	}
	if lc.Path != "" {
		attrs = append(attrs, "path", lc.Path)
	}
	if lc.Method != "" {
		attrs = append(attrs, "method", lc.Method)
	}
	if lc.Provider != "" {
		attrs = append(attrs, "provider", lc.Provider)
	}
	if lc.APIStyle != "" {
		attrs = append(attrs, "api_style", lc.APIStyle)
	}
	if lc.RequestName != "" {
		attrs = append(attrs, "request_name", lc.RequestName)
	}
	if lc.Model != "" {
		attrs = append(attrs, "model", lc.Model)
	}
	if lc.ClientIP != "" {
		attrs = append(attrs, "client_ip", lc.ClientIP)
	}
	if lc.UserAgent != "" {
		attrs = append(attrs, "user_agent", lc.UserAgent)
	}

	for k, v := range lc.Extra {
		if k != "" && v != "" {
			attrs = append(attrs, k, v)
		}
	}

	return attrs
}

// EnrichLogger 使用 RequestLogContext 中的字段丰富 slog.Logger。
//
// 返回的 logger 已附加所有非空字段，可直接用于记录日志。
// 这是 Handler 层构建请求级 logger 的便捷方法。
func (lc RequestLogContext) EnrichLogger(base *slog.Logger) *slog.Logger {
	if base == nil {
		base = slog.Default()
	}
	attrs := lc.SlogAttrs()
	if len(attrs) == 0 {
		return base
	}
	return base.With(attrs...)
}

// requestIDFromGinContext 从 gin.Context 中提取请求 ID。
//
// 优先从 gin.Context 的 "request_id" 键获取（由中间件设置），
// 其次从 HTTP 头部 X-Request-ID / X-Correlation-ID / Request-Id 获取。
func requestIDFromGinContext(c *gin.Context) string {
	if c == nil {
		return ""
	}

	// 优先从 context 中获取（中间件可能已设置）
	if v, ok := c.Get("request_id"); ok {
		if s, ok := v.(string); ok {
			if s = strings.TrimSpace(s); s != "" {
				return s
			}
		}
	}

	// 从 HTTP 头部获取
	headers := []string{"X-Request-ID", "X-Correlation-ID", "Request-Id"}
	for _, key := range headers {
		if value := strings.TrimSpace(c.GetHeader(key)); value != "" {
			return value
		}
	}

	return ""
}
