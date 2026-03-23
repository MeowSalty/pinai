package router

import (
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

// setupFrontendRoutes 配置前端静态资源与 SPA 回退路由。
func setupFrontendRoutes(web *gin.Engine, config Config) {
	if !config.EnableWeb {
		return
	}

	indexFilePath := filepath.Join(config.WebDir, "index.html")

	// 根路径返回 index.html
	web.GET("/", func(c *gin.Context) {
		c.File(indexFilePath)
	})

	// 添加兜底路由：优先返回静态文件，不存在时回退 index.html 以支持 SPA
	web.NoRoute(func(c *gin.Context) {
		requestPath := normalizeRequestPath(c.Request.URL.Path)

		// 保留 API 路径的 404 行为，避免被前端兜底路由吞掉
		if isReservedAPIPath(requestPath) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "接口不存在",
			})
			return
		}

		// 仅对浏览器常见的 GET/HEAD 请求做前端路由回退
		if c.Request.Method != http.MethodGet && c.Request.Method != http.MethodHead {
			c.Status(http.StatusNotFound)
			return
		}

		if requestPath != "/" {
			if filePath, ok := resolveFrontendFilePath(config.WebDir, requestPath); ok {
				fileInfo, err := os.Stat(filePath)
				if err == nil && !fileInfo.IsDir() {
					c.File(filePath)
					return
				}
			}
		}

		c.File(indexFilePath)
	})
}

// normalizeRequestPath 规范化请求路径，确保结果始终以 / 开头。
func normalizeRequestPath(rawPath string) string {
	cleanedPath := path.Clean(rawPath)
	if cleanedPath == "." {
		return "/"
	}

	if !strings.HasPrefix(cleanedPath, "/") {
		return "/" + cleanedPath
	}

	return cleanedPath
}

// resolveFrontendFilePath 根据请求路径解析前端静态文件绝对路径。
func resolveFrontendFilePath(webDir, requestPath string) (string, bool) {
	if requestPath == "/" {
		return "", false
	}

	relativePath := strings.TrimPrefix(requestPath, "/")
	resolvedPath := filepath.Join(webDir, filepath.FromSlash(relativePath))

	if !isPathWithinBase(webDir, resolvedPath) {
		return "", false
	}

	return resolvedPath, true
}

// isPathWithinBase 判断目标路径是否在基础目录内，避免路径穿越。
func isPathWithinBase(basePath, targetPath string) bool {
	baseAbsPath, err := filepath.Abs(basePath)
	if err != nil {
		return false
	}

	targetAbsPath, err := filepath.Abs(targetPath)
	if err != nil {
		return false
	}

	relativePath, err := filepath.Rel(baseAbsPath, targetAbsPath)
	if err != nil {
		return false
	}

	if relativePath == "." {
		return true
	}

	return relativePath != ".." && !strings.HasPrefix(relativePath, ".."+string(filepath.Separator))
}

// isReservedAPIPath 判断路径是否属于后端 API 前缀。
func isReservedAPIPath(requestPath string) bool {
	reservedPrefixes := []string{"/api", "/openai/v1", "/anthropic/v1", "/multi"}
	for _, prefix := range reservedPrefixes {
		if requestPath == prefix || strings.HasPrefix(requestPath, prefix+"/") {
			return true
		}
	}

	return false
}
