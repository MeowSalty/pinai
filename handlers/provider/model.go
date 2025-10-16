package provider

import (
	"context"
	"fmt"
	"strconv"

	"github.com/MeowSalty/pinai/database/types"

	"github.com/gofiber/fiber/v2"
)

// AddModelToPlatform godoc
// @Summary      为指定平台添加新模型
// @Description  为指定平台添加新模型
// @Tags         models
// @Accept       json
// @Produce      json
// @Param        providerId  path      int                             true  "平台 ID"
// @Param        request     body      types.Model                     true  "创建模型的请求体"
// @Success      201         {object}  types.Model                       "创建成功的模型信息"
// @Failure      400         {object}  map[string]interface{}            "请求参数错误"
// @Failure      404         {object}  map[string]interface{}            "平台未找到"
// @Failure      500         {object}  map[string]interface{}            "服务器内部错误"
// @Router       /api/providers/{providerId}/models [post]
func (h *Handler) AddModelToPlatform(c *fiber.Ctx) error {
	providerId, err := strconv.ParseUint(c.Params("providerId"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "无效的平台 ID",
		})
	}

	var model types.Model
	if err := c.BodyParser(&model); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("无法解析请求体: %v", err),
		})
	}

	ctx := context.Background()
	createdModel, err := h.service.AddModelToPlatform(ctx, uint(providerId), model)
	if err != nil {
		// 检查错误类型
		if err.Error() == fmt.Sprintf("未找到 ID 为 %d 的平台", providerId) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "平台未找到",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("为平台添加模型失败: %v", err),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(createdModel)
}

// GetModelsByPlatform godoc
// @Summary      获取指定平台的所有模型列表
// @Description  获取指定平台的所有模型列表
// @Tags         models
// @Produce      json
// @Param        providerId  path      int  true  "平台 ID"
// @Success      200         {array}   types.Model                       "模型列表"
// @Failure      400         {object}  map[string]interface{}            "请求参数错误"
// @Failure      404         {object}  map[string]interface{}            "平台未找到"
// @Failure      500         {object}  map[string]interface{}            "服务器内部错误"
// @Router       /api/providers/{providerId}/models [get]
func (h *Handler) GetModelsByPlatform(c *fiber.Ctx) error {
	providerId, err := strconv.ParseUint(c.Params("providerId"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "无效的平台 ID",
		})
	}

	ctx := context.Background()
	models, err := h.service.GetModelsByPlatform(ctx, uint(providerId))
	if err != nil {
		// 检查错误类型
		if err.Error() == fmt.Sprintf("未找到 ID 为 %d 的平台", providerId) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "平台未找到",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("获取平台模型列表失败: %v", err),
		})
	}

	return c.JSON(models)
}

// UpdateModel godoc
// @Summary      更新指定模型信息
// @Description  更新指定模型信息
// @Tags         models
// @Accept       json
// @Produce      json
// @Param        modelId     path      int                             true  "模型 ID"
// @Param        request     body      types.Model                     true  "更新模型的请求体"
// @Success      200         {object}  types.Model                       "更新后的模型信息"
// @Failure      400         {object}  map[string]interface{}            "请求参数错误"
// @Failure      404         {object}  map[string]interface{}            "平台或模型未找到"
// @Failure      500         {object}  map[string]interface{}            "服务器内部错误"
// @Router       /api/providers/{providerId}/models/{modelId} [put]
func (h *Handler) UpdateModel(c *fiber.Ctx) error {
	modelId, err := strconv.ParseUint(c.Params("modelId"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "无效的模型 ID",
		})
	}

	var model types.Model
	if err := c.BodyParser(&model); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("无法解析请求体: %v", err),
		})
	}

	ctx := context.Background()
	updatedModel, err := h.service.UpdateModel(ctx, uint(modelId), model)
	if err != nil {
		// 检查错误类型
		if err.Error() == fmt.Sprintf("未找到 ID 为 %d 的模型", modelId) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "模型未找到",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("更新模型失败: %v", err),
		})
	}

	return c.JSON(updatedModel)
}

// DeleteModel godoc
// @Summary      删除指定模型
// @Description  删除指定模型
// @Tags         models
// @Produce      json
// @Param        modelId     path      int  true  "模型 ID"
// @Success      200         {object}  map[string]interface{}            "删除成功消息"
// @Failure      400         {object}  map[string]interface{}            "请求参数错误"
// @Failure      404         {object}  map[string]interface{}            "平台或模型未找到"
// @Failure      500         {object}  map[string]interface{}            "服务器内部错误"
// @Router       /api/providers/{providerId}/models/{modelId} [delete]
func (h *Handler) DeleteModel(c *fiber.Ctx) error {
	modelId, err := strconv.ParseUint(c.Params("modelId"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "无效的模型 ID",
		})
	}

	ctx := context.Background()
	err = h.service.DeleteModel(ctx, uint(modelId))
	if err != nil {
		// 检查错误类型
		if err.Error() == fmt.Sprintf("未找到 ID 为 %d 的模型", modelId) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "模型未找到",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("删除模型失败: %v", err),
		})
	}

	return c.JSON(fiber.Map{
		"message": "模型已成功删除",
	})
}
