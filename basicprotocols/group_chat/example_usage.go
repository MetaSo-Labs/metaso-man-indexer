package group_chat

import (
	"log"
	"manindexer/pin"

	"github.com/gin-gonic/gin"
)

// ExampleUsage 展示如何使用 group_chat 模块
func ExampleUsage() {
	// 1. 初始化模块
	err := Init()
	if err != nil {
		log.Printf("Failed to initialize group chat module: %v", err)
		return
	}

	// 2. 创建 Gin 路由
	router := gin.Default()

	// 3. 注册 API 路由
	err = RegisterRoutes(router)
	if err != nil {
		log.Printf("Failed to register routes: %v", err)
		return
	}

	// 4. 启动 HTTP 服务器
	log.Println("Starting HTTP server on :8080...")
	err = router.Run(":8080")
	if err != nil {
		log.Printf("Failed to start server: %v", err)
		return
	}
}

// ExampleProcessPin 展示如何处理 Pin
func ExampleProcessPin() {
	// 初始化模块
	err := Init()
	if err != nil {
		log.Printf("Failed to initialize group chat module: %v", err)
		return
	}

	// 创建示例 Pin
	examplePin := &pin.PinInscription{
		Id:        "example_pin_id",
		Operation: "create",
		Path:      "/protocols/monitor-simple-group-chat",
		// 其他字段...
	}

	// 处理 Pin
	err = ProcessGroupChatPin(examplePin)
	if err != nil {
		log.Printf("Failed to process pin: %v", err)
		return
	}

	log.Println("Pin processed successfully")
}

// ExampleGetStats 展示如何获取服务统计信息
func ExampleGetStats() {
	stats := GetServiceStats()
	log.Printf("Service stats: %+v", stats)
}

// ExampleStop 展示如何停止模块
func ExampleStop() {
	err := Stop()
	if err != nil {
		log.Printf("Failed to stop group chat module: %v", err)
		return
	}

	log.Println("Group chat module stopped successfully")
}
