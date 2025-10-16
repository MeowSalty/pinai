package provider

import (
	"context"
	"fmt"
	"strconv"

	"github.com/MeowSalty/pinai/services/provider"

	"github.com/gofiber/fiber/v2"
)

// CreateProvider godoc
// @Summary      创建一个新的供应方
// @Description  创建一个新的供应方，包括平台、模型和密钥
// @Tags         providers
// @Accept       json
// @Produce      json
// @Param        request  body      provider.CreateRequest  true  "创建供应方的请求体"
// @Success      201      {object}  types.Platform                    "创建成功的供应方信息"
// @Failure      400      {object}  map[string]interface{}            "请求参数错误"
// @Failure      500      {object}  map[string]interface{}            "服务器内部错误"
// @Router       /api/providers [post]
func (h *Handler) CreateProvider(c *fiber.Ctx) error {
	var req provider.CreateRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("无法解析请求体: %v", err),
		})
	}

	ctx := context.Background()
	platform, err := h.service.CreateProvider(ctx, req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("创建供应方失败: %v", err),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(platform)
}

// DeleteProvider godoc
// @Summary      删除指定供应方
// @Description  删除指定供应方 (平台) (将级联删除模型和密钥)
// @Tags         providers
// @Produce      json
// @Param        id   path      int  true  "供应方 ID"
// @Success      200  {object}  map[string]interface{}            "删除成功消息"
// @Failure      400  {object}  map[string]interface{}            "请求参数错误"
// @Failure      404  {object}  map[string]interface{}            "供应方未找到"
// @Failure      500  {object}  map[string]interface{}            "服务器内部错误"
// @Router       /api/providers/{id} [delete]
func (h *Handler) DeleteProvider(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "无效的供应方 ID",
		})
	}

	ctx := context.Background()
	err = h.service.DeleteProvider(ctx, uint(id))
	if err != nil {
		// 检查错误类型
		if err.Error() == fmt.Sprintf("未找到 ID 为 %d 的供应方", id) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "供应方未找到",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("删除供应方失败: %v", err),
		})
	}

	return c.JSON(fiber.Map{
		"message": "供应方及其关联资源已成功删除",
	})
}
