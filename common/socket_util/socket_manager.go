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
	connections sync.Map // store connection info, key: metaid, value: *UserConnections
	mutex       sync.RWMutex

	extraPushAuthKey string
	extraConnection  *ConnectionInfo

	// Memory limit configuration
	maxConnections int
	maxMemoryMB    int

	// Cleanup configuration
	cleanupInterval time.Duration
	connectionTTL   time.Duration

	// Statistics
	stats *ConnectionStats

	// Singleton control
	initialized bool
	initMutex   sync.Mutex
}

const (
	EXTRA_PUSH_SERVICE_METAID = "extra_push_service"
	DEVICE_TYPE_PC            = "pc"
	DEVICE_TYPE_APP           = "app"
	MAX_DEVICES_PER_USER      = 2
)

// ConnectionInfo Connection information
type ConnectionInfo struct {
	SocketID    string
	MetaID      string
	DeviceType  string // "pc" or "app"
	ConnectTime time.Time
	LastActive  time.Time
	IsActive    bool
}

// UserConnections User's device connections
type UserConnections struct {
	MetaID  string
	Devices []*ConnectionInfo // Maximum 2 devices: pc and app
}

// ConnectionStats Connection statistics
type ConnectionStats struct {
	TotalConnections      int64
	ActiveConnections     int64
	TotalUserConnections  int64
	ActiveUserConnections int64
	TotalMessagesSent     int64
	TotalMessagesFailed   int64
	TotalMemoryUsage      int64   // Total memory usage in bytes
	AverageMemoryPerConn  int64   // Average memory per connection in bytes
	TotalMemoryMB         float64 // Total memory usage in MB
	AverageMemoryKB       float64 // Average memory per connection in KB
	MemoryUsagePercent    float64 // Memory usage percentage
	MemoryLimitMB         int     // Memory limit in MB
	mutex                 sync.RWMutex
}

// SocketConfig Socket configuration
type SocketConfig struct {
	MaxConnections   int           // Maximum number of connections
	MaxMemoryMB      int           // Maximum memory usage (MB)
	CleanupInterval  time.Duration // Cleanup interval
	ConnectionTTL    time.Duration // Connection time to live
	Port             int           // Service port
	ExtraPushAuthKey string        // Extra push auth key
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

		// Create Socket.IO server configuration with optimized settings
		c := socket.DefaultServerOptions()
		c.SetServeClient(true)
		// Increase ping intervals to reduce connection pressure
		c.SetPingInterval(2 * time.Second) // Increased from 300ms
		c.SetPingTimeout(5 * time.Second)  // Increased from 200ms
		c.SetMaxHttpBufferSize(1000000)
		c.SetConnectTimeout(10 * time.Second) // Increased from 1s
		// Add connection timeout and cleanup settings
		c.SetUpgradeTimeout(10 * time.Second) // Add upgrade timeout
		c.SetMaxHttpBufferSize(1000000)       // Keep existing buffer size
		c.SetTransports(types.NewSet("polling", "websocket"))
		c.SetCors(&types.Cors{
			Origin:      "*",
			Credentials: true,
		})

		// Add additional Engine.IO configuration for stability
		c.SetAllowEIO3(true) // Allow Engine.IO v3 compatibility

		// Create Socket.IO server with configuration
		server := socket.NewServer(nil, nil)

		globalSocketManager = &SocketManager{
			server:           server,
			maxConnections:   config.MaxConnections,
			maxMemoryMB:      config.MaxMemoryMB,
			cleanupInterval:  config.CleanupInterval,
			connectionTTL:    config.ConnectionTTL,
			stats:            &ConnectionStats{},
			initialized:      true,
			extraPushAuthKey: config.ExtraPushAuthKey,
			extraConnection:  nil,
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
	// Listen for client connection events with panic recovery
	sm.server.On("connection", func(clients ...interface{}) {
		// Add panic recovery for connection handling
		defer func() {
			if r := recover(); r != nil {
				log.Printf("Panic recovered in connection handler: %v", r)
				// Log stack trace for debugging
				log.Printf("Stack trace: %+v", r)
			}
		}()

		if len(clients) == 0 {
			log.Printf("No client provided in connection event")
			return
		}

		client, ok := clients[0].(*socket.Socket)
		if !ok {
			log.Printf("Invalid client type in connection event: %T", clients[0])
			return
		}

		sm.handleClientConnect(client)
	})

	log.Printf("Auto-listeners for client connections setup completed")
}

// handleClientConnect Handle client connection
func (sm *SocketManager) handleClientConnect(client *socket.Socket) {
	// Add nil pointer check to prevent panic
	if client == nil {
		log.Printf("handleClientConnect: client is nil, skipping connection handling")
		return
	}

	// Add panic recovery for client operations
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Panic recovered in handleClientConnect: %v", r)
		}
	}()

	if sm.extraPushAuthKey != "" {
		// Check if this is an extra push connection
		extraAuthKey := sm.getExtraPushAuthKeyFromSocket(client)
		if extraAuthKey != "" && extraAuthKey == sm.extraPushAuthKey {
			// This is an extra push connection, initialize extraConnection
			sm.extraConnection = &ConnectionInfo{
				SocketID:    string(client.Id()),
				MetaID:      EXTRA_PUSH_SERVICE_METAID,
				ConnectTime: time.Now(),
				LastActive:  time.Now(),
				IsActive:    true,
			}

			log.Printf("Extra push connection established: socketID=%s", client.Id())

			// Send connection success response
			response := &SocketData{
				M: WS_RESPONSE_SUCCESS,
				C: WS_CODE_SEND_SUCCESS,
				D: "Extra push connection successful",
			}
			sm.sendMessage(client, response)

			// Setup listeners for extra push connection
			sm.setupExtraPushListeners(client)
			return
		}
	}

	// Get metaid and device type from handshake
	metaid, deviceType := sm.getMetaIDAndDeviceTypeFromSocket(client)
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
	stats := sm.GetStats()
	if stats.TotalMemoryUsage >= int64(sm.maxMemoryMB*1024*1024) {
		log.Printf("Connection failed: memory limit reached (%.2f MB / %d MB), socket: %s",
			float64(stats.TotalMemoryUsage)/(1024*1024), sm.maxMemoryMB, client.Id())
		client.Disconnect(true)
		return
	}

	// Automatically add device connection to manager
	sm.addDeviceConnection(metaid, deviceType, string(client.Id()))

	// Send connection success response
	response := &SocketData{
		M: WS_RESPONSE_SUCCESS,
		C: WS_CODE_SEND_SUCCESS,
		D: "Connection successful",
	}

	sm.sendMessage(client, response)
	log.Printf("[SOCKET] Client connected successfully: socketID=%s, metaid=%s, deviceType=%s", client.Id(), metaid, deviceType)

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
	// Add nil pointer check to prevent panic
	if client == nil {
		log.Printf("[SOCKET] handleClientDisconnect: client is nil, reason: %s", reason)
		return
	}

	// Add panic recovery for client.Id()
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Panic recovered in handleClientDisconnect: %v", r)
		}
	}()

	log.Printf("[SOCKET] Client disconnected: socketID=%s, reason: %s", client.Id(), reason)
	sm.removeDeviceConnection(string(client.Id()))
}

// handleClientMessage Handle client message
func (sm *SocketManager) handleClientMessage(client *socket.Socket, msg string) {
	// Add nil pointer check to prevent panic
	if client == nil {
		log.Printf("handleClientMessage: client is nil, skipping message handling")
		return
	}

	// Add panic recovery for client.Id()
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Panic recovered in handleClientMessage: %v", r)
		}
	}()

	// Update connection activity time
	sm.updateDeviceConnectionActivity(string(client.Id()))

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
	// Add nil pointer check to prevent panic
	if client == nil {
		log.Printf("handleClientPing: client is nil, skipping ping handling")
		return
	}

	// Add panic recovery for client.Id()
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Panic recovered in handleClientPing: %v", r)
		}
	}()

	// Update connection activity time
	sm.updateDeviceConnectionActivity(string(client.Id()))

	// Send pong response
	response := &SocketData{
		M: "pong",
		C: WS_CODE_SEND_SUCCESS,
	}

	sm.sendMessage(client, response)
}

// getMetaIDAndDeviceTypeFromSocket Get metaid and device type from socket handshake
func (sm *SocketManager) getMetaIDAndDeviceTypeFromSocket(client *socket.Socket) (string, string) {
	// Add nil pointer check to prevent panic
	if client == nil {
		log.Printf("getMetaIDAndDeviceTypeFromSocket: client is nil, returning defaults")
		return "", DEVICE_TYPE_PC // Default to PC
	}

	// Add panic recovery for client.Handshake()
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Panic recovered in getMetaIDAndDeviceTypeFromSocket: %v", r)
		}
	}()

	handshake := client.Handshake()
	if handshake == nil {
		return "", DEVICE_TYPE_PC // Default to PC
	}

	metaid := ""
	deviceType := DEVICE_TYPE_PC // Default to PC

	// Try to get from Auth first
	if handshake.Auth != nil {
		if authMap, ok := handshake.Auth.(map[string]interface{}); ok {
			// Get metaid
			if metaidVal, exists := authMap["metaid"]; exists {
				if metaidStr, ok := metaidVal.(string); ok {
					metaid = metaidStr
				}
			}
			// Get device type
			if deviceTypeVal, exists := authMap["type"]; exists {
				if deviceTypeStr, ok := deviceTypeVal.(string); ok {
					if deviceTypeStr == DEVICE_TYPE_APP {
						deviceType = DEVICE_TYPE_APP
					}
				}
			}
		}
	}

	// Try to get from Query if not found in Auth
	if handshake.Query != nil {
		// Get metaid
		if metaid == "" {
			if metaidValues, exists := handshake.Query["metaid"]; exists && len(metaidValues) > 0 {
				metaid = metaidValues[0]
			}
		}
		// Get device type
		if deviceType == DEVICE_TYPE_PC {
			if deviceTypeValues, exists := handshake.Query["type"]; exists && len(deviceTypeValues) > 0 {
				if deviceTypeValues[0] == DEVICE_TYPE_APP {
					deviceType = DEVICE_TYPE_APP
				}
			}
		}
	}

	return metaid, deviceType
}

// getExtraPushAuthKeyFromSocket Get extraPushAuthKey from socket
func (sm *SocketManager) getExtraPushAuthKeyFromSocket(client *socket.Socket) string {
	// Add nil pointer check to prevent panic
	if client == nil {
		log.Printf("getExtraPushAuthKeyFromSocket: client is nil, returning empty string")
		return ""
	}

	// Add panic recovery for client.Handshake()
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Panic recovered in getExtraPushAuthKeyFromSocket: %v", r)
		}
	}()

	// Get extraPushAuthKey from handshake
	handshake := client.Handshake()
	if handshake == nil {
		return ""
	}

	// Try to get extraPushAuthKey from Auth
	if handshake.Auth != nil {
		if authMap, ok := handshake.Auth.(map[string]interface{}); ok {
			if extraAuthKey, exists := authMap["extraPushAuthKey"]; exists {
				if extraAuthKeyStr, ok := extraAuthKey.(string); ok {
					return extraAuthKeyStr
				}
			}
		}
	}

	// Try to get extraPushAuthKey from Query
	if handshake.Query != nil {
		if extraAuthKeyValues, exists := handshake.Query["extraPushAuthKey"]; exists && len(extraAuthKeyValues) > 0 {
			return extraAuthKeyValues[0]
		}
	}

	return ""
}

// setupExtraPushListeners Setup listeners for extra push connection
func (sm *SocketManager) setupExtraPushListeners(client *socket.Socket) {
	// Listen for client messages
	client.On("message", func(args ...interface{}) {
		if len(args) > 0 {
			if msg, ok := args[0].(string); ok {
				sm.handleExtraPushMessage(client, msg)
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
		sm.handleExtraPushDisconnect(client, reason)
	})

	// Listen for client ping
	client.On("ping", func(args ...interface{}) {
		sm.handleExtraPushPing(client)
	})
}

// handleExtraPushMessage Handle extra push message
func (sm *SocketManager) handleExtraPushMessage(client *socket.Socket, msg string) {
	// Add nil pointer check to prevent panic
	if client == nil {
		log.Printf("handleExtraPushMessage: client is nil, skipping message handling")
		return
	}

	// Add panic recovery for client.Id()
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Panic recovered in handleExtraPushMessage: %v", r)
		}
	}()

	// Update connection activity time
	if sm.extraConnection != nil {
		sm.extraConnection.LastActive = time.Now()
	}

	// Parse message
	socketData := SocketDataFromStringMsg(msg)
	if socketData == nil {
		log.Printf("Invalid extra push message format: %s", msg)
		sm.sendError(client, "Invalid message format", WS_CODE_SEND_ERROR)
		return
	}

	// Handle heartbeat
	if socketData.M == HEART_BEAT {
		sm.handleExtraPushHeartbeat(client, socketData)
		return
	}

	log.Printf("Received extra push message: method=%s, socketID=%s, data=%v", socketData.M, client.Id(), socketData.D)
}

// handleExtraPushDisconnect Handle extra push disconnection
func (sm *SocketManager) handleExtraPushDisconnect(client *socket.Socket, reason string) {
	// Add nil pointer check to prevent panic
	if client == nil {
		log.Printf("handleExtraPushDisconnect: client is nil, reason: %s", reason)
		sm.extraConnection = nil
		return
	}

	// Add panic recovery for client.Id()
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Panic recovered in handleExtraPushDisconnect: %v", r)
		}
	}()

	log.Printf("Extra push connection disconnected: socketID=%s, reason: %s", client.Id(), reason)
	sm.extraConnection = nil
}

// handleExtraPushPing Handle extra push ping
func (sm *SocketManager) handleExtraPushPing(client *socket.Socket) {
	// Add nil pointer check to prevent panic
	if client == nil {
		log.Printf("handleExtraPushPing: client is nil, skipping ping handling")
		return
	}

	// Add panic recovery for client operations
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Panic recovered in handleExtraPushPing: %v", r)
		}
	}()

	// Update connection activity time
	if sm.extraConnection != nil {
		sm.extraConnection.LastActive = time.Now()
	}

	// Send pong response
	response := &SocketData{
		M: "pong",
		C: WS_CODE_SEND_SUCCESS,
	}

	sm.sendMessage(client, response)
}

// handleExtraPushHeartbeat Handle extra push heartbeat
func (sm *SocketManager) handleExtraPushHeartbeat(client *socket.Socket, socketData *SocketData) {
	// Add nil pointer check to prevent panic
	if client == nil {
		log.Printf("handleExtraPushHeartbeat: client is nil, skipping heartbeat handling")
		return
	}

	// Add panic recovery for client operations
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Panic recovered in handleExtraPushHeartbeat: %v", r)
		}
	}()

	// Update connection activity time
	if sm.extraConnection != nil {
		sm.extraConnection.LastActive = time.Now()
	}

	// Send heartbeat response
	response := &SocketData{
		M: HEART_BEAT,
		C: WS_CODE_HEART_BEAT_BACK,
	}

	sm.sendMessage(client, response)
}

// sendMessage Send message
func (sm *SocketManager) sendMessage(client *socket.Socket, socketData *SocketData) {
	// Add nil pointer checks to prevent panic
	if client == nil {
		log.Printf("sendMessage: client is nil, skipping message send")
		return
	}

	if socketData == nil {
		log.Printf("sendMessage: socketData is nil, skipping message send")
		return
	}

	msg, err := socketData.ToString()
	if err != nil {
		log.Printf("Message serialization failed: %v", err)
		return
	}

	// Add panic recovery for client.Emit
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Panic recovered in sendMessage: %v", r)
			sm.stats.mutex.Lock()
			sm.stats.TotalMessagesFailed++
			sm.stats.mutex.Unlock()
		}
	}()

	// Check if client is still connected before sending
	// Note: Engine.IO doesn't have a direct way to check connection status,
	// so we rely on the error handling below
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
	// Add nil pointer check to prevent panic
	if client == nil {
		log.Printf("handleHeartbeat: client is nil, skipping heartbeat handling")
		return
	}

	// Add panic recovery for client.Id()
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Panic recovered in handleHeartbeat: %v", r)
		}
	}()

	// Update connection activity time
	sm.updateDeviceConnectionActivity(string(client.Id()))

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
		userConn := value.(*UserConnections)
		count += len(userConn.Devices)
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
		userConn := value.(*UserConnections)
		var devicesToRemove []int

		// Check each device for timeout
		for i, device := range userConn.Devices {
			if now.Sub(device.LastActive) > sm.connectionTTL {
				devicesToRemove = append(devicesToRemove, i)
				removedCount++
			}
		}

		// Remove timed out devices (from back to front to maintain indices)
		for i := len(devicesToRemove) - 1; i >= 0; i-- {
			index := devicesToRemove[i]
			userConn.Devices = append(userConn.Devices[:index], userConn.Devices[index+1:]...)
		}

		// If no devices left, remove user connection
		if len(userConn.Devices) == 0 {
			sm.connections.Delete(key)
		}

		return true
	})

	// Update statistics and memory after cleanup
	if removedCount > 0 {
		sm.stats.mutex.Lock()
		sm.stats.ActiveConnections -= int64(removedCount)
		sm.stats.mutex.Unlock()

		sm.updateMemoryStats()
		log.Printf("[SOCKET] Cleaned up %d inactive device connections", removedCount)
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

	// Calculate user connection counts
	totalUsers, activeUsers := sm.GetUserConnectionCounts()
	totalUserConnections := int64(totalUsers)
	activeUserConnections := int64(activeUsers)

	// Calculate formatted memory values
	totalMemoryMB := float64(sm.stats.TotalMemoryUsage) / (1024 * 1024)
	averageMemoryKB := float64(sm.stats.AverageMemoryPerConn) / 1024
	memoryUsagePercent := float64(sm.stats.TotalMemoryUsage) / float64(sm.maxMemoryMB*1024*1024) * 100

	// Round to 6 decimal places
	totalMemoryMB = float64(int64(totalMemoryMB*1000000)) / 1000000
	averageMemoryKB = float64(int64(averageMemoryKB*1000000)) / 1000000
	memoryUsagePercent = float64(int64(memoryUsagePercent*1000000)) / 1000000

	return &ConnectionStats{
		TotalConnections:      sm.stats.TotalConnections,
		ActiveConnections:     sm.stats.ActiveConnections,
		TotalUserConnections:  totalUserConnections,
		ActiveUserConnections: activeUserConnections,
		TotalMessagesSent:     sm.stats.TotalMessagesSent,
		TotalMessagesFailed:   sm.stats.TotalMessagesFailed,
		TotalMemoryUsage:      sm.stats.TotalMemoryUsage,
		AverageMemoryPerConn:  sm.stats.AverageMemoryPerConn,
		TotalMemoryMB:         totalMemoryMB,
		AverageMemoryKB:       averageMemoryKB,
		MemoryUsagePercent:    memoryUsagePercent,
		MemoryLimitMB:         sm.maxMemoryMB,
	}
}

// GetActiveConnections Get active connection count
func (sm *SocketManager) GetActiveConnections() int {
	count := 0
	sm.connections.Range(func(key, value interface{}) bool {
		userConn := value.(*UserConnections)
		for _, device := range userConn.Devices {
			if device.IsActive {
				count++
			}
		}
		return true
	})
	return count
}

// GetUserConnection Get user connection information (returns first active device)
func (sm *SocketManager) GetUserConnection(metaid string) (*ConnectionInfo, bool) {
	value, exists := sm.connections.Load(metaid)
	if !exists {
		return nil, false
	}

	userConn := value.(*UserConnections)
	for _, device := range userConn.Devices {
		if device.IsActive {
			return device, true
		}
	}
	return nil, false
}

// GetUserConnectionByDevice Get user connection by device type
func (sm *SocketManager) GetUserConnectionByDevice(metaid, deviceType string) (*ConnectionInfo, bool) {
	value, exists := sm.connections.Load(metaid)
	if !exists {
		return nil, false
	}

	userConn := value.(*UserConnections)
	for _, device := range userConn.Devices {
		if device.DeviceType == deviceType && device.IsActive {
			return device, true
		}
	}
	return nil, false
}

// GetUserAllConnections Get all active connections for a user
func (sm *SocketManager) GetUserAllConnections(metaid string) ([]*ConnectionInfo, bool) {
	value, exists := sm.connections.Load(metaid)
	if !exists {
		return nil, false
	}

	userConn := value.(*UserConnections)
	var activeDevices []*ConnectionInfo
	for _, device := range userConn.Devices {
		if device.IsActive {
			activeDevices = append(activeDevices, device)
		}
	}
	return activeDevices, len(activeDevices) > 0
}

// IsUserOnline Check if user is online (any device)
func (sm *SocketManager) IsUserOnline(metaid string) bool {
	value, exists := sm.connections.Load(metaid)
	if !exists {
		return false
	}

	userConn := value.(*UserConnections)
	for _, device := range userConn.Devices {
		if device.IsActive {
			return true
		}
	}
	return false
}

// GetUserConnectionCounts Get user connection counts
func (sm *SocketManager) GetUserConnectionCounts() (int, int) {
	totalUsers := 0
	activeUsers := 0

	sm.connections.Range(func(key, value interface{}) bool {
		userConn := value.(*UserConnections)
		totalUsers++
		// Check if user has any active devices
		for _, device := range userConn.Devices {
			if device.IsActive {
				activeUsers++
				break // Count user as active if any device is active
			}
		}
		return true
	})

	return totalUsers, activeUsers
}

// calculateConnectionMemory Calculate memory usage for a single connection
func (sm *SocketManager) calculateConnectionMemory(connInfo *ConnectionInfo) int64 {
	// Estimate memory usage for connection info
	// This is a rough estimation based on typical Go struct sizes
	memoryUsage := int64(0)

	// String fields: SocketID, MetaID, DeviceType
	memoryUsage += int64(len(connInfo.SocketID))
	memoryUsage += int64(len(connInfo.MetaID))
	memoryUsage += int64(len(connInfo.DeviceType))

	// Time fields: ConnectTime, LastActive (typically 24 bytes each)
	memoryUsage += 48

	// Boolean field: IsActive (1 byte)
	memoryUsage += 1

	// Struct overhead and alignment (rough estimate)
	memoryUsage += 32

	// Additional overhead for sync.Map storage
	memoryUsage += 64

	return memoryUsage
}

// updateMemoryStats Update memory statistics
func (sm *SocketManager) updateMemoryStats() {
	sm.stats.mutex.Lock()
	defer sm.stats.mutex.Unlock()

	totalMemory := int64(0)
	activeConnections := int64(0)

	sm.connections.Range(func(key, value interface{}) bool {
		userConn := value.(*UserConnections)
		for _, device := range userConn.Devices {
			if device.IsActive {
				totalMemory += sm.calculateConnectionMemory(device)
				activeConnections++
			}
		}
		return true
	})

	sm.stats.TotalMemoryUsage = totalMemory
	if activeConnections > 0 {
		sm.stats.AverageMemoryPerConn = totalMemory / activeConnections
	} else {
		sm.stats.AverageMemoryPerConn = 0
	}
}

// GetServer Get Socket.IO server instance
func (sm *SocketManager) GetServer() *socket.Server {
	return sm.server
}

// RefreshMemoryStats Force refresh memory statistics
func (sm *SocketManager) RefreshMemoryStats() {
	sm.updateMemoryStats()
	stats := sm.GetStats()
	log.Printf("Memory statistics refreshed: TotalMemory=%.6f MB, AveragePerConn=%.6f KB, Usage=%.6f%%",
		stats.TotalMemoryMB, stats.AverageMemoryKB, stats.MemoryUsagePercent)
}

// SendMessageToUser Server actively pushes message to specified user (first active device)
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

	log.Printf("Sent message to user: metaid=%s, deviceType=%s, method=%s", metaid, connInfo.DeviceType, socketData.M)
	return nil
}

// SendMessageToUserByDevice Server actively pushes message to specified user's device
func (sm *SocketManager) SendMessageToUserByDevice(metaid, deviceType string, socketData *SocketData) error {
	connInfo, exists := sm.GetUserConnectionByDevice(metaid, deviceType)
	if !exists || !connInfo.IsActive {
		// return fmt.Errorf("user device not connected: %s, %s", metaid, deviceType)
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

	log.Printf("Sent message to user device: metaid=%s, deviceType=%s, method=%s", metaid, deviceType, socketData.M)
	return nil
}

// SendMessageToUserAllDevices Server actively pushes message to all user's devices
func (sm *SocketManager) SendMessageToUserAllDevices(metaid string, socketData *SocketData) error {
	devices, exists := sm.GetUserAllConnections(metaid)
	if !exists {
		// return fmt.Errorf("user not connected: %s", metaid)
		return nil
	}

	successCount := 0
	for _, device := range devices {
		// Find corresponding socket connection through socketID
		var targetSocket *socket.Socket
		sm.server.Sockets().Sockets().Range(func(socketID socket.SocketId, client *socket.Socket) bool {
			if string(socketID) == device.SocketID {
				targetSocket = client
				return false // Stop iteration after finding
			}
			return true
		})

		if targetSocket != nil {
			sm.sendMessage(targetSocket, socketData)
			successCount++
		}
	}

	log.Printf("Sent message to user all devices: metaid=%s, method=%s, successCount=%d/%d",
		metaid, socketData.M, successCount, len(devices))
	return nil
}

// BroadcastMessage Server actively broadcasts message to all users
// Used for server to send messages to all clients
func (sm *SocketManager) BroadcastMessage(socketData *SocketData) error {
	// Iterate through all active connections and send messages
	sm.connections.Range(func(key, value interface{}) bool {
		userConn := value.(*UserConnections)
		for _, device := range userConn.Devices {
			if device.IsActive {
				// Find corresponding socket connection through socketID
				sm.server.Sockets().Sockets().Range(func(socketID socket.SocketId, client *socket.Socket) bool {
					if string(socketID) == device.SocketID {
						// Use sendMessage method to send message
						sm.sendMessage(client, socketData)
						return false // Stop iteration after finding
					}
					return true
				})
			}
		}
		return true
	})

	log.Printf("Broadcast message: method=%s, sent to %d active connections", socketData.M, sm.GetActiveConnections())
	return nil
}

// SendMessageToExtraPush Send message to extra push connection
func (sm *SocketManager) SendMessageToExtraPush(socketData *SocketData) error {
	if sm.extraConnection == nil || !sm.extraConnection.IsActive {
		return fmt.Errorf("extra push connection not available")
	}

	// Find corresponding socket connection through socketID
	var targetSocket *socket.Socket
	sm.server.Sockets().Sockets().Range(func(socketID socket.SocketId, client *socket.Socket) bool {
		if string(socketID) == sm.extraConnection.SocketID {
			targetSocket = client
			return false // Stop iteration after finding
		}
		return true
	})

	if targetSocket == nil {
		return fmt.Errorf("extra push socket connection not found: socketID=%s", sm.extraConnection.SocketID)
	}

	// Use sendMessage method to send message
	sm.sendMessage(targetSocket, socketData)

	log.Printf("Sent message to extra push service: method=%s", socketData.M)
	return nil
}

// Helper methods for device connection management

// findDeviceConnection Find device connection by device type
func (sm *SocketManager) findDeviceConnection(userConn *UserConnections, deviceType string) (*ConnectionInfo, int) {
	for i, device := range userConn.Devices {
		if device.DeviceType == deviceType {
			return device, i
		}
	}
	return nil, -1
}

// addDeviceConnection Add device connection for user
func (sm *SocketManager) addDeviceConnection(metaid, deviceType, socketID string) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	if value, exists := sm.connections.Load(metaid); exists {
		userConn := value.(*UserConnections)

		// Find if device type already exists
		if existingDevice, index := sm.findDeviceConnection(userConn, deviceType); existingDevice != nil {
			// Replace existing device of same type
			userConn.Devices[index] = &ConnectionInfo{
				SocketID:    socketID,
				MetaID:      metaid,
				DeviceType:  deviceType,
				ConnectTime: time.Now(),
				LastActive:  time.Now(),
				IsActive:    true,
			}
			log.Printf("[SOCKET] Replaced device connection: metaid=%s, deviceType=%s, socketID=%s", metaid, deviceType, socketID)
		} else if len(userConn.Devices) < MAX_DEVICES_PER_USER {
			// Add new device (within limit)
			userConn.Devices = append(userConn.Devices, &ConnectionInfo{
				SocketID:    socketID,
				MetaID:      metaid,
				DeviceType:  deviceType,
				ConnectTime: time.Now(),
				LastActive:  time.Now(),
				IsActive:    true,
			})
			log.Printf("[SOCKET] Added device connection: metaid=%s, deviceType=%s, socketID=%s", metaid, deviceType, socketID)
		} else {
			log.Printf("[SOCKET] Device connection limit reached for user: metaid=%s", metaid)
		}
	} else {
		// Create new user connections
		userConn := &UserConnections{
			MetaID: metaid,
			Devices: []*ConnectionInfo{
				{
					SocketID:    socketID,
					MetaID:      metaid,
					DeviceType:  deviceType,
					ConnectTime: time.Now(),
					LastActive:  time.Now(),
					IsActive:    true,
				},
			},
		}
		sm.connections.Store(metaid, userConn)
		log.Printf("[SOCKET] Created new user connection: metaid=%s, deviceType=%s, socketID=%s", metaid, deviceType, socketID)
	}

	// Update statistics
	sm.stats.mutex.Lock()
	sm.stats.TotalConnections++
	sm.stats.ActiveConnections++
	sm.stats.mutex.Unlock()

	// Update memory statistics
	sm.updateMemoryStats()
}

// removeDeviceConnection Remove device connection for user
func (sm *SocketManager) removeDeviceConnection(socketID string) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	// Find and remove connection
	sm.connections.Range(func(key, value interface{}) bool {
		userConn := value.(*UserConnections)

		for i, device := range userConn.Devices {
			if device.SocketID == socketID {
				// Remove device from slice
				userConn.Devices = append(userConn.Devices[:i], userConn.Devices[i+1:]...)

				// If no devices left, remove user connection
				if len(userConn.Devices) == 0 {
					sm.connections.Delete(key)
				}

				// Update statistics
				sm.stats.mutex.Lock()
				sm.stats.ActiveConnections--
				sm.stats.mutex.Unlock()

				// Update memory statistics
				sm.updateMemoryStats()

				log.Printf("[SOCKET] Removed device connection: socketID=%s, metaid=%s, deviceType=%s", socketID, device.MetaID, device.DeviceType)
				return false
			}
		}
		return true
	})
}

// updateDeviceConnectionActivity Update device connection activity time
func (sm *SocketManager) updateDeviceConnectionActivity(socketID string) {
	sm.connections.Range(func(key, value interface{}) bool {
		userConn := value.(*UserConnections)

		for _, device := range userConn.Devices {
			if device.SocketID == socketID {
				device.LastActive = time.Now()
				return false
			}
		}
		return true
	})
}
