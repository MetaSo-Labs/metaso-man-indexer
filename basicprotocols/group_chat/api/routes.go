package api

import (
	"github.com/gin-gonic/gin"
)

// RegisterGroupRoutes 注册群组相关的路由
func RegisterGroupRoutes(router *gin.Engine) {
	// 群组相关路由组
	group := router.Group("/group-chat")
	{
		// 获取群组列表
		group.GET("/group-list", GetGroupList)

		// 获取用户的最新聊天群组列表
		group.GET("/user/latest-group-list", GetLatestChatGroupList)

		// 获取最新聊天信息列表（群聊+私聊）
		group.GET("/user/latest-chat-info-list", GetLatestChatInfoList)

		// 获取群组信息
		group.GET("/group-info", GetGroupInfo)

		// 获取群组聊天记录
		group.GET("/group-chat-list", GetGroupChatList)

		// 获取私聊记录
		group.GET("/private-chat-list", GetPrivateChatList)

		// 获取群组成员列表
		group.GET("/group-member-list", GetGroupMemberList)

		// 获取群组成员信息
		group.GET("/group-person", GetGroupPerson)
	}
}

// RegisterAllRoutes 注册所有路由
func RegisterAllRoutes(router *gin.Engine) {
	RegisterGroupRoutes(router)
	RegisterDbRoutes(router)
}
