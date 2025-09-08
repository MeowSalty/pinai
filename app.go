package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/MeowSalty/pinai/database"
	"github.com/MeowSalty/pinai/router"
	"github.com/MeowSalty/pinai/services"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
	slogfiber "github.com/samber/slog-fiber"
)

var (
	port = flag.String("port", ":3000", "监听端口")
	prod = flag.Bool("prod", false, "在生产环境中启用 prefork")

	// 数据库相关参数
	dbType = flag.String("db-type", "sqlite", "数据库类型 (sqlite, mysql, postgres)")
	dbHost = flag.String("db-host", "", "数据库主机地址")
	dbPort = flag.String("db-port", "", "数据库端口")
	dbUser = flag.String("db-user", "", "数据库用户名")
	dbPass = flag.String("db-pass", "", "数据库密码")
	dbName = flag.String("db-name", "", "数据库名称")
)

func main() {
	// 创建日志记录器
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	// 创建日志组
	appLogger := logger.WithGroup("app")
	fiberLogger := logger.WithGroup("fiber")
	gormLogger := logger.WithGroup("gorm")

	slog.SetDefault(appLogger)

	// 解析命令行参数
	flag.Parse()

	// 连接数据库
	db, err := database.Connect(*dbType, *dbHost, *dbPort, *dbUser, *dbPass, *dbName, gormLogger)
	if err != nil {
		appLogger.Error("数据库连接失败", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	// 创建 fiber 应用
	fiberApp := fiber.New(fiber.Config{
		Prefork: *prod, // go run app.go -prod
	})

	// 中间件
	fiberApp.Use(slogfiber.NewWithConfig(fiberLogger, slogfiber.Config{
		Filters: []slogfiber.Filter{
			// 忽略 /openai 路径下的请求，避免干扰流式传输
			slogfiber.IgnoreHostContains("/openai"),
		},
	}))
	fiberApp.Use(recover.New())

	// 初始化服务
	appContext := context.Background()
	svcs, err := services.NewServices(appContext, appLogger.WithGroup("services"))
	if err != nil {
		appLogger.Error("服务初始化失败", "error", err)
		os.Exit(1)
	}

	// 设置路由
	if err := router.SetupRoutes(fiberApp, svcs); err != nil {
		appLogger.Error("路由设置失败", "error", err)
		os.Exit(1)
	}

	// 启动 Web 服务
	go func() {
		// 监听端口 3000
		// go run app.go -port=:3000
		if err := fiberApp.Listen(*port); err != nil {
			fiberLogger.Error("无法启动 Web 服务", "error", err)
			os.Exit(1) // 如果无法启动 Web 服务，退出应用
		}
	}()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	_ = <-c
	appLogger.Info("收到关闭信号，正在关闭应用...")

	// 关闭 AI 网关服务
	if svcs.AIGatewayService != nil {
		appLogger.Info("正在关闭 AI 网关服务")
		if err := svcs.AIGatewayService.Close(5 * time.Second); err != nil {
			appLogger.Error("关闭 AI 网关服务失败", "error", err)
		} else {
			appLogger.Info("AI 网关服务已成功关闭")
		}
	}

	// 关闭 Web 服务
	err = fiberApp.Shutdown()
	if err != nil {
		fiberLogger.Error("关闭 Web 服务失败", "error", err)
	} else {
		fiberLogger.Info("Web 服务已成功关闭")
	}

	// 关闭数据库连接
	err = db.Close()
	if err != nil {
		appLogger.Error("关闭数据库连接失败", "error", err)
	} else {
		appLogger.Info("数据库连接已成功关闭")
	}
	appLogger.Info("应用已成功关闭")
}
