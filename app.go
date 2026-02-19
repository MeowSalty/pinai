package main

import (
	"github.com/MeowSalty/pinai/config"
	"github.com/MeowSalty/pinai/server"
)

func main() {
	// 加载配置
	cfg := config.LoadConfig()

	// 启动服务器
	server.Run(cfg)
}
