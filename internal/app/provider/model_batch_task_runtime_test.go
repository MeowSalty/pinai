package provider

import (
	"testing"

	"github.com/MeowSalty/pinai/database/types"
)

func TestModelBatchTaskRuntime_CacheClone(t *testing.T) {
	s := &service{
		taskStateCache: make(map[uint]*ModelBatchTaskSummary),
		taskEnqueued:   make(map[uint]struct{}),
		taskQueue:      make(chan uint, 1),
	}

	summary := &ModelBatchTaskSummary{ID: 1, Status: types.ModelBatchTaskStatusPending}
	s.cacheTaskSummary(summary)

	got, ok := s.getCachedTaskSummary(1)
	if !ok {
		t.Fatalf("应命中缓存")
	}
	got.Status = types.ModelBatchTaskStatusFailed

	again, ok := s.getCachedTaskSummary(1)
	if !ok {
		t.Fatalf("应命中缓存")
	}
	if again.Status != types.ModelBatchTaskStatusPending {
		t.Fatalf("缓存应返回副本，避免外部修改: got=%s", again.Status)
	}
}

func TestModelBatchTaskRuntime_EnqueueDeduplicate(t *testing.T) {
	s := &service{
		taskStateCache: make(map[uint]*ModelBatchTaskSummary),
		taskEnqueued:   make(map[uint]struct{}),
		taskQueue:      make(chan uint, 2),
	}

	if err := s.enqueueTaskInMemory(10); err != nil {
		t.Fatalf("首次入队失败: %v", err)
	}
	if err := s.enqueueTaskInMemory(10); err != nil {
		t.Fatalf("重复入队不应报错: %v", err)
	}

	if len(s.taskQueue) != 1 {
		t.Fatalf("重复入队应被去重: got=%d, want=1", len(s.taskQueue))
	}
}
