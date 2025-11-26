package store

import (
	"apkdepot/internal/config"
	"apkdepot/internal/models"
	"encoding/json"
	"log"
	"os"
	"sync"
)

var (
	// 内存缓存: PackageName -> AppConfig
	metadata     = make(map[string]*models.AppConfig)
	metadataLock sync.RWMutex
)

// LoadMetadata 从文件加载元数据
func LoadMetadata() {
	file, err := os.ReadFile(config.MetaFile)
	if err != nil {
		if os.IsNotExist(err) {
			log.Println("Metadata file not found, initializing empty.")
			return
		}
		log.Printf("Error reading metadata: %v", err)
		return
	}
	metadataLock.Lock()
	defer metadataLock.Unlock()
	if err := json.Unmarshal(file, &metadata); err != nil {
		log.Printf("Error parsing metadata: %v", err)
	}
}

// SaveMetadata 持久化元数据到文件
func SaveMetadata() {
	metadataLock.RLock()
	data, err := json.MarshalIndent(metadata, "", "  ")
	metadataLock.RUnlock()
	if err != nil {
		log.Printf("Error marshaling metadata: %v", err)
		return
	}
	if err := os.WriteFile(config.MetaFile, data, 0644); err != nil {
		log.Printf("Error writing metadata: %v", err)
	}
}

// GetAppConfig 线程安全获取配置
func GetAppConfig(packageName string) *models.AppConfig {
	metadataLock.RLock()
	defer metadataLock.RUnlock()
	// 返回指针的副本或直接返回指针（视是否需要保护内部字段而定）
	// 这里简单返回指针，调用方需注意不要直接修改，除非明确知道自己在做什么
	return metadata[packageName]
}

// UpdateConfig 线程安全更新或初始化配置
func UpdateConfig(packageName string, updateFn func(*models.AppConfig)) {
	metadataLock.Lock()
	defer metadataLock.Unlock()

	cfg, exists := metadata[packageName]
	if !exists {
		cfg = &models.AppConfig{
			PackageName: packageName,
			RolloutRate: 0, // 默认不发布
		}
		metadata[packageName] = cfg
	}
	// 执行更新回调
	updateFn(cfg)
}
