package main

import (
	"flag"
	"log/slog"
	"os"

	"github.com/MeowSalty/pinai/router"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
	slogfiber "github.com/samber/slog-fiber"
)

var (
	port = flag.String("port", ":3000", "监听端口")
	prod = flag.Bool("prod", false, "在生产环境中启用 prefork")
)

func main() {
	// 创建日志记录器
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	// 创建日志组
	appLogger := logger.WithGroup("app")
	fiberLogger := logger.WithGroup("fiber")

	slog.SetDefault(appLogger)

	// 解析命令行参数
	flag.Parse()

	// // 创建 fiber 应用
	fiberApp := fiber.New(fiber.Config{
		Prefork: *prod, // go run app.go -prod
	})

	// 中间件
	fiberApp.Use(slogfiber.New(fiberLogger))
	fiberApp.Use(recover.New())

	// 设置路由
	router.SetupRoutes(fiberApp)

	fiberApp.Listen(*port)
}
