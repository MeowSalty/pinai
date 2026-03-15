package health

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/MeowSalty/pinai/database/types"
	"github.com/MeowSalty/pinai/services/health"
)

// Handler 健康状态处理器结构体
type Handler struct {
	healthService health.Service
}

// NewHandler 创建健康状态处理器实例
//
// 参数：
//
//	healthService - 健康服务接口实例
//
// 返回值：
//
//	*Handler - 健康状态处理器实例指针
func NewHandler(healthService health.Service) *Handler {
	return &Handler{healthService: healthService}
}

// GetHealthSummary 获取健康状态统计
//
// 返回值：
//
//	成功 - 健康状态统计数据
//	失败 - 错误信息
func (h *Handler) GetHealthSummary(c *gin.Context) {
	summary, err := h.healthService.GetHealthSummary(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取健康状态统计失败：" + err.Error()})
		return
	}

	c.JSON(http.StatusOK, summary)
}

// GetPlatformHealthList 获取平台健康列表
//
// 查询参数：
//
//	page - 页码，默认为 1
//	page_size - 每页大小，默认为 10，最大 100
//
// 返回值：
//
//	成功 - 平台健康列表数据
//	失败 - 错误信息
func (h *Handler) GetPlatformHealthList(c *gin.Context) {
	// 解析分页参数
	pageStr := c.DefaultQuery("page", "1")
	page, _ := strconv.Atoi(pageStr)
	if page <= 0 {
		page = 1
	}

	pageSizeStr := c.DefaultQuery("page_size", "10")
	pageSize, _ := strconv.Atoi(pageSizeStr)
	if pageSize <= 0 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}

	// 调用服务获取平台健康列表
	result, err := h.healthService.GetPlatformHealthList(c.Request.Context(), page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取平台健康列表失败：" + err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// EnablePlatform 启用/恢复指定平台的健康状态
//
// 路径参数：
//
//	platformId - 平台 ID
//
// 返回值：
//
//	成功 - 操作成功消息
//	失败 - 错误信息
func (h *Handler) EnablePlatform(c *gin.Context) {
	platformId, err := strconv.ParseUint(c.Param("platformId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "无效的平台 ID",
		})
		return
	}

	// 启用健康状态
	if err := h.healthService.EnableHealth(types.ResourceTypePlatform, uint(platformId)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("启用平台健康状态失败: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":     "平台已启用",
		"platform_id": platformId,
		"status":      "unknown",
	})
}

// DisablePlatform 禁用指定平台的健康状态
//
// 路径参数：
//
//	platformId - 平台 ID
//
// 返回值：
//
//	成功 - 操作成功消息
//	失败 - 错误信息
func (h *Handler) DisablePlatform(c *gin.Context) {
	platformId, err := strconv.ParseUint(c.Param("platformId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "无效的平台 ID",
		})
		return
	}

	// 禁用健康状态
	if err := h.healthService.DisableHealth(types.ResourceTypePlatform, uint(platformId)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("禁用平台健康状态失败: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":     "平台已禁用",
		"platform_id": platformId,
		"status":      "unavailable",
	})
}

// GetAPIKeyHealthList 获取密钥健康列表
//
// 查询参数：
//
//	page - 页码，默认为 1
//	page_size - 每页大小，默认为 10，最大 100
//
// 返回值：
//
//	成功 - 密钥健康列表数据
//	失败 - 错误信息
func (h *Handler) GetAPIKeyHealthList(c *gin.Context) {
	// 解析分页参数
	pageStr := c.DefaultQuery("page", "1")
	page, _ := strconv.Atoi(pageStr)
	if page <= 0 {
		page = 1
	}

	pageSizeStr := c.DefaultQuery("page_size", "10")
	pageSize, _ := strconv.Atoi(pageSizeStr)
	if pageSize <= 0 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}

	// 调用服务获取密钥健康列表
	result, err := h.healthService.GetAPIKeyHealthList(c.Request.Context(), page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取密钥健康列表失败：" + err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// EnableAPIKey 启用/恢复指定密钥的健康状态
//
// 路径参数：
//
//	keyId - 密钥 ID
//
// 返回值：
//
//	成功 - 操作成功消息
//	失败 - 错误信息
func (h *Handler) EnableAPIKey(c *gin.Context) {
	keyId, err := strconv.ParseUint(c.Param("keyId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "无效的密钥 ID",
		})
		return
	}

	// 启用健康状态
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
}

// DisableAPIKey 禁用指定密钥的健康状态
//
// 路径参数：
//
//	keyId - 密钥 ID
//
// 返回值：
//
//	成功 - 操作成功消息
//	失败 - 错误信息
func (h *Handler) DisableAPIKey(c *gin.Context) {
	keyId, err := strconv.ParseUint(c.Param("keyId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "无效的密钥 ID",
		})
		return
	}

	// 禁用健康状态
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

// GetModelHealthList 获取模型健康列表
//
// 查询参数：
//
//	page - 页码，默认为 1
//	page_size - 每页大小，默认为 10，最大 100
//
// 返回值：
//
//	成功 - 模型健康列表数据
//	失败 - 错误信息
func (h *Handler) GetModelHealthList(c *gin.Context) {
	// 解析分页参数
	pageStr := c.DefaultQuery("page", "1")
	page, _ := strconv.Atoi(pageStr)
	if page <= 0 {
		page = 1
	}

	pageSizeStr := c.DefaultQuery("page_size", "10")
	pageSize, _ := strconv.Atoi(pageSizeStr)
	if pageSize <= 0 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}

	// 调用服务获取模型健康列表
	result, err := h.healthService.GetModelHealthList(c.Request.Context(), page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取模型健康列表失败：" + err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// EnableModel 启用/恢复指定模型的健康状态
//
// 路径参数：
//
//	modelId - 模型 ID
//
// 返回值：
//
//	成功 - 操作成功消息
//	失败 - 错误信息
func (h *Handler) EnableModel(c *gin.Context) {
	modelId, err := strconv.ParseUint(c.Param("modelId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "无效的模型 ID",
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

// DisableModel 禁用指定模型的健康状态
//
// 路径参数：
//
//	modelId - 模型 ID
//
// 返回值：
//
//	成功 - 操作成功消息
//	失败 - 错误信息
func (h *Handler) DisableModel(c *gin.Context) {
	modelId, err := strconv.ParseUint(c.Param("modelId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "无效的模型 ID",
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

// GetIssues 获取异常资源列表
//
// 返回值：
//
//	成功 - 异常资源列表数据
//	失败 - 错误信息
func (h *Handler) GetIssues(c *gin.Context) {
	result, err := h.healthService.GetIssues(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取异常资源列表失败：" + err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}
