package models

// ApkInfo 返回给前端列表的 APK 信息
type ApkInfo struct {
	FileName    string `json:"fileName"`
	AppName     string `json:"appName"`
	PackageName string `json:"packageName"`
	VersionName string `json:"versionName"`
	VersionCode uint32 `json:"versionCode"`
	FileSize    string `json:"fileSize"`
	IconBase64  string `json:"iconBase64"`
	UploadTime  string `json:"uploadTime"`

	// 注入的策略信息
	RolloutRate         int    `json:"rolloutRate"`
	MinForceVersionCode uint32 `json:"minForceVersionCode"`
}

// AppConfig 单个应用的发布策略配置 (存储于 metadata.json)
type AppConfig struct {
	PackageName         string `json:"packageName"`
	LatestVersionCode   uint32 `json:"latestVersionCode"`
	LatestVersionName   string `json:"latestVersionName"`
	LatestFileName      string `json:"latestFileName"`
	MinForceVersionCode uint32 `json:"minForceVersionCode"` // 低于此版本强制更新
	RolloutRate         int    `json:"rolloutRate"`         // 灰度比例 (0-10000)
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type ConfigUpdateRequest struct {
	PackageName         string `json:"packageName"`
	MinForceVersionCode uint32 `json:"minForceVersionCode"`
	RolloutRate         int    `json:"rolloutRate"`
}
