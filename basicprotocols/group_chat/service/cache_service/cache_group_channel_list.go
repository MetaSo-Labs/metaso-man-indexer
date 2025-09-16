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

// GroupChannelListCache group channel list cache
type GroupChannelListCache struct {
	memoryCache sync.Map      //
	ttl         time.Duration // cache expiration time
	// independent locks for different groupId, managed by sync.Map
	groupLocks sync.Map // map[string]*sync.RWMutex
}

// groupChannelListCacheItem group channel list cache item
type groupChannelListCacheItem struct {
	ChannelList []*models.TalkGroupChannelModel `json:"channelList"`
	UpdateTime  time.Time                       `json:"updateTime"`
	ExpireTime  time.Time                       `json:"expireTime"`
}

var (
	groupChannelListCache *GroupChannelListCache
)

// InitGroupChannelListCache initialize group channel list cache
func InitGroupChannelListCache(ttl time.Duration) {
	groupChannelListCache = &GroupChannelListCache{
		ttl: ttl,
	}
}

// GetGroupChannelListCache get group channel list cache instance
func GetGroupChannelListCache() *GroupChannelListCache {
	return groupChannelListCache
}

// getGroupLock get lock for specified groupId
func (gclc *GroupChannelListCache) getGroupLock(groupId string) *sync.RWMutex {
	// try to get existing lock
	if lockInterface, exists := gclc.groupLocks.Load(groupId); exists {
		return lockInterface.(*sync.RWMutex)
	}

	// create new lock
	lock := &sync.RWMutex{}

	// use LoadOrStore to ensure only one goroutine can create the lock
	if actualLock, loaded := gclc.groupLocks.LoadOrStore(groupId, lock); loaded {
		// if other goroutine has already created the lock, use the existing one
		return actualLock.(*sync.RWMutex)
	}

	// we created the new lock
	return lock
}

// Get get group channel list from cache
func (gclc *GroupChannelListCache) Get(groupId string) ([]*models.TalkGroupChannelModel, bool) {
	if !initialized {
		return nil, false
	}

	if useRedis && redisClient != nil {
		return gclc.getRedisGroupChannelList(groupId)
	} else {
		return gclc.getMemoryGroupChannelList(groupId)
	}
}

// Set set group channel list to cache
func (gclc *GroupChannelListCache) Set(groupId string, channelList []*models.TalkGroupChannelModel) {
	if !initialized {
		return
	}

	if useRedis && redisClient != nil {
		gclc.setRedisGroupChannelList(groupId, channelList)
	} else {
		gclc.setMemoryGroupChannelList(groupId, channelList)
	}
}

// Delete delete group channel list from cache
func (gclc *GroupChannelListCache) Delete(groupId string) {
	if !initialized {
		return
	}

	if useRedis && redisClient != nil {
		gclc.deleteRedisGroupChannelList(groupId)
	} else {
		gclc.deleteMemoryGroupChannelList(groupId)
	}
}

// Clear clear all cache
func (gclc *GroupChannelListCache) Clear() {
	if !initialized {
		return
	}

	if useRedis && redisClient != nil {
		gclc.clearRedisGroupChannelList()
	} else {
		gclc.clearMemoryGroupChannelList()
	}
}

// Size get cache size
func (gclc *GroupChannelListCache) Size() int {
	if !initialized {
		return 0
	}

	if useRedis && redisClient != nil {
		return gclc.getRedisGroupChannelListSize()
	} else {
		return gclc.getMemoryGroupChannelListSize()
	}
}

// GetStats get cache statistics
func (gclc *GroupChannelListCache) GetStats() map[string]interface{} {
	if !initialized {
		return map[string]interface{}{
			"error": "cache service not initialized",
		}
	}

	if useRedis && redisClient != nil {
		return gclc.getRedisGroupChannelListStats()
	} else {
		return gclc.getMemoryGroupChannelListStats()
	}
}

// ==================== Redis Cache Operations ====================

// getRedisGroupChannelList get group channel list from Redis
func (gclc *GroupChannelListCache) getRedisGroupChannelList(groupId string) ([]*models.TalkGroupChannelModel, bool) {
	ctx := context.Background()
	key := fmt.Sprintf("group_channel_list:%s", groupId)

	val, err := redisClient.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, false
		}
		log.Printf("Failed to get group channel list from Redis: %v", err)
		return nil, false
	}

	var channelList []*models.TalkGroupChannelModel
	err = json.Unmarshal([]byte(val), &channelList)
	if err != nil {
		log.Printf("Failed to unmarshal group channel list: %v", err)
		return nil, false
	}

	return channelList, true
}

// setRedisGroupChannelList set group channel list to Redis
func (gclc *GroupChannelListCache) setRedisGroupChannelList(groupId string, channelList []*models.TalkGroupChannelModel) {
	ctx := context.Background()
	key := fmt.Sprintf("group_channel_list:%s", groupId)

	// create copy to avoid external modification
	copiedList := gclc.copyChannelList(channelList)

	data, err := json.Marshal(copiedList)
	if err != nil {
		log.Printf("Failed to marshal group channel list: %v", err)
		return
	}

	err = redisClient.Set(ctx, key, data, gclc.ttl).Err()
	if err != nil {
		log.Printf("Failed to set group channel list to Redis: %v", err)
	}
}

// deleteRedisGroupChannelList delete group channel list from Redis
func (gclc *GroupChannelListCache) deleteRedisGroupChannelList(groupId string) {
	ctx := context.Background()
	key := fmt.Sprintf("group_channel_list:%s", groupId)

	err := redisClient.Del(ctx, key).Err()
	if err != nil {
		log.Printf("Failed to delete group channel list from Redis: %v", err)
	}
}

// clearRedisGroupChannelList clear all group channel lists in Redis
func (gclc *GroupChannelListCache) clearRedisGroupChannelList() {
	ctx := context.Background()
	pattern := "group_channel_list:*"

	keys, err := redisClient.Keys(ctx, pattern).Result()
	if err != nil {
		log.Printf("Failed to get keys from Redis: %v", err)
		return
	}

	if len(keys) > 0 {
		err = redisClient.Del(ctx, keys...).Err()
		if err != nil {
			log.Printf("Failed to clear group channel lists from Redis: %v", err)
		}
	}
}

// getRedisGroupChannelListSize get count of group channel lists in Redis
func (gclc *GroupChannelListCache) getRedisGroupChannelListSize() int {
	ctx := context.Background()
	pattern := "group_channel_list:*"

	keys, err := redisClient.Keys(ctx, pattern).Result()
	if err != nil {
		log.Printf("Failed to get keys from Redis: %v", err)
		return 0
	}

	return len(keys)
}

// getRedisGroupChannelListStats get Redis group channel list cache statistics
func (gclc *GroupChannelListCache) getRedisGroupChannelListStats() map[string]interface{} {
	ctx := context.Background()
	pattern := "group_channel_list:*"

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
		"ttl":        gclc.ttl.String(),
		"keys":       keys,
	}
}

// ==================== Memory Cache Operations ====================

// getMemoryGroupChannelList get group channel list from memory
func (gclc *GroupChannelListCache) getMemoryGroupChannelList(groupId string) ([]*models.TalkGroupChannelModel, bool) {
	// use lock for specific groupId
	lock := gclc.getGroupLock(groupId)
	lock.RLock()
	defer lock.RUnlock()

	if cacheItemInterface, exists := gclc.memoryCache.Load(groupId); exists {
		if cacheItem, ok := cacheItemInterface.(*groupChannelListCacheItem); ok {
			// check if expired
			if time.Now().Before(cacheItem.ExpireTime) {
				return cacheItem.ChannelList, true
			} else {
				// expired, delete cache item
				gclc.memoryCache.Delete(groupId)
			}
		}
	}
	return nil, false
}

// setMemoryGroupChannelList set group channel list to memory
func (gclc *GroupChannelListCache) setMemoryGroupChannelList(groupId string, channelList []*models.TalkGroupChannelModel) {
	// use lock for specific groupId
	lock := gclc.getGroupLock(groupId)
	lock.Lock()
	defer lock.Unlock()

	// create copy to avoid external modification
	copiedList := gclc.copyChannelList(channelList)

	// create cache item
	now := time.Now()
	cacheItem := &groupChannelListCacheItem{
		ChannelList: copiedList,
		UpdateTime:  now,
		ExpireTime:  now.Add(gclc.ttl),
	}

	gclc.memoryCache.Store(groupId, cacheItem)
}

// deleteMemoryGroupChannelList delete group channel list from memory
func (gclc *GroupChannelListCache) deleteMemoryGroupChannelList(groupId string) {
	// use lock for specific groupId
	lock := gclc.getGroupLock(groupId)
	lock.Lock()
	defer lock.Unlock()

	gclc.memoryCache.Delete(groupId)
}

// clearMemoryGroupChannelList clear all group channel lists in memory
func (gclc *GroupChannelListCache) clearMemoryGroupChannelList() {
	// iterate all keys and delete
	gclc.memoryCache.Range(func(key, value interface{}) bool {
		gclc.memoryCache.Delete(key)
		return true
	})
}

// getMemoryGroupChannelListSize get count of group channel lists in memory
func (gclc *GroupChannelListCache) getMemoryGroupChannelListSize() int {
	count := 0
	gclc.memoryCache.Range(func(key, value interface{}) bool {
		count++
		return true
	})
	return count
}

// getMemoryGroupChannelListStats get memory group channel list cache statistics
func (gclc *GroupChannelListCache) getMemoryGroupChannelListStats() map[string]interface{} {
	keys := make([]string, 0)
	now := time.Now()
	expiredCount := 0

	gclc.memoryCache.Range(func(key, value interface{}) bool {
		keys = append(keys, key.(string))
		if cacheItem, ok := value.(*groupChannelListCacheItem); ok {
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
		"ttl":           gclc.ttl.String(),
		"keys":          keys,
	}
}

// ==================== Helper Methods ====================

// copyChannelList create copy of group channel list
func (gclc *GroupChannelListCache) copyChannelList(channelList []*models.TalkGroupChannelModel) []*models.TalkGroupChannelModel {
	copiedList := make([]*models.TalkGroupChannelModel, len(channelList))

	for i, channel := range channelList {
		copiedChannel := &models.TalkGroupChannelModel{
			ChannelId:         channel.ChannelId,
			GroupId:           channel.GroupId,
			TxId:              channel.TxId,
			PinId:             channel.PinId,
			ChannelName:       channel.ChannelName,
			ChannelIcon:       channel.ChannelIcon,
			ChannelNote:       channel.ChannelNote,
			ChannelType:       channel.ChannelType,
			CreateUserMetaId:  channel.CreateUserMetaId,
			CreateUserAddress: channel.CreateUserAddress,
			Chain:             channel.Chain,
			DeleteStatus:      channel.DeleteStatus,
			Timestamp:         channel.Timestamp,
			BlockHeight:       channel.BlockHeight,
		}
		copiedList[i] = copiedChannel
	}

	return copiedList
}

// ==================== Convenience Methods ====================

// GetGroupChannelListFromCache convenience method to get group channel list from cache
func GetGroupChannelListFromCache(groupId string) ([]*models.TalkGroupChannelModel, bool) {
	cache := GetGroupChannelListCache()
	if cache == nil {
		return nil, false
	}
	return cache.Get(groupId)
}

// SetGroupChannelListToCache convenience method to set group channel list to cache
func SetGroupChannelListToCache(groupId string, channelList []*models.TalkGroupChannelModel) {
	cache := GetGroupChannelListCache()
	if cache == nil {
		return
	}
	cache.Set(groupId, channelList)
}

// DeleteGroupChannelListFromCache convenience method to delete group channel list from cache
func DeleteGroupChannelListFromCache(groupId string) {
	cache := GetGroupChannelListCache()
	if cache == nil {
		return
	}
	cache.Delete(groupId)
}

// GetGroupChannelListCacheStats convenience method to get group channel list cache statistics
func GetGroupChannelListCacheStats() map[string]interface{} {
	cache := GetGroupChannelListCache()
	if cache == nil {
		return map[string]interface{}{
			"error": "cache not initialized",
		}
	}
	return cache.GetStats()
}

// CleanExpiredGroupChannelListCache clean expired group channel list cache
func CleanExpiredGroupChannelListCache() {
	cache := GetGroupChannelListCache()
	if cache == nil {
		return
	}

	now := time.Now()

	// clean expired cache items
	cache.memoryCache.Range(func(key, value interface{}) bool {
		if cacheItem, ok := value.(*groupChannelListCacheItem); ok {
			if now.After(cacheItem.ExpireTime) {
				cache.memoryCache.Delete(key)
			}
		}
		return true
	})
}
