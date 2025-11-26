package handlers

import (
	"apkdepot/internal/config"
	"apkdepot/internal/models"
	"apkdepot/internal/store"
	"apkdepot/internal/utils"
	"encoding/json"
	"fmt"
	"github.com/shogo82148/androidbinary/apk"
	"hash/crc32"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
)

// CheckUpdate APP 自动检查更新接口
func CheckUpdate(w http.ResponseWriter, r *http.Request) {
	pkgName := r.URL.Query().Get("packageName")
	verStr := r.URL.Query().Get("versionCode")
	deviceId := r.URL.Query().Get("deviceId")

	if pkgName == "" || verStr == "" || deviceId == "" {
		http.Error(w, `{"error":"Missing params"}`, http.StatusBadRequest)
		return
	}

	clientVer, _ := strconv.Atoi(verStr)
	cfg := store.GetAppConfig(pkgName)

	resp := map[string]interface{}{"hasUpdate": false}
	w.Header().Set("Content-Type", "application/json")

	// 无配置或客户端已是最新
	if cfg == nil || cfg.LatestVersionCode <= uint32(clientVer) {
		json.NewEncoder(w).Encode(resp)
		return
	}

	// 灰度算法判断
	if !isHitGrayScale(deviceId, cfg.RolloutRate) {
		json.NewEncoder(w).Encode(resp)
		return
	}

	// 命中更新
	force := uint32(clientVer) < cfg.MinForceVersionCode

	resp["hasUpdate"] = true
	resp["force"] = force
	resp["versionCode"] = cfg.LatestVersionCode
	resp["versionName"] = cfg.LatestVersionName
	// 假设前端通过 /apks/ 路径访问
	resp["downloadUrl"] = fmt.Sprintf("/apks/%s", cfg.LatestFileName)

	json.NewEncoder(w).Encode(resp)
}

// UpdateConfig 管理端修改发布策略
func UpdateConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req models.ConfigUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"Invalid JSON"}`, http.StatusBadRequest)
		return
	}

	store.UpdateConfig(req.PackageName, func(c *models.AppConfig) {
		c.MinForceVersionCode = req.MinForceVersionCode
		c.RolloutRate = req.RolloutRate
	})
	store.SaveMetadata()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Config updated"})
}

// VersionList APP 手动版本列表接口
func VersionList(w http.ResponseWriter, r *http.Request) {
	pkgName := r.URL.Query().Get("packageName")
	if pkgName == "" {
		http.Error(w, `{"error":"Missing packageName"}`, http.StatusBadRequest)
		return
	}

	// 1. 扫描该包名的所有 APK (简化的扫描逻辑，不带 Icon)
	apks := scanApksForVersionList(pkgName)

	// 2. 获取配置中的基线版本
	cfg := store.GetAppConfig(pkgName)
	minForce := uint32(0)
	if cfg != nil {
		minForce = cfg.MinForceVersionCode
	}

	resp := map[string]interface{}{
		"minForceVersionCode": minForce,
		"versions":            apks,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// --- 内部工具 ---

func isHitGrayScale(id string, rate int) bool {
	if rate <= 0 {
		return false
	}
	if rate >= 10000 {
		return true
	}
	hash := crc32.ChecksumIEEE([]byte(id))
	return int(hash%10000) < rate
}

func scanApksForVersionList(targetPkg string) []models.ApkInfo {
	files, _ := os.ReadDir(config.ApkDir)
	var apks []models.ApkInfo

	for _, file := range files {
		if file.IsDir() || filepath.Ext(file.Name()) != ".apk" {
			continue
		}

		path := filepath.Join(config.ApkDir, file.Name())
		pkg, err := apk.OpenFile(path)
		if err != nil {
			continue
		}

		if pkg.PackageName() == targetPkg {
			info, _ := file.Info()
			apks = append(apks, models.ApkInfo{
				FileName:    file.Name(),
				VersionName: pkg.Manifest().VersionName.MustString(),
				VersionCode: uint32(pkg.Manifest().VersionCode.MustInt32()),
				FileSize:    utils.FormatFileSize(info.Size()),
				UploadTime:  info.ModTime().Format("2006-01-02"),
				// 这里不需要 Icon，节省流量
			})
		}
		pkg.Close()
	}
	// 倒序排列
	sort.Slice(apks, func(i, j int) bool {
		return apks[i].VersionCode > apks[j].VersionCode
	})
	return apks
}
