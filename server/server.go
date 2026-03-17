package server

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"runtime/debug"
	"strings"
	"syscall"
	"time"

	"github.com/MeowSalty/pinai/config"
	"github.com/MeowSalty/pinai/database"
	"github.com/MeowSalty/pinai/frontend"
	"github.com/MeowSalty/pinai/logger"
	"github.com/MeowSalty/pinai/router"
	"github.com/MeowSalty/pinai/services"
	"github.com/gin-gonic/gin"
	sloggin "github.com/samber/slog-gin"
)

// Run 启动服务器
func Run(cfg *config.Config) {
	// 初始化日志记录器
	appLogger, fileHandler := logger.InitLogger(cfg.LogLevel)
	if fileHandler != nil {
		defer fileHandler.Close()
	}

	// 创建日志组
	ginLogger := appLogger.WithGroup("gin")
	gormLogger := appLogger.WithGroup("gorm")
	frontendLogger := appLogger.WithGroup("frontend")
	routerLogger := appLogger.WithGroup("router")

	slog.SetDefault(appLogger)

	// 如果启用了前端支持，则初始化前端
	if cfg.EnableWeb {
		if err := frontend.InitializeWeb(frontendLogger, &cfg.WebDir, cfg.EnableFrontendUpdate, cfg.GitHubProxy); err != nil {
			appLogger.Error("初始化前端失败，本次运行将禁用前端支持", "error", err)
			cfg.EnableWeb = false
		}
	}

	// 连接数据库
	db, err := database.Connect(cfg.DBType, cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPass, cfg.DBName, cfg.DBSSLMode, cfg.DBTLSConfig, gormLogger)
	if err != nil {
		appLogger.Error("数据库连接失败", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	// 非 Debug 等级时，设置 Gin 为 Release 模式以减少冗余输出
	if strings.ToUpper(cfg.LogLevel) != "DEBUG" {
		gin.SetMode(gin.ReleaseMode)
	}

	// 创建 gin 应用
	ginEngine := gin.New()

	// 中间件
	ginEngine.Use(gin.CustomRecoveryWithWriter(nil, func(c *gin.Context, err any) {
		stack := debug.Stack()
		// 将堆栈信息按行分割，以数组形式记录，提高 JSON 日志可读性
		stackLines := strings.Split(strings.TrimSpace(string(stack)), "\n")
		ginLogger.Error("发生 panic",
			"panic", err,
			"path", c.Request.URL.Path,
			"method", c.Request.Method,
			"stack", stackLines,
		)
		c.AbortWithStatus(http.StatusInternalServerError)
	}))
	slogGinConfig := sloggin.DefaultConfig()
	slogGinConfig.HandleGinDebug = true
	ginEngine.Use(sloggin.NewWithConfig(ginLogger, slogGinConfig))

	// 初始化服务
	appContext := context.Background()
	svcs, err := services.NewServices(appContext, appLogger.WithGroup("services"), cfg.ModelMapping)
	if err != nil {
		appLogger.Error("服务初始化失败", "error", err)
		os.Exit(1)
	}

	// 如果没有设置管理令牌，则使用 API Token，并输出警告
	effectiveAdminToken := cfg.AdminToken
	if effectiveAdminToken == "" {
		effectiveAdminToken = cfg.APIToken
		if cfg.APIToken != "" {
			appLogger.Warn("未设置独立的管理 API Token，管理接口将与业务接口使用相同的令牌")
		}
	}
	if cfg.APIToken == "" {
		appLogger.Warn("未启用 API Token，将不进行身份验证")
	}

	// 设置路由
	routerConfig := router.Config{
		AdminToken:         effectiveAdminToken,
		ApiToken:           cfg.APIToken,
		EnableWeb:          cfg.EnableWeb,
		PassthroughHeaders: cfg.PassthroughHeaders,
		ProxyEnabled:       cfg.ProxyEnabled,
		UserAgent:          cfg.UserAgent,
		WebDir:             cfg.WebDir,
	}
	if cfg.ProxyEnabled && effectiveAdminToken == "" {
		appLogger.Warn("代理功能已启用但未设置管理令牌，代理端点将不可用")
	}
	if err := router.SetupRoutes(ginEngine, svcs, routerConfig, routerLogger); err != nil {
		appLogger.Error("路由设置失败", "error", err)
		os.Exit(1)
	}

	// 启动 Web 服务
	srv := &http.Server{
		Addr:    cfg.Port,
		Handler: ginEngine,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			ginLogger.Error("无法启动 Web 服务", "error", err)
			os.Exit(1)
		}
	}()

	// 等待关闭信号
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	_ = <-c
	appLogger.Info("收到关闭信号，正在关闭应用...")

	// 关闭 Portal 服务
	if svcs.PortalService != nil {
		appLogger.Info("正在关闭 Portal 服务")
		if err := svcs.PortalService.Close(5 * time.Second); err != nil {
			appLogger.Error("关闭 Portal 服务失败", "error", err)
		} else {
			appLogger.Info("Portal 服务已成功关闭")
		}
	}

	// 关闭 Web 服务
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		ginLogger.Error("关闭 Web 服务失败", "error", err)
	} else {
		ginLogger.Info("Web 服务已成功关闭")
	}

	// 关闭数据库连接
	if err := db.Close(); err != nil {
		appLogger.Error("关闭数据库连接失败", "error", err)
	} else {
		appLogger.Info("数据库连接已成功关闭")
	}
	appLogger.Info("应用已成功关闭")
}
