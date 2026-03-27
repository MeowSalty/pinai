package portal

import (
	"github.com/MeowSalty/pinai/internal/app/gateway"
)

// Service Portal 服务接口
//
// 显式承接 gateway 应用层定义的 ports，作为 infra 层实现边界。
type Service interface {
	gateway.GatewayPort
}
