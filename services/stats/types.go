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

// ModelCallRankItem 定义了模型调用排名项
type ModelCallRankItem struct {
	ModelName    string  `json:"model_name"`    // 模型名称
	RequestCount int64   `json:"request_count"` // 请求数量
	SuccessRate  float64 `json:"success_rate"`  // 成功率
	Percentage   float64 `json:"percentage"`    // 占比
}

// PlatformCallRankItem 定义了平台调用排名项
type PlatformCallRankItem struct {
	PlatformName string  `json:"platform_name"` // 平台名称
	RequestCount int64   `json:"request_count"` // 请求数量
	SuccessRate  float64 `json:"success_rate"`  // 成功率
	Percentage   float64 `json:"percentage"`    // 占比
}

// ModelUsageRankItem 定义了模型用量排名项
type ModelUsageRankItem struct {
	ModelName        string  `json:"model_name"`        // 模型名称
	TotalTokens      int64   `json:"total_tokens"`      // 总 Token 数
	PromptTokens     int64   `json:"prompt_tokens"`     // 输入 Token 数
	CompletionTokens int64   `json:"completion_tokens"` // 输出 Token 数
	Percentage       float64 `json:"percentage"`        // 占比
}

// PlatformUsageRankItem 定义了平台用量排名项
type PlatformUsageRankItem struct {
	PlatformName     string  `json:"platform_name"`     // 平台名称
	TotalTokens      int64   `json:"total_tokens"`      // 总 Token 数
	PromptTokens     int64   `json:"prompt_tokens"`     // 输入 Token 数
	CompletionTokens int64   `json:"completion_tokens"` // 输出 Token 数
	Percentage       float64 `json:"percentage"`        // 占比
}

// ModelCallRankResponse 定义了模型调用排名响应结构
type ModelCallRankResponse struct {
	TotalRequests int64               `json:"total_requests"` // 总请求量
	Models        []ModelCallRankItem `json:"models"`         // 模型排名列表
}

// PlatformCallRankResponse 定义了平台调用排名响应结构
type PlatformCallRankResponse struct {
	TotalRequests int64                  `json:"total_requests"` // 总请求量
	Platforms     []PlatformCallRankItem `json:"platforms"`      // 平台排名列表
}

// ModelUsageRankResponse 定义了模型用量排名响应结构
type ModelUsageRankResponse struct {
	TotalTokens int64                `json:"total_tokens"` // 总 Token 数
	Models      []ModelUsageRankItem `json:"models"`       // 模型用量排名列表
}

// PlatformUsageRankResponse 定义了平台用量排名响应结构
type PlatformUsageRankResponse struct {
	TotalTokens int64                   `json:"total_tokens"` // 总 Token 数
	Platforms   []PlatformUsageRankItem `json:"platforms"`    // 平台用量排名列表
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
