package main

import (
	"flag"
	"log"
	"manindexer/basicprotocols/group_chat"
	"os"
)

// @title Group Chat API
// @version 1.0
// @description 群聊服务 API 文档，包含数据库查询、群组管理、社区管理等功能
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host 0.0.0.0:7568
// @BasePath /

// @tag.name 数据库查询
// @tag.description 数据库查询相关API，用于查看Pebble数据库中的数据

// @tag.name 群组管理
// @tag.description 群组管理相关API，包括群组信息、成员管理等

// @tag.name 社区管理
// @tag.description 社区管理相关API，包括社区信息、成员管理等

// @tag.name 聊天功能
// @tag.description 聊天功能相关API，包括消息、队列等

// @tag.name 用户管理
// @tag.description 用户管理相关API，包括用户信息、群列表等

func main() {
	// 解析命令行参数
	var (
		host = flag.String("host", "0.0.0.0", "服务器监听地址")
		port = flag.String("port", "8080", "服务器监听端口")
	)
	flag.Parse()

	// 设置日志格式
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("Starting Group Chat Service...")

	// 创建服务器配置
	config := &group_chat.ServerConfig{
		Host: *host,
		Port: *port,
	}

	// 运行服务器
	err := group_chat.RunWithConfig(config, nil)
	if err != nil {
		log.Printf("Failed to start server: %v", err)
		os.Exit(1)
	}
}
