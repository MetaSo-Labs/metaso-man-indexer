package db

import (
	"encoding/json"
	"fmt"
	"manindexer/basicprotocols/group_chat/service/cache_service"
	"strings"
	"time"

	"github.com/cockroachdb/pebble"
)

// GlobalBlockDB Global blocklist database operations
type GlobalBlockDB struct {
	pb *Pebble
}

// NewGlobalBlockDB Create global blocklist database instance
func NewGlobalBlockDB(pb *Pebble) *GlobalBlockDB {
	return &GlobalBlockDB{
		pb: pb,
	}
}

// GlobalBlockItem Global blocklist item
type GlobalBlockItem struct {
	Address string `json:"address"` // Blocked address
	Reason  string `json:"reason"`  // Block reason
}

// SaveGlobalBlockAddress Save global blocklist address
func (gbdb *GlobalBlockDB) SaveGlobalBlockAddress(address, reason string) error {
	// Validate address format
	if address == "" {
		return fmt.Errorf("address cannot be empty")
	}

	// Create blocklist item
	blockItem := &GlobalBlockItem{
		Address: address,
		Reason:  reason,
	}

	// Serialize data
	data, err := json.Marshal(blockItem)
	if err != nil {
		return fmt.Errorf("failed to marshal global block item: %v", err)
	}

	// Save to database using address as key
	key := []byte(address)
	err = Pb[TalkGolbalBlockCollection].Set(key, data, pebble.Sync)
	if err != nil {
		return fmt.Errorf("failed to save global block address: %v", err)
	}

	// Update cache
	cache_service.SetGlobalBlockAddressToCache(address, true)

	return nil
}

// DeleteGlobalBlockAddress Delete global blocklist address
func (gbdb *GlobalBlockDB) DeleteGlobalBlockAddress(address string) error {
	// Validate address format
	if address == "" {
		return fmt.Errorf("address cannot be empty")
	}

	// Delete from database
	key := []byte(address)
	err := Pb[TalkGolbalBlockCollection].Delete(key, pebble.Sync)
	if err != nil {
		return fmt.Errorf("failed to delete global block address: %v", err)
	}

	// Update cache
	cache_service.SetGlobalBlockAddressToCache(address, false)

	return nil
}

// GetGlobalBlockAddress Get blocklist information for specified address
func (gbdb *GlobalBlockDB) GetGlobalBlockAddress(address string) (*GlobalBlockItem, error) {
	// Validate address format
	if address == "" {
		return nil, fmt.Errorf("address cannot be empty")
	}

	// Get from database
	key := []byte(address)
	data, closer, err := Pb[TalkGolbalBlockCollection].Get(key)
	if err != nil {
		if err == pebble.ErrNotFound {
			return nil, nil // Address not in blocklist
		}
		return nil, fmt.Errorf("failed to get global block address: %v", err)
	}
	defer closer.Close()

	// Deserialize data
	var blockItem GlobalBlockItem
	err = json.Unmarshal(data, &blockItem)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal global block item: %v", err)
	}

	return &blockItem, nil
}

// IsAddressGloballyBlocked Check if address is in global blocklist
func (gbdb *GlobalBlockDB) IsAddressGloballyBlocked(address string) (bool, error) {
	// Validate address format
	if address == "" {
		return false, fmt.Errorf("address cannot be empty")
	}

	// Prioritize getting from cache
	if isBlocked, found := cache_service.IsAddressGloballyBlockedFromCache(address); found {
		return isBlocked, nil
	}

	// Cache miss, get from database
	blockItem, err := gbdb.GetGlobalBlockAddress(address)
	if err != nil {
		return false, err
	}

	isBlocked := blockItem != nil

	// Update cache
	cache_service.SetGlobalBlockAddressToCache(address, isBlocked)

	return isBlocked, nil
}

// GetAllGlobalBlockAddresses Get all global blocklist addresses
func (gbdb *GlobalBlockDB) GetAllGlobalBlockAddresses() ([]*GlobalBlockItem, error) {
	var blockItems []*GlobalBlockItem

	// Create iterator
	iter, err := Pb[TalkGolbalBlockCollection].NewIter(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create iterator: %v", err)
	}
	defer iter.Close()

	// Iterate through all records
	for iter.First(); iter.Valid(); iter.Next() {
		// Get data
		data := iter.Value()

		// Deserialize data
		var blockItem GlobalBlockItem
		err = json.Unmarshal(data, &blockItem)
		if err != nil {
			// Log error but continue processing other records
			fmt.Printf("Warning: failed to unmarshal global block item for key %s: %v\n", string(iter.Key()), err)
			continue
		}

		blockItems = append(blockItems, &blockItem)
	}

	// Check iterator error
	if err = iter.Error(); err != nil {
		return nil, fmt.Errorf("iterator error: %v", err)
	}

	return blockItems, nil
}

// GetGlobalBlockAddressesByReason Get global blocklist addresses by block reason
func (gbdb *GlobalBlockDB) GetGlobalBlockAddressesByReason(reason string) ([]*GlobalBlockItem, error) {
	var blockItems []*GlobalBlockItem

	// Get all blocklist addresses
	allItems, err := gbdb.GetAllGlobalBlockAddresses()
	if err != nil {
		return nil, err
	}

	// Filter addresses by specified reason
	for _, item := range allItems {
		if strings.Contains(strings.ToLower(item.Reason), strings.ToLower(reason)) {
			blockItems = append(blockItems, item)
		}
	}

	return blockItems, nil
}

// GetGlobalBlockAddressCount Get total count of global blocklist addresses
func (gbdb *GlobalBlockDB) GetGlobalBlockAddressCount() (int64, error) {
	count := int64(0)

	// Create iterator
	iter, err := Pb[TalkGolbalBlockCollection].NewIter(nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create iterator: %v", err)
	}
	defer iter.Close()

	// Iterate through all records并计数
	for iter.First(); iter.Valid(); iter.Next() {
		count++
	}

	// Check iterator error
	if err = iter.Error(); err != nil {
		return 0, fmt.Errorf("iterator error: %v", err)
	}

	return count, nil
}

// ClearAllGlobalBlockAddresses Clear all global blocklist addresses (use with caution)
func (gbdb *GlobalBlockDB) ClearAllGlobalBlockAddresses() error {
	// Create iterator
	iter, err := Pb[TalkGolbalBlockCollection].NewIter(nil)
	if err != nil {
		return fmt.Errorf("failed to create iterator: %v", err)
	}
	defer iter.Close()

	// Iterate through all records并删除
	for iter.First(); iter.Valid(); iter.Next() {
		key := iter.Key()
		err = Pb[TalkGolbalBlockCollection].Delete(key, pebble.Sync)
		if err != nil {
			return fmt.Errorf("failed to delete global block address %s: %v", string(key), err)
		}
	}

	// Check iterator error
	if err = iter.Error(); err != nil {
		return fmt.Errorf("iterator error: %v", err)
	}

	return nil
}

// GetGlobalBlockStats Get global blocklist statistics
func (gbdb *GlobalBlockDB) GetGlobalBlockStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Get total count
	totalCount, err := gbdb.GetGlobalBlockAddressCount()
	if err != nil {
		return nil, fmt.Errorf("failed to get total count: %v", err)
	}
	stats["totalCount"] = totalCount

	// Get all addresses
	allItems, err := gbdb.GetAllGlobalBlockAddresses()
	if err != nil {
		return nil, fmt.Errorf("failed to get all addresses: %v", err)
	}

	// Count by chain
	chainStats := make(map[string]int64)
	reasonStats := make(map[string]int64)

	for _, item := range allItems {

		// Count block reasons
		reasonStats[item.Reason]++
	}

	stats["allItems"] = allItems
	stats["chainStats"] = chainStats
	stats["reasonStats"] = reasonStats
	stats["lastUpdateTime"] = time.Now().UnixMilli()

	return stats, nil
}

// SaveGlobalLuckBagBlockAddress Save global luck bag blocklist address
func (gbdb *GlobalBlockDB) SaveGlobalLuckBagBlockAddress(address, reason string) error {
	// Validate address format
	if address == "" {
		return fmt.Errorf("address cannot be empty")
	}

	// Create blocklist item
	blockItem := &GlobalBlockItem{
		Address: address,
		Reason:  reason,
	}

	// Serialize data
	data, err := json.Marshal(blockItem)
	if err != nil {
		return fmt.Errorf("failed to marshal global luck bag block item: %v", err)
	}

	// Save to database using address as key
	key := []byte(address)
	err = Pb[TalkGolbalLuckBagBlockCollection].Set(key, data, pebble.Sync)
	if err != nil {
		return fmt.Errorf("failed to save global luck bag block address: %v", err)
	}

	// Update cache
	cache_service.SetGlobalLuckBagBlockAddressToCache(address, true)

	return nil
}

// DeleteGlobalLuckBagBlockAddress Delete global luck bag blocklist address
func (gbdb *GlobalBlockDB) DeleteGlobalLuckBagBlockAddress(address string) error {
	// Validate address format
	if address == "" {
		return fmt.Errorf("address cannot be empty")
	}

	// Delete from database
	key := []byte(address)
	err := Pb[TalkGolbalLuckBagBlockCollection].Delete(key, pebble.Sync)
	if err != nil {
		return fmt.Errorf("failed to delete global luck bag block address: %v", err)
	}

	// Update cache
	cache_service.SetGlobalLuckBagBlockAddressToCache(address, false)

	return nil
}

// GetGlobalLuckBagBlockAddress Get luck bag blocklist information for specified address
func (gbdb *GlobalBlockDB) GetGlobalLuckBagBlockAddress(address string) (*GlobalBlockItem, error) {
	// Validate address format
	if address == "" {
		return nil, fmt.Errorf("address cannot be empty")
	}

	// Get from database
	key := []byte(address)
	data, closer, err := Pb[TalkGolbalLuckBagBlockCollection].Get(key)
	if err != nil {
		if err == pebble.ErrNotFound {
			return nil, nil // Address not in luck bag blocklist
		}
		return nil, fmt.Errorf("failed to get global luck bag block address: %v", err)
	}
	defer closer.Close()

	// Deserialize data
	var blockItem GlobalBlockItem
	err = json.Unmarshal(data, &blockItem)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal global luck bag block item: %v", err)
	}

	return &blockItem, nil
}

// IsAddressGloballyLuckBagBlocked Check if address is in global luck bag blocklist
func (gbdb *GlobalBlockDB) IsAddressGloballyLuckBagBlocked(address string) (bool, error) {
	// Validate address format
	if address == "" {
		return false, fmt.Errorf("address cannot be empty")
	}

	// Prioritize getting from cache
	if isBlocked, found := cache_service.IsAddressGloballyLuckBagBlockedFromCache(address); found {
		return isBlocked, nil
	}

	// Cache miss, get from database
	blockItem, err := gbdb.GetGlobalLuckBagBlockAddress(address)
	if err != nil {
		return false, err
	}

	isBlocked := blockItem != nil

	// Update cache
	cache_service.SetGlobalLuckBagBlockAddressToCache(address, isBlocked)

	return isBlocked, nil
}

// GetAllGlobalLuckBagBlockAddresses Get all global luck bag blocklist addresses
func (gbdb *GlobalBlockDB) GetAllGlobalLuckBagBlockAddresses() ([]*GlobalBlockItem, error) {
	var blockItems []*GlobalBlockItem

	// Create iterator
	iter, err := Pb[TalkGolbalLuckBagBlockCollection].NewIter(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create iterator: %v", err)
	}
	defer iter.Close()

	// Iterate through all records
	for iter.First(); iter.Valid(); iter.Next() {
		// Get data
		data := iter.Value()

		// Deserialize data
		var blockItem GlobalBlockItem
		err = json.Unmarshal(data, &blockItem)
		if err != nil {
			// Log error but continue processing other records
			fmt.Printf("Warning: failed to unmarshal global luck bag block item for key %s: %v\n", string(iter.Key()), err)
			continue
		}

		blockItems = append(blockItems, &blockItem)
	}

	// Check iterator error
	if err = iter.Error(); err != nil {
		return nil, fmt.Errorf("iterator error: %v", err)
	}

	return blockItems, nil
}

// GetGlobalLuckBagBlockStats Get global luck bag blocklist statistics
func (gbdb *GlobalBlockDB) GetGlobalLuckBagBlockStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Get all addresses
	allItems, err := gbdb.GetAllGlobalLuckBagBlockAddresses()
	if err != nil {
		return nil, fmt.Errorf("failed to get all luck bag block addresses: %v", err)
	}

	totalCount := int64(len(allItems))
	stats["totalCount"] = totalCount

	// Count by reason
	reasonStats := make(map[string]int64)
	for _, item := range allItems {
		reasonStats[item.Reason]++
	}

	stats["allItems"] = allItems
	stats["reasonStats"] = reasonStats
	stats["lastUpdateTime"] = time.Now().UnixMilli()

	return stats, nil
}
