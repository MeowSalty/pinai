package provider

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/MeowSalty/pinai/database/types"
)

// addKeyToPlatformApp 以应用服务方式实现单个密钥新增。
// 事务边界在应用服务层声明，仓储仅负责数据访问。
func (s *service) addKeyToPlatformApp(ctx context.Context, platformID uint, key types.APIKey) (*types.APIKey, error) {
	logger := s.logger.With(
		slog.String("operation", "key_create"),
		slog.Uint64("platform_id", uint64(platformID)),
	)
	logger.Debug("开始为平台添加 API 密钥")

	if s.keyControlRepo == nil {
		return nil, fmt.Errorf("创建 API 密钥失败：密钥控制仓储未初始化")
	}
	if s.controlTx == nil {
		return nil, fmt.Errorf("创建 API 密钥失败：事务执行器未初始化")
	}

	exists, err := s.keyControlRepo.ExistsPlatform(ctx, platformID)
	if err != nil {
		logger.Error("检查平台是否存在失败", slog.Any("error", err))
		_ = s.logKeyCreateAudit(ctx, platformID, 0, "failed", fmt.Sprintf("检查平台是否存在失败：%v", err))
		return nil, fmt.Errorf("检查平台是否存在失败：%w", err)
	}
	if !exists {
		logger.Warn("平台不存在")
		err = fmt.Errorf("未找到 ID 为 %d 的平台：%w", platformID, ErrResourceNotFound)
		_ = s.logKeyCreateAudit(ctx, platformID, 0, "failed", err.Error())
		return nil, err
	}

	key.ID = 0
	key.PlatformID = platformID

	err = s.controlTx.WithinTx(ctx, func(txCtx context.Context) error {
		if innerErr := s.keyControlRepo.CreateAPIKey(txCtx, &key); innerErr != nil {
			return innerErr
		}
		return nil
	})
	if err != nil {
		logger.Error("创建 API 密钥失败", slog.Any("error", err))
		_ = s.logKeyCreateAudit(ctx, platformID, 0, "failed", fmt.Sprintf("创建 API 密钥失败：%v", err))
		return nil, fmt.Errorf("创建 API 密钥失败：%w", err)
	}

	logger.Info("成功为平台添加 API 密钥", slog.Uint64("key_id", uint64(key.ID)))
	_ = s.logKeyCreateAudit(ctx, platformID, key.ID, "success", fmt.Sprintf("创建 API 密钥成功，platform_id=%d", platformID))
	return &key, nil
}

func (s *service) logKeyCreateAudit(ctx context.Context, platformID, keyID uint, result, detail string) error {
	if s.controlAudit == nil {
		return nil
	}

	return s.controlAudit.Log(ctx, ControlAuditEvent{
		Action:     "key.create",
		Resource:   "key",
		ResourceID: keyID,
		Result:     result,
		Detail:     detail,
	})
}
