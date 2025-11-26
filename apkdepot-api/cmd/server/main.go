package main

import (
	"apkdepot/internal/config"
	"apkdepot/internal/handlers"
	"apkdepot/internal/store"
	"log"
	"net/http"
	"os"

	"github.com/rs/cors"
)

func main() {
	// 1. 初始化配置与存储
	config.Init()
	store.LoadMetadata()

	mux := http.NewServeMux()

	// 2. 注册公开路由
	mux.HandleFunc("/api/apks", handlers.ListApks)
	mux.HandleFunc("/api/login", handlers.Login)
	mux.HandleFunc("/api/check-update", handlers.CheckUpdate) // APP 自动检查
	mux.HandleFunc("/api/version-list", handlers.VersionList) // APP 手动列表

	// 静态文件服务 (APK 下载)
	fs := http.FileServer(http.Dir(config.ApkDir))
	mux.Handle("/apks/", http.StripPrefix("/apks/", fs))

	// 3. 注册受保护路由 (需鉴权)
	// 上传 APK
	mux.Handle("/api/upload", handlers.AuthMiddleware(http.HandlerFunc(handlers.UploadApk)))
	// 删除 APK
	mux.Handle("/api/apks/", handlers.AuthMiddleware(http.HandlerFunc(handlers.DeleteApk)))
	// 修改发布策略 (灰度/强制更新)
	mux.Handle("/api/config/update", handlers.AuthMiddleware(http.HandlerFunc(handlers.UpdateConfig)))

	// 4. CORS 配置
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"}, // 生产环境建议指定具体域名
		AllowedMethods:   []string{"GET", "POST", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
	})
	handler := c.Handler(mux)

	// 5. 启动服务
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("🚀 ApkDepot Server is running on :%s", port)
	if err := http.ListenAndServe(":"+port, handler); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
