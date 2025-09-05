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

// GroupMemberCache group member cache
type GroupMemberCache struct {
	memoryCache sync.Map      //
	ttl         time.Duration // cache expiration time
	// independent locks for different groupId, managed by sync.Map
	groupLocks sync.Map // map[string]*sync.RWMutex
}

// groupMemberCacheItem group member cache item
type groupMemberCacheItem struct {
	PersonList *models.TalkGroupPersonList `json:"personList"`
	UpdateTime time.Time                   `json:"updateTime"`
	ExpireTime time.Time                   `json:"expireTime"`
}

var (
	groupMemberCache *GroupMemberCache
)

// InitGroupMemberCache initialize group member cache
func InitGroupMemberCache(ttl time.Duration) {
	groupMemberCache = &GroupMemberCache{
		ttl: ttl,
	}
}

// GetGroupMemberCache get group member cache instance
func GetGroupMemberCache() *GroupMemberCache {
	return groupMemberCache
}

// getGroupLock get lock for specified groupId
func (gmc *GroupMemberCache) getGroupLock(groupId string) *sync.RWMutex {
	// try to get existing lock
	if lockInterface, exists := gmc.groupLocks.Load(groupId); exists {
		return lockInterface.(*sync.RWMutex)
	}

	// create new lock
	lock := &sync.RWMutex{}

	// use LoadOrStore to ensure only one goroutine can create the lock
	if actualLock, loaded := gmc.groupLocks.LoadOrStore(groupId, lock); loaded {
		// if other goroutine has already created the lock, use the existing one
		return actualLock.(*sync.RWMutex)
	}

	// we created the new lock
	return lock
}

// Get get group member list from cache
func (gmc *GroupMemberCache) Get(groupId string) (*models.TalkGroupPersonList, bool) {
	if !initialized {
		return nil, false
	}

	if useRedis && redisClient != nil {
		return gmc.getRedisGroupMemberList(groupId)
	} else {
		return gmc.getMemoryGroupMemberList(groupId)
	}
}

// Set set group member list to cache
func (gmc *GroupMemberCache) Set(groupId string, personList *models.TalkGroupPersonList) {
	if !initialized {
		return
	}

	if useRedis && redisClient != nil {
		gmc.setRedisGroupMemberList(groupId, personList)
	} else {
		gmc.setMemoryGroupMemberList(groupId, personList)
	}
}

// Delete delete group member list from cache
func (gmc *GroupMemberCache) Delete(groupId string) {
	if !initialized {
		return
	}

	if useRedis && redisClient != nil {
		gmc.deleteRedisGroupMemberList(groupId)
	} else {
		gmc.deleteMemoryGroupMemberList(groupId)
	}
}

// Clear clear all cache
func (gmc *GroupMemberCache) Clear() {
	if !initialized {
		return
	}

	if useRedis && redisClient != nil {
		gmc.clearRedisGroupMemberList()
	} else {
		gmc.clearMemoryGroupMemberList()
	}
}

// Size get cache size
func (gmc *GroupMemberCache) Size() int {
	if !initialized {
		return 0
	}

	if useRedis && redisClient != nil {
		return gmc.getRedisGroupMemberListSize()
	} else {
		return gmc.getMemoryGroupMemberListSize()
	}
}

// GetStats get cache statistics
func (gmc *GroupMemberCache) GetStats() map[string]interface{} {
	if !initialized {
		return map[string]interface{}{
			"error": "cache service not initialized",
		}
	}

	if useRedis && redisClient != nil {
		return gmc.getRedisGroupMemberListStats()
	} else {
		return gmc.getMemoryGroupMemberListStats()
	}
}

// ==================== Redis Cache Operations ====================

// getRedisGroupMemberList get group member list from Redis
func (gmc *GroupMemberCache) getRedisGroupMemberList(groupId string) (*models.TalkGroupPersonList, bool) {
	ctx := context.Background()
	key := fmt.Sprintf("group_member_list:%s", groupId)

	val, err := redisClient.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, false
		}
		log.Printf("Failed to get group member list from Redis: %v", err)
		return nil, false
	}

	var personList models.TalkGroupPersonList
	err = json.Unmarshal([]byte(val), &personList)
	if err != nil {
		log.Printf("Failed to unmarshal group member list: %v", err)
		return nil, false
	}

	return &personList, true
}

// setRedisGroupMemberList set group member list to Redis
func (gmc *GroupMemberCache) setRedisGroupMemberList(groupId string, personList *models.TalkGroupPersonList) {
	ctx := context.Background()
	key := fmt.Sprintf("group_member_list:%s", groupId)

	// create copy to avoid external modification
	copiedList := gmc.copyPersonList(personList)

	data, err := json.Marshal(copiedList)
	if err != nil {
		log.Printf("Failed to marshal group member list: %v", err)
		return
	}

	err = redisClient.Set(ctx, key, data, gmc.ttl).Err()
	if err != nil {
		log.Printf("Failed to set group member list to Redis: %v", err)
	}
}

// deleteRedisGroupMemberList delete group member list from Redis
func (gmc *GroupMemberCache) deleteRedisGroupMemberList(groupId string) {
	ctx := context.Background()
	key := fmt.Sprintf("group_member_list:%s", groupId)

	err := redisClient.Del(ctx, key).Err()
	if err != nil {
		log.Printf("Failed to delete group member list from Redis: %v", err)
	}
}

// clearRedisGroupMemberList clear all group member lists in Redis
func (gmc *GroupMemberCache) clearRedisGroupMemberList() {
	ctx := context.Background()
	pattern := "group_member_list:*"

	keys, err := redisClient.Keys(ctx, pattern).Result()
	if err != nil {
		log.Printf("Failed to get keys from Redis: %v", err)
		return
	}

	if len(keys) > 0 {
		err = redisClient.Del(ctx, keys...).Err()
		if err != nil {
			log.Printf("Failed to clear group member lists from Redis: %v", err)
		}
	}
}

// getRedisGroupMemberListSize get count of group member lists in Redis
func (gmc *GroupMemberCache) getRedisGroupMemberListSize() int {
	ctx := context.Background()
	pattern := "group_member_list:*"

	keys, err := redisClient.Keys(ctx, pattern).Result()
	if err != nil {
		log.Printf("Failed to get keys from Redis: %v", err)
		return 0
	}

	return len(keys)
}

// getRedisGroupMemberListStats get Redis group member list cache statistics
func (gmc *GroupMemberCache) getRedisGroupMemberListStats() map[string]interface{} {
	ctx := context.Background()
	pattern := "group_member_list:*"

	keys, err := redisClient.Keys(ctx, pattern).Result()
	if err != nil {
		log.Printf("Failed to get keys from Redis: %v", err)
		return map[string]interface{}{
			"error": "failed to get keys from Redis",
		}
	}

	return map[string]interface{}{
		"cache_type": "redis",
		"cache_size": len(keys),
		"ttl":        gmc.ttl.String(),
		"keys":       keys,
	}
}

// ==================== Memory Cache Operations ====================

// getMemoryGroupMemberList get group member list from memory
func (gmc *GroupMemberCache) getMemoryGroupMemberList(groupId string) (*models.TalkGroupPersonList, bool) {
	// use lock for specific groupId
	lock := gmc.getGroupLock(groupId)
	lock.RLock()
	defer lock.RUnlock()

	if cacheItemInterface, exists := gmc.memoryCache.Load(groupId); exists {
		if cacheItem, ok := cacheItemInterface.(*groupMemberCacheItem); ok {
			// check if expired
			if time.Now().Before(cacheItem.ExpireTime) {
				return cacheItem.PersonList, true
			} else {
				// expired, delete cache item
				gmc.memoryCache.Delete(groupId)
			}
		}
	}
	return nil, false
}

// setMemoryGroupMemberList set group member list to memory
func (gmc *GroupMemberCache) setMemoryGroupMemberList(groupId string, personList *models.TalkGroupPersonList) {
	// use lock for specific groupId
	lock := gmc.getGroupLock(groupId)
	lock.Lock()
	defer lock.Unlock()

	// create copy to avoid external modification
	copiedList := gmc.copyPersonList(personList)

	// create cache item
	now := time.Now()
	cacheItem := &groupMemberCacheItem{
		PersonList: copiedList,
		UpdateTime: now,
		ExpireTime: now.Add(gmc.ttl),
	}

	gmc.memoryCache.Store(groupId, cacheItem)
}

// deleteMemoryGroupMemberList delete group member list from memory
func (gmc *GroupMemberCache) deleteMemoryGroupMemberList(groupId string) {
	// use lock for specific groupId
	lock := gmc.getGroupLock(groupId)
	lock.Lock()
	defer lock.Unlock()

	gmc.memoryCache.Delete(groupId)
}

// clearMemoryGroupMemberList clear all group member lists in memory
func (gmc *GroupMemberCache) clearMemoryGroupMemberList() {
	// iterate all keys and delete
	gmc.memoryCache.Range(func(key, value interface{}) bool {
		gmc.memoryCache.Delete(key)
		return true
	})
}

// getMemoryGroupMemberListSize get count of group member lists in memory
func (gmc *GroupMemberCache) getMemoryGroupMemberListSize() int {
	count := 0
	gmc.memoryCache.Range(func(key, value interface{}) bool {
		count++
		return true
	})
	return count
}

// getMemoryGroupMemberListStats get memory group member list cache statistics
func (gmc *GroupMemberCache) getMemoryGroupMemberListStats() map[string]interface{} {
	keys := make([]string, 0)
	now := time.Now()
	expiredCount := 0

	gmc.memoryCache.Range(func(key, value interface{}) bool {
		keys = append(keys, key.(string))
		if cacheItem, ok := value.(*groupMemberCacheItem); ok {
			if now.After(cacheItem.ExpireTime) {
				expiredCount++
			}
		}
		return true
	})

	return map[string]interface{}{
		"cache_type":    "memory",
		"cache_size":    len(keys),
		"expired_count": expiredCount,
		"ttl":           gmc.ttl.String(),
		"keys":          keys,
	}
}

// ==================== Helper Methods ====================

// copyPersonList create copy of group member list
func (gmc *GroupMemberCache) copyPersonList(personList *models.TalkGroupPersonList) *models.TalkGroupPersonList {
	copiedList := &models.TalkGroupPersonList{
		GroupId: personList.GroupId,
		Persons: make([]*models.TalkGroupPerson, len(personList.Persons)),
		Total:   personList.Total,
	}

	for i, person := range personList.Persons {
		copiedPerson := &models.TalkGroupPerson{
			GroupIdMetaIdHash: person.GroupIdMetaIdHash,
			GroupId:           person.GroupId,
			MetaId:            person.MetaId,
			Address:           person.Address,
			AvatarTxId:        person.AvatarTxId,
			UserName:          person.UserName,
			UserNickName:      person.UserNickName,
			GroupState:        person.GroupState,
			Timestamp:         person.Timestamp,
			BlockHeight:       person.BlockHeight,
			PinId:             person.PinId,
		}
		copiedList.Persons[i] = copiedPerson
	}

	return copiedList
}

// ==================== Convenience Methods ====================

// GetGroupMemberListFromCache convenience method to get group member list from cache
func GetGroupMemberListFromCache(groupId string) (*models.TalkGroupPersonList, bool) {
	cache := GetGroupMemberCache()
	if cache == nil {
		return nil, false
	}
	return cache.Get(groupId)
}

// SetGroupMemberListToCache convenience method to set group member list to cache
func SetGroupMemberListToCache(groupId string, personList *models.TalkGroupPersonList) {
	cache := GetGroupMemberCache()
	if cache == nil {
		return
	}
	cache.Set(groupId, personList)
}

// DeleteGroupMemberListFromCache convenience method to delete group member list from cache
func DeleteGroupMemberListFromCache(groupId string) {
	cache := GetGroupMemberCache()
	if cache == nil {
		return
	}
	cache.Delete(groupId)
}

// GetGroupMemberCacheStats convenience method to get group member cache statistics
func GetGroupMemberCacheStats() map[string]interface{} {
	cache := GetGroupMemberCache()
	if cache == nil {
		return map[string]interface{}{
			"error": "cache not initialized",
		}
	}
	return cache.GetStats()
}

// CleanExpiredGroupMemberCache clean expired group member cache
func CleanExpiredGroupMemberCache() {
	cache := GetGroupMemberCache()
	if cache == nil {
		return
	}

	now := time.Now()

	// clean expired cache items
	cache.memoryCache.Range(func(key, value interface{}) bool {
		if cacheItem, ok := value.(*groupMemberCacheItem); ok {
			if now.After(cacheItem.ExpireTime) {
				cache.memoryCache.Delete(key)
			}
		}
		return true
	})
}
