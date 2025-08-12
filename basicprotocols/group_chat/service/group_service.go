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
	groupDB   *db.GroupDB
	chatDB    *db.ChatDB
	privateDB *db.PrivateChatDB
	pebbleDB  *db.Pebble
)

// InitService 初始化服务
func InitService(indexer *indexer.GroupChatIndexer) error {
	// 使用 indexer 的数据库实例

	// 获取数据库实例
	pebbleDB = indexer.GetPebble()
	groupDB = indexer.GetGroupDB()
	chatDB = indexer.GetChatDB()
	privateDB = indexer.GetPrivateDB()

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

	var groups []*models.TalkGroupModel
	var err error

	// 如果metaId不为空，获取用户加入的群组列表
	if req.MetaId != "" {
		groups, err = groupDB.GetGroupListByMetaId(req.MetaId, req.Cursor, req.Size)
	} else {
		// 获取所有群组列表
		groups, err = groupDB.GetGroupList(req.Cursor, req.Size)
	}

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
			Address:     chat.Address,
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
				Address:     replyChat.Address,
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

// FetchLatestChatInfoList 获取最新聊天信息列表（群聊+私聊）
func FetchLatestChatInfoList(req *request.FetchLatestChatInfoListRequest) (*respond.ChatInfoResponse, error) {
	// 设置默认分页参数
	if req.Size <= 0 {
		req.Size = 20
	}
	if req.Cursor <= 0 {
		req.Cursor = 1
	}

	// 获取用户的上下文列表（群聊+私聊）
	contextList, err := chatDB.GetMetaIdContextList(req.MetaId)
	if err != nil {
		return nil, err
	}

	// 转换为响应格式
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

		// 根据类型处理不同字段
		if item.Type == "1" || item.Type == "" {
			item.Type = "1"
			// 群聊类型，获取群组详细信息
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

			// 填充群聊特有字段
			chatInfoItem.CommunityId = group.CommunityId
			chatInfoItem.RoomName = group.RoomName
			chatInfoItem.RoomNote = group.RoomNote
			chatInfoItem.RoomType = group.RoomType
			chatInfoItem.RoomStatus = group.RoomStatus
			chatInfoItem.RoomJoinType = group.RoomJoinType
			chatInfoItem.RoomAvatarUrl = group.RoomAvatarUrl
			chatInfoItem.CreateUserMetaId = group.CreateUserMetaId
			chatInfoItem.UserCount = 0 // 需要计算
			chatInfoItem.ChatSettingType = group.ChatSettingType
			chatInfoItem.DeleteStatus = group.DeleteStatus
			chatInfoItem.Chain = group.Chain

			// 如果获取到了最新聊天信息，更新相关字段
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
			// 私聊类型，获取私聊最新消息
			latestPrivateChat, err := privateDB.GetPrivateChatByPinId(item.LastMessagePinId)
			if err != nil {
				// 如果获取失败，使用默认值
				latestPrivateChat = nil
			}

			// 如果获取到了最新私聊信息，更新相关字段
			if latestPrivateChat != nil {
				chatInfoItem.Content = latestPrivateChat.Content
				chatInfoItem.LastMessagePinId = latestPrivateChat.PinId
				chatInfoItem.Timestamp = latestPrivateChat.Timestamp
				chatInfoItem.ChatType = int64(latestPrivateChat.ChatType)
				chatInfoItem.CreateMetaId = latestPrivateChat.From
				chatInfoItem.CreateAddress = latestPrivateChat.FromAddress
				chatInfoItem.BlockHeight = latestPrivateChat.BlockHeight
				chatInfoItem.Chain = latestPrivateChat.Chain
			}
		}

		chatInfoItems = append(chatInfoItems, chatInfoItem)
	}

	return &respond.ChatInfoResponse{
		Total: int64(len(chatInfoItems)),
		List:  chatInfoItems,
	}, nil
}

// FetchPrivateChatList 获取私聊记录列表
func FetchPrivateChatList(req *request.FetchPrivateChatListRequest) (*respond.PrivateChatResponse, error) {
	// 设置默认分页参数
	if req.Size <= 0 {
		req.Size = 20
	}

	var chats []*models.TalkPrivateChatV3
	var err error

	if req.Timestamp > 0 {
		// 根据时间戳范围获取私聊记录
		chats, err = privateDB.GetPrivateChatsByMetaIdsAndTimestampRange(req.MetaId, req.OtherMetaId, req.Timestamp, req.Size)
	} else {
		// 获取最新的私聊记录
		chats, err = privateDB.GetLatestPrivateChatsByMetaIds(req.MetaId, req.OtherMetaId, req.Size)
	}

	if err != nil {
		return nil, err
	}
	fmt.Printf("private chats: %+v\n", chats)

	// 转换为响应格式
	var chatItems []*respond.PrivateChatItem
	var nextTimestamp int64 = 0

	for i, chat := range chats {
		chatItem := &respond.PrivateChatItem{
			From:        chat.From,
			To:          chat.To,
			TxId:        chat.TxId,
			PinId:       chat.PinId,
			MetaId:      chat.From, // 消息创建者MetaId
			NickName:    "",        // 需要从用户信息中获取
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

		// 处理回复消息
		if chat.ReplyPin != "" {
			replyChat, err := privateDB.GetPrivateChatByPinId(chat.ReplyPin)
			if err != nil {
				replyChat = nil
			}
			if replyChat != nil {
				chatItem.ReplyInfo = &respond.ReplyInfo{
					PinId:       replyChat.PinId,
					MetaId:      replyChat.From,
					NickName:    "", // 私聊消息没有NickName字段
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

		// 记录下一条消息的时间戳（用于分页）
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
