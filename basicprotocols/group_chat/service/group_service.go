package service

import (
	"fmt"
	"manindexer/adapter"
	"manindexer/basicprotocols/group_chat/api/request"
	"manindexer/basicprotocols/group_chat/api/respond"
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
	chainAdapter = adapter

	// Start chat queue processor
	// chatDB.StartQueueProcessor(groupDB)

	// Start private chat queue processor
	// privateDB.StartPrivateQueueProcessor()

	StartOpenLuckyBagQueueProcessor()
	StartResidueLuckyBagQueueProcessor()

	db.SetHandleGroupChatItem(wsForGroupChatItem)

	// Initialize cache service for lucky bag
	cache_service.InitCacheService("", "", 0)

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
		userCount, err := groupDB.GetGroupMemberCount(group.GroupId)
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
		userCount, err := groupDB.GetGroupMemberCount(item.GroupId)
		if err != nil {
			// If failed to get, use default value
			userCount = 0
		}

		groupItem := &respond.GroupItem{
			CommunityId:  group.CommunityId,
			GroupId:      group.GroupId,
			TxId:         group.TxId,
			PinId:        group.PinId,
			RoomName:     group.RoomName,
			RoomNote:     group.RoomNote,
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
		groupItems = append(groupItems, groupItem)
	}

	return &respond.GroupResponse{
		Total: int64(len(groupItems)),
		List:  groupItems,
	}, nil
}

// FetchGroupInfo Get group info
func FetchGroupInfo(req *request.FetchGroupInfoRequest) (*respond.GroupItem, error) {
	// Get group info
	group, err := groupDB.GetGroupInfoByGroupId(req.GroupId)
	if err != nil {
		return nil, err
	}
	if group == nil {
		return nil, nil
	}

	// Get group's latest chat info
	latestChat, err := chatDB.GetGroupLatestChat(req.GroupId)
	if err != nil {
		// If failed to get, use default value
		latestChat = nil
	}

	// Get group member count
	userCount, err := groupDB.GetGroupMemberCount(req.GroupId)
	if err != nil {
		// If failed to get, use default value
		userCount = 0
	}

	// Convert to response format
	groupItem := &respond.GroupItem{
		CommunityId:  group.CommunityId,
		GroupId:      group.GroupId,
		TxId:         group.TxId,
		PinId:        group.PinId,
		RoomName:     group.RoomName,
		RoomNote:     group.RoomNote,
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
	// fmt.Printf("chats: %+v\n", chats)

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
			RedMetaId:   "",
			Timestamp:   chat.Timestamp,
			Chain:       chat.Chain,
			BlockHeight: chat.BlockHeight,
		}
		if strings.Contains(strings.ToLower(chatItem.Protocol), strings.ToLower(protocols.MonitorSimpleGroupOpenLuckyBag)) {
			openLuckyBag, _ := chatDB.GetOpenLuckyBagByPinId(chat.PinId)
			if openLuckyBag != nil {
				// fmt.Printf("openLuckyBag: GrabTxId: %s, PinId: %s, GrabState: %d\n", openLuckyBag.GrabTxId, openLuckyBag.PinId, openLuckyBag.GrabState)
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
			chatItem.RedMetaId = replyChat.MetaId
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

// FetchGroupChatListV2 Get group chat list using TalkGroupChatTimestamp2Collection
func FetchGroupChatListV2(req *request.FetchGroupChatListRequest) (*respond.GroupChatResponse, error) {
	// Set default pagination parameters
	if req.Size <= 0 {
		req.Size = 20
	}

	var chats []*models.TalkGroupChatV3
	var err error

	var nextTimestamp int64 = 0
	if req.Timestamp > 0 {
		// Get chat records by timestamp range using new collection
		chats, nextTimestamp, err = chatDB.GetChatsByGroupIdAndTimestampRange2(req.GroupId, req.Timestamp, req.Size)
	} else {
		// Get latest chat records using new collection
		// For latest messages, we can use a very large timestamp as start point
		currentTimestamp := time.Now().Unix()
		//add 6 number 0
		currentTimestamp = currentTimestamp * 1000000
		chats, nextTimestamp, err = chatDB.GetChatsByGroupIdAndTimestampRange2(req.GroupId, currentTimestamp, req.Size)
	}

	if err != nil {
		return nil, err
	}

	// Convert to response format
	var chatItems []*respond.GroupChatItem

	for _, chat := range chats {
		chatItem := &respond.GroupChatItem{
			GroupId:     chat.GroupId,
			MetanetId:   chat.GroupId, // Use GroupId as MetanetId
			TxId:        chat.TxId,
			PinId:       chat.PinId,
			Address:     chat.Address,
			UserInfo:    common_service.FetchMetaIDUserInfo(chat.Address),
			MetaId:      chat.MetaId,
			NickName:    "", // Need to get from user info
			Protocol:    chat.Protocol,
			Content:     chat.Content,
			ContentType: chat.ContentType,
			Encryption:  chat.Encryption,
			ChatType:    chat.ChatType,
			ReplyPin:    chat.ReplyPin,
			ReplyInfo:   nil,
			RedMetaId:   "",
			Timestamp:   chat.Timestamp,
			Chain:       chat.Chain,
			BlockHeight: chat.BlockHeight,
		}
		if strings.Contains(strings.ToLower(chatItem.Protocol), strings.ToLower(protocols.MonitorSimpleGroupOpenLuckyBag)) {
			openLuckyBag, _ := chatDB.GetOpenLuckyBagByPinId(chat.PinId)
			if openLuckyBag != nil {
				// fmt.Printf("openLuckyBag: GrabTxId: %s, PinId: %s, GrabState: %d\n", openLuckyBag.GrabTxId, openLuckyBag.PinId, openLuckyBag.GrabState)
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
			chatItem.RedMetaId = replyChat.MetaId
			chatItem.BlockHeight = replyChat.BlockHeight
		}

		chatItems = append(chatItems, chatItem)

		// Record next message timestamp (for pagination)
		// if i == len(chats)-1 && len(chats) > 0 {
		// 	nextTimestamp = chat.Timestamp
		// }
	}

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

	var members []*models.TalkGroupJoinModel
	var total int64
	var err error

	// Check if orderBy is "timestamp" for timestamp descending order
	if req.OrderBy == "timestamp" {
		// Get all group members first
		allMembers, err := groupDB.GetGroupMembers(req.GroupId)
		if err != nil {
			return nil, err
		}

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
		members, total, err = groupDB.GetGroupMembersWithPagination(req.GroupId, req.Cursor, req.Size)
		if err != nil {
			return nil, err
		}
	}

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

	// Get user's context list (group chat + private chat)
	contextList, err := chatDB.GetMetaIdContextList(req.MetaId)
	if err != nil {
		return nil, err
	}

	// Convert to response format
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
			// Group chat type, get group detailed info
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

			// Fill group chat specific fields
			chatInfoItem.CommunityId = group.CommunityId
			chatInfoItem.RoomName = group.RoomName
			chatInfoItem.RoomNote = group.RoomNote
			chatInfoItem.RoomType = group.RoomType
			chatInfoItem.RoomStatus = group.RoomStatus
			chatInfoItem.RoomJoinType = group.RoomJoinType
			chatInfoItem.RoomAvatarUrl = group.RoomAvatarUrl
			chatInfoItem.CreateUserMetaId = group.CreateUserMetaId
			chatInfoItem.CreateUserAddress = group.CreateUserAddress
			chatInfoItem.CreateUserInfo = common_service.FetchMetaIDUserInfo(group.CreateUserAddress)
			chatInfoItem.UserCount = 0 // Need to calculate
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
			}
		} else if item.Type == "2" {
			// Private chat type, get latest private chat message
			latestPrivateChat, err := privateDB.GetPrivateChatByPinId(item.LastMessagePinId)
			if err != nil {
				// If failed to get, use default value
				latestPrivateChat = nil
			}

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
				chatInfoItem.UserInfo = common_service.FetchMetaIDUserInfo(latestPrivateChat.FromAddress)
			}
		}

		chatInfoItems = append(chatInfoItems, chatInfoItem)
	}

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
	var err error

	if req.Timestamp > 0 {
		// Get private chat records by timestamp range
		chats, err = privateDB.GetPrivateChatsByMetaIdsAndTimestampRange(req.MetaId, req.OtherMetaId, req.Timestamp, req.Size)
	} else {
		// Get latest private chat records
		chats, err = privateDB.GetLatestPrivateChatsByMetaIds(req.MetaId, req.OtherMetaId, req.Size)
	}

	if err != nil {
		return nil, err
	}
	// fmt.Printf("private chats: %+v\n", chats)

	// Convert to response format
	var chatItems []*respond.PrivateChatItem
	var nextTimestamp int64 = 0

	for i, chat := range chats {
		chatItem := &respond.PrivateChatItem{
			From:        chat.From,
			To:          chat.To,
			TxId:        chat.TxId,
			PinId:       chat.PinId,
			MetaId:      chat.From, // Message creator MetaId
			Address:     chat.FromAddress,
			UserInfo:    common_service.FetchMetaIDUserInfo(chat.FromAddress),
			NickName:    "", // Need to get from user info
			Protocol:    chat.Protocol,
			Content:     chat.Content,
			ContentType: chat.ContentType,
			Encryption:  chat.Encryption,
			ChatType:    int64(chat.ChatType),
			ReplyPin:    chat.ReplyPin,
			ReplyInfo:   nil,
			RedMetaId:   "",
			Timestamp:   chat.Timestamp,
			Chain:       chat.Chain,
			BlockHeight: chat.BlockHeight,
		}

		// Handle reply message
		if chat.ReplyPin != "" {
			replyChat, err := privateDB.GetPrivateChatByPinId(chat.ReplyPin)
			if err != nil {
				replyChat = nil
			}
			if replyChat != nil {
				chatItem.ReplyInfo = &respond.ReplyInfo{
					PinId:       replyChat.PinId,
					MetaId:      replyChat.From,
					Address:     replyChat.FromAddress,
					UserInfo:    common_service.FetchMetaIDUserInfo(replyChat.FromAddress),
					NickName:    "", // Private chat message doesn't have NickName field
					Protocol:    replyChat.Protocol,
					Content:     replyChat.Content,
					ContentType: replyChat.ContentType,
					Encryption:  replyChat.Encryption,
					ChatType:    replyChat.ChatType,
					Timestamp:   replyChat.Timestamp,
					Chain:       replyChat.Chain,
				}
				chatItem.RedMetaId = replyChat.From
			}
		}

		chatItems = append(chatItems, chatItem)

		// Record next message timestamp (for pagination)
		if i == len(chats)-1 && len(chats) > 0 {
			nextTimestamp = chat.Timestamp
		}
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
