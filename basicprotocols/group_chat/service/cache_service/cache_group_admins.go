package cache_service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"manindexer/basicprotocols/group_chat/models"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
)

// GroupAdminsCache group admins cache (separate caches for admins, blockList, whitelist)
type GroupAdminsCache struct {
	adminCache     sync.Map      // map[string]*groupAdminCacheItem
	blockCache     sync.Map      // map[string]*groupBlockCacheItem
	whitelistCache sync.Map      // map[string]*groupWhitelistCacheItem
	ttl            time.Duration // cache expiration time
	// independent locks for different groupId, managed by sync.Map
	groupLocks sync.Map // map[string]*sync.RWMutex
}

// groupAdminCacheItem group admin cache item
type groupAdminCacheItem struct {
	AdminList  *models.GroupAdminList `json:"adminList"`
	UpdateTime time.Time              `json:"updateTime"`
	ExpireTime time.Time              `json:"expireTime"`
}

// groupBlockCacheItem group block cache item
type groupBlockCacheItem struct {
	BlockList  *models.GroupBlockList `json:"blockList"`
	UpdateTime time.Time              `json:"updateTime"`
	ExpireTime time.Time              `json:"expireTime"`
}

// groupWhitelistCacheItem group whitelist cache item
type groupWhitelistCacheItem struct {
	Whitelist  *models.GroupWhitelistList `json:"whitelist"`
	UpdateTime time.Time                  `json:"updateTime"`
	ExpireTime time.Time                  `json:"expireTime"`
}

var (
	groupAdminsCache *GroupAdminsCache
)

// InitGroupAdminsCache initialize group admins cache
func InitGroupAdminsCache(ttl time.Duration) {
	groupAdminsCache = &GroupAdminsCache{
		ttl: ttl,
	}
}

// GetGroupAdminsCache get group admins cache instance
func GetGroupAdminsCache() *GroupAdminsCache {
	return groupAdminsCache
}

// getGroupLock get lock for specified groupId
func (gac *GroupAdminsCache) getGroupLock(groupId string) *sync.RWMutex {
	// try to get existing lock
	if lockInterface, exists := gac.groupLocks.Load(groupId); exists {
		return lockInterface.(*sync.RWMutex)
	}

	// create new lock
	lock := &sync.RWMutex{}

	// use LoadOrStore to ensure only one goroutine can create the lock
	if actualLock, loaded := gac.groupLocks.LoadOrStore(groupId, lock); loaded {
		// if other goroutine has already created the lock, use the existing one
		return actualLock.(*sync.RWMutex)
	}

	// we created the new lock
	return lock
}

// ==================== Admin List Cache Operations ====================

// GetAdminList get group admin list from cache
func (gac *GroupAdminsCache) GetAdminList(groupId string) (*models.GroupAdminList, bool) {
	if !initialized {
		return nil, false
	}

	if useRedis && redisClient != nil {
		return gac.getRedisAdminList(groupId)
	} else {
		return gac.getMemoryAdminList(groupId)
	}
}

// SetAdminList set group admin list to cache
func (gac *GroupAdminsCache) SetAdminList(groupId string, adminList *models.GroupAdminList) {
	if !initialized {
		return
	}

	if useRedis && redisClient != nil {
		gac.setRedisAdminList(groupId, adminList)
	} else {
		gac.setMemoryAdminList(groupId, adminList)
	}
}

// DeleteAdminList delete group admin list from cache
func (gac *GroupAdminsCache) DeleteAdminList(groupId string) {
	if !initialized {
		return
	}

	if useRedis && redisClient != nil {
		gac.deleteRedisAdminList(groupId)
	} else {
		gac.deleteMemoryAdminList(groupId)
	}
}

// ==================== Block List Cache Operations ====================

// GetBlockList get group block list from cache
func (gac *GroupAdminsCache) GetBlockList(groupId string) (*models.GroupBlockList, bool) {
	if !initialized {
		return nil, false
	}

	if useRedis && redisClient != nil {
		return gac.getRedisBlockList(groupId)
	} else {
		return gac.getMemoryBlockList(groupId)
	}
}

// SetBlockList set group block list to cache
func (gac *GroupAdminsCache) SetBlockList(groupId string, blockList *models.GroupBlockList) {
	if !initialized {
		return
	}

	if useRedis && redisClient != nil {
		gac.setRedisBlockList(groupId, blockList)
	} else {
		gac.setMemoryBlockList(groupId, blockList)
	}
}

// DeleteBlockList delete group block list from cache
func (gac *GroupAdminsCache) DeleteBlockList(groupId string) {
	if !initialized {
		return
	}

	if useRedis && redisClient != nil {
		gac.deleteRedisBlockList(groupId)
	} else {
		gac.deleteMemoryBlockList(groupId)
	}
}

// ==================== Whitelist Cache Operations ====================

// GetWhitelist get group whitelist from cache
func (gac *GroupAdminsCache) GetWhitelist(groupId string) (*models.GroupWhitelistList, bool) {
	if !initialized {
		return nil, false
	}

	if useRedis && redisClient != nil {
		return gac.getRedisWhitelist(groupId)
	} else {
		return gac.getMemoryWhitelist(groupId)
	}
}

// SetWhitelist set group whitelist to cache
func (gac *GroupAdminsCache) SetWhitelist(groupId string, whitelist *models.GroupWhitelistList) {
	if !initialized {
		return
	}

	if useRedis && redisClient != nil {
		gac.setRedisWhitelist(groupId, whitelist)
	} else {
		gac.setMemoryWhitelist(groupId, whitelist)
	}
}

// DeleteWhitelist delete group whitelist from cache
func (gac *GroupAdminsCache) DeleteWhitelist(groupId string) {
	if !initialized {
		return
	}

	if useRedis && redisClient != nil {
		gac.deleteRedisWhitelist(groupId)
	} else {
		gac.deleteMemoryWhitelist(groupId)
	}
}

// ==================== Combined Operations ====================

// GetAll get all group admins data from cache
func (gac *GroupAdminsCache) GetAll(groupId string) (*models.GroupAdminList, *models.GroupBlockList, *models.GroupWhitelistList, bool, bool, bool) {
	adminList, adminFound := gac.GetAdminList(groupId)
	blockList, blockFound := gac.GetBlockList(groupId)
	whitelist, whitelistFound := gac.GetWhitelist(groupId)

	return adminList, blockList, whitelist, adminFound, blockFound, whitelistFound
}

// SetAll set all group admins data to cache
func (gac *GroupAdminsCache) SetAll(groupId string, adminList *models.GroupAdminList, blockList *models.GroupBlockList, whitelist *models.GroupWhitelistList) {
	if adminList != nil {
		gac.SetAdminList(groupId, adminList)
	}
	if blockList != nil {
		gac.SetBlockList(groupId, blockList)
	}
	if whitelist != nil {
		gac.SetWhitelist(groupId, whitelist)
	}
}

// DeleteAll delete all group admins data from cache
func (gac *GroupAdminsCache) DeleteAll(groupId string) {
	gac.DeleteAdminList(groupId)
	gac.DeleteBlockList(groupId)
	gac.DeleteWhitelist(groupId)
}

// Clear clear all cache
func (gac *GroupAdminsCache) Clear() {
	if !initialized {
		return
	}

	if useRedis && redisClient != nil {
		gac.clearRedisAll()
	} else {
		gac.clearMemoryAll()
	}
}

// Size get cache size
func (gac *GroupAdminsCache) Size() int {
	if !initialized {
		return 0
	}

	if useRedis && redisClient != nil {
		return gac.getRedisSize()
	} else {
		return gac.getMemorySize()
	}
}

// GetStats get cache statistics
func (gac *GroupAdminsCache) GetStats() map[string]interface{} {
	if !initialized {
		return map[string]interface{}{
			"error": "cache service not initialized",
		}
	}

	if useRedis && redisClient != nil {
		return gac.getRedisStats()
	} else {
		return gac.getMemoryStats()
	}
}

// ==================== Redis Cache Operations ====================

// Admin List Redis Operations
func (gac *GroupAdminsCache) getRedisAdminList(groupId string) (*models.GroupAdminList, bool) {
	ctx := context.Background()
	key := fmt.Sprintf("group_admin_list:%s", groupId)

	val, err := redisClient.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, false
		}
		log.Printf("Failed to get group admin list from Redis: %v", err)
		return nil, false
	}

	var cacheItem groupAdminCacheItem
	err = json.Unmarshal([]byte(val), &cacheItem)
	if err != nil {
		log.Printf("Failed to unmarshal group admin list: %v", err)
		return nil, false
	}

	return cacheItem.AdminList, true
}

func (gac *GroupAdminsCache) setRedisAdminList(groupId string, adminList *models.GroupAdminList) {
	ctx := context.Background()
	key := fmt.Sprintf("group_admin_list:%s", groupId)

	// create copy to avoid external modification
	copiedAdminList := gac.copyAdminList(adminList)

	// create cache item
	now := time.Now()
	cacheItem := &groupAdminCacheItem{
		AdminList:  copiedAdminList,
		UpdateTime: now,
		ExpireTime: now.Add(gac.ttl),
	}

	data, err := json.Marshal(cacheItem)
	if err != nil {
		log.Printf("Failed to marshal group admin list: %v", err)
		return
	}

	err = redisClient.Set(ctx, key, data, gac.ttl).Err()
	if err != nil {
		log.Printf("Failed to set group admin list to Redis: %v", err)
	}
}

func (gac *GroupAdminsCache) deleteRedisAdminList(groupId string) {
	ctx := context.Background()
	key := fmt.Sprintf("group_admin_list:%s", groupId)

	err := redisClient.Del(ctx, key).Err()
	if err != nil {
		log.Printf("Failed to delete group admin list from Redis: %v", err)
	}
}

// Block List Redis Operations
func (gac *GroupAdminsCache) getRedisBlockList(groupId string) (*models.GroupBlockList, bool) {
	ctx := context.Background()
	key := fmt.Sprintf("group_block_list:%s", groupId)

	val, err := redisClient.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, false
		}
		log.Printf("Failed to get group block list from Redis: %v", err)
		return nil, false
	}

	var cacheItem groupBlockCacheItem
	err = json.Unmarshal([]byte(val), &cacheItem)
	if err != nil {
		log.Printf("Failed to unmarshal group block list: %v", err)
		return nil, false
	}

	return cacheItem.BlockList, true
}

func (gac *GroupAdminsCache) setRedisBlockList(groupId string, blockList *models.GroupBlockList) {
	ctx := context.Background()
	key := fmt.Sprintf("group_block_list:%s", groupId)

	// create copy to avoid external modification
	copiedBlockList := gac.copyBlockList(blockList)

	// create cache item
	now := time.Now()
	cacheItem := &groupBlockCacheItem{
		BlockList:  copiedBlockList,
		UpdateTime: now,
		ExpireTime: now.Add(gac.ttl),
	}

	data, err := json.Marshal(cacheItem)
	if err != nil {
		log.Printf("Failed to marshal group block list: %v", err)
		return
	}

	err = redisClient.Set(ctx, key, data, gac.ttl).Err()
	if err != nil {
		log.Printf("Failed to set group block list to Redis: %v", err)
	}
}

func (gac *GroupAdminsCache) deleteRedisBlockList(groupId string) {
	ctx := context.Background()
	key := fmt.Sprintf("group_block_list:%s", groupId)

	err := redisClient.Del(ctx, key).Err()
	if err != nil {
		log.Printf("Failed to delete group block list from Redis: %v", err)
	}
}

// Whitelist Redis Operations
func (gac *GroupAdminsCache) getRedisWhitelist(groupId string) (*models.GroupWhitelistList, bool) {
	ctx := context.Background()
	key := fmt.Sprintf("group_whitelist:%s", groupId)

	val, err := redisClient.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, false
		}
		log.Printf("Failed to get group whitelist from Redis: %v", err)
		return nil, false
	}

	var cacheItem groupWhitelistCacheItem
	err = json.Unmarshal([]byte(val), &cacheItem)
	if err != nil {
		log.Printf("Failed to unmarshal group whitelist: %v", err)
		return nil, false
	}

	return cacheItem.Whitelist, true
}

func (gac *GroupAdminsCache) setRedisWhitelist(groupId string, whitelist *models.GroupWhitelistList) {
	ctx := context.Background()
	key := fmt.Sprintf("group_whitelist:%s", groupId)

	// create copy to avoid external modification
	copiedWhitelist := gac.copyWhitelist(whitelist)

	// create cache item
	now := time.Now()
	cacheItem := &groupWhitelistCacheItem{
		Whitelist:  copiedWhitelist,
		UpdateTime: now,
		ExpireTime: now.Add(gac.ttl),
	}

	data, err := json.Marshal(cacheItem)
	if err != nil {
		log.Printf("Failed to marshal group whitelist: %v", err)
		return
	}

	err = redisClient.Set(ctx, key, data, gac.ttl).Err()
	if err != nil {
		log.Printf("Failed to set group whitelist to Redis: %v", err)
	}
}

func (gac *GroupAdminsCache) deleteRedisWhitelist(groupId string) {
	ctx := context.Background()
	key := fmt.Sprintf("group_whitelist:%s", groupId)

	err := redisClient.Del(ctx, key).Err()
	if err != nil {
		log.Printf("Failed to delete group whitelist from Redis: %v", err)
	}
}

// Combined Redis Operations
func (gac *GroupAdminsCache) clearRedisAll() {
	ctx := context.Background()
	patterns := []string{"group_admin_list:*", "group_block_list:*", "group_whitelist:*"}

	for _, pattern := range patterns {
		keys, err := redisClient.Keys(ctx, pattern).Result()
		if err != nil {
			log.Printf("Failed to get keys from Redis: %v", err)
			continue
		}

		if len(keys) > 0 {
			err = redisClient.Del(ctx, keys...).Err()
			if err != nil {
				log.Printf("Failed to clear %s from Redis: %v", pattern, err)
			}
		}
	}
}

func (gac *GroupAdminsCache) getRedisSize() int {
	ctx := context.Background()
	patterns := []string{"group_admin_list:*", "group_block_list:*", "group_whitelist:*"}
	totalSize := 0

	for _, pattern := range patterns {
		keys, err := redisClient.Keys(ctx, pattern).Result()
		if err != nil {
			log.Printf("Failed to get keys from Redis: %v", err)
			continue
		}
		totalSize += len(keys)
	}

	return totalSize
}

func (gac *GroupAdminsCache) getRedisStats() map[string]interface{} {
	ctx := context.Background()
	patterns := []string{"group_admin_list:*", "group_block_list:*", "group_whitelist:*"}
	stats := map[string]interface{}{
		"cache_type": "redis",
		"ttl":        gac.ttl.String(),
		"details":    make(map[string]int),
	}

	totalSize := 0
	for _, pattern := range patterns {
		keys, err := redisClient.Keys(ctx, pattern).Result()
		if err != nil {
			log.Printf("Failed to get keys from Redis: %v", err)
			continue
		}
		keyCount := len(keys)
		totalSize += keyCount
		stats["details"].(map[string]int)[pattern] = keyCount
	}

	stats["cache_size"] = totalSize
	return stats
}

// ==================== Memory Cache Operations ====================

// Admin List Memory Operations
func (gac *GroupAdminsCache) getMemoryAdminList(groupId string) (*models.GroupAdminList, bool) {
	lock := gac.getGroupLock(groupId)
	lock.RLock()
	defer lock.RUnlock()

	if cacheItemInterface, exists := gac.adminCache.Load(groupId); exists {
		if cacheItem, ok := cacheItemInterface.(*groupAdminCacheItem); ok {
			// check if expired
			if time.Now().Before(cacheItem.ExpireTime) {
				return cacheItem.AdminList, true
			} else {
				// expired, delete cache item
				gac.adminCache.Delete(groupId)
			}
		}
	}
	return nil, false
}

func (gac *GroupAdminsCache) setMemoryAdminList(groupId string, adminList *models.GroupAdminList) {
	lock := gac.getGroupLock(groupId)
	lock.Lock()
	defer lock.Unlock()

	// create copy to avoid external modification
	copiedAdminList := gac.copyAdminList(adminList)

	// create cache item
	now := time.Now()
	cacheItem := &groupAdminCacheItem{
		AdminList:  copiedAdminList,
		UpdateTime: now,
		ExpireTime: now.Add(gac.ttl),
	}

	gac.adminCache.Store(groupId, cacheItem)
}

func (gac *GroupAdminsCache) deleteMemoryAdminList(groupId string) {
	lock := gac.getGroupLock(groupId)
	lock.Lock()
	defer lock.Unlock()

	gac.adminCache.Delete(groupId)
}

// Block List Memory Operations
func (gac *GroupAdminsCache) getMemoryBlockList(groupId string) (*models.GroupBlockList, bool) {
	lock := gac.getGroupLock(groupId)
	lock.RLock()
	defer lock.RUnlock()

	if cacheItemInterface, exists := gac.blockCache.Load(groupId); exists {
		if cacheItem, ok := cacheItemInterface.(*groupBlockCacheItem); ok {
			// check if expired
			if time.Now().Before(cacheItem.ExpireTime) {
				return cacheItem.BlockList, true
			} else {
				// expired, delete cache item
				gac.blockCache.Delete(groupId)
			}
		}
	}
	return nil, false
}

func (gac *GroupAdminsCache) setMemoryBlockList(groupId string, blockList *models.GroupBlockList) {
	lock := gac.getGroupLock(groupId)
	lock.Lock()
	defer lock.Unlock()

	// create copy to avoid external modification
	copiedBlockList := gac.copyBlockList(blockList)

	// create cache item
	now := time.Now()
	cacheItem := &groupBlockCacheItem{
		BlockList:  copiedBlockList,
		UpdateTime: now,
		ExpireTime: now.Add(gac.ttl),
	}

	gac.blockCache.Store(groupId, cacheItem)
}

func (gac *GroupAdminsCache) deleteMemoryBlockList(groupId string) {
	lock := gac.getGroupLock(groupId)
	lock.Lock()
	defer lock.Unlock()

	gac.blockCache.Delete(groupId)
}

// Whitelist Memory Operations
func (gac *GroupAdminsCache) getMemoryWhitelist(groupId string) (*models.GroupWhitelistList, bool) {
	lock := gac.getGroupLock(groupId)
	lock.RLock()
	defer lock.RUnlock()

	if cacheItemInterface, exists := gac.whitelistCache.Load(groupId); exists {
		if cacheItem, ok := cacheItemInterface.(*groupWhitelistCacheItem); ok {
			// check if expired
			if time.Now().Before(cacheItem.ExpireTime) {
				return cacheItem.Whitelist, true
			} else {
				// expired, delete cache item
				gac.whitelistCache.Delete(groupId)
			}
		}
	}
	return nil, false
}

func (gac *GroupAdminsCache) setMemoryWhitelist(groupId string, whitelist *models.GroupWhitelistList) {
	lock := gac.getGroupLock(groupId)
	lock.Lock()
	defer lock.Unlock()

	// create copy to avoid external modification
	copiedWhitelist := gac.copyWhitelist(whitelist)

	// create cache item
	now := time.Now()
	cacheItem := &groupWhitelistCacheItem{
		Whitelist:  copiedWhitelist,
		UpdateTime: now,
		ExpireTime: now.Add(gac.ttl),
	}

	gac.whitelistCache.Store(groupId, cacheItem)
}

func (gac *GroupAdminsCache) deleteMemoryWhitelist(groupId string) {
	lock := gac.getGroupLock(groupId)
	lock.Lock()
	defer lock.Unlock()

	gac.whitelistCache.Delete(groupId)
}

// Combined Memory Operations
func (gac *GroupAdminsCache) clearMemoryAll() {
	// iterate all keys and delete
	gac.adminCache.Range(func(key, value interface{}) bool {
		gac.adminCache.Delete(key)
		return true
	})
	gac.blockCache.Range(func(key, value interface{}) bool {
		gac.blockCache.Delete(key)
		return true
	})
	gac.whitelistCache.Range(func(key, value interface{}) bool {
		gac.whitelistCache.Delete(key)
		return true
	})
}

func (gac *GroupAdminsCache) getMemorySize() int {
	count := 0
	gac.adminCache.Range(func(key, value interface{}) bool {
		count++
		return true
	})
	gac.blockCache.Range(func(key, value interface{}) bool {
		count++
		return true
	})
	gac.whitelistCache.Range(func(key, value interface{}) bool {
		count++
		return true
	})
	return count
}

func (gac *GroupAdminsCache) getMemoryStats() map[string]interface{} {
	now := time.Now()
	stats := map[string]interface{}{
		"cache_type": "memory",
		"ttl":        gac.ttl.String(),
		"details":    make(map[string]interface{}),
	}

	// Admin cache stats
	adminKeys := make([]string, 0)
	adminExpiredCount := 0
	gac.adminCache.Range(func(key, value interface{}) bool {
		adminKeys = append(adminKeys, key.(string))
		if cacheItem, ok := value.(*groupAdminCacheItem); ok {
			if now.After(cacheItem.ExpireTime) {
				adminExpiredCount++
			}
		}
		return true
	})

	// Block cache stats
	blockKeys := make([]string, 0)
	blockExpiredCount := 0
	gac.blockCache.Range(func(key, value interface{}) bool {
		blockKeys = append(blockKeys, key.(string))
		if cacheItem, ok := value.(*groupBlockCacheItem); ok {
			if now.After(cacheItem.ExpireTime) {
				blockExpiredCount++
			}
		}
		return true
	})

	// Whitelist cache stats
	whitelistKeys := make([]string, 0)
	whitelistExpiredCount := 0
	gac.whitelistCache.Range(func(key, value interface{}) bool {
		whitelistKeys = append(whitelistKeys, key.(string))
		if cacheItem, ok := value.(*groupWhitelistCacheItem); ok {
			if now.After(cacheItem.ExpireTime) {
				whitelistExpiredCount++
			}
		}
		return true
	})

	totalSize := len(adminKeys) + len(blockKeys) + len(whitelistKeys)
	totalExpiredCount := adminExpiredCount + blockExpiredCount + whitelistExpiredCount

	stats["cache_size"] = totalSize
	stats["expired_count"] = totalExpiredCount
	stats["details"].(map[string]interface{})["admin_list"] = map[string]interface{}{
		"count":         len(adminKeys),
		"expired_count": adminExpiredCount,
		"keys":          adminKeys,
	}
	stats["details"].(map[string]interface{})["block_list"] = map[string]interface{}{
		"count":         len(blockKeys),
		"expired_count": blockExpiredCount,
		"keys":          blockKeys,
	}
	stats["details"].(map[string]interface{})["whitelist"] = map[string]interface{}{
		"count":         len(whitelistKeys),
		"expired_count": whitelistExpiredCount,
		"keys":          whitelistKeys,
	}

	return stats
}

// ==================== Helper Methods ====================

// copyAdminList create copy of group admin list
func (gac *GroupAdminsCache) copyAdminList(adminList *models.GroupAdminList) *models.GroupAdminList {
	if adminList == nil {
		return nil
	}

	copiedList := &models.GroupAdminList{
		GroupId: adminList.GroupId,
		Items:   make([]*models.GroupAdminItem, len(adminList.Items)),
	}

	for i, item := range adminList.Items {
		copiedItem := &models.GroupAdminItem{
			AdminPinId:     item.AdminPinId,
			AdminType:      item.AdminType,
			AdminTimestamp: item.AdminTimestamp,
			Admins:         make([]string, len(item.Admins)),
			SetByMetaId:    item.SetByMetaId,
			SetByAddress:   item.SetByAddress,
			BlockHeight:    item.BlockHeight,
			Chain:          item.Chain,
		}
		copy(copiedItem.Admins, item.Admins)
		copiedList.Items[i] = copiedItem
	}

	return copiedList
}

// copyBlockList create copy of group block list
func (gac *GroupAdminsCache) copyBlockList(blockList *models.GroupBlockList) *models.GroupBlockList {
	if blockList == nil {
		return nil
	}

	copiedList := &models.GroupBlockList{
		GroupId: blockList.GroupId,
		Items:   make([]*models.GroupBlockItem, len(blockList.Items)),
	}

	for i, item := range blockList.Items {
		copiedItem := &models.GroupBlockItem{
			BlockPinId:     item.BlockPinId,
			BlockType:      item.BlockType,
			BlockTimestamp: item.BlockTimestamp,
			BlockedUsers:   make([]string, len(item.BlockedUsers)),
			SetByMetaId:    item.SetByMetaId,
			SetByAddress:   item.SetByAddress,
			BlockHeight:    item.BlockHeight,
			Chain:          item.Chain,
		}
		copy(copiedItem.BlockedUsers, item.BlockedUsers)
		copiedList.Items[i] = copiedItem
	}

	return copiedList
}

// copyWhitelist create copy of group whitelist
func (gac *GroupAdminsCache) copyWhitelist(whitelist *models.GroupWhitelistList) *models.GroupWhitelistList {
	if whitelist == nil {
		return nil
	}

	copiedList := &models.GroupWhitelistList{
		GroupId: whitelist.GroupId,
		Items:   make([]*models.GroupWhitelistItem, len(whitelist.Items)),
	}

	for i, item := range whitelist.Items {
		copiedItem := &models.GroupWhitelistItem{
			WhitelistPinId:     item.WhitelistPinId,
			WhitelistType:      item.WhitelistType,
			WhitelistTimestamp: item.WhitelistTimestamp,
			WhitelistUsers:     make([]string, len(item.WhitelistUsers)),
			SetByMetaId:        item.SetByMetaId,
			SetByAddress:       item.SetByAddress,
			BlockHeight:        item.BlockHeight,
			Chain:              item.Chain,
		}
		copy(copiedItem.WhitelistUsers, item.WhitelistUsers)
		copiedList.Items[i] = copiedItem
	}

	return copiedList
}

// ==================== Convenience Methods ====================

// Individual convenience methods
func GetGroupAdminListFromCache(groupId string) (*models.GroupAdminList, bool) {
	cache := GetGroupAdminsCache()
	if cache == nil {
		return nil, false
	}
	return cache.GetAdminList(groupId)
}

func SetGroupAdminListToCache(groupId string, adminList *models.GroupAdminList) {
	cache := GetGroupAdminsCache()
	if cache == nil {
		return
	}
	cache.SetAdminList(groupId, adminList)
}

func GetGroupBlockListFromCache(groupId string) (*models.GroupBlockList, bool) {
	cache := GetGroupAdminsCache()
	if cache == nil {
		return nil, false
	}
	return cache.GetBlockList(groupId)
}

func SetGroupBlockListToCache(groupId string, blockList *models.GroupBlockList) {
	cache := GetGroupAdminsCache()
	if cache == nil {
		return
	}
	cache.SetBlockList(groupId, blockList)
}

func GetGroupWhitelistFromCache(groupId string) (*models.GroupWhitelistList, bool) {
	cache := GetGroupAdminsCache()
	if cache == nil {
		return nil, false
	}
	return cache.GetWhitelist(groupId)
}

func SetGroupWhitelistToCache(groupId string, whitelist *models.GroupWhitelistList) {
	cache := GetGroupAdminsCache()
	if cache == nil {
		return
	}
	cache.SetWhitelist(groupId, whitelist)
}

// Combined convenience methods
func GetGroupAdminsFromCache(groupId string) (*models.GroupAdminList, *models.GroupBlockList, *models.GroupWhitelistList, bool, bool, bool) {
	cache := GetGroupAdminsCache()
	if cache == nil {
		return nil, nil, nil, false, false, false
	}
	return cache.GetAll(groupId)
}

func SetGroupAdminsToCache(groupId string, adminList *models.GroupAdminList, blockList *models.GroupBlockList, whitelist *models.GroupWhitelistList) {
	cache := GetGroupAdminsCache()
	if cache == nil {
		return
	}
	cache.SetAll(groupId, adminList, blockList, whitelist)
}

func DeleteGroupAdminsFromCache(groupId string) {
	cache := GetGroupAdminsCache()
	if cache == nil {
		return
	}
	cache.DeleteAll(groupId)
}

func GetGroupAdminsCacheStats() map[string]interface{} {
	cache := GetGroupAdminsCache()
	if cache == nil {
		return map[string]interface{}{
			"error": "cache not initialized",
		}
	}
	return cache.GetStats()
}

func CleanExpiredGroupAdminsCache() {
	cache := GetGroupAdminsCache()
	if cache == nil {
		return
	}

	now := time.Now()

	// clean expired cache items for each type
	cache.adminCache.Range(func(key, value interface{}) bool {
		if cacheItem, ok := value.(*groupAdminCacheItem); ok {
			if now.After(cacheItem.ExpireTime) {
				cache.adminCache.Delete(key)
			}
		}
		return true
	})

	cache.blockCache.Range(func(key, value interface{}) bool {
		if cacheItem, ok := value.(*groupBlockCacheItem); ok {
			if now.After(cacheItem.ExpireTime) {
				cache.blockCache.Delete(key)
			}
		}
		return true
	})

	cache.whitelistCache.Range(func(key, value interface{}) bool {
		if cacheItem, ok := value.(*groupWhitelistCacheItem); ok {
			if now.After(cacheItem.ExpireTime) {
				cache.whitelistCache.Delete(key)
			}
		}
		return true
	})
}
