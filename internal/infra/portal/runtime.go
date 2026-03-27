package portal

import (
	runtimepkg "github.com/MeowSalty/pinai/internal/infra/portal/runtime"
)

// gatewayRuntime 表示 facade 依赖的最小运行时能力。
type gatewayRuntime = runtimepkg.Runtime

// portalRuntime 兼容旧命名，逐步过渡到 gatewayRuntime。
type portalRuntime = gatewayRuntime
