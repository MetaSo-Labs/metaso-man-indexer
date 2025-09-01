package service

import (
	"manindexer/basicprotocols/group_chat/api/respond"
	"manindexer/basicprotocols/group_chat/models"
	"manindexer/basicprotocols/group_chat/service/common_service"
	"manindexer/basicprotocols/group_chat/service/socket_service"
)

func wsPostGroupMsg(chat *models.TalkGroupChatV3) {
	// 1. Get groupInfo by groupId
	groupInfo, err := groupDB.GetGroupInfoByGroupId(chat.GroupId)
	if err != nil {
		return
	}

	// 2. Build metaIdList
	metaIdList := make([]string, 0)

	// Check if there is a community
	if groupInfo != nil && groupInfo.CommunityId != "" {
		// Has community, get community member list
		communityMembers, err := communityDB.GetCommunityMembers(groupInfo.CommunityId)
		if err == nil && communityMembers != nil {
			for _, member := range communityMembers {
				if member.MetaId != "" {
					metaIdList = append(metaIdList, member.MetaId)
				}
			}
		}
	} else {
		// No community, directly get member list by groupId
		groupMembers, err := groupDB.GetGroupMembers(chat.GroupId)
		if err == nil && groupMembers != nil {
			for _, member := range groupMembers {
				if member.MetaId != "" {
					metaIdList = append(metaIdList, member.MetaId)
				}
			}
		}
	}

	// 4. Build reply information
	var replyInfo *respond.ReplyInfo
	replyMetaId := ""
	if chat.ReplyPin != "" {
		// Get the replied message
		replyChat, err := chatDB.GetChatByPinId(chat.ReplyPin)
		if err == nil && replyChat != nil {
			replyMetaId = replyChat.MetaId
			replyInfo = &respond.ReplyInfo{
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
				BlockHeight: replyChat.BlockHeight,
				Index:       replyChat.Index,
			}
		}
	}

	// 5. Build GroupChatItem
	groupChatItem := &respond.GroupChatItem{
		GroupId:     chat.GroupId,
		MetanetId:   chat.GroupId, // Use GroupId as MetanetId
		TxId:        chat.TxId,
		PinId:       chat.PinId,
		MetaId:      chat.MetaId,
		Address:     chat.Address,
		UserInfo:    common_service.FetchMetaIDUserInfo(chat.Address),
		NickName:    chat.NickName,
		Protocol:    chat.Protocol,
		Content:     chat.Content,
		ContentType: chat.ContentType,
		Encryption:  chat.Encryption,
		ChatType:    chat.ChatType,
		Data:        nil,
		ReplyPin:    chat.ReplyPin,
		ReplyInfo:   replyInfo,
		RedMetaId:   replyMetaId,
		Timestamp:   chat.Timestamp,
		Params:      "",
		Chain:       chat.Chain,
		BlockHeight: chat.BlockHeight,
		Index:       chat.Index,
	}

	// 6. Call wsPost to send message
	for _, metaId := range metaIdList {
		socket_service.SendGroupMessageToUser(metaId, groupChatItem)
	}
	// common_service.WsPost(chat.PinId, groupChatItem, metaIdList)

}

func wsPostPrivateMsg(chat *models.TalkPrivateChatV3) {
	metaIdList := make([]string, 0)
	metaIdList = append(metaIdList, chat.From)
	metaIdList = append(metaIdList, chat.To)

	// 4. Build reply information
	var replyInfo *respond.ReplyInfo
	replyMetaId := ""
	if chat.ReplyPin != "" {
		// Get the replied message
		replyChat, err := privateDB.GetPrivateChatByPinId(chat.ReplyPin)
		if err == nil && replyChat != nil {
			replyMetaId = replyChat.From
			replyInfo = &respond.ReplyInfo{
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
				BlockHeight: replyChat.BlockHeight,
				Index:       replyChat.Index,
			}
		}
	}

	// 5. Build PrivateChatItem
	privateChatItem := &respond.PrivateChatItem{
		From:         chat.From,
		FromUserInfo: common_service.FetchMetaIDUserInfoInfoByMetaId(chat.From),
		To:           chat.To,
		ToUserInfo:   common_service.FetchMetaIDUserInfoInfoByMetaId(chat.To),
		TxId:         chat.TxId,
		PinId:        chat.PinId,
		MetaId:       chat.From,
		Address:      chat.FromAddress,
		UserInfo:     common_service.FetchMetaIDUserInfoInfoByMetaId(chat.From),
		NickName:     "",
		Protocol:     chat.Protocol,
		Content:      chat.Content,
		ContentType:  chat.ContentType,
		Encryption:   chat.Encryption,
		ChatType:     int64(chat.ChatType),
		ReplyPin:     chat.ReplyPin,
		ReplyInfo:    replyInfo,
		RedMetaId:    replyMetaId,
		Timestamp:    chat.Timestamp,
		Chain:        chat.Chain,
		BlockHeight:  chat.BlockHeight,
		Index:        chat.Index,
	}

	// 6. Call wsPost to send message
	socket_service.SendPrivateMessageToUser(chat.From, privateChatItem)
	socket_service.SendPrivateMessageToUser(chat.To, privateChatItem)
	// common_service.WsPost(chat.PinId, privateChatItem, metaIdList)
}
