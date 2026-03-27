package portal

import (
	healthadapterpkg "github.com/MeowSalty/pinai/internal/infra/portal/healthadapter"
)

// healthStorageAdapter 兼容既有装配命名，实际实现已下沉到 healthadapter 子模块。
type healthStorageAdapter = healthadapterpkg.Adapter

// newHealthStorageAdapter 兼容既有构造入口，内部委托 healthadapter 子模块。
func newHealthStorageAdapter(healthStorage HealthStorage) *healthStorageAdapter {
	return healthadapterpkg.New(healthStorage)
}
