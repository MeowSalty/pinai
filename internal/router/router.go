package router

import (
	"log/slog"

	appbootstrap "github.com/MeowSalty/pinai/internal/bootstrap"
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
func SetupRoutes(web *gin.Engine, svcs *appbootstrap.Services, config Config, logger *slog.Logger) error {
	setupCORS(web, config)
	webAPI := setupAPIRootGroup(web, config)

	SetupControlPlaneRoutes(webAPI, svcs, ControlConfig{
		ApiToken:     config.ApiToken,
		AdminToken:   config.AdminToken,
		UserAgent:    config.UserAgent,
		ProxyEnabled: config.ProxyEnabled,
	}, logger)
	SetupDataPlaneRoutes(web, svcs, DataPlaneConfig{
		ApiToken:           config.ApiToken,
		UserAgent:          config.UserAgent,
		PassthroughHeaders: config.PassthroughHeaders,
	}, logger)

	setupFrontendRoutes(web, config)

	return nil
}
