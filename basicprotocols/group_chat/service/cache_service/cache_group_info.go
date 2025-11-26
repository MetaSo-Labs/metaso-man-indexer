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

// GroupInfoCache group info cache
type GroupInfoCache struct {
	memoryCache sync.Map      //
	ttl         time.Duration // cache expiration time
	// independent locks for different groupId, managed by sync.Map
	groupLocks sync.Map // map[string]*sync.RWMutex
}

// groupInfoCacheItem group info cache item
type groupInfoCacheItem struct {
	GroupInfo  *models.TalkGroupModel `json:"groupInfo"`
	UpdateTime time.Time              `json:"updateTime"`
	ExpireTime time.Time              `json:"expireTime"`
}

var (
	groupInfoCache *GroupInfoCache
)

// InitGroupInfoCache initialize group info cache
func InitGroupInfoCache(ttl time.Duration) {
	groupInfoCache = &GroupInfoCache{
		ttl: ttl,
	}
}

// GetGroupInfoCache get group info cache instance
func GetGroupInfoCache() *GroupInfoCache {
	return groupInfoCache
}

// getGroupLock get lock for specified groupId
func (gic *GroupInfoCache) getGroupLock(groupId string) *sync.RWMutex {
	// try to get existing lock
	if lockInterface, exists := gic.groupLocks.Load(groupId); exists {
		return lockInterface.(*sync.RWMutex)
	}

	// create new lock
	lock := &sync.RWMutex{}

	// use LoadOrStore to ensure only one goroutine can create the lock
	if actualLock, loaded := gic.groupLocks.LoadOrStore(groupId, lock); loaded {
		// if other goroutine has already created the lock, use the existing one
		return actualLock.(*sync.RWMutex)
	}

	// we created the new lock
	return lock
}

// Get get group info from cache
func (gic *GroupInfoCache) Get(groupId string) (*models.TalkGroupModel, bool) {
	if !initialized {
		return nil, false
	}

	if useRedis && redisClient != nil {
		return gic.getRedisGroupInfo(groupId)
	} else {
		return gic.getMemoryGroupInfo(groupId)
	}
}

// Set set group info to cache
func (gic *GroupInfoCache) Set(groupId string, groupInfo *models.TalkGroupModel) {
	if !initialized {
		return
	}

	if useRedis && redisClient != nil {
		gic.setRedisGroupInfo(groupId, groupInfo)
	} else {
		gic.setMemoryGroupInfo(groupId, groupInfo)
	}
}

// Delete delete group info from cache
func (gic *GroupInfoCache) Delete(groupId string) {
	if !initialized {
		return
	}

	if useRedis && redisClient != nil {
		gic.deleteRedisGroupInfo(groupId)
	} else {
		gic.deleteMemoryGroupInfo(groupId)
	}
}

// Clear clear all cache
func (gic *GroupInfoCache) Clear() {
	if !initialized {
		return
	}

	if useRedis && redisClient != nil {
		gic.clearRedisGroupInfo()
	} else {
		gic.clearMemoryGroupInfo()
	}
}

// Size get cache size
func (gic *GroupInfoCache) Size() int {
	if !initialized {
		return 0
	}

	if useRedis && redisClient != nil {
		return gic.getRedisGroupInfoSize()
	} else {
		return gic.getMemoryGroupInfoSize()
	}
}

// GetStats get cache statistics
func (gic *GroupInfoCache) GetStats() map[string]interface{} {
	if !initialized {
		return map[string]interface{}{
			"error": "cache service not initialized",
		}
	}

	if useRedis && redisClient != nil {
		return gic.getRedisGroupInfoStats()
	} else {
		return gic.getMemoryGroupInfoStats()
	}
}

// ==================== Redis Cache Operations ====================

// getRedisGroupInfo get group info from Redis
func (gic *GroupInfoCache) getRedisGroupInfo(groupId string) (*models.TalkGroupModel, bool) {
	ctx := context.Background()
	key := fmt.Sprintf("group_info:%s", groupId)

	val, err := redisClient.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, false
		}
		log.Printf("Failed to get group info from Redis: %v", err)
		return nil, false
	}

	var groupInfo models.TalkGroupModel
	err = json.Unmarshal([]byte(val), &groupInfo)
	if err != nil {
		log.Printf("Failed to unmarshal group info: %v", err)
		return nil, false
	}

	return &groupInfo, true
}

// setRedisGroupInfo set group info to Redis
func (gic *GroupInfoCache) setRedisGroupInfo(groupId string, groupInfo *models.TalkGroupModel) {
	ctx := context.Background()
	key := fmt.Sprintf("group_info:%s", groupId)

	// create copy to avoid external modification
	copiedInfo := gic.copyGroupInfo(groupInfo)

	data, err := json.Marshal(copiedInfo)
	if err != nil {
		log.Printf("Failed to marshal group info: %v", err)
		return
	}

	err = redisClient.Set(ctx, key, data, gic.ttl).Err()
	if err != nil {
		log.Printf("Failed to set group info to Redis: %v", err)
	}
}

// deleteRedisGroupInfo delete group info from Redis
func (gic *GroupInfoCache) deleteRedisGroupInfo(groupId string) {
	ctx := context.Background()
	key := fmt.Sprintf("group_info:%s", groupId)

	err := redisClient.Del(ctx, key).Err()
	if err != nil {
		log.Printf("Failed to delete group info from Redis: %v", err)
	}
}

// clearRedisGroupInfo clear all group info in Redis
func (gic *GroupInfoCache) clearRedisGroupInfo() {
	ctx := context.Background()
	pattern := "group_info:*"

	keys, err := redisClient.Keys(ctx, pattern).Result()
	if err != nil {
		log.Printf("Failed to get keys from Redis: %v", err)
		return
	}

	if len(keys) > 0 {
		err = redisClient.Del(ctx, keys...).Err()
		if err != nil {
			log.Printf("Failed to clear group info from Redis: %v", err)
		}
	}
}

// getRedisGroupInfoSize get count of group info in Redis
func (gic *GroupInfoCache) getRedisGroupInfoSize() int {
	ctx := context.Background()
	pattern := "group_info:*"

	keys, err := redisClient.Keys(ctx, pattern).Result()
	if err != nil {
		log.Printf("Failed to get keys from Redis: %v", err)
		return 0
	}

	return len(keys)
}

// getRedisGroupInfoStats get Redis group info cache statistics
func (gic *GroupInfoCache) getRedisGroupInfoStats() map[string]interface{} {
	ctx := context.Background()
	pattern := "group_info:*"

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
		"ttl":        gic.ttl.String(),
		"keys":       keys,
	}
}

// ==================== Memory Cache Operations ====================

// getMemoryGroupInfo get group info from memory
func (gic *GroupInfoCache) getMemoryGroupInfo(groupId string) (*models.TalkGroupModel, bool) {
	// use lock for specific groupId
	lock := gic.getGroupLock(groupId)
	lock.RLock()
	defer lock.RUnlock()

	if cacheItemInterface, exists := gic.memoryCache.Load(groupId); exists {
		if cacheItem, ok := cacheItemInterface.(*groupInfoCacheItem); ok {
			// check if expired
			if time.Now().Before(cacheItem.ExpireTime) {
				return cacheItem.GroupInfo, true
			} else {
				// expired, delete cache item
				gic.memoryCache.Delete(groupId)
			}
		}
	}
	return nil, false
}

// setMemoryGroupInfo set group info to memory
func (gic *GroupInfoCache) setMemoryGroupInfo(groupId string, groupInfo *models.TalkGroupModel) {
	// use lock for specific groupId
	lock := gic.getGroupLock(groupId)
	lock.Lock()
	defer lock.Unlock()

	// create copy to avoid external modification
	copiedInfo := gic.copyGroupInfo(groupInfo)

	// create cache item
	now := time.Now()
	cacheItem := &groupInfoCacheItem{
		GroupInfo:  copiedInfo,
		UpdateTime: now,
		ExpireTime: now.Add(gic.ttl),
	}

	gic.memoryCache.Store(groupId, cacheItem)
}

// deleteMemoryGroupInfo delete group info from memory
func (gic *GroupInfoCache) deleteMemoryGroupInfo(groupId string) {
	// use lock for specific groupId
	lock := gic.getGroupLock(groupId)
	lock.Lock()
	defer lock.Unlock()

	gic.memoryCache.Delete(groupId)
}

// clearMemoryGroupInfo clear all group info in memory
func (gic *GroupInfoCache) clearMemoryGroupInfo() {
	// iterate all keys and delete
	gic.memoryCache.Range(func(key, value interface{}) bool {
		gic.memoryCache.Delete(key)
		return true
	})
}

// getMemoryGroupInfoSize get count of group info in memory
func (gic *GroupInfoCache) getMemoryGroupInfoSize() int {
	count := 0
	gic.memoryCache.Range(func(key, value interface{}) bool {
		count++
		return true
	})
	return count
}

// getMemoryGroupInfoStats get memory group info cache statistics
func (gic *GroupInfoCache) getMemoryGroupInfoStats() map[string]interface{} {
	keys := make([]string, 0)
	now := time.Now()
	expiredCount := 0

	gic.memoryCache.Range(func(key, value interface{}) bool {
		keys = append(keys, key.(string))
		if cacheItem, ok := value.(*groupInfoCacheItem); ok {
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
		"ttl":           gic.ttl.String(),
		"keys":          keys,
	}
}

// ==================== Helper Methods ====================

// copyGroupInfo create copy of group info
func (gic *GroupInfoCache) copyGroupInfo(groupInfo *models.TalkGroupModel) *models.TalkGroupModel {
	copiedInfo := &models.TalkGroupModel{
		GroupId:           groupInfo.GroupId,
		CommunityId:       groupInfo.CommunityId,
		TxId:              groupInfo.TxId,
		PinId:             groupInfo.PinId,
		RoomPublicKey:     groupInfo.RoomPublicKey,
		RoomName:          groupInfo.RoomName,
		RoomNote:          groupInfo.RoomNote,
		RoomIcon:          groupInfo.RoomIcon,
		RoomType:          groupInfo.RoomType,
		RoomStatus:        groupInfo.RoomStatus,
		RoomJoinType:      groupInfo.RoomJoinType,
		RoomAvatarUrl:     groupInfo.RoomAvatarUrl,
		CreateUserMetaId:  groupInfo.CreateUserMetaId,
		CreateUserAddress: groupInfo.CreateUserAddress,
		ChatSettingType:   groupInfo.ChatSettingType,
		Chain:             groupInfo.Chain,
		DeleteStatus:      groupInfo.DeleteStatus,
		Path:              groupInfo.Path,
		Timestamp:         groupInfo.Timestamp,
		BlockHeight:       groupInfo.BlockHeight,
	}

	return copiedInfo
}

// ==================== Convenience Methods ====================

// GetGroupInfoFromCache convenience method to get group info from cache
func GetGroupInfoFromCache(groupId string) (*models.TalkGroupModel, bool) {
	cache := GetGroupInfoCache()
	if cache == nil {
		return nil, false
	}
	return cache.Get(groupId)
}

// SetGroupInfoToCache convenience method to set group info to cache
func SetGroupInfoToCache(groupId string, groupInfo *models.TalkGroupModel) {
	cache := GetGroupInfoCache()
	if cache == nil {
		return
	}
	cache.Set(groupId, groupInfo)
}

// DeleteGroupInfoFromCache convenience method to delete group info from cache
func DeleteGroupInfoFromCache(groupId string) {
	cache := GetGroupInfoCache()
	if cache == nil {
		return
	}
	cache.Delete(groupId)
}

// GetGroupInfoCacheStats convenience method to get group info cache statistics
func GetGroupInfoCacheStats() map[string]interface{} {
	cache := GetGroupInfoCache()
	if cache == nil {
		return map[string]interface{}{
			"error": "cache not initialized",
		}
	}
	return cache.GetStats()
}

// CleanExpiredGroupInfoCache clean expired group info cache
func CleanExpiredGroupInfoCache() {
	cache := GetGroupInfoCache()
	if cache == nil {
		return
	}

	now := time.Now()

	// clean expired cache items
	cache.memoryCache.Range(func(key, value interface{}) bool {
		if cacheItem, ok := value.(*groupInfoCacheItem); ok {
			if now.After(cacheItem.ExpireTime) {
				cache.memoryCache.Delete(key)
			}
		}
		return true
	})
}
