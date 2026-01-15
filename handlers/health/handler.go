package health

import (
	"fmt"
	"strconv"

	"github.com/gofiber/fiber/v2"

	"github.com/MeowSalty/pinai/database/types"
	"github.com/MeowSalty/pinai/services/health"
)

// Handler 健康状态处理器结构体
type Handler struct {
	healthService health.Service
}

// NewHandler 创建健康状态处理器实例
//
// 参数：
//
//	healthService - 健康服务接口实例
//
// 返回值：
//
//	*Handler - 健康状态处理器实例指针
func NewHandler(healthService health.Service) *Handler {
	return &Handler{healthService: healthService}
}

// GetHealthSummary 获取健康状态统计
//
// 返回值：
//
//	成功 - 健康状态统计数据
//	失败 - 错误信息
func (h *Handler) GetHealthSummary(c *fiber.Ctx) error {
	summary, err := h.healthService.GetHealthSummary(c.Context())
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "获取健康状态统计失败："+err.Error())
	}

	return c.JSON(summary)
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
func (h *Handler) GetModelHealthList(c *fiber.Ctx) error {
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

	// 调用服务获取模型健康列表
	result, err := h.healthService.GetModelHealthList(c.Context(), page, pageSize)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "获取模型健康列表失败："+err.Error())
	}

	return c.JSON(result)
}

// EnableModel 启用/恢复指定模型的健康状态
//
// 路径参数：
//
//	modelId - 模型 ID
//
// 返回值：
//
//	成功 - 操作成功消息
//	失败 - 错误信息
func (h *Handler) EnableModel(c *fiber.Ctx) error {
	modelId, err := strconv.ParseUint(c.Params("modelId"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "无效的模型 ID",
		})
	}

	// 启用健康状态
	if err := h.healthService.EnableHealth(types.ResourceTypeModel, uint(modelId)); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("启用模型健康状态失败: %v", err),
		})
	}

	return c.JSON(fiber.Map{
		"message":  "模型已启用",
		"model_id": modelId,
		"status":   "unknown",
	})
}
