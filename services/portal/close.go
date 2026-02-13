package portal

import (
	"time"
)

// Close 优雅关闭服务
//
// 停止健康管理器和取消所有相关的上下文
func (s *service) Close(timeout time.Duration) error {
	s.logger.Info("开始优雅关闭服务", "timeout", timeout)

	err := s.portal.Close(timeout)
	if err != nil {
		s.logger.Error("服务关闭失败", "error", err, "timeout", timeout)
		return err
	}

	s.logger.Info("服务关闭成功")
	return nil
}
