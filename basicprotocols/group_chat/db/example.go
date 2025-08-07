package db

import (
	"log"
	"manindexer/basicprotocols/group_chat/models"
)

// 使用示例
func ExampleUsage() {
	// 初始化数据库
	pb := &Pebble{}
	err := pb.InitDatabase()
	defer pb.CloseAll()
	if err != nil {
		log.Printf("初始化数据库失败: %v", err)
		return
	}

	// 创建数据库操作实例
	communityDB := NewCommunityDB(pb)
	groupDB := NewGroupDB(pb)
	chatDB := NewChatDB(pb)

	// 示例1: 创建社区
	community := &models.TalkCommunityModel{
		CommunityId: "community_123",
		Name:        "测试社区",
		Description: "这是一个测试社区",
		MetaId:      "metaid_123",
		Address:     "address_123",
		PublicKey:   "publickey_123",
		TxId:        "tx_community_001",
		Timestamp:   1234567890,
	}

	err = communityDB.SaveCommunityInfo(community)
	if err != nil {
		log.Printf("保存社区失败: %v", err)
	} else {
		log.Printf("保存社区成功: %s", community.Name)
	}

	// 示例2: 保存社区版本信息
	err = communityDB.SaveCommunityVersionInfo(community)
	if err != nil {
		log.Printf("保存社区版本信息失败: %v", err)
	} else {
		log.Printf("保存社区版本信息成功")
	}

	// 示例3: 查询社区
	retrievedCommunity, err := communityDB.GetCommunityInfoByCommunityId("community_123")
	if err != nil {
		log.Printf("查询社区失败: %v", err)
	} else if retrievedCommunity != nil {
		log.Printf("查询到社区: %s", retrievedCommunity.Name)
	}

	// 示例4: 创建群组
	group := &models.TalkGroupModel{
		GroupId:      "group_456",
		CommunityId:  "community_123",
		RoomName:     "测试群组",
		RoomNote:     "这是一个测试群组",
		RoomType:     "1",
		RoomStatus:   "1",
		RoomJoinType: "1",
		TxId:         "tx_group_001",
		Timestamp:    1234567890,
	}

	err = groupDB.SaveGroupInfo(group)
	if err != nil {
		log.Printf("保存群组失败: %v", err)
	} else {
		log.Printf("保存群组成功: %s", group.RoomName)
	}

	// 示例5: 保存群组版本信息
	err = groupDB.SaveGroupVersionInfo(group)
	if err != nil {
		log.Printf("保存群组版本信息失败: %v", err)
	} else {
		log.Printf("保存群组版本信息成功")
	}

	// 示例6: 保存群组社区关联
	err = groupDB.SaveGroupCommunity(group)
	if err != nil {
		log.Printf("保存群组社区关联失败: %v", err)
	} else {
		log.Printf("保存群组社区关联成功")
	}

	// 示例7: 发送聊天消息
	chat := &models.TalkGroupChatV3{
		CommunityId: "community_123",
		GroupId:     "group_456",
		MetaId:      "metaid_123",
		NickName:    "测试用户",
		Protocol:    "chat",
		Content:     "Hello, World!",
		ContentType: "text",
		ChatType:    models.ChatTypeMsg,
		InsideIndex: models.ChatInsideIndexIn,
		TxId:        "tx_789",
		Timestamp:   1234567890,
	}

	err = chatDB.SaveChat(chat)
	if err != nil {
		log.Printf("保存聊天消息失败: %v", err)
	} else {
		log.Printf("保存聊天消息成功")
	}

	// 示例8: 查询群组聊天记录
	chats, err := chatDB.GetChatsByGroupId("group_456", 1, 10)
	if err != nil {
		log.Printf("查询聊天记录失败: %v", err)
	} else {
		log.Printf("查询到 %d 条聊天记录", len(chats))
		for _, c := range chats {
			log.Printf("消息: %s - %s", c.NickName, c.Content)
		}
	}

	// 示例9: 发送红包
	redEnvelope := &models.TalkGroupRedEnvelopeV3{
		CommunityId: "community_123",
		GroupId:     "group_456",
		MetaId:      "metaid_123",
		Protocol:    "red_envelope",
		Content:     "恭喜发财",
		Amount:      "100",
		Count:       "10",
		Type:        "btc",
		TxId:        "tx_red_001",
		Timestamp:   1234567890,
	}

	err = chatDB.SaveRedEnvelope(redEnvelope)
	if err != nil {
		log.Printf("保存红包失败: %v", err)
	} else {
		log.Printf("保存红包成功")
	}

	// 示例10: 抢红包
	openRedEnvelope := &models.TalkGroupOpenRedEnvelopeV3{
		CommunityId:       "community_123",
		GroupId:           "group_456",
		MetaId:            "metaid_456",
		Protocol:          "open_red_envelope",
		Address:           "address_456",
		Amount:            "10",
		RedEnvelopeTxId:   redEnvelope.TxId,
		RedEnvelopeMetaId: redEnvelope.MetaId,
		TxId:              "tx_open_001",
		Timestamp:         1234567890,
	}

	err = chatDB.SaveOpenRedEnvelope(openRedEnvelope)
	if err != nil {
		log.Printf("保存抢红包记录失败: %v", err)
	} else {
		log.Printf("保存抢红包记录成功")
	}

	// 示例11: 查询社区成员
	members, err := communityDB.GetCommunityMembers("community_123")
	if err != nil {
		log.Printf("查询社区成员失败: %v", err)
	} else {
		log.Printf("社区有 %d 个成员", len(members))
	}

	// 示例12: 查询社区群组
	groups, err := groupDB.GetGroupsByCommunityId("community_123")
	if err != nil {
		log.Printf("查询社区群组失败: %v", err)
	} else {
		log.Printf("社区有 %d 个群组", len(groups))
	}

	// 示例13: 加入社区
	join := &models.TalkCommunityJoinModel{
		CommunityId:    "community_123",
		MetaId:         "metaid_789",
		CommunityState: models.RoomStateIn,
		IsValid:        true,
		IsNew:          true,
		TxId:           "tx_join_001",
		Timestamp:      1234567890,
	}

	err = communityDB.SaveCommunityJoin(join)
	if err != nil {
		log.Printf("保存社区加入记录失败: %v", err)
	} else {
		log.Printf("保存社区加入记录成功")
	}

	// 示例14: 查询社区加入信息
	joinInfo, err := communityDB.GetCommunityJoinByCommunityIdAndPinId("community_123", "tx_join_001")
	if err != nil {
		log.Printf("查询社区加入信息失败: %v", err)
	} else if joinInfo != nil {
		log.Printf("查询到社区加入信息: %s", joinInfo.MetaId)
	}

	// 示例15: 保存社区成员信息
	person := &models.TalkCommunityPerson{
		CommunityId:    "community_123",
		MetaId:         "metaid_999",
		UserName:       "测试用户",
		UserNickName:   "昵称",
		CommunityState: models.RoomStateIn,
		Timestamp:      1234567890,
	}

	err = communityDB.SaveCommunityPerson(person)
	if err != nil {
		log.Printf("保存社区成员信息失败: %v", err)
	} else {
		log.Printf("保存社区成员信息成功")
	}
}
