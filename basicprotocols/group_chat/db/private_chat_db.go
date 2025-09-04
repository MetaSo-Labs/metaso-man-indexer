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
	"sync"
	"time"

	"crypto/sha256"
	"encoding/hex"

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
	pb              *Pebble
	isProcessing    bool
	processingMutex sync.Mutex
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
	err = Pb[TalkPrivateChatTimestampCollection].Set(toFromKey, []byte(value), pebble.Sync)
	if err != nil {
		return err
	}

	go dealPrivateChatItem(chat)
	return nil
}

// Save private chat out timestamp index (for blocked messages)
func (pcdb *PrivateChatDB) SavePrivateChatOutTimestamp(chat *models.TalkPrivateChatV3) error {
	// Generate a 6-digit random number for uniqueness
	randomNum := generateRandomNumber(6)

	// Construct timestamp index value: pinId_chatType_timestamp
	value := chat.PinId + "_" + strconv.FormatInt(int64(chat.ChatType), 10) + "_" + strconv.FormatInt(chat.Timestamp, 10) + "_" + randomNum

	// Save from_to_timestamp index to out collection (blocked messages)
	fromToKey := []byte(chat.From + "_" + chat.To + "_" + strconv.FormatInt(chat.Timestamp, 10) + randomNum)
	err := Pb[TalkPrivateChatTimestampOutCollection].Set(fromToKey, []byte(value), pebble.Sync)
	if err != nil {
		return err
	}

	// Save to_from_timestamp index to out collection (blocked messages)
	toFromKey := []byte(chat.To + "_" + chat.From + "_" + strconv.FormatInt(chat.Timestamp, 10) + randomNum)
	err = Pb[TalkPrivateChatTimestampOutCollection].Set(toFromKey, []byte(value), pebble.Sync)
	if err != nil {
		return err
	}

	return nil
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
// This function handles the key format: from_to_timestamp+number(6) and to_from_timestamp+number(6)
// Example: from_to_1755500889000001 (timestamp 1755500889 + random 000001)
func (pcdb *PrivateChatDB) GetPrivateChatsByMetaIdsAndTimestampRange(selfMetaId, otherMetaId string, startTimestamp int64, size int64) ([]*models.TalkPrivateChatV3, int64, error) {
	var chats []*models.TalkPrivateChatV3

	// Check if startTimestamp is 16 digits, if not, pad with zeros
	startTimestampStr := strconv.FormatInt(startTimestamp, 10)
	if len(startTimestampStr) < 16 {
		// Pad with zeros to make it 16 digits
		startTimestampStr = startTimestampStr + strings.Repeat("0", 16-len(startTimestampStr))
	}

	// Create iter options to limit the range to only keys for these two users
	iterOptions := &pebble.IterOptions{
		LowerBound: []byte(selfMetaId + "_" + otherMetaId + "_"),
		UpperBound: []byte(selfMetaId + "_" + otherMetaId + "_" + string([]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff})),
	}

	iter, err := Pb[TalkPrivateChatTimestampCollection].NewIter(iterOptions)
	if err != nil {
		return nil, 0, err
	}
	defer iter.Close()

	nextTimestamp := int64(0)

	// Start reverse iteration from specified timestamp (latest messages first)
	startKey := []byte(selfMetaId + "_" + otherMetaId + "_" + startTimestampStr)
	for iter.SeekLT(startKey); iter.Valid() && iter.Key() != nil; iter.Prev() {
		key := string(iter.Key())
		fmt.Printf("[PrivateChatDB] GetPrivateChatsByMetaIdsAndTimestampRange key: %s\n", key)

		// Parse key to extract timestamp
		// Key format: from_to_timestamp+number(6)
		keyParts := strings.Split(key, "_")
		if len(keyParts) < 3 {
			continue
		}

		// Extract timestamp from key (remove the last 6 digits which is the random number)
		timestampStr := keyParts[2]
		timestampKey := timestampStr
		timestampKeyInt, _ := strconv.ParseInt(timestampKey, 10, 64)

		// Skip messages after our start timestamp (since we're going backwards)
		if timestampKeyInt > startTimestamp {
			continue
		}

		// Parse value: pinId_chatType_timestamp_number
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

		if nextTimestamp == 0 || timestampKeyInt < nextTimestamp {
			nextTimestamp = timestampKeyInt
		}

		// Add to results
		chats = append(chats, chat)

		// Check pagination limit
		if int64(len(chats)) >= size {
			break
		}
	}

	return chats, nextTimestamp, nil
}

// Get latest private chat messages between two users (reverse order based on timestamp)
func (pcdb *PrivateChatDB) GetLatestPrivateChatsByMetaIds(selfMetaId, otherMetaId string, size int64) ([]*models.TalkPrivateChatV3, int64, error) {
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

// Update private chat context list
func (pcdb *PrivateChatDB) UpdatePrivateContextList(chat *models.TalkPrivateChatV3) error {
	// Update sender's context list
	err := pcdb.updateSingleUserPrivateContextList(chat.From, chat.To, chat)
	if err != nil {
		return err
	}

	// Update receiver's context list
	err = pcdb.updateSingleUserPrivateContextList(chat.To, chat.From, chat)
	if err != nil {
		return err
	}

	return nil
}

// Update single user's private chat contact list
func (pcdb *PrivateChatDB) updateSingleUserPrivateContextList(selfMetaId, otherMetaId string, chat *models.TalkPrivateChatV3) error {
	// Get mutex for this MetaId to prevent concurrent updates
	mutex := GetMetaIdMutex(selfMetaId)
	mutex.Lock()
	defer mutex.Unlock()

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
		if item.MetaId == otherMetaId && item.Type == "2" {
			contextList.Items[i] = newItem
			found = true
			// Check if update is needed
			if item.LastMessagePinId != newItem.LastMessagePinId ||
				item.BlockHeight != newItem.BlockHeight {
				// Update existing item
				shouldUpdate = true
			}

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

	// Remove duplicates for private chat items (Type == "2")
	pcdb.removeDuplicatePrivateChatItems(contextList)

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

// Remove duplicate private chat items (Type == "2") based on MetaId
func (pcdb *PrivateChatDB) removeDuplicatePrivateChatItems(contextList *models.MetaIdContextList) {
	metaIdMap := make(map[string]*models.MetaIdContextItem)
	var uniqueItems []*models.MetaIdContextItem

	for _, item := range contextList.Items {
		if item.Type == "2" {
			// For private chat items, check for duplicate MetaId
			if existingItem, exists := metaIdMap[item.MetaId]; exists {
				// If current item has larger timestamp, replace the existing one
				if item.Timestamp >= existingItem.Timestamp {
					metaIdMap[item.MetaId] = item
				}
				// Skip adding to uniqueItems for now
				continue
			}
			metaIdMap[item.MetaId] = item
		} else {
			uniqueItems = append(uniqueItems, item)
		}
	}

	// Add the unique private chat items (with largest timestamps) back to the list
	for _, item := range metaIdMap {
		uniqueItems = append(uniqueItems, item)
	}

	contextList.Items = uniqueItems
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
		hasError := false

		// Update private chat contact list
		err = pcdb.UpdatePrivateContextList(message.Chat)
		if err != nil {
			log.Printf("Failed to update private contact list for pinId %s: %v", message.PinId, err)
			hasError = true
		}

		// Update private chat index
		err = pcdb.UpdatePrivateChatIndex(message.Chat)
		if err != nil {
			log.Printf("Failed to update private chat index for pinId %s: %v", message.PinId, err)
			hasError = true
		}

		// Only delete queue message if both operations succeeded
		if !hasError {
			log.Printf("Processing private chat message for pinId %s", message.PinId)

			err = pcdb.deletePrivateQueueMessage(message.PinId)
			if err != nil {
				// Log error but don't affect main flow
				log.Printf("Failed to delete private queue message for pinId %s: %v", message.PinId, err)
			}
		}
	}

	return nil
}

// Start private chat queue processing goroutine (needs to be called when application starts)
func (pcdb *PrivateChatDB) StartPrivateQueueProcessor() {
	go func() {
		ticker := time.NewTicker(2 * time.Second) // Process every 2 seconds
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				// Check if already processing
				pcdb.processingMutex.Lock()
				if pcdb.isProcessing {
					pcdb.processingMutex.Unlock()
					log.Printf("[PrivateChatDB] Queue processor is already running, skipping this cycle")
					continue
				}
				pcdb.isProcessing = true
				pcdb.processingMutex.Unlock()

				// Batch process queue messages
				err := pcdb.ProcessPrivateQueueMessages(100) // Process 100 messages each time
				if err != nil {
					log.Printf("[PrivateChatDB] Error processing queue messages: %v", err)
				}

				// Mark processing as complete
				pcdb.processingMutex.Lock()
				pcdb.isProcessing = false
				pcdb.processingMutex.Unlock()
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
		} else if strings.ToLower(protocol) == strings.ToLower(protocols.MonitorSimplePrivateBlock) {
			return pcdb.processPrivateChatBlock(pin)
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
	var simpleMsg protocols.SimpleMsg
	err = json.Unmarshal(pin.ContentBody, &simpleMsg)
	if err != nil {
		return err
	}

	// Convert To field to MetaId format if needed
	toMetaId := pcdb.convertToMetaId(simpleMsg.To)

	// Create private chat message model
	chat := &models.TalkPrivateChatV3{
		From:        pin.CreateMetaId,  // Sender MetaId
		FromAddress: pin.CreateAddress, // Sender address
		To:          toMetaId,          // Receiver MetaId (converted if needed)
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

	isGoEnqueue := true
	// Check if the receiver has blocked the sender
	isBlocked, err := pcdb.IsUserBlockedByChatTimestamp(chat.To, chat.From, chat.Timestamp)
	if err != nil {
		log.Printf("Failed to check block status for chat %s: %v", chat.PinId, err)
		// If we can't determine block status, save to normal collection as fallback
		err = pcdb.SavePrivateChatTimestamp(chat)
		if err != nil {
			return err
		}
	} else if isBlocked {
		// If blocked, save to out collection
		err = pcdb.SavePrivateChatOutTimestamp(chat)
		if err != nil {
			return err
		}
		isGoEnqueue = false
	} else {
		// If not blocked, save to normal collection
		err = pcdb.SavePrivateChatTimestamp(chat)
		if err != nil {
			return err
		}
	}

	// Enqueue message for asynchronous processing
	if isGoEnqueue {
		err = pcdb.EnqueuePrivateChatMessage(chat)
		if err != nil {
			return err
		}
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

	// Convert To field to MetaId format if needed
	toMetaId := pcdb.convertToMetaId(simpleFileMsg.To)

	// Create private chat message model
	chat := &models.TalkPrivateChatV3{
		From:        pin.CreateMetaId, // Sender MetaId
		FromAddress: pin.CreateAddress,
		To:          toMetaId, // Receiver MetaId (converted if needed)
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

	isGoEnqueue := true
	// Check if the receiver has blocked the sender
	isBlocked, err := pcdb.IsUserBlockedByChatTimestamp(chat.To, chat.From, chat.Timestamp)
	if err != nil {
		log.Printf("Failed to check block status for file chat %s: %v", chat.PinId, err)
		// If we can't determine block status, save to normal collection as fallback
		err = pcdb.SavePrivateChatTimestamp(chat)
		if err != nil {
			return err
		}
	} else if isBlocked {
		// If blocked, save to out collection
		err = pcdb.SavePrivateChatOutTimestamp(chat)
		if err != nil {
			return err
		}
		isGoEnqueue = false
	} else {
		// If not blocked, save to normal collection
		err = pcdb.SavePrivateChatTimestamp(chat)
		if err != nil {
			return err
		}
	}

	if isGoEnqueue {
		// Enqueue message for asynchronous processing
		err = pcdb.EnqueuePrivateChatMessage(chat)
		if err != nil {
			return err
		}
	}

	return nil
}

// UpdatePrivateChatIndex updates the chat index for a private chat message
func (pcdb *PrivateChatDB) UpdatePrivateChatIndex(chat *models.TalkPrivateChatV3) error {
	// Get the next index for this conversation (from_to)
	nextIndex, err := pcdb.getNextPrivateChatIndex(chat.From, chat.To)
	if err != nil {
		return err
	}

	// Update the chat message with the new index
	chat.Index = nextIndex

	// Save the updated chat message
	err = pcdb.SavePrivateChat(chat)
	if err != nil {
		return err
	}

	// Save the index mapping with zero-padded index for proper sorting
	indexKey1 := chat.From + "_" + chat.To + "_" + fmt.Sprintf("%040d", nextIndex)
	indexValue := chat.PinId + "_" + strconv.FormatInt(int64(chat.ChatType), 10) + "_" + strconv.FormatInt(chat.Timestamp, 10) + "_1"

	indexKey2 := chat.To + "_" + chat.From + "_" + fmt.Sprintf("%040d", nextIndex)
	indexValue2 := chat.PinId + "_" + strconv.FormatInt(int64(chat.ChatType), 10) + "_" + strconv.FormatInt(chat.Timestamp, 10) + "_1"

	err = Pb[TalkPrivateChatIndexCollection].Set([]byte(indexKey1), []byte(indexValue), pebble.Sync)
	if err != nil {
		return err
	}

	err = Pb[TalkPrivateChatIndexCollection].Set([]byte(indexKey2), []byte(indexValue2), pebble.Sync)
	if err != nil {
		return err
	}

	return nil
}

// getNextPrivateChatIndex gets the next available index for a private conversation
func (pcdb *PrivateChatDB) getNextPrivateChatIndex(fromMetaId, toMetaId string) (int64, error) {
	iter, err := Pb[TalkPrivateChatIndexCollection].NewIter(&pebble.IterOptions{
		LowerBound: []byte(fromMetaId + "_" + toMetaId + "_"),
		UpperBound: []byte(fromMetaId + "_" + toMetaId + "_" + string([]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff})),
	})
	if err != nil {
		return 0, err
	}
	defer iter.Close()

	// Find the last key for this conversation (since keys are sorted, the last one has the highest index)
	var lastKey []byte

	// Use Last() to get the last key in the range
	if iter.Last(); iter.Valid() {
		lastKey = iter.Key()
	}

	// If no keys found for this conversation, start with index 1
	if lastKey == nil {
		return 1, nil
	}

	// Extract index from the last key (fromMetaId_toMetaId_index with zero-padding)
	keyStr := string(lastKey)
	parts := strings.Split(keyStr, "_")
	if len(parts) >= 3 {
		// Remove leading zeros and parse the index
		indexStr := strings.TrimLeft(parts[2], "0")
		if indexStr == "" {
			indexStr = "0" // If all zeros, treat as 0
		}
		if index, err := strconv.ParseInt(indexStr, 10, 64); err == nil {
			return index + 1, nil
		}
	}

	// Fallback: if parsing fails, start with index 1
	return 1, nil
}

// GetCurrentMaxPrivateChatIndex gets the current maximum index for a private conversation
func (pcdb *PrivateChatDB) GetCurrentMaxPrivateChatIndex(fromMetaId, toMetaId string) (int64, error) {
	iter, err := Pb[TalkPrivateChatIndexCollection].NewIter(&pebble.IterOptions{
		LowerBound: []byte(fromMetaId + "_" + toMetaId + "_"),
		UpperBound: []byte(fromMetaId + "_" + toMetaId + "_" + string([]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff})),
	})
	if err != nil {
		return 0, err
	}
	defer iter.Close()

	// Find the last key for this conversation (since keys are sorted, the last one has the highest index)
	var lastKey []byte

	// Use Last() to get the last key in the range
	if iter.Last(); iter.Valid() {
		lastKey = iter.Key()
	}

	// If no keys found for this conversation, return 0
	if lastKey == nil {
		return 0, nil
	}

	// Extract index from the last key (fromMetaId_toMetaId_index with zero-padding)
	keyStr := string(lastKey)
	parts := strings.Split(keyStr, "_")
	if len(parts) >= 3 {
		// Remove leading zeros and parse the index
		indexStr := strings.TrimLeft(parts[2], "0")
		if indexStr == "" {
			indexStr = "0" // If all zeros, treat as 0
		}
		if index, err := strconv.ParseInt(indexStr, 10, 64); err == nil {
			return index, nil
		}
	}

	// Fallback: if parsing fails, return 0
	return 0, nil
}

// convertToMetaId converts a string to MetaId format
// If the string is not 64 characters long, it's treated as an address and converted to MetaId using SHA256
func (pcdb *PrivateChatDB) convertToMetaId(input string) string {
	// If input is already 64 characters long, assume it's already a MetaId
	if len(input) == 64 {
		return input
	}

	// Otherwise, treat it as an address and convert to MetaId using SHA256
	hash := sha256.Sum256([]byte(input))
	return hex.EncodeToString(hash[:])
}

// Process private chat block operation
func (pcdb *PrivateChatDB) processPrivateChatBlock(pin *pin.PinInscription) error {
	// Check if this PinId has already been saved
	existingBlock, err := pcdb.GetPrivateChatBlockByPinId(pin.Id)
	if err == nil && existingBlock != nil {
		if existingBlock.BlockHeight != pin.GenesisHeight {
			existingBlock.BlockHeight = pin.GenesisHeight
			err = pcdb.SavePrivateChatBlock(existingBlock)
			if err != nil {
				return err
			}
		}
		// Already exists, skip processing
		return nil
	}

	// Parse protocol data
	var simplePrivateBlock protocols.SimplePrivateBlock
	err = json.Unmarshal(pin.ContentBody, &simplePrivateBlock)
	if err != nil {
		return err
	}

	// Convert To field to MetaId format if needed
	toMetaId := pcdb.convertToMetaId(simplePrivateBlock.To)

	// Create private chat block model
	blockRecord := &models.SimplePrivateBlock{
		PinId:       pin.Id,
		TxId:        pin.Id[:len(pin.Id)-2],
		To:          toMetaId,
		BlockState:  toInt64(simplePrivateBlock.BlockState),
		Protocol:    pin.Path,
		Timestamp:   pin.Timestamp,
		Chain:       pin.ChainName,
		BlockHeight: pin.GenesisHeight,
	}

	// Save block record to TalkPrivateChatBlockPinCollection
	err = pcdb.SavePrivateChatBlock(blockRecord)
	if err != nil {
		return err
	}

	// Update the block list in TalkPrivateChatMetaIdBlockListCollection
	err = pcdb.UpdatePrivateChatBlockList(pin.CreateMetaId, toMetaId, blockRecord)
	if err != nil {
		return err
	}

	return nil
}

// GetPrivateChatBlockByPinId retrieves a private chat block record by PinId
func (pcdb *PrivateChatDB) GetPrivateChatBlockByPinId(pinId string) (*models.SimplePrivateBlock, error) {
	key := []byte(pinId)
	value, closer, err := Pb[TalkPrivateChatBlockPinCollection].Get(key)
	if err != nil {
		if err == pebble.ErrNotFound {
			return nil, nil
		}
		return nil, err
	}
	defer closer.Close()

	var blockRecord models.SimplePrivateBlock
	err = json.Unmarshal(value, &blockRecord)
	if err != nil {
		return nil, err
	}

	return &blockRecord, nil
}

// SavePrivateChatBlock saves a private chat block record to TalkPrivateChatBlockPinCollection
func (pcdb *PrivateChatDB) SavePrivateChatBlock(block *models.SimplePrivateBlock) error {
	data, err := json.Marshal(block)
	if err != nil {
		return err
	}

	// Use PinId as primary key
	key := []byte(block.PinId)
	return Pb[TalkPrivateChatBlockPinCollection].Set(key, data, pebble.Sync)
}

// UpdatePrivateChatBlockList updates the block list in TalkPrivateChatMetaIdBlockListCollection
func (pcdb *PrivateChatDB) UpdatePrivateChatBlockList(fromMetaId, toMetaId string, blockRecord *models.SimplePrivateBlock) error {
	// Get existing block list for fromMetaId
	blockList, err := pcdb.GetPrivateChatBlockList(fromMetaId)
	if err != nil {
		return err
	}

	// Create new block item
	newBlockItem := &models.TalkPrivateChatBlock{
		BlockPinId:     blockRecord.PinId,
		BlockMetaId:    blockRecord.To,
		BlockState:     blockRecord.BlockState,
		BlockTimestamp: blockRecord.Timestamp,
	}

	// Check if this user is already in the block list
	found := false
	for i, item := range blockList.Items {
		if item.BlockPinId == blockRecord.PinId {
			// Update existing item
			blockList.Items[i] = newBlockItem
			found = true
			break
		}
	}

	if !found {
		// Add new item to the list
		blockList.Items = append(blockList.Items, newBlockItem)
	}

	// Sort the list by timestamp in descending order (newest first)
	pcdb.sortPrivateChatBlockListByTimestamp(blockList)

	// Save updated block list
	return pcdb.SavePrivateChatBlockList(fromMetaId, blockList)
}

// GetPrivateChatBlockList gets the block list for a specific user from TalkPrivateChatMetaIdBlockListCollection
func (pcdb *PrivateChatDB) GetPrivateChatBlockList(metaId string) (*models.TalkPrivateChatBlockList, error) {
	key := []byte(metaId)
	value, closer, err := Pb[TalkPrivateChatMetaIdBlockListCollection].Get(key)
	if err != nil {
		if err == pebble.ErrNotFound {
			// Return empty list if not found
			return &models.TalkPrivateChatBlockList{
				MetaId: metaId,
				Items:  []*models.TalkPrivateChatBlock{},
			}, nil
		}
		return nil, err
	}
	defer closer.Close()

	var blockList models.TalkPrivateChatBlockList
	err = json.Unmarshal(value, &blockList)
	if err != nil {
		return nil, err
	}

	return &blockList, nil
}

// SavePrivateChatBlockList saves the block list to TalkPrivateChatMetaIdBlockListCollection
func (pcdb *PrivateChatDB) SavePrivateChatBlockList(metaId string, blockList *models.TalkPrivateChatBlockList) error {
	data, err := json.Marshal(blockList)
	if err != nil {
		return err
	}

	key := []byte(metaId)
	return Pb[TalkPrivateChatMetaIdBlockListCollection].Set(key, data, pebble.Sync)
}

// sortPrivateChatBlockListByTimestamp sorts the block list by timestamp in descending order
func (pcdb *PrivateChatDB) sortPrivateChatBlockListByTimestamp(blockList *models.TalkPrivateChatBlockList) {
	// Simple bubble sort, descending order by timestamp
	for i := 0; i < len(blockList.Items)-1; i++ {
		for j := 0; j < len(blockList.Items)-1-i; j++ {
			if blockList.Items[j].BlockTimestamp < blockList.Items[j+1].BlockTimestamp {
				blockList.Items[j], blockList.Items[j+1] = blockList.Items[j+1], blockList.Items[j]
			}
		}
	}
}

// IsUserBlockedByChatTimestamp checks if fromMetaId has blocked toMetaId at the specific chat timestamp
// This method handles multiple block/unblock operations by checking the block state at the given timestamp
func (pcdb *PrivateChatDB) IsUserBlockedByChatTimestamp(fromMetaId, toMetaId string, chatTimestamp int64) (bool, error) {
	// Get block list for fromMetaId
	blockList, err := pcdb.GetPrivateChatBlockList(fromMetaId)
	if err != nil {
		return false, err
	}

	// Find the most recent block operation that affects this chat timestamp
	var currentBlockState int64 = 0 // 0: no block, 1: blocked, -1: unblocked
	if blockList.Items == nil || len(blockList.Items) == 0 {
		return false, nil
	}

	// Sort block list by timestamp to process operations in chronological order
	// We need to find the last operation before or at the chat timestamp
	for _, item := range blockList.Items {
		// Only consider operations that happened before or at the chat timestamp
		if item.BlockTimestamp <= chatTimestamp {
			// Check if this is the target user
			if item.BlockMetaId == toMetaId {
				// Update the current block state based on this operation
				currentBlockState = item.BlockState
			}
		}
	}

	// Return true if the user is currently blocked (BlockState == 1)
	return currentBlockState == 1, nil
}

// GetUserBlockList gets all block records for a specific user
func (pcdb *PrivateChatDB) GetUserBlockList(metaId string) ([]*models.SimplePrivateBlock, error) {
	// Get block list from TalkPrivateChatMetaIdBlockListCollection
	blockList, err := pcdb.GetPrivateChatBlockList(metaId)
	if err != nil {
		return nil, err
	}

	var blockRecords []*models.SimplePrivateBlock

	// Get actual block records from TalkPrivateChatBlockPinCollection
	for _, item := range blockList.Items {
		blockRecord, err := pcdb.GetPrivateChatBlockByPinId(item.BlockPinId)
		if err != nil || blockRecord == nil {
			continue
		}
		blockRecords = append(blockRecords, blockRecord)
	}

	return blockRecords, nil
}
