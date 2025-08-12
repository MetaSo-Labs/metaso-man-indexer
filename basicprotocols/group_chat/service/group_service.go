package service

import (
	"fmt"
	"manindexer/basicprotocols/group_chat/api/request"
	"manindexer/basicprotocols/group_chat/api/respond"
	"manindexer/basicprotocols/group_chat/db"
	"manindexer/basicprotocols/group_chat/indexer"
	"manindexer/basicprotocols/group_chat/models"
	"time"
)

var (
	groupDB  *db.GroupDB
	chatDB   *db.ChatDB
	pebbleDB *db.Pebble
)

// InitService 初始化服务
func InitService(indexer *indexer.GroupChatIndexer) error {
	// 使用 indexer 的数据库实例

	// 获取数据库实例
	pebbleDB = indexer.GetPebble()
	groupDB = indexer.GetGroupDB()
	chatDB = indexer.GetChatDB()

	// 启动聊天队列处理器
	chatDB.StartQueueProcessor(groupDB)

	return nil
}

// FetchGroupList 获取群组列表
func FetchGroupList(req *request.FetchGroupListRequest) (*respond.GroupResponse, error) {
	// 设置默认分页参数
	if req.Size <= 0 {
		req.Size = 20
	}
	if req.Cursor <= 0 {
		req.Cursor = 1
	}

	// 获取群组列表
	groups, err := groupDB.GetGroupList(req.Cursor, req.Size)
	if err != nil {
		return nil, err
	}

	// 转换为响应格式
	var groupItems []*respond.GroupItem
	for _, group := range groups {
		// 获取群组最新聊天信息
		latestChat, err := chatDB.GetGroupLatestChat(group.GroupId)
		if err != nil {
			// 如果获取失败，使用默认值
			latestChat = nil
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
			// RoomCodeHash:          "", // 暂未实现
			// RoomGenesis:           "", // 暂未实现
			// RoomLimitAmount:       0,  // 暂未实现
			// RoomGenesisSeriesName: "", // 暂未实现
			RoomAvatarUrl:      group.RoomAvatarUrl,
			RoomNinePersonHash: "", // 暂未实现
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
			RoomNewestUserName: "", // 暂未实现，需要从用户信息中获取
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
			CreateUserMetaId: group.CreateUserMetaId,
			UserCount:        0, // 需要计算
			ChatSettingType:  group.ChatSettingType,
			DeleteStatus:     group.DeleteStatus,
			Timestamp:        group.Timestamp,
			Chain:            group.Chain,
			BlockHeight:      group.BlockHeight,
		}
		groupItems = append(groupItems, groupItem)
	}

	return &respond.GroupResponse{
		Total: int64(len(groupItems)),
		List:  groupItems,
	}, nil
}

// FetchLatestChatGroupList 获取最新聊天群组列表
func FetchLatestChatGroupList(req *request.FetchLatestChatGroupListRequest) (*respond.GroupResponse, error) {
	// 设置默认分页参数
	if req.Size <= 0 {
		req.Size = 20
	}
	if req.Cursor <= 0 {
		req.Cursor = 1
	}

	// 获取用户的群列表（基于最新聊天时间）
	contextList, err := chatDB.GetMetaIdContextList(req.MetaId)
	if err != nil {
		return nil, err
	}

	// 转换为响应格式
	var groupItems []*respond.GroupItem
	for _, item := range contextList.Items {
		// 获取群组详细信息
		group, err := groupDB.GetGroupInfoByGroupId(item.GroupId)
		if err != nil || group == nil {
			continue
		}

		// 获取群组最新聊天信息
		latestChat, err := chatDB.GetGroupLatestChat(item.GroupId)
		if err != nil {
			// 如果获取失败，使用默认值
			latestChat = nil
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
			// RoomCodeHash:          "", // 暂未实现
			// RoomGenesis:           "", // 暂未实现
			// RoomLimitAmount:       0,  // 暂未实现
			// RoomGenesisSeriesName: "", // 暂未实现
			RoomAvatarUrl:      group.RoomAvatarUrl,
			RoomNinePersonHash: "", // 暂未实现
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
			RoomNewestUserName: "", // 暂未实现，需要从用户信息中获取
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
			CreateUserMetaId: group.CreateUserMetaId,
			UserCount:        0, // 需要计算
			ChatSettingType:  group.ChatSettingType,
			DeleteStatus:     group.DeleteStatus,
			Timestamp:        group.Timestamp,
			Chain:            group.Chain,
			BlockHeight:      group.BlockHeight,
		}
		groupItems = append(groupItems, groupItem)
	}

	return &respond.GroupResponse{
		Total: int64(len(groupItems)),
		List:  groupItems,
	}, nil
}

// FetchGroupInfo 获取群组信息
func FetchGroupInfo(req *request.FetchGroupInfoRequest) (*respond.GroupItem, error) {
	// 获取群组信息
	group, err := groupDB.GetGroupInfoByGroupId(req.GroupId)
	if err != nil {
		return nil, err
	}
	if group == nil {
		return nil, nil
	}

	// 获取群组最新聊天信息
	latestChat, err := chatDB.GetGroupLatestChat(req.GroupId)
	if err != nil {
		// 如果获取失败，使用默认值
		latestChat = nil
	}

	// 转换为响应格式
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
		// RoomCodeHash:          "", // 暂未实现
		// RoomGenesis:           "", // 暂未实现
		// RoomLimitAmount:       0,  // 暂未实现
		// RoomGenesisSeriesName: "", // 暂未实现
		RoomAvatarUrl:      group.RoomAvatarUrl,
		RoomNinePersonHash: "", // 暂未实现
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
		RoomNewestUserName: "", // 暂未实现，需要从用户信息中获取
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
		CreateUserMetaId: group.CreateUserMetaId,
		UserCount:        0, // 需要计算
		ChatSettingType:  group.ChatSettingType,
		DeleteStatus:     group.DeleteStatus,
		Timestamp:        group.Timestamp,
		Chain:            group.Chain,
		BlockHeight:      group.BlockHeight,
	}

	return groupItem, nil
}

// FetchGroupChatList 获取群组聊天列表
func FetchGroupChatList(req *request.FetchGroupChatListRequest) (*respond.GroupChatResponse, error) {
	// 设置默认分页参数
	if req.Size <= 0 {
		req.Size = 20
	}

	var chats []*models.TalkGroupChatV3
	var err error

	if req.Timestamp > 0 {
		// 根据时间戳范围获取聊天记录
		chats, err = chatDB.GetChatsByGroupIdAndTimestampRange(req.GroupId, req.Timestamp, req.Size)
	} else {
		// 获取最新的聊天记录
		chats, err = chatDB.GetLatestChatsByGroupId(req.GroupId, req.Size)
	}

	if err != nil {
		return nil, err
	}
	fmt.Printf("chats: %+v\n", chats)

	// 转换为响应格式
	var chatItems []*respond.GroupChatItem
	var nextTimestamp int64 = 0

	for i, chat := range chats {
		chatItem := &respond.GroupChatItem{
			GroupId:     chat.GroupId,
			MetanetId:   chat.GroupId, // 使用 GroupId 作为 MetanetId
			TxId:        chat.TxId,
			MetaId:      chat.MetaId,
			NickName:    "", // 需要从用户信息中获取
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
		if chat.ReplyPin != "" {
			replyChat, err := chatDB.GetChatByPinId(chat.ReplyPin)
			if err != nil {
				replyChat = nil
			}
			chatItem.ReplyInfo = &respond.ReplyInfo{
				PinId:       replyChat.PinId,
				MetaId:      replyChat.MetaId,
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

		// 记录下一条消息的时间戳（用于分页）
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

// FetchGroupMemberList 获取群组成员列表
func FetchGroupMemberList(req *request.FetchGroupMemberListRequest) (*respond.GroupMemberResponse, error) {
	// 设置默认分页参数
	if req.Size <= 0 {
		req.Size = 20
	}
	if req.Cursor <= 0 {
		req.Cursor = 1
	}

	// 获取群组成员
	members, err := groupDB.GetGroupMembers(req.GroupId)
	if err != nil {
		return nil, err
	}

	// 转换为响应格式
	var memberItems []*respond.GroupMemberItem
	for _, member := range members {
		memberItem := &respond.GroupMemberItem{
			MetaId:    member.MetaId,
			Name:      "", // 暂未实现，需要从用户信息中获取
			Address:   member.Address,
			TimeStr:   time.Unix(member.Timestamp, 0).Format("2006-01-02 15:04:05"),
			Timestamp: member.Timestamp,
		}
		memberItems = append(memberItems, memberItem)
	}

	return &respond.GroupMemberResponse{
		Total: int64(len(memberItems)),
		List:  memberItems,
	}, nil
}

// FetchGroupPerson 获取群组成员信息
func FetchGroupPerson(req *request.FetchGroupPersonRequest) (*respond.GroupPersonResponse, error) {
	// 参数验证
	if req.MetaId == "" {
		return nil, fmt.Errorf("metaId is empty")
	}
	if req.GroupId == "" {
		return nil, fmt.Errorf("groupId is empty")
	}

	// 获取群组成员信息
	person, err := groupDB.GetGroupPersonByGroupIdAndMetaId(req.GroupId, req.MetaId)
	if err != nil {
		return nil, err
	}

	// 构建响应
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
