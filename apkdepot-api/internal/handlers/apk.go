package handlers

import (
	"apkdepot/internal/config"
	"apkdepot/internal/models"
	"apkdepot/internal/store"
	"apkdepot/internal/utils"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image/png"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/shogo82148/androidbinary/apk"
)

// ListApks 列出所有 APK (注入配置信息)
func ListApks(w http.ResponseWriter, r *http.Request) {
	files, err := os.ReadDir(config.ApkDir)
	if err != nil {
		http.Error(w, `{"error":"Could not read APK directory"}`, http.StatusInternalServerError)
		return
	}

	var apks []models.ApkInfo
	for _, file := range files {
		if file.IsDir() || filepath.Ext(file.Name()) != ".apk" {
			continue
		}

		filePath := filepath.Join(config.ApkDir, file.Name())
		pkg, err := apk.OpenFile(filePath)
		if err != nil {
			log.Printf("Warning: Could not parse %s: %v", file.Name(), err)
			continue
		}

		fileInfo, _ := file.Info()

		// 解析图标
		icon, err := pkg.Icon(nil)
		var iconB64 string
		if err == nil && icon != nil {
			var buf bytes.Buffer
			if err := png.Encode(&buf, icon); err == nil {
				iconB64 = base64.StdEncoding.EncodeToString(buf.Bytes())
			}
		}

		pkgName := pkg.PackageName()
		// 获取配置策略
		cfg := store.GetAppConfig(pkgName)

		rate := 0
		minForce := uint32(0)
		if cfg != nil {
			rate = cfg.RolloutRate
			minForce = cfg.MinForceVersionCode
		}

		apks = append(apks, models.ApkInfo{
			FileName:            file.Name(),
			AppName:             pkg.Manifest().App.Label.MustString(),
			PackageName:         pkgName,
			VersionName:         pkg.Manifest().VersionName.MustString(),
			VersionCode:         uint32(pkg.Manifest().VersionCode.MustInt32()),
			FileSize:            utils.FormatFileSize(fileInfo.Size()),
			IconBase64:          iconB64,
			UploadTime:          fileInfo.ModTime().Format("2006-01-02 15:04:05"),
			RolloutRate:         rate,
			MinForceVersionCode: minForce,
		})
		pkg.Close()
	}

	sort.Slice(apks, func(i, j int) bool {
		return apks[i].UploadTime > apks[j].UploadTime
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(apks)
}

// UploadApk 上传并自动更新元数据
func UploadApk(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, config.MaxUploadSize)
	if err := r.ParseMultipartForm(config.MaxUploadSize); err != nil {
		http.Error(w, `{"error":"File too big"}`, http.StatusBadRequest)
		return
	}

	file, _, err := r.FormFile("apkfile")
	if err != nil {
		http.Error(w, `{"error":"Invalid file"}`, http.StatusBadRequest)
		return
	}
	defer file.Close()

	// 1. 先保存到临时文件，以便读取包名和版本号
	tempFile, err := os.CreateTemp("", "upload-*.apk")
	if err != nil {
		http.Error(w, `{"error":"Temp file creation failed"}`, http.StatusInternalServerError)
		return
	}
	defer os.Remove(tempFile.Name()) // 确保退出时删除临时文件
	defer tempFile.Close()

	if _, err := io.Copy(tempFile, file); err != nil {
		http.Error(w, `{"error":"Temp write failed"}`, http.StatusInternalServerError)
		return
	}

	// 2. 解析 APK 信息
	pkg, err := apk.OpenFile(tempFile.Name())
	if err != nil {
		http.Error(w, `{"error":"Invalid APK file"}`, http.StatusBadRequest)
		return
	}
	pkgName := pkg.PackageName()
	verCode := pkg.Manifest().VersionCode.MustInt32()
	pkg.Close()

	// 3. 构造唯一文件名: PackageName_VersionCode.apk
	// 例如: com.xxliu.adflowpro_102.apk
	finalFileName := fmt.Sprintf("%s_%d.apk", pkgName, verCode)
	dstPath := filepath.Join(config.ApkDir, finalFileName)

	// 4. 将临时文件内容写入最终目标文件 (覆盖同版本号旧文件)
	// 重置临时文件指针
	tempFile.Seek(0, 0)
	dst, err := os.Create(dstPath)
	if err != nil {
		http.Error(w, `{"error":"Save failed"}`, http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, tempFile); err != nil {
		http.Error(w, `{"error":"Final write failed"}`, http.StatusInternalServerError)
		return
	}

	// 上传成功，解析并更新元数据
	updateMetadataFromFile(dstPath, finalFileName)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Upload successful",
		"savedAs": finalFileName,
	})
}

func DeleteApk(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	fileName := filepath.Base(r.URL.Path)
	if fileName == "." || fileName == "/" || strings.Contains(fileName, "..") {
		http.Error(w, `{"error":"Invalid filename"}`, http.StatusBadRequest)
		return
	}
	filePath := filepath.Join(config.ApkDir, fileName)

	if err := os.Remove(filePath); err != nil {
		http.Error(w, `{"error":"Delete failed"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Deleted"})
}

// 辅助：从文件更新元数据
func updateMetadataFromFile(path, fileName string) {
	pkg, err := apk.OpenFile(path)
	if err != nil {
		return
	}
	defer pkg.Close()

	pkgName := pkg.PackageName()
	verCode := uint32(pkg.Manifest().VersionCode.MustInt32())
	verName := pkg.Manifest().VersionName.MustString()

	store.UpdateConfig(pkgName, func(c *models.AppConfig) {
		// 如果是更高版本，则更新 Latest 记录
		if verCode >= c.LatestVersionCode {
			c.LatestVersionCode = verCode
			c.LatestVersionName = verName
			c.LatestFileName = filepath.Base(fileName)
		}
	})
	store.SaveMetadata()
}
