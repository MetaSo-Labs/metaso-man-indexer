package service

// import (
// 	"manindexer/basicprotocols/group_chat/api/respond"
// 	"manindexer/basicprotocols/group_chat/db"
// 	"manindexer/basicprotocols/group_chat/models"
// 	"strconv"
// )

// // GetLuckyBagWithOpenList 根据groupId和pinId获取红包对象和已领取列表
// func GetLuckyBagWithOpenList(groupId, pinId string) (*respond.LuckyBagInfoResponse, error) {
// 	chatDB := db.NewChatDB(&db.Pebble{})

// 	// 获取红包对象
// 	luckyBag, err := chatDB.GetLuckyBagByPinId(pinId)
// 	if err != nil {
// 		return nil, nil, nil, err
// 	}

// 	if luckyBag == nil {
// 		return nil, nil, nil, nil
// 	}

// 	// 验证groupId是否匹配
// 	if luckyBag.GroupId != groupId {
// 		return nil, nil, nil, nil
// 	}

// 	// 获取已领取的抢红包列表
// 	openList, err := chatDB.GetOpenLuckyBagList(pinId)
// 	if err != nil {
// 		return nil, nil, nil, err
// 	}

// 	// 获取已领取的回收红包列表
// 	residueList, err := chatDB.GetResidueLuckyBagList(pinId)
// 	if err != nil {
// 		return nil, nil, nil, err
// 	}

// 	return luckyBag, openList, residueList, nil
// }

// // GetLuckyBagDetail 获取红包详细信息，包括已领取的列表
// func GetLuckyBagDetail(groupId, pinId string) (*models.TalkGroupLuckyBagV3, []*models.TalkGroupOpenLuckyBagV3, []*models.TalkGroupResidueLuckyBagV3, error) {
// 	chatDB := db.NewChatDB(&db.Pebble{})

// 	// 获取红包对象
// 	luckyBag, err := chatDB.GetLuckyBagByPinId(pinId)
// 	if err != nil {
// 		return nil, nil, nil, err
// 	}

// 	if luckyBag == nil {
// 		return nil, nil, nil, nil
// 	}

// 	// 验证groupId是否匹配
// 	if luckyBag.GroupId != groupId {
// 		return nil, nil, nil, nil
// 	}

// 	// 获取已领取的抢红包详细列表
// 	openLuckyBags, err := chatDB.GetOpenLuckyBagsByLuckyBagTxId(luckyBag.TxId)
// 	if err != nil {
// 		return nil, nil, nil, err
// 	}

// 	// 获取已领取的回收红包详细列表
// 	residueLuckyBags := make([]*models.TalkGroupResidueLuckyBagV3, 0)
// 	// 注意：这里需要根据实际情况调整，因为回收红包可能没有直接的查询方法
// 	// 暂时返回空列表，后续可以根据需要添加查询方法

// 	return luckyBag, openLuckyBags, residueLuckyBags, nil
// }

// // GetLuckyBagStatistics 获取红包统计信息
// func GetLuckyBagStatistics(groupId, pinId string) (*models.LuckyBagStatistics, error) {
// 	luckyBag, openList, residueList, err := GetLuckyBagWithOpenList(groupId, pinId)
// 	if err != nil {
// 		return nil, err
// 	}

// 	if luckyBag == nil {
// 		return nil, nil
// 	}

// 	// 计算统计信息
// 	totalCount := len(luckyBag.PayList)
// 	openedCount := len(openList.Items)
// 	residueCount := len(residueList.Items)

// 	// 计算已领取金额
// 	openedAmount := uint64(0)
// 	// 这里需要根据openList.Items获取具体的金额信息
// 	// 暂时使用默认值，后续可以根据需要完善
// 	_ = openList.Items // 避免未使用变量警告

// 	// 计算剩余金额
// 	totalAmount := uint64(0)
// 	for _, payItem := range luckyBag.PayList {
// 		if amount, err := strconv.ParseUint(payItem.Amount, 10, 64); err == nil {
// 			totalAmount += amount
// 		}
// 	}
// 	remainingAmount := totalAmount - openedAmount

// 	statistics := &models.LuckyBagStatistics{
// 		LuckyBagPinId:       pinId,
// 		GroupId:             groupId,
// 		TotalCount:          totalCount,
// 		OpenedCount:         openedCount,
// 		ResidueCount:        residueCount,
// 		TotalAmount:         totalAmount,
// 		OpenedAmount:        openedAmount,
// 		RemainingAmount:     remainingAmount,
// 		IsCompleted:         openedCount >= totalCount,
// 		CreateTime:          luckyBag.CreateTimeStr,
// 		Content:             luckyBag.Content,
// 		Type:                luckyBag.Type,
// 		RequireType:         luckyBag.RequireType,
// 		RequireTickId:       luckyBag.RequireTickId,
// 		RequireCollectionId: luckyBag.RequireCollectionId,
// 		LimitAmount:         luckyBag.LimitAmount,
// 	}

// 	return statistics, nil
// }
