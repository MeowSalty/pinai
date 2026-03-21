package health

import (
	"time"

	"github.com/MeowSalty/pinai/database/types"
)

// ResourceHealthSummary 单个资源类型的健康状态汇总
type ResourceHealthSummary struct {
	Total       int64 `json:"total"`       // 总数
	Available   int64 `json:"available"`   // 可用数量
	Warning     int64 `json:"warning"`     // 警告数量
	Unavailable int64 `json:"unavailable"` // 不可用数量
	Unknown     int64 `json:"unknown"`     // 未知数量
}

// HealthSummaryResponse 健康状态统计响应
type HealthSummaryResponse struct {
	Platform ResourceHealthSummary `json:"platform"` // 平台健康状态统计
	APIKey   ResourceHealthSummary `json:"api_key"`  // 密钥健康状态统计
	Model    ResourceHealthSummary `json:"model"`    // 模型健康状态统计
}

// PlatformHealthItem 单个平台健康状态项
type PlatformHealthItem struct {
	PlatformID              uint               `json:"platform_id"`                // 平台 ID
	PlatformName            string             `json:"platform_name"`              // 平台名称
	Status                  types.HealthStatus `json:"status"`                     // 健康状态
	RetryCount              int                `json:"retry_count"`                // 重试次数
	NextAvailableAt         *time.Time         `json:"next_available_at"`          // 下次可用时间
	BackoffDuration         int64              `json:"backoff_duration"`           // 当前退避时长（秒）
	LastError               string             `json:"last_error"`                 // 最后错误信息
	LastErrorCode           int                `json:"last_error_code"`            // 最后错误码
	LastErrorMessage        string             `json:"last_error_message"`         // 最后错误展示消息
	LastStructuredErrorCode string             `json:"last_structured_error_code"` // 最后稳定错误码
	LastHTTPStatus          *int               `json:"last_http_status"`           // 最后 HTTP 状态码
	LastErrorFrom           string             `json:"last_error_from"`            // 最后错误来源
	LastCauseMessage        string             `json:"last_cause_message"`         // 最后根因文本
	LastCheckAt             time.Time          `json:"last_check_at"`              // 最后检查时间
	LastSuccessAt           *time.Time         `json:"last_success_at"`            // 最后成功时间
	SuccessCount            int                `json:"success_count"`              // 成功次数
	ErrorCount              int                `json:"error_count"`                // 错误次数
}

// PlatformHealthListResponse 平台健康列表响应
type PlatformHealthListResponse struct {
	Items    []PlatformHealthItem `json:"items"`     // 平台健康列表
	Total    int                  `json:"total"`     // 总数
	Page     int                  `json:"page"`      // 当前页码
	PageSize int                  `json:"page_size"` // 每页大小
}

// APIKeyHealthItem 单个密钥健康状态项
type APIKeyHealthItem struct {
	KeyID                   uint               `json:"key_id"`                     // 密钥 ID
	KeyValue                string             `json:"key_value"`                  // 密钥值
	Status                  types.HealthStatus `json:"status"`                     // 健康状态
	RetryCount              int                `json:"retry_count"`                // 重试次数
	NextAvailableAt         *time.Time         `json:"next_available_at"`          // 下次可用时间
	BackoffDuration         int64              `json:"backoff_duration"`           // 当前退避时长（秒）
	LastError               string             `json:"last_error"`                 // 最后错误信息
	LastErrorCode           int                `json:"last_error_code"`            // 最后错误码
	LastErrorMessage        string             `json:"last_error_message"`         // 最后错误展示消息
	LastStructuredErrorCode string             `json:"last_structured_error_code"` // 最后稳定错误码
	LastHTTPStatus          *int               `json:"last_http_status"`           // 最后 HTTP 状态码
	LastErrorFrom           string             `json:"last_error_from"`            // 最后错误来源
	LastCauseMessage        string             `json:"last_cause_message"`         // 最后根因文本
	LastCheckAt             time.Time          `json:"last_check_at"`              // 最后检查时间
	LastSuccessAt           *time.Time         `json:"last_success_at"`            // 最后成功时间
	SuccessCount            int                `json:"success_count"`              // 成功次数
	ErrorCount              int                `json:"error_count"`                // 错误次数
}

// APIKeyHealthListResponse 密钥健康列表响应
type APIKeyHealthListResponse struct {
	Items    []APIKeyHealthItem `json:"items"`     // 密钥健康列表
	Total    int                `json:"total"`     // 总数
	Page     int                `json:"page"`      // 当前页码
	PageSize int                `json:"page_size"` // 每页大小
}

// ModelHealthItem 单个模型健康状态项
type ModelHealthItem struct {
	ModelID                 uint               `json:"model_id"`                   // 模型 ID
	ModelName               string             `json:"model_name"`                 // 模型名称
	ModelAlias              string             `json:"model_alias"`                // 模型别名
	Status                  types.HealthStatus `json:"status"`                     // 健康状态
	RetryCount              int                `json:"retry_count"`                // 重试次数
	NextAvailableAt         *time.Time         `json:"next_available_at"`          // 下次可用时间
	BackoffDuration         int64              `json:"backoff_duration"`           // 当前退避时长（秒）
	LastError               string             `json:"last_error"`                 // 最后错误信息
	LastErrorCode           int                `json:"last_error_code"`            // 最后错误码
	LastErrorMessage        string             `json:"last_error_message"`         // 最后错误展示消息
	LastStructuredErrorCode string             `json:"last_structured_error_code"` // 最后稳定错误码
	LastHTTPStatus          *int               `json:"last_http_status"`           // 最后 HTTP 状态码
	LastErrorFrom           string             `json:"last_error_from"`            // 最后错误来源
	LastCauseMessage        string             `json:"last_cause_message"`         // 最后根因文本
	LastCheckAt             time.Time          `json:"last_check_at"`              // 最后检查时间
	LastSuccessAt           *time.Time         `json:"last_success_at"`            // 最后成功时间
	SuccessCount            int                `json:"success_count"`              // 成功次数
	ErrorCount              int                `json:"error_count"`                // 错误次数
}

// ModelHealthListResponse 模型健康列表响应
type ModelHealthListResponse struct {
	Items    []ModelHealthItem `json:"items"`     // 模型健康列表
	Total    int               `json:"total"`     // 总数
	Page     int               `json:"page"`      // 当前页码
	PageSize int               `json:"page_size"` // 每页大小
}

// IssueItem 单个异常资源项
type IssueItem struct {
	ResourceType types.ResourceType `json:"resource_type"` // 资源类型
	ResourceID   uint               `json:"resource_id"`   // 资源 ID
	ResourceName string             `json:"resource_name"` // 资源名称
	PlatformName *string            `json:"platform_name"` // 所属平台名称（仅密钥和模型类型）
	Status       types.HealthStatus `json:"status"`        // 资源状态
	LastCheckAt  time.Time          `json:"last_check_at"` // 最后检查
	LastError    string             `json:"last_error"`    // 最后错误
}

// IssuesListResponse 异常资源列表响应
type IssuesListResponse struct {
	Items []IssueItem `json:"items"` // 异常资源列表
}
