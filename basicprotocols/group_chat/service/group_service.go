package service

import (
	"fmt"
	"log"
	"manindexer/adapter"
	"manindexer/basicprotocols/group_chat/api/request"
	"manindexer/basicprotocols/group_chat/api/respond"
	"manindexer/basicprotocols/group_chat/common_util/logger"
	"manindexer/basicprotocols/group_chat/db"
	"manindexer/basicprotocols/group_chat/indexer"
	"manindexer/basicprotocols/group_chat/models"
	"manindexer/basicprotocols/group_chat/protocols"
	"manindexer/basicprotocols/group_chat/service/cache_service"
	"manindexer/basicprotocols/group_chat/service/common_service"
	"manindexer/basicprotocols/group_chat/service/socket_service"
	"manindexer/common"
	"sort"
	"strings"
	"time"
)

var (
	communityDB   *db.CommunityDB
	groupDB       *db.GroupDB
	chatDB        *db.ChatDB
	extraDB       *db.ExtraDB
	privateDB     *db.PrivateChatDB
	userInfoDB    *db.UserInfoDB
	socketInfoDB  *db.SocketInfoDB
	pebbleDB      *db.Pebble
	globalBlockDB *db.GlobalBlockDB
	chainAdapter  map[string]adapter.Chain
	syncDbService *db.SyncDBService
)

// InitService Initialize service
func InitService(indexer *indexer.GroupChatIndexer, adapter map[string]adapter.Chain) error {
	// Use indexer's database instance

	// Get database instances
	pebbleDB = indexer.GetPebble()
	communityDB = indexer.GetCommunityDB()
	groupDB = indexer.GetGroupDB()
	chatDB = indexer.GetChatDB()
	extraDB = indexer.GetExtraDB()
	privateDB = indexer.GetPrivateDB()
	userInfoDB = indexer.GetUserInfoDB()
	socketInfoDB = indexer.GetSocketInfoDB()
	globalBlockDB = indexer.GetGlobalBlockDB()
	chainAdapter = adapter
	syncDbService = indexer.GetSyncDBService()

	StartOpenLuckyBagQueueProcessor()
	StartResidueLuckyBagQueueProcessor()
	StartExpiredLuckyBagProcessor()

	db.SetHandleGroupChatItem(wsForGroupChatItem)
	db.SetHandlePrivateChatItem(wsForPrivateChatItem)
	db.SetHandleGroupRoleInfoChangeList(wsForGroupRoleInfoChange)

	// Set process pin function for sync service
	GetSyncService().SetProcessPinFunc(indexer.ProcessPin)

	// Initialize cache service for lucky bag
	cache_service.InitCacheService(
		common.Config.Redis.RedisAddr,
		common.Config.Redis.RedisPassword,
		common.Config.Redis.RedisDB)

	// Start user info polling
	common_service.StartUserInfoPolling()

	startLuckyBagGrabCleanupGoroutine()

	// Start socket info snapshot timer
	startSocketInfoSnapshotTimer()

	return nil
}

// FetchGroupList Get group list
func FetchGroupList(req *request.FetchGroupListRequest) (*respond.GroupResponse, error) {
	// Set default pagination parameters
	if req.Size <= 0 {
		req.Size = 20
	}
	if req.Cursor <= 0 {
		req.Cursor = 0
	}
	if req.Size > 100 {
		req.Size = 100
	}

	var groups []*models.TalkGroupModel
	var total int64
	var err error

	// If metaId is not empty, get user's joined group list
	if req.MetaId != "" {
		groups, total, err = groupDB.GetGroupListByMetaId(req.MetaId, req.Cursor, req.Size)
	} else {
		// Get all group list
		groups, total, err = groupDB.GetGroupList(req.Cursor, req.Size)
	}

	if err != nil {
		return nil, err
	}

	// Convert to response format
	var groupItems []*respond.GroupItem
	for _, group := range groups {
		// Get group's latest chat info
		latestChat, err := chatDB.GetGroupLatestChat(group.GroupId)
		if err != nil {
			// If failed to get, use default value
			latestChat = nil
		}

		// Get group member count
		// userCount, err := groupDB.GetGroupMemberCount(group.GroupId)
		userCount, err := groupDB.GetGroupMemberCountFromList(group.GroupId)
		if err != nil {
			// If failed to get, use default value
			userCount = 0
		}

		groupItem := &respond.GroupItem{
			CommunityId:        group.CommunityId,
			GroupId:            group.GroupId,
			TxId:               group.TxId,
			PinId:              group.PinId,
			RoomName:           group.RoomName,
			RoomNote:           group.RoomNote,
			RoomIcon:           group.RoomIcon,
			RoomType:           group.RoomType,
			RoomStatus:         group.RoomStatus,
			RoomJoinType:       group.RoomJoinType,
			RoomAvatarUrl:      group.RoomAvatarUrl,
			RoomNinePersonHash: "", // Not implemented yet
			RoomNewestTxId: func() string {
				if latestChat != nil {
					return latestChat.TxId
				}
				return ""
			}(),
			RoomNewestPinId: func() string {
				if latestChat != nil {
					return latestChat.LastMessagePinId
				}
				return ""
			}(),
			RoomNewestMetaId: func() string {
				if latestChat != nil {
					return latestChat.MetaId
				}
				return ""
			}(),
			RoomNewestUserName: "", // Not implemented yet, need to get from user info
			RoomNewestProtocol: func() string {
				if latestChat != nil {
					return latestChat.Protocol
				}
				return ""
			}(),
			RoomNewestContent: func() string {
				if latestChat != nil {
					return latestChat.Content
				}
				return ""
			}(),
			RoomNewestTimestamp: func() int64 {
				if latestChat != nil {
					return latestChat.Timestamp
				}
				return 0
			}(),
			CreateUserMetaId:  group.CreateUserMetaId,
			CreateUserAddress: group.CreateUserAddress,
			CreateUserInfo:    common_service.FetchMetaIDUserInfo(group.CreateUserAddress),
			UserCount:         userCount,
			ChatSettingType:   group.ChatSettingType,
			DeleteStatus:      group.DeleteStatus,
			Path:              group.Path,
			Timestamp:         group.Timestamp,
			Chain:             group.Chain,
			BlockHeight:       group.BlockHeight,
		}
		groupItems = append(groupItems, groupItem)
	}

	return &respond.GroupResponse{
		Total: total,
		List:  groupItems,
	}, nil
}

// FetchLatestChatGroupList Get latest chat group list
func FetchLatestChatGroupList(req *request.FetchLatestChatGroupListRequest) (*respond.GroupResponse, error) {
	// Set default pagination parameters
	if req.Size <= 0 {
		req.Size = 20
	}
	if req.Cursor <= 0 {
		req.Cursor = 0
	}

	// Get user's group list (based on latest chat time)
	contextList, err := chatDB.GetMetaIdContextList(req.MetaId)
	if err != nil {
		return nil, err
	}

	// Convert to response format
	var groupItems []*respond.GroupItem
	for _, item := range contextList.Items {
		// Get group detailed info
		group, err := groupDB.GetGroupInfoByGroupId(item.GroupId)
		if err != nil || group == nil {
			continue
		}

		// Get group's latest chat info
		latestChat, err := chatDB.GetGroupLatestChat(item.GroupId)
		if err != nil {
			// If failed to get, use default value
			latestChat = nil
		}

		// Get group member count
		// userCount, err := groupDB.GetGroupMemberCount(item.GroupId)
		userCount, err := groupDB.GetGroupMemberCountFromList(item.GroupId)
		if err != nil {
			// If failed to get, use default value
			userCount = 0
		}

		// Get group chat index
		groupChatIndex := int64(-1)

		chatInfo, _ := chatDB.GetChatByPinId(latestChat.PinId)
		if chatInfo != nil {
			groupChatIndex = chatInfo.Index
		}

		groupItem := &respond.GroupItem{
			CommunityId:  group.CommunityId,
			GroupId:      group.GroupId,
			TxId:         group.TxId,
			PinId:        group.PinId,
			RoomName:     group.RoomName,
			RoomNote:     group.RoomNote,
			RoomIcon:     group.RoomIcon,
			RoomType:     group.RoomType,
			RoomStatus:   group.RoomStatus,
			RoomJoinType: group.RoomJoinType,
			// RoomCodeHash:          "", // Not implemented yet
			// RoomGenesis:           "", // Not implemented yet
			// RoomLimitAmount:       0,  // Not implemented yet
			// RoomGenesisSeriesName: "", // Not implemented yet
			RoomAvatarUrl:      group.RoomAvatarUrl,
			RoomNinePersonHash: "", // Not implemented yet
			RoomNewestTxId: func() string {
				if latestChat != nil {
					return latestChat.TxId
				}
				return ""
			}(),
			RoomNewestPinId: func() string {
				if latestChat != nil {
					return latestChat.LastMessagePinId
				}
				return ""
			}(),
			RoomNewestMetaId: func() string {
				if latestChat != nil {
					return latestChat.MetaId
				}
				return ""
			}(),
			RoomNewestUserName: "", // Not implemented yet, need to get from user info
			RoomNewestProtocol: func() string {
				if latestChat != nil {
					return latestChat.Protocol
				}
				return ""
			}(),
			RoomNewestContent: func() string {
				if latestChat != nil {
					return latestChat.Content
				}
				return ""
			}(),
			RoomNewestTimestamp: func() int64 {
				if latestChat != nil {
					return latestChat.Timestamp
				}
				return 0
			}(),
			CreateUserMetaId:  group.CreateUserMetaId,
			CreateUserAddress: group.CreateUserAddress,
			CreateUserInfo:    common_service.FetchMetaIDUserInfo(group.CreateUserAddress),
			UserCount:         userCount,
			ChatSettingType:   group.ChatSettingType,
			DeleteStatus:      group.DeleteStatus,
			Path:              group.Path,
			Timestamp:         group.Timestamp,
			Chain:             group.Chain,
			BlockHeight:       group.BlockHeight,
			Index:             groupChatIndex,
		}
		groupItems = append(groupItems, groupItem)
	}

	return &respond.GroupResponse{
		Total: int64(len(groupItems)),
		List:  groupItems,
	}, nil
}

// FetchGroupInfo Get group info
func FetchGroupInfo(req *request.FetchGroupInfoRequest) (*respond.GroupItem, error) {
	// Performance monitoring: record start time
	startTime := time.Now()
	var perfStats = struct {
		groupInfoTime      int64
		latestChatTime     int64
		memberCountTime    int64
		responseFormatTime int64
		totalTime          int64
		cacheHit           bool
	}{}

	// Try to get group info from cache first
	t := time.Now().UnixMilli()
	group, found := cache_service.GetGroupInfoFromCache(req.GroupId)
	if !found {
		// Cache miss, get from database
		var err error
		group, err = groupDB.GetGroupInfoByGroupId(req.GroupId)
		if err != nil {
			return nil, err
		}
		if group == nil {
			return nil, nil
		}
		perfStats.cacheHit = false
		perfStats.groupInfoTime = time.Now().UnixMilli() - t

		// Update cache with the fetched data
		cache_service.SetGroupInfoToCache(req.GroupId, group)
	} else {
		perfStats.cacheHit = true
		perfStats.groupInfoTime = time.Now().UnixMilli() - t
	}

	t = time.Now().UnixMilli()
	// Get group's latest chat info
	latestChat, err := chatDB.GetGroupLatestChat(req.GroupId)
	if err != nil {
		// If failed to get, use default value
		latestChat = nil
	}
	perfStats.latestChatTime = time.Now().UnixMilli() - t

	t = time.Now().UnixMilli()
	// Get group member count
	userCount, err := groupDB.GetGroupMemberCountFromList(req.GroupId)
	if err != nil {
		// If failed to get, use default value
		userCount = 0
	}
	perfStats.memberCountTime = time.Now().UnixMilli() - t

	t = time.Now().UnixMilli()
	// Convert to response format
	groupItem := &respond.GroupItem{
		CommunityId:        group.CommunityId,
		GroupId:            group.GroupId,
		TxId:               group.TxId,
		PinId:              group.PinId,
		RoomName:           group.RoomName,
		RoomNote:           group.RoomNote,
		RoomIcon:           group.RoomIcon,
		RoomType:           group.RoomType,
		RoomStatus:         group.RoomStatus,
		RoomJoinType:       group.RoomJoinType,
		RoomAvatarUrl:      group.RoomAvatarUrl,
		RoomNinePersonHash: "", // Not implemented yet
		RoomNewestTxId: func() string {
			if latestChat != nil {
				return latestChat.TxId
			}
			return ""
		}(),
		RoomNewestPinId: func() string {
			if latestChat != nil {
				return latestChat.LastMessagePinId
			}
			return ""
		}(),
		RoomNewestMetaId: func() string {
			if latestChat != nil {
				return latestChat.MetaId
			}
			return ""
		}(),
		RoomNewestUserName: "", // Not implemented yet, need to get from user info
		RoomNewestProtocol: func() string {
			if latestChat != nil {
				return latestChat.Protocol
			}
			return ""
		}(),
		RoomNewestContent: func() string {
			if latestChat != nil {
				return latestChat.Content
			}
			return ""
		}(),
		RoomNewestTimestamp: func() int64 {
			if latestChat != nil {
				return latestChat.Timestamp
			}
			return 0
		}(),
		CreateUserMetaId:  group.CreateUserMetaId,
		CreateUserAddress: group.CreateUserAddress,
		CreateUserInfo:    common_service.FetchMetaIDUserInfo(group.CreateUserAddress),
		UserCount:         userCount,
		ChatSettingType:   group.ChatSettingType,
		DeleteStatus:      group.DeleteStatus,
		Path:              group.Path,
		Timestamp:         group.Timestamp,
		Chain:             group.Chain,
		BlockHeight:       group.BlockHeight,
	}
	perfStats.responseFormatTime = time.Now().UnixMilli() - t
	perfStats.totalTime = time.Since(startTime).Milliseconds()

	// Unified performance logging
	cacheStatus := "DB"
	if perfStats.cacheHit {
		cacheStatus = "Cache"
	}
	logger.Info("[GROUP_SERVICE][FETCH_GROUP_INFO] Performance Stats - "+
		"Total: %dms, GroupInfo: %dms, LatestChat: %dms, MemberCount: %dms, "+
		"ResponseFormat: %dms, Source: %s",
		perfStats.totalTime,
		perfStats.groupInfoTime,
		perfStats.latestChatTime,
		perfStats.memberCountTime,
		perfStats.responseFormatTime,
		cacheStatus)
	return groupItem, nil
}

// FetchGroupChatList Get group chat list
func FetchGroupChatList(req *request.FetchGroupChatListRequest) (*respond.GroupChatResponse, error) {
	// Set default pagination parameters
	if req.Size <= 0 {
		req.Size = 20
	}

	var chats []*models.TalkGroupChatV3
	var err error

	if req.Timestamp > 0 {
		// Get chat records by timestamp range
		chats, err = chatDB.GetChatsByGroupIdAndTimestampRange(req.GroupId, req.Timestamp, req.Size)
	} else {
		// Get latest chat records
		chats, err = chatDB.GetLatestChatsByGroupId(req.GroupId, req.Size)
	}

	if err != nil {
		return nil, err
	}
	// logger.Info("chats: %+v\n", chats)

	// Convert to response format
	var chatItems []*respond.GroupChatItem
	var nextTimestamp int64 = 0

	for i, chat := range chats {
		chatItem := &respond.GroupChatItem{
			GroupId:     chat.GroupId,
			MetanetId:   chat.GroupId, // Use GroupId as MetanetId
			TxId:        chat.TxId,
			PinId:       chat.PinId,
			Address:     chat.Address,
			MetaId:      chat.MetaId,
			UserInfo:    common_service.FetchMetaIDUserInfo(chat.Address),
			NickName:    "", // Need to get from user info
			Protocol:    chat.Protocol,
			Content:     chat.Content,
			ContentType: chat.ContentType,
			Encryption:  chat.Encryption,
			ChatType:    chat.ChatType,
			ReplyPin:    chat.ReplyPin,
			ReplyInfo:   nil,
			ReplyMetaId: "",
			Mention:     chat.Mention,
			Timestamp:   chat.Timestamp,
			Chain:       chat.Chain,
			BlockHeight: chat.BlockHeight,
		}
		if strings.Contains(strings.ToLower(chatItem.Protocol), strings.ToLower(protocols.MonitorSimpleGroupOpenLuckyBag)) {
			openLuckyBag, _ := chatDB.GetOpenLuckyBagByPinId(chat.PinId)
			if openLuckyBag != nil {
				// logger.Info("openLuckyBag: GrabTxId: %s, PinId: %s, GrabState: %d\n", openLuckyBag.GrabTxId, openLuckyBag.PinId, openLuckyBag.GrabState)
				if openLuckyBag.GrabState == models.GrabStateOpenAndSend {
					chatItem.TxId = openLuckyBag.GrabTxId
				} else {
					chatItem.TxId = ""
				}
			}
		}
		if chat.ReplyPin != "" {
			replyChat, err := chatDB.GetChatByPinId(chat.ReplyPin)
			if err != nil {
				replyChat = nil
			}
			chatItem.ReplyInfo = &respond.ReplyInfo{
				PinId:       replyChat.PinId,
				MetaId:      replyChat.MetaId,
				Address:     replyChat.Address,
				UserInfo:    common_service.FetchMetaIDUserInfo(replyChat.Address),
				NickName:    replyChat.NickName,
				Mention:     replyChat.Mention,
				Protocol:    replyChat.Protocol,
				Content:     replyChat.Content,
				ContentType: replyChat.ContentType,
				Encryption:  replyChat.Encryption,
				ChatType:    replyChat.ChatType,
				Timestamp:   replyChat.Timestamp,
				Chain:       replyChat.Chain,
			}
			chatItem.ReplyMetaId = replyChat.MetaId
			chatItem.BlockHeight = replyChat.BlockHeight
		}

		chatItems = append(chatItems, chatItem)

		// Record next message timestamp (for pagination)
		if i == len(chats)-1 && len(chats) > 0 {
			nextTimestamp = chat.Timestamp
		}
	}

	return &respond.GroupChatResponse{
		Total:         int64(len(chatItems)),
		NextTimestamp: nextTimestamp,
		List:          chatItems,
	}, nil
}

// FetchGroupChatListV3 Get group chat list using GetChatsByGroupIdAndTimestampRange3 (test version with IterOptions)
func FetchGroupChatListV3(req *request.FetchGroupChatListRequest) (*respond.GroupChatResponse, error) {
	// Set default pagination parameters
	if req.Size <= 0 {
		req.Size = 20
	}

	// Performance monitoring: record start time
	startTime := time.Now()
	var perfStats = struct {
		getChatsTime       int64
		responseFormatTime int64
		userInfoTime       int64
		totalTime          int64
	}{}

	var chats []*models.TalkGroupChatV3
	var err error

	t := time.Now().UnixMilli()
	var nextTimestamp int64 = 0
	if req.Timestamp > 0 {
		// Get chat records by timestamp range using new method with IterOptions
		chats, nextTimestamp, err = chatDB.GetChatsByGroupIdAndEndTimestampRange3(req.GroupId, req.Timestamp, req.Size)
	} else {
		// Get latest chat records using new method with IterOptions
		// For latest messages, we can use a very large timestamp as start point
		currentTimestamp := time.Now().Unix()
		//add 6 number 0
		currentTimestamp = currentTimestamp * 1000000
		chats, nextTimestamp, err = chatDB.GetChatsByGroupIdAndEndTimestampRange3(req.GroupId, currentTimestamp, req.Size)
	}
	perfStats.getChatsTime = time.Now().UnixMilli() - t

	if err != nil {
		return nil, err
	}

	// Convert to response format
	var chatItems []*respond.GroupChatItem

	t1 := time.Now().UnixMilli()
	for _, chat := range chats {
		chatItem := &respond.GroupChatItem{
			GroupId:   chat.GroupId,
			MetanetId: chat.GroupId, // Use GroupId as MetanetId
			TxId:      chat.TxId,
			PinId:     chat.PinId,
			Address:   chat.Address,
			// UserInfo:    common_service.FetchMetaIDUserInfo(chat.Address),
			MetaId:      chat.MetaId,
			NickName:    "", // Need to get from user info
			Protocol:    chat.Protocol,
			Content:     chat.Content,
			ContentType: chat.ContentType,
			Encryption:  chat.Encryption,
			Version:     chat.Version,
			ChatType:    chat.ChatType,
			ReplyPin:    chat.ReplyPin,
			ReplyInfo:   nil,
			ReplyMetaId: "",
			Mention:     chat.Mention,
			Timestamp:   chat.Timestamp,
			Chain:       chat.Chain,
			BlockHeight: chat.BlockHeight,
			Index:       chat.Index,
		}
		if strings.Contains(strings.ToLower(chatItem.Protocol), strings.ToLower(protocols.MonitorSimpleGroupOpenLuckyBag)) {
			openLuckyBag, _ := chatDB.GetOpenLuckyBagByPinId(chat.PinId)
			if openLuckyBag != nil {
				// logger.Info("openLuckyBag: GrabTxId: %s, PinId: %s, GrabState: %d\n", openLuckyBag.GrabTxId, openLuckyBag.PinId, openLuckyBag.GrabState)
				if openLuckyBag.GrabState == models.GrabStateOpenAndSend {
					chatItem.TxId = openLuckyBag.GrabTxId
				} else {
					chatItem.TxId = ""
				}
			}
		} else if strings.Contains(strings.ToLower(chatItem.Protocol), strings.ToLower(protocols.MonitorSimpleGroupLuckyBag)) {
			luckyBag, _ := chatDB.GetLuckyBagByPinIdFromCache(chat.GroupId, chat.PinId)
			if luckyBag != nil {
				chatItem.Domain = luckyBag.Domain
			}
		}
		if chat.ReplyPin != "" {
			replyChat, _ := chatDB.GetChatByPinId(chat.ReplyPin)
			if replyChat != nil {
				chatItem.ReplyInfo = &respond.ReplyInfo{
					PinId:   replyChat.PinId,
					MetaId:  replyChat.MetaId,
					Address: replyChat.Address,
					// UserInfo:    common_service.FetchMetaIDUserInfo(replyChat.Address),
					NickName:    replyChat.NickName,
					Protocol:    replyChat.Protocol,
					Content:     replyChat.Content,
					ContentType: replyChat.ContentType,
					Encryption:  replyChat.Encryption,
					Version:     replyChat.Version,
					ChatType:    replyChat.ChatType,
					Mention:     replyChat.Mention,
					Timestamp:   replyChat.Timestamp,
					Chain:       replyChat.Chain,
					Index:       replyChat.Index,
				}
			}
			chatItem.ReplyMetaId = replyChat.MetaId
			chatItem.BlockHeight = replyChat.BlockHeight
		}

		chatItems = append(chatItems, chatItem)
	}
	perfStats.responseFormatTime = time.Now().UnixMilli() - t1

	//get user info
	t2 := time.Now().UnixMilli()
	for _, chatItem := range chatItems {
		if chatItem.ReplyInfo != nil {
			chatItem.ReplyInfo.UserInfo = common_service.FetchMetaIDUserInfo(chatItem.ReplyInfo.Address)
		}
		chatItem.UserInfo = common_service.FetchMetaIDUserInfo(chatItem.Address)
	}
	perfStats.userInfoTime = time.Now().UnixMilli() - t2
	perfStats.totalTime = time.Since(startTime).Milliseconds()

	// Unified performance logging
	logger.Info("[CHAT_SERVICE][FETCH_GROUP_CHAT_LIST_V3] Performance Stats - "+
		"Total: %dms, GetChats: %dms, ResponseFormat: %dms, UserInfo: %dms, Items: %d",
		perfStats.totalTime,
		perfStats.getChatsTime,
		perfStats.responseFormatTime,
		perfStats.userInfoTime,
		len(chatItems))

	return &respond.GroupChatResponse{
		Total:         int64(len(chatItems)),
		NextTimestamp: nextTimestamp,
		List:          chatItems,
	}, nil
}

// FetchGroupChatListV2 Get group chat list using TalkGroupChatTimestamp2Collection
func FetchGroupChatListV2(req *request.FetchGroupChatListRequest) (*respond.GroupChatResponse, error) {
	// Set default pagination parameters
	if req.Size <= 0 {
		req.Size = 20
	}

	// Performance monitoring: record start time
	startTime := time.Now()
	var perfStats = struct {
		getChatsTime       int64
		responseFormatTime int64
		userInfoTime       int64
		totalTime          int64
	}{}

	var chats []*models.TalkGroupChatV3
	var err error

	t := time.Now().UnixMilli()
	var nextTimestamp int64 = 0
	if req.Timestamp > 0 {
		// Get chat records by timestamp range using new collection
		chats, nextTimestamp, err = chatDB.GetChatsByGroupIdAndEndTimestampRange2(req.GroupId, req.Timestamp, req.Size)
	} else {
		// Get latest chat records using new collection
		// For latest messages, we can use a very large timestamp as start point
		currentTimestamp := time.Now().Unix()
		//add 6 number 0
		currentTimestamp = currentTimestamp * 1000000
		chats, nextTimestamp, err = chatDB.GetChatsByGroupIdAndEndTimestampRange2(req.GroupId, currentTimestamp, req.Size)
	}
	perfStats.getChatsTime = time.Now().UnixMilli() - t

	if err != nil {
		return nil, err
	}

	// Convert to response format
	var chatItems []*respond.GroupChatItem

	t1 := time.Now().UnixMilli()
	for _, chat := range chats {
		chatItem := &respond.GroupChatItem{
			GroupId:   chat.GroupId,
			MetanetId: chat.GroupId, // Use GroupId as MetanetId
			TxId:      chat.TxId,
			PinId:     chat.PinId,
			Address:   chat.Address,
			// UserInfo:    common_service.FetchMetaIDUserInfo(chat.Address),
			MetaId:      chat.MetaId,
			NickName:    "", // Need to get from user info
			Protocol:    chat.Protocol,
			Content:     chat.Content,
			ContentType: chat.ContentType,
			Encryption:  chat.Encryption,
			Version:     chat.Version,
			ChatType:    chat.ChatType,
			ReplyPin:    chat.ReplyPin,
			ReplyInfo:   nil,
			ReplyMetaId: "",
			Mention:     chat.Mention,
			Timestamp:   chat.Timestamp,
			Chain:       chat.Chain,
			BlockHeight: chat.BlockHeight,
			Index:       chat.Index,
		}
		if strings.Contains(strings.ToLower(chatItem.Protocol), strings.ToLower(protocols.MonitorSimpleGroupOpenLuckyBag)) {
			openLuckyBag, _ := chatDB.GetOpenLuckyBagByPinId(chat.PinId)
			if openLuckyBag != nil {
				// logger.Info("openLuckyBag: GrabTxId: %s, PinId: %s, GrabState: %d\n", openLuckyBag.GrabTxId, openLuckyBag.PinId, openLuckyBag.GrabState)
				if openLuckyBag.GrabState == models.GrabStateOpenAndSend {
					chatItem.TxId = openLuckyBag.GrabTxId
				} else {
					chatItem.TxId = ""
				}
			}
		} else if strings.Contains(strings.ToLower(chatItem.Protocol), strings.ToLower(protocols.MonitorSimpleGroupLuckyBag)) {
			luckyBag, _ := chatDB.GetLuckyBagByPinIdFromCache(chat.GroupId, chat.PinId)
			if luckyBag != nil {
				chatItem.Domain = luckyBag.Domain
			}
		}
		if chat.ReplyPin != "" {
			replyChat, _ := chatDB.GetChatByPinId(chat.ReplyPin)
			if replyChat != nil {
				chatItem.ReplyInfo = &respond.ReplyInfo{
					PinId:   replyChat.PinId,
					MetaId:  replyChat.MetaId,
					Address: replyChat.Address,
					// UserInfo:    common_service.FetchMetaIDUserInfo(replyChat.Address),
					NickName:    replyChat.NickName,
					Protocol:    replyChat.Protocol,
					Content:     replyChat.Content,
					ContentType: replyChat.ContentType,
					Encryption:  replyChat.Encryption,
					Version:     replyChat.Version,
					ChatType:    replyChat.ChatType,
					Mention:     replyChat.Mention,
					Timestamp:   replyChat.Timestamp,
					Chain:       replyChat.Chain,
					Index:       replyChat.Index,
				}
				chatItem.ReplyMetaId = replyChat.MetaId
				chatItem.BlockHeight = replyChat.BlockHeight
			}
		}

		chatItems = append(chatItems, chatItem)

		// Record next message timestamp (for pagination)
		// if i == len(chats)-1 && len(chats) > 0 {
		// 	nextTimestamp = chat.Timestamp
		// }
	}
	perfStats.responseFormatTime = time.Now().UnixMilli() - t1

	//get user info
	t2 := time.Now().UnixMilli()
	for _, chatItem := range chatItems {
		if chatItem.ReplyInfo != nil {
			chatItem.ReplyInfo.UserInfo = common_service.FetchMetaIDUserInfo(chatItem.ReplyInfo.Address)
		}
		chatItem.UserInfo = common_service.FetchMetaIDUserInfo(chatItem.Address)
	}
	perfStats.userInfoTime = time.Now().UnixMilli() - t2
	perfStats.totalTime = time.Since(startTime).Milliseconds()

	// Unified performance logging
	logger.Info("[CHAT_SERVICE][FETCH_GROUP_CHAT_LIST_V2] Performance Stats - "+
		"Total: %dms, GetChats: %dms, ResponseFormat: %dms, UserInfo: %dms, Items: %d",
		perfStats.totalTime,
		perfStats.getChatsTime,
		perfStats.responseFormatTime,
		perfStats.userInfoTime,
		len(chatItems))

	return &respond.GroupChatResponse{
		Total:         int64(len(chatItems)),
		NextTimestamp: nextTimestamp,
		List:          chatItems,
	}, nil
}

// FetchGroupMemberList Get group member list
func FetchGroupMemberList(req *request.FetchGroupMemberListRequest) (*respond.GroupMemberResponse, error) {
	// Set default pagination parameters
	if req.Size <= 0 {
		req.Size = 20
	}
	if req.Cursor <= 0 {
		req.Cursor = 0
	}

	// Performance monitoring: record start time
	startTime := time.Now()
	var perfStats = struct {
		getMembersTime     int64
		responseFormatTime int64
		totalTime          int64
	}{}

	var members []*models.TalkGroupJoinModel
	var total int64
	var err error

	// Check if orderBy is "timestamp" for timestamp descending order
	if req.OrderBy == "timestamp" {
		t := time.Now().UnixMilli()
		// Get all group members first
		allMembers, err := groupDB.GetGroupMembers(req.GroupId)
		if err != nil {
			return nil, err
		}
		perfStats.getMembersTime = time.Now().UnixMilli() - t

		// Sort by timestamp in descending order
		sort.Slice(allMembers, func(i, j int) bool {
			if req.OrderType == "desc" {
				return allMembers[i].Timestamp > allMembers[j].Timestamp
			} else {
				return allMembers[i].Timestamp < allMembers[j].Timestamp
			}
		})

		total = int64(len(allMembers))

		// Apply pagination in code
		start := req.Cursor
		end := start + req.Size
		if start >= total {
			// No more data
			members = []*models.TalkGroupJoinModel{}
		} else if end > total {
			// Last page
			members = allMembers[start:total]
		} else {
			// Regular page
			members = allMembers[start:end]
		}
	} else {
		// Use original pagination logic
		t := time.Now().UnixMilli()
		members, total, err = groupDB.GetGroupMembersWithPagination(req.GroupId, req.Cursor, req.Size)
		if err != nil {
			return nil, err
		}
		perfStats.getMembersTime = time.Now().UnixMilli() - t
	}

	t1 := time.Now().UnixMilli()
	// Convert to response format
	var memberItems []*respond.GroupMemberItem
	for _, member := range members {
		memberItem := &respond.GroupMemberItem{
			MetaId: member.MetaId,
			// Name:      member.UserName, // Use UserName field
			UserInfo:  common_service.FetchMetaIDUserInfo(member.Address),
			Address:   member.Address,
			TimeStr:   time.Unix(member.Timestamp, 0).Format("2006-01-02 15:04:05"),
			Timestamp: member.Timestamp,
		}
		memberItems = append(memberItems, memberItem)
	}
	perfStats.responseFormatTime = time.Now().UnixMilli() - t1
	perfStats.totalTime = time.Since(startTime).Milliseconds()

	// Unified performance logging
	logger.Info("[CHAT_SERVICE][FETCH_GROUP_MEMBER_LIST] Performance Stats - "+
		"Total: %dms, GetMembers: %dms, ResponseFormat: %dms, Items: %d",
		perfStats.totalTime,
		perfStats.getMembersTime,
		perfStats.responseFormatTime,
		len(memberItems))

	return &respond.GroupMemberResponse{
		Total: total,
		List:  memberItems,
	}, nil
}

// FetchGroupMemberListV2 Get group member list using TalkGroupPersonListCollection (already sorted)
func FetchGroupMemberListV2(req *request.FetchGroupMemberListRequest) (*respond.GroupMemberResponse, error) {
	// Set default pagination parameters
	if req.Size <= 0 {
		req.Size = 20
	}
	if req.Cursor <= 0 {
		req.Cursor = 0
	}

	// Performance monitoring: record start time
	startTime := time.Now()
	var perfStats = struct {
		getMembersTime     int64
		getGroupInfoTime   int64
		getAdminListTime   int64
		getBlockListTime   int64
		getWhitelistTime   int64
		responseFormatTime int64
		totalTime          int64
		cacheHit           bool
		groupInfoCacheHit  bool
		adminCacheHit      bool
		blockCacheHit      bool
		whitelistCacheHit  bool
	}{}

	// Try to get from cache first
	var allMembers []*models.TalkGroupPerson
	var err error

	t := time.Now().UnixMilli()
	if personList, found := cache_service.GetGroupMemberListFromCache(req.GroupId); found {
		// Use cached data
		allMembers = personList.Persons
		perfStats.cacheHit = true
		perfStats.getMembersTime = time.Now().UnixMilli() - t
	} else {
		// Cache miss, get from database
		allMembers, err = groupDB.GetGroupMembersFromList(req.GroupId)
		if err != nil {
			return nil, err
		}
		perfStats.cacheHit = false
		perfStats.getMembersTime = time.Now().UnixMilli() - t

		// Update cache with the data from database
		if len(allMembers) > 0 {
			personList := &models.TalkGroupPersonList{
				GroupId: req.GroupId,
				Persons: allMembers,
			}
			cache_service.SetGroupMemberListToCache(req.GroupId, personList)
		}
	}

	total := int64(len(allMembers))

	// Apply pagination in code
	start := req.Cursor
	end := start + req.Size
	var members []*models.TalkGroupPerson
	if start >= total {
		// No more data
		members = []*models.TalkGroupPerson{}
	} else if end > total {
		// Last page
		members = allMembers[start:total]
	} else {
		// Regular page
		members = allMembers[start:end]
	}

	t1 := time.Now().UnixMilli()

	// Try to get group info from cache first
	var group *models.TalkGroupModel
	t2 := time.Now().UnixMilli()
	if cachedGroup, found := cache_service.GetGroupInfoFromCache(req.GroupId); found {
		// Use cached data
		group = cachedGroup
		perfStats.groupInfoCacheHit = true
		perfStats.getGroupInfoTime = time.Now().UnixMilli() - t2
	} else {
		// Cache miss, get from database
		group, err = groupDB.GetGroupInfoByGroupId(req.GroupId)
		if err != nil {
			return nil, err
		}
		perfStats.groupInfoCacheHit = false
		perfStats.getGroupInfoTime = time.Now().UnixMilli() - t2

		// Update cache with the fetched data
		if group != nil {
			cache_service.SetGroupInfoToCache(req.GroupId, group)
		}
	}

	// Try to get admins data from cache first
	var adminList *models.GroupAdminList
	var blockList *models.GroupBlockList
	var whitelistList *models.GroupWhitelistList

	t3 := time.Now().UnixMilli()
	// Get admin list from cache
	adminList, adminFound := cache_service.GetGroupAdminListFromCache(req.GroupId)
	perfStats.adminCacheHit = adminFound
	if !adminFound {
		// Cache miss, get from database
		adminList, err = groupDB.GetGroupAdminList(req.GroupId)
		if err != nil {
			logger.Info(fmt.Sprintf("Failed to get group admin list for groupId %s: %v", req.GroupId, err))
			adminList = &models.GroupAdminList{GroupId: req.GroupId, Items: []*models.GroupAdminItem{}}
		}
		// Update cache with the fetched data
		cache_service.SetGroupAdminListToCache(req.GroupId, adminList)
	}
	perfStats.getAdminListTime = time.Now().UnixMilli() - t3

	t4 := time.Now().UnixMilli()
	// Get block list from cache
	blockList, blockFound := cache_service.GetGroupBlockListFromCache(req.GroupId)
	perfStats.blockCacheHit = blockFound
	if !blockFound {
		// Cache miss, get from database
		blockList, err = groupDB.GetGroupBlockList(req.GroupId)
		if err != nil {
			logger.Info(fmt.Sprintf("Failed to get group block list for groupId %s: %v", req.GroupId, err))
			blockList = &models.GroupBlockList{GroupId: req.GroupId, Items: []*models.GroupBlockItem{}}
		}
		// Update cache with the fetched data
		cache_service.SetGroupBlockListToCache(req.GroupId, blockList)
	}
	perfStats.getBlockListTime = time.Now().UnixMilli() - t4

	t5 := time.Now().UnixMilli()
	// Get whitelist from cache
	whitelistList, whitelistFound := cache_service.GetGroupWhitelistFromCache(req.GroupId)
	perfStats.whitelistCacheHit = whitelistFound
	if !whitelistFound {
		// Cache miss, get from database
		whitelistList, err = groupDB.GetGroupWhitelistList(req.GroupId)
		if err != nil {
			logger.Info(fmt.Sprintf("Failed to get group whitelist for groupId %s: %v", req.GroupId, err))
			whitelistList = &models.GroupWhitelistList{GroupId: req.GroupId, Items: []*models.GroupWhitelistItem{}}
		}
		// Update cache with the fetched data
		cache_service.SetGroupWhitelistToCache(req.GroupId, whitelistList)
	}
	perfStats.getWhitelistTime = time.Now().UnixMilli() - t5

	// Helper function to create GroupMemberItem
	createMemberItem := func(metaId, address string, timestamp int64) *respond.GroupMemberItem {
		return &respond.GroupMemberItem{
			MetaId:    metaId,
			UserInfo:  common_service.FetchMetaIDUserInfo(address),
			Address:   address,
			TimeStr:   time.Unix(timestamp, 0).Format("2006-01-02 15:04:05"),
			Timestamp: timestamp,
		}
	}

	// Find creator
	var creator *respond.GroupMemberItem
	if group != nil && group.CreateUserAddress != "" {
		creator = createMemberItem(group.CreateUserMetaId, group.CreateUserAddress, group.Timestamp)
	}

	// Get current effective admins
	var admins []*respond.GroupMemberItem
	if len(adminList.Items) > 0 {
		// Get the latest admin record (last item since sorted by timestamp ascending)
		latestAdminItem := adminList.Items[len(adminList.Items)-1]
		for _, adminMetaId := range latestAdminItem.Admins {
			// Skip if this is the creator (creator is not in admin list)
			if group != nil && adminMetaId == group.CreateUserMetaId {
				continue
			}
			// Find admin's address from member list
			for _, member := range allMembers {
				if member.MetaId == adminMetaId {
					admins = append(admins, createMemberItem(adminMetaId, member.Address, member.Timestamp))
					break
				}
			}
		}
	}

	// Get current effective blocked users
	var blockListMembers []*respond.GroupMemberItem
	if len(blockList.Items) > 0 {
		// Get the latest block record (last item since sorted by timestamp ascending)
		latestBlockItem := blockList.Items[len(blockList.Items)-1]
		for _, blockedMetaId := range latestBlockItem.BlockedUsers {
			// Find blocked user's address from member list
			for _, member := range allMembers {
				if member.MetaId == blockedMetaId {
					blockListMembers = append(blockListMembers, createMemberItem(blockedMetaId, member.Address, member.Timestamp))
					break
				}
			}
		}
	}

	// Get current effective whitelisted users
	var whitelistMembers []*respond.GroupMemberItem
	if len(whitelistList.Items) > 0 {
		// Get the latest whitelist record (last item since sorted by timestamp ascending)
		latestWhitelistItem := whitelistList.Items[len(whitelistList.Items)-1]
		for _, whitelistMetaId := range latestWhitelistItem.WhitelistUsers {
			// Find whitelist user's address from member list
			for _, member := range allMembers {
				if member.MetaId == whitelistMetaId {
					whitelistMembers = append(whitelistMembers, createMemberItem(whitelistMetaId, member.Address, member.Timestamp))
					break
				}
			}
		}
	}

	// Convert regular members to response format
	var memberItems []*respond.GroupMemberItem
	for _, member := range members {
		memberItem := createMemberItem(member.MetaId, member.Address, member.Timestamp)
		memberItems = append(memberItems, memberItem)
	}

	perfStats.responseFormatTime = time.Now().UnixMilli() - t1
	perfStats.totalTime = time.Since(startTime).Milliseconds()

	// Unified performance logging
	cacheStatus := "DB"
	if perfStats.cacheHit {
		cacheStatus = "Cache"
	}

	groupInfoCacheStatus := "DB"
	if perfStats.groupInfoCacheHit {
		groupInfoCacheStatus = "Cache"
	}

	adminCacheStatus := "DB"
	if perfStats.adminCacheHit {
		adminCacheStatus = "Cache"
	}

	blockCacheStatus := "DB"
	if perfStats.blockCacheHit {
		blockCacheStatus = "Cache"
	}

	whitelistCacheStatus := "DB"
	if perfStats.whitelistCacheHit {
		whitelistCacheStatus = "Cache"
	}

	logger.Info("[CHAT_SERVICE][FETCH_GROUP_MEMBER_LIST_V2] Performance Stats - "+
		"Total: %dms, GetMembers: %dms, GetGroupInfo: %dms(%s), GetAdminList: %dms(%s), "+
		"GetBlockList: %dms(%s), GetWhitelist: %dms(%s), ResponseFormat: %dms, Items: %d, Source: %s",
		perfStats.totalTime,
		perfStats.getMembersTime,
		perfStats.getGroupInfoTime,
		groupInfoCacheStatus,
		perfStats.getAdminListTime,
		adminCacheStatus,
		perfStats.getBlockListTime,
		blockCacheStatus,
		perfStats.getWhitelistTime,
		whitelistCacheStatus,
		perfStats.responseFormatTime,
		len(memberItems),
		cacheStatus)

	return &respond.GroupMemberResponse{
		Total:     total,
		Creator:   creator,
		Admins:    admins,
		List:      memberItems,
		BlockList: blockListMembers,
		WhiteList: whitelistMembers,
	}, nil
}

// FetchGroupPerson Get group member info
func FetchGroupPerson(req *request.FetchGroupPersonRequest) (*respond.GroupPersonResponse, error) {
	// Parameter validation
	if req.MetaId == "" {
		return nil, fmt.Errorf("metaId is empty")
	}
	if req.GroupId == "" {
		return nil, fmt.Errorf("groupId is empty")
	}

	// Get group member info
	person, err := groupDB.GetGroupPersonByGroupIdAndMetaId(req.GroupId, req.MetaId)
	if err != nil {
		return nil, err
	}

	// Build response
	response := &respond.GroupPersonResponse{
		IsInGroup: false,
		Person:    nil,
	}

	if person != nil {
		response.IsInGroup = person.GroupState == models.RoomStateIn
		response.Person = &respond.GroupPersonItem{
			GroupIdMetaIdHash: person.GroupIdMetaIdHash,
			GroupId:           person.GroupId,
			MetaId:            person.MetaId,
			Address:           person.Address,
			UserInfo:          common_service.FetchMetaIDUserInfo(person.Address),
			AvatarTxId:        person.AvatarTxId,
			UserName:          person.UserName,
			UserNickName:      person.UserNickName,
			GroupState:        int64(person.GroupState),
			Timestamp:         person.Timestamp,
			BlockHeight:       person.BlockHeight,
			PinId:             person.PinId,
		}
	}

	return response, nil
}

// FetchLatestChatInfoList Get latest chat info list (group chat + private chat)
func FetchLatestChatInfoList(req *request.FetchLatestChatInfoListRequest) (*respond.ChatInfoResponse, error) {
	// Set default pagination parameters
	if req.Size <= 0 {
		req.Size = 20
	}
	if req.Cursor <= 0 {
		req.Cursor = 0
	}

	// Performance monitoring: record start time
	startTime := time.Now()
	var perfStats = struct {
		contextListTime    int64
		groupInfoCacheTime int64
		groupInfoDBTime    int64
		latestChatTime     int64
		memberCountTime    int64
		chatIndexTime      int64
		privateChatTime    int64
		responseFormatTime int64
		totalTime          int64
	}{}

	// Get user's context list (group chat + private chat)
	t := time.Now().UnixMilli()
	contextList, err := chatDB.GetMetaIdContextList(req.MetaId)
	if err != nil {
		return nil, err
	}
	perfStats.contextListTime = time.Now().UnixMilli() - t

	// Convert to response format
	t = time.Now().UnixMilli()
	var chatInfoItems []*respond.ChatInfoItem
	for _, item := range contextList.Items {
		chatInfoItem := &respond.ChatInfoItem{
			Type:             item.Type,
			GroupId:          item.GroupId,
			MetaId:           item.MetaId,
			Address:          item.Address,
			Timestamp:        item.Timestamp,
			ChatType:         int64(item.ChatType),
			Content:          item.Content,
			CreateMetaId:     item.CreateMetaId,
			CreateAddress:    item.CreateAddress,
			LastMessagePinId: item.LastMessagePinId,
			BlockHeight:      item.BlockHeight,
		}

		// Handle different fields based on type
		if item.Type == "1" || item.Type == "" {
			item.Type = "1"
			t1 := time.Now().UnixMilli()
			group, found := cache_service.GetGroupInfoFromCache(item.GroupId)
			if !found {
				// Cache miss, get from database
				var err error
				group, err = groupDB.GetGroupInfoByGroupId(item.GroupId)
				if err != nil {
					return nil, err
				}
				if group == nil {
					logger.Info("[GROUP_SERVICE][FETCH_LATEST_CHAT_INFO_LIST] Group not found for groupId %s", item.GroupId)
					continue
					// return nil, nil
				}
				perfStats.groupInfoDBTime += time.Now().UnixMilli() - t1

				// Update cache with the fetched data
				cache_service.SetGroupInfoToCache(item.GroupId, group)
			} else {
				perfStats.groupInfoCacheTime += time.Now().UnixMilli() - t1
			}

			t2 := time.Now().UnixMilli()
			// Get group's latest chat info
			latestChat, err := chatDB.GetGroupLatestChat(item.GroupId)
			if err != nil {
				// If failed to get, use default value
				latestChat = nil
			}
			perfStats.latestChatTime += time.Now().UnixMilli() - t2

			t3 := time.Now().UnixMilli()
			// Get group member count
			userCount, err := groupDB.GetGroupMemberCountFromList(item.GroupId)
			if err != nil {
				// If failed to get, use default value
				userCount = 0
			}
			perfStats.memberCountTime += time.Now().UnixMilli() - t3

			// Fill group chat specific fields
			chatInfoItem.CommunityId = group.CommunityId
			chatInfoItem.RoomName = group.RoomName
			chatInfoItem.RoomNote = group.RoomNote
			chatInfoItem.RoomIcon = group.RoomIcon
			chatInfoItem.RoomType = group.RoomType
			chatInfoItem.RoomStatus = group.RoomStatus
			chatInfoItem.RoomJoinType = group.RoomJoinType
			chatInfoItem.RoomAvatarUrl = group.RoomAvatarUrl
			chatInfoItem.CreateUserMetaId = group.CreateUserMetaId
			chatInfoItem.CreateUserAddress = group.CreateUserAddress
			chatInfoItem.CreateUserInfo = common_service.FetchMetaIDUserInfo(group.CreateUserAddress)
			chatInfoItem.UserCount = userCount
			chatInfoItem.ChatSettingType = group.ChatSettingType
			chatInfoItem.DeleteStatus = group.DeleteStatus
			chatInfoItem.Path = group.Path
			chatInfoItem.Chain = group.Chain

			// If latest chat info is obtained, update related fields
			if latestChat != nil {
				chatInfoItem.Content = latestChat.Content
				chatInfoItem.Mention = latestChat.Mention
				chatInfoItem.LastMessagePinId = latestChat.LastMessagePinId
				chatInfoItem.Timestamp = latestChat.Timestamp
				chatInfoItem.ChatType = int64(latestChat.ChatType)
				chatInfoItem.CreateMetaId = latestChat.MetaId
				chatInfoItem.CreateAddress = latestChat.CreateAddress
				chatInfoItem.BlockHeight = latestChat.BlockHeight
				chatInfoItem.UserInfo = common_service.FetchMetaIDUserInfo(latestChat.CreateAddress)

				t4 := time.Now().UnixMilli()
				// Get group chat index
				chatInfoItem.Index = -1
				chatInfo, _ := chatDB.GetChatByPinId(latestChat.LastMessagePinId)
				if chatInfo != nil {
					chatInfoItem.Index = chatInfo.Index
					chatInfoItem.Version = chatInfo.Version
				}
				perfStats.chatIndexTime += time.Now().UnixMilli() - t4
			}
		} else if item.Type == "2" {
			t5 := time.Now().UnixMilli()
			// Private chat type, get latest private chat message
			latestPrivateChat, err := privateDB.GetPrivateChatByPinId(item.LastMessagePinId)
			if err != nil {
				// If failed to get, use default value
				latestPrivateChat = nil
			}
			perfStats.privateChatTime += time.Now().UnixMilli() - t5

			// If latest private chat info is obtained, update related fields
			if latestPrivateChat != nil {
				chatInfoItem.Content = latestPrivateChat.Content
				chatInfoItem.LastMessagePinId = latestPrivateChat.PinId
				chatInfoItem.Timestamp = latestPrivateChat.Timestamp
				chatInfoItem.ChatType = int64(latestPrivateChat.ChatType)
				chatInfoItem.CreateMetaId = latestPrivateChat.From
				chatInfoItem.CreateAddress = latestPrivateChat.FromAddress
				chatInfoItem.BlockHeight = latestPrivateChat.BlockHeight
				chatInfoItem.Chain = latestPrivateChat.Chain
				if item.Address != "" {
					chatInfoItem.UserInfo = common_service.FetchMetaIDUserInfo(item.Address)
				} else if item.MetaId != "" {
					chatInfoItem.UserInfo = common_service.FetchMetaIDUserInfoInfoByMetaId(item.MetaId)
					if chatInfoItem.UserInfo != nil {
						chatInfoItem.Address = chatInfoItem.UserInfo.Address
					}
				} else {
					chatInfoItem.UserInfo = nil
				}
				// Get private chat index
				chatInfoItem.Index = latestPrivateChat.Index
				chatInfoItem.Version = latestPrivateChat.Version
			}
		}

		chatInfoItems = append(chatInfoItems, chatInfoItem)
	}
	perfStats.responseFormatTime = time.Now().UnixMilli() - t
	perfStats.totalTime = time.Since(startTime).Milliseconds()

	// Unified performance logging
	logger.Info("[GROUP_SERVICE][FETCH_LATEST_CHAT_INFO_LIST] Performance Stats - "+
		"Total: %dms, ContextList: %dms, GroupInfoCache: %dms, GroupInfoDB: %dms, "+
		"LatestChat: %dms, MemberCount: %dms, ChatIndex: %dms, PrivateChat: %dms, "+
		"ResponseFormat: %dms, Items: %d",
		perfStats.totalTime,
		perfStats.contextListTime,
		perfStats.groupInfoCacheTime,
		perfStats.groupInfoDBTime,
		perfStats.latestChatTime,
		perfStats.memberCountTime,
		perfStats.chatIndexTime,
		perfStats.privateChatTime,
		perfStats.responseFormatTime,
		len(chatInfoItems))

	return &respond.ChatInfoResponse{
		Total: int64(len(chatInfoItems)),
		List:  chatInfoItems,
	}, nil
}

// FetchPrivateChatList Get private chat record list
func FetchPrivateChatList(req *request.FetchPrivateChatListRequest) (*respond.PrivateChatResponse, error) {
	// Set default pagination parameters
	if req.Size <= 0 {
		req.Size = 20
	}

	var chats []*models.TalkPrivateChatV3
	var nextTimestamp int64
	var err error

	if req.Timestamp > 0 {
		// Get private chat records by timestamp range
		chats, nextTimestamp, err = privateDB.GetPrivateChatsByMetaIdsAndTimestampRange(req.MetaId, req.OtherMetaId, req.Timestamp, req.Size)
	} else {
		// Get latest private chat records
		chats, nextTimestamp, err = privateDB.GetLatestPrivateChatsByMetaIds(req.MetaId, req.OtherMetaId, req.Size)
	}

	if err != nil {
		return nil, err
	}
	// logger.Info("private chats: %+v\n", chats)

	// Convert to response format
	var chatItems []*respond.PrivateChatItem

	for _, chat := range chats {
		chatItem := &respond.PrivateChatItem{
			From:         chat.From,
			FromUserInfo: common_service.FetchMetaIDUserInfoInfoByMetaId(chat.From),
			To:           chat.To,
			ToUserInfo:   common_service.FetchMetaIDUserInfoInfoByMetaId(chat.To),
			TxId:         chat.TxId,
			PinId:        chat.PinId,
			MetaId:       chat.From, // Message creator MetaId
			Address:      chat.FromAddress,
			UserInfo:     common_service.FetchMetaIDUserInfo(chat.FromAddress),
			NickName:     "", // Need to get from user info
			Protocol:     chat.Protocol,
			Content:      chat.Content,
			ContentType:  chat.ContentType,
			Encryption:   chat.Encryption,
			Version:      chat.Version,
			ChatType:     int64(chat.ChatType),
			ReplyPin:     chat.ReplyPin,
			ReplyInfo:    nil,
			ReplyMetaId:  "",
			Timestamp:    chat.Timestamp,
			Chain:        chat.Chain,
			BlockHeight:  chat.BlockHeight,
			Index:        chat.Index,
		}

		// Handle reply message
		if chat.ReplyPin != "" {
			replyChat, _ := privateDB.GetPrivateChatByPinId(chat.ReplyPin)
			if replyChat != nil {
				chatItem.ReplyInfo = &respond.ReplyInfo{
					PinId:       replyChat.PinId,
					MetaId:      replyChat.From,
					Address:     replyChat.FromAddress,
					UserInfo:    common_service.FetchMetaIDUserInfo(replyChat.FromAddress),
					NickName:    "",
					Protocol:    replyChat.Protocol,
					Content:     replyChat.Content,
					ContentType: replyChat.ContentType,
					Encryption:  replyChat.Encryption,
					Version:     replyChat.Version,
					ChatType:    replyChat.ChatType,
					Timestamp:   replyChat.Timestamp,
					Chain:       replyChat.Chain,
					Index:       replyChat.Index,
				}
				if chatItem.ReplyInfo.Address == "" && chatItem.ReplyInfo.MetaId != "" {
					chatItem.ReplyInfo.UserInfo = common_service.FetchMetaIDUserInfoInfoByMetaId(chatItem.ReplyInfo.MetaId)
					if chatItem.ReplyInfo.UserInfo != nil {
						chatItem.ReplyInfo.Address = chatItem.ReplyInfo.UserInfo.Address
					}
				}

				chatItem.ReplyMetaId = replyChat.From
			}
		}

		chatItems = append(chatItems, chatItem)
	}

	return &respond.PrivateChatResponse{
		Total:         int64(len(chatItems)),
		NextTimestamp: nextTimestamp,
		List:          chatItems,
	}, nil
}

// FetchPrivateChatListByIndex Get private chat list by index range (ascending order)
func FetchPrivateChatListByIndex(req *request.FetchPrivateChatListByIndexRequest) (*respond.PrivateChatResponse, error) {
	// Set default pagination parameters
	if req.Size <= 0 {
		req.Size = 20
	}

	// Performance monitoring: record start time
	startTime := time.Now()
	var perfStats = struct {
		getChatsTime       int64
		responseFormatTime int64
		userInfoTime       int64
		totalTime          int64
	}{}

	var chats []*models.TalkPrivateChatV3
	var lastIndex int64
	var err error

	t := time.Now().UnixMilli()
	chats, lastIndex, err = privateDB.GetPrivateChatsByMetaIdsAndStartIndexRange(req.MetaId, req.OtherMetaId, req.StartIndex, req.Size)
	perfStats.getChatsTime = time.Now().UnixMilli() - t

	if err != nil {
		return nil, err
	}

	// Convert to response format
	var chatItems []*respond.PrivateChatItem

	t1 := time.Now().UnixMilli()
	for _, chat := range chats {
		chatItem := &respond.PrivateChatItem{
			From:         chat.From,
			FromUserInfo: common_service.FetchMetaIDUserInfoInfoByMetaId(chat.From),
			To:           chat.To,
			ToUserInfo:   common_service.FetchMetaIDUserInfoInfoByMetaId(chat.To),
			TxId:         chat.TxId,
			PinId:        chat.PinId,
			MetaId:       chat.From, // Message creator MetaId
			Address:      chat.FromAddress,
			UserInfo:     common_service.FetchMetaIDUserInfo(chat.FromAddress),
			NickName:     "", // Need to get from user info
			Protocol:     chat.Protocol,
			Content:      chat.Content,
			ContentType:  chat.ContentType,
			Encryption:   chat.Encryption,
			Version:      chat.Version,
			ChatType:     int64(chat.ChatType),
			ReplyPin:     chat.ReplyPin,
			ReplyInfo:    nil,
			ReplyMetaId:  "",
			Timestamp:    chat.Timestamp,
			Chain:        chat.Chain,
			BlockHeight:  chat.BlockHeight,
			Index:        chat.Index,
		}

		// Handle reply information
		if chat.ReplyPin != "" {
			replyChat, _ := privateDB.GetPrivateChatByPinId(chat.ReplyPin)
			if replyChat != nil {
				chatItem.ReplyInfo = &respond.ReplyInfo{
					PinId:       replyChat.PinId,
					MetaId:      replyChat.From,
					Address:     replyChat.FromAddress,
					NickName:    "", // Private chat doesn't have NickName field
					Protocol:    replyChat.Protocol,
					Content:     replyChat.Content,
					ContentType: replyChat.ContentType,
					Encryption:  replyChat.Encryption,
					Version:     replyChat.Version,
					ChatType:    replyChat.ChatType,
					Timestamp:   replyChat.Timestamp,
					Chain:       replyChat.Chain,
					Index:       replyChat.Index,
				}
				if chatItem.ReplyInfo.Address == "" && chatItem.ReplyInfo.MetaId != "" {
					chatItem.ReplyInfo.UserInfo = common_service.FetchMetaIDUserInfoInfoByMetaId(chatItem.ReplyInfo.MetaId)
					if chatItem.ReplyInfo.UserInfo != nil {
						chatItem.ReplyInfo.Address = chatItem.ReplyInfo.UserInfo.Address
					}
				}

				chatItem.ReplyMetaId = replyChat.From
			}
		}

		chatItems = append(chatItems, chatItem)
	}
	perfStats.responseFormatTime = time.Now().UnixMilli() - t1

	// Performance logging
	perfStats.totalTime = time.Now().UnixMilli() - startTime.UnixMilli()
	if perfStats.totalTime > 1000 { // Log if total time > 1 second
		log.Printf("[PERF] FetchPrivateChatListByIndex - Total: %dms, GetChats: %dms, Format: %dms, UserInfo: %dms",
			perfStats.totalTime, perfStats.getChatsTime, perfStats.responseFormatTime, perfStats.userInfoTime)
	}

	return &respond.PrivateChatResponse{
		Total:         int64(len(chatItems)),
		NextTimestamp: lastIndex, // Use lastIndex as NextTimestamp for index-based pagination
		List:          chatItems,
	}, nil
}

func wsForGroupChatItem(chat *models.TalkGroupChatV3) error {
	wsPostGroupMsg(chat)
	return nil
}

func wsForPrivateChatItem(chat *models.TalkPrivateChatV3) error {
	wsPostPrivateMsg(chat)
	return nil
}

func wsForGroupRoleInfoChange(roleInfo *models.GroupUserRoleInfo) error {
	wsPostGroupRoleInfo(roleInfo)
	return nil
}

// GetUserInfoByAddress Get user information by address
func GetUserInfoByAddress(address string) (*respond.UserInfoResponse, error) {
	if address == "" {
		return nil, fmt.Errorf("address is empty")
	}

	userInfo := common_service.FetchMetaIDUserInfo(address)
	if userInfo == nil {
		return nil, fmt.Errorf("user info not found for address: %s", address)
	}

	if userInfo.ChatPublicKey == "" {
		chatPublicKeyInfo, _ := userInfoDB.GetLatestValidUserInfoByAddress(address)
		if chatPublicKeyInfo != nil {
			userInfo.ChatPublicKey = chatPublicKeyInfo.ChatPublicKey
			userInfo.ChatPublicKeyId = chatPublicKeyInfo.ChatPublicKeyId
		}
	} else {
		if userInfo.ChatPublicKeyId == "" {
			chatPublicKeyInfo, _ := userInfoDB.GetLatestValidUserInfoByAddress(address)
			if chatPublicKeyInfo != nil {
				if chatPublicKeyInfo.ChatPublicKey != "" && chatPublicKeyInfo.ChatPublicKey == userInfo.ChatPublicKey {
					userInfo.ChatPublicKey = chatPublicKeyInfo.ChatPublicKey
					userInfo.ChatPublicKeyId = chatPublicKeyInfo.ChatPublicKeyId
				}
			}
		}
	}

	return &respond.UserInfoResponse{
		Address:  address,
		MetaId:   userInfo.Metaid,
		UserInfo: userInfo,
	}, nil
}

// GetUserInfoByMetaId Get user information by metaId
func GetUserInfoByMetaId(metaId string) (*respond.UserInfoResponse, error) {
	if metaId == "" {
		return nil, fmt.Errorf("metaId is empty")
	}

	userInfo := common_service.FetchMetaIDUserInfoInfoByMetaId(metaId)
	if userInfo == nil {
		return nil, fmt.Errorf("user info not found for metaId: %s", metaId)
	}

	if userInfo.ChatPublicKey == "" {
		chatPublicKeyInfo, _ := userInfoDB.GetLatestValidUserInfoByMetaId(metaId)
		if chatPublicKeyInfo != nil {
			userInfo.ChatPublicKey = chatPublicKeyInfo.ChatPublicKey
			userInfo.ChatPublicKeyId = chatPublicKeyInfo.ChatPublicKeyId
		}
	} else {
		if userInfo.ChatPublicKeyId == "" {
			chatPublicKeyInfo, _ := userInfoDB.GetLatestValidUserInfoByMetaId(metaId)
			if chatPublicKeyInfo != nil {
				if chatPublicKeyInfo.ChatPublicKey != "" && chatPublicKeyInfo.ChatPublicKey == userInfo.ChatPublicKey {
					userInfo.ChatPublicKey = chatPublicKeyInfo.ChatPublicKey
					userInfo.ChatPublicKeyId = chatPublicKeyInfo.ChatPublicKeyId
				}
			}
		}
	}

	return &respond.UserInfoResponse{
		MetaId:   metaId,
		Address:  userInfo.Address,
		UserInfo: userInfo,
	}, nil
}

// GetBatchUserInfo Get user information by addresses or metaIds
func GetBatchUserInfo(addresses []string, metaIds []string) (*respond.BatchUserInfoResponse, error) {
	if len(addresses) == 0 && len(metaIds) == 0 {
		return nil, fmt.Errorf("both addresses and metaIds are empty")
	}

	// limit total count to 100
	totalCount := len(addresses) + len(metaIds)
	if totalCount > 100 {
		return nil, fmt.Errorf("total count exceeds maximum limit of 100, got %d", totalCount)
	}

	var userInfoItems []*respond.UserInfoResponse
	var errors []string

	// Process addresses
	for _, address := range addresses {
		if address == "" {
			continue
		}

		userInfo := common_service.FetchMetaIDUserInfo(address)
		if userInfo == nil {
			errors = append(errors, fmt.Sprintf("user info not found for address: %s", address))
			continue
		}

		// Get chat public key if not available
		if userInfo.ChatPublicKey == "" {
			chatPublicKeyInfo, _ := userInfoDB.GetLatestValidUserInfoByAddress(address)
			if chatPublicKeyInfo != nil {
				userInfo.ChatPublicKey = chatPublicKeyInfo.ChatPublicKey
				userInfo.ChatPublicKeyId = chatPublicKeyInfo.ChatPublicKeyId
			}
		} else {
			if userInfo.ChatPublicKeyId == "" {
				chatPublicKeyInfo, _ := userInfoDB.GetLatestValidUserInfoByAddress(address)
				if chatPublicKeyInfo != nil {
					if chatPublicKeyInfo.ChatPublicKey != "" && chatPublicKeyInfo.ChatPublicKey == userInfo.ChatPublicKey {
						userInfo.ChatPublicKey = chatPublicKeyInfo.ChatPublicKey
						userInfo.ChatPublicKeyId = chatPublicKeyInfo.ChatPublicKeyId
					}
				}
			}
		}

		userInfoItems = append(userInfoItems, &respond.UserInfoResponse{
			Address:  address,
			MetaId:   userInfo.Metaid,
			UserInfo: userInfo,
		})
	}

	// Process metaIds
	for _, metaId := range metaIds {
		if metaId == "" {
			continue
		}

		userInfo := common_service.FetchMetaIDUserInfoInfoByMetaId(metaId)
		if userInfo == nil {
			errors = append(errors, fmt.Sprintf("user info not found for metaId: %s", metaId))
			continue
		}

		// Get chat public key if not available
		if userInfo.ChatPublicKey == "" {
			chatPublicKeyInfo, _ := userInfoDB.GetLatestValidUserInfoByMetaId(metaId)
			if chatPublicKeyInfo != nil {
				userInfo.ChatPublicKey = chatPublicKeyInfo.ChatPublicKey
				userInfo.ChatPublicKeyId = chatPublicKeyInfo.ChatPublicKeyId
			}
		} else {
			if userInfo.ChatPublicKeyId == "" {
				chatPublicKeyInfo, _ := userInfoDB.GetLatestValidUserInfoByMetaId(metaId)
				if chatPublicKeyInfo != nil {
					if chatPublicKeyInfo.ChatPublicKey != "" && chatPublicKeyInfo.ChatPublicKey == userInfo.ChatPublicKey {
						userInfo.ChatPublicKey = chatPublicKeyInfo.ChatPublicKey
						userInfo.ChatPublicKeyId = chatPublicKeyInfo.ChatPublicKeyId
					}
				}
			}
		}

		userInfoItems = append(userInfoItems, &respond.UserInfoResponse{
			MetaId:   metaId,
			Address:  userInfo.Address,
			UserInfo: userInfo,
		})
	}

	return &respond.BatchUserInfoResponse{
		Total:  int64(len(userInfoItems)),
		List:   userInfoItems,
		Errors: errors,
	}, nil
}

// GetCurrentMaxGroupChatIndex Get current maximum index for a group
func GetCurrentMaxGroupChatIndex(groupId string) (*respond.MaxIndexResponse, error) {
	if groupId == "" {
		return nil, fmt.Errorf("groupId is empty")
	}

	maxIndex, err := chatDB.GetCurrentMaxGroupChatIndex(groupId)
	if err != nil {
		return nil, err
	}

	return &respond.MaxIndexResponse{
		GroupId:  groupId,
		MaxIndex: maxIndex,
	}, nil
}

// GetCurrentMaxGroupChannelChatIndex Get current maximum index for a group channel
func GetCurrentMaxGroupChannelChatIndex(channelId string) (*respond.MaxIndexResponse, error) {
	if channelId == "" {
		return nil, fmt.Errorf("channelId is empty")
	}

	maxIndex, err := chatDB.GetCurrentMaxChannelChatIndex(channelId)
	if err != nil {
		return nil, err
	}

	return &respond.MaxIndexResponse{
		ChannelId: channelId,
		MaxIndex:  maxIndex,
	}, nil
}

// GetCurrentMaxPrivateChatIndex Get current maximum index for a private conversation
func GetCurrentMaxPrivateChatIndex(fromMetaId, toMetaId string) (*respond.MaxIndexResponse, error) {
	if fromMetaId == "" {
		return nil, fmt.Errorf("fromMetaId is empty")
	}
	if toMetaId == "" {
		return nil, fmt.Errorf("toMetaId is empty")
	}

	maxIndex, err := privateDB.GetCurrentMaxPrivateChatIndex(fromMetaId, toMetaId)
	if err != nil {
		return nil, err
	}

	return &respond.MaxIndexResponse{
		FromMetaId: fromMetaId,
		ToMetaId:   toMetaId,
		MaxIndex:   maxIndex,
	}, nil
}

// FetchGroupChatListByIndex Get group chat list by index range (ascending order)
func FetchGroupChatListByIndex(req *request.FetchGroupChatListByIndexRequest) (*respond.GroupChatResponse, error) {
	// Set default pagination parameters
	if req.Size <= 0 {
		req.Size = 20
	}

	// Performance monitoring: record start time
	startTime := time.Now()
	var perfStats = struct {
		getChatsTime       int64
		responseFormatTime int64
		userInfoTime       int64
		totalTime          int64
	}{}

	var chats []*models.TalkGroupChatV3
	var lastIndex int64
	var err error

	t := time.Now().UnixMilli()
	chats, lastIndex, err = chatDB.GetChatsByGroupIdAndStartIndexRange(req.GroupId, req.StartIndex, req.Size)
	perfStats.getChatsTime = time.Now().UnixMilli() - t

	if err != nil {
		return nil, err
	}

	// Convert to response format
	var chatItems []*respond.GroupChatItem

	t1 := time.Now().UnixMilli()
	for _, chat := range chats {
		chatItem := &respond.GroupChatItem{
			GroupId:     chat.GroupId,
			MetanetId:   chat.GroupId, // Use GroupId as MetanetId
			TxId:        chat.TxId,
			PinId:       chat.PinId,
			Address:     chat.Address,
			MetaId:      chat.MetaId,
			NickName:    "", // Need to get from user info
			Protocol:    chat.Protocol,
			Content:     chat.Content,
			ContentType: chat.ContentType,
			Encryption:  chat.Encryption,
			ChatType:    chat.ChatType,
			ReplyPin:    chat.ReplyPin,
			ReplyInfo:   nil,
			ReplyMetaId: "",
			Mention:     chat.Mention,
			Timestamp:   chat.Timestamp,
			Chain:       chat.Chain,
			BlockHeight: chat.BlockHeight,
			Index:       chat.Index,
		}
		if strings.Contains(strings.ToLower(chatItem.Protocol), strings.ToLower(protocols.MonitorSimpleGroupOpenLuckyBag)) {
			openLuckyBag, _ := chatDB.GetOpenLuckyBagByPinId(chat.PinId)
			if openLuckyBag != nil {
				if openLuckyBag.GrabState == models.GrabStateOpenAndSend {
					chatItem.TxId = openLuckyBag.GrabTxId
				} else {
					chatItem.TxId = ""
				}
			}
		} else if strings.Contains(strings.ToLower(chatItem.Protocol), strings.ToLower(protocols.MonitorSimpleGroupLuckyBag)) {
			luckyBag, _ := chatDB.GetLuckyBagByPinIdFromCache(chat.GroupId, chat.PinId)
			if luckyBag != nil {
				chatItem.Domain = luckyBag.Domain
			}
		}
		if chat.ReplyPin != "" {
			replyChat, _ := chatDB.GetChatByPinId(chat.ReplyPin)
			if replyChat != nil {
				chatItem.ReplyInfo = &respond.ReplyInfo{
					PinId:       replyChat.PinId,
					MetaId:      replyChat.MetaId,
					Address:     replyChat.Address,
					NickName:    replyChat.NickName,
					Protocol:    replyChat.Protocol,
					Content:     replyChat.Content,
					ContentType: replyChat.ContentType,
					Encryption:  replyChat.Encryption,
					ChatType:    replyChat.ChatType,
					Timestamp:   replyChat.Timestamp,
					Mention:     replyChat.Mention,
					Chain:       replyChat.Chain,
					Index:       replyChat.Index,
				}
				chatItem.ReplyMetaId = replyChat.MetaId
				chatItem.BlockHeight = replyChat.BlockHeight
			}
		}

		chatItems = append(chatItems, chatItem)
	}
	perfStats.responseFormatTime = time.Now().UnixMilli() - t1

	//get user info
	t2 := time.Now().UnixMilli()
	for _, chatItem := range chatItems {
		if chatItem.ReplyInfo != nil {
			chatItem.ReplyInfo.UserInfo = common_service.FetchMetaIDUserInfo(chatItem.ReplyInfo.Address)
		}
		chatItem.UserInfo = common_service.FetchMetaIDUserInfo(chatItem.Address)
	}
	perfStats.userInfoTime = time.Now().UnixMilli() - t2
	perfStats.totalTime = time.Since(startTime).Milliseconds()

	// Unified performance logging
	logger.Info("[CHAT_SERVICE][FETCH_GROUP_CHAT_LIST_BY_INDEX] Performance Stats - "+
		"Total: %dms, GetChats: %dms, ResponseFormat: %dms, UserInfo: %dms, Items: %d",
		perfStats.totalTime,
		perfStats.getChatsTime,
		perfStats.responseFormatTime,
		perfStats.userInfoTime,
		len(chatItems))

	return &respond.GroupChatResponse{
		Total:     int64(len(chatItems)),
		LastIndex: lastIndex,
		List:      chatItems,
	}, nil
}

// FetchGroupChatListByStartTime Get group chat list by start timestamp range (ascending order)
func FetchGroupChatListByStartTime(req *request.FetchGroupChatListByStartTimeRequest) (*respond.GroupChatResponse, error) {
	// Set default pagination parameters
	if req.Size <= 0 {
		req.Size = 20
	}

	// Performance monitoring: record start time
	startTime := time.Now()
	var perfStats = struct {
		getChatsTime       int64
		responseFormatTime int64
		userInfoTime       int64
		totalTime          int64
	}{}

	var chats []*models.TalkGroupChatV3
	var lastTimestamp int64
	var err error

	t := time.Now().UnixMilli()
	chats, lastTimestamp, err = chatDB.GetChatsByGroupIdAndStartTimestampRange(req.GroupId, req.StartTimestamp, req.Size)
	perfStats.getChatsTime = time.Now().UnixMilli() - t

	if err != nil {
		return nil, err
	}

	// Convert to response format
	var chatItems []*respond.GroupChatItem

	t1 := time.Now().UnixMilli()
	for _, chat := range chats {
		chatItem := &respond.GroupChatItem{
			GroupId:     chat.GroupId,
			MetanetId:   chat.GroupId, // Use GroupId as MetanetId
			TxId:        chat.TxId,
			PinId:       chat.PinId,
			Address:     chat.Address,
			MetaId:      chat.MetaId,
			NickName:    "", // Need to get from user info
			Protocol:    chat.Protocol,
			Content:     chat.Content,
			ContentType: chat.ContentType,
			Encryption:  chat.Encryption,
			ChatType:    chat.ChatType,
			ReplyPin:    chat.ReplyPin,
			ReplyInfo:   nil,
			ReplyMetaId: "",
			Mention:     chat.Mention,
			Timestamp:   chat.Timestamp,
			Chain:       chat.Chain,
			BlockHeight: chat.BlockHeight,
			Index:       chat.Index,
		}
		if strings.Contains(strings.ToLower(chatItem.Protocol), strings.ToLower(protocols.MonitorSimpleGroupOpenLuckyBag)) {
			openLuckyBag, _ := chatDB.GetOpenLuckyBagByPinId(chat.PinId)
			if openLuckyBag != nil {
				if openLuckyBag.GrabState == models.GrabStateOpenAndSend {
					chatItem.TxId = openLuckyBag.GrabTxId
				} else {
					chatItem.TxId = ""
				}
			}
		}
		if strings.Contains(strings.ToLower(chatItem.Protocol), strings.ToLower(protocols.MonitorSimpleGroupLuckyBag)) {
			luckyBag, _ := chatDB.GetLuckyBagByPinIdFromCache(chat.GroupId, chat.PinId)
			if luckyBag != nil {
				chatItem.Domain = luckyBag.Domain
			}
		}
		if chat.ReplyPin != "" {
			replyChat, err := chatDB.GetChatByPinId(chat.ReplyPin)
			if err != nil {
				replyChat = nil
			}
			chatItem.ReplyInfo = &respond.ReplyInfo{
				PinId:       replyChat.PinId,
				MetaId:      replyChat.MetaId,
				Address:     replyChat.Address,
				NickName:    replyChat.NickName,
				Protocol:    replyChat.Protocol,
				Content:     replyChat.Content,
				ContentType: replyChat.ContentType,
				Encryption:  replyChat.Encryption,
				ChatType:    replyChat.ChatType,
				Timestamp:   replyChat.Timestamp,
				Mention:     replyChat.Mention,
				Chain:       replyChat.Chain,
				Index:       replyChat.Index,
			}
			chatItem.ReplyMetaId = replyChat.MetaId
			chatItem.BlockHeight = replyChat.BlockHeight
		}

		chatItems = append(chatItems, chatItem)
	}
	perfStats.responseFormatTime = time.Now().UnixMilli() - t1

	//get user info
	t2 := time.Now().UnixMilli()
	for _, chatItem := range chatItems {
		if chatItem.ReplyInfo != nil {
			chatItem.ReplyInfo.UserInfo = common_service.FetchMetaIDUserInfo(chatItem.ReplyInfo.Address)
		}
		chatItem.UserInfo = common_service.FetchMetaIDUserInfo(chatItem.Address)
	}
	perfStats.userInfoTime = time.Now().UnixMilli() - t2
	perfStats.totalTime = time.Since(startTime).Milliseconds()

	// Unified performance logging
	logger.Info("[CHAT_SERVICE][FETCH_GROUP_CHAT_LIST_BY_START_TIME] Performance Stats - "+
		"Total: %dms, GetChats: %dms, ResponseFormat: %dms, UserInfo: %dms, Items: %d",
		perfStats.totalTime,
		perfStats.getChatsTime,
		perfStats.responseFormatTime,
		perfStats.userInfoTime,
		len(chatItems))

	return &respond.GroupChatResponse{
		Total:         int64(len(chatItems)),
		LastTimestamp: lastTimestamp,
		List:          chatItems,
	}, nil
}

// SearchGroupsByNameOrId searches groups by name or ID
func SearchGroupsByNameOrId(req *request.SearchGroupRequest) (*respond.GroupSearchResponse, error) {
	// Set default pagination parameters
	if req.Size <= 0 {
		req.Size = 20
	}

	// Search using GroupDB
	results, err := groupDB.SearchGroups(req.Query, int(req.Size), 10)
	if err != nil {
		return nil, err
	}
	// Convert to response format
	var groupItems []*respond.GroupSearchItem
	for _, result := range results {

		// Get group member count
		// userCount, err := groupDB.GetGroupMemberCount(result.GroupId)
		userCount, err := groupDB.GetGroupMemberCountFromList(result.GroupId)
		if err != nil {
			// If failed to get, use default value
			userCount = 0
		}

		groupItem := &respond.GroupSearchItem{
			GroupId:     result.GroupId,
			GroupName:   result.GroupName,
			GroupIcon:   result.GroupIcon,
			PinId:       result.PinId,
			Timestamp:   result.Timestamp,
			MemberCount: userCount,
		}
		groupItems = append(groupItems, groupItem)
	}

	return &respond.GroupSearchResponse{
		Total: int64(len(groupItems)),
		List:  groupItems,
	}, nil
}

// SearchGroupsAndUserByNameOrId searches both groups and users by name or ID
func SearchGroupsAndUserByNameOrId(req *request.SearchGroupAndUserRequest) (*respond.GroupAndUserSearchResponse, error) {
	// Set default pagination parameters
	if req.Size <= 0 {
		req.Size = 5
	}

	var allResults []*respond.GroupAndUserSearchItem

	// Search users
	userResults, err := common_service.SearchAllMetaIDUserInfoInfo(req.Query, 0)
	if err != nil {
		// Log error but continue
		logger.Info("Failed to search users: %v\n", err)
	} else {
		// Convert user results to combined format
		for _, result := range userResults {
			userItem := &respond.GroupAndUserSearchItem{
				Type:      "user",
				MetaId:    result.Metaid,
				Address:   result.Address,
				UserName:  result.Name,
				Avatar:    result.Avatar,
				AvatarId:  result.AvatarId,
				Timestamp: 0, // User search doesn't provide timestamp, use 0
			}
			allResults = append(allResults, userItem)
		}
	}

	// Search groups
	groupResults, err := groupDB.SearchGroups(req.Query, int(req.Size), 10)
	if err != nil {
		// Log error but continue with user search
		logger.Info("Failed to search groups: %v\n", err)
	} else {
		// Convert group results to combined format
		for _, result := range groupResults {
			// Get group member count
			// userCount, err := groupDB.GetGroupMemberCount(result.GroupId)
			userCount, err := groupDB.GetGroupMemberCountFromList(result.GroupId)
			if err != nil {
				// If failed to get, use default value
				userCount = 0
			}

			groupItem := &respond.GroupAndUserSearchItem{
				Type:        "group",
				GroupId:     result.GroupId,
				GroupName:   result.GroupName,
				GroupIcon:   result.GroupIcon,
				PinId:       result.PinId,
				MemberCount: userCount,
				Timestamp:   result.Timestamp,
			}
			allResults = append(allResults, groupItem)
		}
	}

	// // Sort results by type (groups first, then users) and limit total results
	// var finalResults []*respond.GroupAndUserSearchItem

	// // Add groups first
	// for _, item := range allResults {
	// 	if item.Type == "group" && len(finalResults) < int(req.Size) {
	// 		finalResults = append(finalResults, item)
	// 	}
	// }

	// // Then add users
	// for _, item := range allResults {
	// 	if item.Type == "user" && len(finalResults) < int(req.Size) {
	// 		finalResults = append(finalResults, item)
	// 	}
	// }

	return &respond.GroupAndUserSearchResponse{
		Total: int64(len(allResults)),
		List:  allResults,
	}, nil
}

// SearchUserByNameOrId searches users by name or ID
func SearchUserByNameOrId(req *request.SearchGroupAndUserRequest) (*respond.UserSearchResponse, error) {
	// Set default pagination parameters
	if req.Size <= 0 {
		req.Size = 20
	}

	// Search users
	userResults, err := common_service.SearchAllMetaIDUserInfoInfo(req.Query, int(req.Size))
	if err != nil {
		return nil, fmt.Errorf("failed to search users: %v", err)
	}

	// Convert to response format
	var userItems []*respond.UserSearchItem
	for _, result := range userResults {
		// Get chat public key from database if not available in search result
		chatPublicKey := result.Chatpubkey
		chatPublicKeyId := result.ChatpubkeyId

		// If chat public key is not available, try to get from database
		if chatPublicKey == "" {
			chatPublicKeyInfo, _ := userInfoDB.GetLatestValidUserInfoByMetaId(result.Metaid)
			if chatPublicKeyInfo != nil {
				chatPublicKey = chatPublicKeyInfo.ChatPublicKey
				chatPublicKeyId = chatPublicKeyInfo.ChatPublicKeyId
			}
		} else {
			// If chat public key exists but chat public key id is empty, try to get from database
			if chatPublicKeyId == "" {
				chatPublicKeyInfo, _ := userInfoDB.GetLatestValidUserInfoByMetaId(result.Metaid)
				if chatPublicKeyInfo != nil {
					if chatPublicKeyInfo.ChatPublicKey != "" && chatPublicKeyInfo.ChatPublicKey == chatPublicKey {
						chatPublicKeyId = chatPublicKeyInfo.ChatPublicKeyId
					}
				}
			}
		}

		userItem := &respond.UserSearchItem{
			MetaId:          result.Metaid,
			Address:         result.Address,
			UserName:        result.Name,
			Avatar:          result.Avatar,
			AvatarId:        result.AvatarId,
			ChatPublicKey:   chatPublicKey,
			ChatPublicKeyId: chatPublicKeyId,
			Timestamp:       0, // User search doesn't provide timestamp, use 0
		}
		userItems = append(userItems, userItem)
	}

	return &respond.UserSearchResponse{
		Total: int64(len(userItems)),
		List:  userItems,
	}, nil
}

// GetGroupSearchCacheStats returns group search cache statistics
func GetGroupSearchCacheStats() (map[string]interface{}, error) {
	return groupDB.GetSearchCacheStats(), nil
}

// SearchGroupMembers searches group members by name, metaId, or address
func SearchGroupMembers(req *request.SearchGroupMembersRequest) (*respond.GroupMemberSearchResponse, error) {
	// Set default pagination parameters
	if req.Size <= 0 {
		req.Size = 20
	}

	if req.GroupId == "" {
		return nil, fmt.Errorf("groupId is required")
	}

	if req.Query == "" {
		return nil, fmt.Errorf("search query is required")
	}

	// Get all group members from TalkGroupPersonCollection
	members, err := groupDB.GetGroupPersonList(req.GroupId)
	if err != nil {
		return nil, fmt.Errorf("failed to get group members: %v", err)
	}

	// First, collect all valid members with their user info
	var allMembers []*respond.GroupMemberSearchItem
	queryLower := strings.ToLower(req.Query)

	// Process all members first to get user info
	for _, member := range members {
		// Skip members who are not in the group
		if member.GroupState != models.RoomStateIn {
			continue
		}

		// Get user info from FetchMetaIDUserInfo
		userInfo := common_service.FetchMetaIDUserInfo(member.Address)
		if userInfo == nil {
			// If user info is not available, create a basic one
			userInfo = &respond.UserInfo{
				Address: member.Address,
				Metaid:  member.MetaId,
				Name:    member.UserName,
			}
		}

		// Create member item with all information
		memberItem := &respond.GroupMemberSearchItem{
			MetaId:    member.MetaId,
			Address:   member.Address,
			UserInfo:  userInfo,
			Timestamp: member.Timestamp,
		}
		allMembers = append(allMembers, memberItem)
	}

	// Sort results by timestamp (newest first)
	sort.Slice(allMembers, func(i, j int) bool {
		return allMembers[i].Timestamp > allMembers[j].Timestamp
	})

	// Now perform fuzzy search on all collected members
	var results []*respond.GroupMemberSearchItem
	for _, member := range allMembers {
		// Check if member matches search query
		// Search in: metaId and user info name
		matches := false

		// Check member's metaId
		if strings.Contains(strings.ToLower(member.MetaId), queryLower) {
			matches = true
		}

		// Check user info name (from external API)
		if member.UserInfo.Name != "" && strings.Contains(strings.ToLower(member.UserInfo.Name), queryLower) {
			matches = true
		}

		if matches {
			results = append(results, member)

			// Check limit
			if int64(len(results)) >= req.Size {
				break
			}
		}
	}

	return &respond.GroupMemberSearchResponse{
		Total: int64(len(results)),
		List:  results,
	}, nil
}

// ==================== Channel Chat Service Methods ====================

// FetchChannelChatListV3 gets channel chat records using GetChatsByChannelIdAndEndTimestampRange3 (test version with IterOptions for improved performance)
func FetchChannelChatListV3(req *request.FetchChannelChatListRequest) (*respond.GroupChatResponse, error) {
	// Set default pagination parameters
	if req.Size <= 0 {
		req.Size = 20
	}

	// Performance monitoring: record start time
	startTime := time.Now()
	var perfStats = struct {
		getChatsTime       int64
		responseFormatTime int64
		userInfoTime       int64
		totalTime          int64
	}{}

	var chats []*models.TalkGroupChatV3
	var err error

	t := time.Now().UnixMilli()
	var nextTimestamp int64 = 0
	if req.Timestamp > 0 {
		// Get chat records by timestamp range using new method with IterOptions
		chats, nextTimestamp, err = chatDB.GetChatsByChannelIdAndEndTimestampRange3(req.ChannelId, req.Timestamp, req.Size)
	} else {
		// Get latest chat records using new method with IterOptions
		// For latest messages, we can use a very large timestamp as start point
		currentTimestamp := time.Now().Unix()
		//add 6 number 0
		currentTimestamp = currentTimestamp * 1000000
		chats, nextTimestamp, err = chatDB.GetChatsByChannelIdAndEndTimestampRange3(req.ChannelId, currentTimestamp, req.Size)
	}
	perfStats.getChatsTime = time.Now().UnixMilli() - t

	if err != nil {
		return nil, err
	}

	// Convert to response format
	var chatItems []*respond.GroupChatItem

	t1 := time.Now().UnixMilli()
	for _, chat := range chats {
		chatItem := &respond.GroupChatItem{
			GroupId:   chat.GroupId,
			ChannelId: chat.ChannelId,
			MetanetId: chat.ChannelId, // Use ChannelId as MetanetId for channel chats
			TxId:      chat.TxId,
			PinId:     chat.PinId,
			Address:   chat.Address,
			// UserInfo:    common_service.FetchMetaIDUserInfo(chat.Address),
			MetaId:      chat.MetaId,
			NickName:    "", // Need to get from user info
			Protocol:    chat.Protocol,
			Content:     chat.Content,
			ContentType: chat.ContentType,
			Encryption:  chat.Encryption,
			ChatType:    chat.ChatType,
			ReplyPin:    chat.ReplyPin,
			ReplyInfo:   nil,
			ReplyMetaId: "",
			Mention:     chat.Mention,
			Timestamp:   chat.Timestamp,
			Chain:       chat.Chain,
			BlockHeight: chat.BlockHeight,
			Index:       chat.Index,
			Version:     chat.Version,
		}

		if strings.Contains(strings.ToLower(chatItem.Protocol), strings.ToLower(protocols.MonitorSimpleGroupLuckyBag)) {
			luckyBag, _ := chatDB.GetLuckyBagByPinIdFromCache(chat.GroupId, chat.PinId)
			if luckyBag != nil {
				chatItem.Domain = luckyBag.Domain
			}
		}

		// Get user info
		if chat.MetaId != "" {
			userInfo := common_service.FetchMetaIDUserInfo(chat.Address)
			if userInfo != nil {
				chatItem.UserInfo = userInfo
				chatItem.NickName = userInfo.Name
			}
		}

		// Get reply info if exists
		if chat.ReplyPin != "" {
			replyChat, err := chatDB.GetChatByPinId(chat.ReplyPin)
			if err == nil && replyChat != nil {
				replyUserInfo := common_service.FetchMetaIDUserInfo(replyChat.Address)
				if replyUserInfo != nil {
					chatItem.ReplyInfo = &respond.ReplyInfo{
						ChannelId:   replyChat.ChannelId,
						PinId:       replyChat.PinId,
						MetaId:      replyChat.MetaId,
						Address:     replyChat.Address,
						UserInfo:    replyUserInfo,
						NickName:    replyUserInfo.Name,
						Protocol:    replyChat.Protocol,
						Content:     replyChat.Content,
						ContentType: replyChat.ContentType,
						Encryption:  replyChat.Encryption,
						ChatType:    replyChat.ChatType,
						Mention:     replyChat.Mention,
						Timestamp:   replyChat.Timestamp,
						Chain:       replyChat.Chain,
						Index:       replyChat.Index,
						Version:     replyChat.Version,
					}
				}
			}
		}

		chatItems = append(chatItems, chatItem)
	}
	perfStats.responseFormatTime = time.Now().UnixMilli() - t1

	perfStats.totalTime = time.Now().UnixMilli() - startTime.UnixMilli()

	logger.Info(fmt.Sprintf("[ChannelChatListV3] Performance stats - getChats: %dms, responseFormat: %dms, total: %dms, count: %d",
		perfStats.getChatsTime, perfStats.responseFormatTime, perfStats.totalTime, len(chatItems)))

	return &respond.GroupChatResponse{
		Total:         int64(len(chatItems)),
		NextTimestamp: nextTimestamp,
		List:          chatItems,
	}, nil
}

// FetchChannelChatListByIndex gets channel chat records by index range (ascending order) using TalkGroupChannelChatIndexCollection
func FetchChannelChatListByIndex(req *request.FetchChannelChatListByIndexRequest) (*respond.GroupChatResponse, error) {
	// Set default pagination parameters
	if req.Size <= 0 {
		req.Size = 20
	}

	// Performance monitoring: record start time
	startTime := time.Now()
	var perfStats = struct {
		getChatsTime       int64
		responseFormatTime int64
		userInfoTime       int64
		totalTime          int64
	}{}

	var chats []*models.TalkGroupChatV3
	var lastIndex int64
	var err error

	t := time.Now().UnixMilli()
	chats, lastIndex, err = chatDB.GetChatsByChannelIdAndStartIndexRange(req.ChannelId, req.StartIndex, req.Size)
	perfStats.getChatsTime = time.Now().UnixMilli() - t

	if err != nil {
		return nil, err
	}

	// Convert to response format
	var chatItems []*respond.GroupChatItem

	t1 := time.Now().UnixMilli()
	for _, chat := range chats {
		chatItem := &respond.GroupChatItem{
			GroupId:     chat.GroupId,
			ChannelId:   chat.ChannelId,
			MetanetId:   chat.ChannelId, // Use ChannelId as MetanetId for channel chats
			TxId:        chat.TxId,
			PinId:       chat.PinId,
			Address:     chat.Address,
			MetaId:      chat.MetaId,
			NickName:    "", // Need to get from user info
			Protocol:    chat.Protocol,
			Content:     chat.Content,
			ContentType: chat.ContentType,
			Encryption:  chat.Encryption,
			ChatType:    chat.ChatType,
			ReplyPin:    chat.ReplyPin,
			ReplyInfo:   nil,
			ReplyMetaId: "",
			Mention:     chat.Mention,
			Timestamp:   chat.Timestamp,
			Chain:       chat.Chain,
			BlockHeight: chat.BlockHeight,
			Index:       chat.Index,
			Version:     chat.Version,
		}
		if strings.Contains(strings.ToLower(chatItem.Protocol), strings.ToLower(protocols.MonitorSimpleGroupLuckyBag)) {
			luckyBag, _ := chatDB.GetLuckyBagByPinIdFromCache(chat.GroupId, chat.PinId)
			if luckyBag != nil {
				chatItem.Domain = luckyBag.Domain
			}
		}

		// Get user info
		if chat.MetaId != "" {
			userInfo := common_service.FetchMetaIDUserInfo(chat.Address)
			if userInfo != nil {
				chatItem.UserInfo = userInfo
				chatItem.NickName = userInfo.Name
			}
		}

		// Get reply info if exists
		if chat.ReplyPin != "" {
			replyChat, err := chatDB.GetChatByPinId(chat.ReplyPin)
			if err == nil && replyChat != nil {
				replyUserInfo := common_service.FetchMetaIDUserInfo(replyChat.Address)
				if replyUserInfo != nil {
					chatItem.ReplyInfo = &respond.ReplyInfo{
						ChannelId:   replyChat.ChannelId,
						PinId:       replyChat.PinId,
						MetaId:      replyChat.MetaId,
						Address:     replyChat.Address,
						UserInfo:    replyUserInfo,
						NickName:    replyUserInfo.Name,
						Protocol:    replyChat.Protocol,
						Content:     replyChat.Content,
						ContentType: replyChat.ContentType,
						Encryption:  replyChat.Encryption,
						ChatType:    replyChat.ChatType,
						Mention:     replyChat.Mention,
						Timestamp:   replyChat.Timestamp,
						Chain:       replyChat.Chain,
						Index:       replyChat.Index,
						Version:     replyChat.Version,
					}
				}
			}
		}

		chatItems = append(chatItems, chatItem)
	}
	perfStats.responseFormatTime = time.Now().UnixMilli() - t1

	perfStats.totalTime = time.Now().UnixMilli() - startTime.UnixMilli()

	logger.Info(fmt.Sprintf("[ChannelChatListByIndex] Performance stats - getChats: %dms, responseFormat: %dms, total: %dms, count: %d",
		perfStats.getChatsTime, perfStats.responseFormatTime, perfStats.totalTime, len(chatItems)))

	return &respond.GroupChatResponse{
		Total:     int64(len(chatItems)),
		LastIndex: lastIndex,
		List:      chatItems,
	}, nil
}

// FetchChannelChatListByStartTime gets channel chat records by start timestamp range (ascending order) using TalkGroupChannelChatTimestamp2Collection
func FetchChannelChatListByStartTime(req *request.FetchChannelChatListByStartTimeRequest) (*respond.GroupChatResponse, error) {
	// Set default pagination parameters
	if req.Size <= 0 {
		req.Size = 20
	}

	// Performance monitoring: record start time
	startTime := time.Now()
	var perfStats = struct {
		getChatsTime       int64
		responseFormatTime int64
		userInfoTime       int64
		totalTime          int64
	}{}

	var chats []*models.TalkGroupChatV3
	var lastTimestamp int64
	var err error

	t := time.Now().UnixMilli()
	chats, lastTimestamp, err = chatDB.GetChatsByChannelIdAndStartTimestampRange(req.ChannelId, req.StartTimestamp, req.Size)
	perfStats.getChatsTime = time.Now().UnixMilli() - t

	if err != nil {
		return nil, err
	}

	// Convert to response format
	var chatItems []*respond.GroupChatItem

	t1 := time.Now().UnixMilli()
	for _, chat := range chats {
		chatItem := &respond.GroupChatItem{
			GroupId:     chat.GroupId,
			ChannelId:   chat.ChannelId,
			MetanetId:   chat.ChannelId, // Use ChannelId as MetanetId for channel chats
			TxId:        chat.TxId,
			PinId:       chat.PinId,
			Address:     chat.Address,
			MetaId:      chat.MetaId,
			NickName:    "", // Need to get from user info
			Protocol:    chat.Protocol,
			Content:     chat.Content,
			ContentType: chat.ContentType,
			Encryption:  chat.Encryption,
			ChatType:    chat.ChatType,
			ReplyPin:    chat.ReplyPin,
			ReplyInfo:   nil,
			ReplyMetaId: "",
			Mention:     chat.Mention,
			Timestamp:   chat.Timestamp,
			Chain:       chat.Chain,
			BlockHeight: chat.BlockHeight,
			Index:       chat.Index,
			Version:     chat.Version,
		}
		if strings.Contains(strings.ToLower(chatItem.Protocol), strings.ToLower(protocols.MonitorSimpleGroupLuckyBag)) {
			luckyBag, _ := chatDB.GetLuckyBagByPinIdFromCache(chat.GroupId, chat.PinId)
			if luckyBag != nil {
				chatItem.Domain = luckyBag.Domain
			}
		}

		// Get user info
		if chat.MetaId != "" {
			userInfo := common_service.FetchMetaIDUserInfo(chat.Address)
			if userInfo != nil {
				chatItem.UserInfo = userInfo
				chatItem.NickName = userInfo.Name
			}
		}

		// Get reply info if exists
		if chat.ReplyPin != "" {
			replyChat, err := chatDB.GetChatByPinId(chat.ReplyPin)
			if err == nil && replyChat != nil {
				replyUserInfo := common_service.FetchMetaIDUserInfo(replyChat.Address)
				if replyUserInfo != nil {
					chatItem.ReplyInfo = &respond.ReplyInfo{
						ChannelId:   replyChat.ChannelId,
						PinId:       replyChat.PinId,
						MetaId:      replyChat.MetaId,
						Address:     replyChat.Address,
						UserInfo:    replyUserInfo,
						NickName:    replyUserInfo.Name,
						Protocol:    replyChat.Protocol,
						Content:     replyChat.Content,
						ContentType: replyChat.ContentType,
						Encryption:  replyChat.Encryption,
						ChatType:    replyChat.ChatType,
						Mention:     replyChat.Mention,
						Timestamp:   replyChat.Timestamp,
						Chain:       replyChat.Chain,
						Index:       replyChat.Index,
						Version:     replyChat.Version,
					}
				}
			}
		}

		chatItems = append(chatItems, chatItem)
	}
	perfStats.responseFormatTime = time.Now().UnixMilli() - t1

	perfStats.totalTime = time.Now().UnixMilli() - startTime.UnixMilli()

	logger.Info(fmt.Sprintf("[ChannelChatListByStartTime] Performance stats - getChats: %dms, responseFormat: %dms, total: %dms, count: %d",
		perfStats.getChatsTime, perfStats.responseFormatTime, perfStats.totalTime, len(chatItems)))

	return &respond.GroupChatResponse{
		Total:         int64(len(chatItems)),
		LastTimestamp: lastTimestamp,
		List:          chatItems,
	}, nil
}

// FetchGroupChannelList gets channel list by group ID from TalkGroupChannelCollection
func FetchGroupChannelList(req *request.FetchGroupChannelListRequest) (*respond.GroupChannelResponse, error) {
	// Set default pagination parameters
	if req.Size <= 0 {
		req.Size = 20
	}

	// Get channels from database
	channels, err := groupDB.GetChannelsByGroupId(req.GroupId)
	if err != nil {
		return nil, err
	}

	// Apply pagination
	start := req.Cursor
	end := start + req.Size
	if start >= int64(len(channels)) {
		return &respond.GroupChannelResponse{
			Total: int64(len(channels)),
			List:  []*respond.GroupChannelItem{},
		}, nil
	}
	if end > int64(len(channels)) {
		end = int64(len(channels))
	}

	// Convert to response format
	var channelItems []*respond.GroupChannelItem
	for i := start; i < end; i++ {
		channel := channels[i]

		// Get channel latest chat information
		channelLatestChat, err := chatDB.GetGroupChannelLatestChat(channel.ChannelId)
		if err != nil {
			logger.Info(fmt.Sprintf("Failed to get channel latest chat for channelId %s: %v", channel.ChannelId, err))
		}

		channelItem := &respond.GroupChannelItem{
			ChannelId:         channel.ChannelId,
			GroupId:           channel.GroupId,
			ChannelName:       channel.ChannelName,
			ChannelIcon:       channel.ChannelIcon,
			ChannelNote:       channel.ChannelNote,
			ChannelType:       channel.ChannelType,
			CreateUserMetaId:  channel.CreateUserMetaId,
			CreateUserAddress: channel.CreateUserAddress,
			Timestamp:         channel.Timestamp,
			Chain:             channel.Chain,
			BlockHeight:       channel.BlockHeight,
			Index:             -1, // TalkGroupChannelModel doesn't have Index field, using 0 as default
		}

		// Set channel newest information if available
		if channelLatestChat != nil {
			channelItem.ChannelNewestTxId = channelLatestChat.TxId
			channelItem.ChannelNewestPinId = channelLatestChat.PinId
			channelItem.ChannelNewestMetaId = channelLatestChat.MetaId
			channelItem.ChannelNewestProtocol = channelLatestChat.Protocol
			channelItem.ChannelNewestContent = channelLatestChat.Content
			channelItem.ChannelNewestTimestamp = channelLatestChat.Timestamp

			channelChat, err := chatDB.GetChatByPinId(channelLatestChat.PinId)
			if err == nil && channelChat != nil {
				channelItem.Index = channelChat.Index
				channelItem.Version = channelChat.Version
			}

			// Get user info for channelNewestUserName
			if channelLatestChat.CreateAddress != "" {
				userInfo := common_service.FetchMetaIDUserInfo(channelLatestChat.CreateAddress)
				if userInfo != nil {
					channelItem.ChannelNewestUserName = userInfo.Name
				}
			}
		}

		channelItems = append(channelItems, channelItem)
	}

	return &respond.GroupChannelResponse{
		Total: int64(len(channels)),
		List:  channelItems,
	}, nil
}

// FetchGroupUserRoleInfo
func FetchGroupUserRoleInfo(req *request.FetchGroupUserRoleInfoRequest) (*respond.GroupUserRoleInfo, error) {
	if req.GroupId == "" {
		return nil, fmt.Errorf("groupId is empty")
	}
	if req.MetaId == "" {
		return nil, fmt.Errorf("metaId is empty")
	}

	roleInfo, err := groupDB.GetGroupUserRoleInfo(req.GroupId, req.ChannelId, req.MetaId)
	if err != nil {
		return nil, err
	}

	userInfo := common_service.FetchMetaIDUserInfoInfoByMetaId(req.MetaId)
	if userInfo == nil {
		userInfo = &respond.UserInfo{
			Metaid:  req.MetaId,
			Address: "",
			Name:    "",
			Avatar:  "",
		}
	}

	result := &respond.GroupUserRoleInfo{
		MetaId:      roleInfo.MetaId,
		Address:     userInfo.Address,
		UserInfo:    userInfo,
		GroupId:     roleInfo.GroupId,
		ChannelId:   roleInfo.ChannelId,
		IsCreator:   roleInfo.IsCreator,
		IsAdmin:     roleInfo.IsAdmin,
		IsBlocked:   roleInfo.IsBlocked,
		IsWhitelist: roleInfo.IsWhitelist,
	}

	return result, nil
}

// startSocketInfoSnapshotTimer Start socket info snapshot timer
func startSocketInfoSnapshotTimer() {
	logger.Info("[SOCKET_INFO_SNAPSHOT] Starting socket info snapshot timer - every 5 minutes")

	go func() {
		// Start the periodic timer immediately
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()

		// Take initial snapshot
		takeSocketInfoSnapshot()

		// Take snapshots every 5 minutes
		for range ticker.C {
			if db.GlobalIsStop {
				logger.Info("[SOCKET_INFO_SNAPSHOT] Socket info snapshot timer stopped")
				break
			}
			takeSocketInfoSnapshot()
		}
	}()
}

// takeSocketInfoSnapshot Take a snapshot of socket connection stats and save to database
func takeSocketInfoSnapshot() {
	startTime := time.Now()

	logger.Info("[SOCKET_INFO_SNAPSHOT] Starting to take socket info snapshot")

	// Get connection stats from socket service
	stats := socket_service.GetConnectionStats()
	if stats == nil {
		logger.Info("[SOCKET_INFO_SNAPSHOT] Failed to get connection stats - socket service not available")
		return
	}

	// Create timestamp for this snapshot (use current time)
	timestamp := time.Now().UnixMilli()

	// Create snapshot data
	snapshot := &db.SocketInfoSnapshot{
		Timestamp:             timestamp,
		TotalConnections:      stats.TotalConnections,
		ActiveConnections:     stats.ActiveConnections,
		TotalUserConnections:  stats.TotalUserConnections,
		ActiveUserConnections: stats.ActiveUserConnections,
		TotalMessagesSent:     stats.TotalMessagesSent,
		TotalMessagesFailed:   stats.TotalMessagesFailed,
		TotalMemoryUsage:      stats.TotalMemoryUsage,
		AverageMemoryPerConn:  stats.AverageMemoryPerConn,
		TotalMemoryMB:         stats.TotalMemoryMB,
		AverageMemoryKB:       stats.AverageMemoryKB,
		MemoryUsagePercent:    stats.MemoryUsagePercent,
		MemoryLimitMB:         stats.MemoryLimitMB,
	}

	if db.GlobalIsStop {
		logger.Info("[SOCKET_INFO_SNAPSHOT] Socket info snapshot timer stopped")
		return
	}

	// Save to database
	err := socketInfoDB.SaveSocketInfoSnapshot(timestamp, snapshot)
	if err != nil {
		logger.Info("[SOCKET_INFO_SNAPSHOT] Failed to save snapshot: %v", err)
		return
	}

	elapsed := time.Since(startTime).Milliseconds()
	logger.Info("[SOCKET_INFO_SNAPSHOT] Snapshot taken successfully - "+
		"Timestamp: %d, TotalConnections: %d, ActiveConnections: %d, "+
		"TotalMessagesSent: %d, TotalMessagesFailed: %d, TotalMemoryMB: %.2f, "+
		"MemoryUsagePercent: %.2f%%, Elapsed: %dms",
		timestamp, stats.TotalConnections, stats.ActiveConnections,
		stats.TotalMessagesSent, stats.TotalMessagesFailed, stats.TotalMemoryMB,
		stats.MemoryUsagePercent, elapsed)
}

// IsSyncCompleted Check if synchronization is completed
func IsSyncCompleted() bool {
	return syncDbService.IsSyncCompleted()
}

// FetchPrivateGroupPaths Get private group paths by MetaId
func FetchPrivateGroupPaths(req *request.FetchPrivateGroupPathsRequest) (*respond.PrivateGroupPathsResponse, error) {
	if req.MetaId == "" {
		return nil, fmt.Errorf("metaId is required")
	}

	// Get private group paths from database
	pathItems, err := groupDB.GetPrivateGroupPathsByMetaId(req.MetaId)
	if err != nil {
		return nil, fmt.Errorf("failed to get private group paths: %v", err)
	}

	// Convert to response format
	var items []*respond.PrivateGroupPathItem
	for _, item := range pathItems {
		items = append(items, &respond.PrivateGroupPathItem{
			Path:    item.Path,
			GroupId: item.GroupId,
			PinId:   item.PinId,
		})
	}

	return &respond.PrivateGroupPathsResponse{
		Total: int64(len(items)),
		List:  items,
	}, nil
}

// FetchGroupJoinControlList Get group join block and whitelist metaId list
func FetchGroupJoinControlList(req *request.FetchGroupJoinControlListRequest) (*respond.GroupJoinControlListResponse, error) {
	if req.GroupId == "" {
		return nil, fmt.Errorf("groupId is empty")
	}

	// Get join block list
	joinBlockList, err := groupDB.GetGroupJoinBlockList(req.GroupId)
	if err != nil {
		return nil, fmt.Errorf("failed to get join block list: %v", err)
	}

	// Get join whitelist list
	joinWhitelistList, err := groupDB.GetGroupJoinWhitelistList(req.GroupId)
	if err != nil {
		return nil, fmt.Errorf("failed to get join whitelist list: %v", err)
	}

	// Get current effective join block metaIds (latest record)
	var joinBlockMetaIds []string
	if joinBlockList != nil && len(joinBlockList.Items) > 0 {
		// Data is sorted by timestamp in ascending order, so the last item is the latest
		latestBlockItem := joinBlockList.Items[len(joinBlockList.Items)-1]
		joinBlockMetaIds = latestBlockItem.BlockedUsers
	}

	// Get current effective join whitelist metaIds (latest record)
	var joinWhitelistMetaIds []string
	if joinWhitelistList != nil && len(joinWhitelistList.Items) > 0 {
		// Data is sorted by timestamp in ascending order, so the last item is the latest
		latestWhitelistItem := joinWhitelistList.Items[len(joinWhitelistList.Items)-1]
		joinWhitelistMetaIds = latestWhitelistItem.WhitelistUsers
	}

	return &respond.GroupJoinControlListResponse{
		GroupId:              req.GroupId,
		JoinBlockMetaIds:     joinBlockMetaIds,
		JoinWhitelistMetaIds: joinWhitelistMetaIds,
	}, nil
}

// FetchGroupMetaIdJoinList Get group MetaId join list
func FetchGroupMetaIdJoinList(req *request.FetchGroupMetaIdJoinListRequest) (*respond.GroupMetaIdJoinListResponse, error) {
	if req.MetaId == "" {
		return nil, fmt.Errorf("metaId is required")
	}
	if req.GroupId == "" {
		return nil, fmt.Errorf("groupId is required")
	}

	// Get join list from database
	joinList, err := groupDB.GetGroupMetaIdJoinList(req.MetaId, req.GroupId)
	if err != nil {
		return nil, fmt.Errorf("failed to get group MetaId join list: %v", err)
	}

	// Convert to response format
	var items []*respond.GroupMetaIdJoinItemResponse
	for _, item := range joinList.Items {
		items = append(items, &respond.GroupMetaIdJoinItemResponse{
			JoinPinId:     item.JoinPinId,
			JoinType:      item.JoinType,
			JoinTimestamp: item.JoinTimestamp,
			GroupState:    int64(item.GroupState),
			Address:       item.Address,
			Referrer:      item.Referrer,
			K:             item.K,
			BlockHeight:   item.BlockHeight,
			Chain:         item.Chain,
			ByMetaId:      item.ByMetaId,
			ByAddress:     item.ByAddress,
		})
	}

	return &respond.GroupMetaIdJoinListResponse{
		MetaId: joinList.MetaId,
		Items:  items,
	}, nil
}

// UpdateUserInfoCache Update user info cache by address or metaId
func UpdateUserInfoCache(req *request.UpdateUserInfoCacheRequest) error {
	if req.Address != "" {
		return common_service.TriggerUpdateUserInfo(req.Address)
	} else if req.MetaId != "" {
		return common_service.TriggerUpdateUserInfoByMetaId(req.MetaId)
	}
	return fmt.Errorf("either address or metaId must be provided")
}
