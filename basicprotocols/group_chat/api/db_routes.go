package api

import (
	"github.com/gin-gonic/gin"
)

// RegisterDbRoutes 注册数据库相关的路由
func RegisterDbRoutes(router *gin.Engine) {
	dbController := NewDbController()

	// 数据库查询API组
	dbGroup := router.Group("/api/db")
	{
		// 社区相关API
		communityGroup := dbGroup.Group("/community")
		{
			communityGroup.GET("/version", dbController.GetCommunityVersionInfo)
			communityGroup.GET("/info", dbController.GetCommunityInfo)
			communityGroup.GET("/join", dbController.GetCommunityJoin)
			communityGroup.GET("/person", dbController.GetCommunityPerson)
		}

		// 群组相关API
		groupGroup := dbGroup.Group("/group")
		{
			groupGroup.GET("/info", dbController.GetGroupInfo)
			groupGroup.GET("/version", dbController.GetGroupVersionInfo)
			groupGroup.GET("/version/all", dbController.GetAllGroupVersionInfo)
			groupGroup.GET("/join", dbController.GetGroupJoin)
			groupGroup.GET("/person", dbController.GetGroupPerson)
		}

		// 聊天相关API
		chatGroup := dbGroup.Group("/chat")
		{
			chatGroup.GET("/queue", dbController.GetChatQueue)
			chatGroup.GET("/pin", dbController.GetChatPin)
			chatGroup.GET("/pin/all", dbController.GetAllChatPin)
			chatGroup.GET("/timestamp", dbController.GetChatTimestamp)
		}

		// 用户相关API
		userGroup := dbGroup.Group("/user")
		{
			userGroup.GET("/context", dbController.GetUserContext)
		}

		// 统计相关API
		dbGroup.GET("/stats", dbController.GetDatabaseStats)
		dbGroup.GET("/collections", dbController.GetCollections)
	}
}
