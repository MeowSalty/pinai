package config

import "os"

// Env 环境变量配置
type Env struct {
	Port                 string
	Prod                 bool
	EnableWeb            bool
	WebDir               string
	EnableFrontendUpdate bool
	PassthroughHeaders   bool
	DBType               string
	DBHost               string
	DBPort               string
	DBUser               string
	DBPass               string
	DBName               string
	DBSSLMode            string // PostgreSQL SSL 模式
	DBTLSConfig          string // MySQL TLS 配置
	APIToken             string
	AdminToken           string // 管理 API Token
	GitHubProxy          string // GitHub 代理地址
	ModelMapping         string // 模型映射规则，格式：key1:value1,key2:value2
	LogLevel             string // 日志输出等级
	UserAgent            string // User-Agent 配置
}

// LoadEnv 从环境变量加载配置
func LoadEnv() *Env {
	return &Env{
		Port:                 getEnvOrDefault("PORT", ":3000"),
		Prod:                 getEnvOrDefault("PROD", "") == "true",
		EnableWeb:            getEnvOrDefault("ENABLE_WEB", "") == "true",
		WebDir:               getEnvOrDefault("WEB_DIR", "web"),
		EnableFrontendUpdate: getEnvOrDefault("ENABLE_FRONTEND_UPDATE", "true") == "true",
		PassthroughHeaders:   getEnvOrDefault("PASSTHROUGH_HEADERS", "true") == "true",
		DBType:               getEnvOrDefault("DB_TYPE", "sqlite"),
		DBHost:               getEnvOrDefault("DB_HOST", ""),
		DBPort:               getEnvOrDefault("DB_PORT", ""),
		DBUser:               getEnvOrDefault("DB_USER", ""),
		DBPass:               getEnvOrDefault("DB_PASS", ""),
		DBName:               getEnvOrDefault("DB_NAME", ""),
		DBSSLMode:            getEnvOrDefault("DB_SSL_MODE", ""),
		DBTLSConfig:          getEnvOrDefault("DB_TLS_CONFIG", ""),
		APIToken:             getEnvOrDefault("API_TOKEN", ""),
		AdminToken:           getEnvOrDefault("ADMIN_TOKEN", ""),
		GitHubProxy:          getEnvOrDefault("GITHUB_PROXY", ""),
		ModelMapping:         getEnvOrDefault("MODEL_MAPPING", ""),
		LogLevel:             getEnvOrDefault("LOG_LEVEL", "INFO"),
		UserAgent:            getEnvOrDefault("USER_AGENT", ""),
	}
}

// getEnvOrDefault 获取环境变量，如果不存在则返回默认值
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
