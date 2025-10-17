package types

import (
	"time"
)

// RequestLog 表示单个请求的统计信息
type RequestLog struct {
	ID string `gorm:"primaryKey" json:"id"` // 唯一标识符

	// 请求基本信息
	Timestamp         time.Time `gorm:"index" json:"timestamp"`                     // 请求时间
	RequestType       string    `gorm:"index" json:"request_type"`                  // 请求类型：stream 或 non-stream
	ModelName         string    `gorm:"index" json:"model_name"`                    // 模型名称
	OriginalModelName string    `gorm:"index" json:"original_model_name,omitempty"` // 原始模型名称（用户请求中的模型名称）

	// 通道信息
	PlatformID uint `json:"platform_id"` // 平台 ID
	APIKeyID   uint `json:"api_key_id"`  // 密钥 ID
	ModelID    uint `json:"model_id"`    // 模型 ID

	// 耗时信息
	Duration      int64  `json:"duration"`                  // 总用时 (微秒)
	FirstByteTime *int64 `json:"first_byte_time,omitempty"` // 首字用时（微秒，仅流式）

	// 结果状态
	Success  bool    `gorm:"index" json:"success"` // 是否成功
	ErrorMsg *string `json:"error_msg,omitempty"`  // 错误信息（失败时）

	// Token 使用统计
	PromptTokens     *int `json:"prompt_tokens"`     // 提示 Token 数
	CompletionTokens *int `json:"completion_tokens"` // 完成 Token 数
	TotalTokens      *int `json:"total_tokens"`      // 总 Token 数

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
