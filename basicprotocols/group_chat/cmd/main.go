package main

import (
	"flag"
	"log"
	"manindexer/basicprotocols/group_chat"
	"os"
)

// @title Group Chat API
// @version 1.0
// @description 群聊服务 API 文档
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:8080
// @BasePath /

// @tag.name Group
// @tag.description 群组相关操作

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
	err := group_chat.RunWithConfig(config)
	if err != nil {
		log.Printf("Failed to start server: %v", err)
		os.Exit(1)
	}
}
