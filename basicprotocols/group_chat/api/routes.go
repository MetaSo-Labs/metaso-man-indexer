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
		// Get group chat records (new format)
		group.GET("/group-chat-list-v2", GetGroupChatListV2)
		// Get group chat records (test version with IterOptions)
		group.GET("/group-chat-list-v3", GetGroupChatListV3)

		// Get group chat records by index range (ascending order)
		group.GET("/group-chat-list-by-index", GetGroupChatListByIndex)

		// Get group chat records by start timestamp range (ascending order)
		group.GET("/group-chat-list-by-start-time", GetGroupChatListByStartTime)

		// Get channel chat records (V3 with improved performance)
		group.GET("/channel-chat-list-v3", GetChannelChatListV3)

		// Get channel chat records by index range (ascending order)
		group.GET("/channel-chat-list-by-index", GetChannelChatListByIndex)

		// Get channel chat records by start timestamp range (ascending order)
		group.GET("/channel-chat-list-by-start-time", GetChannelChatListByStartTime)

		// Get group channel list
		group.GET("/group-channel-list", GetGroupChannelList)

		// Get private chat records
		group.GET("/private-chat-list", GetPrivateChatList)

		// Get private chat records by index range
		group.GET("/private-chat-list-by-index", GetPrivateChatListByIndex)

		// Get group member list
		group.GET("/group-member-list", GetGroupMemberList)

		// Get group member info
		group.GET("/group-person", GetGroupPerson)

		// Get group user role info
		group.GET("/group-user-role", GetGroupUserRoleInfo)

		// Check if sync is completed
		group.GET("/sync-completed", IsSyncCompleted)

		// Get user info by address
		group.GET("/user-info", GetUserInfoByAddress)

		// Get batch user info by addresses or metaIds
		group.POST("/batch-user-info", GetBatchUserInfo)

		// Get current maximum chat index
		group.GET("/max-group-chat-index", GetCurrentMaxGroupChatIndex)
		group.GET("/max-group-channel-chat-index", GetCurrentMaxGroupChannelChatIndex)
		group.GET("/max-private-chat-index", GetCurrentMaxPrivateChatIndex)

		// Group search routes
		group.GET("/search-groups", SearchGroups)
		group.GET("/search-groups-and-users", SearchGroupsAndUsers)
		group.GET("/search-groups-cache-stats", GetGroupSearchCacheStats)

		// Search group members
		group.GET("/search-group-members", SearchGroupMembers)

		// Check if chat is sendable
		group.GET("/chat-sendable", CheckChatSendable)

		// Lucky bag related routes
		// Get lucky bag info
		group.GET("/lucky-bag-info", GetLuckyBagInfo)

		// Get lucky bag unused info
		group.GET("/lucky-bag-unused-info", GetLuckyBagUnusedInfo)

		// Grab lucky bag
		group.POST("/grab-lucky-bag", GrabLuckyBag)

		// Reclaim lucky bag
		group.POST("/reclaim-lucky-bag", ReclaimLuckyBag)

		// Generate lucky bag code address key
		group.GET("/generate-lucky-bag-code", GenerateLuckyBagCodeAddressKey)

		// Lucky bag V2 routes (with cache optimization)
		// Get lucky bag info V2
		group.GET("/lucky-bag-info-v2", GetLuckyBagInfoV2)

		// Get lucky bag unused info V2
		group.GET("/lucky-bag-unused-info-v2", GetLuckyBagUnusedInfoV2)

		// Grab lucky bag V2
		group.POST("/grab-lucky-bag-v2", GrabLuckyBagV2)
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

// RegisterSyncRoutes Register synchronization-related routes
func RegisterSyncRoutes(router *gin.Engine) {
	// Sync management API group
	syncGroup := router.Group("/group-chat/sync")
	{
		// Sync operations
		syncGroup.POST("/pins-by-time-range", SyncPinsByTimeRange)

		// Block query operations
		syncGroup.POST("/block-height-by-timestamp", GetBlockHeightByTimestamp)

		// Sync status and control
		syncGroup.GET("/stats", GetSyncStats)
		syncGroup.GET("/status", GetSyncStatus)
		syncGroup.POST("/stop", StopSync)
	}
}

// RegisterAllRoutes Register all routes
func RegisterAllRoutes(router *gin.Engine) {
	RegisterGroupRoutes(router)
	RegisterDbRoutes(router)
	RegisterSocketRoutes(router)
	RegisterSyncRoutes(router)
}
