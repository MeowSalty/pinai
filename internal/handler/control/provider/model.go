package provider

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/MeowSalty/pinai/database/types"
	"github.com/MeowSalty/pinai/internal/app/provider"
	"github.com/MeowSalty/pinai/internal/handler/response"

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
// @Failure      400         {object}  response.ErrorResponse            "请求参数错误"
// @Failure      404         {object}  response.ErrorResponse            "平台未找到"
// @Failure      500         {object}  response.ErrorResponse            "服务器内部错误"
// @Router       /api/platforms/{platformId}/models [post]
func (h *Handler) AddModelToPlatform(c *gin.Context) {
	platformId, err := strconv.ParseUint(c.Param("platformId"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的平台 ID")
		return
	}

	var model types.Model
	if err := c.ShouldBindJSON(&model); err != nil {
		response.BadRequest(c, fmt.Sprintf("无法解析请求体: %v", err))
		return
	}

	ctx := c.Request.Context()
	createdModel, err := h.service.AddModelToPlatform(ctx, uint(platformId), model)
	if err != nil {
		respondProviderServiceError(c, err, "平台未找到", "为平台添加模型失败")
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
// @Failure      400         {object}  response.ErrorResponse                    "请求参数错误"
// @Failure      404         {object}  response.ErrorResponse                    "平台未找到"
// @Failure      500         {object}  response.ErrorResponse                    "服务器内部错误"
// @Router       /api/platforms/{platformId}/models/batch [post]
func (h *Handler) BatchAddModelsToPlatform(c *gin.Context) {
	platformId, err := strconv.ParseUint(c.Param("platformId"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的平台 ID")
		return
	}

	var req provider.BatchCreateModelsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, fmt.Sprintf("无法解析请求体: %v", err))
		return
	}

	// 验证至少有一个模型
	if len(req.Models) == 0 {
		response.BadRequest(c, "必须至少提供一个模型")
		return
	}

	ctx := c.Request.Context()
	createdModels, err := h.service.BatchAddModelsToPlatform(ctx, uint(platformId), req.Models)
	if err != nil {
		respondProviderServiceError(c, err, "平台未找到", "批量创建模型失败")
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
// @Failure      400         {object}  response.ErrorResponse            "请求参数错误"
// @Failure      404         {object}  response.ErrorResponse            "平台未找到"
// @Failure      500         {object}  response.ErrorResponse            "服务器内部错误"
// @Router       /api/platforms/{platformId}/models [get]
func (h *Handler) GetModelsByPlatform(c *gin.Context) {
	platformId, err := strconv.ParseUint(c.Param("platformId"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的平台 ID")
		return
	}

	ctx := c.Request.Context()
	models, err := h.service.GetModelsByPlatform(ctx, uint(platformId))
	if err != nil {
		respondProviderServiceError(c, err, "平台未找到", "获取平台模型列表失败")
		return
	}

	// 检查是否需要包含健康状态
	if c.Query("include") == "health" {
		result := make([]ModelWithHealth, len(models))
		for i, m := range models {
			result[i].Model = m
			status, statusErr := h.service.GetResourceHealthStatus(types.ResourceTypeModel, m.ID)
			if statusErr != nil {
				respondProviderServiceError(c, statusErr, "模型未找到", "获取模型健康状态失败")
				return
			}
			result[i].HealthStatus = &status
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
// @Param        modelId     path      int                             true  "模型 ID"
// @Param        request     body      types.Model                     true  "更新模型的请求体"
// @Success      200         {object}  types.Model                       "更新后的模型信息"
// @Failure      400         {object}  response.ErrorResponse            "请求参数错误"
// @Failure      404         {object}  response.ErrorResponse            "模型未找到"
// @Failure      500         {object}  response.ErrorResponse            "服务器内部错误"
// @Router       /api/models/{modelId} [put]
func (h *Handler) UpdateModel(c *gin.Context) {
	modelId, err := strconv.ParseUint(c.Param("modelId"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的模型 ID")
		return
	}

	var model types.Model
	if err := c.ShouldBindJSON(&model); err != nil {
		response.BadRequest(c, fmt.Sprintf("无法解析请求体: %v", err))
		return
	}

	ctx := c.Request.Context()
	updatedModel, err := h.service.UpdateModel(ctx, uint(modelId), model)
	if err != nil {
		respondProviderServiceError(c, err, "模型未找到", "更新模型失败")
		return
	}

	c.JSON(http.StatusOK, updatedModel)
}

// DeleteModel godoc
// @Summary      删除指定模型
// @Description  删除指定模型
// @Tags         models
// @Produce      json
// @Param        modelId     path      int  true  "模型 ID"
// @Success      204         "删除成功"
// @Failure      400         {object}  response.ErrorResponse            "请求参数错误"
// @Failure      404         {object}  response.ErrorResponse            "模型未找到"
// @Failure      500         {object}  response.ErrorResponse            "服务器内部错误"
// @Router       /api/models/{modelId} [delete]
func (h *Handler) DeleteModel(c *gin.Context) {
	modelId, err := strconv.ParseUint(c.Param("modelId"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的模型 ID")
		return
	}

	ctx := c.Request.Context()
	err = h.service.DeleteModel(ctx, uint(modelId))
	if err != nil {
		respondProviderServiceError(c, err, "模型未找到", "删除模型失败")
		return
	}

	c.Status(http.StatusNoContent)
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
// @Failure      400         {object}  response.ErrorResponse                    "请求参数错误"
// @Failure      404         {object}  response.ErrorResponse                    "平台或模型未找到"
// @Failure      500         {object}  response.ErrorResponse                    "服务器内部错误"
// @Router       /api/platforms/{platformId}/models/batch [put]
func (h *Handler) BatchUpdateModels(c *gin.Context) {
	platformId, err := strconv.ParseUint(c.Param("platformId"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的平台 ID")
		return
	}

	var req provider.BatchUpdateModelsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, fmt.Sprintf("无法解析请求体: %v", err))
		return
	}

	// 验证至少有一个模型更新项
	if len(req.Models) == 0 {
		response.BadRequest(c, "必须至少提供一个模型更新项")
		return
	}

	// 验证每个模型更新项必须包含 ID
	for i, item := range req.Models {
		if item.ID == 0 {
			response.BadRequest(c, fmt.Sprintf("模型更新项 %d 缺少必需的 ID 字段", i))
			return
		}
	}

	ctx := c.Request.Context()
	updatedModels, err := h.service.BatchUpdateModels(ctx, uint(platformId), req.Models)
	if err != nil {
		respondProviderServiceError(c, err, "模型未找到", "批量更新模型失败")
		return
	}

	response := provider.BatchUpdateModelsResponse{
		Models:       updatedModels,
		TotalCount:   len(req.Models),
		UpdatedCount: len(updatedModels),
	}

	c.JSON(http.StatusOK, response)
}

// UpdateModelHealth godoc
// @Summary      更新模型健康状态
// @Description  通过 enabled 字段启用或禁用模型健康状态
// @Tags         models
// @Accept       json
// @Produce      json
// @Param        modelId     path      int                  true  "模型 ID"
// @Param        request     body      HealthUpdateRequest  true  "健康状态更新请求"
// @Success      200  {object}  map[string]interface{}  "操作成功"
// @Failure      400  {object}  response.ErrorResponse  "请求参数错误"
// @Failure      404  {object}  response.ErrorResponse  "模型未找到"
// @Failure      500  {object}  response.ErrorResponse  "服务器内部错误"
// @Router       /api/models/{modelId}/health [patch]
func (h *Handler) UpdateModelHealth(c *gin.Context) {
	var req HealthUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, fmt.Sprintf("无法解析请求体: %v", err))
		return
	}
	h.updateModelHealthWithEnabled(c, req.Enabled)
}

func (h *Handler) updateModelHealthWithEnabled(c *gin.Context, enabled *bool) {
	modelId, err := strconv.ParseUint(c.Param("modelId"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的模型 ID")
		return
	}

	if enabled == nil {
		response.BadRequest(c, "必须提供 enabled 字段")
		return
	}

	ctx := c.Request.Context()
	status, err := h.service.UpdateModelHealthEnabled(ctx, uint(modelId), *enabled)
	if err != nil {
		respondProviderServiceError(c, err, "模型未找到", "更新模型健康状态失败")
		return
	}

	message := "模型已禁用"
	statusText := "unavailable"
	if status == types.HealthStatusUnknown {
		message = "模型已启用"
		statusText = "unknown"
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  message,
		"model_id": modelId,
		"status":   statusText,
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
// @Failure      400         {object}  response.ErrorResponse                    "请求参数错误"
// @Failure      404         {object}  response.ErrorResponse                    "平台或模型未找到"
// @Failure      500         {object}  response.ErrorResponse                    "服务器内部错误"
// @Router       /api/platforms/{platformId}/models/batch [delete]
func (h *Handler) BatchDeleteModels(c *gin.Context) {
	platformId, err := strconv.ParseUint(c.Param("platformId"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的平台 ID")
		return
	}

	var req provider.BatchDeleteModelsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, fmt.Sprintf("无法解析请求体: %v", err))
		return
	}

	// 验证至少有一个模型 ID
	if len(req.ModelIDs) == 0 {
		response.BadRequest(c, "必须至少提供一个模型 ID")
		return
	}

	ctx := c.Request.Context()
	deletedCount, err := h.service.BatchDeleteModels(ctx, uint(platformId), req.ModelIDs)
	if err != nil {
		respondProviderServiceError(c, err, "模型未找到", "批量删除模型失败")
		return
	}

	response := provider.BatchDeleteModelsResponse{
		TotalCount:   len(req.ModelIDs),
		DeletedCount: deletedCount,
	}

	c.JSON(http.StatusOK, response)
}
