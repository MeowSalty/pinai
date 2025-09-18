package main

import "os"

var (
	// 定义默认值变量
	envPort      = ":3000"
	envProd      = false
	envEnableWeb = false
	envWebDir    = "./web"
	envDBType    = "sqlite"
	envDBHost    = ""
	envDBPort    = ""
	envDBUser    = ""
	envDBPass    = ""
	envDBName    = ""
)

// LoadEnvConfig 从环境变量加载配置
func LoadEnv() {
	// 从环境变量加载端口配置
	if os.Getenv("PORT") != "" {
		envPort = os.Getenv("PORT")
	}

	// 从环境变量加载生产环境配置
	if os.Getenv("PROD") != "" {
		envProd = os.Getenv("PROD") == "true"
	}

	// 从环境变量加载前端配置
	if os.Getenv("ENABLE_WEB") != "" {
		envEnableWeb = os.Getenv("ENABLE_WEB") == "true"
	}

	if os.Getenv("WEB_DIR") != "" {
		envWebDir = os.Getenv("WEB_DIR")
	}

	// 从环境变量加载数据库配置
	if os.Getenv("DB_TYPE") != "" {
		envDBType = os.Getenv("DB_TYPE")
	}

	if os.Getenv("DB_HOST") != "" {
		envDBHost = os.Getenv("DB_HOST")
	}

	if os.Getenv("DB_PORT") != "" {
		envDBPort = os.Getenv("DB_PORT")
	}

	if os.Getenv("DB_USER") != "" {
		envDBUser = os.Getenv("DB_USER")
	}

	if os.Getenv("DB_PASS") != "" {
		envDBPass = os.Getenv("DB_PASS")
	}

	if os.Getenv("DB_NAME") != "" {
		envDBName = os.Getenv("DB_NAME")
	}
}
