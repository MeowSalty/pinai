package provider

import (
	"context"
	"fmt"
	"strconv"

	"github.com/MeowSalty/pinai/database/types"
	"github.com/MeowSalty/pinai/services/provider"

	"github.com/gofiber/fiber/v2"
)

// AddEndpointToPlatform godoc
// @Summary      为指定平台添加新端点
// @Description  为指定平台添加新端点
// @Tags         endpoints
// @Accept       json
// @Produce      json
// @Param        platformId  path      int                              true  "平台 ID"
// @Param        request     body      types.Endpoint                   true  "创建端点的请求体"
// @Success      201         {object}  types.Endpoint                    "创建成功的端点信息"
// @Failure      400         {object}  map[string]interface{}            "请求参数错误"
// @Failure      404         {object}  map[string]interface{}            "平台未找到"
// @Failure      500         {object}  map[string]interface{}            "服务器内部错误"
// @Router       /api/platforms/{platformId}/endpoints [post]
func (h *Handler) AddEndpointToPlatform(c *fiber.Ctx) error {
	platformId, err := strconv.ParseUint(c.Params("platformId"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "无效的平台 ID",
		})
	}

	var endpoint types.Endpoint
	if err := c.BodyParser(&endpoint); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("无法解析请求体: %v", err),
		})
	}

	ctx := context.Background()
	createdEndpoint, err := h.service.AddEndpointToPlatform(ctx, uint(platformId), endpoint)
	if err != nil {
		if err.Error() == fmt.Sprintf("未找到 ID 为 %d 的平台", platformId) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "平台未找到",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("为平台添加端点失败: %v", err),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(createdEndpoint)
}

// BatchAddEndpointsToPlatform godoc
// @Summary      批量为指定平台添加端点
// @Description  批量创建多个端点，采用原子性事务（全部成功或全部失败）
// @Tags         endpoints
// @Accept       json
// @Produce      json
// @Param        platformId  path      int                                         true  "平台 ID"
// @Param        request     body      provider.BatchCreateEndpointsRequest         true  "批量创建端点的请求体"
// @Success      201         {object}  provider.BatchCreateEndpointsResponse        "全部创建成功"
// @Failure      400         {object}  map[string]interface{}                        "请求参数错误"
// @Failure      404         {object}  map[string]interface{}                        "平台未找到"
// @Failure      500         {object}  map[string]interface{}                        "服务器内部错误"
// @Router       /api/platforms/{platformId}/endpoints/batch [post]
func (h *Handler) BatchAddEndpointsToPlatform(c *fiber.Ctx) error {
	platformId, err := strconv.ParseUint(c.Params("platformId"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "无效的平台 ID",
		})
	}

	var req provider.BatchCreateEndpointsRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("无法解析请求体: %v", err),
		})
	}

	if len(req.Endpoints) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "必须至少提供一个端点",
		})
	}

	ctx := context.Background()
	createdEndpoints, err := h.service.BatchAddEndpointsToPlatform(ctx, uint(platformId), req.Endpoints)
	if err != nil {
		if err.Error() == fmt.Sprintf("未找到 ID 为 %d 的平台", platformId) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "平台未找到",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("批量创建端点失败: %v", err),
		})
	}

	response := provider.BatchCreateEndpointsResponse{
		Endpoints:    createdEndpoints,
		TotalCount:   len(req.Endpoints),
		CreatedCount: len(createdEndpoints),
	}

	return c.Status(fiber.StatusCreated).JSON(response)
}

// GetEndpointsByPlatform godoc
// @Summary      获取指定平台的所有端点列表
// @Description  获取指定平台的所有端点列表
// @Tags         endpoints
// @Produce      json
// @Param        platformId  path      int     true  "平台 ID"
// @Success      200         {array}   types.Endpoint                   "端点列表"
// @Failure      400         {object}  map[string]interface{}           "请求参数错误"
// @Failure      404         {object}  map[string]interface{}           "平台未找到"
// @Failure      500         {object}  map[string]interface{}           "服务器内部错误"
// @Router       /api/platforms/{platformId}/endpoints [get]
func (h *Handler) GetEndpointsByPlatform(c *fiber.Ctx) error {
	platformId, err := strconv.ParseUint(c.Params("platformId"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "无效的平台 ID",
		})
	}

	ctx := context.Background()
	endpoints, err := h.service.GetEndpointsByPlatform(ctx, uint(platformId))
	if err != nil {
		if err.Error() == fmt.Sprintf("未找到 ID 为 %d 的平台", platformId) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "平台未找到",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("获取平台端点列表失败: %v", err),
		})
	}

	return c.JSON(endpoints)
}

// GetEndpoint godoc
// @Summary      获取指定端点详情
// @Description  获取指定端点详情
// @Tags         endpoints
// @Produce      json
// @Param        platformId  path      int     true  "平台 ID"
// @Param        endpointId  path      int     true  "端点 ID"
// @Success      200         {object}  types.Endpoint                   "端点详情"
// @Failure      400         {object}  map[string]interface{}           "请求参数错误"
// @Failure      404         {object}  map[string]interface{}           "端点未找到"
// @Failure      500         {object}  map[string]interface{}           "服务器内部错误"
// @Router       /api/platforms/{platformId}/endpoints/{endpointId} [get]
func (h *Handler) GetEndpoint(c *fiber.Ctx) error {
	endpointId, err := strconv.ParseUint(c.Params("endpointId"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "无效的端点 ID",
		})
	}

	ctx := context.Background()
	endpoint, err := h.service.GetEndpoint(ctx, uint(endpointId))
	if err != nil {
		if err.Error() == fmt.Sprintf("未找到 ID 为 %d 的端点", endpointId) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "端点未找到",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("获取端点详情失败: %v", err),
		})
	}

	return c.JSON(endpoint)
}

// UpdateEndpoint godoc
// @Summary      更新指定端点信息
// @Description  更新指定端点信息
// @Tags         endpoints
// @Accept       json
// @Produce      json
// @Param        platformId  path      int                              true  "平台 ID"
// @Param        endpointId  path      int                              true  "端点 ID"
// @Param        request     body      types.Endpoint                   true  "更新端点的请求体"
// @Success      200         {object}  types.Endpoint                    "更新后的端点信息"
// @Failure      400         {object}  map[string]interface{}            "请求参数错误"
// @Failure      404         {object}  map[string]interface{}            "平台或端点未找到"
// @Failure      500         {object}  map[string]interface{}            "服务器内部错误"
// @Router       /api/platforms/{platformId}/endpoints/{endpointId} [put]
func (h *Handler) UpdateEndpoint(c *fiber.Ctx) error {
	endpointId, err := strconv.ParseUint(c.Params("endpointId"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "无效的端点 ID",
		})
	}

	var endpoint types.Endpoint
	if err := c.BodyParser(&endpoint); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("无法解析请求体: %v", err),
		})
	}

	ctx := context.Background()
	updatedEndpoint, err := h.service.UpdateEndpoint(ctx, uint(endpointId), endpoint)
	if err != nil {
		if err.Error() == fmt.Sprintf("未找到 ID 为 %d 的端点", endpointId) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "端点未找到",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("更新端点失败: %v", err),
		})
	}

	return c.JSON(updatedEndpoint)
}

// BatchUpdateEndpoints godoc
// @Summary      批量更新指定平台的端点
// @Description  批量更新多个端点的信息，采用原子性事务（全部成功或全部失败）
// @Tags         endpoints
// @Accept       json
// @Produce      json
// @Param        platformId  path      int                                         true  "平台 ID"
// @Param        request     body      provider.BatchUpdateEndpointsRequest         true  "批量更新端点的请求体"
// @Success      200         {object}  provider.BatchUpdateEndpointsResponse        "全部更新成功"
// @Failure      400         {object}  map[string]interface{}                        "请求参数错误"
// @Failure      404         {object}  map[string]interface{}                        "平台或端点未找到"
// @Failure      500         {object}  map[string]interface{}                        "服务器内部错误"
// @Router       /api/platforms/{platformId}/endpoints/batch [put]
func (h *Handler) BatchUpdateEndpoints(c *fiber.Ctx) error {
	platformId, err := strconv.ParseUint(c.Params("platformId"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "无效的平台 ID",
		})
	}

	var req provider.BatchUpdateEndpointsRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("无法解析请求体: %v", err),
		})
	}

	if len(req.Endpoints) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "必须至少提供一个端点更新项",
		})
	}

	for i, item := range req.Endpoints {
		if item.ID == 0 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": fmt.Sprintf("端点更新项 %d 缺少必需的 ID 字段", i),
			})
		}
	}

	ctx := context.Background()
	updatedEndpoints, err := h.service.BatchUpdateEndpoints(ctx, uint(platformId), req.Endpoints)
	if err != nil {
		if err.Error() == fmt.Sprintf("未找到 ID 为 %d 的平台", platformId) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "平台未找到",
			})
		}
		if len(err.Error()) > 0 && (err.Error()[:6] == "未找到" || err.Error()[:6] == "端点 ID") {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("批量更新端点失败: %v", err),
		})
	}

	response := provider.BatchUpdateEndpointsResponse{
		Endpoints:    updatedEndpoints,
		TotalCount:   len(req.Endpoints),
		UpdatedCount: len(updatedEndpoints),
	}

	return c.JSON(response)
}

// DeleteEndpoint godoc
// @Summary      删除指定端点
// @Description  删除指定端点
// @Tags         endpoints
// @Produce      json
// @Param        platformId  path      int  true  "平台 ID"
// @Param        endpointId  path      int  true  "端点 ID"
// @Success      200         {object}  map[string]interface{}            "删除成功消息"
// @Failure      400         {object}  map[string]interface{}            "请求参数错误"
// @Failure      404         {object}  map[string]interface{}            "平台或端点未找到"
// @Failure      500         {object}  map[string]interface{}            "服务器内部错误"
// @Router       /api/platforms/{platformId}/endpoints/{endpointId} [delete]
func (h *Handler) DeleteEndpoint(c *fiber.Ctx) error {
	endpointId, err := strconv.ParseUint(c.Params("endpointId"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "无效的端点 ID",
		})
	}

	ctx := context.Background()
	err = h.service.DeleteEndpoint(ctx, uint(endpointId))
	if err != nil {
		if err.Error() == fmt.Sprintf("未找到 ID 为 %d 的端点", endpointId) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "端点未找到",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("删除端点失败: %v", err),
		})
	}

	return c.JSON(fiber.Map{
		"message": "端点已成功删除",
	})
}
