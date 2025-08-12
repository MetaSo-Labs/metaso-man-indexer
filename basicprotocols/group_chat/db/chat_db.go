package db

import (
	"encoding/json"
	"fmt"
	"log"
	"manindexer/basicprotocols/group_chat/models"
	"manindexer/basicprotocols/group_chat/protocols"
	"manindexer/pin"
	"strconv"
	"strings"
	"time"

	"github.com/cockroachdb/pebble"
)

// 队列消息项
type QueueChatMessage struct {
	PinId      string                  `json:"pinId"`      // 消息PinId
	GroupId    string                  `json:"groupId"`    // 群组ID
	Chat       *models.TalkGroupChatV3 `json:"chat"`       // 聊天消息
	Timestamp  int64                   `json:"timestamp"`  // 入队时间戳
	RetryCount int                     `json:"retryCount"` // 重试次数
	Status     string                  `json:"status"`     // 处理状态：pending, processing, completed, failed
}

// 聊天数据库操作
type ChatDB struct {
	pb *Pebble
}

func NewChatDB(pb *Pebble) *ChatDB {
	return &ChatDB{pb: pb}
}

// 保存聊天消息
func (cdb *ChatDB) SaveChat(chat *models.TalkGroupChatV3) error {
	data, err := json.Marshal(chat)
	if err != nil {
		return err
	}

	// 使用 PinId 作为主键
	key := []byte(chat.PinId)
	return Pb[TalkGroupChatPinCollection].Set(key, data, pebble.Sync)
}

// 保存聊天时间戳索引
func (cdb *ChatDB) SaveChatTimestamp(chat *models.TalkGroupChatV3) error {
	// 构造时间戳索引值：pinId_chatType_timestamp
	value := chat.PinId + "_" + strconv.FormatInt(int64(chat.ChatType), 10) + "_" + strconv.FormatInt(chat.Timestamp, 10)

	// 使用 GroupId_Timestamp 作为主键，支持按时间戳范围查询
	key := []byte(chat.GroupId + "_" + strconv.FormatInt(chat.Timestamp, 10))
	return Pb[TalkGroupChatTimestampCollection].Set(key, []byte(value), pebble.Sync)
}

// 根据PinId获取聊天消息
func (cdb *ChatDB) GetChatByPinId(pinId string) (*models.TalkGroupChatV3, error) {
	key := []byte(pinId)
	value, closer, err := Pb[TalkGroupChatPinCollection].Get(key)
	if err != nil {
		if err == pebble.ErrNotFound {
			return nil, nil
		}
		return nil, err
	}
	defer closer.Close()

	var chat models.TalkGroupChatV3
	err = json.Unmarshal(value, &chat)
	if err != nil {
		return nil, err
	}

	return &chat, nil
}

// 根据群组ID获取聊天消息列表
func (cdb *ChatDB) GetChatsByGroupId(groupId string, page, size int64) ([]*models.TalkGroupChatV3, error) {
	var chats []*models.TalkGroupChatV3
	iter, err := Pb[TalkGroupChatPinCollection].NewIter(nil)
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	count := int64(0)
	skip := (page - 1) * size

	for iter.First(); iter.Valid(); iter.Next() {
		var chat models.TalkGroupChatV3
		err := json.Unmarshal(iter.Value(), &chat)
		if err != nil {
			continue
		}
		if chat.GroupId == groupId {
			if count < skip {
				count++
				continue
			}

			if int64(len(chats)) >= size {
				break
			}

			chats = append(chats, &chat)
		}
	}

	return chats, nil
}

// 根据社区ID获取聊天消息列表
func (cdb *ChatDB) GetChatsByCommunityId(communityId string, page, size int64) ([]*models.TalkGroupChatV3, error) {
	var chats []*models.TalkGroupChatV3
	iter, err := Pb[TalkGroupChatPinCollection].NewIter(nil)
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	count := int64(0)
	skip := (page - 1) * size

	for iter.First(); iter.Valid(); iter.Next() {
		var chat models.TalkGroupChatV3
		err := json.Unmarshal(iter.Value(), &chat)
		if err != nil {
			continue
		}
		if chat.CommunityId == communityId {
			if count < skip {
				count++
				continue
			}

			if int64(len(chats)) >= size {
				break
			}

			chats = append(chats, &chat)
		}
	}

	return chats, nil
}

// 根据群组ID和开始时间戳获取聊天消息列表（倒序，基于时间戳分页）
func (cdb *ChatDB) GetChatsByGroupIdAndTimestampRange(groupId string, startTimestamp int64, size int64) ([]*models.TalkGroupChatV3, error) {
	var chats []*models.TalkGroupChatV3
	iter, err := Pb[TalkGroupChatTimestampCollection].NewIter(nil)
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	// 构造查询起始键：groupId_startTimestamp
	startKey := []byte(groupId + "_" + strconv.FormatInt(startTimestamp, 10))

	// 从指定时间戳开始倒序遍历（最新的消息在前）
	for iter.SeekLT(startKey); iter.Valid() && iter.Key() != nil; iter.Prev() {
		key := string(iter.Key())

		// 检查是否属于指定群组
		if !strings.HasPrefix(key, groupId+"_") {
			continue
		}

		// 解析索引值获取 PinId
		value := string(iter.Value())
		valueParts := strings.Split(value, "_")
		if len(valueParts) < 1 {
			continue
		}
		pinId := valueParts[0]

		// 获取完整的聊天消息
		chat, err := cdb.GetChatByPinId(pinId)
		if err != nil || chat == nil {
			continue
		}

		// 达到分页大小限制
		if int64(len(chats)) >= size {
			break
		}

		chats = append(chats, chat)
	}

	return chats, nil
}

// 获取群组的最新聊天消息（基于时间戳倒序）
func (cdb *ChatDB) GetLatestChatsByGroupId(groupId string, size int64) ([]*models.TalkGroupChatV3, error) {
	// 使用当前时间作为起始时间戳
	currentTimestamp := time.Now().Unix()
	return cdb.GetChatsByGroupIdAndTimestampRange(groupId, currentTimestamp, size)
}

// 删除聊天消息
func (cdb *ChatDB) DeleteChat(pinId string) error {
	key := []byte(pinId)
	return Pb[TalkGroupChatPinCollection].Delete(key, pebble.Sync)
}

// 保存红包信息
func (cdb *ChatDB) SaveLuckyBag(red *models.TalkGroupLuckyBagV3) error {
	data, err := json.Marshal(red)
	if err != nil {
		return err
	}

	// 使用 PinId 作为主键
	key := []byte(red.PinId)
	return Pb[TalkGroupLuckyBagPinCollection].Set(key, data, pebble.Sync)
}

// 根据PinId获取红包信息
func (cdb *ChatDB) GetLuckyBagByPinId(pinId string) (*models.TalkGroupLuckyBagV3, error) {
	key := []byte(pinId)
	value, closer, err := Pb[TalkGroupLuckyBagPinCollection].Get(key)
	if err != nil {
		if err == pebble.ErrNotFound {
			return nil, nil
		}
		return nil, err
	}
	defer closer.Close()

	var red models.TalkGroupLuckyBagV3
	err = json.Unmarshal(value, &red)
	if err != nil {
		return nil, err
	}

	return &red, nil
}

// 根据群组ID获取红包列表
func (cdb *ChatDB) GetLuckyBagsByGroupId(groupId string) ([]*models.TalkGroupLuckyBagV3, error) {
	var reds []*models.TalkGroupLuckyBagV3
	iter, err := Pb[TalkGroupLuckyBagPinCollection].NewIter(nil)
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	for iter.First(); iter.Valid(); iter.Next() {
		var red models.TalkGroupLuckyBagV3
		err := json.Unmarshal(iter.Value(), &red)
		if err != nil {
			continue
		}
		if red.GroupId == groupId {
			reds = append(reds, &red)
		}
	}

	return reds, nil
}

// 保存抢红包信息
func (cdb *ChatDB) SaveOpenLuckyBag(open *models.TalkGroupOpenLuckyBagV3) error {
	data, err := json.Marshal(open)
	if err != nil {
		return err
	}

	// 使用 PinId 作为主键
	key := []byte(open.PinId)
	return Pb[TalkGroupOpenLuckyBagPinCollection].Set(key, data, pebble.Sync)
}

// 根据PinId获取抢红包信息
func (cdb *ChatDB) GetOpenLuckyBagByPinId(pinId string) (*models.TalkGroupOpenLuckyBagV3, error) {
	key := []byte(pinId)
	value, closer, err := Pb[TalkGroupOpenLuckyBagPinCollection].Get(key)
	if err != nil {
		if err == pebble.ErrNotFound {
			return nil, nil
		}
		return nil, err
	}
	defer closer.Close()

	var open models.TalkGroupOpenLuckyBagV3
	err = json.Unmarshal(value, &open)
	if err != nil {
		return nil, err
	}

	return &open, nil
}

// 根据红包TxId获取抢红包列表
func (cdb *ChatDB) GetOpenLuckyBagsByLuckyBagTxId(redEnvelopeTxId string) ([]*models.TalkGroupOpenLuckyBagV3, error) {
	var opens []*models.TalkGroupOpenLuckyBagV3
	iter, err := Pb[TalkGroupOpenLuckyBagPinCollection].NewIter(nil)
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	for iter.First(); iter.Valid(); iter.Next() {
		var open models.TalkGroupOpenLuckyBagV3
		err := json.Unmarshal(iter.Value(), &open)
		if err != nil {
			continue
		}
		if open.LuckyBagTxId == redEnvelopeTxId {
			opens = append(opens, &open)
		}
	}

	return opens, nil
}

// 保存剩余红包信息
func (cdb *ChatDB) SaveResidueLuckyBag(residue *models.TalkGroupResidueLuckyBagV3) error {
	data, err := json.Marshal(residue)
	if err != nil {
		return err
	}

	// 使用 PinId 作为主键
	key := []byte(residue.PinId)
	return Pb[TalkGroupResidueLuckyBagPinCollection].Set(key, data, pebble.Sync)
}

// 根据红包PinId获取剩余红包信息
func (cdb *ChatDB) GetResidueLuckyBagByLuckyBagPinId(redEnvelopePinId string) (*models.TalkGroupResidueLuckyBagV3, error) {
	key := []byte(redEnvelopePinId)
	value, closer, err := Pb[TalkGroupResidueLuckyBagPinCollection].Get(key)
	if err != nil {
		if err == pebble.ErrNotFound {
			return nil, nil
		}
		return nil, err
	}
	defer closer.Close()

	var residue models.TalkGroupResidueLuckyBagV3
	err = json.Unmarshal(value, &residue)
	if err != nil {
		return nil, err
	}

	return &residue, nil
}

// 获取用户的群列表
func (cdb *ChatDB) GetMetaIdContextList(metaId string) (*models.MetaIdContextList, error) {
	key := []byte(metaId)
	value, closer, err := Pb[TalkMetaIdContextListCollection].Get(key)
	if err != nil {
		if err == pebble.ErrNotFound {
			return &models.MetaIdContextList{MetaId: metaId, Items: []*models.MetaIdContextItem{}}, nil
		}
		return nil, err
	}
	defer closer.Close()

	var contextList models.MetaIdContextList
	err = json.Unmarshal(value, &contextList)
	if err != nil {
		return nil, err
	}

	return &contextList, nil
}

// 保存用户的群列表
func (cdb *ChatDB) SaveMetaIdContextList(contextList *models.MetaIdContextList) error {
	data, err := json.Marshal(contextList)
	if err != nil {
		return err
	}

	key := []byte(contextList.MetaId)
	return Pb[TalkMetaIdContextListCollection].Set(key, data, pebble.Sync)
}

// 更新群组中所有成员的群列表（当有新消息时）
func (cdb *ChatDB) UpdateGroupMembersContextList(groupId string, chat *models.TalkGroupChatV3, groupDB *GroupDB) error {
	// 更新群组最新聊天记录
	err := cdb.updateGroupLatestChat(groupId, chat)
	if err != nil {
		return err
	}

	// 获取群组的所有成员
	members, err := groupDB.GetGroupMembers(groupId)
	if err != nil {
		return err
	}

	// 为每个成员更新群列表
	for _, member := range members {
		err = cdb.updateSingleMemberContextList(member.MetaId, groupId, chat)
		if err != nil {
			return err
		}
	}

	return nil
}

// 更新群组最新聊天记录
func (cdb *ChatDB) updateGroupLatestChat(groupId string, chat *models.TalkGroupChatV3) error {
	// 先获取现有的最新聊天记录
	existingLatestChat, err := cdb.GetGroupLatestChat(groupId)
	if err != nil {
		return err
	}

	// 创建新的最新聊天记录
	newLatestChat := &models.TalkGroupLatestChat{
		GroupId:          groupId,
		Timestamp:        chat.Timestamp,
		ChatType:         chat.ChatType,
		Content:          chat.Content,
		CreateAddress:    chat.Address,
		LastMessagePinId: chat.PinId,
		MetaId:           chat.MetaId,
		TxId:             chat.TxId,
		PinId:            chat.PinId,
		Protocol:         chat.Protocol,
		ContentType:      chat.ContentType,
		Encryption:       chat.Encryption,
		ReplyPin:         chat.ReplyPin,
		Chain:            chat.Chain,
		BlockHeight:      chat.BlockHeight,
	}

	// 如果不存在现有数据，直接保存
	if existingLatestChat == nil {
		data, err := json.Marshal(newLatestChat)
		if err != nil {
			return err
		}
		key := []byte(groupId)
		return Pb[TalkGroupLatestChatCollection].Set(key, data, pebble.Sync)
	}

	// 判断是否需要更新
	shouldUpdate := false

	// 1. 先判断pinId是否一样
	if existingLatestChat.LastMessagePinId == chat.PinId {
		// pinId一样，检查blockHeight是否不一样
		if existingLatestChat.BlockHeight != chat.BlockHeight {
			shouldUpdate = true
		}
	} else {
		// pinId不一样，比较timestamp
		if chat.Timestamp > existingLatestChat.Timestamp {
			shouldUpdate = true
		}
	}

	// 如果需要更新，则保存新数据
	if shouldUpdate {
		data, err := json.Marshal(newLatestChat)
		if err != nil {
			return err
		}
		key := []byte(groupId)
		return Pb[TalkGroupLatestChatCollection].Set(key, data, pebble.Sync)
	}

	// 不需要更新，直接返回
	return nil
}

// 获取群组最新聊天记录
func (cdb *ChatDB) GetGroupLatestChat(groupId string) (*models.TalkGroupLatestChat, error) {
	key := []byte(groupId)
	value, closer, err := Pb[TalkGroupLatestChatCollection].Get(key)
	if err != nil {
		if err == pebble.ErrNotFound {
			return nil, nil
		}
		return nil, err
	}
	defer closer.Close()

	var latestChat models.TalkGroupLatestChat
	err = json.Unmarshal(value, &latestChat)
	if err != nil {
		return nil, err
	}

	return &latestChat, nil
}

// 更新单个成员的群列表
func (cdb *ChatDB) updateSingleMemberContextList(metaId, groupId string, chat *models.TalkGroupChatV3) error {
	// 获取用户的群列表
	contextList, err := cdb.GetMetaIdContextList(metaId)
	if err != nil {
		return err
	}

	// 创建新的群列表项
	newItem := &models.MetaIdContextItem{
		GroupId:          groupId,
		MetaId:           "",
		Type:             "1",
		Timestamp:        chat.Timestamp,
		ChatType:         chat.ChatType,
		Content:          chat.Content,
		CreateMetaId:     chat.MetaId,
		CreateAddress:    chat.Address,
		LastMessagePinId: chat.PinId,
		BlockHeight:      chat.BlockHeight,
	}

	//是否需要更新
	shouldUpdate := false
	// 查找是否已存在该群组的项
	found := false
	for i, item := range contextList.Items {
		if item.GroupId == groupId {
			// 更新现有项
			contextList.Items[i] = newItem
			found = true
			if item.LastMessagePinId != newItem.LastMessagePinId ||
				item.BlockHeight != newItem.BlockHeight {
				shouldUpdate = true
			}
			break
		}
	}

	// 如果不存在，添加新项
	if !found {
		contextList.Items = append(contextList.Items, newItem)
	}

	// 如果不需要更新，直接返回
	if !shouldUpdate {
		return nil
	}

	// 按时间戳倒序排序
	cdb.sortContextListByTimestamp(contextList)

	// 保存更新后的群列表
	return cdb.SaveMetaIdContextList(contextList)
}

// 按时间戳倒序排序群列表
func (cdb *ChatDB) sortContextListByTimestamp(contextList *models.MetaIdContextList) {
	// 简单的冒泡排序，按时间戳倒序
	for i := 0; i < len(contextList.Items)-1; i++ {
		for j := 0; j < len(contextList.Items)-1-i; j++ {
			if contextList.Items[j].Timestamp < contextList.Items[j+1].Timestamp {
				contextList.Items[j], contextList.Items[j+1] = contextList.Items[j+1], contextList.Items[j]
			}
		}
	}
}

// 将聊天消息加入队列（异步处理群列表更新）
func (cdb *ChatDB) EnqueueChatMessage(chat *models.TalkGroupChatV3) error {
	queueMessage := &QueueChatMessage{
		PinId:      chat.PinId,
		GroupId:    chat.GroupId,
		Chat:       chat,
		Timestamp:  time.Now().Unix(),
		RetryCount: 0,
		Status:     "pending",
	}

	data, err := json.Marshal(queueMessage)
	if err != nil {
		return err
	}

	// 使用 timestamp_pinId 作为主键，支持按时间顺序处理
	key := []byte(strconv.FormatInt(queueMessage.Timestamp, 10) + "_" + chat.PinId)
	return Pb[TalkGroupChatQueueCollection].Set(key, data, pebble.Sync)
}

// 获取队列中的待处理消息
func (cdb *ChatDB) GetPendingQueueMessages(limit int) ([]*QueueChatMessage, error) {
	var messages []*QueueChatMessage
	iter, err := Pb[TalkGroupChatQueueCollection].NewIter(nil)
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	count := 0
	for iter.First(); iter.Valid() && iter.Key() != nil && count < limit; iter.Next() {
		value := string(iter.Value())

		var queueMessage QueueChatMessage
		err := json.Unmarshal([]byte(value), &queueMessage)
		if err != nil {
			continue
		}

		// 只处理pending状态的消息
		if queueMessage.Status == "pending" {
			messages = append(messages, &queueMessage)
			count++
		}
	}

	return messages, nil
}

// 删除队列消息数据
func (cdb *ChatDB) deleteQueueMessage(pinId string) error {
	iter, err := Pb[TalkGroupChatQueueCollection].NewIter(nil)
	if err != nil {
		return err
	}
	defer iter.Close()

	// 查找包含该pinId的队列消息
	for iter.First(); iter.Valid(); iter.Next() {
		value := string(iter.Value())

		var queueMessage QueueChatMessage
		err := json.Unmarshal([]byte(value), &queueMessage)
		if err != nil {
			continue
		}

		// 找到匹配的pinId，删除该队列消息
		if queueMessage.PinId == pinId {
			return Pb[TalkGroupChatQueueCollection].Delete(iter.Key(), pebble.Sync)
		}
	}

	return nil
}

// 批量处理队列消息（异步更新群列表）
func (cdb *ChatDB) ProcessQueueMessages(groupDB *GroupDB, batchSize int) error {
	// 获取待处理的消息
	messages, err := cdb.GetPendingQueueMessages(batchSize)
	if err != nil {
		return err
	}

	// 批量处理消息
	for _, message := range messages {
		// 更新群组所有成员的群列表
		err = cdb.UpdateGroupMembersContextList(message.GroupId, message.Chat, groupDB)
		if err != nil {
			// 处理失败，记录错误但不删除队列消息，可以稍后重试
			log.Printf("Failed to process queue message for pinId %s: %v", message.PinId, err)
			continue
		}

		// 处理成功，删除队列消息
		err = cdb.deleteQueueMessage(message.PinId)
		if err != nil {
			// 记录错误但不影响主流程
			log.Printf("Failed to delete queue message for pinId %s: %v", message.PinId, err)
		}
	}

	return nil
}

// 启动队列处理协程（需要在应用启动时调用）
func (cdb *ChatDB) StartQueueProcessor(groupDB *GroupDB) {
	go func() {
		ticker := time.NewTicker(5 * time.Second) // 每5秒处理一次
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				// 批量处理队列消息
				err := cdb.ProcessQueueMessages(groupDB, 100) // 每次处理100条消息
				if err != nil {
					// 记录错误日志
					continue
				}
			}
		}
	}()
}

// 总的处理 Group Chat 方法
func (cdb *ChatDB) ProcessGroupChatPin(pin *pin.PinInscription) error {
	switch pin.Operation {
	case "create":
		path := pin.Path
		protocol := strings.Replace(path, "/protocols/", "", -1)
		if strings.ToLower(protocol) == strings.ToLower(protocols.MonitorSimpleGroupChat) {
			return cdb.processGroupChat(pin)
		} else if strings.ToLower(protocol) == strings.ToLower(protocols.MonitorSimpleFileGroupChat) {
			return cdb.processFileGroupChat(pin)
		} else if strings.ToLower(protocol) == strings.ToLower(protocols.MonitorSimpleGroupLuckyBag) {
			return cdb.processGroupLuckyBag(pin)
		} else if strings.ToLower(protocol) == strings.ToLower(protocols.MonitorSimpleGroupOpenLuckyBag) {
			return cdb.processGroupOpenLuckyBag(pin)
		} else if strings.ToLower(protocol) == strings.ToLower(protocols.MonitorSimpleGroupResidueLuckyBag) {
			return cdb.processGroupResidueLuckyBag(pin)
		}
	default:
		return nil // 未知操作类型，跳过
	}
	return nil
}

// 处理群组聊天
func (cdb *ChatDB) processGroupChat(pin *pin.PinInscription) error {
	// 检查是否已经保存过该 PinId
	existingChat, err := cdb.GetChatByPinId(pin.Id)
	if err == nil && existingChat != nil {
		if existingChat.BlockHeight != pin.GenesisHeight {
			existingChat.BlockHeight = pin.GenesisHeight
			err = cdb.SaveChat(existingChat)
			if err != nil {
				return err
			}
		}
		// 已经存在，跳过处理
		return nil
	}

	// 解析协议数据
	var simpleGroupChat protocols.SimpleGroupChat
	err = json.Unmarshal(pin.ContentBody, &simpleGroupChat)
	if err != nil {
		return err
	}

	// 创建聊天消息模型
	chat := &models.TalkGroupChatV3{
		GroupId:     simpleGroupChat.GroupId,
		TxId:        pin.Id[:len(pin.Id)-2],
		PinId:       pin.Id,
		MetaId:      pin.CreateMetaId,
		Address:     pin.CreateAddress,
		Protocol:    pin.Path,
		Content:     simpleGroupChat.Content,
		ContentType: simpleGroupChat.ContentType,
		Encryption:  simpleGroupChat.Encryption,
		ChatType:    models.ChatTypeMsg,       // 默认为消息类型
		InsideIndex: models.ChatInsideIndexIn, // 默认为进入状态
		ReplyPin:    simpleGroupChat.ReplyPin,
		Timestamp:   pin.Timestamp,
		Chain:       pin.ChainName,
		BlockHeight: pin.GenesisHeight,
	}

	// 保存聊天消息到 TalkGroupChatPinCollection
	err = cdb.SaveChat(chat)
	if err != nil {
		return err
	}

	// 保存时间戳索引（根据用户状态决定保存到哪个集合）
	err = cdb.SaveChatTimestampWithState(chat)
	if err != nil {
		return err
	}

	// 将消息加入队列，异步更新群列表
	err = cdb.EnqueueChatMessage(chat)
	if err != nil {
		return err
	}

	return nil
}

// 检查用户是否在群组中
func (cdb *ChatDB) isUserInGroup(metaId, groupId string) (bool, error) {
	// 使用 TalkGroupMetaIdJoinCollection 来检查用户是否在群组中
	// key: metaId_groupId
	key := []byte(metaId + "_" + groupId)
	value, closer, err := Pb[TalkGroupMetaIdJoinCollection].Get(key)
	if err != nil {
		if err == pebble.ErrNotFound {
			return false, nil
		}
		return false, err
	}
	defer closer.Close()

	var joinList GroupMetaIdJoinList
	err = json.Unmarshal(value, &joinList)
	if err != nil {
		return false, err
	}

	// 如果列表为空，用户不在群组中
	if len(joinList.Items) == 0 {
		return false, nil
	}

	// 获取最新的加入记录（按时间戳倒序，第一个是最新的）
	latestItem := joinList.Items[0]
	return latestItem.GroupState == models.RoomStateIn, nil
}

// 获取用户在群组中的状态
func (cdb *ChatDB) getUserGroupState(metaId, groupId string, chatTimestamp int64) (models.RoomState, error) {
	// 使用 TalkGroupMetaIdJoinCollection 来获取用户状态
	// key: metaId_groupId
	key := []byte(metaId + "_" + groupId)
	value, closer, err := Pb[TalkGroupMetaIdJoinCollection].Get(key)
	if err != nil {
		if err == pebble.ErrNotFound {
			return models.RoomStateOut, nil
		}
		return models.RoomStateOut, err
	}
	defer closer.Close()

	var joinList GroupMetaIdJoinList
	err = json.Unmarshal(value, &joinList)
	if err != nil {
		return models.RoomStateOut, err
	}

	// 如果列表为空，用户不在群组中
	if len(joinList.Items) == 0 {
		return models.RoomStateOut, nil
	}

	// 按时间戳正序排序，确保时间顺序正确
	// 简单的冒泡排序，按时间戳正序
	for i := 0; i < len(joinList.Items)-1; i++ {
		for j := 0; j < len(joinList.Items)-1-i; j++ {
			if joinList.Items[j].JoinTimestamp > joinList.Items[j+1].JoinTimestamp {
				joinList.Items[j], joinList.Items[j+1] = joinList.Items[j+1], joinList.Items[j]
			}
		}
	}

	var startItem, endItem *GroupMetaIdJoinItem
	// 遍历加入记录，找到聊天消息时间戳对应的区间
	for i, item := range joinList.Items {
		if chatTimestamp >= item.JoinTimestamp {
			// 找到聊天消息时间戳对应的开始记录
			startItem = item

			// 查找下一个记录作为结束记录
			if i+1 < len(joinList.Items) {
				endItem = joinList.Items[i+1]
			} else {
				// 如果没有下一个记录，说明这是最新的状态
				endItem = nil
			}
		} else {
			// 如果当前记录的时间戳大于聊天时间戳，说明找到了区间的结束
			// 此时startItem应该是前一个记录
			if startItem != nil {
				endItem = item
			}
			break
		}
	}

	// 如果没有找到对应的区间，说明聊天消息在用户加入群组之前
	if startItem == nil {
		return models.RoomStateOut, nil
	}

	// 根据startItem的状态来判断用户在该时间点的状态
	// 如果endItem存在且聊天时间超过了endItem的时间，说明状态已经改变
	if endItem != nil && chatTimestamp >= endItem.JoinTimestamp {
		// 聊天时间在下一个状态变更之后，使用下一个状态
		return endItem.GroupState, nil
	} else {
		// 聊天时间在当前状态区间内，使用当前状态
		return startItem.GroupState, nil
	}
}

// 保存聊天时间戳索引（根据用户状态决定保存到哪个集合）
func (cdb *ChatDB) SaveChatTimestampWithState(chat *models.TalkGroupChatV3) error {
	// 获取用户在群组中的状态
	groupState, err := cdb.getUserGroupState(chat.MetaId, chat.GroupId, chat.Timestamp)
	if err != nil {
		// 如果获取状态失败，默认保存到正常集合
		return cdb.SaveChatTimestamp(chat)
	}

	// 构造时间戳索引值：pinId_chatType_timestamp
	value := chat.PinId + "_" + strconv.FormatInt(int64(chat.ChatType), 10) + "_" + strconv.FormatInt(chat.Timestamp, 10)

	// 根据用户状态决定保存到哪个集合
	var collection string
	if groupState == models.RoomStateIn {
		// 用户在群组中，保存到正常集合
		collection = TalkGroupChatTimestampCollection
	} else {
		// 用户不在群组中，保存到无效集合
		collection = TalkGroupChatTimestampOutCollection
	}

	// 使用 GroupId_Timestamp 作为主键，支持按时间戳范围查询
	key := []byte(chat.GroupId + "_" + strconv.FormatInt(chat.Timestamp, 10))
	return Pb[collection].Set(key, []byte(value), pebble.Sync)
}

// 处理文件群组聊天
func (cdb *ChatDB) processFileGroupChat(pin *pin.PinInscription) error {
	// 检查是否已经保存过该 PinId
	existingChat, err := cdb.GetChatByPinId(pin.Id)
	if err == nil && existingChat != nil {
		if existingChat.BlockHeight != pin.GenesisHeight {
			existingChat.BlockHeight = pin.GenesisHeight
			err = cdb.SaveChat(existingChat)
			if err != nil {
				return err
			}
		}
		// 已经存在，跳过处理
		return nil
	}

	// 解析协议数据
	var simpleFileGroupChat protocols.SimpleFileGroupChat
	err = json.Unmarshal(pin.ContentBody, &simpleFileGroupChat)
	if err != nil {
		return err
	}
	// 创建聊天消息模型
	chat := &models.TalkGroupChatV3{
		GroupId:     simpleFileGroupChat.GroupId,
		TxId:        pin.Id[:len(pin.Id)-2],
		PinId:       pin.Id,
		MetaId:      pin.CreateMetaId,
		Address:     pin.CreateAddress,
		Protocol:    pin.Path,
		Content:     simpleFileGroupChat.Attachment, // 文件附件
		ContentType: simpleFileGroupChat.FileType,   // 文件类型
		Encryption:  simpleFileGroupChat.Encrypt,
		ChatType:    models.ChatTypeFile,      // 文件类型
		InsideIndex: models.ChatInsideIndexIn, // 默认为进入状态
		ReplyPin:    simpleFileGroupChat.ReplyPin,
		Timestamp:   pin.Timestamp,
		BlockHeight: pin.GenesisHeight,
	}

	// 保存聊天消息到 TalkGroupChatPinCollection
	err = cdb.SaveChat(chat)
	if err != nil {
		return err
	}

	// 保存时间戳索引（根据用户状态决定保存到哪个集合）
	err = cdb.SaveChatTimestampWithState(chat)
	if err != nil {
		return err
	}

	// 将消息加入队列，异步更新群列表
	err = cdb.EnqueueChatMessage(chat)
	if err != nil {
		return err
	}

	return nil
}

// 处理群组红包
func (cdb *ChatDB) processGroupLuckyBag(pin *pin.PinInscription) error {
	// 检查是否已经保存过该 PinId
	existingRed, err := cdb.GetLuckyBagByPinId(pin.Id)
	if err == nil && existingRed != nil {
		// 已经存在，跳过处理
		if existingRed.BlockHeight != pin.GenesisHeight {
			existingRed.BlockHeight = pin.GenesisHeight
			err = cdb.SaveLuckyBag(existingRed)
			if err != nil {
				return err
			}
		}
		return nil
	}

	// 解析协议数据
	var simpleLuckyBag protocols.SimpleGroupLuckyBag
	err = json.Unmarshal(pin.ContentBody, &simpleLuckyBag)
	if err != nil {
		return err
	}

	// 转换支付列表
	var payList []*models.ProInfoPayList
	for _, pay := range simpleLuckyBag.PayList {
		payList = append(payList, &models.ProInfoPayList{
			Amount:  toString(pay.Amount),
			Address: pay.Address,
			Index:   toInt64(pay.Index),
		})
	}

	// 创建红包模型
	redEnvelope := &models.TalkGroupLuckyBagV3{
		CommunityId:         "", // 需要从群组信息中获取
		GroupId:             simpleLuckyBag.GroupId,
		TxId:                pin.Id[:len(pin.Id)-2],
		PinId:               pin.Id,
		MetaId:              pin.CreateMetaId,
		Protocol:            pin.Path,
		SubId:               simpleLuckyBag.SubId,
		Code:                simpleLuckyBag.Code,
		CreateTimeStr:       toString(simpleLuckyBag.CreateTime),
		Content:             simpleLuckyBag.Content,
		Img:                 simpleLuckyBag.Img,
		ImgType:             simpleLuckyBag.ImgType,
		Amount:              toString(simpleLuckyBag.Amount),
		Count:               toString(simpleLuckyBag.Count),
		PayList:             payList,
		LuckyBagVouts:       []*models.LuckyBagOutput{}, // 需要从交易中解析
		Type:                simpleLuckyBag.Type,
		RequireType:         toString(simpleLuckyBag.RequireType),
		RequireTickId:       simpleLuckyBag.RequireTickId,
		RequireCollectionId: simpleLuckyBag.RequireCollectionId,
		LimitAmount:         toUint64(simpleLuckyBag.LimitAmount),
		Timestamp:           pin.Timestamp,
		BlockHeight:         pin.GenesisHeight,
		Chain:               pin.ChainName,
	}

	// 保存红包信息
	err = cdb.SaveLuckyBag(redEnvelope)
	if err != nil {
		return err
	}

	// 创建聊天消息模型（用于群聊显示）
	chat := &models.TalkGroupChatV3{
		GroupId:     simpleLuckyBag.GroupId,
		TxId:        pin.Id[:len(pin.Id)-2],
		PinId:       pin.Id,
		MetaId:      pin.CreateMetaId,
		Address:     pin.CreateAddress,
		Protocol:    pin.Path,
		Content:     "[LuckyBag]:" + simpleLuckyBag.Content, // 红包祝福语
		ContentType: "text/plain",
		Encryption:  "",
		ChatType:    models.ChatTypeLuckyBag,  // 红包类型
		InsideIndex: models.ChatInsideIndexIn, // 默认为进入状态
		ReplyPin:    "",
		Timestamp:   pin.Timestamp,
		Chain:       pin.ChainName,
		BlockHeight: pin.GenesisHeight,
	}

	// 保存聊天消息到 TalkGroupChatPinCollection
	err = cdb.SaveChat(chat)
	if err != nil {
		return err
	}

	// 保存时间戳索引（根据用户状态决定保存到哪个集合）
	err = cdb.SaveChatTimestampWithState(chat)
	if err != nil {
		return err
	}

	// 将消息加入队列，异步更新群列表
	err = cdb.EnqueueChatMessage(chat)
	if err != nil {
		return err
	}

	return nil
}

// 处理群组抢红包
func (cdb *ChatDB) processGroupOpenLuckyBag(pin *pin.PinInscription) error {
	// 检查是否已经保存过该 PinId
	existingOpen, err := cdb.GetOpenLuckyBagByPinId(pin.Id)
	if err == nil && existingOpen != nil {
		// 已经存在，跳过处理
		if existingOpen.BlockHeight != pin.GenesisHeight {
			existingOpen.BlockHeight = pin.GenesisHeight
			err = cdb.SaveOpenLuckyBag(existingOpen)
			if err != nil {
				return err
			}
		}
		return nil
	}

	// 解析协议数据
	var simpleOpenLuckyBag protocols.SimpleGroupOpenLuckyBag
	err = json.Unmarshal(pin.ContentBody, &simpleOpenLuckyBag)
	if err != nil {
		return err
	}

	// 转换输入交易
	var vins []*models.TxIn
	if simpleOpenLuckyBag.Used != nil {
		vins = append(vins, &models.TxIn{
			OutTxID: "", // 需要从交易中解析
			Index:   0,  // 需要从交易中解析
		})
	}

	// 创建抢红包模型
	openLuckyBag := &models.TalkGroupOpenLuckyBagV3{
		CommunityId:         "", // 需要从群组信息中获取
		GroupId:             simpleOpenLuckyBag.GroupId,
		TxId:                pin.Id[:len(pin.Id)-2],
		PinId:               pin.Id,
		MetaId:              pin.CreateMetaId,
		Protocol:            pin.Path,
		SubId:               simpleOpenLuckyBag.SubId,
		Code:                simpleOpenLuckyBag.Code,
		CreateTimeStr:       toString(simpleOpenLuckyBag.CreateTime),
		Address:             pin.CreateAddress,
		Index:               0,  // 需要从交易中解析
		Amount:              "", // 需要从交易中解析
		Vins:                vins,
		Type:                simpleOpenLuckyBag.Type,
		RequireTickId:       "",
		RequireCollectionId: "",
		LuckyBagTxId:        simpleOpenLuckyBag.LuckyBagTxId,
		LuckyBagPinId:       simpleOpenLuckyBag.LuckyBagPinId,
		LuckyBagMetaId:      simpleOpenLuckyBag.LuckyBagMetaId,
		IsWithdraw:          toBool(simpleOpenLuckyBag.IsWithdraw),
		Timestamp:           pin.Timestamp,
		BlockHeight:         pin.GenesisHeight,
		Chain:               pin.ChainName,
	}

	// 保存抢红包信息
	err = cdb.SaveOpenLuckyBag(openLuckyBag)
	if err != nil {
		return err
	}

	// 创建聊天消息模型（用于群聊显示）
	chat := &models.TalkGroupChatV3{
		GroupId:     simpleOpenLuckyBag.GroupId,
		TxId:        pin.Id[:len(pin.Id)-2],
		PinId:       pin.Id,
		MetaId:      pin.CreateMetaId,
		Address:     pin.CreateAddress,
		Protocol:    pin.Path,
		Content:     "[Grab LuckyBag]:" + simpleOpenLuckyBag.Code, // 可以根据实际金额显示
		ContentType: "text/plain",
		Encryption:  "",
		ChatType:    models.ChatTypeOpenLuckyBag, // 抢红包类型
		InsideIndex: models.ChatInsideIndexIn,    // 默认为进入状态
		ReplyPin:    "",
		Timestamp:   pin.Timestamp,
		Chain:       pin.ChainName,
		BlockHeight: pin.GenesisHeight,
	}

	// 保存聊天消息到 TalkGroupChatPinCollection
	err = cdb.SaveChat(chat)
	if err != nil {
		return err
	}

	// 保存时间戳索引（根据用户状态决定保存到哪个集合）
	err = cdb.SaveChatTimestampWithState(chat)
	if err != nil {
		return err
	}

	// 将消息加入队列，异步更新群列表
	err = cdb.EnqueueChatMessage(chat)
	if err != nil {
		return err
	}

	return nil
}

// 处理群组回收红包
func (cdb *ChatDB) processGroupResidueLuckyBag(pin *pin.PinInscription) error {
	// 检查是否已经保存过该 PinId
	existingResidue, err := cdb.GetResidueLuckyBagByLuckyBagPinId(pin.Id)
	if err == nil && existingResidue != nil {
		// 已经存在，跳过处理
		if existingResidue.BlockHeight != pin.GenesisHeight {
			existingResidue.BlockHeight = pin.GenesisHeight
			err = cdb.SaveResidueLuckyBag(existingResidue)
			if err != nil {
				return err
			}
		}
		return nil
	}

	// 解析协议数据
	var simpleResidueLuckyBag protocols.SimpleGroupResidueLuckyBag
	err = json.Unmarshal(pin.ContentBody, &simpleResidueLuckyBag)
	if err != nil {
		return err
	}

	// 转换已使用列表
	var usedList []*models.ProInfoPayList
	for _, used := range simpleResidueLuckyBag.Used {
		usedList = append(usedList, &models.ProInfoPayList{
			Amount:  toString(used.Amount),
			Address: used.Address,
			Index:   toInt64(used.Index),
		})
	}

	// 转换输入交易
	var vins []*models.TxIn
	// 这里需要根据实际情况解析输入交易

	// 创建回收红包模型
	residueLuckyBag := &models.TalkGroupResidueLuckyBagV3{
		CommunityId:         "", // 需要从群组信息中获取
		GroupId:             simpleResidueLuckyBag.GroupId,
		TxId:                pin.Id[:len(pin.Id)-2],
		PinId:               pin.Id,
		MetaId:              pin.CreateMetaId,
		Protocol:            pin.Path,
		SubId:               simpleResidueLuckyBag.SubId,
		Code:                simpleResidueLuckyBag.Code,
		CreateTimeStr:       toString(simpleResidueLuckyBag.CreateTime),
		UsedList:            usedList,
		Vins:                vins,
		Type:                simpleResidueLuckyBag.Type,
		RequireTickId:       "",
		RequireCollectionId: "",
		LuckyBagTxId:        simpleResidueLuckyBag.LuckyBagTxId,
		LuckyBagPinId:       simpleResidueLuckyBag.LuckyBagPinId,
		LuckyBagMetaId:      simpleResidueLuckyBag.LuckyBagMetaId,
		Timestamp:           pin.Timestamp,
		BlockHeight:         pin.GenesisHeight,
		Chain:               pin.ChainName,
	}

	// 保存回收红包信息
	err = cdb.SaveResidueLuckyBag(residueLuckyBag)
	if err != nil {
		return err
	}

	// 创建聊天消息模型（用于群聊显示）
	chat := &models.TalkGroupChatV3{
		GroupId:     simpleResidueLuckyBag.GroupId,
		TxId:        pin.Id[:len(pin.Id)-2],
		PinId:       pin.Id,
		MetaId:      pin.CreateMetaId,
		Address:     pin.CreateAddress,
		Protocol:    pin.Path,
		Content:     "[Recycle LuckyBag]:" + simpleResidueLuckyBag.Code, // 可以根据实际情况显示
		ContentType: "text/plain",
		Encryption:  "",
		ChatType:    models.ChatTypeRecycleLuckyBag, // 回收红包类型
		InsideIndex: models.ChatInsideIndexIn,       // 默认为进入状态
		ReplyPin:    "",
		Timestamp:   pin.Timestamp,
		Chain:       pin.ChainName,
		BlockHeight: pin.GenesisHeight,
	}

	// 保存聊天消息到 TalkGroupChatPinCollection
	err = cdb.SaveChat(chat)
	if err != nil {
		return err
	}

	// 保存时间戳索引（根据用户状态决定保存到哪个集合）
	err = cdb.SaveChatTimestampWithState(chat)
	if err != nil {
		return err
	}

	// 将消息加入队列，异步更新群列表
	err = cdb.EnqueueChatMessage(chat)
	if err != nil {
		return err
	}

	return nil
}

// 辅助函数：将 interface{} 转换为 string
func toString(v interface{}) string {
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		return val
	case int, int32, int64, float32, float64:
		return fmt.Sprintf("%v", val)
	default:
		return fmt.Sprintf("%v", val)
	}
}

// 辅助函数：将 interface{} 转换为 int64
func toInt64(v interface{}) int64 {
	if v == nil {
		return 0
	}
	switch val := v.(type) {
	case int:
		return int64(val)
	case int32:
		return int64(val)
	case int64:
		return val
	case float32:
		return int64(val)
	case float64:
		return int64(val)
	case string:
		if i, err := strconv.ParseInt(val, 10, 64); err == nil {
			return i
		}
		return 0
	default:
		return 0
	}
}

// 辅助函数：将 interface{} 转换为 uint64
func toUint64(v interface{}) uint64 {
	if v == nil {
		return 0
	}
	switch val := v.(type) {
	case int:
		return uint64(val)
	case int32:
		return uint64(val)
	case int64:
		return uint64(val)
	case uint64:
		return val
	case float32:
		return uint64(val)
	case float64:
		return uint64(val)
	case string:
		if i, err := strconv.ParseUint(val, 10, 64); err == nil {
			return i
		}
		return 0
	default:
		return 0
	}
}

// 辅助函数：将 interface{} 转换为 bool
func toBool(v interface{}) bool {
	if v == nil {
		return false
	}
	switch val := v.(type) {
	case bool:
		return val
	case int:
		return val != 0
	case int32:
		return val != 0
	case int64:
		return val != 0
	case string:
		return val == "true" || val == "1"
	default:
		return false
	}
}
