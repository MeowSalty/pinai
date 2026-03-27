package common

import (
	"errors"
	"testing"
)

func TestNewAnthropicErrorResponse_主文案追加内部细节(t *testing.T) {
	resp := NewAnthropicErrorResponse("请求处理失败", 500, errors.New("上游服务暂不可用"))

	if resp.Error.Message != "请求处理失败（内部细节：上游服务暂不可用）" {
		t.Fatalf("Anthropic 错误消息拼装不符合预期，实际值=%q", resp.Error.Message)
	}
}

func TestNewAnthropicErrorResponse_避免重复拼接内部细节(t *testing.T) {
	resp := NewAnthropicErrorResponse("鉴权失败", 401, errors.New("鉴权失败"))

	if resp.Error.Message != "鉴权失败" {
		t.Fatalf("Anthropic 错误消息不应重复拼接，实际值=%q", resp.Error.Message)
	}
}

func TestNewGeminiErrorResponse_主文案追加内部细节(t *testing.T) {
	resp := NewGeminiErrorResponse("请求处理失败", 500, errors.New("上游网关超时"))

	if resp.Error.Message != "请求处理失败（内部细节：上游网关超时）" {
		t.Fatalf("Gemini 错误消息拼装不符合预期，实际值=%q", resp.Error.Message)
	}
}

func TestNewGeminiErrorResponse_避免重复拼接内部细节(t *testing.T) {
	resp := NewGeminiErrorResponse("请求处理失败", 500, errors.New("请求处理失败"))

	if resp.Error.Message != "请求处理失败" {
		t.Fatalf("Gemini 错误消息不应重复拼接，实际值=%q", resp.Error.Message)
	}
}
