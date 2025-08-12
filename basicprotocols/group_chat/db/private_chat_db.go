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

// 私聊队列消息项
type PrivateQueueChatMessage struct {
	PinId      string                    `json:"pinId"`      // 消息PinId
	From       string                    `json:"from"`       // 发送者MetaId
	To         string                    `json:"to"`         // 接收者MetaId
	Chat       *models.TalkPrivateChatV3 `json:"chat"`       // 私聊消息
	Timestamp  int64                     `json:"timestamp"`  // 入队时间戳
	RetryCount int                       `json:"retryCount"` // 重试次数
	Status     string                    `json:"status"`     // 处理状态：pending, processing, completed, failed
}

// 私聊数据库操作
type PrivateChatDB struct {
	pb *Pebble
}

func NewPrivateChatDB(pb *Pebble) *PrivateChatDB {
	return &PrivateChatDB{pb: pb}
}

// 保存私聊消息
func (pcdb *PrivateChatDB) SavePrivateChat(chat *models.TalkPrivateChatV3) error {
	data, err := json.Marshal(chat)
	if err != nil {
		return err
	}

	// 使用 PinId 作为主键
	key := []byte(chat.PinId)
	return Pb[TalkPrivateChatPinCollection].Set(key, data, pebble.Sync)
}

// 保存私聊时间戳索引（双向索引：from_to_timestamp 和 to_from_timestamp）
func (pcdb *PrivateChatDB) SavePrivateChatTimestamp(chat *models.TalkPrivateChatV3) error {
	// 构造时间戳索引值：pinId_chatType_timestamp
	value := chat.PinId + "_" + strconv.FormatInt(int64(chat.ChatType), 10) + "_" + strconv.FormatInt(chat.Timestamp, 10)

	// 保存 from_to_timestamp 索引
	fromToKey := []byte(chat.From + "_" + chat.To + "_" + strconv.FormatInt(chat.Timestamp, 10))
	err := Pb[TalkPrivateChatTimestampCollection].Set(fromToKey, []byte(value), pebble.Sync)
	if err != nil {
		return err
	}

	// 保存 to_from_timestamp 索引（反向索引，便于查询）
	toFromKey := []byte(chat.To + "_" + chat.From + "_" + strconv.FormatInt(chat.Timestamp, 10))
	return Pb[TalkPrivateChatTimestampCollection].Set(toFromKey, []byte(value), pebble.Sync)
}

// 根据PinId获取私聊消息
func (pcdb *PrivateChatDB) GetPrivateChatByPinId(pinId string) (*models.TalkPrivateChatV3, error) {
	key := []byte(pinId)
	value, closer, err := Pb[TalkPrivateChatPinCollection].Get(key)
	if err != nil {
		if err == pebble.ErrNotFound {
			return nil, nil
		}
		return nil, err
	}
	defer closer.Close()

	var chat models.TalkPrivateChatV3
	err = json.Unmarshal(value, &chat)
	if err != nil {
		return nil, err
	}

	return &chat, nil
}

// 根据两个MetaId获取私聊消息列表
func (pcdb *PrivateChatDB) GetPrivateChatsByMetaIds(selfMetaId, otherMetaId string, page, size int64) ([]*models.TalkPrivateChatV3, error) {
	var chats []*models.TalkPrivateChatV3
	iter, err := Pb[TalkPrivateChatPinCollection].NewIter(nil)
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	count := int64(0)
	skip := (page - 1) * size

	for iter.First(); iter.Valid(); iter.Next() {
		var chat models.TalkPrivateChatV3
		err := json.Unmarshal(iter.Value(), &chat)
		if err != nil {
			continue
		}
		// 检查是否是这两个用户之间的聊天
		if (chat.From == selfMetaId && chat.To == otherMetaId) ||
			(chat.From == otherMetaId && chat.To == selfMetaId) {
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

// 删除私聊消息
func (pcdb *PrivateChatDB) DeletePrivateChat(pinId string) error {
	key := []byte(pinId)
	return Pb[TalkPrivateChatPinCollection].Delete(key, pebble.Sync)
}

// 将私聊消息加入队列（异步处理）
func (pcdb *PrivateChatDB) EnqueuePrivateChatMessage(chat *models.TalkPrivateChatV3) error {
	queueMessage := &PrivateQueueChatMessage{
		PinId:      chat.PinId,
		From:       chat.From,
		To:         chat.To,
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
	return Pb[TalkPrivateChatQueueCollection].Set(key, data, pebble.Sync)
}

// 获取队列中的待处理私聊消息
func (pcdb *PrivateChatDB) GetPendingPrivateQueueMessages(limit int) ([]*PrivateQueueChatMessage, error) {
	var messages []*PrivateQueueChatMessage
	iter, err := Pb[TalkPrivateChatQueueCollection].NewIter(nil)
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	count := 0
	for iter.First(); iter.Valid() && iter.Key() != nil && count < limit; iter.Next() {
		value := string(iter.Value())

		var queueMessage PrivateQueueChatMessage
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

// 删除私聊队列消息数据
func (pcdb *PrivateChatDB) deletePrivateQueueMessage(pinId string) error {
	iter, err := Pb[TalkPrivateChatQueueCollection].NewIter(nil)
	if err != nil {
		return err
	}
	defer iter.Close()

	// 查找包含该pinId的队列消息
	for iter.First(); iter.Valid(); iter.Next() {
		value := string(iter.Value())

		var queueMessage PrivateQueueChatMessage
		err := json.Unmarshal([]byte(value), &queueMessage)
		if err != nil {
			continue
		}

		// 找到匹配的pinId，删除该队列消息
		if queueMessage.PinId == pinId {
			return Pb[TalkPrivateChatQueueCollection].Delete(iter.Key(), pebble.Sync)
		}
	}

	return nil
}

// 获取用户的上下文列表（群聊+私聊）
func (pcdb *PrivateChatDB) GetMetaIdContextList(metaId string) (*models.MetaIdContextList, error) {
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

// 保存用户的上下文列表
func (pcdb *PrivateChatDB) SaveMetaIdContextList(contextList *models.MetaIdContextList) error {
	data, err := json.Marshal(contextList)
	if err != nil {
		return err
	}

	key := []byte(contextList.MetaId)
	return Pb[TalkMetaIdContextListCollection].Set(key, data, pebble.Sync)
}

// 更新私聊联系人列表
func (pcdb *PrivateChatDB) UpdatePrivateContactList(chat *models.TalkPrivateChatV3) error {
	// 更新发送者的联系人列表
	err := pcdb.updateSingleUserPrivateContactList(chat.From, chat.To, chat)
	if err != nil {
		return err
	}

	// 更新接收者的联系人列表
	err = pcdb.updateSingleUserPrivateContactList(chat.To, chat.From, chat)
	if err != nil {
		return err
	}

	return nil
}

// 更新单个用户的私聊联系人列表
func (pcdb *PrivateChatDB) updateSingleUserPrivateContactList(selfMetaId, otherMetaId string, chat *models.TalkPrivateChatV3) error {
	// 获取用户的上下文列表
	contextList, err := pcdb.GetMetaIdContextList(selfMetaId)
	if err != nil {
		return err
	}

	// 创建新的私聊联系人项
	newItem := &models.MetaIdContextItem{
		GroupId:          "",          //
		MetaId:           otherMetaId, // 对方的MetaId
		Type:             "2",         // 2表示私聊
		Timestamp:        chat.Timestamp,
		ChatType:         chat.ChatType,
		Content:          chat.Content,
		CreateMetaId:     chat.From,        // 消息创建者的MetaId
		CreateAddress:    chat.FromAddress, // 私聊消息中没有地址字段
		LastMessagePinId: chat.PinId,
		BlockHeight:      chat.BlockHeight,
	}

	// 查找是否已存在该私聊联系人的项
	found := false
	shouldUpdate := false
	for i, item := range contextList.Items {
		// 对于私聊，通过GroupId和Type来识别
		if item.GroupId == otherMetaId && item.Type == "2" {
			// 检查是否需要更新
			if item.LastMessagePinId != newItem.LastMessagePinId ||
				item.BlockHeight != newItem.BlockHeight {
				// 更新现有项
				contextList.Items[i] = newItem
				shouldUpdate = true
			}
			found = true
			break
		}
	}

	// 如果不存在，添加新项
	if !found {
		contextList.Items = append(contextList.Items, newItem)
		shouldUpdate = true
	}

	// 如果不需要更新，直接返回
	if !shouldUpdate {
		return nil
	}

	// 按时间戳倒序排序
	pcdb.sortContextListByTimestamp(contextList)

	// 保存更新后的上下文列表
	return pcdb.SaveMetaIdContextList(contextList)
}

// 按时间戳倒序排序上下文列表
func (pcdb *PrivateChatDB) sortContextListByTimestamp(contextList *models.MetaIdContextList) {
	// 简单的冒泡排序，按时间戳倒序
	for i := 0; i < len(contextList.Items)-1; i++ {
		for j := 0; j < len(contextList.Items)-1-i; j++ {
			if contextList.Items[j].Timestamp < contextList.Items[j+1].Timestamp {
				contextList.Items[j], contextList.Items[j+1] = contextList.Items[j+1], contextList.Items[j]
			}
		}
	}
}

// 批量处理私聊队列消息
func (pcdb *PrivateChatDB) ProcessPrivateQueueMessages(batchSize int) error {
	// 获取待处理的消息
	messages, err := pcdb.GetPendingPrivateQueueMessages(batchSize)
	if err != nil {
		return err
	}

	// 批量处理消息
	for _, message := range messages {
		// 更新私聊联系人列表
		err = pcdb.UpdatePrivateContactList(message.Chat)
		if err != nil {
			// 处理失败，记录错误但不删除队列消息，可以稍后重试
			log.Printf("Failed to update private contact list for pinId %s: %v", message.PinId, err)
			continue
		}

		log.Printf("Processing private chat message for pinId %s", message.PinId)

		// 处理成功，删除队列消息
		err = pcdb.deletePrivateQueueMessage(message.PinId)
		if err != nil {
			// 记录错误但不影响主流程
			log.Printf("Failed to delete private queue message for pinId %s: %v", message.PinId, err)
		}
	}

	return nil
}

// 启动私聊队列处理协程（需要在应用启动时调用）
func (pcdb *PrivateChatDB) StartPrivateQueueProcessor() {
	go func() {
		ticker := time.NewTicker(5 * time.Second) // 每5秒处理一次
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				// 批量处理队列消息
				err := pcdb.ProcessPrivateQueueMessages(100) // 每次处理100条消息
				if err != nil {
					// 记录错误日志
					continue
				}
			}
		}
	}()
}

// 总的处理私聊 Pin 方法
func (pcdb *PrivateChatDB) ProcessPrivateChatPin(pin *pin.PinInscription) error {
	switch pin.Operation {
	case "create":
		path := pin.Path
		protocol := strings.Replace(path, "/protocols/", "", -1)
		if strings.ToLower(protocol) == strings.ToLower(protocols.MonitorSimpleMsg) {
			return pcdb.processPrivateChat(pin)
		} else if strings.ToLower(protocol) == strings.ToLower(protocols.MonitorSimpleFileMsg) {
			return pcdb.processFilePrivateChat(pin)
		}
	default:
		return nil // 未知操作类型，跳过
	}
	return nil
}

// 处理私聊消息
func (pcdb *PrivateChatDB) processPrivateChat(pin *pin.PinInscription) error {
	// 检查是否已经保存过该 PinId
	existingChat, err := pcdb.GetPrivateChatByPinId(pin.Id)
	if err == nil && existingChat != nil {
		// 已经存在，跳过处理
		return nil
	}

	// 解析协议数据
	var simpleMsg protocols.SimpleMsg
	err = json.Unmarshal(pin.ContentBody, &simpleMsg)
	if err != nil {
		return err
	}

	// 创建私聊消息模型
	chat := &models.TalkPrivateChatV3{
		From:        pin.CreateMetaId,  // 发送者MetaId
		FromAddress: pin.CreateAddress, // 发送者地址
		To:          simpleMsg.To,      // 接收者MetaId
		ToAddress:   "",                // 接收者地址
		TxId:        pin.Id[:len(pin.Id)-2],
		PinId:       pin.Id,
		Protocol:    pin.Path,
		Content:     simpleMsg.Content,
		ContentType: simpleMsg.ContentType,
		Encryption:  simpleMsg.Encrypt,
		ChatType:    models.ChatTypeMsg, // 默认为消息类型
		ReplyPin:    simpleMsg.ReplyPin,
		Timestamp:   pin.Timestamp,
		Chain:       pin.ChainName,
		BlockHeight: pin.GenesisHeight,
	}

	// 保存私聊消息到 TalkPrivateChatPinCollection
	err = pcdb.SavePrivateChat(chat)
	if err != nil {
		return err
	}

	// 保存时间戳索引
	err = pcdb.SavePrivateChatTimestamp(chat)
	if err != nil {
		return err
	}

	// 将消息加入队列，异步处理
	err = pcdb.EnqueuePrivateChatMessage(chat)
	if err != nil {
		return err
	}

	return nil
}

// 处理文件私聊消息
func (pcdb *PrivateChatDB) processFilePrivateChat(pin *pin.PinInscription) error {
	// 检查是否已经保存过该 PinId
	existingChat, err := pcdb.GetPrivateChatByPinId(pin.Id)
	if err == nil && existingChat != nil {
		if existingChat.BlockHeight != pin.GenesisHeight {
			existingChat.BlockHeight = pin.GenesisHeight
			err = pcdb.SavePrivateChat(existingChat)
			if err != nil {
				return err
			}
		}
		// 已经存在，跳过处理
		return nil
	}

	// 解析协议数据
	var simpleFileMsg protocols.SimpleFileMsg
	err = json.Unmarshal(pin.ContentBody, &simpleFileMsg)
	if err != nil {
		return err
	}

	// 创建私聊消息模型
	chat := &models.TalkPrivateChatV3{
		From:        pin.CreateMetaId, // 发送者MetaId
		To:          simpleFileMsg.To, // 接收者MetaId
		TxId:        pin.Id[:len(pin.Id)-2],
		PinId:       pin.Id,
		Protocol:    pin.Path,
		Content:     simpleFileMsg.Attachment, // 文件附件
		ContentType: simpleFileMsg.FileType,   // 文件类型
		Encryption:  simpleFileMsg.Encrypt,
		ChatType:    models.ChatTypeFile, // 文件类型
		ReplyPin:    simpleFileMsg.ReplyPin,
		Timestamp:   pin.Timestamp,
		BlockHeight: pin.GenesisHeight,
	}

	// 保存私聊消息到 TalkPrivateChatPinCollection
	err = pcdb.SavePrivateChat(chat)
	if err != nil {
		return err
	}

	// 保存时间戳索引
	err = pcdb.SavePrivateChatTimestamp(chat)
	if err != nil {
		return err
	}

	// 将消息加入队列，异步处理
	err = pcdb.EnqueuePrivateChatMessage(chat)
	if err != nil {
		return err
	}

	return nil
}
