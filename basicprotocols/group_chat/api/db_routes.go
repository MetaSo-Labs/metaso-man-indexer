package api

import (
	"github.com/gin-gonic/gin"
)

// RegisterDbRoutes Register database-related routes
func RegisterDbRoutes(router *gin.Engine) {
	dbController := NewDbController()

	// Database query API group
	dbGroup := router.Group("/api/db")
	{
		// Community-related APIs
		communityGroup := dbGroup.Group("/community")
		{
			communityGroup.GET("/version", dbController.GetCommunityVersionInfo)
			communityGroup.GET("/info", dbController.GetCommunityInfo)
			communityGroup.GET("/join", dbController.GetCommunityJoin)
			communityGroup.GET("/person", dbController.GetCommunityPerson)
		}

		// Group-related APIs
		groupGroup := dbGroup.Group("/group")
		{
			groupGroup.GET("/info", dbController.GetGroupInfo)
			groupGroup.GET("/version", dbController.GetGroupVersionInfo)
			groupGroup.GET("/version/all", dbController.GetAllGroupVersionInfo)
			groupGroup.GET("/join", dbController.GetGroupJoin)
			groupGroup.GET("/person", dbController.GetGroupPerson)
		}

		// Chat-related APIs
		chatGroup := dbGroup.Group("/chat")
		{
			chatGroup.GET("/queue", dbController.GetChatQueue)
			chatGroup.GET("/pin", dbController.GetChatPin)
			chatGroup.GET("/pin/all", dbController.GetAllChatPin)
			chatGroup.GET("/timestamp", dbController.GetChatTimestamp)
		}

		// User-related APIs
		userGroup := dbGroup.Group("/user")
		{
			userGroup.GET("/context", dbController.GetUserContext)
		}

		// Statistics-related APIs
		dbGroup.GET("/stats", dbController.GetDatabaseStats)
		dbGroup.GET("/collections", dbController.GetCollections)
	}
}
