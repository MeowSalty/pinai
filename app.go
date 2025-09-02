package main

import (
	"flag"
	"log/slog"
	"os"

	"github.com/MeowSalty/pinai/database"
	"github.com/MeowSalty/pinai/router"
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

	slog.SetDefault(appLogger)

	// 解析命令行参数
	flag.Parse()

	// 连接数据库
	db, err := database.Connect(*dbType, *dbHost, *dbPort, *dbUser, *dbPass, *dbName)
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
	fiberApp.Use(slogfiber.New(fiberLogger))
	fiberApp.Use(recover.New())

	// 设置路由
	router.SetupRoutes(fiberApp)

	fiberApp.Listen(*port)
}
