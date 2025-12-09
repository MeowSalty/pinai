package provider

import (
	"context"
	"fmt"
	"strconv"

	"github.com/MeowSalty/pinai/database/types"
	"github.com/MeowSalty/pinai/services/provider"

	"github.com/gofiber/fiber/v2"
)

// AddModelToPlatform godoc
// @Summary      为指定平台添加新模型
// @Description  为指定平台添加新模型
// @Tags         models
// @Accept       json
// @Produce      json
// @Param        platformId  path      int                             true  "平台 ID"
// @Param        request     body      types.Model                     true  "创建模型的请求体"
// @Success      201         {object}  types.Model                       "创建成功的模型信息"
// @Failure      400         {object}  map[string]interface{}            "请求参数错误"
// @Failure      404         {object}  map[string]interface{}            "平台未找到"
// @Failure      500         {object}  map[string]interface{}            "服务器内部错误"
// @Router       /api/platforms/{platformId}/models [post]
func (h *Handler) AddModelToPlatform(c *fiber.Ctx) error {
	platformId, err := strconv.ParseUint(c.Params("platformId"), 10, 64)
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
	createdModel, err := h.service.AddModelToPlatform(ctx, uint(platformId), model)
	if err != nil {
		// 检查错误类型
		if err.Error() == fmt.Sprintf("未找到 ID 为 %d 的平台", platformId) {
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

// BatchAddModelsToPlatform godoc
// @Summary      批量为指定平台添加模型
// @Description  批量创建多个模型，采用原子性事务（全部成功或全部失败）
// @Tags         models
// @Accept       json
// @Produce      json
// @Param        platformId  path      int                                      true  "平台 ID"
// @Param        request     body      provider.BatchCreateModelsRequest        true  "批量创建模型的请求体"
// @Success      201         {object}  provider.BatchCreateModelsResponse       "全部创建成功"
// @Failure      400         {object}  map[string]interface{}                   "请求参数错误"
// @Failure      404         {object}  map[string]interface{}                   "平台未找到"
// @Failure      500         {object}  map[string]interface{}                   "服务器内部错误"
// @Router       /api/platforms/{platformId}/models/batch [post]
func (h *Handler) BatchAddModelsToPlatform(c *fiber.Ctx) error {
	platformId, err := strconv.ParseUint(c.Params("platformId"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "无效的平台 ID",
		})
	}

	var req provider.BatchCreateModelsRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("无法解析请求体: %v", err),
		})
	}

	// 验证至少有一个模型
	if len(req.Models) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "必须至少提供一个模型",
		})
	}

	ctx := context.Background()
	createdModels, err := h.service.BatchAddModelsToPlatform(ctx, uint(platformId), req.Models)
	if err != nil {
		// 检查错误类型
		if err.Error() == fmt.Sprintf("未找到 ID 为 %d 的平台", platformId) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "平台未找到",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("批量创建模型失败: %v", err),
		})
	}

	response := provider.BatchCreateModelsResponse{
		Models:       createdModels,
		TotalCount:   len(req.Models),
		CreatedCount: len(createdModels),
	}

	return c.Status(fiber.StatusCreated).JSON(response)
}

// GetModelsByPlatform godoc
// @Summary      获取指定平台的所有模型列表
// @Description  获取指定平台的所有模型列表
// @Tags         models
// @Produce      json
// @Param        platformId  path      int  true  "平台 ID"
// @Success      200         {array}   types.Model                       "模型列表"
// @Failure      400         {object}  map[string]interface{}            "请求参数错误"
// @Failure      404         {object}  map[string]interface{}            "平台未找到"
// @Failure      500         {object}  map[string]interface{}            "服务器内部错误"
// @Router       /api/platforms/{platformId}/models [get]
func (h *Handler) GetModelsByPlatform(c *fiber.Ctx) error {
	platformId, err := strconv.ParseUint(c.Params("platformId"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "无效的平台 ID",
		})
	}

	ctx := context.Background()
	models, err := h.service.GetModelsByPlatform(ctx, uint(platformId))
	if err != nil {
		// 检查错误类型
		if err.Error() == fmt.Sprintf("未找到 ID 为 %d 的平台", platformId) {
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
// @Param        platformId  path      int                             true  "平台 ID"
// @Param        modelId     path      int                             true  "模型 ID"
// @Param        request     body      types.Model                     true  "更新模型的请求体"
// @Success      200         {object}  types.Model                       "更新后的模型信息"
// @Failure      400         {object}  map[string]interface{}            "请求参数错误"
// @Failure      404         {object}  map[string]interface{}            "平台或模型未找到"
// @Failure      500         {object}  map[string]interface{}            "服务器内部错误"
// @Router       /api/platforms/{platformId}/models/{modelId} [put]
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
// @Param        platformId  path      int  true  "平台 ID"
// @Param        modelId     path      int  true  "模型 ID"
// @Success      200         {object}  map[string]interface{}            "删除成功消息"
// @Failure      400         {object}  map[string]interface{}            "请求参数错误"
// @Failure      404         {object}  map[string]interface{}            "平台或模型未找到"
// @Failure      500         {object}  map[string]interface{}            "服务器内部错误"
// @Router       /api/platforms/{platformId}/models/{modelId} [delete]
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

// BatchUpdateModels godoc
// @Summary      批量更新指定平台的模型
// @Description  批量更新多个模型的信息，采用原子性事务（全部成功或全部失败）
// @Tags         models
// @Accept       json
// @Produce      json
// @Param        platformId  path      int                                      true  "平台 ID"
// @Param        request     body      provider.BatchUpdateModelsRequest        true  "批量更新模型的请求体"
// @Success      200         {object}  provider.BatchUpdateModelsResponse       "全部更新成功"
// @Failure      400         {object}  map[string]interface{}                   "请求参数错误"
// @Failure      404         {object}  map[string]interface{}                   "平台或模型未找到"
// @Failure      500         {object}  map[string]interface{}                   "服务器内部错误"
// @Router       /api/platforms/{platformId}/models/batch [put]
func (h *Handler) BatchUpdateModels(c *fiber.Ctx) error {
	platformId, err := strconv.ParseUint(c.Params("platformId"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "无效的平台 ID",
		})
	}

	var req provider.BatchUpdateModelsRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("无法解析请求体: %v", err),
		})
	}

	// 验证至少有一个模型更新项
	if len(req.Models) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "必须至少提供一个模型更新项",
		})
	}

	// 验证每个模型更新项必须包含 ID
	for i, item := range req.Models {
		if item.ID == 0 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": fmt.Sprintf("模型更新项 %d 缺少必需的 ID 字段", i),
			})
		}
	}

	ctx := context.Background()
	updatedModels, err := h.service.BatchUpdateModels(ctx, uint(platformId), req.Models)
	if err != nil {
		// 检查错误类型
		if err.Error() == fmt.Sprintf("未找到 ID 为 %d 的平台", platformId) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "平台未找到",
			})
		}
		// 检查是否是模型不存在的错误
		if len(err.Error()) > 0 && (err.Error()[:6] == "未找到" || err.Error()[:6] == "模型 ID") {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("批量更新模型失败: %v", err),
		})
	}

	response := provider.BatchUpdateModelsResponse{
		Models:       updatedModels,
		TotalCount:   len(req.Models),
		UpdatedCount: len(updatedModels),
	}

	return c.JSON(response)
}

// BatchDeleteModels godoc
// @Summary      批量删除指定平台的模型
// @Description  批量删除多个模型，采用原子性事务（全部成功或全部失败）
// @Tags         models
// @Accept       json
// @Produce      json
// @Param        platformId  path      int                                      true  "平台 ID"
// @Param        request     body      provider.BatchDeleteModelsRequest        true  "批量删除模型的请求体"
// @Success      200         {object}  provider.BatchDeleteModelsResponse       "全部删除成功"
// @Failure      400         {object}  map[string]interface{}                   "请求参数错误"
// @Failure      404         {object}  map[string]interface{}                   "平台或模型未找到"
// @Failure      500         {object}  map[string]interface{}                   "服务器内部错误"
// @Router       /api/platforms/{platformId}/models/batch [delete]
func (h *Handler) BatchDeleteModels(c *fiber.Ctx) error {
	platformId, err := strconv.ParseUint(c.Params("platformId"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "无效的平台 ID",
		})
	}

	var req provider.BatchDeleteModelsRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("无法解析请求体: %v", err),
		})
	}

	// 验证至少有一个模型 ID
	if len(req.ModelIDs) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "必须至少提供一个模型 ID",
		})
	}

	ctx := context.Background()
	deletedCount, err := h.service.BatchDeleteModels(ctx, uint(platformId), req.ModelIDs)
	if err != nil {
		// 检查错误类型
		if err.Error() == fmt.Sprintf("未找到 ID 为 %d 的平台", platformId) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "平台未找到",
			})
		}
		// 检查是否是模型不存在的错误
		if len(err.Error()) > 0 && err.Error()[:2] == "以下" {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		// 检查是否是模型不属于平台的错误
		if len(err.Error()) > 6 && err.Error()[:6] == "模型 ID" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("批量删除模型失败: %v", err),
		})
	}

	response := provider.BatchDeleteModelsResponse{
		TotalCount:   len(req.ModelIDs),
		DeletedCount: deletedCount,
	}

	return c.JSON(response)
}
