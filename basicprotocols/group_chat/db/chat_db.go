package db

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"manindexer/basicprotocols/group_chat/models"
	"manindexer/basicprotocols/group_chat/protocols"
	"manindexer/basicprotocols/group_chat/service/cache_service"
	"manindexer/common"
	"manindexer/pin"
	"math"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/cockroachdb/pebble"
)

// Queue message item
type QueueChatMessage struct {
	PinId      string                  `json:"pinId"`      // Message PinId
	GroupId    string                  `json:"groupId"`    // Group ID
	ChannelId  string                  `json:"channelId"`  // Channel ID
	Chat       *models.TalkGroupChatV3 `json:"chat"`       // Chat message
	Timestamp  int64                   `json:"timestamp"`  // Enqueue timestamp
	RetryCount int                     `json:"retryCount"` // Retry count
	Status     string                  `json:"status"`     // Processing status: pending, processing, completed, failed
	IsResync   bool                    `json:"isResync"`   // Is resync
}

// Chat database operations
type ChatDB struct {
	pb                   *Pebble
	gdb                  *GroupDB // Reference to GroupDB for admin/block/whitelist checks
	luckyBagListMutexMap sync.Map
	processingMutex      sync.Mutex
	isProcessing         bool
}

// luckyBagMutexItem lock item
type luckyBagMutexItem struct {
	mutex       *sync.Mutex
	lastUsed    time.Time
	accessCount int64
}

func NewChatDB(pb *Pebble) *ChatDB {
	cdb := &ChatDB{pb: pb}

	go cdb.startCleanupGoroutine()

	return cdb
}

func (cdb *ChatDB) SetGdb(gdb *GroupDB) {
	cdb.gdb = gdb
}

// getLuckyBagMutex get or create lucky bag mutex
func (cdb *ChatDB) getLuckyBagMutex(luckyBagPinId string) *sync.Mutex {
	// try to get existing lock from sync.Map
	if value, exists := cdb.luckyBagListMutexMap.Load(luckyBagPinId); exists {
		if item, ok := value.(*luckyBagMutexItem); ok {
			// update access statistics
			item.lastUsed = time.Now()
			item.accessCount++
			return item.mutex
		}
	}

	// if not exists, create new lock item
	newItem := &luckyBagMutexItem{
		mutex:       &sync.Mutex{},
		lastUsed:    time.Now(),
		accessCount: 1,
	}

	// use LoadOrStore to ensure atomicity, avoid duplicate creation
	if value, loaded := cdb.luckyBagListMutexMap.LoadOrStore(luckyBagPinId, newItem); loaded {
		// if already exists, return existing lock and update statistics
		if item, ok := value.(*luckyBagMutexItem); ok {
			item.lastUsed = time.Now()
			item.accessCount++
			return item.mutex
		}
	}

	// return new created lock
	return newItem.mutex
}

// startCleanupGoroutine start cleanup goroutine
func (cdb *ChatDB) startCleanupGoroutine() {
	ticker := time.NewTicker(5 * time.Minute) // every 5 minutes
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			cdb.cleanupUnusedLocks()
		}
	}
}

// cleanupUnusedLocks cleanup unused locks
func (cdb *ChatDB) cleanupUnusedLocks() {
	now := time.Now()
	cleanupThreshold := 30 * time.Minute // 30 minutes not used to clean up

	var keysToDelete []string

	// traverse all locks, find locks to clean up
	cdb.luckyBagListMutexMap.Range(func(key, value interface{}) bool {
		if item, ok := value.(*luckyBagMutexItem); ok {
			// check if it exceeds the cleanup threshold
			if now.Sub(item.lastUsed) > cleanupThreshold {
				keysToDelete = append(keysToDelete, key.(string))
			}
		}
		return true
	})

	// delete unused locks
	for _, key := range keysToDelete {
		cdb.luckyBagListMutexMap.Delete(key)
	}

	if len(keysToDelete) > 0 {
		log.Printf("Cleaned up %d unused lucky bag locks", len(keysToDelete))
	}
}

// CleanupLuckyBagLock manually clean up specific lucky bag locks
func (cdb *ChatDB) CleanupLuckyBagLock(luckyBagPinId string) bool {
	// check if the lock is being used
	if value, exists := cdb.luckyBagListMutexMap.Load(luckyBagPinId); exists {
		if item, ok := value.(*luckyBagMutexItem); ok {
			// if the lock is being used (a goroutine holds the lock), it cannot be deleted
			// here we use a simple heuristic: if there has been access in the last 5 minutes, it will not be deleted
			if time.Since(item.lastUsed) < 5*time.Minute {
				return false // the lock is still being used
			}
		}
	}

	// delete lock
	cdb.luckyBagListMutexMap.Delete(luckyBagPinId)
	return true
}

// GetLuckyBagLockStats get lock statistics
func (cdb *ChatDB) GetLuckyBagLockStats() map[string]interface{} {
	stats := make(map[string]interface{})
	totalLocks := 0
	activeLocks := 0
	now := time.Now()

	cdb.luckyBagListMutexMap.Range(func(key, value interface{}) bool {
		totalLocks++
		if item, ok := value.(*luckyBagMutexItem); ok {
			// if there has been access in the last 5 minutes, it is considered active
			if now.Sub(item.lastUsed) < 5*time.Minute {
				activeLocks++
			}
		}
		return true
	})

	stats["totalLocks"] = totalLocks
	stats["activeLocks"] = activeLocks
	stats["inactiveLocks"] = totalLocks - activeLocks

	return stats
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
func (cdb *ChatDB) SaveChatTimestamp(chat *models.TalkGroupChatV3, isResync bool) error {
	// Construct timestamp index value: pinId_chatType_timestamp
	value := chat.PinId + "_" + strconv.FormatInt(int64(chat.ChatType), 10) + "_" + strconv.FormatInt(chat.Timestamp, 10)

	// Use GroupId_Timestamp as primary key to support timestamp range queries
	key := []byte(chat.GroupId + "_" + strconv.FormatInt(chat.Timestamp, 10))
	if err := Pb[TalkGroupChatTimestampCollection].Set(key, []byte(value), pebble.Sync); err != nil {
		return err
	}

	// Use GroupId_Timestamp_PinId as primary key to support timestamp range queries
	cdb.saveChatTimestamp2(chat, isResync)
	return nil
}

func (cdb *ChatDB) saveChatTimestamp2(chat *models.TalkGroupChatV3, isResync bool) error {
	return cdb.saveChatTimestamp2WithCollection(chat, TalkGroupChatTimestamp2Collection, isResync)
}

// saveChatTimestamp2WithCollection saves chat timestamp to a specific collection with timestamp + random number format
// This method handles the new key format: groupId_timestamp+number(6) for collection2
// Example: groupId_1755500889000001 (timestamp 1755500889 + random 000001)
func (cdb *ChatDB) saveChatTimestamp2WithCollection(chat *models.TalkGroupChatV3, collection string, isResync bool) error {
	//if isResync, frist get chat from database
	if isResync {
		isDuplicate, err := cdb.CheckDuplicatePinIdInTimeRange(chat, collection)
		if err != nil {
			return err
		}
		if isDuplicate {
			//already has index, skip
			return nil
		}
	}

	// Generate a 6-digit random number for uniqueness
	randomNum := generateRandomNumber(6)

	// Construct key: groupId_timestamp+number(6)
	// Example: timestamp 1755500889 + random 000001 = 1755500889000001
	key := chat.GroupId + "_" + strconv.FormatInt(chat.Timestamp, 10) + randomNum

	// Construct value: pinId_chatType_timestamp_number
	value := chat.PinId + "_" + strconv.FormatInt(int64(chat.ChatType), 10) + "_" + strconv.FormatInt(chat.Timestamp, 10) + "_" + randomNum

	fmt.Printf("[CHAT_DB]saveChatTimestamp2WithCollection[%s][%s]: %s, %s\n", collection, chat.Chain, key, value)

	err := Pb[collection].Set([]byte(key), []byte(value), pebble.Sync)
	if err != nil {
		fmt.Printf("[CHAT_DB]saveChatTimestamp2WithCollection[%s][%s] key:%s, value:%s, error: %s\n", collection, chat.Chain, key, value, err)
		return err
	}

	return nil
}

// generateRandomNumber generates a random number with specified digits
func generateRandomNumber(digits int) string {
	// Generate a random number between 0 and 10^digits - 1
	max := int64(math.Pow10(digits)) - 1
	randomNum := rand.Int63n(max + 1)

	// Format with leading zeros to ensure consistent length
	return fmt.Sprintf("%0*d", digits, randomNum)
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

// Get chat message list by group ID and end timestamp (reverse order, pagination based on timestamp)
func (cdb *ChatDB) GetChatsByGroupIdAndTimestampRange(groupId string, endTimestamp int64, size int64) ([]*models.TalkGroupChatV3, error) {
	var chats []*models.TalkGroupChatV3
	iter, err := Pb[TalkGroupChatTimestampCollection].NewIter(nil)
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	// Construct query start key: groupId_endTimestamp
	startKey := []byte(groupId + "_" + strconv.FormatInt(endTimestamp, 10))

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

// Get chat message list by group ID and end timestamp using TalkGroupChatTimestamp2Collection
// This function handles the new key format: groupId_timestamp+number(6)
// Example: groupId_1755500889000001 (timestamp 1755500889 + random 000001)
// which provides better support for multiple messages at the same timestamp
func (cdb *ChatDB) GetChatsByGroupIdAndEndTimestampRange2(groupId string, endTimestamp int64, size int64) ([]*models.TalkGroupChatV3, int64, error) {
	var chats []*models.TalkGroupChatV3
	iter, err := Pb[TalkGroupChatTimestamp2Collection].NewIter(nil)
	if err != nil {
		return nil, 0, err
	}
	defer iter.Close()

	nextTimestamp := int64(0)

	// Construct query start key: groupId_endTimestamp
	// Since key format is now groupId_timestamp+number(6), we can use proper range scanning
	// Example: groupId_1755500889000001 (timestamp 1755500889 + random 000001)
	startKey := []byte(groupId + "_" + strconv.FormatInt(endTimestamp, 10))

	// Start reverse iteration from specified timestamp (latest messages first)
	// Use SeekLT to find the last key that is less than our startKey
	for iter.SeekLT(startKey); iter.Valid() && iter.Key() != nil; iter.Prev() {
		key := string(iter.Key())

		// Check if it belongs to the specified group
		if !strings.HasPrefix(key, groupId+"_") {
			continue
		}

		// Parse key to extract timestamp
		// Key format: groupId_timestamp+number(6)
		keyParts := strings.Split(key, "_")
		if len(keyParts) < 2 {
			continue
		}

		// Extract timestamp from key (remove the last 6 digits which is the random number)
		timestampStr := keyParts[1]
		timestampKey := timestampStr
		timestampKeyInt, _ := strconv.ParseInt(timestampKey, 10, 64)
		if len(timestampStr) > 6 {
			timestampStr = timestampStr[:len(timestampStr)-6]
		}

		timestamp, err := strconv.ParseInt(timestampStr, 10, 64)
		if err != nil {
			continue
		}

		// Skip messages before our end timestamp (since we're going backwards)
		if timestamp > endTimestamp {
			continue
		}

		// Parse value: pinId_chatType_timestamp_number
		value := string(iter.Value())
		valueParts := strings.Split(value, "_")
		if len(valueParts) < 1 {
			continue
		}

		pinId := valueParts[0]
		// fmt.Printf("[CHAT_DB]timestampStr: %s, pinId: %s\n", timestampStr, pinId)

		// Get complete chat message
		chat, err := cdb.GetChatByPinId(pinId)
		if err != nil || chat == nil {
			continue
		}

		// if chat.Timestamp > nextTimestamp {
		if nextTimestamp == 0 || timestampKeyInt < nextTimestamp {
			// nextTimestamp = chat.Timestamp
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

// GetChatsByGroupIdAndTimestampRange3 is a test version that uses IterOptions to limit the range
// This function handles the key format: groupId_timestamp+number(6)
// Example: groupId_1755500889000001 (timestamp 1755500889 + random 000001)
func (cdb *ChatDB) GetChatsByGroupIdAndEndTimestampRange3(groupId string, endTimestamp int64, size int64) ([]*models.TalkGroupChatV3, int64, error) {
	var chats []*models.TalkGroupChatV3

	// Create iter options to limit the range to only keys for this group
	iterOptions := &pebble.IterOptions{
		LowerBound: []byte(groupId + "_"),
		UpperBound: []byte(groupId + "_" + string([]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff})),
	}

	iter, err := Pb[TalkGroupChatTimestamp2Collection].NewIter(iterOptions)
	if err != nil {
		return nil, 0, err
	}
	defer iter.Close()

	nextTimestamp := int64(0)

	// Construct query start key: groupId_endTimestamp
	// Since key format is now groupId_timestamp+number(6), we can use proper range scanning
	// Example: groupId_1755500889000001 (timestamp 1755500889 + random 000001)
	startKey := []byte(groupId + "_" + strconv.FormatInt(endTimestamp, 10))

	// Start reverse iteration from specified timestamp (latest messages first)
	// Use SeekLT to find the last key that is less than our startKey
	for iter.SeekLT(startKey); iter.Valid() && iter.Key() != nil; iter.Prev() {
		key := string(iter.Key())
		fmt.Printf("[CHAT_DB] GetChatsByGroupIdAndTimestampRange3 key: %s\n", key)

		// Parse key to extract timestamp
		// Key format: groupId_timestamp+number(6)
		keyParts := strings.Split(key, "_")
		if len(keyParts) < 2 {
			continue
		}

		// Extract timestamp from key (remove the last 6 digits which is the random number)
		timestampStr := keyParts[1]
		timestampKey := timestampStr
		timestampKeyInt, _ := strconv.ParseInt(timestampKey, 10, 64)

		// Skip messages after our end timestamp (since we're going backwards)
		if timestampKeyInt > endTimestamp {
			continue
		}

		// Parse value: pinId_chatType_timestamp_number
		value := string(iter.Value())
		valueParts := strings.Split(value, "_")
		if len(valueParts) < 1 {
			continue
		}

		pinId := valueParts[0]
		// fmt.Printf("[CHAT_DB] GetChatsByGroupIdAndTimestampRange3 timestampStr: %s, pinId: %s\n", timestampStr, pinId)

		// Get complete chat message
		chat, err := cdb.GetChatByPinId(pinId)
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
// note: this method has concurrency issues, it is recommended to use SaveOpenLuckyBagListAtomic
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

// SaveOpenLuckyBagListAtomic save grab lucky bag list record atomically
// use mutex + optimistic lock mechanism to avoid concurrency problems
func (cdb *ChatDB) SaveOpenLuckyBagListAtomic(luckyBagPinId string, openPinId string, groupId string, timestamp int64, createMetaId string, createAddress string, luckyBagOutIndex int64) error {
	maxRetries := 3
	for attempt := 0; attempt < maxRetries; attempt++ {
		// use specific lucky bag mutex to ensure only one goroutine operates on the same lucky bag list at the same time
		cdb.getLuckyBagMutex(luckyBagPinId).Lock()
		t := time.Now().UnixMilli()
		fmt.Printf("[SaveOpenLuckyBagListAtomic]: %s, address: %s, index: %d [Lock] %d\n", luckyBagPinId, createAddress, luckyBagOutIndex, t)

		// Get existing list
		list, err := cdb.GetOpenLuckyBagList(luckyBagPinId)
		if err != nil {
			cdb.getLuckyBagMutex(luckyBagPinId).Unlock()
			fmt.Printf("[SaveOpenLuckyBagListAtomic]: %s, address: %s, index: %d [UnLock]FindError: %v [%d]\n", luckyBagPinId, createAddress, luckyBagOutIndex, err, time.Now().UnixMilli()-t)
			return err
		}

		// Check if already exists
		found := false
		for _, item := range list.Items {
			if item.OpenPinId == openPinId {
				found = true
				break
			}
		}

		if found {
			// Already exists, no need to save
			cdb.getLuckyBagMutex(luckyBagPinId).Unlock()
			fmt.Printf("[SaveOpenLuckyBagListAtomic]: %s, address: %s, index: %d [UnLock]Found [%d]\n", luckyBagPinId, createAddress, luckyBagOutIndex, time.Now().UnixMilli()-t)
			return nil
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

		list.Items = append(list.Items, newItem)

		// Save updated list
		data, err := json.Marshal(list)
		if err != nil {
			cdb.getLuckyBagMutex(luckyBagPinId).Unlock()
			fmt.Printf("[SaveOpenLuckyBagListAtomic]: %s, address: %s, index: %d [UnLock]JsonError: %v [%d]\n", luckyBagPinId, createAddress, luckyBagOutIndex, err, time.Now().UnixMilli()-t)
			return err
		}

		key := []byte(luckyBagPinId)
		err = Pb[TalkGroupOpenLuckyBagListCollection].Set(key, data, pebble.Sync)
		if err != nil {
			cdb.getLuckyBagMutex(luckyBagPinId).Unlock()
			fmt.Printf("[SaveOpenLuckyBagListAtomic]: %s, address: %s, index: %d [UnLock]SaveError: %v [%d]\n", luckyBagPinId, createAddress, luckyBagOutIndex, err, time.Now().UnixMilli()-t)
			return err
		}

		// release lock, allow other goroutines to operate
		cdb.getLuckyBagMutex(luckyBagPinId).Unlock()
		fmt.Printf("[SaveOpenLuckyBagListAtomic]: %s, address: %s, index: %d [UnLock]Save [%d]\n", luckyBagPinId, createAddress, luckyBagOutIndex, time.Now().UnixMilli()-t)

		// optimistic lock verification: re-read and verify that our record is properly saved
		verifyList, err := cdb.GetOpenLuckyBagList(luckyBagPinId)
		if err != nil {
			return err
		}

		// check if our record exists
		verified := false
		for _, item := range verifyList.Items {
			if item.OpenPinId == openPinId {
				verified = true
				break
			}
		}

		if verified {
			// save successfully
			return nil
		}

		// save failed, possibly due to concurrency conflicts, retry
		if attempt < maxRetries-1 {
			// wait briefly and retry
			time.Sleep(time.Duration(attempt+1) * 10 * time.Millisecond)
			continue
		}
	}

	return fmt.Errorf("failed to save open lucky bag list after %d attempts", maxRetries)
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

// RemoveOpenLuckyBagListItem removes a specific item from the open lucky bag list
func (cdb *ChatDB) RemoveOpenLuckyBagListItem(luckyBagPinId string, openPinId string) error {
	// Get existing list
	list, err := cdb.GetOpenLuckyBagList(luckyBagPinId)
	if err != nil {
		return err
	}

	// Find and remove the specific item
	var newItems []*models.OpenLuckyBagListItem
	for _, item := range list.Items {
		if item.OpenPinId != openPinId {
			newItems = append(newItems, item)
		}
	}

	// Update the list
	list.Items = newItems

	// Save updated list
	data, err := json.Marshal(list)
	if err != nil {
		return err
	}

	key := []byte(luckyBagPinId)
	return Pb[TalkGroupOpenLuckyBagListCollection].Set(key, data, pebble.Sync)
}

// Save reclaim lucky bag list record
// note: this method has concurrency issues, it is recommended to use SaveResidueLuckyBagListAtomic
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

// SaveResidueLuckyBagListAtomic save reclaim lucky bag list record atomically
// use mutex + optimistic lock mechanism to avoid concurrency problems
func (cdb *ChatDB) SaveResidueLuckyBagListAtomic(luckyBagPinId string, residuePinId string, groupId string, timestamp int64, createMetaId string, createAddress string, luckyBagOutIndexList []int64) error {
	maxRetries := 3
	for attempt := 0; attempt < maxRetries; attempt++ {
		// use specific lucky bag mutex to ensure only one goroutine operates on the same lucky bag list at the same time
		cdb.getLuckyBagMutex(luckyBagPinId).Lock()

		// Get existing list
		list, err := cdb.GetResidueLuckyBagList(luckyBagPinId)
		if err != nil {
			cdb.getLuckyBagMutex(luckyBagPinId).Unlock()
			return err
		}

		// Check if already exists
		found := false
		for _, item := range list.Items {
			if item.ResiduePinId == residuePinId {
				found = true
				break
			}
		}

		if found {
			// Already exists, no need to save
			cdb.getLuckyBagMutex(luckyBagPinId).Unlock()
			return nil
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

		list.Items = append(list.Items, newItem)

		// Save updated list
		data, err := json.Marshal(list)
		if err != nil {
			cdb.getLuckyBagMutex(luckyBagPinId).Unlock()
			return err
		}

		key := []byte(luckyBagPinId)
		err = Pb[TalkGroupResidueLuckyBagListCollection].Set(key, data, pebble.Sync)
		if err != nil {
			cdb.getLuckyBagMutex(luckyBagPinId).Unlock()
			return err
		}

		// release lock, allow other goroutines to operate
		cdb.getLuckyBagMutex(luckyBagPinId).Unlock()

		// optimistic lock verification: re-read and verify that our record is properly saved
		verifyList, err := cdb.GetResidueLuckyBagList(luckyBagPinId)
		if err != nil {
			return err
		}

		// check if our record exists
		verified := false
		for _, item := range verifyList.Items {
			if item.ResiduePinId == residuePinId {
				verified = true
				break
			}
		}

		if verified {
			// save successfully
			return nil
		}

		// save failed, possibly due to concurrency conflicts, retry
		if attempt < maxRetries-1 {
			// wait briefly and retry
			time.Sleep(time.Duration(attempt+1) * 10 * time.Millisecond)
			continue
		}
	}

	return fmt.Errorf("failed to save residue lucky bag list after %d attempts", maxRetries)
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
func (cdb *ChatDB) UpdateGroupMembersContextList(groupId string, chat *models.TalkGroupChatV3, groupDB *GroupDB, isResync bool) error {

	//if isResync, frist get chat from database
	if isResync {
		dbChat, err := cdb.GetChatByPinId(chat.PinId)
		if err != nil && err != pebble.ErrNotFound {
			return err
		}
		if dbChat != nil && dbChat.Index != -1 {
			//already has index, skip
			return nil
		}
	}

	// Update group latest chat record
	err := cdb.updateGroupLatestChat(groupId, chat)
	if err != nil {
		return err
	}

	// Get all members of the group
	// members, err := groupDB.GetGroupMembers(groupId)
	members, err := groupDB.GetGroupMembersFromList(groupId)
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

// Update group channel latest chat record
func (cdb *ChatDB) updateGroupChannelLatestChat(channelId string, chat *models.TalkGroupChatV3, isResync bool) error {
	//if isResync, frist get chat from database
	if isResync {
		dbChat, err := cdb.GetChatByPinId(chat.PinId)
		if err != nil && err != pebble.ErrNotFound {
			return err
		}
		if dbChat != nil && dbChat.Index != -1 {
			//already has index, skip
			return nil
		}
	}

	// First get existing latest chat record
	existingLatestChat, err := cdb.GetGroupChannelLatestChat(channelId)
	if err != nil {
		return err
	}

	// Create new latest chat record
	newLatestChat := &models.TalkGroupChannelLatestChat{
		ChannelId:        channelId,
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
		key := []byte(channelId)
		return Pb[TalkGroupChannelLatestChatCollection].Set(key, data, pebble.Sync)
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
		key := []byte(channelId)
		return Pb[TalkGroupChannelLatestChatCollection].Set(key, data, pebble.Sync)
	}

	// No update needed, return directly
	return nil
}

// Get group channel latest chat record
func (cdb *ChatDB) GetGroupChannelLatestChat(channelId string) (*models.TalkGroupChannelLatestChat, error) {
	key := []byte(channelId)
	value, closer, err := Pb[TalkGroupChannelLatestChatCollection].Get(key)
	if err != nil {
		if err == pebble.ErrNotFound {
			return nil, nil
		}
		return nil, err
	}
	defer closer.Close()

	var latestChat models.TalkGroupChannelLatestChat
	err = json.Unmarshal(value, &latestChat)
	if err != nil {
		return nil, err
	}

	return &latestChat, nil
}

// Update single member's group list
func (cdb *ChatDB) updateSingleMemberContextList(metaId, groupId string, chat *models.TalkGroupChatV3) error {
	// Get mutex for this MetaId to prevent concurrent updates
	mutex := GetMetaIdMutex(metaId)
	mutex.Lock()
	defer mutex.Unlock()

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
			found = true
			if item.LastMessagePinId != newItem.LastMessagePinId && item.Timestamp < newItem.Timestamp {
				// Update existing item
				contextList.Items[i] = newItem
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
func (cdb *ChatDB) EnqueueChatMessage(chat *models.TalkGroupChatV3, isResync bool) error {
	queueMessage := &QueueChatMessage{
		PinId:      chat.PinId,
		GroupId:    chat.GroupId,
		Chat:       chat,
		Timestamp:  time.Now().Unix(),
		RetryCount: 0,
		Status:     "pending",
		IsResync:   isResync,
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

// Get total count of grab lucky bag queue collection
func (cdb *ChatDB) GetOpenLuckyBagQueueCount() (int64, error) {
	iter, err := Pb[TalkGroupOpenLuckyBagQueueCollection].NewIter(nil)
	if err != nil {
		return 0, err
	}
	defer iter.Close()

	count := 0
	for iter.First(); iter.Valid(); iter.Next() {
		count++
	}

	return int64(count), nil
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

// Update grab lucky bag queue message data
func (cdb *ChatDB) UpdateOpenLuckyBagQueueMessage(pinId string, retryCount int64, status string) error {
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

		// Found matching pinId, update this queue message
		if queueMessage.PinId == pinId {
			// Update retry count and status
			queueMessage.RetryCount = int(retryCount)
			if status != "" {
				queueMessage.Status = status
			}

			// Update the OpenLuckyBag object's RetryCount as well
			if queueMessage.OpenLuckyBag != nil {
				queueMessage.OpenLuckyBag.RetryCount = retryCount
			}

			// Marshal updated message
			data, err := json.Marshal(queueMessage)
			if err != nil {
				return err
			}

			// Update in database
			return Pb[TalkGroupOpenLuckyBagQueueCollection].Set(iter.Key(), data, pebble.Sync)
		}
	}

	return fmt.Errorf("queue message with pinId %s not found", pinId)
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

// Update reclaim lucky bag queue message data
func (cdb *ChatDB) UpdateResidueLuckyBagQueueMessage(pinId string, retryCount int64, status string) error {
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

		// Found matching pinId, update this queue message
		if queueMessage.PinId == pinId {
			// Update retry count and status
			queueMessage.RetryCount = int(retryCount)
			if status != "" {
				queueMessage.Status = status
			}

			// Update the ResidueLuckyBag object's RetryCount as well
			if queueMessage.ResidueLuckyBag != nil {
				queueMessage.ResidueLuckyBag.RetryCount = retryCount
			}

			// Marshal updated message
			data, err := json.Marshal(queueMessage)
			if err != nil {
				return err
			}

			// Update in database
			return Pb[TalkGroupResidueLuckyBagQueueCollection].Set(iter.Key(), data, pebble.Sync)
		}
	}

	return fmt.Errorf("queue message with pinId %s not found", pinId)
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
		hasError := false

		if message.ChannelId != "" {
			err = cdb.updateGroupChannelLatestChat(message.ChannelId, message.Chat, message.IsResync)
			if err != nil {
				log.Printf("Failed to update group channel latest chat for pinId %s: %v", message.PinId, err)
				hasError = true
			}

			// Update channel chat index
			err = cdb.UpdateChannelChatIndex(message.Chat, message.IsResync)
			if err != nil {
				log.Printf("Failed to update channel chat index for pinId %s: %v", message.PinId, err)
				hasError = true
			}

		} else {
			// Update group list for all members in the group
			err = cdb.UpdateGroupMembersContextList(message.GroupId, message.Chat, groupDB, message.IsResync)
			if err != nil {
				log.Printf("Failed to update group members context list for pinId %s: %v", message.PinId, err)
				hasError = true
			}

			// Update chat index
			err = cdb.UpdateChatIndex(message.Chat, message.IsResync)
			if err != nil {
				log.Printf("Failed to update chat index for pinId %s: %v", message.PinId, err)
				hasError = true
			}
		}

		// Only delete queue message if both operations succeeded
		if !hasError {
			err = cdb.deleteQueueMessage(message.PinId)
			if err != nil {
				// Log error but don't affect main flow
				log.Printf("Failed to delete queue message for pinId %s: %v", message.PinId, err)
			}
		}

	}

	return nil
}

// Start queue processing goroutine (needs to be called when application starts)
func (cdb *ChatDB) StartQueueProcessor(groupDB *GroupDB) {
	go func() {
		ticker := time.NewTicker(2 * time.Second) // Process every 2 seconds
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:

				if GlobalIsStop {
					log.Printf("[ChatDB] Queue processor is stopped, skipping this cycle")
					continue
				}

				// Check if already processing
				cdb.processingMutex.Lock()
				if cdb.isProcessing {
					cdb.processingMutex.Unlock()
					log.Printf("[ChatDB] Queue processor is already running, skipping this cycle")
					continue
				}
				cdb.isProcessing = true
				cdb.processingMutex.Unlock()

				// Batch process queue messages
				err := cdb.ProcessQueueMessages(groupDB, 100) // Process 100 messages each time
				if err != nil {
					log.Printf("[ChatDB] Error processing queue messages: %v", err)
				}

				// Mark processing as complete
				cdb.processingMutex.Lock()
				cdb.isProcessing = false
				cdb.processingMutex.Unlock()
			}
		}
	}()
}

// Main method to process Group Chat
func (cdb *ChatDB) ProcessGroupChatPin(pin *pin.PinInscription, tx interface{}, isResync bool) error {
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
			return cdb.processGroupChat(pin, isResync)
		} else if strings.ToLower(protocol) == strings.ToLower(protocols.MonitorSimpleFileGroupChat) {
			return cdb.processFileGroupChat(pin, isResync)
		} else if strings.ToLower(protocol) == strings.ToLower(protocols.MonitorSimpleGroupLuckyBag) {
			return cdb.processGroupLuckyBag(pin, txData, isResync)
		} else if strings.ToLower(protocol) == strings.ToLower(protocols.MonitorSimpleGroupOpenLuckyBag) {
			return cdb.processGroupOpenLuckyBag(pin, txData, isResync)
		} else if strings.ToLower(protocol) == strings.ToLower(protocols.MonitorSimpleGroupResidueLuckyBag) {
			return cdb.processGroupResidueLuckyBag(pin, txData, isResync)
		}
	default:
		return nil // Unknown operation type, skip
	}
	return nil
}

// Process group chat
func (cdb *ChatDB) processGroupChat(pin *pin.PinInscription, isResync bool) error {
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

	isSynced, err := IsPinSynced(pin.Id)
	if err != nil {
		return err
	}
	if isSynced {
		// Already synced, skip processing
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
		ChannelId:   simpleGroupChat.ChannelId,
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
		Version:     pin.Version,
		Chain:       pin.ChainName,
		BlockHeight: pin.GenesisHeight,
		Index:       -1,
	}

	// Save chat message to TalkGroupChatPinCollection
	err = cdb.SaveChat(chat)
	if err != nil {
		return err
	}

	// Process chat timestamp and enqueue based on whether it's a channel chat or group chat
	if chat.ChannelId != "" {
		// This is a channel chat
		err = cdb.processChannelChatTimestampAndEnqueue(chat, isResync)
	} else {
		// This is a group chat
		err = cdb.processChatTimestampAndEnqueue(chat, isResync)
	}
	if err != nil {
		return err
	}

	// Mark pin as synced
	err = MarkPinAsSynced(pin.Id, true)
	if err != nil {
		return err
	}

	return nil
}

// Check if user is in group
func (cdb *ChatDB) IsUserInGroup(metaId, groupId string) (bool, error) {
	// Use TalkGroupPersonCollection to check if user is in group
	// key: groupId_metaId
	key := []byte(groupId + "_" + metaId)
	value, closer, err := Pb[TalkGroupPersonCollection].Get(key)
	if err != nil {
		if err == pebble.ErrNotFound {
			return false, nil
		}
		return false, err
	}
	defer closer.Close()

	var person models.TalkGroupPerson
	err = json.Unmarshal(value, &person)
	if err != nil {
		return false, err
	}

	// Check if user is in the group based on GroupState
	return person.GroupState == models.RoomStateIn, nil
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

// Check if chat message should be placed in group (not in OutCollection)
// Returns true if message should be in group, false if should be in OutCollection
// Also returns the reason if message should be in OutCollection
func (cdb *ChatDB) shouldPlaceMessageInGroup(chat *models.TalkGroupChatV3) (bool, string, error) {
	// Get group info to check RoomJoinType
	// groupInfo, err := cdb.getGroupInfo(chat.GroupId)
	// if err != nil {
	// 	// If getting group info fails, default to placing in group
	// 	return true, "", nil
	// }

	// // Check if group is in launch-mode (RoomJoinType = "4")
	// if groupInfo != nil && groupInfo.RoomJoinType == "4" {
	// 	if cdb.gdb != nil {
	// 		// Check if user is admin
	// 		isAdmin, err := cdb.gdb.IsUserAdmin(chat.GroupId, chat.MetaId, chat.Timestamp)
	// 		if err != nil {
	// 			// If getting admin status fails, default to placing in group
	// 			return true, "", nil
	// 		}
	// 		if isAdmin {
	// 			return true, "", nil
	// 		}

	// 		// Check if group has whitelist and user is not whitelisted
	// 		isWhitelisted, err := cdb.gdb.IsUserWhitelist(chat.GroupId, chat.MetaId, chat.Timestamp)
	// 		if err != nil {
	// 			// If getting whitelist status fails, default to placing in group
	// 			return true, "", nil
	// 		}
	// 		if isWhitelisted {
	// 			return true, "", nil
	// 		}
	// 	}
	// 	// Group is in launch-mode, message should be placed in OutCollection
	// 	reason := "group is in launch-mode, messages are not displayed in group"
	// 	return false, reason, nil
	// }

	// Check if user is blocked in the group
	if cdb.gdb != nil {
		isBlocked, err := cdb.gdb.IsUserBlock(chat.GroupId, chat.MetaId, chat.Timestamp)
		if err != nil {
			// If getting block status fails, default to placing in group
			return true, "", nil
		}
		if isBlocked {
			// User is blocked, message should be placed in OutCollection
			reason := "user is blocked in the group"
			return false, reason, nil
		}
	}

	// Group is not in launch-mode, check user's state in group
	groupState, err := cdb.getUserGroupState(chat.MetaId, chat.GroupId, chat.Timestamp)
	if err != nil {
		// If getting state fails, default to placing in group
		return true, "", nil
	}

	// Check if user is in group
	if groupState == models.RoomStateIn {
		// User is in group, message should be placed in group
		return true, "", nil
	} else {
		// User is not in group, message should be placed in OutCollection
		reason := fmt.Sprintf("user is not in group, groupState: %d", groupState)
		return false, reason, nil
	}
}

// Check if chat message should be placed in channel (not in OutCollection)
// Returns true if message should be in channel, false if should be in OutCollection
// Also returns the reason if message should be in OutCollection
func (cdb *ChatDB) shouldPlaceMessageInChannel(chat *models.TalkGroupChatV3) (bool, string, error) {
	// Get channel info to check ChannelType
	channelInfo, err := cdb.GetChannelInfo(chat.ChannelId)
	if err != nil {
		// If getting group info fails, default to placing in group
		return true, "", nil
	}

	// Check if group is in launch-mode (RoomJoinType = "4")
	if channelInfo != nil && channelInfo.ChannelType == 1 {
		if cdb.gdb != nil {
			//Check if user is creator
			if channelInfo.CreateUserMetaId == chat.MetaId {
				return true, "", nil
			}

			// Check if user is admin
			isAdmin, err := cdb.gdb.IsUserAdmin(chat.GroupId, chat.MetaId, chat.Timestamp)
			if err != nil {
				// If getting admin status fails, default to placing in group
				return true, "", nil
			}
			if isAdmin {
				return true, "", nil
			}

			// Check if group has whitelist and user is not whitelisted
			isWhitelisted, err := cdb.gdb.IsUserWhitelist(chat.GroupId, chat.MetaId, chat.Timestamp)
			if err != nil {
				// If getting whitelist status fails, default to placing in group
				return true, "", nil
			}
			if isWhitelisted {
				return true, "", nil
			}
		}
		// Channel is in launch-mode, message should be placed in OutCollection
		reason := "channel is in launch-mode, messages are not displayed in channel"
		return false, reason, nil
	}

	// Check if user is blocked in the group
	if cdb.gdb != nil {
		isBlocked, err := cdb.gdb.IsUserBlock(chat.GroupId, chat.MetaId, chat.Timestamp)
		if err != nil {
			// If getting block status fails, default to placing in group
			return true, "", nil
		}
		if isBlocked {
			// User is blocked, message should be placed in OutCollection
			reason := "user is blocked in the channel"
			return false, reason, nil
		}
	}

	// Group is not in launch-mode, check user's state in group
	groupState, err := cdb.getUserGroupState(chat.MetaId, chat.GroupId, chat.Timestamp)
	if err != nil {
		// If getting state fails, default to placing in group
		return true, "", nil
	}

	// Check if user is in group
	if groupState == models.RoomStateIn {
		// User is in group, message should be placed in group
		return true, "", nil
	} else {
		// User is not in group, message should be placed in OutCollection
		reason := fmt.Sprintf("user is not in group, groupState: %d", groupState)
		return false, reason, nil
	}
}

// Get group info by group ID (helper method)
func (cdb *ChatDB) getGroupInfo(groupId string) (*models.TalkGroupModel, error) {
	key := []byte(groupId)
	value, closer, err := Pb[TalkGroupInfoCollection].Get(key)
	if err != nil {
		if err == pebble.ErrNotFound {
			return nil, nil
		}
		return nil, err
	}
	defer closer.Close()

	var groupInfo models.TalkGroupModel
	err = json.Unmarshal(value, &groupInfo)
	if err != nil {
		return nil, err
	}

	return &groupInfo, nil
}

func (cdb *ChatDB) GetChannelInfo(channelId string) (*models.TalkGroupChannelModel, error) {
	// Try to get channel info from cache first
	channelInfo, found := cache_service.GetGroupChannelInfoFromCache(channelId)
	if found {
		return channelInfo, nil
	}

	// Cache miss, get from database
	channelInfo, err := cdb.getChannelInfo(channelId)
	if err != nil {
		return nil, err
	}

	// Update cache with the fetched data
	if channelInfo != nil {
		cache_service.SetGroupChannelInfoToCache(channelId, channelInfo)
	}

	return channelInfo, nil
}

// Get channel info by channel ID (helper method)
func (cdb *ChatDB) getChannelInfo(channelId string) (*models.TalkGroupChannelModel, error) {
	key := []byte(channelId)
	value, closer, err := Pb[TalkGroupChannelInfoCollection].Get(key)
	if err != nil {
		if err == pebble.ErrNotFound {
			return nil, nil
		}
		return nil, err
	}
	defer closer.Close()

	var channelInfo models.TalkGroupChannelModel
	err = json.Unmarshal(value, &channelInfo)
	if err != nil {
		return nil, err
	}

	return &channelInfo, nil
}

// Save chat timestamp index (decide which collection to save to based on user state and group type)
func (cdb *ChatDB) SaveChatTimestampWithState(chat *models.TalkGroupChatV3, isResync bool) (bool, error) {
	// Check if message should be placed in group
	t := time.Now().UnixMilli()
	shouldPlaceInGroup, outReason, err := cdb.shouldPlaceMessageInGroup(chat)
	fmt.Println("[indexer]SaveChatTimestampWithState time:", time.Now().UnixMilli()-t)
	if err != nil {
		// If check fails, default to saving to normal collection
		return true, cdb.SaveChatTimestamp(chat, isResync)
	}

	// Construct timestamp index value: pinId_chatType_timestamp
	value := chat.PinId + "_" + strconv.FormatInt(int64(chat.ChatType), 10) + "_" + strconv.FormatInt(chat.Timestamp, 10)

	// Decide which collection to save to based on shouldPlaceInGroup
	var (
		collection  string
		collection2 string
		isGoEnqueue bool = true
	)
	if shouldPlaceInGroup {
		// Message should be in group, save to normal collection
		collection = TalkGroupChatTimestampCollection
		collection2 = TalkGroupChatTimestamp2Collection
	} else {
		// Message should not be in group, save to invalid collection
		collection = TalkGroupChatTimestampOutCollection
		collection2 = TalkGroupChatTimestamp2OutCollection
		isGoEnqueue = false
		chat.OutReason = outReason
	}

	// Use GroupId_Timestamp as primary key to support timestamp range queries
	key := []byte(chat.GroupId + "_" + strconv.FormatInt(chat.Timestamp, 10))
	if err = Pb[collection].Set(key, []byte(value), pebble.Sync); err != nil {
		return isGoEnqueue, err
	}

	// Use GroupId_Timestamp as primary key to support timestamp range queries
	// For collection2, we need to handle array format
	err = cdb.saveChatTimestamp2WithCollection(chat, collection2, isResync)
	if err != nil {
		return isGoEnqueue, err
	}

	if shouldPlaceInGroup {
		if !isResync {
			go dealGroupChatItem(chat)
		}
	} else {
		cdb.SaveChat(chat)
	}

	return isGoEnqueue, nil
}

// Process file group chat
func (cdb *ChatDB) processFileGroupChat(pin *pin.PinInscription, isResync bool) error {
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

	isSynced, err := IsPinSynced(pin.Id)
	if err != nil {
		return err
	}
	if isSynced {
		// Already synced, skip processing
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
		ChannelId:   simpleFileGroupChat.ChannelId,
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
		Version:     pin.Version,
		BlockHeight: pin.GenesisHeight,
		Chain:       pin.ChainName,
		Index:       -1,
	}

	// Save chat message to TalkGroupChatPinCollection
	err = cdb.SaveChat(chat)
	if err != nil {
		return err
	}

	// Process chat timestamp and enqueue based on whether it's a channel chat or group chat
	if chat.ChannelId != "" {
		// This is a channel chat
		err = cdb.processChannelChatTimestampAndEnqueue(chat, isResync)
	} else {
		// This is a group chat
		err = cdb.processChatTimestampAndEnqueue(chat, isResync)
	}
	if err != nil {
		return err
	}

	// Mark pin as synced
	err = MarkPinAsSynced(pin.Id, true)
	if err != nil {
		return err
	}

	return nil
}

// Process group lucky bag
func (cdb *ChatDB) processGroupLuckyBag(pin *pin.PinInscription, txData *wire.MsgTx, isResync bool) error {
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

	isSynced, err := IsPinSynced(pin.Id)
	if err != nil {
		return err
	}
	if isSynced {
		// Already synced, skip processing
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
		payList          []*models.ProInfoPayList = make([]*models.ProInfoPayList, 0)
		errPayList       []*models.ProInfoPayList = make([]*models.ProInfoPayList, 0)
		luckyBagVouts    []*models.LuckyBagOutput = make([]*models.LuckyBagOutput, 0)
		errLuckyBagVouts []*models.LuckyBagOutput = make([]*models.LuckyBagOutput, 0)

		hasLuckyFeeRate      bool  = false
		txFeeRate            int64 = 1
		openLuckyTxSize      int64 = protocols.OpenLuckyTxSize
		openLuckyTxFee       int64 = 0
		openLuckyTotalAmount int64 = 0
		openLuckyTotalFee    int64 = 0
	)

	if simpleLuckyBag.FeeRate != nil && simpleLuckyBag.FeeRate != "0" && simpleLuckyBag.FeeRate != "" {
		feeRate := toInt64(simpleLuckyBag.FeeRate)
		if feeRate >= 1 {
			txFeeRate = feeRate
			hasLuckyFeeRate = true
		}
	}
	openLuckyTxFee = openLuckyTxSize * txFeeRate

	// Create a map to store PayList items by index for quick lookup
	payListByIndex := make(map[int64]*models.ProInfoPayList)
	luckyTxOutList := make([]*models.LuckyBagOutput, 0)

	payAddressList := make([]string, 0)
	// First, process all PayList items
	for _, pay := range simpleLuckyBag.PayList {
		payAddressList = append(payAddressList, pay.Address)
		itemLuckyAmount := toInt64(pay.Amount) - openLuckyTxFee
		itemLuckyFee := openLuckyTxFee
		itemLuckyFeeRate := toString(txFeeRate)
		if itemLuckyAmount < 546 {
			itemLuckyAmount = 546
			itemLuckyFee = toInt64(pay.Amount) - itemLuckyAmount
			itemLuckyFeeRate = toString(float64(itemLuckyFee) / float64(openLuckyTxSize))
		}

		payItem := &models.ProInfoPayList{
			Amount:  toString(pay.Amount),
			Address: pay.Address,
			Index:   toInt64(pay.Index),
			// LuckyAmount:  toString(itemLuckyAmount),
			// LuckyFee:     toString(itemLuckyFee),
			// LuckyFeeRate: itemLuckyFeeRate,
		}
		if hasLuckyFeeRate {
			payItem.LuckyAmount = toString(itemLuckyAmount)
			payItem.LuckyFee = toString(itemLuckyFee)
			payItem.LuckyFeeRate = itemLuckyFeeRate
		}

		openLuckyTotalAmount += toInt64(payItem.LuckyAmount)
		openLuckyTotalFee += toInt64(payItem.LuckyFee)
		if _, ok := payListByIndex[toInt64(pay.Index)]; ok {
			errPayList = append(errPayList, payItem)
		} else {
			payListByIndex[toInt64(pay.Index)] = payItem
		}
	}

	if txData != nil {
		// First, extract all TxOut addresses and find matching UTXOs
		for i, vout := range txData.TxOut {
			index := int64(i)

			// Extract address from vout
			voutAddress := ""
			// Get chain params based on chain name
			var netParams *chaincfg.Params = &chaincfg.MainNetParams
			if common.TestNet == "1" {
				netParams = &chaincfg.TestNet3Params
			} else if common.TestNet == "2" {
				netParams = &chaincfg.RegressionNetParams
			}

			class, addresses, _, _ := txscript.ExtractPkScriptAddrs(vout.PkScript, netParams)
			if class.String() != "nulldata" && class.String() != "nonstandard" && len(addresses) > 0 {
				voutAddress = addresses[0].String()
			}

			// Check if this vout address is in payAddressList (belongs to this lucky bag)
			if len(payAddressList) > 0 && contains(payAddressList, voutAddress) {
				// This is a lucky bag UTXO, add it to the list
				luckyBagVout := &models.LuckyBagOutput{
					ScriptPubKey: hex.EncodeToString(vout.PkScript),
					Amount:       uint64(vout.Value),
					Address:      voutAddress,
					Index:        index,
				}
				luckyTxOutList = append(luckyTxOutList, luckyBagVout)
			}
		}

		// Now process only the lucky bag UTXOs
		for _, luckyBagVout := range luckyTxOutList {
			// Check if this index exists in PayList
			if payItem, exists := payListByIndex[luckyBagVout.Index]; exists {
				// Index exists in PayList, check if address matches
				if payItem.Address == luckyBagVout.Address {
					// Both index and address match - this is correct
					payList = append(payList, payItem)
					luckyBagVouts = append(luckyBagVouts, luckyBagVout)
				} else {
					// Index exists but address doesn't match - this is an error
					errPayList = append(errPayList, payItem)
					errLuckyBagVouts = append(errLuckyBagVouts, luckyBagVout)
				}
			} else {
				// Index doesn't exist in PayList - this is an error
				errLuckyBagVouts = append(errLuckyBagVouts, luckyBagVout)
			}
		}

		// Check for PayList items that don't have corresponding TxOut
		for index, payItem := range payListByIndex {
			found := false
			for _, vout := range luckyBagVouts {
				if vout.Index == index {
					found = true
					break
				}
			}
			for _, vout := range errLuckyBagVouts {
				if vout.Index == index {
					found = true
					break
				}
			}

			if !found {
				// PayList item exists but no corresponding TxOut - this is an error
				errPayList = append(errPayList, payItem)
			}
		}
	} else {
		// No txData available, all PayList items are considered errors
		for _, payItem := range payListByIndex {
			errPayList = append(errPayList, payItem)
		}
	}
	count := toInt64(simpleLuckyBag.Count)

	// Create lucky bag model
	redEnvelope := &models.TalkGroupLuckyBagV3{
		CommunityId:         "", // Need to get from group info
		GroupId:             simpleLuckyBag.GroupId,
		ChannelId:           simpleLuckyBag.ChannelId,
		TxId:                pin.Id[:len(pin.Id)-2],
		PinId:               pin.Id,
		MetaId:              pin.CreateMetaId,
		Address:             pin.CreateAddress,
		Protocol:            pin.Path,
		SubId:               simpleLuckyBag.SubId,
		Code:                simpleLuckyBag.Code,
		CreateTimeStr:       formatInt64(simpleLuckyBag.CreateTime),
		Domain:              simpleLuckyBag.Domain,
		LuckyBagAddress:     simpleLuckyBag.LuckyBagAddress,
		GenType:             0,
		GenState:            0,
		Content:             simpleLuckyBag.Content,
		Img:                 simpleLuckyBag.Img,
		ImgType:             simpleLuckyBag.ImgType,
		Amount:              formatInt64(simpleLuckyBag.Amount),
		LuckyTotalAmount:    formatInt64(openLuckyTotalAmount),
		LuckyTotalFee:       formatInt64(openLuckyTotalFee),
		FeeRate:             formatInt64(txFeeRate),
		Count:               toString(simpleLuckyBag.Count),
		ValidCount:          toString(len(payList)),
		ErrCount:            toString(count - int64(len(payList))),
		PayList:             payList,
		ErrPayList:          errPayList,
		LuckyBagVouts:       luckyBagVouts,
		ErrLuckyBagVouts:    errLuckyBagVouts,
		Type:                simpleLuckyBag.Type,
		RequireType:         toString(simpleLuckyBag.RequireType),
		RequireTickId:       simpleLuckyBag.RequireTickId,
		RequireCollectionId: simpleLuckyBag.RequireCollectionId,
		LimitAmount:         toUint64(simpleLuckyBag.LimitAmount),
		Timestamp:           pin.Timestamp,
		BlockHeight:         pin.GenesisHeight,
		Chain:               pin.ChainName,
	}

	chatType := models.ChatTypeLuckyBag
	// Check if this is an internal lucky bag (has domain and LuckyBagAddress)
	if simpleLuckyBag.Domain != "" && simpleLuckyBag.LuckyBagAddress != "" {
		chatType = models.ChatTypeLuckyBagV2
		// This is an internal lucky bag, verify the code and address
		codeAddressKey, err := cdb.GetLuckyBagCodeAddressKeyByCodeAndAddress(simpleLuckyBag.Code, simpleLuckyBag.LuckyBagAddress)
		if err != nil {
			log.Printf("Failed to get lucky bag code address key for code %s and address %s: %v", simpleLuckyBag.Code, simpleLuckyBag.LuckyBagAddress, err)
			// Set as error state if verification fails
			redEnvelope.GenType = 1  // Internal type
			redEnvelope.GenState = 2 // Error state
		} else if codeAddressKey != nil {
			// Verification successful, set as internal type and completed state
			redEnvelope.GenType = 1  // Internal type
			redEnvelope.GenState = 1 // Completed state
		} else {
			// Code and address not found, set as error state
			redEnvelope.GenType = 2 // External type
			redEnvelope.GenState = 1
		}
	} else {
		// This is an external lucky bag, set default values
		redEnvelope.GenType = 0  // External type
		redEnvelope.GenState = 0 // Default state
	}

	if len(errPayList) > 0 || len(errLuckyBagVouts) > 0 {
		redEnvelope.State = 4 // err
	} else {
		redEnvelope.State = 1 // pending
	}

	// Save lucky bag info
	err = cdb.SaveLuckyBag(redEnvelope)
	if err != nil {
		return err
	}

	// If this is an internal lucky bag with successful verification, move the key-value pair to completed collection
	if redEnvelope.GenType == 1 && redEnvelope.GenState == 1 {
		err = cdb.moveLuckyBagCodeAddressKeyToCompleted(simpleLuckyBag.Code, simpleLuckyBag.LuckyBagAddress)
		if err != nil {
			log.Printf("Failed to move lucky bag code address key to completed collection: %v", err)
			// Don't return error here as the main lucky bag save was successful
		}
	}

	if redEnvelope.GenType != 0 && redEnvelope.GenType != 2 {
		// Save lucky bag to appropriate collection based on error status
		err = cdb.SaveLuckyBagPendingToCollection(redEnvelope)
		if err != nil {
			return err
		}
	}

	// Create chat message model (for group chat display)
	chat := &models.TalkGroupChatV3{
		GroupId:     simpleLuckyBag.GroupId,
		ChannelId:   simpleLuckyBag.ChannelId,
		TxId:        pin.Id[:len(pin.Id)-2],
		PinId:       pin.Id,
		MetaId:      pin.CreateMetaId,
		Address:     pin.CreateAddress,
		Protocol:    pin.Path,
		Content:     "[LuckyBag]:" + simpleLuckyBag.Content, // Lucky bag blessing message
		ContentType: "text/plain",
		Encryption:  "",
		ChatType:    chatType,                 // Lucky bag type
		InsideIndex: models.ChatInsideIndexIn, // Default to in state
		ReplyPin:    "",
		Timestamp:   pin.Timestamp,
		Chain:       pin.ChainName,
		BlockHeight: pin.GenesisHeight,
		Version:     pin.Version,
		Index:       -1,
	}

	// Save chat message to TalkGroupChatPinCollection
	err = cdb.SaveChat(chat)
	if err != nil {
		return err
	}

	// Process chat timestamp and enqueue based on whether it's a channel chat or group chat
	if chat.ChannelId != "" {
		// This is a channel chat
		err = cdb.processChannelChatTimestampAndEnqueue(chat, isResync)
	} else {
		// This is a group chat
		err = cdb.processChatTimestampAndEnqueue(chat, isResync)
	}
	if err != nil {
		return err
	}

	// Mark pin as synced
	err = MarkPinAsSynced(pin.Id, true)
	if err != nil {
		return err
	}

	return nil
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// Process group grab lucky bag
func (cdb *ChatDB) processGroupOpenLuckyBag(pin *pin.PinInscription, txData *wire.MsgTx, isResync bool) error {
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

	isSynced, err := IsPinSynced(pin.Id)
	if err != nil {
		return err
	}
	if isSynced {
		// Already synced, skip processing
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
	err = cdb.SaveOpenLuckyBagListAtomic(simpleOpenLuckyBag.LuckyBagPinId, pin.Id, simpleOpenLuckyBag.GroupId, pin.Timestamp, pin.CreateMetaId, pin.CreateAddress, int64(index))
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
		Index:       -1,
	}

	// Save chat message to TalkGroupChatPinCollection
	err = cdb.SaveChat(chat)
	if err != nil {
		return err
	}

	// Save timestamp index (decide which collection to save to based on user state)
	isGoEnqueue, err := cdb.SaveChatTimestampWithState(chat, isResync)
	if err != nil {
		return err
	}

	// Enqueue message for asynchronous group list updates
	if isGoEnqueue {
		err = cdb.EnqueueChatMessage(chat, isResync)
		if err != nil {
			return err
		}
	}

	// Mark pin as synced
	err = MarkPinAsSynced(pin.Id, true)
	if err != nil {
		return err
	}

	return nil
}

// Process group reclaim lucky bag
func (cdb *ChatDB) processGroupResidueLuckyBag(pin *pin.PinInscription, txData *wire.MsgTx, isResync bool) error {
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

	isSynced, err := IsPinSynced(pin.Id)
	if err != nil {
		return err
	}
	if isSynced {
		// Already synced, skip processing
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

	// Mark pin as synced
	err = MarkPinAsSynced(pin.Id, true)
	if err != nil {
		return err
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

// UpdateChatIndex updates the chat index for a group chat message
func (cdb *ChatDB) UpdateChatIndex(chat *models.TalkGroupChatV3, isResync bool) error {
	//if isResync, frist get chat from database
	if isResync {
		dbChat, err := cdb.GetChatByPinId(chat.PinId)
		if err != nil && err != pebble.ErrNotFound {
			return err
		}
		if dbChat != nil && dbChat.Index != -1 {
			//already has index, skip
			return nil
		}
	}

	// Get the next index for this group
	nextIndex, err := cdb.getNextGroupChatIndex(chat.GroupId)
	if err != nil {
		return err
	}

	// Update the chat message with the new index
	chat.Index = nextIndex

	log.Printf("[UpdateChatIndex]nextIndex: %d", nextIndex)

	// Save the updated chat message
	err = cdb.SaveChat(chat)
	if err != nil {
		return err
	}

	// Save the index mapping with zero-padded index for proper sorting
	indexKey := chat.GroupId + "_" + fmt.Sprintf("%040d", nextIndex)
	indexValue := chat.PinId + "_" + strconv.FormatInt(int64(chat.ChatType), 10) + "_" + strconv.FormatInt(chat.Timestamp, 10) + "_1"

	return Pb[TalkGroupChatIndexCollection].Set([]byte(indexKey), []byte(indexValue), pebble.Sync)
}

// getNextGroupChatIndex gets the next available index for a group
func (cdb *ChatDB) getNextGroupChatIndex(groupId string) (int64, error) {
	iter, err := Pb[TalkGroupChatIndexCollection].NewIter(&pebble.IterOptions{
		LowerBound: []byte(groupId + "_"),
		UpperBound: []byte(groupId + "_" + string([]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff})),
	})
	if err != nil {
		return 0, err
	}
	defer iter.Close()

	// Find the last key for this group (since keys are sorted, the last one has the highest index)
	var lastKey []byte

	// Use Last() to get the last key in the range
	if iter.Last(); iter.Valid() {
		lastKey = iter.Key()
	}

	// If no keys found for this group, start with index 1
	if lastKey == nil {
		return 1, nil
	}

	// Extract index from the last key (groupId_index with zero-padding)
	keyStr := string(lastKey)

	log.Printf("[getNextGroupChatIndex]keyStr: %s", keyStr)
	parts := strings.Split(keyStr, "_")
	if len(parts) >= 2 {
		// Remove leading zeros and parse the index
		indexStr := strings.TrimLeft(parts[1], "0")
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

// GetCurrentMaxGroupChatIndex gets the current maximum index for a group
func (cdb *ChatDB) GetCurrentMaxGroupChatIndex(groupId string) (int64, error) {
	iter, err := Pb[TalkGroupChatIndexCollection].NewIter(&pebble.IterOptions{
		LowerBound: []byte(groupId + "_"),
		UpperBound: []byte(groupId + "_" + string([]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff})),
	})
	if err != nil {
		return 0, err
	}
	defer iter.Close()

	// Find the last key for this group (since keys are sorted, the last one has the highest index)
	var lastKey []byte

	// Use Last() to get the last key in the range
	if iter.Last(); iter.Valid() {
		lastKey = iter.Key()
	}

	// If no keys found for this group, return 0
	if lastKey == nil {
		return 0, nil
	}

	// Extract index from the last key (groupId_index with zero-padding)
	keyStr := string(lastKey)
	parts := strings.Split(keyStr, "_")
	if len(parts) >= 2 {
		// Remove leading zeros and parse the index
		indexStr := strings.TrimLeft(parts[1], "0")
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

// SaveLuckyBagPendingToCollection saves lucky bag to either pending or error pending collection based on error status
func (cdb *ChatDB) SaveLuckyBagPendingToCollection(luckyBag *models.TalkGroupLuckyBagV3) error {
	// Determine which collection to save to based on error status
	var collection string
	if len(luckyBag.ErrLuckyBagVouts) == 0 && len(luckyBag.ErrPayList) == 0 {
		// No errors, save to pending collection
		collection = TalkGroupLuckyBagPinPendingCollection
	} else {
		// Has errors, save to error collection
		collection = TalkGroupLuckyBagPinErrPendingCollection
	}

	// Use PinId as primary key
	key := []byte(luckyBag.PinId)
	return Pb[collection].Set(key, []byte(luckyBag.PinId), pebble.Sync)
}

// GetChatsByGroupIdAndStartIndexRange gets chat messages by group ID and index range (ascending order)
// This function handles the key format: groupId_index (with zero-padding)
// Example: groupId_0000000000000000000000000000000000000001
// Returns chat list, last index, and error
func (cdb *ChatDB) GetChatsByGroupIdAndStartIndexRange(groupId string, startIndex int64, size int64) ([]*models.TalkGroupChatV3, int64, error) {
	var chats []*models.TalkGroupChatV3

	// Create iter options to limit the range to only keys for this group
	iterOptions := &pebble.IterOptions{
		LowerBound: []byte(groupId + "_"),
		UpperBound: []byte(groupId + "_" + string([]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff})),
	}

	iter, err := Pb[TalkGroupChatIndexCollection].NewIter(iterOptions)
	if err != nil {
		return nil, 0, err
	}
	defer iter.Close()

	lastIndex := int64(0)

	// Construct query start key: groupId_startIndex (with zero-padding)
	// Since key format is groupId_index with zero-padding, we can use proper range scanning
	// Example: groupId_0000000000000000000000000000000000000001
	startKey := []byte(groupId + "_" + fmt.Sprintf("%040d", startIndex))

	// Start iteration from specified index (ascending order)
	// Use SeekGE to find the first key that is greater than or equal to our startKey
	for iter.SeekGE(startKey); iter.Valid() && iter.Key() != nil; iter.Next() {
		key := string(iter.Key())
		// fmt.Printf("[CHAT_DB] GetChatsByGroupIdAndStartIndexRange key: %s\n", key)

		// Parse key to extract index
		// Key format: groupId_index (with zero-padding)
		keyParts := strings.Split(key, "_")
		if len(keyParts) < 2 {
			continue
		}

		// Extract index from key (remove leading zeros)
		indexStr := strings.TrimLeft(keyParts[1], "0")
		if indexStr == "" {
			indexStr = "0" // If all zeros, treat as 0
		}
		index, err := strconv.ParseInt(indexStr, 10, 64)
		if err != nil {
			continue
		}

		// Skip messages before our start index (since we're going forwards)
		if index < startIndex {
			continue
		}

		// Parse value: pinId_chatType_timestamp_state
		value := string(iter.Value())
		valueParts := strings.Split(value, "_")
		if len(valueParts) < 1 {
			continue
		}

		pinId := valueParts[0]
		// fmt.Printf("[CHAT_DB] GetChatsByGroupIdAndStartIndexRange index: %d, pinId: %s\n", index, pinId)

		// Get complete chat message
		chat, err := cdb.GetChatByPinId(pinId)
		if err != nil || chat == nil {
			continue
		}

		if index > lastIndex {
			lastIndex = index
		}

		// Add to results
		chats = append(chats, chat)

		// Check pagination limit
		if int64(len(chats)) >= size {
			break
		}
	}

	return chats, lastIndex, nil
}

// GetChatsByGroupIdAndStartTimestampRange gets chat messages by group ID and start timestamp range (ascending order)
// This function handles the key format: groupId_timestamp+number(6)
// Example: groupId_1755500889000001 (timestamp 1755500889 + random 000001)
// Returns chat list, last time, and error
func (cdb *ChatDB) GetChatsByGroupIdAndStartTimestampRange(groupId string, startTimestamp int64, size int64) ([]*models.TalkGroupChatV3, int64, error) {
	var chats []*models.TalkGroupChatV3

	// Create iter options to limit the range to only keys for this group
	iterOptions := &pebble.IterOptions{
		LowerBound: []byte(groupId + "_"),
		UpperBound: []byte(groupId + "_" + string([]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff})),
	}

	iter, err := Pb[TalkGroupChatTimestamp2Collection].NewIter(iterOptions)
	if err != nil {
		return nil, 0, err
	}
	defer iter.Close()

	lastTimestamp := int64(0)

	// Construct query start key: groupId_startTimestamp
	// Since key format is now groupId_timestamp+number(6), we can use proper range scanning
	// Example: groupId_1755500889000001 (timestamp 1755500889 + random 000001)
	startKey := []byte(groupId + "_" + strconv.FormatInt(startTimestamp, 10))

	// Start iteration from specified timestamp (ascending order)
	// Use SeekGE to find the first key that is greater than or equal to our startKey
	for iter.SeekGE(startKey); iter.Valid() && iter.Key() != nil; iter.Next() {
		key := string(iter.Key())
		// fmt.Printf("[CHAT_DB] GetChatsByGroupIdAndStartTimestampRange key: %s\n", key)

		// Parse key to extract timestamp
		// Key format: groupId_timestamp+number(6)
		keyParts := strings.Split(key, "_")
		if len(keyParts) < 2 {
			continue
		}

		// Extract timestamp from key (remove the last 6 digits which is the random number)
		timestampStr := keyParts[1]
		timestampKeyInt, _ := strconv.ParseInt(timestampStr, 10, 64)

		// Skip messages before our start timestamp (since we're going forwards)
		if timestampKeyInt < startTimestamp {
			continue
		}

		// Parse value: pinId_chatType_timestamp_number
		value := string(iter.Value())
		valueParts := strings.Split(value, "_")
		if len(valueParts) < 1 {
			continue
		}

		pinId := valueParts[0]
		// fmt.Printf("[CHAT_DB] GetChatsByGroupIdAndStartTimestampRange timestampStr: %s, pinId: %s\n", timestampStr, pinId)

		// Get complete chat message
		chat, err := cdb.GetChatByPinId(pinId)
		if err != nil || chat == nil {
			continue
		}

		if timestampKeyInt > lastTimestamp {
			lastTimestamp = timestampKeyInt
		}

		// Add to results
		chats = append(chats, chat)

		// Check pagination limit
		if int64(len(chats)) >= size {
			break
		}
	}

	return chats, lastTimestamp, nil
}

// ==================== Channel Chat Query Methods ====================

// GetChatsByChannelIdAndEndTimestampRange3 gets chat messages by channel ID and end timestamp range (descending order)
// This function handles the key format: channelId_timestamp+number(6)
// Example: channelId_1755500889000001 (timestamp 1755500889 + random 000001)
// Returns chat list, next timestamp, and error
func (cdb *ChatDB) GetChatsByChannelIdAndEndTimestampRange3(channelId string, endTimestamp int64, size int64) ([]*models.TalkGroupChatV3, int64, error) {
	var chats []*models.TalkGroupChatV3

	// Create iter options to limit the range to only keys for this channel
	iterOptions := &pebble.IterOptions{
		LowerBound: []byte(channelId + "_"),
		UpperBound: []byte(channelId + "_" + string([]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff})),
	}

	iter, err := Pb[TalkGroupChannelChatTimestamp2Collection].NewIter(iterOptions)
	if err != nil {
		return nil, 0, err
	}
	defer iter.Close()

	nextTimestamp := int64(0)

	// Construct query start key: channelId_endTimestamp
	// Since key format is now channelId_timestamp+number(6), we can use proper range scanning
	// Example: channelId_1755500889000001 (timestamp 1755500889 + random 000001)
	startKey := []byte(channelId + "_" + strconv.FormatInt(endTimestamp, 10))

	// Seek to start key and iterate backwards
	for iter.SeekLT(startKey); iter.Valid(); iter.Prev() {
		key := string(iter.Key())
		// fmt.Printf("[CHAT_DB] GetChatsByChannelIdAndEndTimestampRange3 key: %s\n", key)

		// Parse key to extract timestamp
		// Key format: channelId_timestamp+number(6)
		keyParts := strings.Split(key, "_")
		if len(keyParts) < 2 {
			continue
		}

		// Extract timestamp from key (remove the last 6 digits which is the random number)
		timestampStr := keyParts[1]
		timestampKeyInt, _ := strconv.ParseInt(timestampStr, 10, 64)

		// Parse value: pinId_chatType_timestamp_number
		value := string(iter.Value())
		valueParts := strings.Split(value, "_")
		if len(valueParts) < 4 {
			continue
		}
		pinId := valueParts[0]
		// fmt.Printf("[CHAT_DB] GetChatsByChannelIdAndEndTimestampRange3 timestampStr: %s, pinId: %s\n", timestampStr, pinId)

		// Get complete chat message
		chat, err := cdb.GetChatByPinId(pinId)
		if err != nil || chat == nil {
			continue
		}

		// Update next timestamp for pagination (this will be the timestamp of the last message we return)
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

// GetChatsByChannelIdAndStartIndexRange gets chat messages by channel ID and index range (ascending order)
// This function handles the key format: channelId_index (with zero-padding)
// Example: channelId_0000000000000000000000000000000000000001
// Returns chat list, last index, and error
func (cdb *ChatDB) GetChatsByChannelIdAndStartIndexRange(channelId string, startIndex int64, size int64) ([]*models.TalkGroupChatV3, int64, error) {
	var chats []*models.TalkGroupChatV3

	// Create iter options to limit the range to only keys for this channel
	iterOptions := &pebble.IterOptions{
		LowerBound: []byte(channelId + "_"),
		UpperBound: []byte(channelId + "_" + string([]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff})),
	}

	iter, err := Pb[TalkGroupChannelChatIndexCollection].NewIter(iterOptions)
	if err != nil {
		return nil, 0, err
	}
	defer iter.Close()

	lastIndex := int64(0)

	// Construct query start key: channelId_startIndex (with zero-padding)
	// Since key format is channelId_index with zero-padding, we can use proper range scanning
	// Example: channelId_0000000000000000000000000000000000000001
	startKey := []byte(channelId + "_" + fmt.Sprintf("%040d", startIndex))

	// Seek to start key and iterate forwards
	for iter.SeekGE(startKey); iter.Valid(); iter.Next() {
		key := string(iter.Key())
		// fmt.Printf("[CHAT_DB] GetChatsByChannelIdAndStartIndexRange key: %s\n", key)

		// Parse key to extract index
		// Key format: channelId_index (with zero-padding)
		keyParts := strings.Split(key, "_")
		if len(keyParts) < 2 {
			continue
		}

		// Extract index from key (remove leading zeros)
		indexStr := strings.TrimLeft(keyParts[1], "0")
		if indexStr == "" {
			indexStr = "0" // If all zeros, treat as 0
		}
		index, err := strconv.ParseInt(indexStr, 10, 64)
		if err != nil {
			continue
		}

		// Skip messages before our start index (since we're going forwards)
		if index < startIndex {
			continue
		}

		// Parse value: pinId_chatType_timestamp_isSet
		value := string(iter.Value())
		valueParts := strings.Split(value, "_")
		if len(valueParts) < 4 {
			continue
		}
		pinId := valueParts[0]
		// fmt.Printf("[CHAT_DB] GetChatsByChannelIdAndStartIndexRange index: %d, pinId: %s\n", index, pinId)

		// Get complete chat message
		chat, err := cdb.GetChatByPinId(pinId)
		if err != nil || chat == nil {
			continue
		}

		if index > lastIndex {
			lastIndex = index
		}

		// Add to results
		chats = append(chats, chat)

		// Check pagination limit
		if int64(len(chats)) >= size {
			break
		}
	}

	return chats, lastIndex, nil
}

// GetChatsByChannelIdAndStartTimestampRange gets chat messages by channel ID and start timestamp range (ascending order)
// This function handles the key format: channelId_timestamp+number(6)
// Example: channelId_1755500889000001 (timestamp 1755500889 + random 000001)
// Returns chat list, last time, and error
func (cdb *ChatDB) GetChatsByChannelIdAndStartTimestampRange(channelId string, startTimestamp int64, size int64) ([]*models.TalkGroupChatV3, int64, error) {
	var chats []*models.TalkGroupChatV3

	// Create iter options to limit the range to only keys for this channel
	iterOptions := &pebble.IterOptions{
		LowerBound: []byte(channelId + "_"),
		UpperBound: []byte(channelId + "_" + string([]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff})),
	}

	iter, err := Pb[TalkGroupChannelChatTimestamp2Collection].NewIter(iterOptions)
	if err != nil {
		return nil, 0, err
	}
	defer iter.Close()

	lastTimestamp := int64(0)

	// Construct query start key: channelId_startTimestamp
	// Since key format is now channelId_timestamp+number(6), we can use proper range scanning
	// Example: channelId_1755500889000001 (timestamp 1755500889 + random 000001)
	startKey := []byte(channelId + "_" + strconv.FormatInt(startTimestamp, 10))

	// Seek to start key and iterate forwards
	for iter.SeekGE(startKey); iter.Valid(); iter.Next() {
		key := string(iter.Key())
		// fmt.Printf("[CHAT_DB] GetChatsByChannelIdAndStartTimestampRange key: %s\n", key)

		// Parse key to extract timestamp
		// Key format: channelId_timestamp+number(6)
		keyParts := strings.Split(key, "_")
		if len(keyParts) < 2 {
			continue
		}

		// Extract timestamp from key (remove the last 6 digits which is the random number)
		timestampStr := keyParts[1]
		timestampKeyInt, _ := strconv.ParseInt(timestampStr, 10, 64)

		// Skip messages before our start timestamp (since we're going forwards)
		if timestampKeyInt < startTimestamp {
			continue
		}

		// Parse value: pinId_chatType_timestamp_number
		value := string(iter.Value())
		valueParts := strings.Split(value, "_")
		if len(valueParts) < 4 {
			continue
		}
		pinId := valueParts[0]
		// fmt.Printf("[CHAT_DB] GetChatsByChannelIdAndStartTimestampRange timestampStr: %s, pinId: %s\n", timestampStr, pinId)

		// Get complete chat message
		chat, err := cdb.GetChatByPinId(pinId)
		if err != nil || chat == nil {
			continue
		}

		if timestampKeyInt > lastTimestamp {
			lastTimestamp = timestampKeyInt
		}

		// Add to results
		chats = append(chats, chat)

		// Check pagination limit
		if int64(len(chats)) >= size {
			break
		}
	}

	return chats, lastTimestamp, nil
}

// LuckyBagCodeAddressKey represents the structure stored in TalkGroupLuckyBagCodeAddressKeyCollection
type LuckyBagCodeAddressKey struct {
	Key             string `json:"key"`             // Private key
	Code            string `json:"code"`            // 6-digit random code
	LuckyBagAddress string `json:"luckyBagAddress"` // Lucky bag address
	Timestamp       int64  `json:"timestamp"`       // Creation timestamp
}

// GenerateLuckyBagCodeAddressKey generates a new lucky bag code address key
// This method generates a private key, address, and 6-digit random code
// and saves it to TalkGroupLuckyBagCodeAddressKeyCollection
func (cdb *ChatDB) GenerateLuckyBagCodeAddressKey() (*LuckyBagCodeAddressKey, error) {
	// Get chain parameters based on configuration
	var netParams *chaincfg.Params = &chaincfg.MainNetParams
	if common.TestNet == "1" {
		netParams = &chaincfg.TestNet3Params
	} else if common.TestNet == "2" {
		netParams = &chaincfg.RegressionNetParams
	}

	// Generate private key and address
	privateKey, address, err := common.GenerateKeyAndLegacyAddress(netParams)
	if err != nil {
		return nil, fmt.Errorf("failed to generate key and address: %v", err)
	}

	// Generate 6-digit random code (alphanumeric, case-sensitive)
	code := generateRandomCode(8)

	// Create the key structure
	codeAddressKey := &LuckyBagCodeAddressKey{
		Key:             privateKey,
		Code:            code,
		LuckyBagAddress: address,
		Timestamp:       time.Now().UnixMilli(),
	}

	// Save to database
	// Key format: code_address
	key := code + "_" + address
	data, err := json.Marshal(codeAddressKey)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal code address key: %v", err)
	}

	err = Pb[TalkGroupLuckyBagCodeAddressKeyCollection].Set([]byte(key), data, pebble.Sync)
	if err != nil {
		return nil, fmt.Errorf("failed to save code address key: %v", err)
	}

	return codeAddressKey, nil
}

// GetLuckyBagCodeAddressKeyByCodeAndAddress retrieves the lucky bag code address key
// based on the provided code and address
func (cdb *ChatDB) GetLuckyBagCodeAddressKeyByCodeAndAddress(code, address string) (*LuckyBagCodeAddressKey, error) {
	if code == "" || address == "" {
		return nil, fmt.Errorf("code and address cannot be empty")
	}

	// Construct key: code_address
	key := code + "_" + address

	// First, try to get from TalkGroupLuckyBagCodeAddressKeyCollection
	value, closer, err := Pb[TalkGroupLuckyBagCodeAddressKeyCollection].Get([]byte(key))
	if err != nil {
		if err == pebble.ErrNotFound {
			// Not found in main collection, try completed collection
			value, closer, err = Pb[TalkGroupLuckyBagCodeAddressKeyCompletedCollection].Get([]byte(key))
			if err != nil {
				if err == pebble.ErrNotFound {
					return nil, nil // Not found in either collection
				}
				return nil, fmt.Errorf("failed to get code address key from completed collection: %v", err)
			}
		} else {
			return nil, fmt.Errorf("failed to get code address key from main collection: %v", err)
		}
	}
	defer closer.Close()

	// Unmarshal the value
	var codeAddressKey LuckyBagCodeAddressKey
	err = json.Unmarshal(value, &codeAddressKey)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal code address key: %v", err)
	}

	return &codeAddressKey, nil
}

// GetLuckyBagCodeAddressKeyFromCompleted retrieves the lucky bag code address key
// directly from TalkGroupLuckyBagCodeAddressKeyCompletedCollection
func (cdb *ChatDB) GetLuckyBagCodeAddressKeyFromCompleted(code, address string) (*LuckyBagCodeAddressKey, error) {
	if code == "" || address == "" {
		return nil, fmt.Errorf("code and address cannot be empty")
	}

	// Construct key: code_address
	key := code + "_" + address

	// Get directly from completed collection
	value, closer, err := Pb[TalkGroupLuckyBagCodeAddressKeyCompletedCollection].Get([]byte(key))
	if err != nil {
		if err == pebble.ErrNotFound {
			return nil, nil // Not found in completed collection
		}
		return nil, fmt.Errorf("failed to get code address key from completed collection: %v", err)
	}
	defer closer.Close()

	// Unmarshal the value
	var codeAddressKey LuckyBagCodeAddressKey
	err = json.Unmarshal(value, &codeAddressKey)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal code address key from completed collection: %v", err)
	}

	return &codeAddressKey, nil
}

// generateRandomCode generates a random alphanumeric code with specified length
// Characters include: 0-9, A-Z, a-z (case-sensitive)
func generateRandomCode(length int) string {
	const charset = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	code := make([]byte, length)

	for i := range code {
		code[i] = charset[rand.Intn(len(charset))]
	}

	return string(code)
}

// moveLuckyBagCodeAddressKeyToCompleted moves a lucky bag code address key from the main collection to the completed collection
func (cdb *ChatDB) moveLuckyBagCodeAddressKeyToCompleted(code, address string) error {
	if code == "" || address == "" {
		return fmt.Errorf("code and address cannot be empty")
	}

	// Construct key: code_address
	key := code + "_" + address

	// Get the value from TalkGroupLuckyBagCodeAddressKeyCollection
	value, closer, err := Pb[TalkGroupLuckyBagCodeAddressKeyCollection].Get([]byte(key))
	if err != nil {
		if err == pebble.ErrNotFound {
			return fmt.Errorf("lucky bag code address key not found: %s", key)
		}
		return fmt.Errorf("failed to get lucky bag code address key: %v", err)
	}
	defer closer.Close()

	// Save to TalkGroupLuckyBagCodeAddressKeyCompletedCollection
	err = Pb[TalkGroupLuckyBagCodeAddressKeyCompletedCollection].Set([]byte(key), value, pebble.Sync)
	if err != nil {
		return fmt.Errorf("failed to save to completed collection: %v", err)
	}

	// Delete from the original collection
	err = Pb[TalkGroupLuckyBagCodeAddressKeyCollection].Delete([]byte(key), pebble.Sync)
	if err != nil {
		return fmt.Errorf("failed to delete from original collection: %v", err)
	}

	return nil
}

// SaveOpenLuckyBagError saves open lucky bag error record to TalkGroupOpenLuckyBagErrCollection
func (cdb *ChatDB) SaveOpenLuckyBagError(pinId string, luckyBagPinId string) error {
	// Use PinId as key, LuckyBagPinId as value
	key := []byte(pinId)
	value := []byte(luckyBagPinId)
	return Pb[TalkGroupOpenLuckyBagErrCollection].Set(key, value, pebble.Sync)
}

// SaveResidueLuckyBagError saves residue lucky bag error record to TalkGroupResidueLuckyBagErrCollection
func (cdb *ChatDB) SaveResidueLuckyBagError(pinId string, luckyBagPinId string) error {
	// Use PinId as key, LuckyBagPinId as value
	key := []byte(pinId)
	value := []byte(luckyBagPinId)
	return Pb[TalkGroupResidueLuckyBagErrCollection].Set(key, value, pebble.Sync)
}

// ==================== Chat Processing Helper Methods ====================

// Process chat timestamp and enqueue for group chats
func (cdb *ChatDB) processChatTimestampAndEnqueue(chat *models.TalkGroupChatV3, isResync bool) error {
	// Save timestamp index (decide which collection to save to based on user state)
	isGoEnqueue, err := cdb.SaveChatTimestampWithState(chat, isResync)
	if err != nil {
		return err
	}

	// Enqueue message for asynchronous group list updates
	if isGoEnqueue {
		err = cdb.EnqueueChatMessage(chat, isResync)
		if err != nil {
			return err
		}
	}

	return nil
}

// Process chat timestamp and enqueue for channel chats
func (cdb *ChatDB) processChannelChatTimestampAndEnqueue(chat *models.TalkGroupChatV3, isResync bool) error {
	// Save timestamp index (decide which collection to save to based on user state)
	isGoEnqueue, err := cdb.SaveChannelChatTimestampWithState(chat, isResync)
	if err != nil {
		return err
	}

	// Enqueue message for asynchronous channel list updates
	if isGoEnqueue {
		err = cdb.EnqueueChannelChatMessage(chat, isResync)
		if err != nil {
			return err
		}
	}

	return nil
}

// Save channel chat timestamp index (decide which collection to save to based on user state and channel type)
func (cdb *ChatDB) SaveChannelChatTimestampWithState(chat *models.TalkGroupChatV3, isResync bool) (bool, error) {
	// Check if message should be placed in channel
	t := time.Now().UnixMilli()
	shouldPlaceInChannel, outReason, err := cdb.shouldPlaceMessageInChannel(chat)
	fmt.Println("[indexer]SaveChannelChatTimestampWithState time:", time.Now().UnixMilli()-t)
	if err != nil {
		// If check fails, default to saving to normal collection
		return true, cdb.SaveChannelChatTimestamp(chat, isResync)
	}

	// Decide which collection to save to based on shouldPlaceInChannel
	var (
		collection2 string
		isGoEnqueue bool = true
	)
	if shouldPlaceInChannel {
		// Message should be in channel, save to normal collection
		collection2 = TalkGroupChannelChatTimestamp2Collection
	} else {
		// Message should not be in channel, save to invalid collection
		collection2 = TalkGroupChannelChatTimestamp2OutCollection
		isGoEnqueue = false
		chat.OutReason = outReason
	}

	// Use ChannelId_Timestamp as primary key to support timestamp range queries
	// For collection2, we need to handle array format
	err = cdb.saveChannelChatTimestamp2WithCollection(chat, collection2, isResync)
	if err != nil {
		return isGoEnqueue, err
	}

	if shouldPlaceInChannel {
		if !isResync {
			go dealGroupChatItem(chat)
		}
	} else {
		cdb.SaveChat(chat)
	}

	return isGoEnqueue, nil
}

// Enqueue channel chat message for asynchronous processing
func (cdb *ChatDB) EnqueueChannelChatMessage(chat *models.TalkGroupChatV3, isResync bool) error {
	queueMessage := &QueueChatMessage{
		PinId:      chat.PinId,
		GroupId:    chat.GroupId,
		ChannelId:  chat.ChannelId,
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

// Save channel chat timestamp (helper method for channel chats)
func (cdb *ChatDB) SaveChannelChatTimestamp(chat *models.TalkGroupChatV3, isResync bool) error {
	//if isResync, frist get chat from database
	if isResync {
		isDuplicate, err := cdb.CheckDuplicatePinIdInTimeRange(chat, TalkGroupChannelChatTimestamp2Collection)
		if err != nil {
			return err
		}
		if isDuplicate {
			//already has index, skip
			return nil
		}
	}

	// Generate a 6-digit random number for uniqueness
	randomNum := generateRandomNumber(6)

	// Construct timestamp index value: pinId_chatType_timestamp
	value := chat.PinId + "_" + strconv.FormatInt(int64(chat.ChatType), 10) + "_" + strconv.FormatInt(chat.Timestamp, 10) + "_" + randomNum

	// Use ChannelId_Timestamp as primary key to support timestamp range queries
	key := []byte(chat.ChannelId + "_" + strconv.FormatInt(chat.Timestamp, 10) + randomNum)
	return Pb[TalkGroupChannelChatTimestamp2Collection].Set(key, []byte(value), pebble.Sync)
}

// Save channel chat timestamp2 with collection (helper method)
func (cdb *ChatDB) saveChannelChatTimestamp2WithCollection(chat *models.TalkGroupChatV3, collection string, isResync bool) error {
	//if isResync, frist get chat from database
	if isResync {
		isDuplicate, err := cdb.CheckDuplicatePinIdInTimeRange(chat, collection)
		if err != nil {
			return err
		}
		if isDuplicate {
			//already has index, skip
			return nil
		}
	}

	// Generate a 6-digit random number for uniqueness
	randomNum := generateRandomNumber(6)

	// Use ChannelId_Timestamp as primary key to support timestamp range queries
	key := []byte(chat.ChannelId + "_" + strconv.FormatInt(chat.Timestamp, 10) + randomNum)

	// For now, we'll use the same logic as group chats but with channel-specific collections
	// You may need to adjust this based on your specific channel timestamp handling requirements
	return Pb[collection].Set(key, []byte(chat.PinId+"_"+strconv.FormatInt(int64(chat.ChatType), 10)+"_"+strconv.FormatInt(chat.Timestamp, 10)+"_"+randomNum), pebble.Sync)
}

// ==================== Channel Chat Index Methods ====================

// UpdateChannelChatIndex updates the chat index for channel chats
func (cdb *ChatDB) UpdateChannelChatIndex(chat *models.TalkGroupChatV3, isResync bool) error {
	//if isResync, frist get chat from database
	if isResync {
		dbChat, err := cdb.GetChatByPinId(chat.PinId)
		if err != nil && err != pebble.ErrNotFound {
			return err
		}
		if dbChat != nil && dbChat.Index != -1 {
			//already has index, skip
			return nil
		}
	}

	// Get the next index for this channel
	nextIndex, err := cdb.getNextChannelChatIndex(chat.ChannelId)
	if err != nil {
		return err
	}

	// Update the chat message with the new index
	chat.Index = nextIndex

	log.Printf("[UpdateChannelChatIndex]nextIndex: %d", nextIndex)

	// Save the updated chat message
	err = cdb.SaveChat(chat)
	if err != nil {
		return err
	}

	// Save the index mapping with zero-padded index for proper sorting
	indexKey := chat.ChannelId + "_" + fmt.Sprintf("%040d", nextIndex)
	indexValue := chat.PinId + "_" + strconv.FormatInt(int64(chat.ChatType), 10) + "_" + strconv.FormatInt(chat.Timestamp, 10) + "_1"

	return Pb[TalkGroupChannelChatIndexCollection].Set([]byte(indexKey), []byte(indexValue), pebble.Sync)
}

// getNextChannelChatIndex gets the next available index for a channel
func (cdb *ChatDB) getNextChannelChatIndex(channelId string) (int64, error) {
	iter, err := Pb[TalkGroupChannelChatIndexCollection].NewIter(&pebble.IterOptions{
		LowerBound: []byte(channelId + "_"),
		UpperBound: []byte(channelId + "_" + string([]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff})),
	})
	if err != nil {
		return 0, err
	}
	defer iter.Close()

	// Find the last key for this channel (since keys are sorted, the last one has the highest index)
	var lastKey []byte

	// Use Last() to get the last key in the range
	if iter.Last(); iter.Valid() {
		lastKey = iter.Key()
	}

	// If no keys found for this channel, start with index 1
	if lastKey == nil {
		return 1, nil
	}

	// Extract index from the last key (channelId_index with zero-padding)
	keyStr := string(lastKey)

	log.Printf("[getNextChannelChatIndex]keyStr: %s", keyStr)
	parts := strings.Split(keyStr, "_")
	if len(parts) >= 2 {
		// Remove leading zeros and parse the index
		indexStr := strings.TrimLeft(parts[1], "0")
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

// GetCurrentMaxChannelChatIndex gets the current maximum index for a channel
func (cdb *ChatDB) GetCurrentMaxChannelChatIndex(channelId string) (int64, error) {
	iter, err := Pb[TalkGroupChannelChatIndexCollection].NewIter(&pebble.IterOptions{
		LowerBound: []byte(channelId + "_"),
		UpperBound: []byte(channelId + "_" + string([]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff})),
	})
	if err != nil {
		return 0, err
	}
	defer iter.Close()

	// Find the last key for this channel (since keys are sorted, the last one has the highest index)
	var lastKey []byte

	// Use Last() to get the last key in the range
	if iter.Last(); iter.Valid() {
		lastKey = iter.Key()
	}

	// If no keys found for this channel, return 0
	if lastKey == nil {
		return 0, nil
	}

	// Extract index from the last key (channelId_index with zero-padding)
	keyStr := string(lastKey)

	log.Printf("[GetCurrentMaxChannelChatIndex]keyStr: %s", keyStr)
	parts := strings.Split(keyStr, "_")
	if len(parts) >= 2 {
		// Remove leading zeros and parse the index
		indexStr := strings.TrimLeft(parts[1], "0")
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

// CheckDuplicatePinIdInTimeRange 检查指定时间戳前1小时内是否存在相同的pinId
// 用于重跑数据时避免重复处理
func (cdb *ChatDB) CheckDuplicatePinIdInTimeRange(chat *models.TalkGroupChatV3, collection string) (bool, error) {
	if chat == nil {
		return false, fmt.Errorf("chat cannot be nil")
	}

	// 计算1小时前的时间戳（秒）
	oneHourAgo := chat.Timestamp - 3600 // 3600秒 = 1小时

	// 构建查询范围：groupId_timestamp
	// 使用LowerBound和UpperBound来限制查询范围，提高性能
	lowerBound := []byte(chat.GroupId + "_" + strconv.FormatInt(oneHourAgo, 10))
	upperBound := []byte(chat.GroupId + "_" + strconv.FormatInt(chat.Timestamp, 10) + "999999") // 添加最大随机数后缀

	// 创建迭代器，使用范围限制
	iter, err := Pb[collection].NewIter(&pebble.IterOptions{
		LowerBound: lowerBound,
		UpperBound: upperBound,
	})
	if err != nil {
		return false, fmt.Errorf("failed to create iterator: %v", err)
	}
	defer iter.Close()

	// 遍历指定范围内的记录
	for iter.First(); iter.Valid(); iter.Next() {
		key := iter.Key()
		value := iter.Value()

		// 解析key: groupId_timestamp+number(6)
		keyStr := string(key)
		parts := strings.Split(keyStr, "_")
		if len(parts) < 2 {
			continue
		}

		// 提取时间戳部分（去掉最后6位随机数）
		timestampStr := parts[1]
		if len(timestampStr) > 6 {
			timestampStr = timestampStr[:len(timestampStr)-6] // 去掉最后6位随机数
		}

		// 解析时间戳
		recordTimestamp, err := strconv.ParseInt(timestampStr, 10, 64)
		if err != nil {
			continue
		}

		// 检查时间戳是否在1小时前到当前时间戳之间
		if recordTimestamp >= oneHourAgo && recordTimestamp <= chat.Timestamp {
			// 解析value: pinId_chatType_timestamp_number
			valueStr := string(value)
			valueParts := strings.Split(valueStr, "_")
			if len(valueParts) >= 1 {
				recordPinId := valueParts[0]

				// 检查pinId是否相同
				if recordPinId == chat.PinId {
					log.Printf("Found duplicate pinId %s in time range [%d, %d] for groupId %s",
						chat.PinId, oneHourAgo, chat.Timestamp, chat.GroupId)
					return true, nil
				}
			}
		}
	}

	// 检查迭代器错误
	if err = iter.Error(); err != nil {
		return false, fmt.Errorf("iterator error: %v", err)
	}

	return false, nil
}
