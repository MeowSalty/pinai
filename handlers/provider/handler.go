package provider

import (
	"context"
	"fmt"
	"strconv"

	"github.com/MeowSalty/pinai/database/types"
	"github.com/MeowSalty/pinai/services/provider"

	"github.com/gofiber/fiber/v2"
)

// Handler 结构体封装了 LLM 服务
type Handler struct {
	service provider.Service
}

// NewHandler 创建一个新的 Handler 实例
//
// 参数：
//   - service: provider.Service 服务接口实例
//
// 返回值：
//   - *Handler: Handler 实例指针
func NewHandler(service provider.Service) *Handler {
	return &Handler{service: service}
}

// CreateProvider godoc
// @Summary      创建一个新的供应方
// @Description  创建一个新的供应方，包括平台、模型和密钥
// @Tags         providers
// @Accept       json
// @Produce      json
// @Param        request  body      provider.CreateRequest  true  "创建供应方的请求体"
// @Success      201      {object}  types.Platform                    "创建成功的供应方信息"
// @Failure      400      {object}  map[string]interface{}            "请求参数错误"
// @Failure      500      {object}  map[string]interface{}            "服务器内部错误"
// @Router       /api/providers [post]
func (h *Handler) CreateProvider(c *fiber.Ctx) error {
	var req provider.CreateRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("无法解析请求体: %v", err),
		})
	}

	ctx := context.Background()
	platform, err := h.service.CreateProvider(ctx, req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("创建供应方失败: %v", err),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(platform)
}

// GetProviders godoc
// @Summary      获取所有供应方列表
// @Description  获取所有供应方 (平台) 列表
// @Tags         providers
// @Produce      json
// @Success      200  {array}   types.Platform                    "供应方列表"
// @Failure      500  {object}  map[string]interface{}            "服务器内部错误"
// @Router       /api/providers [get]
func (h *Handler) GetProviders(c *fiber.Ctx) error {
	ctx := context.Background()
	platforms, err := h.service.GetProviders(ctx)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("获取供应方列表失败: %v", err),
		})
	}

	return c.JSON(platforms)
}

// GetProvider godoc
// @Summary      获取指定供应方详情
// @Description  获取指定供应方 (平台) 详情
// @Tags         providers
// @Produce      json
// @Param        id   path      int  true  "供应方 ID"
// @Success      200  {object}  types.Platform                    "供应方详情"
// @Failure      400  {object}  map[string]interface{}            "请求参数错误"
// @Failure      404  {object}  map[string]interface{}            "供应方未找到"
// @Failure      500  {object}  map[string]interface{}            "服务器内部错误"
// @Router       /api/providers/{id} [get]
func (h *Handler) GetProvider(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "无效的供应方 ID",
		})
	}

	ctx := context.Background()
	platform, err := h.service.GetProvider(ctx, uint(id))
	if err != nil {
		// 检查错误类型，如果未找到则返回 404
		if err.Error() == fmt.Sprintf("未找到 ID 为 %d 的供应方", id) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "供应方未找到",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("获取供应方详情失败: %v", err),
		})
	}

	return c.JSON(platform)
}

// UpdateProvider godoc
// @Summary      更新指定供应方信息
// @Description  更新指定供应方 (平台) 信息
// @Tags         providers
// @Accept       json
// @Produce      json
// @Param        id       path      int                             true  "供应方 ID"
// @Param        request  body      types.Platform                  true  "更新供应方的请求体"
// @Success      200      {object}  types.Platform                    "更新后的供应方信息"
// @Failure      400      {object}  map[string]interface{}            "请求参数错误"
// @Failure      404      {object}  map[string]interface{}            "供应方未找到"
// @Failure      500      {object}  map[string]interface{}            "服务器内部错误"
// @Router       /api/providers/{id} [put]
func (h *Handler) UpdateProvider(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "无效的供应方 ID",
		})
	}

	var platform types.Platform
	if err := c.BodyParser(&platform); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("无法解析请求体: %v", err),
		})
	}

	ctx := context.Background()
	updatedPlatform, err := h.service.UpdateProvider(ctx, uint(id), platform)
	if err != nil {
		// 检查错误类型
		if err.Error() == fmt.Sprintf("未找到 ID 为 %d 的供应方", id) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "供应方未找到",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("更新供应方失败: %v", err),
		})
	}

	return c.JSON(updatedPlatform)
}

// DeleteProvider godoc
// @Summary      删除指定供应方
// @Description  删除指定供应方 (平台) (将级联删除模型和密钥)
// @Tags         providers
// @Produce      json
// @Param        id   path      int  true  "供应方 ID"
// @Success      200  {object}  map[string]interface{}            "删除成功消息"
// @Failure      400  {object}  map[string]interface{}            "请求参数错误"
// @Failure      404  {object}  map[string]interface{}            "供应方未找到"
// @Failure      500  {object}  map[string]interface{}            "服务器内部错误"
// @Router       /api/providers/{id} [delete]
func (h *Handler) DeleteProvider(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "无效的供应方 ID",
		})
	}

	ctx := context.Background()
	err = h.service.DeleteProvider(ctx, uint(id))
	if err != nil {
		// 检查错误类型
		if err.Error() == fmt.Sprintf("未找到 ID 为 %d 的供应方", id) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "供应方未找到",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("删除供应方失败: %v", err),
		})
	}

	return c.JSON(fiber.Map{
		"message": "供应方及其关联资源已成功删除",
	})
}

// AddModelToProvider godoc
// @Summary      为指定供应方添加新模型
// @Description  为指定供应方添加新模型
// @Tags         models
// @Accept       json
// @Produce      json
// @Param        providerId  path      int                             true  "供应方 ID"
// @Param        request     body      types.Model                     true  "创建模型的请求体"
// @Success      201         {object}  types.Model                       "创建成功的模型信息"
// @Failure      400         {object}  map[string]interface{}            "请求参数错误"
// @Failure      404         {object}  map[string]interface{}            "供应方未找到"
// @Failure      500         {object}  map[string]interface{}            "服务器内部错误"
// @Router       /api/providers/{providerId}/models [post]
func (h *Handler) AddModelToProvider(c *fiber.Ctx) error {
	providerId, err := strconv.ParseUint(c.Params("providerId"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "无效的供应方 ID",
		})
	}

	var model types.Model
	if err := c.BodyParser(&model); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("无法解析请求体: %v", err),
		})
	}

	ctx := context.Background()
	createdModel, err := h.service.AddModelToProvider(ctx, uint(providerId), model)
	if err != nil {
		// 检查错误类型
		if err.Error() == fmt.Sprintf("未找到 ID 为 %d 的供应方", providerId) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "供应方未找到",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("为供应方添加模型失败: %v", err),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(createdModel)
}

// GetModelsByProvider godoc
// @Summary      获取指定供应方的所有模型列表
// @Description  获取指定供应方的所有模型列表
// @Tags         models
// @Produce      json
// @Param        providerId  path      int  true  "供应方 ID"
// @Success      200         {array}   types.Model                       "模型列表"
// @Failure      400         {object}  map[string]interface{}            "请求参数错误"
// @Failure      404         {object}  map[string]interface{}            "供应方未找到"
// @Failure      500         {object}  map[string]interface{}            "服务器内部错误"
// @Router       /api/providers/{providerId}/models [get]
func (h *Handler) GetModelsByProvider(c *fiber.Ctx) error {
	providerId, err := strconv.ParseUint(c.Params("providerId"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "无效的供应方 ID",
		})
	}

	ctx := context.Background()
	models, err := h.service.GetModelsByProvider(ctx, uint(providerId))
	if err != nil {
		// 检查错误类型
		if err.Error() == fmt.Sprintf("未找到 ID 为 %d 的供应方", providerId) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "供应方未找到",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("获取供应方模型列表失败: %v", err),
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
// @Param        providerId  path      int                             true  "供应方 ID"
// @Param        modelId     path      int                             true  "模型 ID"
// @Param        request     body      types.Model                     true  "更新模型的请求体"
// @Success      200         {object}  types.Model                       "更新后的模型信息"
// @Failure      400         {object}  map[string]interface{}            "请求参数错误"
// @Failure      404         {object}  map[string]interface{}            "供应方或模型未找到"
// @Failure      500         {object}  map[string]interface{}            "服务器内部错误"
// @Router       /api/providers/{providerId}/models/{modelId} [put]
func (h *Handler) UpdateModel(c *fiber.Ctx) error {
	providerId, err := strconv.ParseUint(c.Params("providerId"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "无效的供应方 ID",
		})
	}

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
	updatedModel, err := h.service.UpdateModel(ctx, uint(providerId), uint(modelId), model)
	if err != nil {
		// 检查错误类型
		if err.Error() == fmt.Sprintf("未找到 ID 为 %d 的供应方", providerId) ||
			err.Error() == fmt.Sprintf("在供应方 %d 中未找到 ID 为 %d 的模型", providerId, modelId) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "供应方或模型未找到",
			})
		}
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
// @Param        providerId  path      int  true  "供应方 ID"
// @Param        modelId     path      int  true  "模型 ID"
// @Success      200         {object}  map[string]interface{}            "删除成功消息"
// @Failure      400         {object}  map[string]interface{}            "请求参数错误"
// @Failure      404         {object}  map[string]interface{}            "供应方或模型未找到"
// @Failure      500         {object}  map[string]interface{}            "服务器内部错误"
// @Router       /api/providers/{providerId}/models/{modelId} [delete]
func (h *Handler) DeleteModel(c *fiber.Ctx) error {
	providerId, err := strconv.ParseUint(c.Params("providerId"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "无效的供应方 ID",
		})
	}

	modelId, err := strconv.ParseUint(c.Params("modelId"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "无效的模型 ID",
		})
	}

	ctx := context.Background()
	err = h.service.DeleteModel(ctx, uint(providerId), uint(modelId))
	if err != nil {
		// 检查错误类型
		if err.Error() == fmt.Sprintf("未找到 ID 为 %d 的供应方", providerId) ||
			err.Error() == fmt.Sprintf("在供应方 %d 中未找到 ID 为 %d 的模型", providerId, modelId) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "供应方或模型未找到",
			})
		}
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

// AddKeyToProvider godoc
// @Summary      为指定供应方添加新密钥
// @Description  为指定供应方添加新密钥
// @Tags         keys
// @Accept       json
// @Produce      json
// @Param        providerId  path      int                             true  "供应方 ID"
// @Param        request     body      types.APIKey                    true  "创建密钥的请求体"
// @Success      201         {object}  types.APIKey                      "创建成功的密钥信息 (不包含 value)"
// @Failure      400         {object}  map[string]interface{}            "请求参数错误"
// @Failure      404         {object}  map[string]interface{}            "供应方未找到"
// @Failure      500         {object}  map[string]interface{}            "服务器内部错误"
// @Router       /api/providers/{providerId}/keys [post]
func (h *Handler) AddKeyToProvider(c *fiber.Ctx) error {
	providerId, err := strconv.ParseUint(c.Params("providerId"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "无效的供应方 ID",
		})
	}

	var key types.APIKey
	if err := c.BodyParser(&key); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("无法解析请求体: %v", err),
		})
	}

	ctx := context.Background()
	createdKey, err := h.service.AddKeyToProvider(ctx, uint(providerId), key)
	if err != nil {
		// 检查错误类型
		if err.Error() == fmt.Sprintf("未找到 ID 为 %d 的供应方", providerId) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "供应方未找到",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("为供应方添加密钥失败: %v", err),
		})
	}

	// 出于安全考虑，不返回密钥值
	createdKey.Value = ""
	return c.Status(fiber.StatusCreated).JSON(createdKey)
}

// GetKeysByProvider godoc
// @Summary      获取指定供应方的所有密钥列表
// @Description  获取指定供应方的所有密钥列表 (不包含密钥值)
// @Tags         keys
// @Produce      json
// @Param        providerId  path      int  true  "供应方 ID"
// @Success      200         {array}   types.APIKey                      "密钥列表 (不包含 value)"
// @Failure      400         {object}  map[string]interface{}            "请求参数错误"
// @Failure      404         {object}  map[string]interface{}            "供应方未找到"
// @Failure      500         {object}  map[string]interface{}            "服务器内部错误"
// @Router       /api/providers/{providerId}/keys [get]
func (h *Handler) GetKeysByProvider(c *fiber.Ctx) error {
	providerId, err := strconv.ParseUint(c.Params("providerId"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "无效的供应方 ID",
		})
	}

	ctx := context.Background()
	keys, err := h.service.GetKeysByProvider(ctx, uint(providerId))
	if err != nil {
		// 检查错误类型
		if err.Error() == fmt.Sprintf("未找到 ID 为 %d 的供应方", providerId) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "供应方未找到",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("获取供应方密钥列表失败: %v", err),
		})
	}

	return c.JSON(keys)
}

// DeleteKey godoc
// @Summary      删除指定密钥
// @Description  删除指定密钥
// @Tags         keys
// @Produce      json
// @Param        providerId  path      int  true  "供应方 ID"
// @Param        keyId       path      int  true  "密钥 ID"
// @Success      200         {object}  map[string]interface{}            "删除成功消息"
// @Failure      400         {object}  map[string]interface{}            "请求参数错误"
// @Failure      404         {object}  map[string]interface{}            "供应方或密钥未找到"
// @Failure      500         {object}  map[string]interface{}            "服务器内部错误"
// @Router       /api/providers/{providerId}/keys/{keyId} [delete]
func (h *Handler) DeleteKey(c *fiber.Ctx) error {
	providerId, err := strconv.ParseUint(c.Params("providerId"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "无效的供应方 ID",
		})
	}

	keyId, err := strconv.ParseUint(c.Params("keyId"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "无效的密钥 ID",
		})
	}

	ctx := context.Background()
	err = h.service.DeleteKey(ctx, uint(providerId), uint(keyId))
	if err != nil {
		// 检查错误类型
		if err.Error() == fmt.Sprintf("未找到 ID 为 %d 的供应方", providerId) ||
			err.Error() == fmt.Sprintf("在供应方 %d 中未找到 ID 为 %d 的密钥", providerId, keyId) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "供应方或密钥未找到",
			})
		}
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
// @Param        providerId  path      int                             true  "供应方 ID"
// @Param        keyId       path      int                             true  "密钥 ID"
// @Param        request     body      types.APIKey                    true  "更新密钥的请求体"
// @Success      200         {object}  types.APIKey                      "更新后的密钥信息 (不包含 value)"
// @Failure      400         {object}  map[string]interface{}            "请求参数错误"
// @Failure      404         {object}  map[string]interface{}            "供应方或密钥未找到"
// @Failure      500         {object}  map[string]interface{}            "服务器内部错误"
// @Router       /api/providers/{providerId}/keys/{keyId} [put]
func (h *Handler) UpdateKey(c *fiber.Ctx) error {
	providerId, err := strconv.ParseUint(c.Params("providerId"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "无效的供应方 ID",
		})
	}

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
	updatedKey, err := h.service.UpdateKey(ctx, uint(providerId), uint(keyId), key)
	if err != nil {
		// 检查错误类型
		if err.Error() == fmt.Sprintf("未找到 ID 为 %d 的供应方", providerId) ||
			err.Error() == fmt.Sprintf("在供应方 %d 中未找到 ID 为 %d 的密钥", providerId, keyId) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "供应方或密钥未找到",
			})
		}
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
