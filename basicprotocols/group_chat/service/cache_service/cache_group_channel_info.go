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

// GroupChannelInfoCache group channel info cache
type GroupChannelInfoCache struct {
	memoryCache sync.Map      //
	ttl         time.Duration // cache expiration time
	// independent locks for different channelId, managed by sync.Map
	channelLocks sync.Map // map[string]*sync.RWMutex
}

// groupChannelInfoCacheItem group channel info cache item
type groupChannelInfoCacheItem struct {
	ChannelInfo *models.TalkGroupChannelModel `json:"channelInfo"`
	UpdateTime  time.Time                     `json:"updateTime"`
	ExpireTime  time.Time                     `json:"expireTime"`
}

var (
	groupChannelInfoCache *GroupChannelInfoCache
)

// InitGroupChannelInfoCache initialize group channel info cache
func InitGroupChannelInfoCache(ttl time.Duration) {
	groupChannelInfoCache = &GroupChannelInfoCache{
		ttl: ttl,
	}
}

// GetGroupChannelInfoCache get group channel info cache instance
func GetGroupChannelInfoCache() *GroupChannelInfoCache {
	return groupChannelInfoCache
}

// getChannelLock get lock for specified channelId
func (gcic *GroupChannelInfoCache) getChannelLock(channelId string) *sync.RWMutex {
	// try to get existing lock
	if lockInterface, exists := gcic.channelLocks.Load(channelId); exists {
		return lockInterface.(*sync.RWMutex)
	}

	// create new lock
	lock := &sync.RWMutex{}

	// use LoadOrStore to ensure only one goroutine can create the lock
	if actualLock, loaded := gcic.channelLocks.LoadOrStore(channelId, lock); loaded {
		// if other goroutine has already created the lock, use the existing one
		return actualLock.(*sync.RWMutex)
	}

	// we created the new lock
	return lock
}

// Get get group channel info from cache
func (gcic *GroupChannelInfoCache) Get(channelId string) (*models.TalkGroupChannelModel, bool) {
	if !initialized {
		return nil, false
	}

	if useRedis && redisClient != nil {
		return gcic.getRedisGroupChannelInfo(channelId)
	} else {
		return gcic.getMemoryGroupChannelInfo(channelId)
	}
}

// Set set group channel info to cache
func (gcic *GroupChannelInfoCache) Set(channelId string, channelInfo *models.TalkGroupChannelModel) {
	if !initialized {
		return
	}

	if useRedis && redisClient != nil {
		gcic.setRedisGroupChannelInfo(channelId, channelInfo)
	} else {
		gcic.setMemoryGroupChannelInfo(channelId, channelInfo)
	}
}

// Delete delete group channel info from cache
func (gcic *GroupChannelInfoCache) Delete(channelId string) {
	if !initialized {
		return
	}

	if useRedis && redisClient != nil {
		gcic.deleteRedisGroupChannelInfo(channelId)
	} else {
		gcic.deleteMemoryGroupChannelInfo(channelId)
	}
}

// Clear clear all cache
func (gcic *GroupChannelInfoCache) Clear() {
	if !initialized {
		return
	}

	if useRedis && redisClient != nil {
		gcic.clearRedisGroupChannelInfo()
	} else {
		gcic.clearMemoryGroupChannelInfo()
	}
}

// Size get cache size
func (gcic *GroupChannelInfoCache) Size() int {
	if !initialized {
		return 0
	}

	if useRedis && redisClient != nil {
		return gcic.getRedisGroupChannelInfoSize()
	} else {
		return gcic.getMemoryGroupChannelInfoSize()
	}
}

// GetStats get cache statistics
func (gcic *GroupChannelInfoCache) GetStats() map[string]interface{} {
	if !initialized {
		return map[string]interface{}{
			"error": "cache service not initialized",
		}
	}

	if useRedis && redisClient != nil {
		return gcic.getRedisGroupChannelInfoStats()
	} else {
		return gcic.getMemoryGroupChannelInfoStats()
	}
}

// ==================== Redis Cache Operations ====================

// getRedisGroupChannelInfo get group channel info from Redis
func (gcic *GroupChannelInfoCache) getRedisGroupChannelInfo(channelId string) (*models.TalkGroupChannelModel, bool) {
	ctx := context.Background()
	key := fmt.Sprintf("group_channel_info:%s", channelId)

	val, err := redisClient.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, false
		}
		log.Printf("Failed to get group channel info from Redis: %v", err)
		return nil, false
	}

	var channelInfo models.TalkGroupChannelModel
	err = json.Unmarshal([]byte(val), &channelInfo)
	if err != nil {
		log.Printf("Failed to unmarshal group channel info: %v", err)
		return nil, false
	}

	return &channelInfo, true
}

// setRedisGroupChannelInfo set group channel info to Redis
func (gcic *GroupChannelInfoCache) setRedisGroupChannelInfo(channelId string, channelInfo *models.TalkGroupChannelModel) {
	ctx := context.Background()
	key := fmt.Sprintf("group_channel_info:%s", channelId)

	// create copy to avoid external modification
	copiedInfo := gcic.copyChannelInfo(channelInfo)

	data, err := json.Marshal(copiedInfo)
	if err != nil {
		log.Printf("Failed to marshal group channel info: %v", err)
		return
	}

	err = redisClient.Set(ctx, key, data, gcic.ttl).Err()
	if err != nil {
		log.Printf("Failed to set group channel info to Redis: %v", err)
	}
}

// deleteRedisGroupChannelInfo delete group channel info from Redis
func (gcic *GroupChannelInfoCache) deleteRedisGroupChannelInfo(channelId string) {
	ctx := context.Background()
	key := fmt.Sprintf("group_channel_info:%s", channelId)

	err := redisClient.Del(ctx, key).Err()
	if err != nil {
		log.Printf("Failed to delete group channel info from Redis: %v", err)
	}
}

// clearRedisGroupChannelInfo clear all group channel info in Redis
func (gcic *GroupChannelInfoCache) clearRedisGroupChannelInfo() {
	ctx := context.Background()
	pattern := "group_channel_info:*"

	keys, err := redisClient.Keys(ctx, pattern).Result()
	if err != nil {
		log.Printf("Failed to get keys from Redis: %v", err)
		return
	}

	if len(keys) > 0 {
		err = redisClient.Del(ctx, keys...).Err()
		if err != nil {
			log.Printf("Failed to clear group channel info from Redis: %v", err)
		}
	}
}

// getRedisGroupChannelInfoSize get count of group channel info in Redis
func (gcic *GroupChannelInfoCache) getRedisGroupChannelInfoSize() int {
	ctx := context.Background()
	pattern := "group_channel_info:*"

	keys, err := redisClient.Keys(ctx, pattern).Result()
	if err != nil {
		log.Printf("Failed to get keys from Redis: %v", err)
		return 0
	}

	return len(keys)
}

// getRedisGroupChannelInfoStats get Redis group channel info cache statistics
func (gcic *GroupChannelInfoCache) getRedisGroupChannelInfoStats() map[string]interface{} {
	ctx := context.Background()
	pattern := "group_channel_info:*"

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
		"ttl":        gcic.ttl.String(),
		"keys":       keys,
	}
}

// ==================== Memory Cache Operations ====================

// getMemoryGroupChannelInfo get group channel info from memory
func (gcic *GroupChannelInfoCache) getMemoryGroupChannelInfo(channelId string) (*models.TalkGroupChannelModel, bool) {
	// use lock for specific channelId
	lock := gcic.getChannelLock(channelId)
	lock.RLock()
	defer lock.RUnlock()

	if cacheItemInterface, exists := gcic.memoryCache.Load(channelId); exists {
		if cacheItem, ok := cacheItemInterface.(*groupChannelInfoCacheItem); ok {
			// check if expired
			if time.Now().Before(cacheItem.ExpireTime) {
				return cacheItem.ChannelInfo, true
			} else {
				// expired, delete cache item
				gcic.memoryCache.Delete(channelId)
			}
		}
	}
	return nil, false
}

// setMemoryGroupChannelInfo set group channel info to memory
func (gcic *GroupChannelInfoCache) setMemoryGroupChannelInfo(channelId string, channelInfo *models.TalkGroupChannelModel) {
	// use lock for specific channelId
	lock := gcic.getChannelLock(channelId)
	lock.Lock()
	defer lock.Unlock()

	// create copy to avoid external modification
	copiedInfo := gcic.copyChannelInfo(channelInfo)

	// create cache item
	now := time.Now()
	cacheItem := &groupChannelInfoCacheItem{
		ChannelInfo: copiedInfo,
		UpdateTime:  now,
		ExpireTime:  now.Add(gcic.ttl),
	}

	gcic.memoryCache.Store(channelId, cacheItem)
}

// deleteMemoryGroupChannelInfo delete group channel info from memory
func (gcic *GroupChannelInfoCache) deleteMemoryGroupChannelInfo(channelId string) {
	// use lock for specific channelId
	lock := gcic.getChannelLock(channelId)
	lock.Lock()
	defer lock.Unlock()

	gcic.memoryCache.Delete(channelId)
}

// clearMemoryGroupChannelInfo clear all group channel info in memory
func (gcic *GroupChannelInfoCache) clearMemoryGroupChannelInfo() {
	// iterate all keys and delete
	gcic.memoryCache.Range(func(key, value interface{}) bool {
		gcic.memoryCache.Delete(key)
		return true
	})
}

// getMemoryGroupChannelInfoSize get count of group channel info in memory
func (gcic *GroupChannelInfoCache) getMemoryGroupChannelInfoSize() int {
	count := 0
	gcic.memoryCache.Range(func(key, value interface{}) bool {
		count++
		return true
	})
	return count
}

// getMemoryGroupChannelInfoStats get memory group channel info cache statistics
func (gcic *GroupChannelInfoCache) getMemoryGroupChannelInfoStats() map[string]interface{} {
	keys := make([]string, 0)
	now := time.Now()
	expiredCount := 0

	gcic.memoryCache.Range(func(key, value interface{}) bool {
		keys = append(keys, key.(string))
		if cacheItem, ok := value.(*groupChannelInfoCacheItem); ok {
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
		"ttl":           gcic.ttl.String(),
		"keys":          keys,
	}
}

// ==================== Helper Methods ====================

// copyChannelInfo create copy of group channel info
func (gcic *GroupChannelInfoCache) copyChannelInfo(channelInfo *models.TalkGroupChannelModel) *models.TalkGroupChannelModel {
	copiedInfo := &models.TalkGroupChannelModel{
		ChannelId:         channelInfo.ChannelId,
		GroupId:           channelInfo.GroupId,
		TxId:              channelInfo.TxId,
		PinId:             channelInfo.PinId,
		ChannelName:       channelInfo.ChannelName,
		ChannelIcon:       channelInfo.ChannelIcon,
		ChannelNote:       channelInfo.ChannelNote,
		ChannelType:       channelInfo.ChannelType,
		CreateUserMetaId:  channelInfo.CreateUserMetaId,
		CreateUserAddress: channelInfo.CreateUserAddress,
		Chain:             channelInfo.Chain,
		DeleteStatus:      channelInfo.DeleteStatus,
		Timestamp:         channelInfo.Timestamp,
		BlockHeight:       channelInfo.BlockHeight,
	}

	return copiedInfo
}

// ==================== Convenience Methods ====================

// GetGroupChannelInfoFromCache convenience method to get group channel info from cache
func GetGroupChannelInfoFromCache(channelId string) (*models.TalkGroupChannelModel, bool) {
	cache := GetGroupChannelInfoCache()
	if cache == nil {
		return nil, false
	}
	return cache.Get(channelId)
}

// SetGroupChannelInfoToCache convenience method to set group channel info to cache
func SetGroupChannelInfoToCache(channelId string, channelInfo *models.TalkGroupChannelModel) {
	cache := GetGroupChannelInfoCache()
	if cache == nil {
		return
	}
	cache.Set(channelId, channelInfo)
}

// DeleteGroupChannelInfoFromCache convenience method to delete group channel info from cache
func DeleteGroupChannelInfoFromCache(channelId string) {
	cache := GetGroupChannelInfoCache()
	if cache == nil {
		return
	}
	cache.Delete(channelId)
}

// GetGroupChannelInfoCacheStats convenience method to get group channel info cache statistics
func GetGroupChannelInfoCacheStats() map[string]interface{} {
	cache := GetGroupChannelInfoCache()
	if cache == nil {
		return map[string]interface{}{
			"error": "cache not initialized",
		}
	}
	return cache.GetStats()
}

// CleanExpiredGroupChannelInfoCache clean expired group channel info cache
func CleanExpiredGroupChannelInfoCache() {
	cache := GetGroupChannelInfoCache()
	if cache == nil {
		return
	}

	now := time.Now()

	// clean expired cache items
	cache.memoryCache.Range(func(key, value interface{}) bool {
		if cacheItem, ok := value.(*groupChannelInfoCacheItem); ok {
			if now.After(cacheItem.ExpireTime) {
				cache.memoryCache.Delete(key)
			}
		}
		return true
	})
}
