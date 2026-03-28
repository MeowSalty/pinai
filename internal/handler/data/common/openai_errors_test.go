package common

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/MeowSalty/pinai/internal/app/gateway"
)

type failWriter struct{}

func (w failWriter) Write(_ []byte) (int, error) {
	return 0, io.ErrClosedPipe
}

func TestNewOpenAIHTTPErrorResponse_使用结构化协议错误字段(t *testing.T) {
	protocolErr := &gateway.DataPlaneError{
		StatusCode: 429,
		Message:    "配额不足",
		ErrorType:  OpenAIErrorTypeRateLimit,
		ErrorCode:  "insufficient_quota",
		Param:      "model",
	}

	resp := NewOpenAIHTTPErrorResponse("兜底错误", 500, errors.New("内部错误"), protocolErr)

	if resp.Error.Message != "配额不足" {
		t.Fatalf("错误消息应优先使用协议错误字段，实际值=%q", resp.Error.Message)
	}
	if resp.Error.Type != OpenAIErrorTypeRateLimit {
		t.Fatalf("错误类型应优先使用协议错误字段，实际值=%q", resp.Error.Type)
	}
	if resp.Error.Code == nil || *resp.Error.Code != "insufficient_quota" {
		t.Fatalf("错误码应优先使用协议错误字段，实际值=%v", resp.Error.Code)
	}
	if resp.Error.Param == nil || *resp.Error.Param != "model" {
		t.Fatalf("参数字段应优先使用协议错误字段，实际值=%v", resp.Error.Param)
	}
}

func TestWriteOpenAIChatSSEError_写入结构化错误事件(t *testing.T) {
	protocolErr := &gateway.DataPlaneError{
		StatusCode: 400,
		Message:    "请求参数错误",
		ErrorType:  OpenAIErrorTypeInvalidRequest,
		ErrorCode:  "invalid_param",
		Param:      "messages",
	}

	var buf bytes.Buffer
	if err := WriteOpenAIChatSSEError(&buf, "兜底错误", 500, errors.New("内部错误"), protocolErr); err != nil {
		t.Fatalf("写入 OpenAI Chat SSE 错误事件失败：%v", err)
	}

	out := buf.String()
	const prefix = "event: error\ndata: "
	if !strings.HasPrefix(out, prefix) {
		t.Fatalf("SSE 输出应以 event: error + data: 前缀开头，实际输出=%q", out)
	}
	if !strings.HasSuffix(out, "\n\n") {
		t.Fatalf("SSE 输出应以双换行结尾，实际输出=%q", out)
	}

	payload := strings.TrimSuffix(strings.TrimPrefix(out, prefix), "\n\n")
	var resp OpenAIHTTPErrorResponse
	if err := json.Unmarshal([]byte(payload), &resp); err != nil {
		t.Fatalf("解析 SSE 中的 JSON 负载失败：%v", err)
	}

	if resp.Error.Message != "请求参数错误" {
		t.Fatalf("错误消息不符合预期，实际值=%q", resp.Error.Message)
	}
	if resp.Error.Type != OpenAIErrorTypeInvalidRequest {
		t.Fatalf("错误类型不符合预期，实际值=%q", resp.Error.Type)
	}
	if resp.Error.Code == nil || *resp.Error.Code != "invalid_param" {
		t.Fatalf("错误码不符合预期，实际值=%v", resp.Error.Code)
	}
}

func TestWriteOpenAIResponsesTypedEventError_写入TypedEvent错误事件(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteOpenAIResponsesTypedEventError(&buf, "处理失败", 500, errors.New("boom")); err != nil {
		t.Fatalf("写入 OpenAI Responses typed event 错误事件失败：%v", err)
	}

	out := buf.String()
	const prefix = "event: response.error\ndata: "
	if !strings.HasPrefix(out, prefix) {
		t.Fatalf("typed event 输出前缀不符合预期，实际输出=%q", out)
	}
	if !strings.HasSuffix(out, "\n\n") {
		t.Fatalf("typed event 输出应以双换行结尾，实际输出=%q", out)
	}

	payload := strings.TrimSuffix(strings.TrimPrefix(out, prefix), "\n\n")
	var typedErr OpenAIResponsesTypedEventError
	if err := json.Unmarshal([]byte(payload), &typedErr); err != nil {
		t.Fatalf("解析 typed event JSON 失败：%v", err)
	}

	if typedErr.Type != "response.error" {
		t.Fatalf("typed event type 不符合预期，实际值=%q", typedErr.Type)
	}
	if typedErr.Error.Code == nil || *typedErr.Error.Code != OpenAIErrorTypeInternal {
		t.Fatalf("internal 错误应带默认 code=internal_error，实际值=%v", typedErr.Error.Code)
	}
}

func TestNewOpenAIHTTPErrorResponse_主文案追加内部细节(t *testing.T) {
	resp := NewOpenAIHTTPErrorResponse("请求处理失败", 500, errors.New("上游连接超时"))

	if resp.Error.Message != "请求处理失败（内部细节：上游连接超时）" {
		t.Fatalf("错误消息拼装不符合预期，实际值=%q", resp.Error.Message)
	}
}

func TestNewOpenAIHTTPErrorResponse_避免重复拼接内部细节(t *testing.T) {
	resp := NewOpenAIHTTPErrorResponse("请求参数错误", 400, errors.New("请求参数错误"))

	if resp.Error.Message != "请求参数错误" {
		t.Fatalf("错误消息不应重复拼接，实际值=%q", resp.Error.Message)
	}
}

func TestWriteOpenAIChatSSEError_写入失败时返回流式写入错误(t *testing.T) {
	err := WriteOpenAIChatSSEError(failWriter{}, "写入失败", 500, errors.New("boom"))
	if err == nil {
		t.Fatalf("预期返回写入失败错误")
	}
	if !IsOpenAIStreamWriteError(err) {
		t.Fatalf("预期为 OpenAI 流式写入错误，实际=%v", err)
	}
}

func TestWriteOpenAIResponsesTypedEventError_写入失败时返回流式写入错误(t *testing.T) {
	err := WriteOpenAIResponsesTypedEventError(failWriter{}, "写入失败", 500, errors.New("boom"))
	if err == nil {
		t.Fatalf("预期返回写入失败错误")
	}
	if !IsOpenAIStreamWriteError(err) {
		t.Fatalf("预期为 OpenAI 流式写入错误，实际=%v", err)
	}
}

func TestIsOpenAIStreamWriteError_非写入错误返回false(t *testing.T) {
	if IsOpenAIStreamWriteError(errors.New("普通错误")) {
		t.Fatalf("普通错误不应被识别为流式写入错误")
	}
}
