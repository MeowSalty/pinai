package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"runtime/debug"
	"strings"
	"syscall"
	"time"

	"github.com/MeowSalty/pinai/database"
	"github.com/MeowSalty/pinai/frontend"
	"github.com/MeowSalty/pinai/router"
	"github.com/MeowSalty/pinai/services"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
	slogfiber "github.com/samber/slog-fiber"
)

var (
	port *string
	prod *bool

	enableWeb            *bool
	webDir               *string
	enableFrontendUpdate *bool

	dbType      *string
	dbHost      *string
	dbPort      *string
	dbUser      *string
	dbPass      *string
	dbName      *string
	dbSSLMode   *string
	dbTLSConfig *string

	apiToken   *string
	adminToken *string

	githubProxy *string

	modelMapping *string
)

func loadFlag() {
	port = flag.String("port", envPort, "监听端口")
	prod = flag.Bool("prod", envProd, "在生产环境中启用 prefork")

	// 前端相关参数
	enableWeb = flag.Bool("enable-web", envEnableWeb, "启用前端支持")
	webDir = flag.String("web-dir", envWebDir, "前端文件目录")
	enableFrontendUpdate = flag.Bool("enable-frontend-update", envEnableFrontendUpdate, "启用前端更新检查")

	// 数据库相关参数
	dbType = flag.String("db-type", envDBType, "数据库类型 (sqlite, mysql, postgres)")
	dbHost = flag.String("db-host", envDBHost, "数据库主机地址")
	dbPort = flag.String("db-port", envDBPort, "数据库端口")
	dbUser = flag.String("db-user", envDBUser, "数据库用户名")
	dbPass = flag.String("db-pass", envDBPass, "数据库密码")
	dbName = flag.String("db-name", envDBName, "数据库名称")
	dbSSLMode = flag.String("db-ssl-mode", envDBSSLMode, "PostgreSQL SSL 模式 (disable, require, verify-ca, verify-full)")
	dbTLSConfig = flag.String("db-tls-config", envDBTLSConfig, "MySQL TLS 配置 (true, false, skip-verify, preferred)")

	// API Token 参数
	apiToken = flag.String("api-token", envAPIToken, "API Token，如果为空则不启用身份验证")
	adminToken = flag.String("admin-token", envAdminToken, "管理 API Token，如果为空则使用 API Token")

	// GitHub 代理参数
	githubProxy = flag.String("github-proxy", envGitHubProxy, "GitHub 代理地址，用于加速 GitHub 访问")

	// 模型映射规则参数
	modelMapping = flag.String("model-mapping", envModelMapping, "模型映射规则，格式：key1:value1,key2:value2")

	flag.Parse()
}

func loadConfig() {
	loadEnv()
	loadFlag()
}

func main() {
	// 创建日志记录器
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	// 创建日志组
	appLogger := logger.WithGroup("app")
	fiberLogger := logger.WithGroup("fiber")
	gormLogger := logger.WithGroup("gorm")
	frontendLogger := logger.WithGroup("frontend")

	slog.SetDefault(appLogger)

	// 加载配置
	loadConfig()

	// 如果启用了前端支持，则初始化前端
	if *enableWeb {
		if err := frontend.InitializeWeb(frontendLogger, webDir, *enableFrontendUpdate, *githubProxy); err != nil {
			appLogger.Error("初始化前端失败，本次运行将禁用前端支持", "error", err)
			*enableWeb = false
		}
	}

	// 连接数据库
	db, err := database.Connect(*dbType, *dbHost, *dbPort, *dbUser, *dbPass, *dbName, *dbSSLMode, *dbTLSConfig, gormLogger)
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
	fiberApp.Use(recover.New(recover.Config{
		EnableStackTrace: true, // 启用堆栈跟踪
		StackTraceHandler: func(c *fiber.Ctx, e any) {
			stack := debug.Stack()
			// 将堆栈信息按行分割，以数组形式记录，提高 JSON 日志可读性
			stackLines := strings.Split(strings.TrimSpace(string(stack)), "\n")
			fiberLogger.Error("发生 panic",
				"panic", e,
				"path", c.Path(),
				"method", c.Method(),
				"body", string(c.Body()),
				"stack", stackLines,
			)
		},
	}))
	fiberApp.Use(slogfiber.NewWithConfig(fiberLogger, slogfiber.Config{
		Filters: []slogfiber.Filter{
			// 忽略 /completions 路径下的请求，避免干扰流式传输
			slogfiber.IgnorePathContains("/completions"),
			slogfiber.IgnorePathContains("/messages"),
		},
	}))

	// 初始化服务
	appContext := context.Background()
	svcs, err := services.NewServices(appContext, appLogger.WithGroup("services"), *modelMapping)
	if err != nil {
		appLogger.Error("服务初始化失败", "error", err)
		os.Exit(1)
	}

	// 如果没有设置管理令牌，则使用 API Token，并输出警告
	effectiveAdminToken := *adminToken
	if effectiveAdminToken == "" {
		effectiveAdminToken = *apiToken
		if *apiToken != "" {
			appLogger.Warn("未设置独立的管理 API Token，管理接口将与业务接口使用相同的令牌")
		}
	}
	if *apiToken == "" {
		appLogger.Warn("未启用 API Token，将不进行身份验证")
	}

	// 设置路由
	if err := router.SetupRoutes(fiberApp, svcs, *enableWeb, *webDir, *apiToken, effectiveAdminToken); err != nil {
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
	signal.Notify(c, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

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
