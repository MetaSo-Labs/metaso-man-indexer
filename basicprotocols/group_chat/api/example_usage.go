package api

import (
	"log"
	"manindexer/basicprotocols/group_chat/indexer"
	"manindexer/basicprotocols/group_chat/service"

	"github.com/gin-gonic/gin"
)

// ExampleSetup 展示如何设置和使用 group_chat API
func ExampleSetup() {
	// 1. 创建并初始化索引器
	groupChatIndexer, err := indexer.NewGroupChatIndexer()
	if err != nil {
		log.Fatalf("Failed to create group chat indexer: %v", err)
	}

	// 2. 启动索引器
	err = groupChatIndexer.Start()
	if err != nil {
		log.Fatalf("Failed to start group chat indexer: %v", err)
	}

	// 3. 初始化服务（使用索引器的数据库实例）
	err = service.InitService(groupChatIndexer, nil)
	if err != nil {
		log.Fatalf("Failed to initialize service: %v", err)
	}

	// 4. 创建 Gin 路由
	router := gin.Default()

	// 5. 注册 API 路由
	RegisterAllRoutes(router)

	// 6. 启动服务器
	log.Println("Starting group chat API server on :8080")
	err = router.Run(":8080")
	if err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

// ExampleAPIEndpoints 展示 API 端点的使用示例
func ExampleAPIEndpoints() {
	// API 端点示例：

	// 1. 获取群组列表
	// GET /group-chat/group-list?metaId=user123&cursor=1&size=20&timestamp=1234567890

	// 2. 获取用户的最新聊天群组列表
	// GET /group-chat/user/latest-group-list?metaId=user123&cursor=1&size=20&timestamp=1234567890

	// 3. 获取群组信息
	// GET /group-chat/group-info?groupId=group123

	// 4. 获取群组聊天记录
	// GET /group-chat/group-chat-list?groupId=group123&metaId=user123&cursor=1&size=20&timestamp=1234567890

	// 5. 获取群组成员列表
	// GET /group-chat/group-member-list?groupId=group123&cursor=1&size=20&timestamp=1234567890
}
