package stats

import (
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/MeowSalty/pinai/services"
)

// StatsHandlerInterface 定义统计处理器接口
type StatsHandlerInterface interface {
	// GetOverview 获取全局概览数据
	GetOverview(c *fiber.Ctx) error

	// ListRequestStats 获取请求状态列表
	ListRequestStats(c *fiber.Ctx) error
}

// StatsHandler 统计处理器结构体
type StatsHandler struct {
	StatsService services.StatsServiceInterface
}

// NewStatsHandler 创建统计处理器实例
//
// 参数：
//   - statsService: 统计服务接口实例
//
// 返回值：
//   - StatsHandlerInterface: 统计处理器接口实例
func NewStatsHandler(statsService services.StatsServiceInterface) StatsHandlerInterface {
	return &StatsHandler{
		StatsService: statsService,
	}
}

// GetOverview 获取全局概览数据
//
// 返回值：
//   - 成功：全局概览数据
//   - 失败：错误信息
func (h *StatsHandler) GetOverview(c *fiber.Ctx) error {
	overview, err := h.StatsService.GetOverview(c.Context())
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "获取统计概览数据失败："+err.Error())
	}

	return c.JSON(overview)
}

// ListRequestStats 获取请求状态列表
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
func (h *StatsHandler) ListRequestStats(c *fiber.Ctx) error {
	// 解析查询参数
	var opts services.ListRequestStatsOptions

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
	result, count, err := h.StatsService.ListRequestStats(c.Context(), opts)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "获取请求状态列表失败："+err.Error())
	}

	// 构造响应
	response := map[string]interface{}{
		"data":        result,
		"count":       count,
		"page":        opts.Page,
		"page_size":   opts.PageSize,
		"total_pages": (count + int64(opts.PageSize) - 1) / int64(opts.PageSize),
	}

	return c.JSON(response)
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
