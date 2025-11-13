package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"github.com/rs/cors"
	"github.com/shogo82148/androidbinary/apk"
	"image/png"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// --- 配置 (从环境变量读取) ---
var (
	JWT_SECRET     = []byte(os.Getenv("JWT_SECRET"))
	ADMIN_USERNAME = os.Getenv("ADMIN_USERNAME")
	ADMIN_PASSWORD = os.Getenv("ADMIN_PASSWORD")
)

const (
	apkDir        = "apks"
	maxUploadSize = 1024 * 1024 * 500 // 500 MB
)

// ApkInfo 持有关于 APK 文件的元数据
type ApkInfo struct {
	FileName    string `json:"fileName"` // 4. (新增) 添加 JSON 标签
	AppName     string `json:"appName"`
	PackageName string `json:"packageName"`
	VersionName string `json:"versionName"`
	VersionCode uint32 `json:"versionCode"`
	FileSize    string `json:"fileSize"`
	IconBase64  string `json:"iconBase64"`
	UploadTime  string `json:"uploadTime"`
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func main() {
	// 检查关键环境变量是否设置
	if string(JWT_SECRET) == "" || ADMIN_USERNAME == "" || ADMIN_PASSWORD == "" {
		log.Fatal("FATAL: JWT_SECRET, ADMIN_USERNAME, and ADMIN_PASSWORD environment variables must be set.")
	}

	if err := os.MkdirAll(apkDir, 0755); err != nil {
		log.Fatalf("Failed to create APK directory: %v", err)
	}

	mux := http.NewServeMux()

	// 公开路由 (无需登录)
	mux.HandleFunc("/api/apks", listApksHandler) // GET
	mux.HandleFunc("/api/login", loginHandler)   // POST
	mux.Handle("/apks/", http.StripPrefix("/apks/", http.FileServer(http.Dir(apkDir))))

	// 受保护的路由 (需要管理员权限)
	mux.Handle("/api/upload", authMiddleware(http.HandlerFunc(uploadApkHandler))) // POST
	mux.Handle("/api/apks/", authMiddleware(http.HandlerFunc(deleteApkHandler)))  // DELETE

	c := cors.New(cors.Options{
		AllowedOrigins: []string{"*"}, // 生产环境应设为前端域名
		AllowedMethods: []string{"GET", "POST", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Content-Type", "Authorization"},
	})
	handler := c.Handler(mux)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Starting ApkDepot API server on :%s", port)
	// 8. (修改) 使用带有 CORS 的 handler 启动服务
	if err := http.ListenAndServe(":"+port, handler); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"Invalid request body"}`, http.StatusBadRequest)
		return
	}

	if req.Username == ADMIN_USERNAME && req.Password == ADMIN_PASSWORD {
		// 凭证正确，生成 JWT
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"username": req.Username,
			"exp":      time.Now().Add(time.Hour * 24).Unix(), // Token 有效期 24 小时
		})

		tokenString, err := token.SignedString(JWT_SECRET)
		if err != nil {
			http.Error(w, `{"error":"Could not generate token"}`, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"token": tokenString})
	} else {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid credentials"})
	}
}

func authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, `{"error":"Authorization header required"}`, http.StatusUnauthorized)
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			http.Error(w, `{"error":"Bearer token required"}`, http.StatusUnauthorized)
			return
		}

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return JWT_SECRET, nil
		})

		if err != nil || !token.Valid {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"error": "Invalid or expired token"})
			return
		}

		next.ServeHTTP(w, r)
	})
}

// --- 业务 Handler ---

func deleteApkHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	fileName := filepath.Base(r.URL.Path)
	if fileName == "." || fileName == "/" || strings.Contains(fileName, "..") {
		http.Error(w, `{"error":"Invalid filename"}`, http.StatusBadRequest)
		return
	}

	filePath := filepath.Join(apkDir, fileName)

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		http.Error(w, `{"error":"File not found"}`, http.StatusNotFound)
		return
	}

	if err := os.Remove(filePath); err != nil {
		http.Error(w, `{"error":"Failed to delete file"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "File deleted successfully"})
}

// listApksHandler 处理主页请求，现在返回 JSON
func listApksHandler(w http.ResponseWriter, r *http.Request) {
	files, err := os.ReadDir(apkDir)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Could not read APK directory"})
		return
	}

	var apks []ApkInfo
	for _, file := range files {
		if file.IsDir() || filepath.Ext(file.Name()) != ".apk" {
			continue
		}

		filePath := filepath.Join(apkDir, file.Name())
		pkg, err := apk.OpenFile(filePath)
		if err != nil {
			log.Printf("Warning: Could not parse %s: %v", file.Name(), err)
			continue
		}

		fileInfo, _ := file.Info()

		icon, err := pkg.Icon(nil)
		var iconB64 string
		if err == nil && icon != nil {
			var buf bytes.Buffer
			if err := png.Encode(&buf, icon); err == nil {
				iconB64 = base64.StdEncoding.EncodeToString(buf.Bytes())
			}
		}

		// 9. (保留) 使用您提供的、可以正常工作的 APK 解析逻辑
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
		pkg.Close() // 及时关闭文件句柄
	}

	sort.Slice(apks, func(i, j int) bool {
		return apks[i].UploadTime > apks[j].UploadTime
	})

	// 10. (修改) 设置响应头为 JSON 并返回数据
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(apks)
}

// uploadApkHandler 处理文件上传，现在返回 JSON
func uploadApkHandler(w http.ResponseWriter, r *http.Request) {
	// 预检请求，直接返回成功
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method == http.MethodPost {
		r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)
		if err := r.ParseMultipartForm(maxUploadSize); err != nil {
			http.Error(w, `{"error":"The uploaded file is too big."}`, http.StatusBadRequest)
			return
		}

		file, handler, err := r.FormFile("apkfile")
		if err != nil {
			http.Error(w, `{"error":"Invalid file."}`, http.StatusBadRequest)
			return
		}
		defer file.Close()

		if filepath.Ext(handler.Filename) != ".apk" {
			http.Error(w, `{"error":"Invalid file type. Only .apk files are allowed."}`, http.StatusBadRequest)
			return
		}

		dstPath := filepath.Join(apkDir, filepath.Base(handler.Filename))
		dst, err := os.Create(dstPath)
		if err != nil {
			http.Error(w, `{"error":"Could not save file."}`, http.StatusInternalServerError)
			return
		}
		defer dst.Close()

		if _, err := io.Copy(dst, file); err != nil {
			http.Error(w, `{"error":"Could not write file."}`, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"message": "Upload successful"})
	}
}

// formatFileSize 函数保持不变
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
