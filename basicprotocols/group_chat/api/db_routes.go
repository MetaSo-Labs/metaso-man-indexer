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
			groupGroup.GET("/member-list", GetGroupMemberList)
			groupGroup.GET("/member-list-v2", GetGroupMemberListV2)
			groupGroup.GET("/person-list-collection", GetGroupPersonListCollection)

			// Group admin, block, and whitelist APIs
			groupGroup.GET("/admin-collection", GetGroupAdminCollection)
			groupGroup.GET("/block-collection", GetGroupBlockCollection)
			groupGroup.GET("/whitelist-collection", GetGroupWhitelistCollection)
			groupGroup.GET("/admin/:groupId", GetGroupAdminByGroupId)
			groupGroup.GET("/block/:groupId", GetGroupBlockByGroupId)
			groupGroup.GET("/whitelist/:groupId", GetGroupWhitelistByGroupId)
		}

		// Chat-related APIs
		chatGroup := dbGroup.Group("/chat")
		{
			chatGroup.GET("/queue", GetChatQueue)
			chatGroup.GET("/pin", GetChatPin)
			chatGroup.GET("/pin/all", GetAllChatPin)
			chatGroup.GET("/timestamp", GetChatTimestamp)
			chatGroup.GET("/timestamp2/out", GetGroupChatTimestamp2OutList)
			chatGroup.GET("/timestamp2/out/channel", GetGroupChannelChatTimestampOutList)

			// Chat index-related APIs
			indexGroup := chatGroup.Group("/index")
			{
				indexGroup.GET("/group", GetGroupChatIndexList)
				indexGroup.GET("/group/keys", GetGroupChatIndexKeys)
				indexGroup.GET("/channel", GetGroupChannelChatIndexList)
				indexGroup.GET("/channel/keys", GetGroupChannelChatIndexKeys)
				indexGroup.GET("/private", GetPrivateChatIndexList)
			}
		}

		// Private chat-related APIs
		privateChatGroup := dbGroup.Group("/private-chat")
		{
			privateChatGroup.GET("/timestamp/list", GetPrivateChatTimestampList)
		}

		// User-related APIs
		userGroup := dbGroup.Group("/user")
		{
			userGroup.GET("/context", GetUserContext)
		}

		// MetaId-related APIs
		metaIdGroup := dbGroup.Group("/metaid")
		{
			metaIdGroup.GET("/join", GetMetaIdJoinList)
			metaIdGroup.GET("/context", GetMetaIdContextListByMetaId)
		}

		// Statistics-related APIs
		dbGroup.GET("/stats", GetDatabaseStats)
		dbGroup.GET("/collections", GetCollections)
		dbGroup.GET("/chat-statistics", GetChatStatistics)
		dbGroup.GET("/sync-stats", GetDbSyncStats)

		// Pin sync status-related APIs
		pinSyncGroup := dbGroup.Group("/pin-sync")
		{
			pinSyncGroup.GET("/status", GetPinSyncStatus)
			pinSyncGroup.GET("/synced-pins", GetAllSyncedPins)
			pinSyncGroup.GET("/pins-count-by-time-range", GetPinsCountByTimeRange)
			pinSyncGroup.GET("/check-pin-exists", CheckPinExistsByChainAndHeight)
			pinSyncGroup.GET("/get-pin-ids-by-height", GetPinIdsByChainAndHeight)
		}

		// Lucky bag-related APIs
		luckyBagGroup := dbGroup.Group("/luckybag")
		{
			luckyBagGroup.GET("/statistics", GetLuckyBagStatistics)
			luckyBagGroup.GET("/lock-stats", GetLuckyBagLockStats)
			luckyBagGroup.GET("/open/list", GetOpenLuckyBagList)
			luckyBagGroup.GET("/update-validation", UpdateLuckyBagValidation)
			luckyBagGroup.GET("/process-expired", ProcessExpiredLuckyBagByPinId)

			// Lucky bag extra FT info API
			luckyBagGroup.GET("/extra/txid", GetLuckyBagExtraByTxId)

			// Lucky bag manual extra gas APIs
			luckyBagGroup.POST("/process-manual-extra-gas", ProcessLuckyBagMetaContractFtManualExtraGas)
			luckyBagGroup.GET("/manual-extra-gas", GetLuckyBagManualExtraGasByLuckyBagPin)

			// Lucky bag collection-related APIs
			collectionGroup := luckyBagGroup.Group("/collection")
			{
				collectionGroup.GET("/list", GetLuckyBagCollectionList)
				collectionGroup.GET("/pinid", GetLuckyBagCollectionByPinId)
			}

			// Lucky bag queue-related APIs
			queueGroup := luckyBagGroup.Group("/queue")
			{
				queueGroup.GET("/list", GetLuckyBagQueueList)
			}

			// Lucky bag error collection-related APIs
			luckyBagGroup.GET("/error-keys", GetLuckyBagErrorCollectionKeys)
			luckyBagGroup.GET("/pin", GetLuckyBagPinByPinId)
			luckyBagGroup.GET("/code-address-key", GetLuckyBagCodeAddressKeyFromCompleted)
			luckyBagGroup.POST("/retry", RetryFailedLuckyBagOperation)
			luckyBagGroup.POST("/retry-by-luckybag-id", RetryFailedLuckyBagOperationsByLuckyBagId)
		}

		// Residue lucky bag-related APIs
		residueLuckyBagGroup := dbGroup.Group("/residue-luckybag")
		{
			residueLuckyBagGroup.GET("/pinid", GetResidueLuckyBagByPinId)
			residueLuckyBagGroup.GET("/list", GetResidueLuckyBagList)
		}

		// Migration-related APIs
		migrationGroup := dbGroup.Group("/migration")
		{
			migrationGroup.GET("/info", GetMigrationInfo)
		}

		// Global block list-related APIs
		globalBlockGroup := dbGroup.Group("/global-block")
		{
			globalBlockGroup.POST("/set", SetGlobalBlockAddress)
			globalBlockGroup.POST("/delete", DeleteGlobalBlockAddress)
			globalBlockGroup.GET("/stats", GetGlobalBlockStats)
			globalBlockGroup.GET("/addresses", GetGlobalBlockAddresses)
			globalBlockGroup.GET("/check", CheckGlobalBlockAddress)
		}

		// Global lucky bag block-related APIs
		globalLuckyBagBlockGroup := dbGroup.Group("/global-lucky-bag-block")
		{
			globalLuckyBagBlockGroup.POST("/set", SetGlobalLuckBagBlockAddress)
			globalLuckyBagBlockGroup.POST("/delete", DeleteGlobalLuckBagBlockAddress)
			globalLuckyBagBlockGroup.GET("/stats", GetGlobalLuckBagBlockStats)
			globalLuckyBagBlockGroup.GET("/addresses", GetGlobalLuckBagBlockAddresses)
			globalLuckyBagBlockGroup.GET("/check", CheckGlobalLuckBagBlockAddress)
		}
	}
}
