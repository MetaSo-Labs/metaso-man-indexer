package group_chat

import (
	"log"
	"manindexer/adapter"
	"manindexer/basicprotocols/group_chat/api/swagger"
	"manindexer/common"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"
)

// ServerConfig 服务器配置
type ServerConfig struct {
	Port string
	Host string
}

// DefaultConfig 默认配置
func DefaultConfig() *ServerConfig {
	return &ServerConfig{
		Port: common.Config.GroupChat.Port,
		Host: common.Config.GroupChat.Host,
	}
}

// Server 群聊服务器
type Server struct {
	config *ServerConfig
	router *gin.Engine
}

// NewServer 创建新的服务器实例
func NewServer(config *ServerConfig) *Server {
	if config == nil {
		config = DefaultConfig()
	}

	// 设置 Gin 模式
	gin.SetMode(gin.DebugMode)

	router := gin.Default()

	// 添加中间件
	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	router.Use(corsMiddleware())

	return &Server{
		config: config,
		router: router,
	}
}

// corsMiddleware CORS 中间件
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// SetupRoutes 设置路由
func (s *Server) SetupRoutes() error {
	// 注册 group_chat 路由
	err := RegisterRoutes(s.router)
	if err != nil {
		return err
	}

	// 设置群聊模块的Swagger
	swagger.SetupSwagger(s.router)

	// 添加健康检查路由
	s.router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"service": "group-chat",
		})
	})

	// 添加服务状态路由
	s.router.GET("/status", func(c *gin.Context) {
		stats := GetServiceStats()
		c.JSON(http.StatusOK, gin.H{
			"service": "group-chat",
			"stats":   stats,
		})
	})

	// 添加根路由
	s.router.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"service": "group-chat",
			"version": "1.0.0",
			"docs":    "/group-chat/docs/index.html",
			"health":  "/health",
			"status":  "/status",
		})
	})

	return nil
}

// Start 启动服务器
func (s *Server) Start(indexerChainAdapter map[string]adapter.Chain) error {
	// 初始化 group_chat 模块
	err := Init(indexerChainAdapter)
	if err != nil {
		log.Printf("Failed to initialize group chat module: %v", err)
		return err
	}

	// 设置路由
	err = s.SetupRoutes()
	if err != nil {
		log.Printf("Failed to setup routes: %v", err)
		return err
	}

	// 创建 HTTP 服务器
	addr := s.config.Host + ":" + s.config.Port
	server := &http.Server{
		Addr:    addr,
		Handler: s.router,
	}

	// 优雅关闭
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan

		log.Println("Shutting down server...")

		// 停止 group_chat 模块
		err := Stop()
		if err != nil {
			log.Printf("Failed to stop group chat module: %v", err)
		}

		// 关闭 HTTP 服务器
		if err := server.Close(); err != nil {
			log.Printf("Failed to close server: %v", err)
		}
	}()

	log.Printf("Starting Group Chat server on %s", addr)
	log.Printf("Swagger documentation available at: %s", swagger.GetSwaggerURL(s.config.Host, s.config.Port))
	log.Printf("Health check available at: http://%s/health", addr)

	return server.ListenAndServe()
}

// Run 运行服务器的便捷方法
func Run(indexerChainAdapter map[string]adapter.Chain) error {
	config := DefaultConfig()
	server := NewServer(config)
	return server.Start(indexerChainAdapter)
}

// RunWithConfig 使用自定义配置运行服务器
func RunWithConfig(config *ServerConfig, indexerChainAdapter map[string]adapter.Chain) error {
	server := NewServer(config)
	return server.Start(indexerChainAdapter)
}
