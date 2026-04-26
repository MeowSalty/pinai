package provider

import (
	"context"
	"log/slog"
	"testing"

	"github.com/MeowSalty/pinai/database/query"
	"github.com/MeowSalty/pinai/database/types"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newModelBatchTaskRepoTestContext(t *testing.T) (context.Context, *gorm.DB) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}

	if err := db.AutoMigrate(&types.ModelBatchTask{}); err != nil {
		t.Fatalf("迁移模型批量任务表失败: %v", err)
	}

	q := query.Use(db)
	ctx := context.WithValue(context.Background(), controlTxQueryKey{}, q)
	return ctx, db
}

func TestModelBatchTaskRepository_CreateAndGetTask(t *testing.T) {
	ctx, _ := newModelBatchTaskRepoTestContext(t)
	repo := NewModelBatchTaskGormRepository(slog.Default())

	task := &types.ModelBatchTask{
		Type:       types.ModelBatchTaskTypeDelete,
		Status:     types.ModelBatchTaskStatusPending,
		PlatformID: 7,
		Payload:    `{"platform_id":7,"model_ids":[1,2]}`,
	}

	if err := repo.CreateModelBatchTask(ctx, task); err != nil {
		t.Fatalf("创建模型批量任务失败: %v", err)
	}
	if task.ID == 0 {
		t.Fatalf("创建后任务 ID 不应为 0")
	}

	got, err := repo.GetModelBatchTaskByID(ctx, task.ID)
	if err != nil {
		t.Fatalf("按 ID 查询模型批量任务失败: %v", err)
	}

	if got.Type != types.ModelBatchTaskTypeDelete {
		t.Fatalf("任务类型不匹配: got=%s", got.Type)
	}
	if got.Status != types.ModelBatchTaskStatusPending {
		t.Fatalf("任务状态不匹配: got=%s", got.Status)
	}
}

func TestModelBatchTaskRepository_ClaimNextPendingTask(t *testing.T) {
	ctx, db := newModelBatchTaskRepoTestContext(t)
	repo := NewModelBatchTaskGormRepository(slog.Default())

	seed := &types.ModelBatchTask{
		Type:       types.ModelBatchTaskTypeUpdate,
		Status:     types.ModelBatchTaskStatusPending,
		PlatformID: 11,
		Payload:    `{"platform_id":11,"models":[]}`,
	}
	if err := db.WithContext(ctx).Create(seed).Error; err != nil {
		t.Fatalf("准备测试任务数据失败: %v", err)
	}

	claimed, err := repo.ClaimNextPendingModelBatchTask(ctx)
	if err != nil {
		t.Fatalf("抢占待执行任务失败: %v", err)
	}
	if claimed == nil {
		t.Fatalf("应抢占到一个待执行任务")
	}
	if claimed.ID != seed.ID {
		t.Fatalf("抢占到的任务 ID 不匹配: got=%d, want=%d", claimed.ID, seed.ID)
	}
	if claimed.Status != types.ModelBatchTaskStatusRunning {
		t.Fatalf("抢占后任务状态应为 running: got=%s", claimed.Status)
	}
	if claimed.StartedAt == nil {
		t.Fatalf("抢占后 StartedAt 不应为 nil")
	}

	var stored types.ModelBatchTask
	if err := db.WithContext(ctx).Where("id = ?", seed.ID).First(&stored).Error; err != nil {
		t.Fatalf("查询抢占后的任务记录失败: %v", err)
	}
	if stored.Status != types.ModelBatchTaskStatusRunning {
		t.Fatalf("数据库中的任务状态应为 running: got=%s", stored.Status)
	}
}

func TestModelBatchTaskRepository_FinishTask(t *testing.T) {
	ctx, db := newModelBatchTaskRepoTestContext(t)
	repo := NewModelBatchTaskGormRepository(slog.Default())

	seed := &types.ModelBatchTask{
		Type:       types.ModelBatchTaskTypeAdd,
		Status:     types.ModelBatchTaskStatusRunning,
		PlatformID: 23,
		Payload:    `{"platform_id":23,"models":[]}`,
	}
	if err := db.WithContext(ctx).Create(seed).Error; err != nil {
		t.Fatalf("准备测试任务数据失败: %v", err)
	}

	resultJSON := `{"success_count":1,"failed_count":0}`
	if err := repo.FinishModelBatchTask(ctx, seed.ID, types.ModelBatchTaskStatusSucceeded, resultJSON, ""); err != nil {
		t.Fatalf("完成模型批量任务失败: %v", err)
	}

	var stored types.ModelBatchTask
	if err := db.WithContext(ctx).Where("id = ?", seed.ID).First(&stored).Error; err != nil {
		t.Fatalf("查询完成后的任务记录失败: %v", err)
	}
	if stored.Status != types.ModelBatchTaskStatusSucceeded {
		t.Fatalf("任务状态应为 succeeded: got=%s", stored.Status)
	}
	if stored.Result != resultJSON {
		t.Fatalf("任务结果不匹配: got=%s", stored.Result)
	}
	if stored.FinishedAt == nil {
		t.Fatalf("完成后 FinishedAt 不应为 nil")
	}
}
