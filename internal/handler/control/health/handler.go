package health

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/MeowSalty/pinai/handlers/query"
	"github.com/MeowSalty/pinai/internal/handler/response"
	"github.com/MeowSalty/pinai/services/health"
)

// Handler 健康状态处理器结构体
type Handler struct {
	healthService health.Service
	logger        *slog.Logger
}

// NewHandler 创建健康状态处理器实例
//
// 参数：
//
//	healthService - 健康服务接口实例
//	logger - 日志记录器
//
// 返回值：
//
//	*Handler - 健康状态处理器实例指针
func NewHandler(healthService health.Service, logger *slog.Logger) *Handler {
	if logger == nil {
		logger = slog.Default()
	}

	return &Handler{
		healthService: healthService,
		logger:        logger.With("component", "health_handler"),
	}
}

// GetHealthSummary 获取健康状态统计
//
// 返回值：
//
//	成功 - 健康状态统计数据
//	失败 - 错误信息
func (h *Handler) GetHealthSummary(c *gin.Context) {
	logger := h.logger.With(
		"operation", "get_summary",
		"method", c.Request.Method,
		"path", c.FullPath(),
	)

	summary, err := h.healthService.GetHealthSummary(c.Request.Context())
	if err != nil {
		logger.Error("获取健康状态统计失败", "error", err)
		response.InternalError(c, "获取健康状态统计失败："+err.Error())
		return
	}

	logger.Debug("获取健康状态统计成功")

	c.JSON(http.StatusOK, summary)
}

// GetPlatformHealthList 获取平台健康列表
//
// 查询参数：
//
//	page - 页码，默认为 1
//	page_size - 每页大小，默认为 10，最大 100
//
// 返回值：
//
//	成功 - 平台健康列表数据
//	失败 - 错误信息
func (h *Handler) GetPlatformHealthList(c *gin.Context) {
	logger := h.logger.With(
		"operation", "list_platforms",
		"method", c.Request.Method,
		"path", c.FullPath(),
	)

	page, pageSize, err := query.Pagination(c)
	if err != nil {
		logger.Warn("分页参数解析失败",
			"page_raw", c.Query("page"),
			"page_size_raw", c.Query("page_size"),
			"error", err)
		response.BadRequest(c, err.Error())
		return
	}

	logger = logger.With(
		"page", page,
		"page_size", pageSize,
	)

	// 调用服务获取平台健康列表
	result, err := h.healthService.GetPlatformHealthList(c.Request.Context(), page, pageSize)
	if err != nil {
		logger.Error("获取平台健康列表失败", "error", err)
		response.InternalError(c, "获取平台健康列表失败："+err.Error())
		return
	}

	logger.Debug("获取平台健康列表成功", "total", result.Total, "item_count", len(result.Items))

	c.JSON(http.StatusOK, result)
}

// GetAPIKeyHealthList 获取密钥健康列表
//
// 查询参数：
//
//	page - 页码，默认为 1
//	page_size - 每页大小，默认为 10，最大 100
//
// 返回值：
//
//	成功 - 密钥健康列表数据
//	失败 - 错误信息
func (h *Handler) GetAPIKeyHealthList(c *gin.Context) {
	logger := h.logger.With(
		"operation", "list_keys",
		"method", c.Request.Method,
		"path", c.FullPath(),
	)

	page, pageSize, err := query.Pagination(c)
	if err != nil {
		logger.Warn("分页参数解析失败",
			"page_raw", c.Query("page"),
			"page_size_raw", c.Query("page_size"),
			"error", err)
		response.BadRequest(c, err.Error())
		return
	}

	logger = logger.With(
		"page", page,
		"page_size", pageSize,
	)

	// 调用服务获取密钥健康列表
	result, err := h.healthService.GetAPIKeyHealthList(c.Request.Context(), page, pageSize)
	if err != nil {
		logger.Error("获取密钥健康列表失败", "error", err)
		response.InternalError(c, "获取密钥健康列表失败："+err.Error())
		return
	}

	logger.Debug("获取密钥健康列表成功", "total", result.Total, "item_count", len(result.Items))

	c.JSON(http.StatusOK, result)
}

// GetModelHealthList 获取模型健康列表
//
// 查询参数：
//
//	page - 页码，默认为 1
//	page_size - 每页大小，默认为 10，最大 100
//
// 返回值：
//
//	成功 - 模型健康列表数据
//	失败 - 错误信息
func (h *Handler) GetModelHealthList(c *gin.Context) {
	logger := h.logger.With(
		"operation", "list_models",
		"method", c.Request.Method,
		"path", c.FullPath(),
	)

	page, pageSize, err := query.Pagination(c)
	if err != nil {
		logger.Warn("分页参数解析失败",
			"page_raw", c.Query("page"),
			"page_size_raw", c.Query("page_size"),
			"error", err)
		response.BadRequest(c, err.Error())
		return
	}

	logger = logger.With(
		"page", page,
		"page_size", pageSize,
	)

	// 调用服务获取模型健康列表
	result, err := h.healthService.GetModelHealthList(c.Request.Context(), page, pageSize)
	if err != nil {
		logger.Error("获取模型健康列表失败", "error", err)
		response.InternalError(c, "获取模型健康列表失败："+err.Error())
		return
	}

	logger.Debug("获取模型健康列表成功", "total", result.Total, "item_count", len(result.Items))

	c.JSON(http.StatusOK, result)
}

// GetIssues 获取异常资源列表
//
// 返回值：
//
//	成功 - 异常资源列表数据
//	失败 - 错误信息
func (h *Handler) GetIssues(c *gin.Context) {
	logger := h.logger.With(
		"operation", "list_issues",
		"method", c.Request.Method,
		"path", c.FullPath(),
	)

	result, err := h.healthService.GetIssues(c.Request.Context())
	if err != nil {
		logger.Error("获取异常资源列表失败", "error", err)
		response.InternalError(c, "获取异常资源列表失败："+err.Error())
		return
	}

	logger.Debug("获取异常资源列表成功", "item_count", len(result.Items))

	c.JSON(http.StatusOK, result)
}
