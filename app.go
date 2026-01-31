package main

import (
	"bytes"
	"context"
	"flag"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"runtime/debug"
	"strings"
	"sync"
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

// plainTextHandler 实现普通文本格式的日志处理器
type plainTextHandler struct {
	opts slog.HandlerOptions
	mu   sync.Mutex
	out  io.Writer
}

// newPlainTextHandler 创建普通文本格式的日志处理器
func newPlainTextHandler(out io.Writer, opts *slog.HandlerOptions) *plainTextHandler {
	if opts == nil {
		opts = &slog.HandlerOptions{}
	}
	return &plainTextHandler{
		opts: *opts,
		out:  out,
	}
}

// Enabled 检查日志级别是否启用
func (h *plainTextHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return level >= h.opts.Level.Level()
}

// Handle 处理日志记录
func (h *plainTextHandler) Handle(ctx context.Context, r slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	var buf bytes.Buffer

	// 格式：[时间] [级别] [组] 消息
	buf.WriteString(r.Time.Format("2006/01/02 15:04:05.000"))

	// 日志级别
	levelStr := "INFO"
	switch r.Level {
	case slog.LevelDebug:
		levelStr = "DEBUG"
	case slog.LevelInfo:
		levelStr = "INFO"
	case slog.LevelWarn:
		levelStr = "WARN"
	case slog.LevelError:
		levelStr = "ERROR"
	}
	buf.WriteString(" ")
	buf.WriteString(levelStr)
	buf.WriteString(" ")

	// 消息
	buf.WriteString(r.Message)

	// 属性
	r.Attrs(func(a slog.Attr) bool {
		buf.WriteString(" ")
		buf.WriteString(a.Key)
		buf.WriteString("=")
		buf.WriteString(a.Value.String())
		return true
	})

	buf.WriteString("\n")

	_, err := h.out.Write(buf.Bytes())
	return err
}

// WithAttrs 返回带有额外属性的处理器
func (h *plainTextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &plainTextHandler{
		opts: h.opts,
		out:  h.out,
	}
}

// WithGroup 返回带有组的处理器
func (h *plainTextHandler) WithGroup(name string) slog.Handler {
	return &plainTextHandler{
		opts: h.opts,
		out:  h.out,
	}
}

// multiHandler 实现多输出日志处理器
type multiHandler struct {
	handlers []slog.Handler
}

// newMultiHandler 创建多输出日志处理器
func newMultiHandler(handlers ...slog.Handler) slog.Handler {
	return &multiHandler{handlers: handlers}
}

// Enabled 检查日志级别是否启用
func (h *multiHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, handler := range h.handlers {
		if handler.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

// Handle 处理日志记录
func (h *multiHandler) Handle(ctx context.Context, r slog.Record) error {
	for _, handler := range h.handlers {
		if handler.Enabled(ctx, r.Level) {
			// 复制记录以避免修改原始记录
			r := r
			if err := handler.Handle(ctx, r); err != nil {
				return err
			}
		}
	}
	return nil
}

// WithAttrs 返回带有额外属性的处理器
func (h *multiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	handlers := make([]slog.Handler, len(h.handlers))
	for i, handler := range h.handlers {
		handlers[i] = handler.WithAttrs(attrs)
	}
	return &multiHandler{handlers: handlers}
}

// WithGroup 返回带有组的处理器
func (h *multiHandler) WithGroup(name string) slog.Handler {
	handlers := make([]slog.Handler, len(h.handlers))
	for i, handler := range h.handlers {
		handlers[i] = handler.WithGroup(name)
	}
	return &multiHandler{handlers: handlers}
}

// dailyRotateHandler 实现按日期分割的日志文件处理器
type dailyRotateHandler struct {
	logDir      string
	baseName    string
	level       slog.Level
	mu          sync.Mutex
	currentDate string
	file        *os.File
	handler     slog.Handler
}

// newDailyRotateHandler 创建按日期分割的日志文件处理器
func newDailyRotateHandler(logDir, baseName string, level slog.Level) (*dailyRotateHandler, error) {
	h := &dailyRotateHandler{
		logDir:   logDir,
		baseName: baseName,
		level:    level,
	}
	if err := h.rotate(); err != nil {
		return nil, err
	}
	return h, nil
}

// rotate 检查并轮转日志文件
func (h *dailyRotateHandler) rotate() error {
	currentDate := time.Now().Format("2006-01-02")

	h.mu.Lock()
	defer h.mu.Unlock()

	// 如果日期未变化，无需轮转
	if h.currentDate == currentDate && h.file != nil {
		return nil
	}

	// 关闭旧文件
	if h.file != nil {
		h.file.Close()
	}

	// 创建新文件
	fileName := h.baseName + "-" + currentDate + ".log"
	filePath := h.logDir + "/" + fileName
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}

	h.currentDate = currentDate
	h.file = file
	h.handler = slog.NewJSONHandler(file, &slog.HandlerOptions{
		Level: h.level,
	})

	return nil
}

// Enabled 检查日志级别是否启用
func (h *dailyRotateHandler) Enabled(ctx context.Context, level slog.Level) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.handler == nil {
		return false
	}
	return h.handler.Enabled(ctx, level)
}

// Handle 处理日志记录
func (h *dailyRotateHandler) Handle(ctx context.Context, r slog.Record) error {
	// 检查是否需要轮转
	if err := h.rotate(); err != nil {
		return err
	}

	h.mu.Lock()
	defer h.mu.Unlock()
	if h.handler == nil {
		return nil
	}
	return h.handler.Handle(ctx, r)
}

// WithAttrs 返回带有额外属性的处理器
func (h *dailyRotateHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.handler == nil {
		return h
	}
	return &dailyRotateHandler{
		logDir:      h.logDir,
		baseName:    h.baseName,
		level:       h.level,
		currentDate: h.currentDate,
		file:        h.file,
		handler:     h.handler.WithAttrs(attrs),
	}
}

// WithGroup 返回带有组的处理器
func (h *dailyRotateHandler) WithGroup(name string) slog.Handler {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.handler == nil {
		return h
	}
	return &dailyRotateHandler{
		logDir:      h.logDir,
		baseName:    h.baseName,
		level:       h.level,
		currentDate: h.currentDate,
		file:        h.file,
		handler:     h.handler.WithGroup(name),
	}
}

// Close 关闭日志文件
func (h *dailyRotateHandler) Close() error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.file != nil {
		return h.file.Close()
	}
	return nil
}

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

	logLevel *string

	userAgent *string
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

	// 日志等级参数
	logLevel = flag.String("log-level", envLogLevel, "日志输出等级 (DEBUG, INFO, WARN, ERROR)")

	// User-Agent 参数
	userAgent = flag.String("user-agent", envUserAgent, "User-Agent 配置，空则透传客户端 UA，\"default\" 使用 fasthttp 默认值，其他字符串则复写")

	flag.Parse()
}

func loadConfig() {
	loadEnv()
	loadFlag()
}

func main() {
	// 加载配置
	loadConfig()

	// 解析日志等级
	var level slog.Level
	switch strings.ToUpper(*logLevel) {
	case "DEBUG":
		level = slog.LevelDebug
	case "INFO":
		level = slog.LevelInfo
	case "WARN":
		level = slog.LevelWarn
	case "ERROR":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	// 创建 log 目录
	if err := os.MkdirAll("log", 0755); err != nil {
		// 创建临时日志记录器用于输出错误
		tempLogger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: level,
		}))
		tempLogger.Error("创建日志目录失败", "error", err)
	}

	// 创建终端处理器（普通文本格式）
	consoleHandler := newPlainTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	})

	// 创建按日期分割的日志文件处理器（JSON 格式）
	fileHandler, err := newDailyRotateHandler("log", "pinai", level)
	if err != nil {
		// 如果无法创建日志文件，仅使用终端输出
		tempLogger := slog.New(consoleHandler)
		tempLogger.Error("无法创建日志文件，将仅输出到终端", "error", err)
		fileHandler = nil
	}

	// 创建多输出日志记录器
	var logger *slog.Logger
	if fileHandler != nil {
		logger = slog.New(newMultiHandler(consoleHandler, fileHandler))
		defer fileHandler.Close()
	} else {
		logger = slog.New(consoleHandler)
	}

	// 创建日志组
	appLogger := logger.WithGroup("app")
	fiberLogger := logger.WithGroup("fiber")
	gormLogger := logger.WithGroup("gorm")
	frontendLogger := logger.WithGroup("frontend")
	routerLogger := logger.WithGroup("router")

	slog.SetDefault(appLogger)

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
	routerConfig := router.Config{
		AdminToken: effectiveAdminToken,
		ApiToken:   *apiToken,
		EnableWeb:  *enableWeb,
		UserAgent:  *userAgent,
		WebDir:     *webDir,
	}
	if err := router.SetupRoutes(fiberApp, svcs, routerConfig, routerLogger); err != nil {
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
