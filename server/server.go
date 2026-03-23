package server

import (
	"github.com/MeowSalty/pinai/config"
)

// Run 启动服务器
func Run(cfg *config.Config) {
	runtime := newBootstrapRuntime(cfg)
	runtime.start()
	runtime.waitForShutdownSignal()
	runtime.shutdown()
}
