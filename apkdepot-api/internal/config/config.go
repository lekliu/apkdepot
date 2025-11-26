package config

import (
	"log"
	"os"
)

var (
	JwtSecret     []byte
	AdminUsername string
	AdminPassword string

	// 常量配置
	ApkDir        = "apks"
	MetaFile      = "metadata.json"
	MaxUploadSize = int64(1024 * 1024 * 500) // 500 MB
)

func Init() {
	JwtSecret = []byte(os.Getenv("JWT_SECRET"))
	AdminUsername = os.Getenv("ADMIN_USERNAME")
	AdminPassword = os.Getenv("ADMIN_PASSWORD")

	if len(JwtSecret) == 0 || AdminUsername == "" || AdminPassword == "" {
		log.Fatal("FATAL: JWT_SECRET, ADMIN_USERNAME, and ADMIN_PASSWORD environment variables must be set.")
	}

	if err := os.MkdirAll(ApkDir, 0755); err != nil {
		log.Fatalf("Failed to create APK directory: %v", err)
	}
}
