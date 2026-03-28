package common

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/MeowSalty/pinai/internal/app/gateway"
	geminiTypes "github.com/MeowSalty/portal/request/adapter/gemini/types"
)

type geminiFailWriter struct{}

func (w geminiFailWriter) Write(_ []byte) (int, error) {
	return 0, io.ErrClosedPipe
}

func TestWriteGeminiStreamError_写入Gemini标准错误块(t *testing.T) {
	protocolErr := &gateway.DataPlaneError{
		StatusCode: 429,
		Message:    "配额不足",
		ErrorCode:  GeminiErrorStatusResourceExhausted,
	}

	var buf bytes.Buffer
	if err := WriteGeminiStreamError(&buf, "", 500, nil, protocolErr); err != nil {
		t.Fatalf("写入 Gemini 流式错误块失败：%v", err)
	}

	out := buf.String()
	const prefix = "data: "
	if !strings.HasPrefix(out, prefix) {
		t.Fatalf("Gemini 流式错误块应以 data: 前缀开头，实际输出=%q", out)
	}
	if !strings.HasSuffix(out, "\n\n") {
		t.Fatalf("Gemini 流式错误块应以双换行结尾，实际输出=%q", out)
	}

	payload := strings.TrimSuffix(strings.TrimPrefix(out, prefix), "\n\n")
	var resp geminiTypes.ErrorResponse
	if err := json.Unmarshal([]byte(payload), &resp); err != nil {
		t.Fatalf("解析 Gemini 流式错误 JSON 负载失败：%v", err)
	}

	if resp.Error.Code != 429 {
		t.Fatalf("错误状态码不符合预期，实际值=%d", resp.Error.Code)
	}
	if resp.Error.Message != "配额不足" {
		t.Fatalf("错误消息不符合预期，实际值=%q", resp.Error.Message)
	}
	if resp.Error.Status != GeminiErrorStatusResourceExhausted {
		t.Fatalf("错误状态字段不符合预期，实际值=%q", resp.Error.Status)
	}
}

func TestWriteGeminiStreamError_写入失败时返回流式写入错误(t *testing.T) {
	err := WriteGeminiStreamError(geminiFailWriter{}, "写入失败", 500, errors.New("boom"))
	if err == nil {
		t.Fatalf("预期返回写入失败错误")
	}
	if !IsGeminiStreamWriteError(err) {
		t.Fatalf("预期为 Gemini 流式写入错误，实际=%v", err)
	}
}

func TestIsGeminiStreamWriteError_非写入错误返回false(t *testing.T) {
	if IsGeminiStreamWriteError(errors.New("普通错误")) {
		t.Fatalf("普通错误不应被识别为流式写入错误")
	}
}
