package stats

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/MeowSalty/pinai/handlers/query"
	"github.com/MeowSalty/pinai/handlers/response"
	"github.com/MeowSalty/pinai/services/stats"
)

// StatsHandlerInterface 定义统计处理器接口
type StatsHandlerInterface interface {
	// GetDashboard 获取仪表盘聚合数据
	GetDashboard(c *gin.Context)

	// ListRequestLogs 获取请求状态列表
	ListRequestLogs(c *gin.Context)

	// GetRealtime 获取实时数据
	GetRealtime(c *gin.Context)
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

// GetDashboard 处理仪表盘数据请求，路径为 GET /api/stats/dashboard。
// 通过单次查询获取所有仪表盘数据，包括概览、排名和趋势，避免多次数据库查询。
// 成功时返回 200 和完整的仪表盘数据，参数无效时返回 400。
//
// @Summary      获取仪表盘数据
// @Description  单次查询获取仪表盘所有数据，包括概览、排名（模型/平台调用和用量）、趋势分析
// @Tags         统计
// @Accept       json
// @Produce      json
// @Param        range      query     string  false  "时间范围"  Enums(24h, 7d, 30d)  default(24h)
// @Success      200        {object}  stats.DashboardResponse
// @Failure      400        {object}  gin.H  "无效的时间范围参数"
// @Failure      500        {object}  gin.H  "服务器内部错误"
// @Router       /api/stats/dashboard [get]
func (h *StatsHandler) GetDashboard(c *gin.Context) {
	rangeStr := c.DefaultQuery("range", string(stats.TrendRange24h))
	trendRange := stats.TrendRange(rangeStr)

	switch trendRange {
	case stats.TrendRange24h, stats.TrendRange7d, stats.TrendRange30d:
		// 参数有效
	default:
		response.BadRequest(c, "无效的时间范围参数，可选值：24h, 7d, 30d")
		return
	}

	dashboard, err := h.StatsService.GetDashboard(c.Request.Context(), trendRange)
	if err != nil {
		response.InternalError(c, "获取仪表盘数据失败："+err.Error())
		return
	}

	c.JSON(http.StatusOK, dashboard)
}

// GetRealtime 获取实时数据
//
// 返回值：
//   - 成功：实时数据
//   - 失败：错误信息
func (h *StatsHandler) GetRealtime(c *gin.Context) {
	realtime, err := h.StatsService.GetRealtime(c.Request.Context())
	if err != nil {
		response.InternalError(c, "获取实时数据失败："+err.Error())
		return
	}

	c.JSON(http.StatusOK, realtime)
}

// ListRequestLogs 获取请求状态列表
//
// 查询参数：
//   - start_time: 开始时间 (可选，支持 RFC3339 格式或 Unix 时间戳毫秒格式)
//   - end_time: 结束时间 (可选，支持 RFC3339 格式或 Unix 时间戳毫秒格式)
//   - success: 成功状态 (可选)
//   - is_stream: 是否流式请求 (可选)
//   - is_native: 是否原生请求 (可选)
//   - model_name: 模型名称 (可选)
//   - platform_id: 平台 ID (可选)
//   - page: 页码 (默认为 1)
//   - page_size: 每页大小 (默认为 10, 最大 100)
//
// 返回值：
//   - 成功：请求状态列表和分页信息
//   - 失败：错误信息
//
// @Summary      获取请求状态列表
// @Description  支持按时间、成功状态、流式/原生类型筛选请求日志，并提供分页
// @Tags         统计
// @Accept       json
// @Produce      json
// @Param        start_time    query     string  false  "开始时间 (RFC3339 或 Unix 毫秒时间戳)"
// @Param        end_time      query     string  false  "结束时间 (RFC3339 或 Unix 毫秒时间戳)"
// @Param        success       query     bool    false  "成功状态"
// @Param        is_stream     query     bool    false  "是否流式请求"
// @Param        is_native     query     bool    false  "是否原生请求"
// @Param        model_name    query     string  false  "模型名称"
// @Param        platform_id   query     int     false  "平台 ID"
// @Param        page          query     int     false  "页码"  default(1)
// @Param        page_size     query     int     false  "每页大小"  default(10)
// @Success      200           {object}  map[string]interface{}
// @Failure      400           {object}  gin.H  "参数错误"
// @Failure      500           {object}  gin.H  "服务器内部错误"
// @Router       /api/stats/requests [get]
func (h *StatsHandler) ListRequestLogs(c *gin.Context) {
	// 解析查询参数
	var opts stats.ListRequestLogsOptions

	// 解析时间范围参数
	if startTimeStr := c.Query("start_time"); startTimeStr != "" {
		startTime, err := parseTime(startTimeStr)
		if err != nil {
			response.BadRequest(c, "开始时间格式错误")
			return
		}
		opts.StartTime = &startTime
	}

	if endTimeStr := c.Query("end_time"); endTimeStr != "" {
		endTime, err := parseTime(endTimeStr)
		if err != nil {
			response.BadRequest(c, "结束时间格式错误")
			return
		}
		opts.EndTime = &endTime
	}

	// 解析结果状态参数
	success, err := query.OptionalBool(c, "success")
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	opts.Success = success

	// 解析是否流式请求参数
	isStream, err := query.OptionalBool(c, "is_stream")
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	opts.IsStream = isStream

	// 解析是否原生请求参数
	isNative, err := query.OptionalBool(c, "is_native")
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	opts.IsNative = isNative

	// 解析模型名称参数
	if modelName := c.Query("model_name"); modelName != "" {
		opts.ModelName = &modelName
	}

	// 解析平台 ID 参数
	platformID, err := query.OptionalUint(c, "platform_id")
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	opts.PlatformID = platformID

	// 解析分页参数
	page, pageSize, err := query.Pagination(c)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	opts.Page = page
	opts.PageSize = pageSize

	// 调用服务获取数据
	result, count, err := h.StatsService.ListRequestLogs(c.Request.Context(), opts)
	if err != nil {
		response.InternalError(c, "获取请求状态列表失败："+err.Error())
		return
	}

	// 构造响应
	response := map[string]interface{}{
		"data":  result,
		"count": count,
	}

	c.JSON(http.StatusOK, response)
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
