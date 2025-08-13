package group_chat

import (
	"log"
	"manindexer/adapter"
	"manindexer/basicprotocols/group_chat/api"
	"manindexer/basicprotocols/group_chat/indexer"
	"manindexer/basicprotocols/group_chat/service"
	"manindexer/pin"
	"sync"

	"github.com/gin-gonic/gin"
)

var (
	groupChatIndexer *indexer.GroupChatIndexer
	initialized      bool
	mu               sync.Mutex
)

// Init 初始化 group_chat 模块
func Init(indexerChainAdapter map[string]adapter.Chain) error {
	mu.Lock()
	defer mu.Unlock()

	if initialized {
		return nil
	}

	log.Println("Initializing Group Chat module...")

	// 1. 创建并初始化索引器
	var err error
	groupChatIndexer, err = indexer.NewGroupChatIndexer()
	if err != nil {
		log.Printf("Failed to create group chat indexer: %v", err)
		return err
	}

	// 2. 启动索引器
	err = groupChatIndexer.Start()
	if err != nil {
		log.Printf("Failed to start group chat indexer: %v", err)
		return err
	}

	// 3. 初始化服务（使用索引器的数据库实例）
	err = service.InitService(groupChatIndexer, indexerChainAdapter)
	if err != nil {
		log.Printf("Failed to initialize service: %v", err)
		return err
	}

	initialized = true
	log.Println("Group Chat module initialized successfully")
	return nil
}

// GetIndexer 获取索引器实例
func GetIndexer() *indexer.GroupChatIndexer {
	if !initialized {
		log.Println("Warning: Group Chat module not initialized, calling Init()...")
		err := Init(nil)
		if err != nil {
			log.Printf("Failed to initialize Group Chat module: %v", err)
			return nil
		}
	}
	return groupChatIndexer
}

// RegisterRoutes 注册 API 路由到 Gin 路由
func RegisterRoutes(router *gin.Engine) error {
	if !initialized {
		err := Init(nil)
		if err != nil {
			return err
		}
	}

	api.RegisterAllRoutes(router)
	log.Println("Group Chat API routes registered successfully")
	return nil
}

// ProcessPin 处理单个 Pin（对外暴露的接口）
func ProcessGroupChatPin(pin *pin.PinInscription, tx interface{}) error {
	if !initialized {
		err := Init(nil)
		if err != nil {
			return err
		}
	}

	err := groupChatIndexer.ProcessPin(pin, tx)
	if err != nil {
		log.Printf("Failed to process group chat pin: %v", err)
	}

	return nil
}

// Stop 停止 group_chat 模块
func Stop() error {
	mu.Lock()
	defer mu.Unlock()

	if !initialized {
		return nil
	}

	log.Println("Stopping Group Chat module...")

	if groupChatIndexer != nil {
		err := groupChatIndexer.Stop()
		if err != nil {
			log.Printf("Failed to stop group chat indexer: %v", err)
			return err
		}
	}

	initialized = false
	log.Println("Group Chat module stopped successfully")
	return nil
}

// IsInitialized 检查模块是否已初始化
func IsInitialized() bool {
	mu.Lock()
	defer mu.Unlock()
	return initialized
}

// GetServiceStats 获取服务统计信息
func GetServiceStats() map[string]interface{} {
	if !initialized {
		return map[string]interface{}{
			"initialized": false,
			"error":       "Module not initialized",
		}
	}

	stats := map[string]interface{}{
		"initialized": true,
		"indexer":     "running",
	}

	// 可以添加更多统计信息
	// 例如：队列状态、数据库连接状态等

	return stats
}
