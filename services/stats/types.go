package stats

import (
	"log/slog"
	"time"
)

// service 是 ServiceInterface 接口的具体实现
type service struct {
	logger *slog.Logger
}

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
	RPM               int64 `json:"rpm"`                // 每分钟请求数
	ActiveConnections int64 `json:"active_connections"` // 当前活动连接数
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

// TrendRange 定义趋势分析的时间范围类型
type TrendRange string

const (
	// TrendRange24h 表示最近 24 小时
	TrendRange24h TrendRange = "24h"
	// TrendRange7d 表示最近 7 天
	TrendRange7d TrendRange = "7d"
	// TrendRange30d 表示最近 30 天
	TrendRange30d TrendRange = "30d"
)

// TrendDataPoint 定义单个趋势数据点
type TrendDataPoint struct {
	Timestamp    time.Time `json:"timestamp"`     // 数据点时间戳
	RequestCount int64     `json:"request_count"` // 请求数
	TotalTokens  int64     `json:"total_tokens"`  // Token 用量
}

// TrendSummary 定义趋势汇总统计
type TrendSummary struct {
	TotalRequests       int64   `json:"total_requests"`         // 总请求数
	TotalTokens         int64   `json:"total_tokens"`           // 总 Token 用量
	AvgRequestsPerPoint float64 `json:"avg_requests_per_point"` // 平均每点请求数
	AvgTokensPerPoint   float64 `json:"avg_tokens_per_point"`   // 平均每点 Token 用量
}

// TrendResponse 定义趋势分析响应结构
type TrendResponse struct {
	Range       string           `json:"range"`       // 时间范围
	Granularity string           `json:"granularity"` // 颗粒度
	DataPoints  []TrendDataPoint `json:"data_points"` // 数据点列表
	Summary     TrendSummary     `json:"summary"`     // 汇总统计
}

// DashboardRequest 仪表盘数据请求参数
type DashboardRequest struct {
	Range TrendRange `json:"range"` // 时间范围：24h/7d/30d
}

// DashboardResponse 仪表盘数据响应
type DashboardResponse struct {
	Range    string            `json:"range"`    // 时间范围
	Overview DashboardOverview `json:"overview"` // 概览数据
	Ranks    DashboardRanks    `json:"ranks"`    // 排名数据
	Trend    *TrendResponse    `json:"trend"`    // 趋势数据
}

// DashboardOverview 仪表盘概览数据
type DashboardOverview struct {
	TotalRequests         int64   `json:"total_requests"`          // 总请求量
	SuccessRate           float64 `json:"success_rate"`            // 成功率
	AvgFirstByteTime      float64 `json:"avg_first_byte"`          // 平均首字时间（微秒）
	ActiveModels          int     `json:"active_models"`           // 活跃模型数（时间范围内有请求的去重模型）
	ActivePlatforms       int     `json:"active_platforms"`        // 活跃平台数（时间范围内有请求的去重平台）
	TotalPromptTokens     int64   `json:"total_prompt_tokens"`     // 总输入 Token
	TotalCompletionTokens int64   `json:"total_completion_tokens"` // 总输出 Token
	TotalTokens           int64   `json:"total_tokens"`            // 总 Token
}

// DashboardRanks 仪表盘排名数据
type DashboardRanks struct {
	ModelCall     []ModelCallRankItem     `json:"model_call"`     // 模型调用排名
	PlatformCall  []PlatformCallRankItem  `json:"platform_call"`  // 平台调用排名
	ModelUsage    []ModelUsageRankItem    `json:"model_usage"`    // 模型用量排名
	PlatformUsage []PlatformUsageRankItem `json:"platform_usage"` // 平台用量排名
}

// ListRequestLogsOptions 定义了获取请求状态列表的筛选选项
type ListRequestLogsOptions struct {
	// 时间范围筛选
	StartTime   *time.Time `json:"start_time,omitempty"`
	EndTime     *time.Time `json:"end_time,omitempty"`
	Success     *bool      `json:"success,omitempty"`
	RequestType *string    `json:"request_type,omitempty"`
	ModelName   *string    `json:"model_name,omitempty"`
	PlatformID  *uint      `json:"platform_id,omitempty"`
	Page        int        `json:"page"`
	PageSize    int        `json:"page_size"`
}
