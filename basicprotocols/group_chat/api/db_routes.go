package api

import (
	"github.com/gin-gonic/gin"
)

// RegisterDbRoutes Register database-related routes
func RegisterDbRoutes(router *gin.Engine) {
	// Database query API group
	dbGroup := router.Group("/api/db")
	{
		// Community-related APIs
		communityGroup := dbGroup.Group("/community")
		{
			communityGroup.GET("/version", GetCommunityVersionInfo)
			communityGroup.GET("/info", GetCommunityInfo)
			communityGroup.GET("/join", GetCommunityJoin)
			communityGroup.GET("/person", GetCommunityPerson)
		}

		// Group-related APIs
		groupGroup := dbGroup.Group("/group")
		{
			groupGroup.GET("/info", GetDbGroupInfo)
			groupGroup.GET("/version", GetGroupVersionInfo)
			groupGroup.GET("/version/all", GetAllGroupVersionInfo)
			groupGroup.GET("/join", GetGroupJoin)
			groupGroup.GET("/person", GetDbGroupPerson)
		}

		// Chat-related APIs
		chatGroup := dbGroup.Group("/chat")
		{
			chatGroup.GET("/queue", GetChatQueue)
			chatGroup.GET("/pin", GetChatPin)
			chatGroup.GET("/pin/all", GetAllChatPin)
			chatGroup.GET("/timestamp", GetChatTimestamp)
		}

		// User-related APIs
		userGroup := dbGroup.Group("/user")
		{
			userGroup.GET("/context", GetUserContext)
		}

		// Statistics-related APIs
		dbGroup.GET("/stats", GetDatabaseStats)
		dbGroup.GET("/collections", GetCollections)

		// Lucky bag-related APIs
		luckyBagGroup := dbGroup.Group("/luckybag")
		{
			luckyBagGroup.GET("/statistics", GetLuckyBagStatistics)
		}

		// Migration-related APIs
		migrationGroup := dbGroup.Group("/migration")
		{
			migrationGroup.GET("/info", GetMigrationInfo)
		}
	}
}
