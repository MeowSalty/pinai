package router

import (
	"log/slog"

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
	setupDataPlaneRoutes(web, svcs, config, logger)

	setupFrontendRoutes(web, config)

	return nil
}
