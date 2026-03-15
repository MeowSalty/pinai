package provider

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/MeowSalty/pinai/database/types"

	"github.com/gin-gonic/gin"
)

// KeyWithHealth 带健康状态的密钥响应
type KeyWithHealth struct {
	*types.APIKey
	HealthStatus *types.HealthStatus `json:"health_status,omitempty"`
}

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
func (h *Handler) AddKeyToPlatform(c *gin.Context) {
	platformId, err := strconv.ParseUint(c.Param("platformId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "无效的平台 ID",
		})
		return
	}

	var key types.APIKey
	if err := c.ShouldBindJSON(&key); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("无法解析请求体: %v", err),
		})
		return
	}

	ctx := c.Request.Context()
	createdKey, err := h.service.AddKeyToPlatform(ctx, uint(platformId), key)
	if err != nil {
		// 检查错误类型
		if err.Error() == fmt.Sprintf("未找到 ID 为 %d 的平台", platformId) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "平台未找到",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("为平台添加密钥失败: %v", err),
		})
		return
	}

	// 出于安全考虑，不返回密钥值
	createdKey.Value = ""
	c.JSON(http.StatusCreated, createdKey)
}

// GetKeysByPlatform godoc
// @Summary      获取指定平台的所有密钥列表
// @Description  获取指定平台的所有密钥列表 (不包含密钥值)，可通过 include=health 参数包含健康状态
// @Tags         keys
// @Produce      json
// @Param        platformId  path      int     true   "平台 ID"
// @Param        include     query     string  false  "包含额外信息，支持 health"
// @Success      200         {array}   KeyWithHealth                     "密钥列表 (不包含 value)"
// @Failure      400         {object}  map[string]interface{}            "请求参数错误"
// @Failure      404         {object}  map[string]interface{}            "平台未找到"
// @Failure      500         {object}  map[string]interface{}            "服务器内部错误"
// @Router       /api/platforms/{platformId}/keys [get]
func (h *Handler) GetKeysByPlatform(c *gin.Context) {
	platformId, err := strconv.ParseUint(c.Param("platformId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "无效的平台 ID",
		})
		return
	}

	ctx := c.Request.Context()
	keys, err := h.service.GetKeysByPlatform(ctx, uint(platformId))
	if err != nil {
		// 检查错误类型
		if err.Error() == fmt.Sprintf("未找到 ID 为 %d 的平台", platformId) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "平台未找到",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("获取平台密钥列表失败: %v", err),
		})
		return
	}

	// 检查是否需要包含健康状态
	if c.Query("include") == "health" {
		storage := h.healthService.GetStorage()
		result := make([]KeyWithHealth, len(keys))
		for i, k := range keys {
			result[i].APIKey = k
			if health, _ := storage.Get(types.ResourceTypeAPIKey, k.ID); health != nil {
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

	c.JSON(http.StatusOK, keys)
}

// DeleteKey godoc
// @Summary      删除指定密钥
// @Description  删除指定密钥
// @Tags         keys
// @Produce      json
// @Param        keyId       path      int  true  "密钥 ID"
// @Success      200         {object}  map[string]interface{}            "删除成功消息"
// @Failure      400         {object}  map[string]interface{}            "请求参数错误"
// @Failure      404         {object}  map[string]interface{}            "密钥未找到"
// @Failure      500         {object}  map[string]interface{}            "服务器内部错误"
// @Router       /api/keys/{keyId} [delete]
func (h *Handler) DeleteKey(c *gin.Context) {
	keyId, err := strconv.ParseUint(c.Param("keyId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "无效的密钥 ID",
		})
		return
	}

	ctx := c.Request.Context()
	err = h.service.DeleteKey(ctx, uint(keyId))
	if err != nil {
		if err.Error() == fmt.Sprintf("未找到 ID 为 %d 的密钥", keyId) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "密钥未找到",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("删除密钥失败: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "密钥已成功删除",
	})
}

// UpdateKey godoc
// @Summary      更新指定密钥信息
// @Description  更新指定密钥信息
// @Tags         keys
// @Accept       json
// @Produce      json
// @Param        keyId       path      int                             true  "密钥 ID"
// @Param        request     body      types.APIKey                    true  "更新密钥的请求体"
// @Success      200         {object}  types.APIKey                      "更新后的密钥信息 (不包含 value)"
// @Failure      400         {object}  map[string]interface{}            "请求参数错误"
// @Failure      404         {object}  map[string]interface{}            "密钥未找到"
// @Failure      500         {object}  map[string]interface{}            "服务器内部错误"
// @Router       /api/keys/{keyId} [put]
func (h *Handler) UpdateKey(c *gin.Context) {
	keyId, err := strconv.ParseUint(c.Param("keyId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "无效的密钥 ID",
		})
		return
	}

	var key types.APIKey
	if err := c.ShouldBindJSON(&key); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("无法解析请求体: %v", err),
		})
		return
	}

	ctx := c.Request.Context()
	updatedKey, err := h.service.UpdateKey(ctx, uint(keyId), key)
	if err != nil {
		// 检查错误类型
		if err.Error() == fmt.Sprintf("未找到 ID 为 %d 的密钥", keyId) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "密钥未找到",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("更新密钥失败: %v", err),
		})
		return
	}

	// 出于安全考虑，不返回密钥值
	updatedKey.Value = ""
	c.JSON(http.StatusOK, updatedKey)
}

// UpdateKeyHealth godoc
// @Summary      更新密钥健康状态
// @Description  通过 enabled 字段启用或禁用密钥健康状态
// @Tags         keys
// @Accept       json
// @Produce      json
// @Param        keyId       path      int                  true  "密钥 ID"
// @Param        request     body      HealthUpdateRequest  true  "健康状态更新请求"
// @Success      200  {object}  map[string]interface{}  "操作成功"
// @Failure      400  {object}  map[string]interface{}  "请求参数错误"
// @Failure      404  {object}  map[string]interface{}  "密钥未找到"
// @Failure      500  {object}  map[string]interface{}  "服务器内部错误"
// @Router       /api/keys/{keyId}/health [patch]
func (h *Handler) UpdateKeyHealth(c *gin.Context) {
	var req HealthUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("无法解析请求体: %v", err),
		})
		return
	}
	h.updateKeyHealthWithEnabled(c, req.Enabled)
}

func (h *Handler) updateKeyHealthWithEnabled(c *gin.Context, enabled *bool) {
	keyId, err := strconv.ParseUint(c.Param("keyId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "无效的密钥 ID",
		})
		return
	}

	if enabled == nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "必须提供 enabled 字段",
		})
		return
	}

	// 验证密钥是否存在
	ctx := c.Request.Context()
	_, err = h.service.GetKey(ctx, uint(keyId))
	if err != nil {
		if err.Error() == fmt.Sprintf("未找到 ID 为 %d 的密钥", keyId) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "密钥未找到",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("获取密钥失败: %v", err),
		})
		return
	}

	if *enabled {
		if err := h.healthService.EnableHealth(types.ResourceTypeAPIKey, uint(keyId)); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": fmt.Sprintf("启用密钥健康状态失败: %v", err),
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"message": "密钥已启用",
			"key_id":  keyId,
			"status":  "unknown",
		})
		return
	}

	if err := h.healthService.DisableHealth(types.ResourceTypeAPIKey, uint(keyId)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("禁用密钥健康状态失败: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "密钥已禁用",
		"key_id":  keyId,
		"status":  "unavailable",
	})
}
