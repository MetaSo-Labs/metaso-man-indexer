package group_chat

import (
	"log"
	"manindexer/adapter"
	"manindexer/basicprotocols/group_chat/api"
	"manindexer/basicprotocols/group_chat/indexer"
	"manindexer/basicprotocols/group_chat/service"
	"manindexer/basicprotocols/group_chat/service/socket_service"
	"manindexer/pin"
	"sync"

	"github.com/gin-gonic/gin"
)

var (
	groupChatIndexer *indexer.GroupChatIndexer
	initialized      bool
	mu               sync.Mutex
)

// Init initialize group_chat module
func Init(indexerChainAdapter map[string]adapter.Chain) error {
	mu.Lock()
	defer mu.Unlock()

	if initialized {
		return nil
	}

	log.Println("Initializing Group Chat module...")

	// 1. Create and initialize indexer
	var err error
	groupChatIndexer, err = indexer.NewGroupChatIndexer()
	if err != nil {
		log.Printf("Failed to create group chat indexer: %v", err)
		return err
	}

	// 2. Start indexer
	err = groupChatIndexer.Start()
	if err != nil {
		log.Printf("Failed to start group chat indexer: %v", err)
		return err
	}

	// 3. Initialize service (using indexer's database instance)
	err = service.InitService(groupChatIndexer, indexerChainAdapter)
	if err != nil {
		log.Printf("Failed to initialize service: %v", err)
		return err
	}

	// 4. Initialize Socket service
	err = socket_service.InitGroupChatSocketService()
	if err != nil {
		log.Printf("Failed to initialize Socket service: %v", err)
		return err
	}

	initialized = true
	log.Println("Group Chat module initialized successfully")
	return nil
}

// GetIndexer get indexer instance
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

// RegisterRoutes register API routes to Gin router
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

// ProcessPin process single Pin (external interface)
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

// Stop stop group_chat module
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

// IsInitialized check if module is initialized
func IsInitialized() bool {
	mu.Lock()
	defer mu.Unlock()
	return initialized
}

// GetServiceStats get service statistics
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

	// Can add more statistics
	// For example: queue status, database connection status, etc.

	return stats
}
