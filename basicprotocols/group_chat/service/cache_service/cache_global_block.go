package cache_service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
)

// GlobalBlockCache Global blocklist cache
type GlobalBlockCache struct {
	memoryCache sync.Map // map[string]bool - address -> isBlocked
	ttl         time.Duration
	// Independent lock management, address-based locks
	addressLocks sync.Map // map[string]*sync.RWMutex
	// Key prefix for Redis operations
	keyPrefix string
}

// globalBlockCacheItem Global blocklist cache item
type globalBlockCacheItem struct {
	IsBlocked  bool      `json:"isBlocked"`  // Whether blocked
	UpdateTime time.Time `json:"updateTime"` // Update time
	ExpireTime time.Time `json:"expireTime"` // Expire time
}

var (
	globalBlockCache        *GlobalBlockCache
	globalLuckBagBlockCache *GlobalBlockCache
)

// InitGlobalBlockCache Initialize global blocklist cache
func InitGlobalBlockCache(ttl time.Duration) {
	globalBlockCache = &GlobalBlockCache{
		ttl:       ttl,
		keyPrefix: "global_block:",
	}
}

// InitGlobalLuckBagBlockCache Initialize global luck bag blocklist cache
func InitGlobalLuckBagBlockCache(ttl time.Duration) {
	globalLuckBagBlockCache = &GlobalBlockCache{
		ttl:       ttl,
		keyPrefix: "global_lucky_bag_block:",
	}
}

// GetGlobalBlockCache Get global blocklist cache instance
func GetGlobalBlockCache() *GlobalBlockCache {
	return globalBlockCache
}

// GetGlobalLuckBagBlockCache Get global luck bag blocklist cache instance
func GetGlobalLuckBagBlockCache() *GlobalBlockCache {
	return globalLuckBagBlockCache
}

// getAddressLock Get lock for specified address
func (gbc *GlobalBlockCache) getAddressLock(address string) *sync.RWMutex {
	// Try to get existing lock
	if lockInterface, exists := gbc.addressLocks.Load(address); exists {
		return lockInterface.(*sync.RWMutex)
	}

	// Create new lock
	lock := &sync.RWMutex{}

	// Use LoadOrStore to ensure only one goroutine can create the lock
	if actualLock, loaded := gbc.addressLocks.LoadOrStore(address, lock); loaded {
		// If another goroutine has already created the lock, use the existing one
		return actualLock.(*sync.RWMutex)
	}

	// We created a new lock
	return lock
}

// Get Get whether address is blocked from cache
func (gbc *GlobalBlockCache) Get(address string) (bool, bool) {
	if !initialized {
		return false, false
	}

	if useRedis && redisClient != nil {
		return gbc.getRedisGlobalBlock(address)
	} else {
		return gbc.getMemoryGlobalBlock(address)
	}
}

// Set Set address block status to cache
func (gbc *GlobalBlockCache) Set(address string, isBlocked bool) {
	if !initialized {
		return
	}

	if useRedis && redisClient != nil {
		gbc.setRedisGlobalBlock(address, isBlocked)
	} else {
		gbc.setMemoryGlobalBlock(address, isBlocked)
	}
}

// Delete Delete address block status from cache
func (gbc *GlobalBlockCache) Delete(address string) {
	if !initialized {
		return
	}

	if useRedis && redisClient != nil {
		gbc.deleteRedisGlobalBlock(address)
	} else {
		gbc.deleteMemoryGlobalBlock(address)
	}
}

// Clear Clear all cache
func (gbc *GlobalBlockCache) Clear() {
	if !initialized {
		return
	}

	if useRedis && redisClient != nil {
		gbc.clearRedisGlobalBlock()
	} else {
		gbc.clearMemoryGlobalBlock()
	}
}

// Size Get cache size
func (gbc *GlobalBlockCache) Size() int {
	if !initialized {
		return 0
	}

	if useRedis && redisClient != nil {
		return gbc.getRedisGlobalBlockSize()
	} else {
		return gbc.getMemoryGlobalBlockSize()
	}
}

// GetStats Get cache statistics
func (gbc *GlobalBlockCache) GetStats() map[string]interface{} {
	if !initialized {
		return map[string]interface{}{
			"error": "cache service not initialized",
		}
	}

	if useRedis && redisClient != nil {
		return gbc.getRedisGlobalBlockStats()
	} else {
		return gbc.getMemoryGlobalBlockStats()
	}
}

// ==================== Redis Cache Operations ====================

// getRedisGlobalBlock Get address block status from Redis
func (gbc *GlobalBlockCache) getRedisGlobalBlock(address string) (bool, bool) {
	ctx := context.Background()
	key := fmt.Sprintf("%s%s", gbc.keyPrefix, address)

	val, err := redisClient.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return false, false
		}
		log.Printf("Failed to get global block from Redis: %v", err)
		return false, false
	}

	var cacheItem globalBlockCacheItem
	err = json.Unmarshal([]byte(val), &cacheItem)
	if err != nil {
		log.Printf("Failed to unmarshal global block item: %v", err)
		return false, false
	}

	// Check if expired
	if time.Now().After(cacheItem.ExpireTime) {
		// Expired, delete from Redis
		redisClient.Del(ctx, key)
		return false, false
	}

	return cacheItem.IsBlocked, true
}

// setRedisGlobalBlock Set address block status to Redis
func (gbc *GlobalBlockCache) setRedisGlobalBlock(address string, isBlocked bool) {
	ctx := context.Background()
	key := fmt.Sprintf("%s%s", gbc.keyPrefix, address)

	now := time.Now()
	cacheItem := &globalBlockCacheItem{
		IsBlocked:  isBlocked,
		UpdateTime: now,
		ExpireTime: now.Add(gbc.ttl),
	}

	data, err := json.Marshal(cacheItem)
	if err != nil {
		log.Printf("Failed to marshal global block item: %v", err)
		return
	}

	err = redisClient.Set(ctx, key, data, gbc.ttl).Err()
	if err != nil {
		log.Printf("Failed to set global block to Redis: %v", err)
	}
}

// deleteRedisGlobalBlock Delete address block status from Redis
func (gbc *GlobalBlockCache) deleteRedisGlobalBlock(address string) {
	ctx := context.Background()
	key := fmt.Sprintf("%s%s", gbc.keyPrefix, address)

	err := redisClient.Del(ctx, key).Err()
	if err != nil {
		log.Printf("Failed to delete global block from Redis: %v", err)
	}
}

// clearRedisGlobalBlock Clear all global blocklist cache in Redis
func (gbc *GlobalBlockCache) clearRedisGlobalBlock() {
	ctx := context.Background()
	pattern := gbc.keyPrefix + "*"

	keys, err := redisClient.Keys(ctx, pattern).Result()
	if err != nil {
		log.Printf("Failed to get global block keys from Redis: %v", err)
		return
	}

	if len(keys) > 0 {
		err = redisClient.Del(ctx, keys...).Err()
		if err != nil {
			log.Printf("Failed to clear global block cache from Redis: %v", err)
		}
	}
}

// getRedisGlobalBlockSize Get size of global blocklist cache in Redis
func (gbc *GlobalBlockCache) getRedisGlobalBlockSize() int {
	ctx := context.Background()
	pattern := gbc.keyPrefix + "*"

	keys, err := redisClient.Keys(ctx, pattern).Result()
	if err != nil {
		log.Printf("Failed to get global block keys from Redis: %v", err)
		return 0
	}

	return len(keys)
}

// getRedisGlobalBlockStats Get Redis global blocklist cache statistics
func (gbc *GlobalBlockCache) getRedisGlobalBlockStats() map[string]interface{} {
	stats := make(map[string]interface{})
	stats["cache_type"] = "redis"
	stats["ttl"] = gbc.ttl.String()
	stats["last_update_time"] = time.Now().UnixMilli()

	ctx := context.Background()
	pattern := gbc.keyPrefix + "*"

	keys, err := redisClient.Keys(ctx, pattern).Result()
	if err != nil {
		stats["error"] = fmt.Sprintf("Failed to get keys: %v", err)
		return stats
	}

	stats["cache_size"] = len(keys)

	// Count blocked addresses
	blockedCount := 0
	expiredCount := 0
	now := time.Now()

	for _, key := range keys {
		val, err := redisClient.Get(ctx, key).Result()
		if err == nil {
			var cacheItem globalBlockCacheItem
			if err := json.Unmarshal([]byte(val), &cacheItem); err == nil {
				if now.After(cacheItem.ExpireTime) {
					expiredCount++
				} else if cacheItem.IsBlocked {
					blockedCount++
				}
			}
		}
	}

	stats["blocked_count"] = blockedCount
	stats["expired_count"] = expiredCount

	return stats
}

// ==================== Memory Cache Operations ====================

// getMemoryGlobalBlock Get address block status from memory
func (gbc *GlobalBlockCache) getMemoryGlobalBlock(address string) (bool, bool) {
	// Get address lock
	lock := gbc.getAddressLock(address)
	lock.RLock()
	defer lock.RUnlock()

	// Get from memory cache
	if cacheInterface, exists := gbc.memoryCache.Load(address); exists {
		cacheItem := cacheInterface.(*globalBlockCacheItem)

		// Check if expired
		if time.Now().After(cacheItem.ExpireTime) {
			// Expired, delete cache item
			gbc.memoryCache.Delete(address)
			return false, false
		}

		return cacheItem.IsBlocked, true
	}

	return false, false
}

// setMemoryGlobalBlock Set address block status to memory
func (gbc *GlobalBlockCache) setMemoryGlobalBlock(address string, isBlocked bool) {
	// Get address lock
	lock := gbc.getAddressLock(address)
	lock.Lock()
	defer lock.Unlock()

	now := time.Now()
	cacheItem := &globalBlockCacheItem{
		IsBlocked:  isBlocked,
		UpdateTime: now,
		ExpireTime: now.Add(gbc.ttl),
	}

	// Save to memory cache
	gbc.memoryCache.Store(address, cacheItem)
}

// deleteMemoryGlobalBlock Delete address block status from memory
func (gbc *GlobalBlockCache) deleteMemoryGlobalBlock(address string) {
	// Get address lock
	lock := gbc.getAddressLock(address)
	lock.Lock()
	defer lock.Unlock()

	// Delete from memory cache
	gbc.memoryCache.Delete(address)
}

// clearMemoryGlobalBlock Clear all global blocklist cache in memory
func (gbc *GlobalBlockCache) clearMemoryGlobalBlock() {
	gbc.memoryCache.Range(func(key, value interface{}) bool {
		gbc.memoryCache.Delete(key)
		return true
	})
}

// getMemoryGlobalBlockSize Get size of global blocklist cache in memory
func (gbc *GlobalBlockCache) getMemoryGlobalBlockSize() int {
	size := 0
	gbc.memoryCache.Range(func(key, value interface{}) bool {
		size++
		return true
	})
	return size
}

// getMemoryGlobalBlockStats Get memory global blocklist cache statistics
func (gbc *GlobalBlockCache) getMemoryGlobalBlockStats() map[string]interface{} {
	stats := make(map[string]interface{})
	stats["cache_type"] = "memory"
	stats["ttl"] = gbc.ttl.String()
	stats["last_update_time"] = time.Now().UnixMilli()

	// Count expired items and blocked addresses
	expiredCount := 0
	blockedCount := 0
	now := time.Now()

	gbc.memoryCache.Range(func(key, value interface{}) bool {
		cacheItem := value.(*globalBlockCacheItem)
		if now.After(cacheItem.ExpireTime) {
			expiredCount++
		} else if cacheItem.IsBlocked {
			blockedCount++
		}
		return true
	})

	stats["cache_size"] = gbc.getMemoryGlobalBlockSize()
	stats["blocked_count"] = blockedCount
	stats["expired_count"] = expiredCount

	return stats
}

// CleanExpired Clean expired cache items
func (gbc *GlobalBlockCache) CleanExpired() int {
	if !initialized {
		return 0
	}

	if useRedis && redisClient != nil {
		return gbc.cleanExpiredRedisGlobalBlock()
	} else {
		return gbc.cleanExpiredMemoryGlobalBlock()
	}
}

// cleanExpiredRedisGlobalBlock Clean expired cache items in Redis
func (gbc *GlobalBlockCache) cleanExpiredRedisGlobalBlock() int {
	ctx := context.Background()
	pattern := gbc.keyPrefix + "*"

	keys, err := redisClient.Keys(ctx, pattern).Result()
	if err != nil {
		log.Printf("Failed to get global block keys from Redis: %v", err)
		return 0
	}

	cleanedCount := 0
	now := time.Now()

	for _, key := range keys {
		val, err := redisClient.Get(ctx, key).Result()
		if err == nil {
			var cacheItem globalBlockCacheItem
			if err := json.Unmarshal([]byte(val), &cacheItem); err == nil {
				if now.After(cacheItem.ExpireTime) {
					redisClient.Del(ctx, key)
					cleanedCount++
				}
			}
		}
	}

	return cleanedCount
}

// cleanExpiredMemoryGlobalBlock Clean expired cache items in memory
func (gbc *GlobalBlockCache) cleanExpiredMemoryGlobalBlock() int {
	cleanedCount := 0
	now := time.Now()

	gbc.memoryCache.Range(func(key, value interface{}) bool {
		cacheItem := value.(*globalBlockCacheItem)
		if now.After(cacheItem.ExpireTime) {
			gbc.memoryCache.Delete(key)
			cleanedCount++
		}
		return true
	})

	return cleanedCount
}

// Convenience methods

// IsAddressGloballyBlockedFromCache Check if address is globally blocked from cache
func IsAddressGloballyBlockedFromCache(address string) (bool, bool) {
	if globalBlockCache == nil {
		return false, false
	}
	return globalBlockCache.Get(address)
}

// SetGlobalBlockAddressToCache Set address block status to cache
func SetGlobalBlockAddressToCache(address string, isBlocked bool) {
	if globalBlockCache == nil {
		return
	}
	globalBlockCache.Set(address, isBlocked)
}

// DeleteGlobalBlockAddressFromCache Delete address block status from cache
func DeleteGlobalBlockAddressFromCache(address string) {
	if globalBlockCache == nil {
		return
	}
	globalBlockCache.Delete(address)
}

// GetGlobalBlockCacheStats Get global blocklist cache statistics
func GetGlobalBlockCacheStats() map[string]interface{} {
	if globalBlockCache == nil {
		return map[string]interface{}{
			"error": "cache not initialized",
		}
	}
	return globalBlockCache.GetStats()
}

// CleanExpiredGlobalBlockCache Clean expired global blocklist cache
func CleanExpiredGlobalBlockCache() int {
	if globalBlockCache == nil {
		return 0
	}
	return globalBlockCache.CleanExpired()
}

// RefreshGlobalBlockCacheFromDB Refresh global blocklist cache from database
// Note: This method needs to receive a list of blocked addresses, because cache service should not directly depend on database package
func RefreshGlobalBlockCacheFromDB(blockedAddresses []string) error {
	if globalBlockCache == nil {
		return fmt.Errorf("global block cache not initialized")
	}

	// Update cache
	for _, address := range blockedAddresses {
		globalBlockCache.Set(address, true)
	}

	log.Printf("Refreshed global block cache with %d addresses", len(blockedAddresses))
	return nil
}

// ==================== Luck Bag Block Cache Convenience Methods ====================

// IsAddressGloballyLuckBagBlockedFromCache Check if address is globally luck bag blocked from cache
func IsAddressGloballyLuckBagBlockedFromCache(address string) (bool, bool) {
	if globalLuckBagBlockCache == nil {
		return false, false
	}
	return globalLuckBagBlockCache.Get(address)
}

// SetGlobalLuckBagBlockAddressToCache Set address luck bag block status to cache
func SetGlobalLuckBagBlockAddressToCache(address string, isBlocked bool) {
	if globalLuckBagBlockCache == nil {
		return
	}
	globalLuckBagBlockCache.Set(address, isBlocked)
}

// DeleteGlobalLuckBagBlockAddressFromCache Delete address luck bag block status from cache
func DeleteGlobalLuckBagBlockAddressFromCache(address string) {
	if globalLuckBagBlockCache == nil {
		return
	}
	globalLuckBagBlockCache.Delete(address)
}

// GetGlobalLuckBagBlockCacheStats Get global luck bag blocklist cache statistics
func GetGlobalLuckBagBlockCacheStats() map[string]interface{} {
	if globalLuckBagBlockCache == nil {
		return map[string]interface{}{
			"error": "cache not initialized",
		}
	}
	return globalLuckBagBlockCache.GetStats()
}

// CleanExpiredGlobalLuckBagBlockCache Clean expired global luck bag blocklist cache
func CleanExpiredGlobalLuckBagBlockCache() int {
	if globalLuckBagBlockCache == nil {
		return 0
	}
	return globalLuckBagBlockCache.CleanExpired()
}

// RefreshGlobalLuckBagBlockCacheFromDB Refresh global luck bag blocklist cache from database
// Note: This method needs to receive a list of blocked addresses, because cache service should not directly depend on database package
func RefreshGlobalLuckBagBlockCacheFromDB(blockedAddresses []string) error {
	if globalLuckBagBlockCache == nil {
		return fmt.Errorf("global luck bag block cache not initialized")
	}

	// Update cache
	for _, address := range blockedAddresses {
		globalLuckBagBlockCache.Set(address, true)
	}

	log.Printf("Refreshed global luck bag block cache with %d addresses", len(blockedAddresses))
	return nil
}
