package router

import (
	"log/slog"

	datarouter "github.com/MeowSalty/pinai/internal/router"
	"github.com/MeowSalty/pinai/services"
	"github.com/gin-gonic/gin"
)

type Config struct {
	EnableWeb          bool
	CORSAllowAll       bool
	WebDir             string
	ApiToken           string
	AdminToken         string
	UserAgent          string
	PassthroughHeaders bool
	ProxyEnabled       bool
}

// SetupRoutes 配置 API 路由
func SetupRoutes(web *gin.Engine, svcs *services.Services, config Config, logger *slog.Logger) error {
	setupCORS(web, config)
	webAPI := setupAPIRootGroup(web, config)

	setupControlPlaneRoutes(webAPI, svcs, config, logger)
	datarouter.SetupDataPlaneRoutes(web, svcs, datarouter.Config{
		ApiToken:           config.ApiToken,
		UserAgent:          config.UserAgent,
		PassthroughHeaders: config.PassthroughHeaders,
	}, logger)

	setupFrontendRoutes(web, config)

	return nil
}
