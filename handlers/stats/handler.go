package stats

import (
	"github.com/gofiber/fiber/v2"

	"github.com/MeowSalty/pinai/services"
)

// StatsHandlerInterface 定义统计处理器接口
type StatsHandlerInterface interface {
	// GetOverview 获取全局概览数据
	GetOverview(c *fiber.Ctx) error
}

// StatsHandler 统计处理器结构体
type StatsHandler struct {
	StatsService services.StatsServiceInterface
}

// NewStatsHandler 创建统计处理器实例
//
// 参数：
//   - statsService: 统计服务接口实例
//
// 返回值：
//   - StatsHandlerInterface: 统计处理器接口实例
func NewStatsHandler(statsService services.StatsServiceInterface) StatsHandlerInterface {
	return &StatsHandler{
		StatsService: statsService,
	}
}

// GetOverview 获取全局概览数据
//
// 返回值：
//   - 成功：全局概览数据
//   - 失败：错误信息
func (h *StatsHandler) GetOverview(c *fiber.Ctx) error {
	overview, err := h.StatsService.GetOverview(c.Context())
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "获取统计概览数据失败："+err.Error())
	}

	return c.JSON(overview)
}
