package service

import (
	"fmt"
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
	"sort"
	"strings"
	"time"
)

var (
	communityDB  *db.CommunityDB
	groupDB      *db.GroupDB
	chatDB       *db.ChatDB
	privateDB    *db.PrivateChatDB
	userInfoDB   *db.UserInfoDB
	pebbleDB     *db.Pebble
	chainAdapter map[string]adapter.Chain
)

// InitService Initialize service
func InitService(indexer *indexer.GroupChatIndexer, adapter map[string]adapter.Chain) error {
	// Use indexer's database instance

	// Get database instances
	pebbleDB = indexer.GetPebble()
	communityDB = indexer.GetCommunityDB()
	groupDB = indexer.GetGroupDB()
	chatDB = indexer.GetChatDB()
	privateDB = indexer.GetPrivateDB()
	userInfoDB = indexer.GetUserInfoDB()
	chainAdapter = adapter

	// Start chat queue processor
	// chatDB.StartQueueProcessor(groupDB)

	// Start private chat queue processor
	// privateDB.StartPrivateQueueProcessor()

	StartOpenLuckyBagQueueProcessor()
	StartResidueLuckyBagQueueProcessor()
	StartExpiredLuckyBagProcessor()

	db.SetHandleGroupChatItem(wsForGroupChatItem)
	db.SetHandlePrivateChatItem(wsForPrivateChatItem)

	// Initialize cache service for lucky bag
	cache_service.InitCacheService("", "", 0)

	// Start user info polling
	common_service.StartUserInfoPolling()

	startLuckyBagGrabCleanupGoroutine()

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

	var groups []*models.TalkGroupModel
	var err error

	// If metaId is not empty, get user's joined group list
	if req.MetaId != "" {
		groups, err = groupDB.GetGroupListByMetaId(req.MetaId, req.Cursor, req.Size)
	} else {
		// Get all group list
		groups, err = groupDB.GetGroupList(req.Cursor, req.Size)
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
			Timestamp:         group.Timestamp,
			Chain:             group.Chain,
			BlockHeight:       group.BlockHeight,
		}
		groupItems = append(groupItems, groupItem)
	}

	return &respond.GroupResponse{
		Total: int64(len(groupItems)),
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
			ChatType:    chat.ChatType,
			ReplyPin:    chat.ReplyPin,
			ReplyInfo:   nil,
			ReplyMetaId: "",
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
					ChatType:    replyChat.ChatType,
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
			ChatType:    chat.ChatType,
			ReplyPin:    chat.ReplyPin,
			ReplyInfo:   nil,
			ReplyMetaId: "",
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
					ChatType:    replyChat.ChatType,
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
		responseFormatTime int64
		totalTime          int64
		cacheHit           bool
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
	// Convert to response format
	var memberItems []*respond.GroupMemberItem
	for _, member := range members {
		memberItem := &respond.GroupMemberItem{
			MetaId:    member.MetaId,
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
	cacheStatus := "DB"
	if perfStats.cacheHit {
		cacheStatus = "Cache"
	}
	logger.Info("[CHAT_SERVICE][FETCH_GROUP_MEMBER_LIST_V2] Performance Stats - "+
		"Total: %dms, GetMembers: %dms, ResponseFormat: %dms, Items: %d, Source: %s",
		perfStats.totalTime,
		perfStats.getMembersTime,
		perfStats.responseFormatTime,
		len(memberItems),
		cacheStatus)

	return &respond.GroupMemberResponse{
		Total: total,
		List:  memberItems,
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
					return nil, nil
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
			chatInfoItem.Chain = group.Chain

			// If latest chat info is obtained, update related fields
			if latestChat != nil {
				chatInfoItem.Content = latestChat.Content
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

func wsForGroupChatItem(chat *models.TalkGroupChatV3) error {
	wsPostGroupMsg(chat)
	return nil
}

func wsForPrivateChatItem(chat *models.TalkPrivateChatV3) error {
	wsPostPrivateMsg(chat)
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
	results, err := groupDB.SearchGroups(req.Query, int(req.Size))
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
	userResults, err := common_service.SearchAllMetaIDUserInfoInfo(req.Query)
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
	groupResults, err := groupDB.SearchGroups(req.Query, int(req.Size))
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
