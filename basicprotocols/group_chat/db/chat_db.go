package db

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"manindexer/basicprotocols/group_chat/models"
	"manindexer/basicprotocols/group_chat/protocols"
	"manindexer/pin"
	"strconv"
	"strings"
	"time"

	"github.com/btcsuite/btcd/wire"
	"github.com/cockroachdb/pebble"
)

// Queue message item
type QueueChatMessage struct {
	PinId      string                  `json:"pinId"`      // Message PinId
	GroupId    string                  `json:"groupId"`    // Group ID
	Chat       *models.TalkGroupChatV3 `json:"chat"`       // Chat message
	Timestamp  int64                   `json:"timestamp"`  // Enqueue timestamp
	RetryCount int                     `json:"retryCount"` // Retry count
	Status     string                  `json:"status"`     // Processing status: pending, processing, completed, failed
}

// Chat database operations
type ChatDB struct {
	pb *Pebble
}

func NewChatDB(pb *Pebble) *ChatDB {
	return &ChatDB{pb: pb}
}

// Save chat message
func (cdb *ChatDB) SaveChat(chat *models.TalkGroupChatV3) error {
	data, err := json.Marshal(chat)
	if err != nil {
		return err
	}

	// Use PinId as primary key
	key := []byte(chat.PinId)
	return Pb[TalkGroupChatPinCollection].Set(key, data, pebble.Sync)
}

// Save chat timestamp index
func (cdb *ChatDB) SaveChatTimestamp(chat *models.TalkGroupChatV3) error {
	// Construct timestamp index value: pinId_chatType_timestamp
	value := chat.PinId + "_" + strconv.FormatInt(int64(chat.ChatType), 10) + "_" + strconv.FormatInt(chat.Timestamp, 10)

	// Use GroupId_Timestamp as primary key to support timestamp range queries
	key := []byte(chat.GroupId + "_" + strconv.FormatInt(chat.Timestamp, 10))
	return Pb[TalkGroupChatTimestampCollection].Set(key, []byte(value), pebble.Sync)
}

// Get chat message by PinId
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

// Get chat message list by group ID
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

// Get chat message list by community ID
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

// Get chat message list by group ID and start timestamp (reverse order, pagination based on timestamp)
func (cdb *ChatDB) GetChatsByGroupIdAndTimestampRange(groupId string, startTimestamp int64, size int64) ([]*models.TalkGroupChatV3, error) {
	var chats []*models.TalkGroupChatV3
	iter, err := Pb[TalkGroupChatTimestampCollection].NewIter(nil)
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	// Construct query start key: groupId_startTimestamp
	startKey := []byte(groupId + "_" + strconv.FormatInt(startTimestamp, 10))

	// Start reverse iteration from specified timestamp (latest messages first)
	for iter.SeekLT(startKey); iter.Valid() && iter.Key() != nil; iter.Prev() {
		key := string(iter.Key())

		// Check if it belongs to the specified group
		if !strings.HasPrefix(key, groupId+"_") {
			continue
		}

		// Parse index value to get PinId
		value := string(iter.Value())
		valueParts := strings.Split(value, "_")
		if len(valueParts) < 1 {
			continue
		}
		pinId := valueParts[0]

		// Get complete chat message
		chat, err := cdb.GetChatByPinId(pinId)
		if err != nil || chat == nil {
			continue
		}

		// Reach pagination size limit
		if int64(len(chats)) >= size {
			break
		}

		chats = append(chats, chat)
	}

	return chats, nil
}

// Get latest chat messages for group (reverse order based on timestamp)
func (cdb *ChatDB) GetLatestChatsByGroupId(groupId string, size int64) ([]*models.TalkGroupChatV3, error) {
	// Use current time as start timestamp
	currentTimestamp := time.Now().Unix()
	return cdb.GetChatsByGroupIdAndTimestampRange(groupId, currentTimestamp, size)
}

// Delete chat message
func (cdb *ChatDB) DeleteChat(pinId string) error {
	key := []byte(pinId)
	return Pb[TalkGroupChatPinCollection].Delete(key, pebble.Sync)
}

// Save lucky bag info
func (cdb *ChatDB) SaveLuckyBag(red *models.TalkGroupLuckyBagV3) error {
	data, err := json.Marshal(red)
	if err != nil {
		return err
	}

	// Use PinId as primary key
	key := []byte(red.PinId)
	return Pb[TalkGroupLuckyBagPinCollection].Set(key, data, pebble.Sync)
}

// Get lucky bag info by PinId
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

// Get lucky bag list by group ID
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

// Save grab lucky bag info
func (cdb *ChatDB) SaveOpenLuckyBag(open *models.TalkGroupOpenLuckyBagV3) error {
	data, err := json.Marshal(open)
	if err != nil {
		return err
	}

	// Use PinId as primary key
	key := []byte(open.PinId)
	return Pb[TalkGroupOpenLuckyBagPinCollection].Set(key, data, pebble.Sync)
}

// Get grab lucky bag info by PinId
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

// Get grab lucky bag list by lucky bag TxId
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

// Save residue lucky bag info
func (cdb *ChatDB) SaveResidueLuckyBag(residue *models.TalkGroupResidueLuckyBagV3) error {
	data, err := json.Marshal(residue)
	if err != nil {
		return err
	}

	// Use PinId as primary key
	key := []byte(residue.PinId)
	return Pb[TalkGroupResidueLuckyBagPinCollection].Set(key, data, pebble.Sync)
}

// Get residue lucky bag info by lucky bag PinId
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

// Save grab lucky bag list record
func (cdb *ChatDB) SaveOpenLuckyBagList(luckyBagPinId string, openPinId string, groupId string, timestamp int64, createMetaId string, createAddress string, luckyBagOutIndex int64) error {
	// Get existing list
	list, err := cdb.GetOpenLuckyBagList(luckyBagPinId)
	if err != nil {
		return err
	}

	// Add new record
	newItem := &models.OpenLuckyBagListItem{
		OpenPinId:        openPinId,
		GroupId:          groupId,
		Timestamp:        timestamp,
		CreateAddress:    createAddress,
		CreateMetaId:     createMetaId,
		LuckyBagOutIndex: luckyBagOutIndex,
	}

	// Check if already exists
	found := false
	for _, item := range list.Items {
		if item.OpenPinId == openPinId {
			found = true
			break
		}
	}

	if !found {
		list.Items = append(list.Items, newItem)
	}

	// Save updated list
	data, err := json.Marshal(list)
	if err != nil {
		return err
	}

	key := []byte(luckyBagPinId)
	return Pb[TalkGroupOpenLuckyBagListCollection].Set(key, data, pebble.Sync)
}

// Get grab lucky bag list
func (cdb *ChatDB) GetOpenLuckyBagList(luckyBagPinId string) (*models.OpenLuckyBagList, error) {
	key := []byte(luckyBagPinId)
	value, closer, err := Pb[TalkGroupOpenLuckyBagListCollection].Get(key)
	if err != nil {
		if err == pebble.ErrNotFound {
			return &models.OpenLuckyBagList{LuckyBagPinId: luckyBagPinId, Items: []*models.OpenLuckyBagListItem{}}, nil
		}
		return nil, err
	}
	defer closer.Close()

	var list models.OpenLuckyBagList
	err = json.Unmarshal(value, &list)
	if err != nil {
		return nil, err
	}

	return &list, nil
}

// Save reclaim lucky bag list record
func (cdb *ChatDB) SaveResidueLuckyBagList(luckyBagPinId string, residuePinId string, groupId string, timestamp int64, createMetaId string, createAddress string, luckyBagOutIndexList []int64) error {
	// Get existing list
	list, err := cdb.GetResidueLuckyBagList(luckyBagPinId)
	if err != nil {
		return err
	}

	// Add new record
	newItem := &models.ResidueLuckyBagListItem{
		ResiduePinId:         residuePinId,
		GroupId:              groupId,
		Timestamp:            timestamp,
		CreateMetaId:         createMetaId,
		CreateAddress:        createAddress,
		LuckyBagOutIndexList: luckyBagOutIndexList,
	}

	// Check if already exists
	found := false
	for _, item := range list.Items {
		if item.ResiduePinId == residuePinId {
			found = true
			break
		}
	}

	if !found {
		list.Items = append(list.Items, newItem)
	}

	// Save updated list
	data, err := json.Marshal(list)
	if err != nil {
		return err
	}

	key := []byte(luckyBagPinId)
	return Pb[TalkGroupResidueLuckyBagListCollection].Set(key, data, pebble.Sync)
}

// Get reclaim lucky bag list
func (cdb *ChatDB) GetResidueLuckyBagList(luckyBagPinId string) (*models.ResidueLuckyBagList, error) {
	key := []byte(luckyBagPinId)
	value, closer, err := Pb[TalkGroupResidueLuckyBagListCollection].Get(key)
	if err != nil {
		if err == pebble.ErrNotFound {
			return &models.ResidueLuckyBagList{LuckyBagPinId: luckyBagPinId, Items: []*models.ResidueLuckyBagListItem{}}, nil
		}
		return nil, err
	}
	defer closer.Close()

	var list models.ResidueLuckyBagList
	err = json.Unmarshal(value, &list)
	if err != nil {
		return nil, err
	}

	return &list, nil
}

// Get user's group list
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

// Save user's group list
func (cdb *ChatDB) SaveMetaIdContextList(contextList *models.MetaIdContextList) error {
	data, err := json.Marshal(contextList)
	if err != nil {
		return err
	}

	key := []byte(contextList.MetaId)
	return Pb[TalkMetaIdContextListCollection].Set(key, data, pebble.Sync)
}

// Update group list for all members in the group (when there's a new message)
func (cdb *ChatDB) UpdateGroupMembersContextList(groupId string, chat *models.TalkGroupChatV3, groupDB *GroupDB) error {
	// Update group latest chat record
	err := cdb.updateGroupLatestChat(groupId, chat)
	if err != nil {
		return err
	}

	// Get all members of the group
	members, err := groupDB.GetGroupMembers(groupId)
	if err != nil {
		return err
	}

	// Update group list for each member
	for _, member := range members {
		err = cdb.updateSingleMemberContextList(member.MetaId, groupId, chat)
		if err != nil {
			return err
		}
	}

	return nil
}

// Update group latest chat record
func (cdb *ChatDB) updateGroupLatestChat(groupId string, chat *models.TalkGroupChatV3) error {
	// First get existing latest chat record
	existingLatestChat, err := cdb.GetGroupLatestChat(groupId)
	if err != nil {
		return err
	}

	// Create new latest chat record
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

	// If no existing data, save directly
	if existingLatestChat == nil {
		data, err := json.Marshal(newLatestChat)
		if err != nil {
			return err
		}
		key := []byte(groupId)
		return Pb[TalkGroupLatestChatCollection].Set(key, data, pebble.Sync)
	}

	// Determine if update is needed
	shouldUpdate := false

	// 1. First check if pinId is the same
	if existingLatestChat.LastMessagePinId == chat.PinId {
		// pinId is the same, check if blockHeight is different
		if existingLatestChat.BlockHeight != chat.BlockHeight {
			shouldUpdate = true
		}
	} else {
		// pinId is different, compare timestamp
		if chat.Timestamp > existingLatestChat.Timestamp {
			shouldUpdate = true
		}
	}

	// If update is needed, save new data
	if shouldUpdate {
		data, err := json.Marshal(newLatestChat)
		if err != nil {
			return err
		}
		key := []byte(groupId)
		return Pb[TalkGroupLatestChatCollection].Set(key, data, pebble.Sync)
	}

	// No update needed, return directly
	return nil
}

// Get group latest chat record
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

// Update single member's group list
func (cdb *ChatDB) updateSingleMemberContextList(metaId, groupId string, chat *models.TalkGroupChatV3) error {
	// Get user's group list
	contextList, err := cdb.GetMetaIdContextList(metaId)
	if err != nil {
		return err
	}

	// Create new group list item
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

	// Whether update is needed
	shouldUpdate := false
	// Check if group item already exists
	found := false
	for i, item := range contextList.Items {
		if item.GroupId == groupId {
			// Update existing item
			contextList.Items[i] = newItem
			found = true
			if item.LastMessagePinId != newItem.LastMessagePinId ||
				item.BlockHeight != newItem.BlockHeight {
				shouldUpdate = true
			}
			break
		}
	}

	// If not found, add new item
	if !found {
		contextList.Items = append(contextList.Items, newItem)
	}

	// If no update needed, return directly
	if !shouldUpdate {
		return nil
	}

	// Sort by timestamp in reverse order
	cdb.sortContextListByTimestamp(contextList)

	// Save updated group list
	return cdb.SaveMetaIdContextList(contextList)
}

// Sort group list by timestamp in reverse order
func (cdb *ChatDB) sortContextListByTimestamp(contextList *models.MetaIdContextList) {
	// Simple bubble sort, reverse order by timestamp
	for i := 0; i < len(contextList.Items)-1; i++ {
		for j := 0; j < len(contextList.Items)-1-i; j++ {
			if contextList.Items[j].Timestamp < contextList.Items[j+1].Timestamp {
				contextList.Items[j], contextList.Items[j+1] = contextList.Items[j+1], contextList.Items[j]
			}
		}
	}
}

// Enqueue chat message (asynchronous processing for group list updates)
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

	// Use timestamp_pinId as primary key to support processing in time order
	key := []byte(strconv.FormatInt(queueMessage.Timestamp, 10) + "_" + chat.PinId)
	return Pb[TalkGroupChatQueueCollection].Set(key, data, pebble.Sync)
}

// Get pending messages from queue
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

		// Only process messages with pending status
		if queueMessage.Status == "pending" {
			messages = append(messages, &queueMessage)
			count++
		}
	}

	return messages, nil
}

// Delete queue message data
func (cdb *ChatDB) deleteQueueMessage(pinId string) error {
	iter, err := Pb[TalkGroupChatQueueCollection].NewIter(nil)
	if err != nil {
		return err
	}
	defer iter.Close()

	// Find queue message containing this pinId
	for iter.First(); iter.Valid(); iter.Next() {
		value := string(iter.Value())

		var queueMessage QueueChatMessage
		err := json.Unmarshal([]byte(value), &queueMessage)
		if err != nil {
			continue
		}

		// Found matching pinId, delete this queue message
		if queueMessage.PinId == pinId {
			return Pb[TalkGroupChatQueueCollection].Delete(iter.Key(), pebble.Sync)
		}
	}

	return nil
}

// Grab lucky bag queue message item
type QueueOpenLuckyBagMessage struct {
	PinId        string                          `json:"pinId"`        // Message PinId
	OpenLuckyBag *models.TalkGroupOpenLuckyBagV3 `json:"openLuckyBag"` // Grab lucky bag record
	Timestamp    int64                           `json:"timestamp"`    // Enqueue timestamp
	RetryCount   int                             `json:"retryCount"`   // Retry count
	Status       string                          `json:"status"`       // Processing status: pending, processing, completed, failed
}

// Enqueue grab lucky bag record
func (cdb *ChatDB) EnqueueOpenLuckyBagMessage(openLuckyBag *models.TalkGroupOpenLuckyBagV3) error {
	queueMessage := &QueueOpenLuckyBagMessage{
		PinId:        openLuckyBag.PinId,
		OpenLuckyBag: openLuckyBag,
		Timestamp:    time.Now().Unix(),
		RetryCount:   0,
		Status:       "pending",
	}

	data, err := json.Marshal(queueMessage)
	if err != nil {
		return err
	}

	// Use timestamp_pinId as primary key to support processing in time order
	key := []byte(strconv.FormatInt(queueMessage.Timestamp, 10) + "_" + openLuckyBag.PinId)
	return Pb[TalkGroupOpenLuckyBagQueueCollection].Set(key, data, pebble.Sync)
}

// Get pending messages from grab lucky bag queue
func (cdb *ChatDB) GetPendingOpenLuckyBagMessages(limit int) ([]*QueueOpenLuckyBagMessage, error) {
	var messages []*QueueOpenLuckyBagMessage
	iter, err := Pb[TalkGroupOpenLuckyBagQueueCollection].NewIter(nil)
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	count := 0
	for iter.First(); iter.Valid() && iter.Key() != nil && count < limit; iter.Next() {
		value := string(iter.Value())

		var queueMessage QueueOpenLuckyBagMessage
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

// Delete grab lucky bag queue message data
func (cdb *ChatDB) DeleteOpenLuckyBagQueueMessage(pinId string) error {
	iter, err := Pb[TalkGroupOpenLuckyBagQueueCollection].NewIter(nil)
	if err != nil {
		return err
	}
	defer iter.Close()

	// Find queue message containing this pinId
	for iter.First(); iter.Valid(); iter.Next() {
		value := string(iter.Value())

		var queueMessage QueueOpenLuckyBagMessage
		err := json.Unmarshal([]byte(value), &queueMessage)
		if err != nil {
			continue
		}

		// Found matching pinId, delete this queue message
		if queueMessage.PinId == pinId {
			return Pb[TalkGroupOpenLuckyBagQueueCollection].Delete(iter.Key(), pebble.Sync)
		}
	}

	return nil
}

// Reclaim lucky bag queue message item
type QueueResidueLuckyBagMessage struct {
	PinId           string                             `json:"pinId"`           // Message PinId
	ResidueLuckyBag *models.TalkGroupResidueLuckyBagV3 `json:"residueLuckyBag"` // Reclaim lucky bag record
	Timestamp       int64                              `json:"timestamp"`       // Enqueue timestamp
	RetryCount      int                                `json:"retryCount"`      // Retry count
	Status          string                             `json:"status"`          // Processing status: pending, processing, completed, failed
}

// Enqueue reclaim lucky bag record
func (cdb *ChatDB) EnqueueResidueLuckyBagMessage(residueLuckyBag *models.TalkGroupResidueLuckyBagV3) error {
	queueMessage := &QueueResidueLuckyBagMessage{
		PinId:           residueLuckyBag.PinId,
		ResidueLuckyBag: residueLuckyBag,
		Timestamp:       time.Now().Unix(),
		RetryCount:      0,
		Status:          "pending",
	}

	data, err := json.Marshal(queueMessage)
	if err != nil {
		return err
	}

	// Use timestamp_pinId as primary key to support processing in time order
	key := []byte(strconv.FormatInt(queueMessage.Timestamp, 10) + "_" + residueLuckyBag.PinId)
	return Pb[TalkGroupResidueLuckyBagQueueCollection].Set(key, data, pebble.Sync)
}

// Get pending messages from reclaim lucky bag queue
func (cdb *ChatDB) GetPendingResidueLuckyBagMessages(limit int) ([]*QueueResidueLuckyBagMessage, error) {
	var messages []*QueueResidueLuckyBagMessage
	iter, err := Pb[TalkGroupResidueLuckyBagQueueCollection].NewIter(nil)
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	count := 0
	for iter.First(); iter.Valid() && iter.Key() != nil && count < limit; iter.Next() {
		value := string(iter.Value())

		var queueMessage QueueResidueLuckyBagMessage
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

// Delete reclaim lucky bag queue message data
func (cdb *ChatDB) DeleteResidueLuckyBagQueueMessage(pinId string) error {
	iter, err := Pb[TalkGroupResidueLuckyBagQueueCollection].NewIter(nil)
	if err != nil {
		return err
	}
	defer iter.Close()

	// Find queue message containing this pinId
	for iter.First(); iter.Valid(); iter.Next() {
		value := string(iter.Value())

		var queueMessage QueueResidueLuckyBagMessage
		err := json.Unmarshal([]byte(value), &queueMessage)
		if err != nil {
			continue
		}
		// Found matching pinId, delete this queue message
		if queueMessage.PinId == pinId {
			return Pb[TalkGroupResidueLuckyBagQueueCollection].Delete(iter.Key(), pebble.Sync)
		}
	}

	return nil
}

// Batch process queue messages (asynchronous update of group lists)
func (cdb *ChatDB) ProcessQueueMessages(groupDB *GroupDB, batchSize int) error {
	// Get pending messages
	messages, err := cdb.GetPendingQueueMessages(batchSize)
	if err != nil {
		return err
	}

	// Batch process messages
	for _, message := range messages {
		// Update group list for all members in the group
		err = cdb.UpdateGroupMembersContextList(message.GroupId, message.Chat, groupDB)
		if err != nil {
			// Processing failed, log error but don't delete queue message, can retry later
			log.Printf("Failed to process queue message for pinId %s: %v", message.PinId, err)
			continue
		}

		// Processing successful, delete queue message
		err = cdb.deleteQueueMessage(message.PinId)
		if err != nil {
			// Log error but don't affect main flow
			log.Printf("Failed to delete queue message for pinId %s: %v", message.PinId, err)
		}
	}

	return nil
}

// Start queue processing goroutine (needs to be called when application starts)
func (cdb *ChatDB) StartQueueProcessor(groupDB *GroupDB) {
	go func() {
		ticker := time.NewTicker(5 * time.Second) // Process every 5 seconds
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				// Batch process queue messages
				err := cdb.ProcessQueueMessages(groupDB, 100) // Process 100 messages each time
				if err != nil {
					// Log error
					continue
				}
			}
		}
	}()
}

// Main method to process Group Chat
func (cdb *ChatDB) ProcessGroupChatPin(pin *pin.PinInscription, tx interface{}) error {
	switch pin.Operation {
	case "create":
		path := pin.Path
		protocol := strings.Replace(path, "/protocols/", "", -1)

		var txData *wire.MsgTx
		if strings.ToLower(protocol) == strings.ToLower(protocols.MonitorSimpleGroupLuckyBag) ||
			strings.ToLower(protocol) == strings.ToLower(protocols.MonitorSimpleGroupOpenLuckyBag) ||
			strings.ToLower(protocol) == strings.ToLower(protocols.MonitorSimpleGroupResidueLuckyBag) {
			if tx != nil {
				// Determine tx type, if it's wire.MsgTx or *wire.MsgTx, convert to *wire.MsgTx
				switch txType := tx.(type) {
				case *wire.MsgTx:
					txData = txType
				case wire.MsgTx:
					txData = &txType
				default:
					// Other types, try direct conversion
					if msgTx, ok := tx.(*wire.MsgTx); ok {
						txData = msgTx
					} else if msgTx, ok := tx.(wire.MsgTx); ok {
						txData = &msgTx
					}
				}
			} else {

				// Need to change to non-man
				// var (
				// 	chain string = "btc"
				// 	txId  string = ""
				// )
				// chain = pin.ChainName
				// txId = pin.Id[:len(pin.Id)-2]
				// trst, err := man.ChainAdapter[chain].GetTransaction(txId)
				// if err != nil {
				// 	return err
				// }
				// txD := trst.(*btcutil.Tx)
				// if txD != nil {
				// 	txData = txD.MsgTx()
				// }
			}
		}

		if strings.ToLower(protocol) == strings.ToLower(protocols.MonitorSimpleGroupChat) {
			return cdb.processGroupChat(pin)
		} else if strings.ToLower(protocol) == strings.ToLower(protocols.MonitorSimpleFileGroupChat) {
			return cdb.processFileGroupChat(pin)
		} else if strings.ToLower(protocol) == strings.ToLower(protocols.MonitorSimpleGroupLuckyBag) {
			return cdb.processGroupLuckyBag(pin, txData)
		} else if strings.ToLower(protocol) == strings.ToLower(protocols.MonitorSimpleGroupOpenLuckyBag) {
			return cdb.processGroupOpenLuckyBag(pin, txData)
		} else if strings.ToLower(protocol) == strings.ToLower(protocols.MonitorSimpleGroupResidueLuckyBag) {
			return cdb.processGroupResidueLuckyBag(pin, txData)
		}
	default:
		return nil // Unknown operation type, skip
	}
	return nil
}

// Process group chat
func (cdb *ChatDB) processGroupChat(pin *pin.PinInscription) error {
	// Check if this PinId has already been saved
	existingChat, err := cdb.GetChatByPinId(pin.Id)
	if err == nil && existingChat != nil {
		if existingChat.BlockHeight != pin.GenesisHeight {
			existingChat.BlockHeight = pin.GenesisHeight
			err = cdb.SaveChat(existingChat)
			if err != nil {
				return err
			}
		}
		// Already exists, skip processing
		return nil
	}

	// Parse protocol data
	var simpleGroupChat protocols.SimpleGroupChat
	err = json.Unmarshal(pin.ContentBody, &simpleGroupChat)
	if err != nil {
		return err
	}

	// Create chat message model
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
		ChatType:    models.ChatTypeMsg,       // Default to message type
		InsideIndex: models.ChatInsideIndexIn, // Default to in state
		ReplyPin:    simpleGroupChat.ReplyPin,
		Timestamp:   pin.Timestamp,
		Chain:       pin.ChainName,
		BlockHeight: pin.GenesisHeight,
	}

	// Save chat message to TalkGroupChatPinCollection
	err = cdb.SaveChat(chat)
	if err != nil {
		return err
	}

	// Save timestamp index (decide which collection to save to based on user state)
	err = cdb.SaveChatTimestampWithState(chat)
	if err != nil {
		return err
	}

	// Enqueue message for asynchronous group list updates
	err = cdb.EnqueueChatMessage(chat)
	if err != nil {
		return err
	}

	return nil
}

// Check if user is in group
func (cdb *ChatDB) isUserInGroup(metaId, groupId string) (bool, error) {
	// Use TalkGroupMetaIdJoinCollection to check if user is in group
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

	// If list is empty, user is not in group
	if len(joinList.Items) == 0 {
		return false, nil
	}

	// Get latest join record (reverse order by timestamp, first is latest)
	latestItem := joinList.Items[0]
	return latestItem.GroupState == models.RoomStateIn, nil
}

// Get user's state in group
func (cdb *ChatDB) getUserGroupState(metaId, groupId string, chatTimestamp int64) (models.RoomState, error) {
	// Use TalkGroupMetaIdJoinCollection to get user state
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

	// If list is empty, user is not in group
	if len(joinList.Items) == 0 {
		return models.RoomStateOut, nil
	}

	// Sort by timestamp in ascending order to ensure correct time sequence
	// Simple bubble sort, ascending order by timestamp
	for i := 0; i < len(joinList.Items)-1; i++ {
		for j := 0; j < len(joinList.Items)-1-i; j++ {
			if joinList.Items[j].JoinTimestamp > joinList.Items[j+1].JoinTimestamp {
				joinList.Items[j], joinList.Items[j+1] = joinList.Items[j+1], joinList.Items[j]
			}
		}
	}

	var startItem, endItem *GroupMetaIdJoinItem
	// Iterate through join records to find the interval corresponding to chat message timestamp
	for i, item := range joinList.Items {
		if chatTimestamp >= item.JoinTimestamp {
			// Found the start record corresponding to chat message timestamp
			startItem = item

			// Find next record as end record
			if i+1 < len(joinList.Items) {
				endItem = joinList.Items[i+1]
			} else {
				// If no next record, this is the latest state
				endItem = nil
			}
		} else {
			// If current record timestamp is greater than chat timestamp, found the end of interval
			// At this point startItem should be the previous record
			if startItem != nil {
				endItem = item
			}
			break
		}
	}

	// If no corresponding interval found, chat message is before user joined group
	if startItem == nil {
		return models.RoomStateOut, nil
	}

	// Determine user's state at this time point based on startItem's state
	// If endItem exists and chat time exceeds endItem's time, state has changed
	if endItem != nil && chatTimestamp >= endItem.JoinTimestamp {
		// Chat time is after next state change, use next state
		return endItem.GroupState, nil
	} else {
		// Chat time is within current state interval, use current state
		return startItem.GroupState, nil
	}
}

// Save chat timestamp index (decide which collection to save to based on user state)
func (cdb *ChatDB) SaveChatTimestampWithState(chat *models.TalkGroupChatV3) error {
	// Get user's state in group
	groupState, err := cdb.getUserGroupState(chat.MetaId, chat.GroupId, chat.Timestamp)
	if err != nil {
		// If getting state fails, default to saving to normal collection
		return cdb.SaveChatTimestamp(chat)
	}

	// Construct timestamp index value: pinId_chatType_timestamp
	value := chat.PinId + "_" + strconv.FormatInt(int64(chat.ChatType), 10) + "_" + strconv.FormatInt(chat.Timestamp, 10)

	// Decide which collection to save to based on user state
	var collection string
	if groupState == models.RoomStateIn {
		// User is in group, save to normal collection
		collection = TalkGroupChatTimestampCollection
	} else {
		// User is not in group, save to invalid collection
		collection = TalkGroupChatTimestampOutCollection
	}

	// Use GroupId_Timestamp as primary key to support timestamp range queries
	key := []byte(chat.GroupId + "_" + strconv.FormatInt(chat.Timestamp, 10))
	if err = Pb[collection].Set(key, []byte(value), pebble.Sync); err != nil {
		return err
	}

	go dealGroupChatItem(chat)
	return nil
}

// Process file group chat
func (cdb *ChatDB) processFileGroupChat(pin *pin.PinInscription) error {
	// Check if this PinId has already been saved
	existingChat, err := cdb.GetChatByPinId(pin.Id)
	if err == nil && existingChat != nil {
		if existingChat.BlockHeight != pin.GenesisHeight {
			existingChat.BlockHeight = pin.GenesisHeight
			err = cdb.SaveChat(existingChat)
			if err != nil {
				return err
			}
		}
		// Already exists, skip processing
		return nil
	}

	// Parse protocol data
	var simpleFileGroupChat protocols.SimpleFileGroupChat
	err = json.Unmarshal(pin.ContentBody, &simpleFileGroupChat)
	if err != nil {
		return err
	}
	// Create chat message model
	chat := &models.TalkGroupChatV3{
		GroupId:     simpleFileGroupChat.GroupId,
		TxId:        pin.Id[:len(pin.Id)-2],
		PinId:       pin.Id,
		MetaId:      pin.CreateMetaId,
		Address:     pin.CreateAddress,
		Protocol:    pin.Path,
		Content:     simpleFileGroupChat.Attachment, // File attachment
		ContentType: simpleFileGroupChat.FileType,   // File type
		Encryption:  simpleFileGroupChat.Encrypt,
		ChatType:    models.ChatTypeFile,      // File type
		InsideIndex: models.ChatInsideIndexIn, // Default to in state
		ReplyPin:    simpleFileGroupChat.ReplyPin,
		Timestamp:   pin.Timestamp,
		BlockHeight: pin.GenesisHeight,
	}

	// Save chat message to TalkGroupChatPinCollection
	err = cdb.SaveChat(chat)
	if err != nil {
		return err
	}

	// Save timestamp index (decide which collection to save to based on user state)
	err = cdb.SaveChatTimestampWithState(chat)
	if err != nil {
		return err
	}

	// Enqueue message for asynchronous group list updates
	err = cdb.EnqueueChatMessage(chat)
	if err != nil {
		return err
	}

	return nil
}

// Process group lucky bag
func (cdb *ChatDB) processGroupLuckyBag(pin *pin.PinInscription, txData *wire.MsgTx) error {
	// Check if this PinId has already been saved
	existingRed, err := cdb.GetLuckyBagByPinId(pin.Id)
	if err == nil && existingRed != nil {
		// Already exists, skip processing
		if existingRed.BlockHeight != pin.GenesisHeight {
			existingRed.BlockHeight = pin.GenesisHeight
			err = cdb.SaveLuckyBag(existingRed)
			if err != nil {
				return err
			}
		}
		return nil
	}

	// Parse protocol data
	var simpleLuckyBag protocols.SimpleGroupLuckyBag
	err = json.Unmarshal(pin.ContentBody, &simpleLuckyBag)
	if err != nil {
		return err
	}

	// Convert payment list
	var (
		payList       []*models.ProInfoPayList
		luckyBagVouts []*models.LuckyBagOutput = make([]*models.LuckyBagOutput, 0)
	)
	for _, pay := range simpleLuckyBag.PayList {
		payList = append(payList, &models.ProInfoPayList{
			Amount:  toString(pay.Amount),
			Address: pay.Address,
			Index:   toInt64(pay.Index),
		})
	}
	if txData != nil {
		for i, vout := range txData.TxOut {
			luckyBagVouts = append(luckyBagVouts, &models.LuckyBagOutput{
				ScriptPubKey: hex.EncodeToString(vout.PkScript),
				Amount:       uint64(vout.Value),
				Address:      "",
				Index:        int64(i),
			})
		}
	}

	// Create lucky bag model
	redEnvelope := &models.TalkGroupLuckyBagV3{
		CommunityId:         "", // Need to get from group info
		GroupId:             simpleLuckyBag.GroupId,
		TxId:                pin.Id[:len(pin.Id)-2],
		PinId:               pin.Id,
		MetaId:              pin.CreateMetaId,
		Address:             pin.CreateAddress,
		Protocol:            pin.Path,
		SubId:               simpleLuckyBag.SubId,
		Code:                simpleLuckyBag.Code,
		CreateTimeStr:       formatInt64(simpleLuckyBag.CreateTime),
		Content:             simpleLuckyBag.Content,
		Img:                 simpleLuckyBag.Img,
		ImgType:             simpleLuckyBag.ImgType,
		Amount:              formatInt64(simpleLuckyBag.Amount),
		Count:               toString(simpleLuckyBag.Count),
		PayList:             payList,
		LuckyBagVouts:       luckyBagVouts,
		Type:                simpleLuckyBag.Type,
		RequireType:         toString(simpleLuckyBag.RequireType),
		RequireTickId:       simpleLuckyBag.RequireTickId,
		RequireCollectionId: simpleLuckyBag.RequireCollectionId,
		LimitAmount:         toUint64(simpleLuckyBag.LimitAmount),
		Timestamp:           pin.Timestamp,
		BlockHeight:         pin.GenesisHeight,
		Chain:               pin.ChainName,
	}

	//

	// Save lucky bag info
	err = cdb.SaveLuckyBag(redEnvelope)
	if err != nil {
		return err
	}

	// Create chat message model (for group chat display)
	chat := &models.TalkGroupChatV3{
		GroupId:     simpleLuckyBag.GroupId,
		TxId:        pin.Id[:len(pin.Id)-2],
		PinId:       pin.Id,
		MetaId:      pin.CreateMetaId,
		Address:     pin.CreateAddress,
		Protocol:    pin.Path,
		Content:     "[LuckyBag]:" + simpleLuckyBag.Content, // Lucky bag blessing message
		ContentType: "text/plain",
		Encryption:  "",
		ChatType:    models.ChatTypeLuckyBag,  // Lucky bag type
		InsideIndex: models.ChatInsideIndexIn, // Default to in state
		ReplyPin:    "",
		Timestamp:   pin.Timestamp,
		Chain:       pin.ChainName,
		BlockHeight: pin.GenesisHeight,
	}

	// Save chat message to TalkGroupChatPinCollection
	err = cdb.SaveChat(chat)
	if err != nil {
		return err
	}

	// Save timestamp index (decide which collection to save to based on user state)
	err = cdb.SaveChatTimestampWithState(chat)
	if err != nil {
		return err
	}

	// Enqueue message for asynchronous group list updates
	err = cdb.EnqueueChatMessage(chat)
	if err != nil {
		return err
	}

	return nil
}

// Process group grab lucky bag
func (cdb *ChatDB) processGroupOpenLuckyBag(pin *pin.PinInscription, txData *wire.MsgTx) error {
	// Check if this PinId has already been saved
	existingOpen, err := cdb.GetOpenLuckyBagByPinId(pin.Id)
	if err == nil && existingOpen != nil {
		// Already exists, skip processing
		if existingOpen.BlockHeight != pin.GenesisHeight {
			existingOpen.BlockHeight = pin.GenesisHeight
			err = cdb.SaveOpenLuckyBag(existingOpen)
			if err != nil {
				return err
			}
		}
		return nil
	}

	// Parse protocol data
	var simpleOpenLuckyBag protocols.SimpleGroupOpenLuckyBag
	err = json.Unmarshal(pin.ContentBody, &simpleOpenLuckyBag)
	if err != nil {
		return err
	}

	// Convert input transaction
	var (
		vins   []*models.TxIn
		amount int64  = 0
		index  uint32 = 0
	)

	if simpleOpenLuckyBag.Used != nil {
		if txData != nil {
			for _, vin := range txData.TxIn {
				if vin.PreviousOutPoint.Hash.String() == simpleOpenLuckyBag.LuckyBagTxId &&
					vin.PreviousOutPoint.Index == uint32(toInt64(simpleOpenLuckyBag.Used.Index)) {
					vins = append(vins, &models.TxIn{
						OutTxID: simpleOpenLuckyBag.LuckyBagTxId,
						Index:   uint64(vin.PreviousOutPoint.Index),
					})
					amount = toInt64(simpleOpenLuckyBag.Used.Amount)
					index = vin.PreviousOutPoint.Index
				}
			}
		}

	}

	// Create grab lucky bag model
	openLuckyBag := &models.TalkGroupOpenLuckyBagV3{
		CommunityId:         "", // Need to get from group info
		GroupId:             simpleOpenLuckyBag.GroupId,
		TxId:                pin.Id[:len(pin.Id)-2],
		PinId:               pin.Id,
		MetaId:              pin.CreateMetaId,
		Protocol:            pin.Path,
		SubId:               simpleOpenLuckyBag.SubId,
		Code:                simpleOpenLuckyBag.Code,
		CreateTimeStr:       formatInt64(simpleOpenLuckyBag.CreateTime),
		Address:             pin.CreateAddress,
		Index:               int64(index),
		Amount:              formatInt64(amount),
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
		GrabState:           models.GrabStateChain,
		GrabTxId:            pin.Id[:len(pin.Id)-2],
		GrabMsg:             "success",
	}

	// Save grab lucky bag info
	err = cdb.SaveOpenLuckyBag(openLuckyBag)
	if err != nil {
		return err
	}

	// Save grab lucky bag list record
	err = cdb.SaveOpenLuckyBagList(simpleOpenLuckyBag.LuckyBagPinId, pin.Id, simpleOpenLuckyBag.GroupId, pin.Timestamp, pin.CreateMetaId, pin.CreateAddress, int64(index))
	if err != nil {
		log.Printf("SaveOpenLuckyBagList err: %v", err)
		// Don't return error because main flow has succeeded
	}

	// Create chat message model (for group chat display)
	chat := &models.TalkGroupChatV3{
		GroupId:     simpleOpenLuckyBag.GroupId,
		TxId:        pin.Id[:len(pin.Id)-2],
		PinId:       pin.Id,
		MetaId:      pin.CreateMetaId,
		Address:     pin.CreateAddress,
		Protocol:    pin.Path,
		Content:     "[Grab LuckyBag]", // Can display based on actual amount
		ContentType: "text/plain",
		Encryption:  "",
		ChatType:    models.ChatTypeOpenLuckyBag, // Grab lucky bag type
		InsideIndex: models.ChatInsideIndexIn,    // Default to in state
		ReplyPin:    simpleOpenLuckyBag.LuckyBagPinId,
		Timestamp:   pin.Timestamp,
		Chain:       pin.ChainName,
		BlockHeight: pin.GenesisHeight,
	}

	// Save chat message to TalkGroupChatPinCollection
	err = cdb.SaveChat(chat)
	if err != nil {
		return err
	}

	// Save timestamp index (decide which collection to save to based on user state)
	err = cdb.SaveChatTimestampWithState(chat)
	if err != nil {
		return err
	}

	// Enqueue message for asynchronous group list updates
	err = cdb.EnqueueChatMessage(chat)
	if err != nil {
		return err
	}

	return nil
}

// Process group reclaim lucky bag
func (cdb *ChatDB) processGroupResidueLuckyBag(pin *pin.PinInscription, txData *wire.MsgTx) error {
	// Check if this PinId has already been saved
	existingResidue, err := cdb.GetResidueLuckyBagByLuckyBagPinId(pin.Id)
	if err == nil && existingResidue != nil {
		// Already exists, skip processing
		if existingResidue.BlockHeight != pin.GenesisHeight {
			existingResidue.BlockHeight = pin.GenesisHeight
			err = cdb.SaveResidueLuckyBag(existingResidue)
			if err != nil {
				return err
			}
		}
		return nil
	}

	// Parse protocol data
	var simpleResidueLuckyBag protocols.SimpleGroupResidueLuckyBag
	err = json.Unmarshal(pin.ContentBody, &simpleResidueLuckyBag)
	if err != nil {
		return err
	}

	// Convert used list
	var (
		usedList             []*models.ProInfoPayList = make([]*models.ProInfoPayList, 0)
		vins                 []*models.TxIn           = make([]*models.TxIn, 0)
		luckyBagOutIndexList []int64                  = make([]int64, 0)
	)
	for _, used := range simpleResidueLuckyBag.Used {
		usedList = append(usedList, &models.ProInfoPayList{
			Amount:  toString(used.Amount),
			Address: used.Address,
			Index:   toInt64(used.Index),
		})
		if txData != nil {
			for _, vin := range txData.TxIn {
				if vin.PreviousOutPoint.Hash.String() == simpleResidueLuckyBag.LuckyBagTxId &&
					vin.PreviousOutPoint.Index == uint32(toInt64(used.Index)) {
					vins = append(vins, &models.TxIn{
						OutTxID: simpleResidueLuckyBag.LuckyBagTxId,
						Index:   uint64(vin.PreviousOutPoint.Index),
					})
					luckyBagOutIndexList = append(luckyBagOutIndexList, int64(vin.PreviousOutPoint.Index))
				}
			}
		}
	}

	// Create reclaim lucky bag model
	residueLuckyBag := &models.TalkGroupResidueLuckyBagV3{
		CommunityId:         "", // Need to get from group info
		GroupId:             simpleResidueLuckyBag.GroupId,
		TxId:                pin.Id[:len(pin.Id)-2],
		PinId:               pin.Id,
		MetaId:              pin.CreateMetaId,
		Protocol:            pin.Path,
		SubId:               simpleResidueLuckyBag.SubId,
		Code:                simpleResidueLuckyBag.Code,
		CreateTimeStr:       formatInt64(simpleResidueLuckyBag.CreateTime),
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

	// Save reclaim lucky bag info
	err = cdb.SaveResidueLuckyBag(residueLuckyBag)
	if err != nil {
		return err
	}

	// Save reclaim lucky bag list record
	err = cdb.SaveResidueLuckyBagList(simpleResidueLuckyBag.LuckyBagPinId, pin.Id, simpleResidueLuckyBag.GroupId, pin.Timestamp, pin.CreateMetaId, pin.CreateAddress, luckyBagOutIndexList)
	if err != nil {
		log.Printf("SaveResidueLuckyBagList err: %v", err)
		// Don't return error because main flow has succeeded
	}

	return nil

	// // Create chat message model (for group chat display)
	// chat := &models.TalkGroupChatV3{
	// 	GroupId:     simpleResidueLuckyBag.GroupId,
	// 	TxId:        pin.Id[:len(pin.Id)-2],
	// 	PinId:       pin.Id,
	// 	MetaId:      pin.CreateMetaId,
	// 	Address:     pin.CreateAddress,
	// 	Protocol:    pin.Path,
	// 	Content:     "[Recycle LuckyBag]:" + simpleResidueLuckyBag.Code, // Can display based on actual situation
	// 	ContentType: "text/plain",
	// 	Encryption:  "",
	// 	ChatType:    models.ChatTypeRecycleLuckyBag, // Reclaim lucky bag type
	// 	InsideIndex: models.ChatInsideIndexIn,       // Default to in state
	// 	ReplyPin:    "",
	// 	Timestamp:   pin.Timestamp,
	// 	Chain:       pin.ChainName,
	// 	BlockHeight: pin.GenesisHeight,
	// }

	// // Save chat message to TalkGroupChatPinCollection
	// err = cdb.SaveChat(chat)
	// if err != nil {
	// 	return err
	// }

	// // Save timestamp index (decide which collection to save to based on user state)
	// err = cdb.SaveChatTimestampWithState(chat)
	// if err != nil {
	// 	return err
	// }

	// // Enqueue message for asynchronous group list updates
	// err = cdb.EnqueueChatMessage(chat)
	// if err != nil {
	// 	return err
	// }

	return nil
}

// Helper function: convert interface{} to string
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

// Helper function: format create time to avoid scientific notation
func formatInt64(v interface{}) string {
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		return val
	case int:
		return strconv.FormatInt(int64(val), 10)
	case int32:
		return strconv.FormatInt(int64(val), 10)
	case int64:
		return strconv.FormatInt(val, 10)
	case float32:
		return strconv.FormatInt(int64(val), 10)
	case float64:
		return strconv.FormatInt(int64(val), 10)
	default:
		return fmt.Sprintf("%v", val)
	}
}

// Helper function: convert interface{} to int64
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

// Helper function: convert interface{} to uint64
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

// Helper function: convert interface{} to bool
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
