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

// ServerConfig server configuration
type ServerConfig struct {
	Port string
	Host string
}

// DefaultConfig default configuration
func DefaultConfig() *ServerConfig {
	return &ServerConfig{
		Port: common.Config.GroupChat.Port,
		Host: common.Config.GroupChat.Host,
	}
}

// Server group chat server
type Server struct {
	config *ServerConfig
	router *gin.Engine
}

// NewServer create new server instance
func NewServer(config *ServerConfig) *Server {
	if config == nil {
		config = DefaultConfig()
	}

	// Set Gin mode
	gin.SetMode(gin.DebugMode)

	router := gin.Default()

	// Add middleware
	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	router.Use(corsMiddleware())

	return &Server{
		config: config,
		router: router,
	}
}

// corsMiddleware CORS middleware
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

// SetupRoutes setup routes
func (s *Server) SetupRoutes() error {
	// Register group_chat routes
	err := RegisterRoutes(s.router)
	if err != nil {
		return err
	}

	// Setup Swagger for group chat module
	swagger.SetupSwagger(s.router)

	// Add health check route
	s.router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"service": "group-chat",
		})
	})

	// Add service status route
	s.router.GET("/status", func(c *gin.Context) {
		stats := GetServiceStats()
		c.JSON(http.StatusOK, gin.H{
			"service": "group-chat",
			"stats":   stats,
		})
	})

	// Add root route
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

// Start start server
func (s *Server) Start(indexerChainAdapter map[string]adapter.Chain) error {
	// Initialize group_chat module
	err := Init(indexerChainAdapter)
	if err != nil {
		log.Printf("Failed to initialize group chat module: %v", err)
		return err
	}

	// Setup routes
	err = s.SetupRoutes()
	if err != nil {
		log.Printf("Failed to setup routes: %v", err)
		return err
	}

	// Create HTTP server
	addr := s.config.Host + ":" + s.config.Port
	server := &http.Server{
		Addr:    addr,
		Handler: s.router,
	}

	// Graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan

		log.Println("Shutting down server...")

		// Stop group_chat module
		err := Stop()
		if err != nil {
			log.Printf("Failed to stop group chat module: %v", err)
		}

		// Close HTTP server
		if err := server.Close(); err != nil {
			log.Printf("Failed to close server: %v", err)
		}
	}()

	log.Printf("Starting Group Chat server on %s", addr)
	log.Printf("Swagger documentation available at: %s", swagger.GetSwaggerURL(s.config.Host, s.config.Port))
	log.Printf("Health check available at: http://%s/health", addr)

	return server.ListenAndServe()
}

// Run convenient method to run server
func Run(indexerChainAdapter map[string]adapter.Chain) error {
	config := DefaultConfig()
	server := NewServer(config)
	return server.Start(indexerChainAdapter)
}

// RunWithConfig run server with custom configuration
func RunWithConfig(config *ServerConfig, indexerChainAdapter map[string]adapter.Chain) error {
	server := NewServer(config)
	return server.Start(indexerChainAdapter)
}
