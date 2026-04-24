package gateway

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"testing"
	"time"
)

// --- enrichLoggerFromContext 测试 ---

func TestEnrichLoggerFromContext_无桥接函数时返回原始logger(t *testing.T) {
	// 保存并恢复全局变量
	origFn := logCtxAttrsFromContext
	logCtxAttrsFromContext = func(ctx context.Context) []any { return nil }
	defer func() { logCtxAttrsFromContext = origFn }()

	base := slog.Default()
	result := enrichLoggerFromContext(context.Background(), base)

	if result != base {
		t.Error("无桥接函数时应返回原始 logger")
	}
}

func TestEnrichLoggerFromContext_上下文无属性时返回原始logger(t *testing.T) {
	origFn := logCtxAttrsFromContext
	logCtxAttrsFromContext = func(ctx context.Context) []any { return nil }
	defer func() { logCtxAttrsFromContext = origFn }()

	base := slog.Default()
	result := enrichLoggerFromContext(context.Background(), base)

	if result != base {
		t.Error("上下文无属性时应返回原始 logger")
	}
}

func TestEnrichLoggerFromContext_附加上下文字段到logger(t *testing.T) {
	origFn := logCtxAttrsFromContext
	logCtxAttrsFromContext = func(ctx context.Context) []any {
		return []any{"request_id", "req-001", "provider", "openai", "client_ip", "1.2.3.4"}
	}
	defer func() { logCtxAttrsFromContext = origFn }()

	var buf strings.Builder
	base := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	result := enrichLoggerFromContext(context.Background(), base)
	result.Info("测试日志")

	output := buf.String()
	if !strings.Contains(output, "request_id=req-001") {
		t.Errorf("输出应包含 request_id，实际输出: %s", output)
	}
	if !strings.Contains(output, "provider=openai") {
		t.Errorf("输出应包含 provider，实际输出: %s", output)
	}
	if !strings.Contains(output, "client_ip=1.2.3.4") {
		t.Errorf("输出应包含 client_ip，实际输出: %s", output)
	}
}

func TestEnrichLoggerFromContext_过滤request_name和model字段(t *testing.T) {
	origFn := logCtxAttrsFromContext
	logCtxAttrsFromContext = func(ctx context.Context) []any {
		return []any{
			"request_id", "req-002",
			"request_name", "chat_completions",
			"model", "gpt-4o",
			"provider", "anthropic",
		}
	}
	defer func() { logCtxAttrsFromContext = origFn }()

	var buf strings.Builder
	base := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	result := enrichLoggerFromContext(context.Background(), base)
	result.Info("测试日志")

	output := buf.String()
	// 应保留 request_id 和 provider
	if !strings.Contains(output, "request_id=req-002") {
		t.Errorf("输出应包含 request_id，实际输出: %s", output)
	}
	if !strings.Contains(output, "provider=anthropic") {
		t.Errorf("输出应包含 provider，实际输出: %s", output)
	}
	// 应过滤 request_name 和 model（防止与 gateway 侧 attrs 重复）
	if strings.Contains(output, "request_name=chat_completions") {
		t.Errorf("输出不应包含 request_name（应被过滤），实际输出: %s", output)
	}
	if strings.Contains(output, "model=gpt-4o") {
		t.Errorf("输出不应包含 model（应被过滤），实际输出: %s", output)
	}
}

func TestEnrichLoggerFromContext_全部被过滤后返回原始logger(t *testing.T) {
	origFn := logCtxAttrsFromContext
	logCtxAttrsFromContext = func(ctx context.Context) []any {
		// 只有 request_name 和 model，全部会被过滤
		return []any{"request_name", "chat_completions", "model", "gpt-4o"}
	}
	defer func() { logCtxAttrsFromContext = origFn }()

	base := slog.Default()
	result := enrichLoggerFromContext(context.Background(), base)

	if result != base {
		t.Error("全部字段被过滤后应返回原始 logger")
	}
}

// --- newStreamLogContext 测试 ---

func TestNewStreamLogContext_基本字段(t *testing.T) {
	origFn := logCtxAttrsFromContext
	logCtxAttrsFromContext = func(ctx context.Context) []any { return nil }
	defer func() { logCtxAttrsFromContext = origFn }()

	base := slog.Default()
	slc := newStreamLogContext(context.Background(), base, "gateway", "chat_completions", "gpt-4o")

	if slc.logger == nil {
		t.Error("logger 不应为 nil")
	}
	// attrs 应包含 request_name 和 model
	attrMap := attrsToMapGateway(slc.attrs)
	if attrMap["request_name"] != "chat_completions" {
		t.Errorf("attrs[request_name] = %q, 期望 %q", attrMap["request_name"], "chat_completions")
	}
	if attrMap["model"] != "gpt-4o" {
		t.Errorf("attrs[model] = %q, 期望 %q", attrMap["model"], "gpt-4o")
	}
}

func TestNewStreamLogContext_从上下文丰富logger(t *testing.T) {
	origFn := logCtxAttrsFromContext
	logCtxAttrsFromContext = func(ctx context.Context) []any {
		return []any{"request_id", "req-003", "client_ip", "10.0.0.1"}
	}
	defer func() { logCtxAttrsFromContext = origFn }()

	var buf strings.Builder
	base := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	slc := newStreamLogContext(context.Background(), base, "gateway", "messages", "claude-3")
	slc.logger.Info("测试", slc.attrs...)

	output := buf.String()
	if !strings.Contains(output, "request_id=req-003") {
		t.Errorf("输出应包含从上下文透传的 request_id，实际输出: %s", output)
	}
	if !strings.Contains(output, "client_ip=10.0.0.1") {
		t.Errorf("输出应包含从上下文透传的 client_ip，实际输出: %s", output)
	}
}

// --- logNonStreamError 分级逻辑测试 ---

// levelRecorder 记录 slog 日志级别的测试 handler。
type levelRecorder struct {
	slog.Handler
	recordedLevel slog.Level
	recordedMsg   string
}

func (h *levelRecorder) Handle(_ context.Context, r slog.Record) error {
	h.recordedLevel = r.Level
	h.recordedMsg = r.Message
	return nil
}

func (h *levelRecorder) Enabled(_ context.Context, level slog.Level) bool {
	return true
}

// httpError 实现 StatusCode() 接口，用于构造携带 HTTP 状态码的测试错误。
// MapDataPlaneError 通过 extractByInterfaces 提取 StatusCode，
// 使 logNonStreamError 的 4xx/5xx 分级逻辑可被正确触发。
type httpError struct {
	statusCode int
	msg        string
}

func (e *httpError) Error() string   { return e.msg }
func (e *httpError) StatusCode() int { return e.statusCode }

func TestLogNonStreamError_4xx记WARN(t *testing.T) {
	rec := &levelRecorder{}
	logger := slog.New(rec)

	svc := &service{logger: logger}

	// 构造一个携带 404 状态码的结构化错误
	err := &httpError{statusCode: 404, msg: "resource not found"}

	svc.logNonStreamError(logger, "test_action", err, 100*time.Millisecond, "test-model")

	if rec.recordedLevel != slog.LevelWarn {
		t.Errorf("4xx 错误应记 WARN，实际级别 = %v", rec.recordedLevel)
	}
	if rec.recordedMsg != "非流式请求上游返回协议错误" {
		t.Errorf("4xx 错误消息 = %q, 期望 %q", rec.recordedMsg, "非流式请求上游返回协议错误")
	}
}

func TestLogNonStreamError_5xx记ERROR(t *testing.T) {
	rec := &levelRecorder{}
	logger := slog.New(rec)

	svc := &service{logger: logger}

	// 构造一个携带 500 状态码的结构化错误
	err := &httpError{statusCode: 500, msg: "internal server failure"}

	svc.logNonStreamError(logger, "test_action", err, 200*time.Millisecond, "test-model")

	if rec.recordedLevel != slog.LevelError {
		t.Errorf("5xx 错误应记 ERROR，实际级别 = %v", rec.recordedLevel)
	}
	if rec.recordedMsg != "非流式请求执行失败" {
		t.Errorf("5xx 错误消息 = %q, 期望 %q", rec.recordedMsg, "非流式请求执行失败")
	}
}

func TestLogNonStreamError_400记WARN(t *testing.T) {
	rec := &levelRecorder{}
	logger := slog.New(rec)

	svc := &service{logger: logger}

	// 构造一个携带 400 状态码的结构化错误
	err := &httpError{statusCode: 400, msg: "bad request: invalid parameter"}

	svc.logNonStreamError(logger, "test_action", err, 50*time.Millisecond, "test-model")

	if rec.recordedLevel != slog.LevelWarn {
		t.Errorf("400 错误应记 WARN，实际级别 = %v", rec.recordedLevel)
	}
}

func TestLogNonStreamError_429记WARN(t *testing.T) {
	rec := &levelRecorder{}
	logger := slog.New(rec)

	svc := &service{logger: logger}

	// 构造一个携带 429 状态码的结构化错误
	err := &httpError{statusCode: 429, msg: "rate limit exceeded"}

	svc.logNonStreamError(logger, "test_action", err, 300*time.Millisecond, "test-model")

	if rec.recordedLevel != slog.LevelWarn {
		t.Errorf("429 错误应记 WARN，实际级别 = %v", rec.recordedLevel)
	}
}

func TestLogNonStreamError_超时记ERROR(t *testing.T) {
	rec := &levelRecorder{}
	logger := slog.New(rec)

	svc := &service{logger: logger}

	// 超时错误会被映射为 504（Gateway Timeout），属于 5xx
	err := fmt.Errorf("upstream timeout: %w", context.DeadlineExceeded)

	svc.logNonStreamError(logger, "test_action", err, 5000*time.Millisecond, "test-model")

	if rec.recordedLevel != slog.LevelError {
		t.Errorf("超时 504 应记 ERROR，实际级别 = %v", rec.recordedLevel)
	}
}

func TestLogNonStreamError_边界399记ERROR(t *testing.T) {
	rec := &levelRecorder{}
	logger := slog.New(rec)

	svc := &service{logger: logger}

	// 399 不属于 4xx 范围，应走 ERROR 路径
	err := &httpError{statusCode: 399, msg: "redirect or similar"}

	svc.logNonStreamError(logger, "test_action", err, 10*time.Millisecond, "test-model")

	if rec.recordedLevel != slog.LevelError {
		t.Errorf("399 不属于 4xx 范围应记 ERROR，实际级别 = %v", rec.recordedLevel)
	}
}

// --- RegisterLogCtxAttrsFromContext 测试 ---

func TestRegisterLogCtxAttrsFromContext_注册非nil函数(t *testing.T) {
	origFn := logCtxAttrsFromContext
	defer func() { logCtxAttrsFromContext = origFn }()

	called := false
	RegisterLogCtxAttrsFromContext(func(ctx context.Context) []any {
		called = true
		return []any{"test_key", "test_val"}
	})

	// 验证注册后调用 logCtxAttrsFromContext 会执行新函数
	result := logCtxAttrsFromContext(context.Background())
	if !called {
		t.Error("注册后应调用新函数")
	}
	if len(result) != 2 || result[0] != "test_key" || result[1] != "test_val" {
		t.Errorf("注册后返回值不符，实际 = %v", result)
	}
}

func TestRegisterLogCtxAttrsFromContext_注册nil不覆盖(t *testing.T) {
	origFn := logCtxAttrsFromContext
	defer func() { logCtxAttrsFromContext = origFn }()

	// 先注册一个有效函数
	RegisterLogCtxAttrsFromContext(func(ctx context.Context) []any {
		return []any{"existing", "value"}
	})

	// 再注册 nil，不应覆盖
	RegisterLogCtxAttrsFromContext(nil)

	result := logCtxAttrsFromContext(context.Background())
	if len(result) != 2 || result[0] != "existing" {
		t.Errorf("注册 nil 不应覆盖已有函数，实际 = %v", result)
	}
}

// --- 辅助函数 ---

func attrsToMapGateway(attrs []any) map[string]string {
	m := make(map[string]string, len(attrs)/2)
	for i := 0; i+1 < len(attrs); i += 2 {
		key, _ := attrs[i].(string)
		val, _ := attrs[i+1].(string)
		m[key] = val
	}
	return m
}
