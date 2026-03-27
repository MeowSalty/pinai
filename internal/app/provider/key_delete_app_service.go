package provider

import (
	"context"
	"fmt"
	"log/slog"
)

// deleteKeyApp 以应用服务方式实现单个密钥删除。
// 事务边界在应用服务层声明，仓储仅负责数据访问。
func (s *service) deleteKeyApp(ctx context.Context, keyID uint) error {
	logger := s.logger.With(
		slog.String("operation", "key_delete"),
		slog.Uint64("key_id", uint64(keyID)),
	)
	logger.Debug("开始删除 API 密钥")

	if s.keyControlRepo == nil {
		return fmt.Errorf("删除 API 密钥失败：密钥控制仓储未初始化")
	}
	if s.controlTx == nil {
		return fmt.Errorf("删除 API 密钥失败：事务执行器未初始化")
	}

	apiKey, err := s.keyControlRepo.GetAPIKey(ctx, keyID)
	if err != nil {
		logger.Warn("查询 API 密钥失败", slog.Any("error", err))
		_ = s.logKeyDeleteAudit(ctx, keyID, "failed", fmt.Sprintf("查询 API 密钥失败：%v", err))
		return err
	}

	backupModels, err := s.keyControlRepo.ListModelsByAPIKey(ctx, keyID)
	if err != nil {
		logger.Error("查询密钥关联模型失败", slog.Any("error", err))
		_ = s.logKeyDeleteAudit(ctx, keyID, "failed", fmt.Sprintf("查询密钥关联模型失败：%v", err))
		return fmt.Errorf("查询密钥关联模型失败：%w", err)
	}

	relationCount := len(backupModels)
	if relationCount > 0 {
		logger.Debug("开始清理密钥与模型关联关系", slog.Int("model_count", relationCount))
		if err = s.keyControlRepo.ClearAPIKeyModelRelations(ctx, keyID); err != nil {
			logger.Error("清理密钥与模型关联关系失败", slog.Any("error", err))
			_ = s.logKeyDeleteAudit(ctx, keyID, "failed", fmt.Sprintf("清理密钥与模型关联关系失败：%v", err))
			return fmt.Errorf("清理密钥与模型关联关系失败：%w", err)
		}
	}

	err = s.controlTx.WithinTx(ctx, func(txCtx context.Context) error {
		rowsAffected, innerErr := s.keyControlRepo.DeleteAPIKeyByID(txCtx, keyID)
		if innerErr != nil {
			return fmt.Errorf("删除 API 密钥失败：%w", innerErr)
		}
		if rowsAffected == 0 {
			return fmt.Errorf("未找到 ID 为 %d 的 API 密钥：%w", keyID, ErrResourceNotFound)
		}
		return nil
	})
	if err != nil {
		if relationCount > 0 {
			logger.Warn("删除事务失败，开始恢复密钥与模型关联关系", slog.Any("error", err))
			if restoreErr := s.keyControlRepo.AppendAPIKeyModels(ctx, keyID, backupModels); restoreErr != nil {
				logger.Error("恢复密钥与模型关联关系失败", slog.Any("error", restoreErr))
			}
		}
		_ = s.logKeyDeleteAudit(ctx, keyID, "failed", fmt.Sprintf("删除 API 密钥失败：%v", err))
		return fmt.Errorf("删除 API 密钥失败：%w", err)
	}

	orphanedCount, err := s.removeOrphanedModels(ctx, apiKey.PlatformID, logger)
	if err != nil {
		logger.Error("删除孤立模型失败", slog.Any("error", err))
		_ = s.logKeyDeleteAudit(ctx, keyID, "failed", fmt.Sprintf("删除孤立模型失败：%v", err))
		return err
	}

	logger.Info("成功删除 API 密钥",
		slog.Uint64("platform_id", uint64(apiKey.PlatformID)),
		slog.Int("model_relation_count", relationCount),
		slog.Int64("orphaned_model_deleted_count", orphanedCount),
	)
	_ = s.logKeyDeleteAudit(ctx, keyID, "success", fmt.Sprintf("删除 API 密钥成功，platform_id=%d，关联模型数=%d，删除孤立模型数=%d", apiKey.PlatformID, relationCount, orphanedCount))

	return nil
}

func (s *service) logKeyDeleteAudit(ctx context.Context, keyID uint, result, detail string) error {
	if s.controlAudit == nil {
		return nil
	}

	return s.controlAudit.Log(ctx, ControlAuditEvent{
		Action:     "key.delete",
		Resource:   "key",
		ResourceID: keyID,
		Result:     result,
		Detail:     detail,
	})
}
