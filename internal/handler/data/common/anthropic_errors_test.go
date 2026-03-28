package common

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/MeowSalty/pinai/internal/app/gateway"
	anthropicTypes "github.com/MeowSalty/portal/request/adapter/anthropic/types"
)

type anthropicFailWriter struct{}

func (w anthropicFailWriter) Write(_ []byte) (int, error) {
	return 0, io.ErrClosedPipe
}

func TestWriteAnthropicSSEError_写入Anthropic标准错误事件(t *testing.T) {
	protocolErr := &gateway.DataPlaneError{
		StatusCode: 429,
		Message:    "请求频率过高",
		ErrorType:  AnthropicErrorTypeRateLimit,
	}

	var buf bytes.Buffer
	if err := WriteAnthropicSSEError(&buf, "", 500, nil, protocolErr); err != nil {
		t.Fatalf("写入 Anthropic SSE 错误事件失败：%v", err)
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
	var resp anthropicTypes.ErrorResponse
	if err := json.Unmarshal([]byte(payload), &resp); err != nil {
		t.Fatalf("解析 SSE 中的 JSON 负载失败：%v", err)
	}

	if resp.Type != "error" {
		t.Fatalf("错误响应顶层 type 不符合预期，实际值=%q", resp.Type)
	}
	if resp.Error.Type != AnthropicErrorTypeRateLimit {
		t.Fatalf("错误类型不符合预期，实际值=%q", resp.Error.Type)
	}
	if resp.Error.Message != "请求频率过高" {
		t.Fatalf("错误消息不符合预期，实际值=%q", resp.Error.Message)
	}
}

func TestWriteAnthropicSSEError_写入失败时返回流式写入错误(t *testing.T) {
	err := WriteAnthropicSSEError(anthropicFailWriter{}, "写入失败", 500, errors.New("boom"))
	if err == nil {
		t.Fatalf("预期返回写入失败错误")
	}
	if !IsAnthropicStreamWriteError(err) {
		t.Fatalf("预期为 Anthropic 流式写入错误，实际=%v", err)
	}
}

func TestIsAnthropicStreamWriteError_非写入错误返回false(t *testing.T) {
	if IsAnthropicStreamWriteError(errors.New("普通错误")) {
		t.Fatalf("普通错误不应被识别为流式写入错误")
	}
}
