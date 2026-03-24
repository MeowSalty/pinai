package provider

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/MeowSalty/pinai/database/types"
)

// batchAddModelsToPlatformApp 以应用服务方式实现批量创建模型。
// 事务边界在应用服务层声明，仓储仅负责数据访问。
func (s *service) batchAddModelsToPlatformApp(ctx context.Context, platformID uint, models []types.Model) ([]*types.Model, error) {
	logger := s.logger.With(
		slog.String("operation", "batch_add_models"),
		slog.Uint64("platform_id", uint64(platformID)),
		slog.Int("model_count", len(models)),
	)
	logger.Debug("开始批量为平台添加模型")

	if s.modelControlRepo == nil {
		return nil, fmt.Errorf("批量创建模型失败：模型控制仓储未初始化")
	}
	if s.controlTx == nil {
		return nil, fmt.Errorf("批量创建模型失败：事务执行器未初始化")
	}

	if len(models) == 0 {
		logger.Warn("未提供任何模型")
		return nil, fmt.Errorf("必须至少提供一个模型")
	}

	exists, err := s.modelControlRepo.ExistsPlatform(ctx, platformID)
	if err != nil {
		logger.Error("检查平台是否存在失败", slog.Any("error", err))
		return nil, fmt.Errorf("检查平台是否存在失败：%w", err)
	}
	if !exists {
		logger.Warn("平台不存在")
		return nil, fmt.Errorf("未找到 ID 为 %d 的平台：%w", platformID, ErrResourceNotFound)
	}

	apiKeyIDSet := make(map[uint]struct{})
	for _, model := range models {
		if len(model.APIKeys) == 0 {
			return nil, fmt.Errorf("模型 '%s' 必须至少关联一个 API 密钥：%w", model.Name, ErrInvalidArgument)
		}
		for _, key := range model.APIKeys {
			apiKeyIDSet[key.ID] = struct{}{}
		}
	}

	apiKeyIDs := make([]uint, 0, len(apiKeyIDSet))
	for id := range apiKeyIDSet {
		apiKeyIDs = append(apiKeyIDs, id)
	}

	validKeys, err := s.modelControlRepo.ListAPIKeysByPlatformAndIDs(ctx, platformID, apiKeyIDs)
	if err != nil {
		logger.Error("查询平台密钥失败", slog.Any("error", err))
		return nil, fmt.Errorf("校验模型关联密钥失败：%w", err)
	}
	if len(validKeys) != len(apiKeyIDs) {
		logger.Warn("部分 API 密钥不存在或不属于平台")
		return nil, fmt.Errorf("部分 API 密钥不存在或不属于平台 ID %d：%w", platformID, ErrResourceNotBelong)
	}

	validKeyMap := make(map[uint]*types.APIKey, len(validKeys))
	for _, key := range validKeys {
		validKeyMap[key.ID] = key
	}

	createdModels := make([]*types.Model, 0, len(models))
	err = s.controlTx.WithinTx(ctx, func(txCtx context.Context) error {
		for i := range models {
			model := models[i]
			model.ID = 0
			model.PlatformID = platformID

			mappedAPIKeys := make([]types.APIKey, 0, len(model.APIKeys))
			for _, key := range model.APIKeys {
				validKey, ok := validKeyMap[key.ID]
				if !ok {
					return fmt.Errorf("模型 '%s' 关联了无效 API 密钥 ID %d：%w", model.Name, key.ID, ErrResourceNotBelong)
				}
				mappedAPIKeys = append(mappedAPIKeys, *validKey)
			}
			model.APIKeys = mappedAPIKeys

			if innerErr := s.modelControlRepo.CreateModel(txCtx, &model); innerErr != nil {
				return fmt.Errorf("创建模型 '%s' 失败：%w", model.Name, innerErr)
			}

			createdModels = append(createdModels, &model)
		}

		return nil
	})
	if err != nil {
		logger.Error("批量创建模型事务失败", slog.Any("error", err))
		_ = s.logModelBatchCreateAudit(ctx, platformID, "failed", fmt.Sprintf("批量创建模型失败：%v", err))
		return nil, fmt.Errorf("批量创建模型失败：%w", err)
	}

	logger.Info("成功批量为平台添加模型", slog.Int("created_count", len(createdModels)))
	_ = s.logModelBatchCreateAudit(ctx, platformID, "success", fmt.Sprintf("批量创建模型成功，创建数量 %d", len(createdModels)))
	return createdModels, nil
}

func (s *service) logModelBatchCreateAudit(ctx context.Context, platformID uint, result, detail string) error {
	if s.controlAudit == nil {
		return nil
	}

	return s.controlAudit.Log(ctx, ControlAuditEvent{
		Action:     "model.batch_create",
		Resource:   "platform",
		ResourceID: platformID,
		Result:     result,
		Detail:     detail,
	})
}
