package db

import (
	"encoding/json"
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
	value := chat.PinId + "_" + string(rune(chat.ChatType)) + "_" + string(rune(chat.Timestamp))

	// 使用 GroupId_Timestamp 作为主键，支持按时间戳范围查询
	key := []byte(chat.GroupId + "_" + string(rune(chat.Timestamp)))
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
	startKey := []byte(groupId + "_" + string(rune(startTimestamp)))

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
	var chats []*models.TalkGroupChatV3
	iter, err := Pb[TalkGroupChatTimestampCollection].NewIter(nil)
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	// 构造群组前缀
	groupPrefix := groupId + "_"

	// 从最新时间戳开始倒序遍历
	for iter.Last(); iter.Valid() && iter.Key() != nil; iter.Prev() {
		key := string(iter.Key())

		// 检查是否属于指定群组
		if !strings.HasPrefix(key, groupPrefix) {
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

// 删除聊天消息
func (cdb *ChatDB) DeleteChat(pinId string) error {
	key := []byte(pinId)
	return Pb[TalkGroupChatPinCollection].Delete(key, pebble.Sync)
}

// 保存红包信息
func (cdb *ChatDB) SaveRedEnvelope(red *models.TalkGroupRedEnvelopeV3) error {
	data, err := json.Marshal(red)
	if err != nil {
		return err
	}

	// 使用 PinId 作为主键
	key := []byte(red.PinId)
	return Pb[TalkGroupRedEnvelopePinCollection].Set(key, data, pebble.Sync)
}

// 根据PinId获取红包信息
func (cdb *ChatDB) GetRedEnvelopeByPinId(pinId string) (*models.TalkGroupRedEnvelopeV3, error) {
	key := []byte(pinId)
	value, closer, err := Pb[TalkGroupRedEnvelopePinCollection].Get(key)
	if err != nil {
		if err == pebble.ErrNotFound {
			return nil, nil
		}
		return nil, err
	}
	defer closer.Close()

	var red models.TalkGroupRedEnvelopeV3
	err = json.Unmarshal(value, &red)
	if err != nil {
		return nil, err
	}

	return &red, nil
}

// 根据群组ID获取红包列表
func (cdb *ChatDB) GetRedEnvelopesByGroupId(groupId string) ([]*models.TalkGroupRedEnvelopeV3, error) {
	var reds []*models.TalkGroupRedEnvelopeV3
	iter, err := Pb[TalkGroupRedEnvelopePinCollection].NewIter(nil)
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	for iter.First(); iter.Valid(); iter.Next() {
		var red models.TalkGroupRedEnvelopeV3
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
func (cdb *ChatDB) SaveOpenRedEnvelope(open *models.TalkGroupOpenRedEnvelopeV3) error {
	data, err := json.Marshal(open)
	if err != nil {
		return err
	}

	// 使用 PinId 作为主键
	key := []byte(open.PinId)
	return Pb[TalkGroupOpenRedEnvelopePinCollection].Set(key, data, pebble.Sync)
}

// 根据PinId获取抢红包信息
func (cdb *ChatDB) GetOpenRedEnvelopeByPinId(pinId string) (*models.TalkGroupOpenRedEnvelopeV3, error) {
	key := []byte(pinId)
	value, closer, err := Pb[TalkGroupOpenRedEnvelopePinCollection].Get(key)
	if err != nil {
		if err == pebble.ErrNotFound {
			return nil, nil
		}
		return nil, err
	}
	defer closer.Close()

	var open models.TalkGroupOpenRedEnvelopeV3
	err = json.Unmarshal(value, &open)
	if err != nil {
		return nil, err
	}

	return &open, nil
}

// 根据红包TxId获取抢红包列表
func (cdb *ChatDB) GetOpenRedEnvelopesByRedEnvelopeTxId(redEnvelopeTxId string) ([]*models.TalkGroupOpenRedEnvelopeV3, error) {
	var opens []*models.TalkGroupOpenRedEnvelopeV3
	iter, err := Pb[TalkGroupOpenRedEnvelopePinCollection].NewIter(nil)
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	for iter.First(); iter.Valid(); iter.Next() {
		var open models.TalkGroupOpenRedEnvelopeV3
		err := json.Unmarshal(iter.Value(), &open)
		if err != nil {
			continue
		}
		if open.RedEnvelopeTxId == redEnvelopeTxId {
			opens = append(opens, &open)
		}
	}

	return opens, nil
}

// 保存剩余红包信息
func (cdb *ChatDB) SaveResidueRedEnvelope(residue *models.TalkGroupResidueRedEnvelopeV3) error {
	data, err := json.Marshal(residue)
	if err != nil {
		return err
	}

	// 使用 PinId 作为主键
	key := []byte(residue.PinId)
	return Pb[TalkGroupResidueRedEnvelopePinCollection].Set(key, data, pebble.Sync)
}

// 根据红包PinId获取剩余红包信息
func (cdb *ChatDB) GetResidueRedEnvelopeByRedEnvelopePinId(redEnvelopePinId string) (*models.TalkGroupResidueRedEnvelopeV3, error) {
	key := []byte(redEnvelopePinId)
	value, closer, err := Pb[TalkGroupResidueRedEnvelopePinCollection].Get(key)
	if err != nil {
		if err == pebble.ErrNotFound {
			return nil, nil
		}
		return nil, err
	}
	defer closer.Close()

	var residue models.TalkGroupResidueRedEnvelopeV3
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
	// 创建最新聊天记录
	latestChat := &models.TalkGroupLatestChat{
		GroupId:          groupId,
		Timestamp:        chat.Timestamp,
		ChatType:         chat.ChatType,
		Content:          chat.Content,
		CreateAddress:    chat.Address,
		LastMessagePinId: chat.PinId,
		MetaId:           chat.MetaId,
		TxId:             chat.TxId,
		Protocol:         chat.Protocol,
		ContentType:      chat.ContentType,
		Encryption:       chat.Encryption,
		ReplyTx:          chat.ReplyTx,
		Chain:            chat.Chain,
	}

	// 序列化数据
	data, err := json.Marshal(latestChat)
	if err != nil {
		return err
	}

	// 使用 GroupId 作为主键保存到 TalkGroupLatestChatCollection
	key := []byte(groupId)
	return Pb[TalkGroupLatestChatCollection].Set(key, data, pebble.Sync)
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
		Timestamp:        chat.Timestamp,
		ChatType:         chat.ChatType,
		Content:          chat.Content,
		CreateAddress:    chat.Address,
		LastMessagePinId: chat.PinId,
	}

	// 查找是否已存在该群组的项
	found := false
	for i, item := range contextList.Items {
		if item.GroupId == groupId {
			// 更新现有项
			contextList.Items[i] = newItem
			found = true
			break
		}
	}

	// 如果不存在，添加新项
	if !found {
		contextList.Items = append(contextList.Items, newItem)
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
		ReplyTx:     simpleGroupChat.ReplyTx,
		Timestamp:   pin.Timestamp,
		Chain:       pin.ChainName,
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
	value := chat.PinId + "_" + string(rune(chat.ChatType)) + "_" + string(rune(chat.Timestamp))

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
	key := []byte(chat.GroupId + "_" + string(rune(chat.Timestamp)))
	return Pb[collection].Set(key, []byte(value), pebble.Sync)
}

// 处理文件群组聊天
func (cdb *ChatDB) processFileGroupChat(pin *pin.PinInscription) error {
	// 检查是否已经保存过该 PinId
	existingChat, err := cdb.GetChatByPinId(pin.Id)
	if err == nil && existingChat != nil {
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
		ReplyTx:     simpleFileGroupChat.ReplyTx,
		Timestamp:   pin.Timestamp,
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
