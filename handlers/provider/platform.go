package provider

import (
	"context"
	"fmt"
	"strconv"

	"github.com/MeowSalty/pinai/database/types"

	"github.com/gofiber/fiber/v2"
)

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
// @Description  获取所有平台列表
// @Tags         platforms
// @Produce      json
// @Success      200  {array}   types.Platform                    "平台列表"
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
