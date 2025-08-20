package socket_service

import (
	"fmt"
	"log"
	"manindexer/common"
	"manindexer/common/socket_util"
	"time"
)

// InitGroupChatSocketService Initialize group chat Socket service
func InitGroupChatSocketService() error {
	if !common.Config.Socket.IsEnble {
		log.Printf("Socket service is disabled in configuration")
		return nil
	}
	port := common.Config.Socket.Port
	maxConnections := common.Config.Socket.MaxConnections
	maxMemoryMB := common.Config.Socket.MaxMemoryMB
	cleanupInterval := common.Config.Socket.CleanupInterval
	connectionTTL := common.Config.Socket.ConnectionTTL
	// // Parse configuration values
	// port, err := strconv.Atoi(common.Config.Socket.Port)
	// if err != nil {
	// 	log.Printf("Invalid socket port configuration: %v", err)
	// 	return err
	// }

	// maxConnections, err := strconv.Atoi(common.Config.Socket.MaxConnections)
	// if err != nil {
	// 	log.Printf("Invalid maxConnections configuration: %v", err)
	// 	return err
	// }

	// maxMemoryMB, err := strconv.Atoi(common.Config.Socket.MaxMemoryMB)
	// if err != nil {
	// 	log.Printf("Invalid maxMemoryMB configuration: %v", err)
	// 	return err
	// }

	// cleanupInterval, err := strconv.Atoi(common.Config.Socket.CleanupInterval)
	// if err != nil {
	// 	log.Printf("Invalid cleanupInterval configuration: %v", err)
	// 	return err
	// }

	// connectionTTL, err := strconv.Atoi(common.Config.Socket.ConnectionTTL)
	// if err != nil {
	// 	log.Printf("Invalid connectionTTL configuration: %v", err)
	// 	return err
	// }
	fmt.Println("[Socket]isEnble", common.Config.Socket.IsEnble)
	fmt.Println("[Socket]maxConnections", maxConnections)
	fmt.Println("[Socket]maxMemoryMB", maxMemoryMB)
	fmt.Println("[Socket]cleanupInterval", cleanupInterval)
	fmt.Println("[Socket]connectionTTL", connectionTTL)
	fmt.Println("[Socket]port", port)

	// Initialize Socket manager with configuration
	config := &socket_util.SocketConfig{
		MaxConnections:  int(maxConnections),                          // Maximum number of connections
		MaxMemoryMB:     int(maxMemoryMB),                             // Maximum memory usage (MB)
		CleanupInterval: time.Duration(cleanupInterval) * time.Minute, // Cleanup interval
		ConnectionTTL:   time.Duration(connectionTTL) * time.Minute,   // Connection time to live
		Port:            int(port),                                    // Socket service port
	}

	err := socket_util.InitSocketManager(config)
	if err != nil {
		log.Printf("Failed to initialize Socket manager: %v", err)
		return err
	}

	log.Printf("Group chat Socket service initialized successfully with port: %d", port)
	return nil
}

// SendGroupMessageToUser Send message to specified user (based on metaid)
func SendGroupMessageToUser(metaid string, message interface{}) error {
	socketManager := socket_util.GetSocketManager()
	if socketManager == nil {
		log.Printf("Socket manager not initialized")
		return nil
	}

	// Create message
	socketData := &socket_util.SocketData{
		M: socket_util.WS_SERVER_NOTIFY_GROUP_CHAT,
		C: socket_util.WS_CODE_SERVER,
		D: message,
	}

	// Send message to specified user
	err := socketManager.SendMessageToUser(metaid, socketData)
	if err != nil {
		log.Printf("Failed to send message to user: metaid=%s, error=%v", metaid, err)
		return err
	}

	log.Printf("Message sent successfully: metaid=%s", metaid)
	return nil
}

// SendPrivateMessageToUser Send private message to specified user (based on metaid)
func SendPrivateMessageToUser(metaid string, message interface{}) error {
	socketManager := socket_util.GetSocketManager()
	if socketManager == nil {
		log.Printf("Socket manager not initialized")
		return nil
	}

	// Create message
	socketData := &socket_util.SocketData{
		M: socket_util.WS_SERVER_NOTIFY_PRIVATE_CHAT,
		C: socket_util.WS_CODE_SERVER,
		D: message,
	}

	// Send message to specified user
	err := socketManager.SendMessageToUser(metaid, socketData)
	if err != nil {
		log.Printf("Failed to send private message to user: metaid=%s, error=%v", metaid, err)
		return err
	}

	log.Printf("Private message sent successfully: metaid=%s", metaid)
	return nil
}

// GetSocketManager Get Socket manager instance
func GetSocketManager() *socket_util.SocketManager {
	return socket_util.GetSocketManager()
}

// GetConnectionStats Get connection statistics
func GetConnectionStats() *socket_util.ConnectionStats {
	socketManager := socket_util.GetSocketManager()
	return socketManager.GetStats()
}

// IsUserOnline Check if user is online
func IsUserOnline(metaid string) bool {
	socketManager := socket_util.GetSocketManager()
	connInfo, exists := socketManager.GetUserConnection(metaid)
	return exists && connInfo.IsActive
}
