package provider

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/MeowSalty/pinai/database/types"
	"github.com/MeowSalty/pinai/handlers/response"

	"github.com/gin-gonic/gin"
)

// ResourceHealthCount 资源健康状态计数
type ResourceHealthCount struct {
	Available   int64 `json:"available"`
	Warning     int64 `json:"warning"`
	Unavailable int64 `json:"unavailable"`
	Unknown     int64 `json:"unknown"`
}

// PlatformWithHealth 带健康状态的平台响应
type PlatformWithHealth struct {
	*types.Platform
	HealthStatus     *types.HealthStatus  `json:"health_status,omitempty"`
	KeyCount         int64                `json:"key_count"`
	ModelCount       int64                `json:"model_count"`
	KeyHealthCount   *ResourceHealthCount `json:"key_health_count"`
	ModelHealthCount *ResourceHealthCount `json:"model_health_count"`
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
func (h *Handler) CreatePlatform(c *gin.Context) {
	var platform types.Platform
	if err := c.ShouldBindJSON(&platform); err != nil {
		response.BadRequest(c, fmt.Sprintf("无法解析请求体: %v", err))
		return
	}

	ctx := c.Request.Context()
	createdPlatform, err := h.service.CreatePlatform(ctx, platform)
	if err != nil {
		respondProviderServiceError(c, err, "平台未找到", "创建平台失败")
		return
	}

	c.JSON(http.StatusCreated, createdPlatform)
}

// GetPlatforms godoc
// @Summary      获取所有平台列表
// @Description  获取所有平台列表，响应默认包含各平台的健康状态
// @Tags         platforms
// @Produce      json
// @Success      200  {array}   PlatformWithHealth                "平台列表"
// @Failure      500  {object}  map[string]interface{}            "服务器内部错误"
// @Router       /api/platforms [get]
func (h *Handler) GetPlatforms(c *gin.Context) {
	ctx := c.Request.Context()
	platforms, err := h.service.GetPlatforms(ctx)
	if err != nil {
		respondProviderServiceError(c, err, "平台未找到", "获取平台列表失败")
		return
	}

	keyCounts, modelCounts, err := h.service.GetPlatformResourceCounts(ctx)
	if err != nil {
		respondProviderServiceError(c, err, "平台未找到", "获取平台资源统计失败")
		return
	}

	// 获取资源到平台的映射，用于按平台统计健康状态
	keyMap, modelMap, err := h.service.GetResourcePlatformMaps(ctx)
	if err != nil {
		respondProviderServiceError(c, err, "平台未找到", "获取资源平台映射失败")
		return
	}

	storage := h.healthStorage

	// 按平台统计密钥和模型的健康分布
	keyHealthCounts := storage.CountByPlatform(types.ResourceTypeAPIKey, keyMap)
	modelHealthCounts := storage.CountByPlatform(types.ResourceTypeModel, modelMap)

	result := make([]PlatformWithHealth, len(platforms))
	for i, p := range platforms {
		result[i].Platform = p
		result[i].KeyCount = keyCounts[p.ID]
		result[i].ModelCount = modelCounts[p.ID]
		if health, _ := storage.Get(types.ResourceTypePlatform, p.ID); health != nil {
			result[i].HealthStatus = &health.Status
		} else {
			unknownStatus := types.HealthStatusUnknown
			result[i].HealthStatus = &unknownStatus
		}

		// 组装密钥健康计数
		kc := keyHealthCounts[p.ID]
		keyUnknown := keyCounts[p.ID] - kc.Available - kc.Warning - kc.Unavailable
		result[i].KeyHealthCount = &ResourceHealthCount{
			Available:   kc.Available,
			Warning:     kc.Warning,
			Unavailable: kc.Unavailable,
			Unknown:     keyUnknown,
		}

		// 组装模型健康计数
		mc := modelHealthCounts[p.ID]
		modelUnknown := modelCounts[p.ID] - mc.Available - mc.Warning - mc.Unavailable
		result[i].ModelHealthCount = &ResourceHealthCount{
			Available:   mc.Available,
			Warning:     mc.Warning,
			Unavailable: mc.Unavailable,
			Unknown:     modelUnknown,
		}
	}
	c.JSON(http.StatusOK, result)
}

// GetPlatform godoc
// @Summary      获取指定平台详情
// @Description  获取指定平台详情
// @Tags         platforms
// @Produce      json
// @Param        platformId   path      int  true  "平台 ID"
// @Success      200  {object}  types.Platform                    "平台详情"
// @Failure      400  {object}  map[string]interface{}            "请求参数错误"
// @Failure      404  {object}  map[string]interface{}            "平台未找到"
// @Failure      500  {object}  map[string]interface{}            "服务器内部错误"
// @Router       /api/platforms/{platformId} [get]
func (h *Handler) GetPlatform(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("platformId"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的平台 ID")
		return
	}

	ctx := c.Request.Context()
	platform, err := h.service.GetPlatform(ctx, uint(id))
	if err != nil {
		respondProviderServiceError(c, err, "平台未找到", "获取平台详情失败")
		return
	}

	c.JSON(http.StatusOK, platform)
}

// UpdatePlatform godoc
// @Summary      更新指定平台信息
// @Description  更新指定平台信息
// @Tags         platforms
// @Accept       json
// @Produce      json
// @Param        platformId  path      int                             true  "平台 ID"
// @Param        request  body      types.Platform                  true  "更新平台的请求体"
// @Success      200      {object}  types.Platform                    "更新后的平台信息"
// @Failure      400      {object}  map[string]interface{}            "请求参数错误"
// @Failure      404      {object}  map[string]interface{}            "平台未找到"
// @Failure      500      {object}  map[string]interface{}            "服务器内部错误"
// @Router       /api/platforms/{platformId} [put]
func (h *Handler) UpdatePlatform(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("platformId"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的平台 ID")
		return
	}

	var platform types.Platform
	if err := c.ShouldBindJSON(&platform); err != nil {
		response.BadRequest(c, fmt.Sprintf("无法解析请求体: %v", err))
		return
	}

	ctx := c.Request.Context()
	updatedPlatform, err := h.service.UpdatePlatform(ctx, uint(id), platform)
	if err != nil {
		respondProviderServiceError(c, err, "平台未找到", "更新平台失败")
		return
	}

	c.JSON(http.StatusOK, updatedPlatform)
}

// DeletePlatform godoc
// @Summary      删除指定平台
// @Description  删除指定平台及其所有关联的模型、密钥和关联关系
// @Tags         platforms
// @Produce      json
// @Param        platformId   path      int  true  "平台 ID"
// @Success      204  "删除成功"
// @Failure      400  {object}  map[string]interface{}            "请求参数错误"
// @Failure      404  {object}  map[string]interface{}            "平台未找到"
// @Failure      500  {object}  map[string]interface{}            "服务器内部错误"
// @Router       /api/platforms/{platformId} [delete]
func (h *Handler) DeletePlatform(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("platformId"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的平台 ID")
		return
	}

	ctx := c.Request.Context()
	err = h.service.DeletePlatform(ctx, uint(id))
	if err != nil {
		respondProviderServiceError(c, err, "平台未找到", "删除平台失败")
		return
	}

	c.Status(http.StatusNoContent)
}

// UpdatePlatformHealth godoc
// @Summary      更新平台健康状态
// @Description  通过 enabled 字段启用或禁用平台健康状态
// @Tags         platforms
// @Accept       json
// @Produce      json
// @Param        platformId  path      int                  true  "平台 ID"
// @Param        request     body      HealthUpdateRequest  true  "健康状态更新请求"
// @Success      200  {object}  map[string]interface{}  "操作成功"
// @Failure      400  {object}  map[string]interface{}  "请求参数错误"
// @Failure      404  {object}  map[string]interface{}  "平台未找到"
// @Failure      500  {object}  map[string]interface{}  "服务器内部错误"
// @Router       /api/platforms/{platformId}/health [patch]
func (h *Handler) UpdatePlatformHealth(c *gin.Context) {
	platformId, err := strconv.ParseUint(c.Param("platformId"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的平台 ID")
		return
	}

	var req HealthUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, fmt.Sprintf("无法解析请求体: %v", err))
		return
	}

	// 验证平台是否存在
	ctx := c.Request.Context()
	_, err = h.service.GetPlatform(ctx, uint(platformId))
	if err != nil {
		respondProviderServiceError(c, err, "平台未找到", "获取平台失败")
		return
	}

	if req.Enabled == nil {
		response.BadRequest(c, "必须提供 enabled 字段")
		return
	}

	if *req.Enabled {
		if err := h.healthService.EnableHealth(types.ResourceTypePlatform, uint(platformId)); err != nil {
			response.InternalError(c, "启用平台健康状态失败")
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"message":     "平台已启用",
			"platform_id": platformId,
			"status":      "unknown",
		})
		return
	}

	if err := h.healthService.DisableHealth(types.ResourceTypePlatform, uint(platformId)); err != nil {
		response.InternalError(c, "禁用平台健康状态失败")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":     "平台已禁用",
		"platform_id": platformId,
		"status":      "unavailable",
	})
}
