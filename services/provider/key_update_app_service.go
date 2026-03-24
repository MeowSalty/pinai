package provider

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/MeowSalty/pinai/database/types"
)

// updateKeyApp 以应用服务方式实现单个密钥更新。
// 事务边界在应用服务层声明，仓储仅负责数据访问。
func (s *service) updateKeyApp(ctx context.Context, keyID uint, key types.APIKey) (*types.APIKey, error) {
	logger := s.logger.With(
		slog.String("operation", "key_update"),
		slog.Uint64("key_id", uint64(keyID)),
	)
	logger.Debug("开始更新 API 密钥")

	if s.keyControlRepo == nil {
		return nil, fmt.Errorf("更新 API 密钥失败：密钥控制仓储未初始化")
	}
	if s.controlTx == nil {
		return nil, fmt.Errorf("更新 API 密钥失败：事务执行器未初始化")
	}

	_, err := s.keyControlRepo.GetAPIKey(ctx, keyID)
	if err != nil {
		logger.Warn("查询 API 密钥失败", slog.Any("error", err))
		_ = s.logKeyUpdateAudit(ctx, keyID, "failed", fmt.Sprintf("查询 API 密钥失败：%v", err))
		return nil, err
	}

	err = s.controlTx.WithinTx(ctx, func(txCtx context.Context) error {
		rowsAffected, innerErr := s.keyControlRepo.UpdateAPIKey(txCtx, keyID, key)
		if innerErr != nil {
			return fmt.Errorf("更新 API 密钥失败：%w", innerErr)
		}
		if rowsAffected == 0 {
			return fmt.Errorf("未找到 ID 为 %d 的密钥：%w", keyID, ErrResourceNotFound)
		}
		return nil
	})
	if err != nil {
		logger.Error("更新 API 密钥事务失败", slog.Any("error", err))
		_ = s.logKeyUpdateAudit(ctx, keyID, "failed", fmt.Sprintf("更新 API 密钥失败：%v", err))
		return nil, fmt.Errorf("更新 API 密钥失败：%w", err)
	}

	updatedKey, err := s.keyControlRepo.GetAPIKey(ctx, keyID)
	if err != nil {
		logger.Error("获取更新后的 API 密钥失败", slog.Any("error", err))
		_ = s.logKeyUpdateAudit(ctx, keyID, "failed", fmt.Sprintf("获取更新后的 API 密钥失败：%v", err))
		return nil, err
	}

	logger.Info("成功更新 API 密钥", slog.Uint64("platform_id", uint64(updatedKey.PlatformID)))
	_ = s.logKeyUpdateAudit(ctx, keyID, "success", fmt.Sprintf("更新 API 密钥成功，platform_id=%d", updatedKey.PlatformID))

	return updatedKey, nil
}

func (s *service) logKeyUpdateAudit(ctx context.Context, keyID uint, result, detail string) error {
	if s.controlAudit == nil {
		return nil
	}

	return s.controlAudit.Log(ctx, ControlAuditEvent{
		Action:     "key.update",
		Resource:   "key",
		ResourceID: keyID,
		Result:     result,
		Detail:     detail,
	})
}
