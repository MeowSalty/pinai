package frontend

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/valyala/fasthttp"
)

// GitHubReleaseAsset 表示 GitHub 发布资产
type GitHubReleaseAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

// GitHubRelease 表示 GitHub 发布信息
type GitHubRelease struct {
	TagName string               `json:"tag_name"`
	Name    string               `json:"name"`
	Assets  []GitHubReleaseAsset `json:"assets"`
}

// VersionInfo 存储当前前端版本信息
type VersionInfo struct {
	Version string `json:"version"`
}

const (
	// VersionFileName 版本信息文件名
	VersionFileName = ".version"
	// GitHubAPIURL GitHub API 地址
	GitHubAPIURL = "https://api.github.com/repos/MeowSalty/pinai-frontend/releases/latest"
)

var (
	// githubProxy GitHub 代理地址
	githubProxy string
)

// InitializeWeb 初始化前端支持
func InitializeWeb(logger *slog.Logger, webDir *string, checkUpdate bool, proxy string) error {
	githubProxy = proxy
	*webDir = "data/" + *webDir
	logger.Info("初始化前端支持", "webDir", *webDir, "checkUpdate", checkUpdate)

	// 创建前端目录
	if err := os.MkdirAll(*webDir, 0755); err != nil {
		return fmt.Errorf("创建前端目录失败：%w", err)
	}

	// 检查前端目录是否为空
	entries, err := os.ReadDir(*webDir)
	if err != nil {
		return fmt.Errorf("读取前端目录失败：%w", err)
	}

	// 如果目录为空，从 GitHub 下载最新发布的前端文件
	if len(entries) == 0 {
		logger.Info("前端目录为空，从 GitHub 下载最新发布的前端文件")
		return DownloadLatestFrontendRelease(logger, webDir)
	}

	// 目录不为空，如果启用了更新检查，则检查是否存在新版本
	if checkUpdate {
		logger.Info("前端目录已包含文件，检查更新")
		err = CheckAndUpdateFrontend(logger, webDir)
		if err != nil {
			logger.Warn("检查前端更新失败", slog.String("error", err.Error()))
		}
		return nil
	}

	logger.Info("前端目录已包含文件，更新检查已禁用")
	return nil
}

// CheckAndUpdateFrontend 检查并更新前端文件
func CheckAndUpdateFrontend(logger *slog.Logger, webDir *string) error {
	// 获取当前版本
	currentVersion, err := getCurrentVersion(webDir)
	if err != nil {
		logger.Warn("获取当前前端版本失败", "error", err)
	}

	// 获取最新版本信息
	latestRelease, err := getLatestRelease(logger)
	if err != nil {
		return err
	}

	// 比较版本
	if currentVersion != latestRelease.TagName {
		logger.Info("发现新版本前端文件", "current", currentVersion, "latest", latestRelease.TagName)
		// 需要更新前端文件
		return updateFrontend(logger, webDir, latestRelease)
	}

	logger.Info("前端文件已是最新版本", "version", currentVersion)
	return nil
}

// getCurrentVersion 获取当前前端版本
func getCurrentVersion(webDir *string) (string, error) {
	versionFile := filepath.Join(*webDir, VersionFileName)
	content, err := os.ReadFile(versionFile)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil // 文件不存在，返回空版本
		}
		return "", fmt.Errorf("读取版本文件失败：%w", err)
	}

	var versionInfo VersionInfo
	if err := json.Unmarshal(content, &versionInfo); err != nil {
		return "", fmt.Errorf("解析版本信息失败：%w", err)
	}

	return versionInfo.Version, nil
}

// getLatestRelease 获取最新的 GitHub 发布信息
func getLatestRelease(logger *slog.Logger) (*GitHubRelease, error) {
	// GitHub API URL
	releaseURL := GitHubAPIURL

	// 如果配置了 GitHub 代理，则使用代理地址
	if githubProxy != "" {
		releaseURL = githubProxy + "/" + GitHubAPIURL
		logger.Info("使用 GitHub 代理", "proxy", githubProxy, "url", releaseURL)
	}

	// 获取最新发布信息
	logger.Info("获取最新发布信息", "url", releaseURL)
	agent := fiber.Get(releaseURL).Timeout(5 * time.Second)
	statusCode, body, errs := agent.Bytes()
	if len(errs) > 0 {
		// 构建错误信息，包含错误数量和每个错误的详细信息
		errorMessages := make([]string, len(errs))
		for i, err := range errs {
			errorMessages[i] = err.Error()
		}

		return nil, fmt.Errorf("获取发布信息失败，共 %d 个错误: %s", len(errs), strings.Join(errorMessages, "; "))
	}

	if statusCode != fiber.StatusOK {
		return nil, fmt.Errorf("获取发布信息失败，状态码：%d", statusCode)
	}

	// 解析响应
	var release GitHubRelease
	if err := json.Unmarshal(body, &release); err != nil {
		return nil, fmt.Errorf("解析发布信息失败：%w", err)
	}

	logger.Info("找到最新发布", "tag", release.TagName, "name", release.Name)
	return &release, nil
}

// downloadAndExtractFrontend 从指定 URL 下载前端资产并解压到目标目录
func downloadAndExtractFrontend(logger *slog.Logger, webDir *string, frontendAsset *GitHubReleaseAsset) error {
	logger.Info("找到前端资产文件", "name", frontendAsset.Name, "url", frontendAsset.BrowserDownloadURL)

	// 下载资产文件
	logger.Info("下载前端资产文件...")

	client := &fasthttp.Client{
		ReadBufferSize: 8192, // 增加读取缓冲区大小以处理大的响应头
	}

	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	statusCode, body, err := client.Get(resp.Body(), frontendAsset.BrowserDownloadURL)
	if err != nil {
		return fmt.Errorf("下载文件失败：%w", err)
	}

	if statusCode != fasthttp.StatusOK {
		return fmt.Errorf("下载文件失败，状态码：%d", statusCode)
	}

	// 保存到临时文件
	tmpFile := filepath.Join(*webDir, "frontend.zip")
	if err := os.WriteFile(tmpFile, body, 0644); err != nil {
		// 尝试删除可能创建的临时文件
		if rm_err := os.Remove(tmpFile); rm_err != nil {
			return fmt.Errorf("保存临时文件失败：%w，并尝试删除失败：%w", err, rm_err)
		}
		return fmt.Errorf("保存临时文件失败：%w", err)
	}

	logger.Info("前端资产文件下载完成", "file", tmpFile)

	// 解压文件
	logger.Info("解压前端文件...", slog.String("file", tmpFile))
	if err := Unzip(tmpFile, *webDir); err != nil {
		// 尝试删除临时文件
		if rm_err := os.Remove(tmpFile); rm_err != nil {
			return fmt.Errorf("解压文件失败：%w，并尝试删除失败：%w", err, rm_err)
		}
		return fmt.Errorf("解压文件失败：%w", err)
	}

	// 删除临时文件
	if err := os.Remove(tmpFile); err != nil {
		logger.Warn("删除临时文件失败", "error", err)
	}

	return nil
}

// findFrontendAsset 查找前端资产文件
func findFrontendAsset(assets []GitHubReleaseAsset) (*GitHubReleaseAsset, error) {
	for _, asset := range assets {
		if filepath.Ext(asset.Name) == ".zip" {
			return &asset, nil
		}
	}
	return nil, fmt.Errorf("未找到前端资产文件")
}

// updateFrontend 更新前端文件
func updateFrontend(logger *slog.Logger, webDir *string, release *GitHubRelease) error {
	// 查找前端资产文件 (zip 格式)
	frontendAsset, err := findFrontendAsset(release.Assets)
	if err != nil {
		return err
	}

	// 清空前端目录（保留版本文件）
	if err := clearWebDir(webDir); err != nil {
		return fmt.Errorf("清空前端目录失败：%w", err)
	}

	// 下载并解压前端文件
	if err := downloadAndExtractFrontend(logger, webDir, frontendAsset); err != nil {
		return fmt.Errorf("下载并解压前端文件失败：%w", err)
	}

	// 保存新版本信息
	if err := saveVersionInfo(webDir, release.TagName); err != nil {
		logger.Warn("保存版本信息失败", "error", err)
		// 这里不返回错误，因为下载和解压已经成功
	}

	logger.Info("前端文件更新完成", "version", release.TagName)
	return nil
}

// clearWebDir 清空前端目录，但保留版本文件
func clearWebDir(webDir *string) error {
	entries, err := os.ReadDir(*webDir)
	if err != nil {
		return fmt.Errorf("读取目录失败：%w", err)
	}

	for _, entry := range entries {
		// 保留版本文件
		if entry.Name() == VersionFileName {
			continue
		}

		// 保留临时下载的前端 zip 文件
		if entry.Name() == "frontend" {
			continue
		}

		// 删除其他文件和目录
		path := filepath.Join(*webDir, entry.Name())
		if err := os.RemoveAll(path); err != nil {
			return fmt.Errorf("删除文件失败 %s: %w", path, err)
		}
	}

	return nil
}

// saveVersionInfo 保存版本信息
func saveVersionInfo(webDir *string, version string) error {
	versionInfo := VersionInfo{Version: version}
	data, err := json.Marshal(versionInfo)
	if err != nil {
		return fmt.Errorf("序列化版本信息失败：%w", err)
	}

	versionFile := filepath.Join(*webDir, VersionFileName)
	if err := os.WriteFile(versionFile, data, 0644); err != nil {
		return fmt.Errorf("写入版本文件失败：%w", err)
	}
	return nil
}

// DownloadLatestFrontendRelease 从 GitHub 下载最新发布的前端文件
func DownloadLatestFrontendRelease(logger *slog.Logger, webDir *string) error {
	// 获取最新版本信息
	release, err := getLatestRelease(logger)
	if err != nil {
		return fmt.Errorf("获取最新版本信息失败：%w", err)
	}

	// 查找前端资产文件 (zip 格式)
	frontendAsset, err := findFrontendAsset(release.Assets)
	if err != nil {
		return err
	}

	// 下载并解压前端文件
	if err := downloadAndExtractFrontend(logger, webDir, frontendAsset); err != nil {
		return fmt.Errorf("下载并解压前端文件失败：%w", err)
	}

	// 保存版本信息
	if err := saveVersionInfo(webDir, release.TagName); err != nil {
		logger.Warn("保存版本信息失败", "error", err)
		// 这里不返回错误，因为下载和解压已经成功
	}

	logger.Info("前端文件初始化完成", "version", release.TagName)
	return nil
}

// Unzip 解压 zip 文件到指定目录
func Unzip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return fmt.Errorf("打开 zip 文件失败：%w", err)
	}
	defer r.Close()

	for _, f := range r.File {
		// 构建目标文件路径
		fpath := filepath.Join(dest, f.Name)

		// 检查文件路径遍历漏洞
		if !strings.HasPrefix(fpath, filepath.Clean(dest)+string(os.PathSeparator)) {
			return fmt.Errorf("%s: 非法文件路径", fpath)
		}

		if f.FileInfo().IsDir() {
			// 创建目录
			if err := os.MkdirAll(fpath, os.ModePerm); err != nil {
				return fmt.Errorf("创建目录失败 %s: %w", fpath, err)
			}
			continue
		}

		// 创建文件所在的目录
		if err := os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return fmt.Errorf("创建文件目录失败 %s: %w", filepath.Dir(fpath), err)
		}

		// 打开文件
		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return fmt.Errorf("创建文件失败 %s: %w", fpath, err)
		}

		// 打开压缩文件中的文件
		rc, err := f.Open()
		if err != nil {
			_ = outFile.Close()
			return fmt.Errorf("打开压缩文件失败 %s: %w", f.Name, err)
		}

		// 复制文件内容
		_, err = io.Copy(outFile, rc)

		// 关闭文件
		outFile.Close()
		rc.Close()

		if err != nil {
			return fmt.Errorf("复制文件内容失败 %s: %w", fpath, err)
		}
	}
	return nil
}
