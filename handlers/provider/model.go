package provider

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/MeowSalty/pinai/database/types"
	"github.com/MeowSalty/pinai/services/provider"

	"github.com/gin-gonic/gin"
)

// ModelWithHealth 带健康状态的模型响应
type ModelWithHealth struct {
	*types.Model
	HealthStatus *types.HealthStatus `json:"health_status,omitempty"`
}

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
func (h *Handler) AddModelToPlatform(c *gin.Context) {
	platformId, err := strconv.ParseUint(c.Param("platformId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "无效的平台 ID",
		})
		return
	}

	var model types.Model
	if err := c.ShouldBindJSON(&model); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("无法解析请求体: %v", err),
		})
		return
	}

	ctx := context.Background()
	createdModel, err := h.service.AddModelToPlatform(ctx, uint(platformId), model)
	if err != nil {
		// 检查错误类型
		if err.Error() == fmt.Sprintf("未找到 ID 为 %d 的平台", platformId) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "平台未找到",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("为平台添加模型失败: %v", err),
		})
		return
	}

	c.JSON(http.StatusCreated, createdModel)
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
func (h *Handler) BatchAddModelsToPlatform(c *gin.Context) {
	platformId, err := strconv.ParseUint(c.Param("platformId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "无效的平台 ID",
		})
		return
	}

	var req provider.BatchCreateModelsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("无法解析请求体: %v", err),
		})
		return
	}

	// 验证至少有一个模型
	if len(req.Models) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "必须至少提供一个模型",
		})
		return
	}

	ctx := context.Background()
	createdModels, err := h.service.BatchAddModelsToPlatform(ctx, uint(platformId), req.Models)
	if err != nil {
		// 检查错误类型
		if err.Error() == fmt.Sprintf("未找到 ID 为 %d 的平台", platformId) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "平台未找到",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("批量创建模型失败: %v", err),
		})
		return
	}

	response := provider.BatchCreateModelsResponse{
		Models:       createdModels,
		TotalCount:   len(req.Models),
		CreatedCount: len(createdModels),
	}

	c.JSON(http.StatusCreated, response)
}

// GetModelsByPlatform godoc
// @Summary      获取指定平台的所有模型列表
// @Description  获取指定平台的所有模型列表，可通过 include=health 参数包含健康状态
// @Tags         models
// @Produce      json
// @Param        platformId  path      int     true   "平台 ID"
// @Param        include     query     string  false  "包含额外信息，支持 health"
// @Success      200         {array}   ModelWithHealth                   "模型列表"
// @Failure      400         {object}  map[string]interface{}            "请求参数错误"
// @Failure      404         {object}  map[string]interface{}            "平台未找到"
// @Failure      500         {object}  map[string]interface{}            "服务器内部错误"
// @Router       /api/platforms/{platformId}/models [get]
func (h *Handler) GetModelsByPlatform(c *gin.Context) {
	platformId, err := strconv.ParseUint(c.Param("platformId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "无效的平台 ID",
		})
		return
	}

	ctx := context.Background()
	models, err := h.service.GetModelsByPlatform(ctx, uint(platformId))
	if err != nil {
		// 检查错误类型
		if err.Error() == fmt.Sprintf("未找到 ID 为 %d 的平台", platformId) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "平台未找到",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("获取平台模型列表失败: %v", err),
		})
		return
	}

	// 检查是否需要包含健康状态
	if c.Query("include") == "health" {
		storage := h.healthService.GetStorage()
		result := make([]ModelWithHealth, len(models))
		for i, m := range models {
			result[i].Model = m
			if health, _ := storage.Get(types.ResourceTypeModel, m.ID); health != nil {
				result[i].HealthStatus = &health.Status
			} else {
				// 没有健康数据时使用未知状态
				unknownStatus := types.HealthStatusUnknown
				result[i].HealthStatus = &unknownStatus
			}
		}
		c.JSON(http.StatusOK, result)
		return
	}

	c.JSON(http.StatusOK, models)
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
func (h *Handler) UpdateModel(c *gin.Context) {
	modelId, err := strconv.ParseUint(c.Param("modelId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "无效的模型 ID",
		})
		return
	}

	var model types.Model
	if err := c.ShouldBindJSON(&model); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("无法解析请求体: %v", err),
		})
		return
	}

	ctx := context.Background()
	updatedModel, err := h.service.UpdateModel(ctx, uint(modelId), model)
	if err != nil {
		// 检查错误类型
		if err.Error() == fmt.Sprintf("未找到 ID 为 %d 的模型", modelId) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "模型未找到",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("更新模型失败: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, updatedModel)
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
func (h *Handler) DeleteModel(c *gin.Context) {
	modelId, err := strconv.ParseUint(c.Param("modelId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "无效的模型 ID",
		})
		return
	}

	ctx := context.Background()
	err = h.service.DeleteModel(ctx, uint(modelId))
	if err != nil {
		// 检查错误类型
		if err.Error() == fmt.Sprintf("未找到 ID 为 %d 的模型", modelId) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "模型未找到",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("删除模型失败: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
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
func (h *Handler) BatchUpdateModels(c *gin.Context) {
	platformId, err := strconv.ParseUint(c.Param("platformId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "无效的平台 ID",
		})
		return
	}

	var req provider.BatchUpdateModelsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("无法解析请求体: %v", err),
		})
		return
	}

	// 验证至少有一个模型更新项
	if len(req.Models) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "必须至少提供一个模型更新项",
		})
		return
	}

	// 验证每个模型更新项必须包含 ID
	for i, item := range req.Models {
		if item.ID == 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": fmt.Sprintf("模型更新项 %d 缺少必需的 ID 字段", i),
			})
			return
		}
	}

	ctx := context.Background()
	updatedModels, err := h.service.BatchUpdateModels(ctx, uint(platformId), req.Models)
	if err != nil {
		// 检查错误类型
		if err.Error() == fmt.Sprintf("未找到 ID 为 %d 的平台", platformId) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "平台未找到",
			})
			return
		}
		// 检查是否是模型不存在的错误
		if len(err.Error()) > 0 && (err.Error()[:6] == "未找到" || err.Error()[:6] == "模型 ID") {
			c.JSON(http.StatusNotFound, gin.H{
				"error": err.Error(),
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("批量更新模型失败: %v", err),
		})
		return
	}

	response := provider.BatchUpdateModelsResponse{
		Models:       updatedModels,
		TotalCount:   len(req.Models),
		UpdatedCount: len(updatedModels),
	}

	c.JSON(http.StatusOK, response)
}

// EnableModelHealth godoc
// @Summary      启用/恢复模型健康状态
// @Description  删除模型的健康记录，让系统重新评估健康状态
// @Tags         models
// @Produce      json
// @Param        platformId  path      int  true  "平台 ID"
// @Param        modelId     path      int  true  "模型 ID"
// @Success      200  {object}  map[string]interface{}  "操作成功"
// @Failure      400  {object}  map[string]interface{}  "请求参数错误"
// @Failure      404  {object}  map[string]interface{}  "模型未找到"
// @Failure      500  {object}  map[string]interface{}  "服务器内部错误"
// @Router       /api/platforms/{platformId}/models/{modelId}/health/enable [post]
func (h *Handler) EnableModelHealth(c *gin.Context) {
	modelId, err := strconv.ParseUint(c.Param("modelId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "无效的模型 ID",
		})
		return
	}

	// 验证模型是否存在
	ctx := context.Background()
	_, err = h.service.GetModel(ctx, uint(modelId))
	if err != nil {
		if err.Error() == fmt.Sprintf("未找到 ID 为 %d 的模型", modelId) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "模型未找到",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("获取模型失败: %v", err),
		})
		return
	}

	// 启用健康状态
	if err := h.healthService.EnableHealth(types.ResourceTypeModel, uint(modelId)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("启用模型健康状态失败: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "模型已启用",
		"model_id": modelId,
		"status":   "unknown",
	})
}

// DisableModelHealth godoc
// @Summary      禁用模型健康状态
// @Description  将模型健康状态设置为不可用
// @Tags         models
// @Produce      json
// @Param        platformId  path      int  true  "平台 ID"
// @Param        modelId     path      int  true  "模型 ID"
// @Success      200  {object}  map[string]interface{}  "操作成功"
// @Failure      400  {object}  map[string]interface{}  "请求参数错误"
// @Failure      404  {object}  map[string]interface{}  "模型未找到"
// @Failure      500  {object}  map[string]interface{}  "服务器内部错误"
// @Router       /api/platforms/{platformId}/models/{modelId}/health/disable [post]
func (h *Handler) DisableModelHealth(c *gin.Context) {
	modelId, err := strconv.ParseUint(c.Param("modelId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "无效的模型 ID",
		})
		return
	}

	// 验证模型是否存在
	ctx := context.Background()
	_, err = h.service.GetModel(ctx, uint(modelId))
	if err != nil {
		if err.Error() == fmt.Sprintf("未找到 ID 为 %d 的模型", modelId) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "模型未找到",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("获取模型失败: %v", err),
		})
		return
	}

	// 禁用健康状态
	if err := h.healthService.DisableHealth(types.ResourceTypeModel, uint(modelId)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("禁用模型健康状态失败: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "模型已禁用",
		"model_id": modelId,
		"status":   "unavailable",
	})
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
func (h *Handler) BatchDeleteModels(c *gin.Context) {
	platformId, err := strconv.ParseUint(c.Param("platformId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "无效的平台 ID",
		})
		return
	}

	var req provider.BatchDeleteModelsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("无法解析请求体: %v", err),
		})
		return
	}

	// 验证至少有一个模型 ID
	if len(req.ModelIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "必须至少提供一个模型 ID",
		})
		return
	}

	ctx := context.Background()
	deletedCount, err := h.service.BatchDeleteModels(ctx, uint(platformId), req.ModelIDs)
	if err != nil {
		// 检查错误类型
		if err.Error() == fmt.Sprintf("未找到 ID 为 %d 的平台", platformId) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "平台未找到",
			})
			return
		}
		// 检查是否是模型不存在的错误
		if len(err.Error()) > 0 && err.Error()[:2] == "以下" {
			c.JSON(http.StatusNotFound, gin.H{
				"error": err.Error(),
			})
			return
		}
		// 检查是否是模型不属于平台的错误
		if len(err.Error()) > 6 && err.Error()[:6] == "模型 ID" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("批量删除模型失败: %v", err),
		})
		return
	}

	response := provider.BatchDeleteModelsResponse{
		TotalCount:   len(req.ModelIDs),
		DeletedCount: deletedCount,
	}

	c.JSON(http.StatusOK, response)
}
