package provider

import (
	"context"
	"fmt"
	"strconv"

	"github.com/MeowSalty/pinai/database/types"

	"github.com/gofiber/fiber/v2"
)

// PlatformWithHealth 带健康状态的平台响应
type PlatformWithHealth struct {
	*types.Platform
	HealthStatus *types.HealthStatus `json:"health_status,omitempty"`
}

// CreatePlatform godoc
// @Summary      创建一个新的平台
// @Description  创建一个新的平台
// @Tags         platforms
// @Accept       json
// @Produce      json
// @Param        request  body      types.Platform  true  "创建平台的请求体"
// @Success      201      {object}  types.Platform                    "创建成功的平台信息"
// @Failure      400      {object}  map[string]interface{}            "请求参数错误"
// @Failure      500      {object}  map[string]interface{}            "服务器内部错误"
// @Router       /api/platforms [post]
func (h *Handler) CreatePlatform(c *fiber.Ctx) error {
	var platform types.Platform
	if err := c.BodyParser(&platform); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("无法解析请求体: %v", err),
		})
	}

	ctx := context.Background()
	createdPlatform, err := h.service.CreatePlatform(ctx, platform)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("创建平台失败: %v", err),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(createdPlatform)
}

// GetPlatforms godoc
// @Summary      获取所有平台列表
// @Description  获取所有平台列表，可通过 include=health 参数包含健康状态
// @Tags         platforms
// @Produce      json
// @Param        include  query     string  false  "包含额外信息，支持 health"
// @Success      200  {array}   PlatformWithHealth                "平台列表"
// @Failure      500  {object}  map[string]interface{}            "服务器内部错误"
// @Router       /api/platforms [get]
func (h *Handler) GetPlatforms(c *fiber.Ctx) error {
	ctx := context.Background()
	platforms, err := h.service.GetPlatforms(ctx)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("获取平台列表失败: %v", err),
		})
	}

	// 检查是否需要包含健康状态
	if c.Query("include") == "health" {
		storage := h.healthService.GetStorage()
		result := make([]PlatformWithHealth, len(platforms))
		for i, p := range platforms {
			result[i].Platform = p
			if health, _ := storage.Get(types.ResourceTypePlatform, p.ID); health != nil {
				result[i].HealthStatus = &health.Status
			} else {
				// 没有健康数据时使用未知状态
				unknownStatus := types.HealthStatusUnknown
				result[i].HealthStatus = &unknownStatus
			}
		}
		return c.JSON(result)
	}

	return c.JSON(platforms)
}

// GetPlatform godoc
// @Summary      获取指定平台详情
// @Description  获取指定平台详情
// @Tags         platforms
// @Produce      json
// @Param        id   path      int  true  "平台 ID"
// @Success      200  {object}  types.Platform                    "平台详情"
// @Failure      400  {object}  map[string]interface{}            "请求参数错误"
// @Failure      404  {object}  map[string]interface{}            "平台未找到"
// @Failure      500  {object}  map[string]interface{}            "服务器内部错误"
// @Router       /api/platforms/{id} [get]
func (h *Handler) GetPlatform(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "无效的平台 ID",
		})
	}

	ctx := context.Background()
	platform, err := h.service.GetPlatform(ctx, uint(id))
	if err != nil {
		// 检查错误类型，如果未找到则返回 404
		if err.Error() == fmt.Sprintf("未找到 ID 为 %d 的平台", id) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "平台未找到",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("获取平台详情失败: %v", err),
		})
	}

	return c.JSON(platform)
}

// UpdatePlatform godoc
// @Summary      更新指定平台信息
// @Description  更新指定平台信息
// @Tags         platforms
// @Accept       json
// @Produce      json
// @Param        id       path      int                             true  "平台 ID"
// @Param        request  body      types.Platform                  true  "更新平台的请求体"
// @Success      200      {object}  types.Platform                    "更新后的平台信息"
// @Failure      400      {object}  map[string]interface{}            "请求参数错误"
// @Failure      404      {object}  map[string]interface{}            "平台未找到"
// @Failure      500      {object}  map[string]interface{}            "服务器内部错误"
// @Router       /api/platforms/{id} [put]
func (h *Handler) UpdatePlatform(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "无效的平台 ID",
		})
	}

	var platform types.Platform
	if err := c.BodyParser(&platform); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("无法解析请求体: %v", err),
		})
	}

	ctx := context.Background()
	updatedPlatform, err := h.service.UpdatePlatform(ctx, uint(id), platform)
	if err != nil {
		// 检查错误类型
		if err.Error() == fmt.Sprintf("未找到 ID 为 %d 的平台", id) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "平台未找到",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("更新平台失败: %v", err),
		})
	}

	return c.JSON(updatedPlatform)
}

// DeletePlatform godoc
// @Summary      删除指定平台
// @Description  删除指定平台及其所有关联的模型、密钥和关联关系
// @Tags         platforms
// @Produce      json
// @Param        id   path      int  true  "平台 ID"
// @Success      204  "删除成功"
// @Failure      400  {object}  map[string]interface{}            "请求参数错误"
// @Failure      404  {object}  map[string]interface{}            "平台未找到"
// @Failure      500  {object}  map[string]interface{}            "服务器内部错误"
// @Router       /api/platforms/{id} [delete]
func (h *Handler) DeletePlatform(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "无效的平台 ID",
		})
	}

	ctx := context.Background()
	err = h.service.DeletePlatform(ctx, uint(id))
	if err != nil {
		// 检查错误类型
		if err.Error() == fmt.Sprintf("未找到 ID 为 %d 的平台", id) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "平台未找到",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("删除平台失败: %v", err),
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// EnablePlatformHealth godoc
// @Summary      启用/恢复平台健康状态
// @Description  删除平台的健康记录，让系统重新评估健康状态
// @Tags         platforms
// @Produce      json
// @Param        id   path      int  true  "平台 ID"
// @Success      200  {object}  map[string]interface{}  "操作成功"
// @Failure      400  {object}  map[string]interface{}  "请求参数错误"
// @Failure      404  {object}  map[string]interface{}  "平台未找到"
// @Failure      500  {object}  map[string]interface{}  "服务器内部错误"
// @Router       /api/platforms/{id}/health/enable [post]
func (h *Handler) EnablePlatformHealth(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "无效的平台 ID",
		})
	}

	// 验证平台是否存在
	ctx := context.Background()
	_, err = h.service.GetPlatform(ctx, uint(id))
	if err != nil {
		if err.Error() == fmt.Sprintf("未找到 ID 为 %d 的平台", id) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "平台未找到",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("获取平台失败: %v", err),
		})
	}

	// 启用健康状态
	if err := h.healthService.EnableHealth(types.ResourceTypePlatform, uint(id)); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("启用平台健康状态失败: %v", err),
		})
	}

	return c.JSON(fiber.Map{
		"message":     "平台已启用",
		"platform_id": id,
		"status":      "unknown",
	})
}

// DisablePlatformHealth godoc
// @Summary      禁用平台健康状态
// @Description  将平台健康状态设置为不可用
// @Tags         platforms
// @Produce      json
// @Param        id   path      int  true  "平台 ID"
// @Success      200  {object}  map[string]interface{}  "操作成功"
// @Failure      400  {object}  map[string]interface{}  "请求参数错误"
// @Failure      404  {object}  map[string]interface{}  "平台未找到"
// @Failure      500  {object}  map[string]interface{}  "服务器内部错误"
// @Router       /api/platforms/{id}/health/disable [post]
func (h *Handler) DisablePlatformHealth(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "无效的平台 ID",
		})
	}

	// 验证平台是否存在
	ctx := context.Background()
	_, err = h.service.GetPlatform(ctx, uint(id))
	if err != nil {
		if err.Error() == fmt.Sprintf("未找到 ID 为 %d 的平台", id) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "平台未找到",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("获取平台失败: %v", err),
		})
	}

	// 禁用健康状态
	if err := h.healthService.DisableHealth(types.ResourceTypePlatform, uint(id)); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("禁用平台健康状态失败: %v", err),
		})
	}

	return c.JSON(fiber.Map{
		"message":     "平台已禁用",
		"platform_id": id,
		"status":      "unavailable",
	})
}
