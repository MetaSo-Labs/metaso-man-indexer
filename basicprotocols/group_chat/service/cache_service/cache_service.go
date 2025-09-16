package cache_service

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
)

// GiftCache gift cache
type GiftCache struct {
	memoryCache sync.Map //
	ttl         time.Duration
}

var (
	redisClient *redis.Client
	useRedis    bool
	initialized bool

	// gift cache instance
	giftCache *GiftCache

	expireTime = 5 * time.Minute
)

// InitGiftCache initialize gift cache
func InitGiftCache(ttl time.Duration) {
	giftCache = &GiftCache{
		ttl: ttl,
	}
}

// InitCacheService Initialize cache service
func InitCacheService(redisAddr, redisPassword string, redisDB int) {
	if initialized {
		return
	}

	// initialize gift cache
	InitGiftCache(expireTime)

	// initialize user info cache
	InitUserInfoCache(30 * time.Minute)

	// initialize group member cache
	InitGroupMemberCache(30 * time.Minute)

	// initialize group admins cache
	InitGroupAdminsCache(30 * time.Minute)

	// initialize group info cache
	InitGroupInfoCache(30 * time.Minute)

	// initialize group channel info cache
	InitGroupChannelInfoCache(30 * time.Minute)

	// initialize group channel list cache
	InitGroupChannelListCache(30 * time.Minute)

	// initialize lucky bag cache
	// InitLuckyBagCache(10 * time.Minute)

	if redisAddr != "" {
		// Try to connect to Redis
		redisClient = redis.NewClient(&redis.Options{
			Addr:     redisAddr,
			Password: redisPassword,
			DB:       redisDB,
		})

		// Test connection
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_, err := redisClient.Ping(ctx).Result()
		if err != nil {
			log.Printf("Redis connection failed, will use memory cache: %v", err)
			useRedis = false
			redisClient = nil
		} else {
			log.Printf("Redis connection successful, will use Redis cache")
			useRedis = true
		}
	} else {
		log.Printf("No Redis address configured, will use memory cache")
		useRedis = false
	}

	// Start memory cache cleaner
	if !useRedis {
		StartMemoryCacheCleaner()
	}

	initialized = true
}

// GetCacheGiftInfo Get lucky bag info cache
func GetCacheGiftInfo(groupId, luckyBagPinId string, index int64) (string, error) {
	if !initialized {
		return "", fmt.Errorf("cache service not initialized")
	}

	if useRedis && redisClient != nil {
		return getRedisGiftInfo(groupId, luckyBagPinId, index)
	} else {
		return getMemoryGiftInfo(groupId, luckyBagPinId, index)
	}
}

// SetCacheGiftInfo Set lucky bag info cache
func SetCacheGiftInfo(groupId, luckyBagPinId, metaId string, index int64) (bool, error) {
	if !initialized {
		return false, fmt.Errorf("cache service not initialized")
	}

	if useRedis && redisClient != nil {
		return setRedisGiftInfo(groupId, luckyBagPinId, metaId, index)
	} else {
		return setMemoryGiftInfo(groupId, luckyBagPinId, metaId, index)
	}
}

// getRedisGiftInfo Get lucky bag info from Redis
func getRedisGiftInfo(groupId, luckyBagPinId string, index int64) (string, error) {
	key := fmt.Sprintf("gift:%s:%s:%d", groupId, luckyBagPinId, index)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := redisClient.Get(ctx, key).Result()
	if err == redis.Nil {
		// Key does not exist
		return "", nil
	} else if err != nil {
		return "", fmt.Errorf("Redis get failed: %v", err)
	}

	return result, nil
}

// setRedisGiftInfo Set lucky bag info to Redis
func setRedisGiftInfo(groupId, luckyBagPinId, metaId string, index int64) (bool, error) {
	key := fmt.Sprintf("gift:%s:%s:%d", groupId, luckyBagPinId, index)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Set cache with expiration time
	err := redisClient.Set(ctx, key, metaId, expireTime).Err()
	if err != nil {
		return false, fmt.Errorf("Redis set failed: %v", err)
	}

	return true, nil
}

// getMemoryGiftInfo Get lucky bag info from memory
func getMemoryGiftInfo(groupId, luckyBagPinId string, index int64) (string, error) {
	key := fmt.Sprintf("%s:%s:%d", groupId, luckyBagPinId, index)

	if value, ok := giftCache.memoryCache.Load(key); ok {
		if cacheItem, ok := value.(*cacheItem); ok {
			// Check if expired
			if time.Now().Before(cacheItem.expireTime) {
				return cacheItem.value, nil
			} else {
				// Expired, delete it
				giftCache.memoryCache.Delete(key)
			}
		}
	}

	return "", nil
}

// setMemoryGiftInfo Set lucky bag info to memory
func setMemoryGiftInfo(groupId, luckyBagPinId, metaId string, index int64) (bool, error) {
	key := fmt.Sprintf("%s:%s:%d", groupId, luckyBagPinId, index)

	// Create cache item with expiration time
	cacheItem := &cacheItem{
		value:      metaId,
		expireTime: time.Now().Add(giftCache.ttl),
	}

	giftCache.memoryCache.Store(key, cacheItem)
	return true, nil
}

// cacheItem Memory cache item
type cacheItem struct {
	value      string
	expireTime time.Time
}

// CleanExpiredMemoryCache Clean expired memory cache
func CleanExpiredMemoryCache() {
	now := time.Now()
	giftCache.memoryCache.Range(func(key, value interface{}) bool {
		// clear gift info cache
		if cacheItem, ok := value.(*cacheItem); ok {
			if now.After(cacheItem.expireTime) {
				giftCache.memoryCache.Delete(key)
			}
		}
		return true
	})
}

// StartMemoryCacheCleaner Start memory cache cleaner
func StartMemoryCacheCleaner() {
	go func() {
		ticker := time.NewTicker(5 * time.Minute) // Clean every 5 minutes
		defer ticker.Stop()

		for range ticker.C {
			// clean gift cache
			CleanExpiredMemoryCache()
			// clean user info cache
			CleanExpiredUserInfoCache()
			// clean group member cache
			CleanExpiredGroupMemberCache()
			// clean group admins cache
			CleanExpiredGroupAdminsCache()
			// clean group info cache
			CleanExpiredGroupInfoCache()
			// clean group channel info cache
			CleanExpiredGroupChannelInfoCache()
			// clean group channel list cache
			CleanExpiredGroupChannelListCache()
		}
	}()
}

// GetCacheStatus Get cache status information
func GetCacheStatus() map[string]interface{} {
	status := map[string]interface{}{
		"useRedis":    useRedis,
		"initialized": initialized,
	}

	if !useRedis {
		// Count gift cache items
		count := 0
		giftCache.memoryCache.Range(func(key, value interface{}) bool {
			count++
			return true
		})
		status["giftCacheCount"] = count
	}

	return status
}

// CloseCacheService Close cache service
func CloseCacheService() {
	if redisClient != nil {
		redisClient.Close()
	}
}
