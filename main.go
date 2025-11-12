package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"github.com/shogo82148/androidbinary/apk"
	"html/template"
	"image/png" // 用于将解码后的图标编码为 PNG
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
)

const (
	apkDir        = "apks"
	templateDir   = "templates"
	staticDir     = "static"
	maxUploadSize = 1024 * 1024 * 500 // 500 MB
)

// ApkInfo 持有关于 APK 文件的元数据
type ApkInfo struct {
	FileName    string
	AppName     string
	PackageName string
	VersionName string
	VersionCode uint32
	FileSize    string
	IconBase64  string
	UploadTime  string
}

// 预加载所有 HTML 模板
var templates = template.Must(template.ParseGlob(filepath.Join(templateDir, "*.html")))

func main() {
	// 确保 APK 存储目录存在
	if err := os.MkdirAll(apkDir, 0755); err != nil {
		log.Fatalf("Failed to create APK directory: %v", err)
	}

	mux := http.NewServeMux()

	// 提供静态文件服务 (CSS)
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir(staticDir))))

	// 提供 APK 文件下载服务
	mux.Handle("/apks/", http.StripPrefix("/apks/", http.FileServer(http.Dir(apkDir))))

	// 注册应用路由
	mux.HandleFunc("/", listApksHandler)
	mux.HandleFunc("/upload", uploadApkHandler)

	// 从环境变量获取端口，否则使用默认值 8080
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Starting ApkDepot server on :%s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

// listApksHandler 处理主页请求，显示所有 APK 列表
func listApksHandler(w http.ResponseWriter, r *http.Request) {
	files, err := os.ReadDir(apkDir)
	if err != nil {
		http.Error(w, "Could not read APK directory", http.StatusInternalServerError)
		return
	}

	var apks []ApkInfo
	for _, file := range files {
		// 跳过目录和非 .apk 文件
		if file.IsDir() || filepath.Ext(file.Name()) != ".apk" {
			continue
		}

		filePath := filepath.Join(apkDir, file.Name())
		// 使用第三方库解析 APK 文件
		pkg, err := apk.OpenFile(filePath)
		if err != nil {
			log.Printf("Warning: Could not parse %s: %v", file.Name(), err)
			continue
		}
		defer pkg.Close()

		fileInfo, _ := file.Info()

		// 提取应用图标并编码为 Base64
		icon, err := pkg.Icon(nil)
		var iconB64 string
		if err == nil && icon != nil {
			var buf bytes.Buffer
			if err := png.Encode(&buf, icon); err == nil {
				iconB64 = base64.StdEncoding.EncodeToString(buf.Bytes())
			}
		}

		// 填充 APK 信息结构体
		appInfo := ApkInfo{
			FileName:    file.Name(),
			AppName:     pkg.Manifest().App.Label.MustString(),
			PackageName: pkg.PackageName(),
			VersionName: pkg.Manifest().VersionName.MustString(),
			VersionCode: uint32(pkg.Manifest().VersionCode.MustInt32()),
			FileSize:    formatFileSize(fileInfo.Size()),
			IconBase64:  iconB64,
			UploadTime:  fileInfo.ModTime().Format("2006-01-02 15:04:05"),
		}
		apks = append(apks, appInfo)
	}

	// 按应用名称排序
	sort.Slice(apks, func(i, j int) bool {
		// String comparison works here because of the YYYY-MM-DD HH:MM:SS format
		return apks[i].UploadTime > apks[j].UploadTime
	})

	// 渲染 index.html 模板并传入 APK 数据
	if err := templates.ExecuteTemplate(w, "index.html", apks); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// uploadApkHandler 处理文件上传页面和上传逻辑
func uploadApkHandler(w http.ResponseWriter, r *http.Request) {
	// 如果是 GET 请求，显示上传表单
	if r.Method == "GET" {
		if err := templates.ExecuteTemplate(w, "upload.html", nil); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	// 如果是 POST 请求，处理文件上传
	if r.Method == "POST" {
		r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)
		if err := r.ParseMultipartForm(maxUploadSize); err != nil {
			http.Error(w, "The uploaded file is too big. Max size is 500MB.", http.StatusBadRequest)
			return
		}

		file, handler, err := r.FormFile("apkfile")
		if err != nil {
			http.Error(w, "Invalid file", http.StatusBadRequest)
			return
		}
		defer file.Close()

		// 验证文件扩展名
		if filepath.Ext(handler.Filename) != ".apk" {
			http.Error(w, "Invalid file type. Only .apk files are allowed.", http.StatusBadRequest)
			return
		}

		// 创建目标文件（如果已存在则覆盖）
		dstPath := filepath.Join(apkDir, filepath.Base(handler.Filename))
		dst, err := os.Create(dstPath)
		if err != nil {
			http.Error(w, "Could not save file", http.StatusInternalServerError)
			return
		}
		defer dst.Close()

		// 将上传的文件内容复制到目标文件
		if _, err := io.Copy(dst, file); err != nil {
			http.Error(w, "Could not save file", http.StatusInternalServerError)
			return
		}

		// 上传成功后，重定向到主页
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

// formatFileSize 将文件大小（字节）格式化为更易读的字符串
func formatFileSize(size int64) string {
	if size < 1024 {
		return strconv.FormatInt(size, 10) + " B"
	}
	kb := float64(size) / 1024
	if kb < 1024 {
		return fmt.Sprintf("%.2f KB", kb)
	}
	mb := kb / 1024
	return fmt.Sprintf("%.2f MB", mb)
}
