package socket_util

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/zishang520/engine.io/v2/types"
	"github.com/zishang520/socket.io/v2/socket"
)

// SocketManager Generic WebSocket connection manager
type SocketManager struct {
	server      *socket.Server
	connections sync.Map // 存储连接信息，key: metaid, value: *ConnectionInfo
	mutex       sync.RWMutex

	// Memory limit configuration
	maxConnections int
	maxMemoryMB    int
	currentMemory  int64

	// Cleanup configuration
	cleanupInterval time.Duration
	connectionTTL   time.Duration

	// Statistics
	stats *ConnectionStats

	// Singleton control
	initialized bool
	initMutex   sync.Mutex
}

// ConnectionInfo Connection information
type ConnectionInfo struct {
	SocketID    string
	MetaID      string
	ConnectTime time.Time
	LastActive  time.Time
	IsActive    bool
}

// ConnectionStats Connection statistics
type ConnectionStats struct {
	TotalConnections    int64
	ActiveConnections   int64
	TotalMessagesSent   int64
	TotalMessagesFailed int64
	mutex               sync.RWMutex
}

// SocketConfig Socket configuration
type SocketConfig struct {
	MaxConnections  int           // Maximum number of connections
	MaxMemoryMB     int           // Maximum memory usage (MB)
	CleanupInterval time.Duration // Cleanup interval
	ConnectionTTL   time.Duration // Connection time to live
	Port            int           // Service port
}

// DefaultConfig Default configuration
func DefaultConfig() *SocketConfig {
	return &SocketConfig{
		MaxConnections:  10000,
		MaxMemoryMB:     512,
		CleanupInterval: 5 * time.Minute,
		ConnectionTTL:   30 * time.Minute,
		Port:            7555,
	}
}

// Global singleton instance
var globalSocketManager *SocketManager
var globalOnce sync.Once

// GetSocketManager Get global Socket manager instance (singleton pattern)
func GetSocketManager() *SocketManager {
	if globalSocketManager == nil {
		return nil
	}
	return globalSocketManager
}

// InitSocketManager Initialize Socket manager (singleton pattern)
func InitSocketManager(config *SocketConfig) error {
	var initErr error
	globalOnce.Do(func() {
		if config == nil {
			config = DefaultConfig()
		}

		// Create Socket.IO server configuration
		c := socket.DefaultServerOptions()
		c.SetServeClient(true)
		c.SetPingInterval(300 * time.Millisecond)
		c.SetPingTimeout(200 * time.Millisecond)
		c.SetMaxHttpBufferSize(1000000)
		c.SetConnectTimeout(1000 * time.Millisecond)
		c.SetTransports(types.NewSet("polling", "websocket"))
		c.SetCors(&types.Cors{
			Origin:      "*",
			Credentials: true,
		})

		// Create Socket.IO server with configuration
		server := socket.NewServer(nil, nil)

		globalSocketManager = &SocketManager{
			server:          server,
			maxConnections:  config.MaxConnections,
			maxMemoryMB:     config.MaxMemoryMB,
			cleanupInterval: config.CleanupInterval,
			connectionTTL:   config.ConnectionTTL,
			stats:           &ConnectionStats{},
			initialized:     true,
		}

		// Setup auto-listeners for client connections
		globalSocketManager.setupAutoListeners()

		// Start cleanup routine
		go globalSocketManager.startCleanupRoutine()

		// Start the socket server in a goroutine to avoid blocking
		go func() {
			err := globalSocketManager.start(config.Port)
			if err != nil {
				log.Printf("Failed to start Socket server: %v", err)
				initErr = err
			}
		}()

		log.Printf("SocketManager initialized and started successfully, port: %d", config.Port)
	})

	return initErr
}

// setupAutoListeners Setup auto-listeners for client connections
func (sm *SocketManager) setupAutoListeners() {
	// Listen for client connection events
	sm.server.On("connection", func(clients ...interface{}) {
		client := clients[0].(*socket.Socket)
		sm.handleClientConnect(client)
	})

	log.Printf("Auto-listeners for client connections setup completed")
}

// handleClientConnect Handle client connection
func (sm *SocketManager) handleClientConnect(client *socket.Socket) {
	// Get metaid from handshake
	metaid := sm.getMetaIDFromSocket(client)
	if metaid == "" {
		log.Printf("Connection failed: missing metaid parameter, socket: %s", client.Id())
		client.Disconnect(true)
		return
	}

	// Check connection limit
	if sm.getConnectionCount() >= sm.maxConnections {
		log.Printf("Connection failed: connection limit reached, socket: %s", client.Id())
		client.Disconnect(true)
		return
	}

	// Check memory limit
	if sm.currentMemory >= int64(sm.maxMemoryMB*1024*1024) {
		log.Printf("Connection failed: memory limit reached, socket: %s", client.Id())
		client.Disconnect(true)
		return
	}

	// Automatically add connection to manager
	sm.addConnection(string(client.Id()), metaid)

	// Send connection success response
	response := &SocketData{
		M: WS_RESPONSE_SUCCESS,
		C: WS_CODE_SEND_SUCCESS,
		D: "Connection successful",
	}

	sm.sendMessage(client, response)
	log.Printf("Client connected successfully: socketID=%s, metaid=%s", client.Id(), metaid)

	// Listen for client messages
	client.On("message", func(args ...interface{}) {
		if len(args) > 0 {
			if msg, ok := args[0].(string); ok {
				sm.handleClientMessage(client, msg)
			}
		}
	})

	// Listen for client disconnection
	client.On("disconnect", func(args ...interface{}) {
		reason := "unknown"
		if len(args) > 0 {
			if r, ok := args[0].(string); ok {
				reason = r
			}
		}
		sm.handleClientDisconnect(client, reason)
	})

	// Listen for client ping
	client.On("ping", func(args ...interface{}) {
		sm.handleClientPing(client)
	})
}

// handleClientDisconnect Handle client disconnection
func (sm *SocketManager) handleClientDisconnect(client *socket.Socket, reason string) {
	log.Printf("Client disconnected: socketID=%s, reason: %s", client.Id(), reason)
	sm.removeConnection(string(client.Id()))
}

// handleClientMessage Handle client message
func (sm *SocketManager) handleClientMessage(client *socket.Socket, msg string) {
	// Update connection activity time
	sm.updateConnectionActivity(string(client.Id()))

	// Parse message
	socketData := SocketDataFromStringMsg(msg)
	if socketData == nil {
		log.Printf("Invalid message format: %s", msg)
		sm.sendError(client, "Invalid message format", WS_CODE_SEND_ERROR)
		return
	}

	// Handle heartbeat
	if socketData.M == HEART_BEAT {
		sm.handleHeartbeat(client, socketData)
		return
	}
	log.Printf("Received client message: method=%s, socketID=%s, data=%v", socketData.M, client.Id(), socketData.D)

}

// handleClientPing Handle client ping
func (sm *SocketManager) handleClientPing(client *socket.Socket) {
	// Update connection activity time
	sm.updateConnectionActivity(string(client.Id()))

	// Send pong response
	response := &SocketData{
		M: "pong",
		C: WS_CODE_SEND_SUCCESS,
	}

	sm.sendMessage(client, response)
}

// getMetaIDFromSocket Get metaid from socket
func (sm *SocketManager) getMetaIDFromSocket(client *socket.Socket) string {
	// Get metaid from handshake
	handshake := client.Handshake()
	if handshake == nil {
		return ""
	}

	// Try to get metaid from Auth
	if handshake.Auth != nil {
		if authMap, ok := handshake.Auth.(map[string]interface{}); ok {
			if metaid, exists := authMap["metaid"]; exists {
				if metaidStr, ok := metaid.(string); ok {
					return metaidStr
				}
			}
		}
	}

	// Try to get metaid from Query
	if handshake.Query != nil {
		if metaidValues, exists := handshake.Query["metaid"]; exists && len(metaidValues) > 0 {
			return metaidValues[0]
		}
	}

	return ""
}

// sendMessage Send message
func (sm *SocketManager) sendMessage(client *socket.Socket, socketData *SocketData) {
	msg, err := socketData.ToString()
	if err != nil {
		log.Printf("Message serialization failed: %v", err)
		return
	}

	err = client.Emit("message", msg)
	if err != nil {
		log.Printf("Failed to send message: %v", err)
		sm.stats.mutex.Lock()
		sm.stats.TotalMessagesFailed++
		sm.stats.mutex.Unlock()
	} else {
		sm.stats.mutex.Lock()
		sm.stats.TotalMessagesSent++
		sm.stats.mutex.Unlock()
	}
}

// sendError Send error message
func (sm *SocketManager) sendError(client *socket.Socket, message string, code int) {
	response := &SocketData{
		M: WS_RESPONSE_ERROR,
		C: code,
		D: message,
	}

	sm.sendMessage(client, response)
}

// handleHeartbeat Handle heartbeat event
func (sm *SocketManager) handleHeartbeat(client *socket.Socket, socketData *SocketData) {
	// Update connection activity time
	sm.updateConnectionActivity(string(client.Id()))

	// Send heartbeat response
	response := &SocketData{
		M: HEART_BEAT,
		C: WS_CODE_HEART_BEAT_BACK,
	}

	sm.sendMessage(client, response)
}

// getConnectionCount Get current connection count
func (sm *SocketManager) getConnectionCount() int {
	count := 0
	sm.connections.Range(func(key, value interface{}) bool {
		count++
		return true
	})
	return count
}

// startCleanupRoutine Start cleanup routine
func (sm *SocketManager) startCleanupRoutine() {
	ticker := time.NewTicker(sm.cleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		sm.cleanupInactiveConnections()
	}
}

// cleanupInactiveConnections Clean up inactive connections
func (sm *SocketManager) cleanupInactiveConnections() {
	now := time.Now()
	removedCount := 0

	sm.connections.Range(func(key, value interface{}) bool {
		connInfo := value.(*ConnectionInfo)

		// Check if connection has timed out
		if now.Sub(connInfo.LastActive) > sm.connectionTTL {
			sm.connections.Delete(key)
			removedCount++

			// Update statistics
			sm.stats.mutex.Lock()
			sm.stats.ActiveConnections--
			sm.stats.mutex.Unlock()
		}
		return true
	})

	if removedCount > 0 {
		log.Printf("Cleaned up %d inactive connections", removedCount)
	}
}

// Start Start Socket server
func (sm *SocketManager) start(port int) error {
	// Create HTTP server
	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	// Add CORS middleware
	router.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
		c.Header("Access-Control-Allow-Credentials", "true")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// Add Socket.IO routes
	handler := sm.server.ServeHandler(socket.DefaultServerOptions())
	router.POST("/socket.io/*f", gin.WrapH(handler))
	router.GET("/socket.io/*f", gin.WrapH(handler))

	log.Printf("Socket server started on port %d", port)
	return router.Run(fmt.Sprintf(":%d", port))
}

// GetStats Get statistics
func (sm *SocketManager) GetStats() *ConnectionStats {
	sm.stats.mutex.RLock()
	defer sm.stats.mutex.RUnlock()

	return &ConnectionStats{
		TotalConnections:    sm.stats.TotalConnections,
		ActiveConnections:   sm.stats.ActiveConnections,
		TotalMessagesSent:   sm.stats.TotalMessagesSent,
		TotalMessagesFailed: sm.stats.TotalMessagesFailed,
	}
}

// GetActiveConnections Get active connection count
func (sm *SocketManager) GetActiveConnections() int {
	count := 0
	sm.connections.Range(func(key, value interface{}) bool {
		connInfo := value.(*ConnectionInfo)
		if connInfo.IsActive {
			count++
		}
		return true
	})
	return count
}

// GetUserConnection Get user connection information
func (sm *SocketManager) GetUserConnection(metaid string) (*ConnectionInfo, bool) {
	value, exists := sm.connections.Load(metaid)
	if !exists {
		return nil, false
	}

	connInfo := value.(*ConnectionInfo)
	return connInfo, connInfo.IsActive
}

// AddConnection Add connection
func (sm *SocketManager) addConnection(socketID, metaid string) {
	connInfo := &ConnectionInfo{
		SocketID:    socketID,
		MetaID:      metaid,
		ConnectTime: time.Now(),
		LastActive:  time.Now(),
		IsActive:    true,
	}

	sm.connections.Store(metaid, connInfo)

	// Update statistics
	sm.stats.mutex.Lock()
	sm.stats.TotalConnections++
	sm.stats.ActiveConnections++
	sm.stats.mutex.Unlock()

	log.Printf("Added connection: socketID=%s, metaid=%s", socketID, metaid)
}

// RemoveConnection Remove connection
func (sm *SocketManager) removeConnection(socketID string) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	// Find and remove connection
	sm.connections.Range(func(key, value interface{}) bool {
		connInfo := value.(*ConnectionInfo)
		if connInfo.SocketID == socketID {
			sm.connections.Delete(key)

			// Update statistics
			sm.stats.mutex.Lock()
			sm.stats.ActiveConnections--
			sm.stats.mutex.Unlock()

			log.Printf("Removed connection: socketID=%s, metaid=%s", socketID, connInfo.MetaID)
			return false
		}
		return true
	})
}

// UpdateConnectionActivity Update connection activity time
func (sm *SocketManager) updateConnectionActivity(socketID string) {
	sm.connections.Range(func(key, value interface{}) bool {
		connInfo := value.(*ConnectionInfo)
		if connInfo.SocketID == socketID {
			connInfo.LastActive = time.Now()
			return false
		}
		return true
	})
}

// GetServer Get Socket.IO server instance
func (sm *SocketManager) GetServer() *socket.Server {
	return sm.server
}

// SendMessageToUser Server actively pushes message to specified user
// Used for server to send messages to client
func (sm *SocketManager) SendMessageToUser(metaid string, socketData *SocketData) error {
	connInfo, exists := sm.GetUserConnection(metaid)
	if !exists || !connInfo.IsActive {
		// return fmt.Errorf("user not connected: %s", metaid)
		return nil
	}

	// Find corresponding socket connection through socketID
	var targetSocket *socket.Socket
	sm.server.Sockets().Sockets().Range(func(socketID socket.SocketId, client *socket.Socket) bool {
		if string(socketID) == connInfo.SocketID {
			targetSocket = client
			return false // Stop iteration after finding
		}
		return true
	})

	if targetSocket == nil {
		return fmt.Errorf("socket connection not found: socketID=%s", connInfo.SocketID)
	}

	// Use sendMessage method to send message
	sm.sendMessage(targetSocket, socketData)

	// log.Printf("Sent message to user: metaid=%s, method=%s", metaid, socketData.M)
	return nil
}

// BroadcastMessage Server actively broadcasts message to all users
// Used for server to send messages to all clients
func (sm *SocketManager) BroadcastMessage(socketData *SocketData) error {
	// Iterate through all active connections and send messages
	sm.connections.Range(func(key, value interface{}) bool {
		connInfo := value.(*ConnectionInfo)
		if connInfo.IsActive {
			// Find corresponding socket connection through socketID
			sm.server.Sockets().Sockets().Range(func(socketID socket.SocketId, client *socket.Socket) bool {
				if string(socketID) == connInfo.SocketID {
					// Use sendMessage method to send message
					sm.sendMessage(client, socketData)
					return false // Stop iteration after finding
				}
				return true
			})
		}
		return true
	})

	log.Printf("Broadcast message: method=%s, sent to %d active connections", socketData.M, sm.GetActiveConnections())
	return nil
}
