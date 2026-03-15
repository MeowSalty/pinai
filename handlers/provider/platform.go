package provider

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/MeowSalty/pinai/database/types"

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
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("无法解析请求体: %v", err),
		})
		return
	}

	ctx := context.Background()
	createdPlatform, err := h.service.CreatePlatform(ctx, platform)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("创建平台失败: %v", err),
		})
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
	ctx := context.Background()
	platforms, err := h.service.GetPlatforms(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("获取平台列表失败: %v", err),
		})
		return
	}

	keyCounts, modelCounts, err := h.service.GetPlatformResourceCounts(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("获取平台资源统计失败: %v", err),
		})
		return
	}

	// 获取资源到平台的映射，用于按平台统计健康状态
	keyMap, modelMap, err := h.service.GetResourcePlatformMaps(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("获取资源平台映射失败: %v", err),
		})
		return
	}

	storage := h.healthService.GetStorage()

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
// @Param        id   path      int  true  "平台 ID"
// @Success      200  {object}  types.Platform                    "平台详情"
// @Failure      400  {object}  map[string]interface{}            "请求参数错误"
// @Failure      404  {object}  map[string]interface{}            "平台未找到"
// @Failure      500  {object}  map[string]interface{}            "服务器内部错误"
// @Router       /api/platforms/{id} [get]
func (h *Handler) GetPlatform(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "无效的平台 ID",
		})
		return
	}

	ctx := context.Background()
	platform, err := h.service.GetPlatform(ctx, uint(id))
	if err != nil {
		// 检查错误类型，如果未找到则返回 404
		if err.Error() == fmt.Sprintf("未找到 ID 为 %d 的平台", id) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "平台未找到",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("获取平台详情失败: %v", err),
		})
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
// @Param        id       path      int                             true  "平台 ID"
// @Param        request  body      types.Platform                  true  "更新平台的请求体"
// @Success      200      {object}  types.Platform                    "更新后的平台信息"
// @Failure      400      {object}  map[string]interface{}            "请求参数错误"
// @Failure      404      {object}  map[string]interface{}            "平台未找到"
// @Failure      500      {object}  map[string]interface{}            "服务器内部错误"
// @Router       /api/platforms/{id} [put]
func (h *Handler) UpdatePlatform(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "无效的平台 ID",
		})
		return
	}

	var platform types.Platform
	if err := c.ShouldBindJSON(&platform); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("无法解析请求体: %v", err),
		})
		return
	}

	ctx := context.Background()
	updatedPlatform, err := h.service.UpdatePlatform(ctx, uint(id), platform)
	if err != nil {
		// 检查错误类型
		if err.Error() == fmt.Sprintf("未找到 ID 为 %d 的平台", id) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "平台未找到",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("更新平台失败: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, updatedPlatform)
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
func (h *Handler) DeletePlatform(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "无效的平台 ID",
		})
		return
	}

	ctx := context.Background()
	err = h.service.DeletePlatform(ctx, uint(id))
	if err != nil {
		// 检查错误类型
		if err.Error() == fmt.Sprintf("未找到 ID 为 %d 的平台", id) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "平台未找到",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("删除平台失败: %v", err),
		})
		return
	}

	c.Status(http.StatusNoContent)
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
func (h *Handler) EnablePlatformHealth(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "无效的平台 ID",
		})
		return
	}

	// 验证平台是否存在
	ctx := context.Background()
	_, err = h.service.GetPlatform(ctx, uint(id))
	if err != nil {
		if err.Error() == fmt.Sprintf("未找到 ID 为 %d 的平台", id) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "平台未找到",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("获取平台失败: %v", err),
		})
		return
	}

	// 启用健康状态
	if err := h.healthService.EnableHealth(types.ResourceTypePlatform, uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("启用平台健康状态失败: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
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
func (h *Handler) DisablePlatformHealth(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "无效的平台 ID",
		})
		return
	}

	// 验证平台是否存在
	ctx := context.Background()
	_, err = h.service.GetPlatform(ctx, uint(id))
	if err != nil {
		if err.Error() == fmt.Sprintf("未找到 ID 为 %d 的平台", id) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "平台未找到",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("获取平台失败: %v", err),
		})
		return
	}

	// 禁用健康状态
	if err := h.healthService.DisableHealth(types.ResourceTypePlatform, uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("禁用平台健康状态失败: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":     "平台已禁用",
		"platform_id": id,
		"status":      "unavailable",
	})
}
