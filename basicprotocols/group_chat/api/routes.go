package api

import (
	"github.com/gin-gonic/gin"
)

// RegisterGroupRoutes Register group-related routes
func RegisterGroupRoutes(router *gin.Engine) {
	// Group-related route group
	group := router.Group("/group-chat")
	{
		// Get group list
		group.GET("/group-list", GetGroupList)

		// Get user's latest chat group list
		group.GET("/user/latest-group-list", GetLatestChatGroupList)

		// Get latest chat info list (group chat + private chat)
		group.GET("/user/latest-chat-info-list", GetLatestChatInfoList)

		// Get group info
		group.GET("/group-info", GetGroupInfo)

		// Get group chat records
		group.GET("/group-chat-list", GetGroupChatList)

		// Get private chat records
		group.GET("/private-chat-list", GetPrivateChatList)

		// Get group member list
		group.GET("/group-member-list", GetGroupMemberList)

		// Get group member info
		group.GET("/group-person", GetGroupPerson)

		// Lucky bag related routes
		// Get lucky bag info
		group.GET("/lucky-bag-info", GetLuckyBagInfo)

		// Get lucky bag unused info
		group.GET("/lucky-bag-unused-info", GetLuckyBagUnusedInfo)

		// Grab lucky bag
		group.POST("/grab-lucky-bag", GrabLuckyBag)

		// Reclaim lucky bag
		group.POST("/reclaim-lucky-bag", ReclaimLuckyBag)
	}
}

// RegisterSocketRoutes Register socket-related routes
func RegisterSocketRoutes(router *gin.Engine) {
	// Socket-related route group
	socket := router.Group("/group-chat/socket")
	{
		// Get connection statistics
		socket.GET("/stats", GetConnectionStats)

		// Check if user is online
		socket.GET("/user-online", IsUserOnline)
	}
}

// RegisterAllRoutes Register all routes
func RegisterAllRoutes(router *gin.Engine) {
	RegisterGroupRoutes(router)
	RegisterDbRoutes(router)
	RegisterSocketRoutes(router)
}
