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

// Private chat queue message item
type PrivateQueueChatMessage struct {
	PinId      string                    `json:"pinId"`      // Message PinId
	From       string                    `json:"from"`       // Sender MetaId
	To         string                    `json:"to"`         // Receiver MetaId
	Chat       *models.TalkPrivateChatV3 `json:"chat"`       // Private chat message
	Timestamp  int64                     `json:"timestamp"`  // Enqueue timestamp
	RetryCount int                       `json:"retryCount"` // Retry count
	Status     string                    `json:"status"`     // Processing status: pending, processing, completed, failed
}

// Private chat database operations
type PrivateChatDB struct {
	pb *Pebble
}

func NewPrivateChatDB(pb *Pebble) *PrivateChatDB {
	return &PrivateChatDB{pb: pb}
}

// Save private chat message
func (pcdb *PrivateChatDB) SavePrivateChat(chat *models.TalkPrivateChatV3) error {
	data, err := json.Marshal(chat)
	if err != nil {
		return err
	}

	// Use PinId as primary key
	key := []byte(chat.PinId)
	return Pb[TalkPrivateChatPinCollection].Set(key, data, pebble.Sync)
}

// Save private chat timestamp index (bidirectional index: from_to_timestamp and to_from_timestamp)
func (pcdb *PrivateChatDB) SavePrivateChatTimestamp(chat *models.TalkPrivateChatV3) error {
	// Generate a 6-digit random number for uniqueness
	randomNum := generateRandomNumber(6)

	// Construct timestamp index value: pinId_chatType_timestamp
	value := chat.PinId + "_" + strconv.FormatInt(int64(chat.ChatType), 10) + "_" + strconv.FormatInt(chat.Timestamp, 10) + "_" + randomNum

	// Save from_to_timestamp index
	fromToKey := []byte(chat.From + "_" + chat.To + "_" + strconv.FormatInt(chat.Timestamp, 10) + randomNum)
	err := Pb[TalkPrivateChatTimestampCollection].Set(fromToKey, []byte(value), pebble.Sync)
	if err != nil {
		return err
	}

	// Save to_from_timestamp index (reverse index for easy querying)
	toFromKey := []byte(chat.To + "_" + chat.From + "_" + strconv.FormatInt(chat.Timestamp, 10) + randomNum)
	return Pb[TalkPrivateChatTimestampCollection].Set(toFromKey, []byte(value), pebble.Sync)
}

// Get private chat message by PinId
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

// Get private chat message list by two MetaIds
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
		// Check if it's a chat between these two users
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

// Get private chat message list by two MetaIds and timestamp range (reverse order, pagination based on timestamp)
func (pcdb *PrivateChatDB) GetPrivateChatsByMetaIdsAndTimestampRange(selfMetaId, otherMetaId string, startTimestamp int64, size int64) ([]*models.TalkPrivateChatV3, error) {
	var chats []*models.TalkPrivateChatV3
	iter, err := Pb[TalkPrivateChatTimestampCollection].NewIter(nil)
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	// Construct query start keys: from_to_startTimestamp and to_from_startTimestamp
	fromToStartKey := []byte(selfMetaId + "_" + otherMetaId + "_" + strconv.FormatInt(startTimestamp, 10))
	toFromStartKey := []byte(otherMetaId + "_" + selfMetaId + "_" + strconv.FormatInt(startTimestamp, 10))

	// Start reverse iteration from specified timestamp (latest messages first)
	for iter.SeekLT(fromToStartKey); iter.Valid() && iter.Key() != nil; iter.Prev() {
		key := string(iter.Key())

		// Check if it belongs to chat between these two users
		if !strings.HasPrefix(key, selfMetaId+"_"+otherMetaId+"_") &&
			!strings.HasPrefix(key, otherMetaId+"_"+selfMetaId+"_") {
			continue
		}

		// Parse index value to get PinId
		value := string(iter.Value())
		valueParts := strings.Split(value, "_")
		if len(valueParts) < 1 {
			continue
		}
		pinId := valueParts[0]

		// Get complete private chat message
		chat, err := pcdb.GetPrivateChatByPinId(pinId)
		if err != nil || chat == nil {
			continue
		}

		// Reach pagination size limit
		if int64(len(chats)) >= size {
			break
		}

		chats = append(chats, chat)
	}

	// If not enough messages found from from_to direction, continue searching from to_from direction
	if int64(len(chats)) < size {
		for iter.SeekLT(toFromStartKey); iter.Valid() && iter.Key() != nil; iter.Prev() {
			key := string(iter.Key())

			// Check if it belongs to chat between these two users
			if !strings.HasPrefix(key, otherMetaId+"_"+selfMetaId+"_") {
				continue
			}

			// Parse index value to get PinId
			value := string(iter.Value())
			valueParts := strings.Split(value, "_")
			if len(valueParts) < 1 {
				continue
			}
			pinId := valueParts[0]

			// Get complete private chat message
			chat, err := pcdb.GetPrivateChatByPinId(pinId)
			if err != nil || chat == nil {
				continue
			}

			// Reach pagination size limit
			if int64(len(chats)) >= size {
				break
			}

			chats = append(chats, chat)
		}
	}

	return chats, nil
}

// Get latest private chat messages between two users (reverse order based on timestamp)
func (pcdb *PrivateChatDB) GetLatestPrivateChatsByMetaIds(selfMetaId, otherMetaId string, size int64) ([]*models.TalkPrivateChatV3, error) {
	// Use current time as start timestamp
	currentTimestamp := time.Now().Unix()
	currentTimestamp = currentTimestamp * 1000000
	return pcdb.GetPrivateChatsByMetaIdsAndTimestampRange(selfMetaId, otherMetaId, currentTimestamp, size)
}

// Delete private chat message
func (pcdb *PrivateChatDB) DeletePrivateChat(pinId string) error {
	key := []byte(pinId)
	return Pb[TalkPrivateChatPinCollection].Delete(key, pebble.Sync)
}

// Enqueue private chat message (asynchronous processing)
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

	// Use timestamp_pinId as primary key to support processing in time order
	key := []byte(strconv.FormatInt(queueMessage.Timestamp, 10) + "_" + chat.PinId)
	return Pb[TalkPrivateChatQueueCollection].Set(key, data, pebble.Sync)
}

// Get pending private chat messages from queue
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

		// Only process messages with pending status
		if queueMessage.Status == "pending" {
			messages = append(messages, &queueMessage)
			count++
		}
	}

	return messages, nil
}

// Delete private chat queue message data
func (pcdb *PrivateChatDB) deletePrivateQueueMessage(pinId string) error {
	iter, err := Pb[TalkPrivateChatQueueCollection].NewIter(nil)
	if err != nil {
		return err
	}
	defer iter.Close()

	// Find queue message containing this pinId
	for iter.First(); iter.Valid(); iter.Next() {
		value := string(iter.Value())

		var queueMessage PrivateQueueChatMessage
		err := json.Unmarshal([]byte(value), &queueMessage)
		if err != nil {
			continue
		}

		// Found matching pinId, delete this queue message
		if queueMessage.PinId == pinId {
			return Pb[TalkPrivateChatQueueCollection].Delete(iter.Key(), pebble.Sync)
		}
	}

	return nil
}

// Get user's context list (group chat + private chat)
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

// Save user's context list
func (pcdb *PrivateChatDB) SaveMetaIdContextList(contextList *models.MetaIdContextList) error {
	data, err := json.Marshal(contextList)
	if err != nil {
		return err
	}

	key := []byte(contextList.MetaId)
	return Pb[TalkMetaIdContextListCollection].Set(key, data, pebble.Sync)
}

// Update private chat contact list
func (pcdb *PrivateChatDB) UpdatePrivateContactList(chat *models.TalkPrivateChatV3) error {
	// Update sender's contact list
	err := pcdb.updateSingleUserPrivateContactList(chat.From, chat.To, chat)
	if err != nil {
		return err
	}

	// Update receiver's contact list
	err = pcdb.updateSingleUserPrivateContactList(chat.To, chat.From, chat)
	if err != nil {
		return err
	}

	return nil
}

// Update single user's private chat contact list
func (pcdb *PrivateChatDB) updateSingleUserPrivateContactList(selfMetaId, otherMetaId string, chat *models.TalkPrivateChatV3) error {
	// Get user's context list
	contextList, err := pcdb.GetMetaIdContextList(selfMetaId)
	if err != nil {
		return err
	}

	// Create new private chat contact item
	newItem := &models.MetaIdContextItem{
		GroupId:          "",             //
		MetaId:           otherMetaId,    // Other party's MetaId
		Address:          chat.ToAddress, // Other party's address
		Type:             "2",            // 2 indicates private chat
		Timestamp:        chat.Timestamp,
		ChatType:         chat.ChatType,
		Content:          chat.Content,
		CreateMetaId:     chat.From,        // MetaId of message creator
		CreateAddress:    chat.FromAddress, // Private chat message doesn't have address field
		LastMessagePinId: chat.PinId,
		BlockHeight:      chat.BlockHeight,
	}

	// Check if private chat contact item already exists
	found := false
	shouldUpdate := false
	for i, item := range contextList.Items {
		// For private chat, identify by GroupId and Type
		if item.GroupId == otherMetaId && item.Type == "2" {
			// Check if update is needed
			if item.LastMessagePinId != newItem.LastMessagePinId ||
				item.BlockHeight != newItem.BlockHeight {
				// Update existing item
				contextList.Items[i] = newItem
				shouldUpdate = true
			}
			found = true
			break
		}
	}

	// If not found, add new item
	if !found {
		contextList.Items = append(contextList.Items, newItem)
		shouldUpdate = true
	}

	// If no update needed, return directly
	if !shouldUpdate {
		return nil
	}

	// Sort context list by timestamp in reverse order
	pcdb.sortContextListByTimestamp(contextList)

	// Save updated context list
	return pcdb.SaveMetaIdContextList(contextList)
}

// Sort context list by timestamp in reverse order
func (pcdb *PrivateChatDB) sortContextListByTimestamp(contextList *models.MetaIdContextList) {
	// Simple bubble sort, reverse order by timestamp
	for i := 0; i < len(contextList.Items)-1; i++ {
		for j := 0; j < len(contextList.Items)-1-i; j++ {
			if contextList.Items[j].Timestamp < contextList.Items[j+1].Timestamp {
				contextList.Items[j], contextList.Items[j+1] = contextList.Items[j+1], contextList.Items[j]
			}
		}
	}
}

// Batch process private chat queue messages
func (pcdb *PrivateChatDB) ProcessPrivateQueueMessages(batchSize int) error {
	// Get pending messages
	messages, err := pcdb.GetPendingPrivateQueueMessages(batchSize)
	if err != nil {
		return err
	}

	// Batch process messages
	for _, message := range messages {
		// Update private chat contact list
		err = pcdb.UpdatePrivateContactList(message.Chat)
		if err != nil {
			// Processing failed, log error but don't delete queue message, can retry later
			log.Printf("Failed to update private contact list for pinId %s: %v", message.PinId, err)
			continue
		}

		log.Printf("Processing private chat message for pinId %s", message.PinId)

		// Processing successful, delete queue message
		err = pcdb.deletePrivateQueueMessage(message.PinId)
		if err != nil {
			// Log error but don't affect main flow
			log.Printf("Failed to delete private queue message for pinId %s: %v", message.PinId, err)
		}
	}

	return nil
}

// Start private chat queue processing goroutine (needs to be called when application starts)
func (pcdb *PrivateChatDB) StartPrivateQueueProcessor() {
	go func() {
		ticker := time.NewTicker(5 * time.Second) // Process every 5 seconds
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				// Batch process queue messages
				err := pcdb.ProcessPrivateQueueMessages(100) // Process 100 messages each time
				if err != nil {
					// Log error
					continue
				}
			}
		}
	}()
}

// Main method to process private chat Pin
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
		return nil // Unknown operation type, skip
	}
	return nil
}

// Process private chat message
func (pcdb *PrivateChatDB) processPrivateChat(pin *pin.PinInscription) error {
	// Check if this PinId has already been saved
	existingChat, err := pcdb.GetPrivateChatByPinId(pin.Id)
	if err == nil && existingChat != nil {
		// Already exists, skip processing
		return nil
	}

	// Parse protocol data
	var simpleMsg protocols.SimpleMsg
	err = json.Unmarshal(pin.ContentBody, &simpleMsg)
	if err != nil {
		return err
	}

	// Create private chat message model
	chat := &models.TalkPrivateChatV3{
		From:        pin.CreateMetaId,  // Sender MetaId
		FromAddress: pin.CreateAddress, // Sender address
		To:          simpleMsg.To,      // Receiver MetaId
		ToAddress:   "",                // Receiver address
		TxId:        pin.Id[:len(pin.Id)-2],
		PinId:       pin.Id,
		Protocol:    pin.Path,
		Content:     simpleMsg.Content,
		ContentType: simpleMsg.ContentType,
		Encryption:  simpleMsg.Encrypt,
		ChatType:    models.ChatTypeMsg, // Default to message type
		ReplyPin:    simpleMsg.ReplyPin,
		Timestamp:   pin.Timestamp,
		Chain:       pin.ChainName,
		BlockHeight: pin.GenesisHeight,
	}

	// Save private chat message to TalkPrivateChatPinCollection
	err = pcdb.SavePrivateChat(chat)
	if err != nil {
		return err
	}

	// Save timestamp index
	err = pcdb.SavePrivateChatTimestamp(chat)
	if err != nil {
		return err
	}

	// Enqueue message for asynchronous processing
	err = pcdb.EnqueuePrivateChatMessage(chat)
	if err != nil {
		return err
	}

	return nil
}

// Process file private chat message
func (pcdb *PrivateChatDB) processFilePrivateChat(pin *pin.PinInscription) error {
	// Check if this PinId has already been saved
	existingChat, err := pcdb.GetPrivateChatByPinId(pin.Id)
	if err == nil && existingChat != nil {
		if existingChat.BlockHeight != pin.GenesisHeight {
			existingChat.BlockHeight = pin.GenesisHeight
			err = pcdb.SavePrivateChat(existingChat)
			if err != nil {
				return err
			}
		}
		// Already exists, skip processing
		return nil
	}

	// Parse protocol data
	var simpleFileMsg protocols.SimpleFileMsg
	err = json.Unmarshal(pin.ContentBody, &simpleFileMsg)
	if err != nil {
		return err
	}

	// Create private chat message model
	chat := &models.TalkPrivateChatV3{
		From:        pin.CreateMetaId, // Sender MetaId
		To:          simpleFileMsg.To, // Receiver MetaId
		TxId:        pin.Id[:len(pin.Id)-2],
		PinId:       pin.Id,
		Protocol:    pin.Path,
		Content:     simpleFileMsg.Attachment, // File attachment
		ContentType: simpleFileMsg.FileType,   // File type
		Encryption:  simpleFileMsg.Encrypt,
		ChatType:    models.ChatTypeFile, // File type
		ReplyPin:    simpleFileMsg.ReplyPin,
		Timestamp:   pin.Timestamp,
		BlockHeight: pin.GenesisHeight,
	}

	// Save private chat message to TalkPrivateChatPinCollection
	err = pcdb.SavePrivateChat(chat)
	if err != nil {
		return err
	}

	// Save timestamp index
	err = pcdb.SavePrivateChatTimestamp(chat)
	if err != nil {
		return err
	}

	// Enqueue message for asynchronous processing
	err = pcdb.EnqueuePrivateChatMessage(chat)
	if err != nil {
		return err
	}

	return nil
}
