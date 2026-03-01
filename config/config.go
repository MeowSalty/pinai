package config

import (
	"flag"
)

// Config 应用配置
type Config struct {
	// 服务器配置
	Port string
	Prod bool

	// 前端配置
	EnableWeb            bool
	WebDir               string
	EnableFrontendUpdate bool
	PassthroughHeaders   bool

	// 数据库配置
	DBType      string
	DBHost      string
	DBPort      string
	DBUser      string
	DBPass      string
	DBName      string
	DBSSLMode   string
	DBTLSConfig string

	// API Token 配置
	APIToken   string
	AdminToken string

	// GitHub 代理配置
	GitHubProxy string

	// 模型映射规则配置
	ModelMapping string

	// 日志配置
	LogLevel string

	// User-Agent 配置
	UserAgent string
}

// LoadConfig 加载配置
func LoadConfig() *Config {
	// 从环境变量加载默认值
	env := LoadEnv()

	cfg := &Config{
		Port:                 env.Port,
		Prod:                 env.Prod,
		EnableWeb:            env.EnableWeb,
		WebDir:               env.WebDir,
		EnableFrontendUpdate: env.EnableFrontendUpdate,
		PassthroughHeaders:   env.PassthroughHeaders,
		DBType:               env.DBType,
		DBHost:               env.DBHost,
		DBPort:               env.DBPort,
		DBUser:               env.DBUser,
		DBPass:               env.DBPass,
		DBName:               env.DBName,
		DBSSLMode:            env.DBSSLMode,
		DBTLSConfig:          env.DBTLSConfig,
		APIToken:             env.APIToken,
		AdminToken:           env.AdminToken,
		GitHubProxy:          env.GitHubProxy,
		ModelMapping:         env.ModelMapping,
		LogLevel:             env.LogLevel,
		UserAgent:            env.UserAgent,
	}

	// 从命令行参数加载配置
	cfg.loadFlags()

	return cfg
}

// loadFlags 从命令行参数加载配置
func (c *Config) loadFlags() {
	flag.StringVar(&c.Port, "port", c.Port, "监听端口")
	flag.BoolVar(&c.Prod, "prod", c.Prod, "在生产环境中启用 prefork")

	// 前端相关参数
	flag.BoolVar(&c.EnableWeb, "enable-web", c.EnableWeb, "启用前端支持")
	flag.StringVar(&c.WebDir, "web-dir", c.WebDir, "前端文件目录")
	flag.BoolVar(&c.EnableFrontendUpdate, "enable-frontend-update", c.EnableFrontendUpdate, "启用前端更新检查")
	flag.BoolVar(&c.PassthroughHeaders, "passthrough-headers", c.PassthroughHeaders, "是否透传 HTTP 请求头到 portal 请求 headers")

	// 数据库相关参数
	flag.StringVar(&c.DBType, "db-type", c.DBType, "数据库类型 (sqlite, mysql, postgres)")
	flag.StringVar(&c.DBHost, "db-host", c.DBHost, "数据库主机地址")
	flag.StringVar(&c.DBPort, "db-port", c.DBPort, "数据库端口")
	flag.StringVar(&c.DBUser, "db-user", c.DBUser, "数据库用户名")
	flag.StringVar(&c.DBPass, "db-pass", c.DBPass, "数据库密码")
	flag.StringVar(&c.DBName, "db-name", c.DBName, "数据库名称")
	flag.StringVar(&c.DBSSLMode, "db-ssl-mode", c.DBSSLMode, "PostgreSQL SSL 模式 (disable, require, verify-ca, verify-full)")
	flag.StringVar(&c.DBTLSConfig, "db-tls-config", c.DBTLSConfig, "MySQL TLS 配置 (true, false, skip-verify, preferred)")

	// API Token 参数
	flag.StringVar(&c.APIToken, "api-token", c.APIToken, "API Token，如果为空则不启用身份验证")
	flag.StringVar(&c.AdminToken, "admin-token", c.AdminToken, "管理 API Token，如果为空则使用 API Token")

	// GitHub 代理参数
	flag.StringVar(&c.GitHubProxy, "github-proxy", c.GitHubProxy, "GitHub 代理地址，用于加速 GitHub 访问")

	// 模型映射规则参数
	flag.StringVar(&c.ModelMapping, "model-mapping", c.ModelMapping, "模型映射规则，格式：key1:value1,key2:value2")

	// 日志等级参数
	flag.StringVar(&c.LogLevel, "log-level", c.LogLevel, "日志输出等级 (DEBUG, INFO, WARN, ERROR)")

	// User-Agent 参数
	flag.StringVar(&c.UserAgent, "user-agent", c.UserAgent, "User-Agent 配置，空则透传客户端 UA，\"default\" 使用 fasthttp 默认值，其他字符串则复写")

	flag.Parse()
}
