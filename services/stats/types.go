package stats

import "time"

// service 是 ServiceInterface 接口的具体实现
type service struct{}

// StatsOverviewResponse 定义了全局概览数据的响应结构
type StatsOverviewResponse struct {
	TotalRequests         int64   `json:"total_requests"`          // 总请求量
	SuccessRate           float64 `json:"success_rate"`            // 成功率
	AvgFirstByteTime      float64 `json:"avg_first_byte"`          // 平均首字时间 (微秒)
	TotalPromptTokens     int64   `json:"total_prompt_tokens"`     // 总输入 Token
	TotalCompletionTokens int64   `json:"total_completion_tokens"` // 总输出 Token
	TotalTokens           int64   `json:"total_tokens"`            // 总 Token
}

// StatsRealtimeResponse 定义了实时数据的响应结构
type StatsRealtimeResponse struct {
	RPM float64 `json:"rpm"` // 每分钟请求数
}

// ModelRankItem 定义了模型排名项
type ModelRankItem struct {
	ModelName    string  `json:"model_name"`    // 模型名称
	RequestCount int64   `json:"request_count"` // 请求数量
	SuccessRate  float64 `json:"success_rate"`  // 成功率
	Percentage   float64 `json:"percentage"`    // 占比
}

// PlatformRankItem 定义了平台排名项
type PlatformRankItem struct {
	PlatformName string  `json:"platform_name"` // 平台名称
	RequestCount int64   `json:"request_count"` // 请求数量
	SuccessRate  float64 `json:"success_rate"`  // 成功率
	Percentage   float64 `json:"percentage"`    // 占比
}

// ModelRankResponse 定义了模型排名响应结构
type ModelRankResponse struct {
	TotalRequests int64           `json:"total_requests"` // 总请求量
	Models        []ModelRankItem `json:"models"`         // 模型排名列表
}

// PlatformRankResponse 定义了平台排名响应结构
type PlatformRankResponse struct {
	TotalRequests int64              `json:"total_requests"` // 总请求量
	Platforms     []PlatformRankItem `json:"platforms"`      // 平台排名列表
}

// ListRequestLogsOptions 定义了获取请求状态列表的筛选选项
type ListRequestLogsOptions struct {
	// 时间范围筛选
	StartTime   *time.Time `json:"start_time,omitempty"`
	EndTime     *time.Time `json:"end_time,omitempty"`
	Success     *bool      `json:"success,omitempty"`
	RequestType *string    `json:"request_type,omitempty"`
	ModelName   *string    `json:"model_name,omitempty"`
	Page        int        `json:"page"`
	PageSize    int        `json:"page_size"`
}
