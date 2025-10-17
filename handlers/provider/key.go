package provider

import (
	"context"
	"fmt"
	"strconv"

	"github.com/MeowSalty/pinai/database/types"

	"github.com/gofiber/fiber/v2"
)

// AddKeyToPlatform godoc
// @Summary      为指定平台添加新密钥
// @Description  为指定平台添加新密钥
// @Tags         keys
// @Accept       json
// @Produce      json
// @Param        platformId  path      int                             true  "平台 ID"
// @Param        request     body      types.APIKey                    true  "创建密钥的请求体"
// @Success      201         {object}  types.APIKey                      "创建成功的密钥信息 (不包含 value)"
// @Failure      400         {object}  map[string]interface{}            "请求参数错误"
// @Failure      404         {object}  map[string]interface{}            "平台未找到"
// @Failure      500         {object}  map[string]interface{}            "服务器内部错误"
// @Router       /api/platforms/{platformId}/keys [post]
func (h *Handler) AddKeyToPlatform(c *fiber.Ctx) error {
	platformId, err := strconv.ParseUint(c.Params("platformId"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "无效的平台 ID",
		})
	}

	var key types.APIKey
	if err := c.BodyParser(&key); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("无法解析请求体: %v", err),
		})
	}

	ctx := context.Background()
	createdKey, err := h.service.AddKeyToPlatform(ctx, uint(platformId), key)
	if err != nil {
		// 检查错误类型
		if err.Error() == fmt.Sprintf("未找到 ID 为 %d 的平台", platformId) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "平台未找到",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("为平台添加密钥失败: %v", err),
		})
	}

	// 出于安全考虑，不返回密钥值
	createdKey.Value = ""
	return c.Status(fiber.StatusCreated).JSON(createdKey)
}

// GetKeysByPlatform godoc
// @Summary      获取指定平台的所有密钥列表
// @Description  获取指定平台的所有密钥列表 (不包含密钥值)
// @Tags         keys
// @Produce      json
// @Param        platformId  path      int  true  "平台 ID"
// @Success      200         {array}   types.APIKey                      "密钥列表 (不包含 value)"
// @Failure      400         {object}  map[string]interface{}            "请求参数错误"
// @Failure      404         {object}  map[string]interface{}            "平台未找到"
// @Failure      500         {object}  map[string]interface{}            "服务器内部错误"
// @Router       /api/platforms/{platformId}/keys [get]
func (h *Handler) GetKeysByPlatform(c *fiber.Ctx) error {
	platformId, err := strconv.ParseUint(c.Params("platformId"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "无效的平台 ID",
		})
	}

	ctx := context.Background()
	keys, err := h.service.GetKeysByPlatform(ctx, uint(platformId))
	if err != nil {
		// 检查错误类型
		if err.Error() == fmt.Sprintf("未找到 ID 为 %d 的平台", platformId) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "平台未找到",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("获取平台密钥列表失败: %v", err),
		})
	}

	return c.JSON(keys)
}

// DeleteKey godoc
// @Summary      删除指定密钥
// @Description  删除指定密钥
// @Tags         keys
// @Produce      json
// @Param        platformId  path      int  true  "平台 ID"
// @Param        keyId       path      int  true  "密钥 ID"
// @Success      200         {object}  map[string]interface{}            "删除成功消息"
// @Failure      400         {object}  map[string]interface{}            "请求参数错误"
// @Failure      404         {object}  map[string]interface{}            "平台或密钥未找到"
// @Failure      500         {object}  map[string]interface{}            "服务器内部错误"
// @Router       /api/platforms/{platformId}/keys/{keyId} [delete]
func (h *Handler) DeleteKey(c *fiber.Ctx) error {
	keyId, err := strconv.ParseUint(c.Params("keyId"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "无效的密钥 ID",
		})
	}

	ctx := context.Background()
	err = h.service.DeleteKey(ctx, uint(keyId))
	if err != nil {
		if err.Error() == fmt.Sprintf("未找到 ID 为 %d 的密钥", keyId) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "密钥未找到",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("删除密钥失败: %v", err),
		})
	}

	return c.JSON(fiber.Map{
		"message": "密钥已成功删除",
	})
}

// UpdateKey godoc
// @Summary      更新指定密钥信息
// @Description  更新指定密钥信息
// @Tags         keys
// @Accept       json
// @Produce      json
// @Param        platformId  path      int                             true  "平台 ID"
// @Param        keyId       path      int                             true  "密钥 ID"
// @Param        request     body      types.APIKey                    true  "更新密钥的请求体"
// @Success      200         {object}  types.APIKey                      "更新后的密钥信息 (不包含 value)"
// @Failure      400         {object}  map[string]interface{}            "请求参数错误"
// @Failure      404         {object}  map[string]interface{}            "平台或密钥未找到"
// @Failure      500         {object}  map[string]interface{}            "服务器内部错误"
// @Router       /api/platforms/{platformId}/keys/{keyId} [put]
func (h *Handler) UpdateKey(c *fiber.Ctx) error {
	keyId, err := strconv.ParseUint(c.Params("keyId"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "无效的密钥 ID",
		})
	}

	var key types.APIKey
	if err := c.BodyParser(&key); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("无法解析请求体: %v", err),
		})
	}

	ctx := context.Background()
	updatedKey, err := h.service.UpdateKey(ctx, uint(keyId), key)
	if err != nil {
		// 检查错误类型
		if err.Error() == fmt.Sprintf("未找到 ID 为 %d 的密钥", keyId) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "密钥未找到",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("更新密钥失败: %v", err),
		})
	}

	// 出于安全考虑，不返回密钥值
	updatedKey.Value = ""
	return c.JSON(updatedKey)
}
