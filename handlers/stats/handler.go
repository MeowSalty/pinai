package stats

import (
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/MeowSalty/pinai/services/stats"
)

// StatsHandlerInterface 定义统计处理器接口
type StatsHandlerInterface interface {
	// GetOverview 获取全局概览数据
	GetOverview(c *fiber.Ctx) error

	// ListRequestLogs 获取请求状态列表
	ListRequestLogs(c *fiber.Ctx) error

	// GetRealtime 获取实时数据
	GetRealtime(c *fiber.Ctx) error

	// GetModelCallRank 获取模型调用排名前 5
	GetModelCallRank(c *fiber.Ctx) error

	// GetPlatformCallRank 获取平台调用排名前 5
	GetPlatformCallRank(c *fiber.Ctx) error

	// GetModelUsageRank 获取模型用量排名前 5
	GetModelUsageRank(c *fiber.Ctx) error

	// GetPlatformUsageRank 获取平台用量排名前 5
	GetPlatformUsageRank(c *fiber.Ctx) error
}

// StatsHandler 统计处理器结构体
type StatsHandler struct {
	StatsService stats.Service
}

// NewStatsHandler 创建统计处理器实例
//
// 参数：
//   - statsService: 统计服务接口实例
//
// 返回值：
//   - StatsHandlerInterface: 统计处理器接口实例
func NewStatsHandler(statsService stats.Service) StatsHandlerInterface {
	return &StatsHandler{
		StatsService: statsService,
	}
}

// GetOverview 获取全局概览数据
//
// 查询参数：
//   - duration: 时间范围 (可选，支持 24h, 7d 等格式，默认为 24h)
//
// 返回值：
//   - 成功：全局概览数据
//   - 失败：错误信息
func (h *StatsHandler) GetOverview(c *fiber.Ctx) error {
	durationStr := c.Query("duration", "24h")
	duration, err := time.ParseDuration(durationStr)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "时间范围格式错误，请使用如 24h, 5m 等格式")
	}

	overview, err := h.StatsService.GetOverview(c.Context(), duration)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "获取统计概览数据失败："+err.Error())
	}

	return c.JSON(overview)
}

// GetRealtime 获取实时数据
//
// 返回值：
//   - 成功：实时数据
//   - 失败：错误信息
func (h *StatsHandler) GetRealtime(c *fiber.Ctx) error {
	realtime, err := h.StatsService.GetRealtime(c.Context())
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "获取实时数据失败："+err.Error())
	}

	return c.JSON(realtime)
}

// ListRequestLogs 获取请求状态列表
//
// 查询参数：
//   - start_time: 开始时间 (可选，支持 RFC3339 格式或 Unix 时间戳毫秒格式)
//   - end_time: 结束时间 (可选，支持 RFC3339 格式或 Unix 时间戳毫秒格式)
//   - success: 成功状态 (可选)
//   - request_type: 请求类型 (可选)
//   - model_name: 模型名称 (可选)
//   - page: 页码 (默认为 1)
//   - page_size: 每页大小 (默认为 10, 最大 100)
//
// 返回值：
//   - 成功：请求状态列表和分页信息
//   - 失败：错误信息
func (h *StatsHandler) ListRequestLogs(c *fiber.Ctx) error {
	// 解析查询参数
	var opts stats.ListRequestLogsOptions

	// 解析时间范围参数
	if startTimeStr := c.Query("start_time"); startTimeStr != "" {
		startTime, err := parseTime(startTimeStr)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "开始时间格式错误")
		}
		opts.StartTime = &startTime
	}

	if endTimeStr := c.Query("end_time"); endTimeStr != "" {
		endTime, err := parseTime(endTimeStr)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "结束时间格式错误")
		}
		opts.EndTime = &endTime
	}

	// 解析结果状态参数
	if successStr := c.Query("success"); successStr != "" {
		success := c.QueryBool("success")
		opts.Success = &success
	}

	// 解析请求类型参数
	if requestType := c.Query("request_type"); requestType != "" {
		opts.RequestType = &requestType
	}

	// 解析模型名称参数
	if modelName := c.Query("model_name"); modelName != "" {
		opts.ModelName = &modelName
	}

	// 解析分页参数
	page := c.QueryInt("page", 1)
	if page <= 0 {
		page = 1
	}

	pageSize := c.QueryInt("page_size", 10)
	if pageSize <= 0 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}

	opts.Page = page
	opts.PageSize = pageSize

	// 调用服务获取数据
	result, count, err := h.StatsService.ListRequestLogs(c.Context(), opts)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "获取请求状态列表失败："+err.Error())
	}

	// 构造响应
	response := map[string]interface{}{
		"data":  result,
		"count": count,
	}

	return c.JSON(response)
}

// GetModelCallRank 获取模型调用排名前 5
//
// 查询参数：
//   - duration: 时间范围 (可选，支持 24h, 7d 等格式，默认为 24h)
//
// 返回值：
//   - 成功：模型调用排名数据
//   - 失败：错误信息
func (h *StatsHandler) GetModelCallRank(c *fiber.Ctx) error {
	durationStr := c.Query("duration", "24h")
	duration, err := time.ParseDuration(durationStr)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "时间范围格式错误，请使用如 24h, 7d 等格式")
	}

	modelCallRank, err := h.StatsService.GetModelCallRank(c.Context(), duration)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "获取模型调用排名失败："+err.Error())
	}

	return c.JSON(modelCallRank)
}

// GetPlatformCallRank 获取平台调用排名前 5
//
// 查询参数：
//   - duration: 时间范围 (可选，支持 24h, 7d 等格式，默认为 24h)
//
// 返回值：
//   - 成功：平台调用排名数据
//   - 失败：错误信息
func (h *StatsHandler) GetPlatformCallRank(c *fiber.Ctx) error {
	durationStr := c.Query("duration", "24h")
	duration, err := time.ParseDuration(durationStr)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "时间范围格式错误，请使用如 24h, 7d 等格式")
	}

	platformCallRank, err := h.StatsService.GetPlatformCallRank(c.Context(), duration)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "获取平台调用排名失败："+err.Error())
	}

	return c.JSON(platformCallRank)
}

// GetModelUsageRank 获取模型用量排名前 5
//
// 查询参数：
//   - duration: 时间范围 (可选，支持 24h, 7d 等格式，默认为 24h)
//
// 返回值：
//   - 成功：模型用量排名数据
//   - 失败：错误信息
func (h *StatsHandler) GetModelUsageRank(c *fiber.Ctx) error {
	durationStr := c.Query("duration", "24h")
	duration, err := time.ParseDuration(durationStr)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "时间范围格式错误，请使用如 24h, 7d 等格式")
	}

	modelUsageRank, err := h.StatsService.GetModelUsageRank(c.Context(), duration)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "获取模型用量排名失败："+err.Error())
	}

	return c.JSON(modelUsageRank)
}

// GetPlatformUsageRank 获取平台用量排名前 5
//
// 查询参数：
//   - duration: 时间范围 (可选，支持 24h, 7d 等格式，默认为 24h)
//
// 返回值：
//   - 成功：平台用量排名数据
//   - 失败：错误信息
func (h *StatsHandler) GetPlatformUsageRank(c *fiber.Ctx) error {
	durationStr := c.Query("duration", "24h")
	duration, err := time.ParseDuration(durationStr)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "时间范围格式错误，请使用如 24h, 7d 等格式")
	}

	platformUsageRank, err := h.StatsService.GetPlatformUsageRank(c.Context(), duration)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "获取平台用量排名失败："+err.Error())
	}

	return c.JSON(platformUsageRank)
}

// parseTime 解析时间字符串，支持 RFC3339 格式和 Unix 时间戳 (毫秒)
//
// 参数：
//   - timeStr: 时间字符串，可以是 RFC3339 格式或 Unix 时间戳 (毫秒)
//
// 返回值：
//   - 成功：解析后的时间
//   - 失败：错误信息
func parseTime(timeStr string) (time.Time, error) {
	// 首先尝试解析 RFC3339 格式
	if t, err := time.Parse(time.RFC3339, timeStr); err == nil {
		return t, nil
	}

	// 如果不是 RFC3339 格式，则尝试解析为 Unix 时间戳 (毫秒)
	ts, err := strconv.ParseInt(timeStr, 10, 64)
	if err != nil {
		return time.Time{}, err
	}

	// 将毫秒时间戳转换为 time.Time 类型
	return time.Unix(0, ts*int64(time.Millisecond)), nil
}
