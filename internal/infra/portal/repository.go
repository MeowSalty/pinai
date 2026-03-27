package portal

import (
	"log/slog"

	repositorypkg "github.com/MeowSalty/pinai/internal/infra/portal/repository"
)

// portalAdapterRepository 兼容既有装配命名，实际实现已下沉到 repository 子模块。
type portalAdapterRepository = repositorypkg.Repository

// newPortalAdapterRepository 兼容既有构造入口，内部委托 repository 子模块。
func newPortalAdapterRepository(logger *slog.Logger) *portalAdapterRepository {
	return repositorypkg.New(logger)
}
