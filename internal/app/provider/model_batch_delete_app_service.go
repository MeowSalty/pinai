package provider

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/MeowSalty/pinai/database/types"
)

// batchDeleteModelsApp 以应用服务方式实现批量删除模型。
// 事务边界在应用服务层声明，仓储仅负责数据访问。
func (s *service) batchDeleteModelsApp(ctx context.Context, platformID uint, modelIDs []uint) (int, error) {
	type modelAPIKeysBackup struct {
		modelID uint
		apiKeys []*types.APIKey
	}

	logger := s.logger.With(
		slog.String("operation", "batch_delete_models"),
		slog.Uint64("platform_id", uint64(platformID)),
		slog.Int("model_count", len(modelIDs)),
	)
	logger.Debug("开始批量删除模型")

	if s.modelControlRepo == nil {
		return 0, fmt.Errorf("批量删除模型失败：模型控制仓储未初始化")
	}
	if s.controlTx == nil {
		return 0, fmt.Errorf("批量删除模型失败：事务执行器未初始化")
	}

	if len(modelIDs) == 0 {
		logger.Warn("未提供任何模型 ID")
		return 0, fmt.Errorf("必须至少提供一个模型 ID")
	}

	exists, err := s.modelControlRepo.ExistsPlatform(ctx, platformID)
	if err != nil {
		logger.Error("检查平台是否存在失败", slog.Any("error", err))
		_ = s.logModelBatchDeleteAudit(ctx, platformID, "failed", fmt.Sprintf("检查平台是否存在失败：%v", err))
		return 0, fmt.Errorf("检查平台是否存在失败：%w", err)
	}
	if !exists {
		err = fmt.Errorf("未找到 ID 为 %d 的平台：%w", platformID, ErrResourceNotFound)
		logger.Warn("平台不存在")
		_ = s.logModelBatchDeleteAudit(ctx, platformID, "failed", err.Error())
		return 0, err
	}

	models, err := s.modelControlRepo.ListModelsByIDs(ctx, modelIDs)
	if err != nil {
		logger.Error("批量查询模型失败", slog.Any("error", err))
		_ = s.logModelBatchDeleteAudit(ctx, platformID, "failed", fmt.Sprintf("批量查询模型失败：%v", err))
		return 0, err
	}

	if len(models) != len(modelIDs) {
		foundIDs := make(map[uint]struct{}, len(models))
		for _, model := range models {
			foundIDs[model.ID] = struct{}{}
		}

		var missingIDs []uint
		for _, id := range modelIDs {
			if _, ok := foundIDs[id]; !ok {
				missingIDs = append(missingIDs, id)
			}
		}

		err = fmt.Errorf("以下模型不存在：%v: %w", missingIDs, ErrResourceNotFound)
		logger.Warn("部分模型不存在",
			slog.Int("requested_count", len(modelIDs)),
			slog.Int("found_count", len(models)),
		)
		_ = s.logModelBatchDeleteAudit(ctx, platformID, "failed", err.Error())
		return 0, err
	}

	backups := make([]modelAPIKeysBackup, 0, len(models))
	for _, model := range models {
		if model.PlatformID != platformID {
			err = fmt.Errorf("模型 ID %d 不属于平台 ID %d: %w", model.ID, platformID, ErrResourceNotBelong)
			logger.Warn("模型不属于指定平台",
				slog.Uint64("model_id", uint64(model.ID)),
				slog.Uint64("model_platform_id", uint64(model.PlatformID)),
				slog.Uint64("expected_platform_id", uint64(platformID)),
			)
			_ = s.logModelBatchDeleteAudit(ctx, platformID, "failed", err.Error())
			return 0, err
		}

		apiKeys, innerErr := s.modelControlRepo.ListAPIKeysByModel(ctx, model.ID)
		if innerErr != nil {
			err = fmt.Errorf("查询模型 ID 为 %d 关联的密钥失败：%w", model.ID, innerErr)
			logger.Error("查询模型关联密钥失败",
				slog.Uint64("model_id", uint64(model.ID)),
				slog.Any("error", innerErr),
			)
			_ = s.logModelBatchDeleteAudit(ctx, platformID, "failed", err.Error())
			return 0, err
		}

		if len(apiKeys) > 0 {
			backups = append(backups, modelAPIKeysBackup{modelID: model.ID, apiKeys: apiKeys})
		}
	}

	logger.Debug("开始清理模型与密钥关联关系", slog.Int("backup_count", len(backups)))
	for _, model := range models {
		if clearErr := s.modelControlRepo.ClearModelAPIKeyRelations(ctx, model.ID); clearErr != nil {
			err = fmt.Errorf("清理模型 ID 为 %d 与密钥的关联关系失败：%w", model.ID, clearErr)
			logger.Error("清理模型与密钥关联关系失败",
				slog.Uint64("model_id", uint64(model.ID)),
				slog.Any("error", clearErr),
			)
			_ = s.logModelBatchDeleteAudit(ctx, platformID, "failed", err.Error())
			return 0, err
		}
	}

	var deletedCount int
	err = s.controlTx.WithinTx(ctx, func(txCtx context.Context) error {
		rowsAffected, innerErr := s.modelControlRepo.DeleteModelsByIDs(txCtx, modelIDs)
		if innerErr != nil {
			return fmt.Errorf("批量删除模型失败：%w", innerErr)
		}
		deletedCount = int(rowsAffected)
		return nil
	})
	if err != nil {
		logger.Warn("批量删除模型事务失败，开始恢复关联关系", slog.Any("error", err))
		recoveryCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		for _, backup := range backups {
			if restoreErr := s.modelControlRepo.AppendModelAPIKeys(recoveryCtx, backup.modelID, backup.apiKeys); restoreErr != nil {
				logger.Error("恢复模型与密钥关联关系失败",
					slog.Uint64("model_id", uint64(backup.modelID)),
					slog.Any("error", restoreErr),
				)
			}
		}
		_ = s.logModelBatchDeleteAudit(ctx, platformID, "failed", fmt.Sprintf("批量删除模型失败：%v", err))
		return 0, fmt.Errorf("批量删除模型失败：%w", err)
	}

	logger.Info("成功批量删除模型", slog.Int("deleted_count", deletedCount))
	_ = s.logModelBatchDeleteAudit(ctx, platformID, "success", fmt.Sprintf("批量删除模型成功，删除数量 %d", deletedCount))
	return deletedCount, nil
}

func (s *service) logModelBatchDeleteAudit(ctx context.Context, platformID uint, result, detail string) error {
	if s.controlAudit == nil {
		return nil
	}

	return s.controlAudit.Log(ctx, ControlAuditEvent{
		Action:     "model.batch_delete",
		Resource:   "platform",
		ResourceID: platformID,
		Result:     result,
		Detail:     detail,
	})
}
