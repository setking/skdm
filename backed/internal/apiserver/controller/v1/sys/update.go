package sys

import (
	"changeme/backed/cmd"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
)

func init() {
	application.RegisterEvent[UpdateCheckResult]("update-check")
}

type UpdateSrv interface {
}

// UpdateCheckResult 更新检查结果
type UpdateCheckResult struct {
	HasUpdate      bool   `json:"has_update"`
	CurrentVersion string `json:"current_version"`
	LatestVersion  string `json:"latest_version"`
	ReleaseURL     string `json:"release_url"`
	ReleaseNotes   string `json:"release_notes"`
	DownloadURL    string `json:"download_url"`
	Error          string `json:"error,omitempty"`
}

// GitHubRelease 表示 GitHub Release API 响应的部分字段
type GitHubRelease struct {
	TagName string        `json:"tag_name"`
	HTMLURL string        `json:"html_url"`
	Body    string        `json:"body"`
	Assets  []GitHubAsset `json:"assets"`
}

// GitHubAsset 表示 GitHub Release 中的附件
type GitHubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

// GetAppVersion 返回当前应用版本号
func GetAppVersion() string {
	return cmd.AppVersion
}

// CheckForUpdate 检查 GitHub Release 是否有新版本
func CheckForUpdate() *UpdateCheckResult {
	result := &UpdateCheckResult{
		HasUpdate:      false,
		CurrentVersion: cmd.AppVersion,
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get("https://api.github.com/repos/setking/skdm/releases/latest")
	if err != nil {
		result.Error = fmt.Sprintf("网络请求失败: %v", err)
		return result
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		result.Error = fmt.Sprintf("GitHub API 返回状态码 %d", resp.StatusCode)
		return result
	}

	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		result.Error = fmt.Sprintf("解析响应失败: %v", err)
		return result
	}

	result.LatestVersion = extractVersionFromTag(release.TagName)
	result.ReleaseURL = release.HTMLURL
	result.ReleaseNotes = release.Body
	// 取第一个附件作为下载链接
	if len(release.Assets) > 0 {
		result.DownloadURL = release.Assets[0].BrowserDownloadURL
	}
	result.HasUpdate = compareVersions(result.LatestVersion, cmd.AppVersion) > 0
	return result
}

// extractVersionFromTag 从 tag 名称中提取版本号
// 例如 "v0.1.1-a9fa3ac" → "0.1.1"
func extractVersionFromTag(tag string) string {
	tag = strings.TrimPrefix(tag, "v")
	// 去掉 -commit_hash 后缀
	if idx := strings.Index(tag, "-"); idx != -1 {
		tag = tag[:idx]
	}
	return tag
}

// compareVersions 比较两个 semver 版本号
// 返回 >0 如果 a > b, 0 如果等于, <0 如果 a < b
func compareVersions(a, b string) int {
	partsA := parseVersion(a)
	partsB := parseVersion(b)

	for i := 0; i < 3; i++ {
		if partsA[i] > partsB[i] {
			return 1
		}
		if partsA[i] < partsB[i] {
			return -1
		}
	}
	return 0
}

// parseVersion 解析 "MAJOR.MINOR.PATCH" 格式的版本号
func parseVersion(v string) [3]int {
	var parts [3]int
	_, _ = fmt.Sscanf(strings.TrimSpace(v), "%d.%d.%d", &parts[0], &parts[1], &parts[2])
	return parts
}
