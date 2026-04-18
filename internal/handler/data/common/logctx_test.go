package common

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// --- NewRequestLogContext 测试 ---

func TestNewRequestLogContext_正常提取字段(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/multi/v1/chat/completions", nil)
	c.Request.Header.Set("X-Request-ID", "req-123")
	c.Request.Header.Set("User-Agent", "test-client/1.0")

	lc := NewRequestLogContext(c, "openai", "compat", "chat_completions")

	if lc.RequestID != "req-123" {
		t.Errorf("RequestID = %q, 期望 %q", lc.RequestID, "req-123")
	}
	if lc.Path != "/multi/v1/chat/completions" {
		t.Errorf("Path = %q, 期望 %q", lc.Path, "/multi/v1/chat/completions")
	}
	if lc.Method != http.MethodPost {
		t.Errorf("Method = %q, 期望 %q", lc.Method, http.MethodPost)
	}
	if lc.Provider != "openai" {
		t.Errorf("Provider = %q, 期望 %q", lc.Provider, "openai")
	}
	if lc.APIStyle != "compat" {
		t.Errorf("APIStyle = %q, 期望 %q", lc.APIStyle, "compat")
	}
	if lc.RequestName != "chat_completions" {
		t.Errorf("RequestName = %q, 期望 %q", lc.RequestName, "chat_completions")
	}
	if lc.ClientIP == "" {
		t.Error("ClientIP 不应为空")
	}
	if lc.UserAgent != "test-client/1.0" {
		t.Errorf("UserAgent = %q, 期望 %q", lc.UserAgent, "test-client/1.0")
	}
	if lc.Model != "" {
		t.Errorf("Model 应默认为空，实际 = %q", lc.Model)
	}
}

func TestNewRequestLogContext_nil_gin_context(t *testing.T) {
	lc := NewRequestLogContext(nil, "anthropic", "native", "messages")

	if lc.RequestID != "" {
		t.Errorf("RequestID 应为空，实际 = %q", lc.RequestID)
	}
	if lc.Path != "" {
		t.Errorf("Path 应为空，实际 = %q", lc.Path)
	}
	if lc.Method != "" {
		t.Errorf("Method 应为空，实际 = %q", lc.Method)
	}
	if lc.Provider != "anthropic" {
		t.Errorf("Provider = %q, 期望 %q", lc.Provider, "anthropic")
	}
	if lc.ClientIP != "" {
		t.Errorf("ClientIP 应为空，实际 = %q", lc.ClientIP)
	}
}

func TestNewRequestLogContext_request_id从gin_context_key获取(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/test", nil)
	c.Set("request_id", "gin-set-id")

	lc := NewRequestLogContext(c, "", "", "")
	if lc.RequestID != "gin-set-id" {
		t.Errorf("RequestID = %q, 期望 %q", lc.RequestID, "gin-set-id")
	}
}

func TestNewRequestLogContext_request_id从备用头部获取(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/test", nil)
	c.Request.Header.Set("X-Correlation-ID", "corr-456")

	lc := NewRequestLogContext(c, "", "", "")
	if lc.RequestID != "corr-456" {
		t.Errorf("RequestID = %q, 期望 %q", lc.RequestID, "corr-456")
	}
}

func TestNewRequestLogContext_request_id为空时(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/test", nil)

	lc := NewRequestLogContext(c, "", "", "")
	if lc.RequestID != "" {
		t.Errorf("RequestID 应为空，实际 = %q", lc.RequestID)
	}
}

// --- WithModel / WithExtra 测试 ---

func TestWithModel_设置模型名称(t *testing.T) {
	lc := RequestLogContext{Provider: "openai"}
	lc2 := lc.WithModel("gpt-4o")

	if lc.Model != "" {
		t.Error("原始结构体 Model 应不变")
	}
	if lc2.Model != "gpt-4o" {
		t.Errorf("WithModel 后 Model = %q, 期望 %q", lc2.Model, "gpt-4o")
	}
	if lc2.Provider != "openai" {
		t.Errorf("WithModel 应保留原有字段，Provider = %q", lc2.Provider)
	}
}

func TestWithExtra_合并附加字段(t *testing.T) {
	lc := RequestLogContext{}
	lc2 := lc.WithExtra(map[string]string{
		"protocol_mode": "auto",
		"stream":        "true",
	})

	if len(lc.Extra) != 0 {
		t.Error("原始结构体 Extra 应不变")
	}
	if lc2.Extra["protocol_mode"] != "auto" {
		t.Errorf("Extra[protocol_mode] = %q, 期望 %q", lc2.Extra["protocol_mode"], "auto")
	}
	if lc2.Extra["stream"] != "true" {
		t.Errorf("Extra[stream] = %q, 期望 %q", lc2.Extra["stream"], "true")
	}
}

func TestWithExtra_覆盖同名键(t *testing.T) {
	lc := RequestLogContext{
		Extra: map[string]string{"stream": "false"},
	}
	lc2 := lc.WithExtra(map[string]string{"stream": "true"})

	if lc.Extra["stream"] != "false" {
		t.Error("原始结构体 Extra 应不变")
	}
	if lc2.Extra["stream"] != "true" {
		t.Errorf("覆盖后 Extra[stream] = %q, 期望 %q", lc2.Extra["stream"], "true")
	}
}

// --- WithContext / FromContext 测试 ---

func TestWithContext_写入和读取(t *testing.T) {
	lc := RequestLogContext{
		RequestID: "req-001",
		Provider:  "anthropic",
		Model:     "claude-3",
	}
	ctx := lc.WithContext(context.Background())

	lc2, ok := FromContext(ctx)
	if !ok {
		t.Error("FromContext 应返回 true")
	}
	if lc2.RequestID != "req-001" {
		t.Errorf("RequestID = %q, 期望 %q", lc2.RequestID, "req-001")
	}
	if lc2.Provider != "anthropic" {
		t.Errorf("Provider = %q, 期望 %q", lc2.Provider, "anthropic")
	}
	if lc2.Model != "claude-3" {
		t.Errorf("Model = %q, 期望 %q", lc2.Model, "claude-3")
	}
}

func TestFromContext_nil_context(t *testing.T) {
	lc, ok := FromContext(nil)
	if ok {
		t.Error("FromContext(nil) 应返回 false")
	}
	if lc.RequestID != "" {
		t.Errorf("空 context 应返回零值结构体，RequestID = %q", lc.RequestID)
	}
}

func TestFromContext_无日志上下文(t *testing.T) {
	lc, ok := FromContext(context.Background())
	if ok {
		t.Error("未写入日志上下文时 FromContext 应返回 false")
	}
	if lc.RequestID != "" {
		t.Errorf("未写入日志上下文时 RequestID 应为空，实际 = %q", lc.RequestID)
	}
}

// --- SlogAttrs 测试 ---

func TestSlogAttrs_仅输出非空字段(t *testing.T) {
	lc := RequestLogContext{
		RequestID:   "req-001",
		Provider:    "openai",
		RequestName: "chat_completions",
		Model:       "gpt-4o",
	}
	attrs := lc.SlogAttrs()

	// 检查非空字段存在
	hasKey := func(key string) bool {
		for i := 0; i < len(attrs)-1; i += 2 {
			if attrs[i] == key {
				return true
			}
		}
		return false
	}

	if !hasKey("request_id") {
		t.Error("SlogAttrs 应包含 request_id")
	}
	if !hasKey("provider") {
		t.Error("SlogAttrs 应包含 provider")
	}
	if !hasKey("request_name") {
		t.Error("SlogAttrs 应包含 request_name")
	}
	if !hasKey("model") {
		t.Error("SlogAttrs 应包含 model")
	}
	if hasKey("path") {
		t.Error("SlogAttrs 不应包含空 path")
	}
	if hasKey("method") {
		t.Error("SlogAttrs 不应包含空 method")
	}
	if hasKey("api_style") {
		t.Error("SlogAttrs 不应包含空 api_style")
	}
}

func TestSlogAttrs_包含Extra字段(t *testing.T) {
	lc := RequestLogContext{
		Provider: "gemini",
		Extra: map[string]string{
			"protocol_mode": "sse",
			"stream":        "true",
		},
	}
	attrs := lc.SlogAttrs()

	attrMap := attrsToMap(attrs)
	if attrMap["protocol_mode"] != "sse" {
		t.Errorf("Extra[protocol_mode] = %q, 期望 %q", attrMap["protocol_mode"], "sse")
	}
	if attrMap["stream"] != "true" {
		t.Errorf("Extra[stream] = %q, 期望 %q", attrMap["stream"], "true")
	}
}

func TestSlogAttrs_跳过Extra中空键或空值(t *testing.T) {
	lc := RequestLogContext{
		Extra: map[string]string{
			"":          "should_skip",
			"valid_key": "",
		},
	}
	attrs := lc.SlogAttrs()

	for i := 0; i < len(attrs)-1; i += 2 {
		if attrs[i] == "" {
			t.Error("SlogAttrs 不应包含空键")
		}
		if attrs[i] == "valid_key" {
			t.Error("SlogAttrs 不应包含值为空的 Extra 条目")
		}
	}
}

func TestSlogAttrs_全部为空时(t *testing.T) {
	lc := RequestLogContext{}
	attrs := lc.SlogAttrs()
	if len(attrs) != 0 {
		t.Errorf("全部字段为空时 SlogAttrs 应返回空切片，实际长度 = %d", len(attrs))
	}
}

// --- EnrichLogger 测试 ---

func TestEnrichLogger_附加字段到logger(t *testing.T) {
	var buf strings.Builder
	base := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))

	lc := RequestLogContext{
		RequestID: "req-001",
		Provider:  "openai",
	}
	logger := lc.EnrichLogger(base)
	logger.Info("测试日志")

	output := buf.String()
	if !strings.Contains(output, "request_id=req-001") {
		t.Errorf("输出应包含 request_id，实际输出: %s", output)
	}
	if !strings.Contains(output, "provider=openai") {
		t.Errorf("输出应包含 provider，实际输出: %s", output)
	}
}

func TestEnrichLogger_nil_logger使用默认(t *testing.T) {
	lc := RequestLogContext{Provider: "test"}
	logger := lc.EnrichLogger(nil)
	if logger == nil {
		t.Error("EnrichLogger(nil) 应返回非 nil logger")
	}
}

func TestEnrichLogger_空字段不附加(t *testing.T) {
	var buf strings.Builder
	base := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))

	lc := RequestLogContext{} // 全空
	logger := lc.EnrichLogger(base)

	// 应返回原始 logger（无附加字段）
	if logger != base {
		t.Error("全空字段时应返回原始 logger")
	}
}

// --- 辅助函数 ---

func attrsToMap(attrs []any) map[string]string {
	m := make(map[string]string, len(attrs)/2)
	for i := 0; i < len(attrs)-1; i += 2 {
		key, _ := attrs[i].(string)
		val, _ := attrs[i+1].(string)
		m[key] = val
	}
	return m
}
