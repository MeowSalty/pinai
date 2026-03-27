package portal

import (
	runtimepkg "github.com/MeowSalty/pinai/internal/infra/portal/runtime"
)

// portalRuntime 兼容既有 service 结构字段命名，实际能力定义由 runtime 子模块提供。
type portalRuntime = runtimepkg.Runtime
