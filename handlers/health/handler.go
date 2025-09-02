package health

import (
	"strconv"

	"github.com/gofiber/fiber/v2"

	dbtypes "github.com/MeowSalty/pinai/database/types"
	"github.com/MeowSalty/pinai/services"
)

// HealthHandlerInterface 定义健康检查处理器接口
type HealthHandlerInterface interface {
	// GetResourceHealth 获取指定资源的详细健康状态
	GetResourceHealth(c *fiber.Ctx) error

	// GetPlatformsHealthOverview 获取所有平台的健康状态概览
	GetPlatformsHealthOverview(c *fiber.Ctx) error

	// GetModelsHealthOverview 获取所有模型的健康状态概览
	GetModelsHealthOverview(c *fiber.Ctx) error

	// GetPlatformResourcesHealth 获取指定平台下所有资源的健康状态
	GetPlatformResourcesHealth(c *fiber.Ctx) error
}

// HealthHandler 健康检查处理器结构体
type HealthHandler struct {
	HealthService services.HealthServiceInterface
}

// NewHealthHandler 创建健康检查处理器实例
//
// 参数：
//   - healthService: 健康服务接口实例
//
// 返回值：
//   - HealthHandlerInterface: 健康检查处理器接口实例
func NewHealthHandler(healthService services.HealthServiceInterface) HealthHandlerInterface {
	return &HealthHandler{
		HealthService: healthService,
	}
}

// GetResourceHealth 获取指定资源的详细健康状态
//
// 路径参数：
//   - resourceType: 资源类型 (platform, model, api_key)
//   - id: 资源 ID
//
// 返回值：
//   - 成功：资源的详细健康状态信息
//   - 失败：错误信息
func (h *HealthHandler) GetResourceHealth(c *fiber.Ctx) error {
	resourceTypeStr := c.Params("resourceType")
	idStr := c.Params("id")

	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "无效的 ID 格式",
		})
	}

	var resourceType dbtypes.ResourceType
	switch resourceTypeStr {
	case "platform":
		resourceType = dbtypes.ResourceTypePlatform
	case "model":
		resourceType = dbtypes.ResourceTypeModel
	case "api_key":
		resourceType = dbtypes.ResourceTypeAPIKey
	default:
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "无效的资源类型",
		})
	}

	healthStatus, err := h.HealthService.GetResourceHealth(c.Context(), resourceType, uint(id))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(healthStatus)
}

// GetPlatformsHealthOverview 获取所有平台的健康状态概览
//
// 返回值：
//   - 成功：平台健康状态概览信息
//   - 失败：错误信息
func (h *HealthHandler) GetPlatformsHealthOverview(c *fiber.Ctx) error {
	overview, err := h.HealthService.GetPlatformsHealthOverview(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	return c.JSON(overview)
}

// GetModelsHealthOverview 获取所有模型的健康状态概览
//
// 返回值：
//   - 成功：模型健康状态概览信息
//   - 失败：错误信息
func (h *HealthHandler) GetModelsHealthOverview(c *fiber.Ctx) error {
	overview, err := h.HealthService.GetModelsHealthOverview(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	return c.JSON(overview)
}

// GetPlatformResourcesHealth 获取指定平台下所有资源的健康状态
//
// 路径参数：
//   - id: 平台 ID
//
// 返回值：
//   - 成功：平台下所有资源的健康状态信息
//   - 失败：错误信息
func (h *HealthHandler) GetPlatformResourcesHealth(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "无效的平台 ID 格式",
		})
	}

	resources, err := h.HealthService.GetPlatformResourcesHealth(c.Context(), uint(id))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(resources)
}
