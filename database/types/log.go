package types

import (
	"time"
)

// RequestLog 表示单个请求的统计信息
type RequestLog struct {
	ID uint `json:"id"` // 唯一标识符

	// 请求基本信息
	Timestamp         time.Time `gorm:"index" json:"timestamp"`                     // 请求时间
	ModelName         string    `gorm:"index" json:"model_name"`                    // 模型名称
	OriginalModelName string    `gorm:"index" json:"original_model_name,omitempty"` // 原始模型名称（用户请求中的模型名称）
	IsStream          bool      `gorm:"index;default:false" json:"is_stream"`       // 是否为流式请求
	IsNative          bool      `gorm:"index;default:false" json:"is_native"`       // 是否为原生（native）请求

	// 通道信息
	PlatformID uint `gorm:"index" json:"platform_id"` // 平台 ID
	APIKeyID   uint `json:"api_key_id"`               // 密钥 ID
	ModelID    uint `json:"model_id"`                 // 模型 ID

	// 耗时信息
	Duration      int64  `json:"duration"`                  // 总用时 (微秒)
	FirstByteTime *int64 `json:"first_byte_time,omitempty"` // 首字用时（微秒，仅流式）

	// 结果状态
	Success  bool    `gorm:"index" json:"success"` // 是否成功
	ErrorMsg *string `json:"error_msg,omitempty"`  // 错误信息（失败时）

	// 结构化错误字段
	ErrorCode  *string `json:"error_code,omitempty"`
	ErrorLevel *string `json:"error_level,omitempty"`
	HTTPStatus *int    `json:"http_status,omitempty"`
	ErrorFrom  *string `json:"error_from,omitempty"`

	// 上游错误字段
	UpstreamErrorType    *string `json:"upstream_error_type,omitempty"`
	UpstreamErrorCode    *string `json:"upstream_error_code,omitempty"`
	UpstreamErrorParam   *string `json:"upstream_error_param,omitempty"`
	UpstreamErrorMessage *string `json:"upstream_error_message,omitempty"`
	UpstreamRequestID    *string `json:"upstream_request_id,omitempty"`

	// 响应体解析状态
	ResponseBodyIsJSON *bool   `json:"response_body_is_json,omitempty"`
	ResponseBodyRaw    *string `json:"response_body_raw,omitempty"`

	// Token 使用统计
	PromptTokens     *int `json:"prompt_tokens"`     // 提示 Token 数
	CompletionTokens *int `json:"completion_tokens"` // 完成 Token 数
	TotalTokens      *int `json:"total_tokens"`      // 总 Token 数
}
