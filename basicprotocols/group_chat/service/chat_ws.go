package service

import (
	"manindexer/basicprotocols/group_chat/api/respond"
	"manindexer/basicprotocols/group_chat/models"
	"manindexer/basicprotocols/group_chat/service/common_service"
)

func wsPostGroupMsg(chat *models.TalkGroupChatV3) {
	// 1. 根据groupId获取groupInfo
	groupInfo, err := groupDB.GetGroupInfoByGroupId(chat.GroupId)
	if err != nil {
		return
	}

	// 2. 构建metaIdList
	metaIdList := make([]string, 0)

	// 检查是否有community
	if groupInfo != nil && groupInfo.CommunityId != "" {
		// 有community，获取community成员列表
		communityMembers, err := communityDB.GetCommunityMembers(groupInfo.CommunityId)
		if err == nil && communityMembers != nil {
			for _, member := range communityMembers {
				if member.MetaId != "" {
					metaIdList = append(metaIdList, member.MetaId)
				}
			}
		}
	} else {
		// 没有community，直接根据groupId获取member列表
		groupMembers, err := groupDB.GetGroupMembers(chat.GroupId)
		if err == nil && groupMembers != nil {
			for _, member := range groupMembers {
				if member.MetaId != "" {
					metaIdList = append(metaIdList, member.MetaId)
				}
			}
		}
	}

	// 4. 构建回复信息
	var replyInfo *respond.ReplyInfo
	replyMetaId := ""
	if chat.ReplyPin != "" {
		// 获取回复的消息
		replyChat, err := chatDB.GetChatByPinId(chat.ReplyPin)
		if err == nil && replyChat != nil {
			replyMetaId = replyChat.MetaId
			replyInfo = &respond.ReplyInfo{
				PinId:       replyChat.PinId,
				MetaId:      replyChat.MetaId,
				Address:     replyChat.Address,
				UserInfo:    nil,
				NickName:    replyChat.NickName,
				Protocol:    replyChat.Protocol,
				Content:     replyChat.Content,
				ContentType: replyChat.ContentType,
				Encryption:  replyChat.Encryption,
				ChatType:    replyChat.ChatType,
				Timestamp:   replyChat.Timestamp,
				Chain:       replyChat.Chain,
				BlockHeight: replyChat.BlockHeight,
			}
		}
	}

	// 5. 构建GroupChatItem
	groupChatItem := &respond.GroupChatItem{
		GroupId:     chat.GroupId,
		MetanetId:   chat.GroupId, // 使用GroupId作为MetanetId
		TxId:        chat.TxId,
		PinId:       chat.PinId,
		MetaId:      chat.MetaId,
		Address:     chat.Address,
		UserInfo:    nil,
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
	}

	// 6. 调用wsPost发送消息
	common_service.WsPost(chat.PinId, groupChatItem, metaIdList)
}
