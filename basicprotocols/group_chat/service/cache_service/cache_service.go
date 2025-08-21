package cache_service

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"manindexer/basicprotocols/group_chat/api/respond"

	"github.com/go-redis/redis/v8"
)

var (
	redisClient *redis.Client
	useRedis    bool
	memoryCache sync.Map
	cacheMutex  sync.RWMutex
	initialized bool

	expireTime = 5 * time.Minute
)

// InitCacheService Initialize cache service
func InitCacheService(redisAddr, redisPassword string, redisDB int) {
	if initialized {
		return
	}

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
func GetCacheGiftInfo(groupId, txId string, index int64) (string, error) {
	if !initialized {
		return "", fmt.Errorf("cache service not initialized")
	}

	if useRedis && redisClient != nil {
		return getRedisGiftInfo(groupId, txId, index)
	} else {
		return getMemoryGiftInfo(groupId, txId, index)
	}
}

// SetCacheGiftInfo Set lucky bag info cache
func SetCacheGiftInfo(groupId, txId, metaId string, index int64) (bool, error) {
	if !initialized {
		return false, fmt.Errorf("cache service not initialized")
	}

	if useRedis && redisClient != nil {
		return setRedisGiftInfo(groupId, txId, metaId, index)
	} else {
		return setMemoryGiftInfo(groupId, txId, metaId, index)
	}
}

// getRedisGiftInfo Get lucky bag info from Redis
func getRedisGiftInfo(groupId, txId string, index int64) (string, error) {
	key := fmt.Sprintf("gift:%s:%s:%d", groupId, txId, index)
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
func setRedisGiftInfo(groupId, txId, metaId string, index int64) (bool, error) {
	key := fmt.Sprintf("gift:%s:%s:%d", groupId, txId, index)
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
func getMemoryGiftInfo(groupId, txId string, index int64) (string, error) {
	key := fmt.Sprintf("%s:%s:%d", groupId, txId, index)

	cacheMutex.RLock()
	defer cacheMutex.RUnlock()

	if value, ok := memoryCache.Load(key); ok {
		if cacheItem, ok := value.(*cacheItem); ok {
			// Check if expired
			if time.Now().Before(cacheItem.expireTime) {
				return cacheItem.value, nil
			} else {
				// Expired, delete it
				memoryCache.Delete(key)
			}
		}
	}

	return "", nil
}

// setMemoryGiftInfo Set lucky bag info to memory
func setMemoryGiftInfo(groupId, txId, metaId string, index int64) (bool, error) {
	key := fmt.Sprintf("%s:%s:%d", groupId, txId, index)

	cacheMutex.Lock()
	defer cacheMutex.Unlock()

	// Create cache item with expiration time
	cacheItem := &cacheItem{
		value:      metaId,
		expireTime: time.Now().Add(expireTime),
	}

	memoryCache.Store(key, cacheItem)
	return true, nil
}

// cacheItem Memory cache item
type cacheItem struct {
	value      string
	expireTime time.Time
}

// userInfoCacheItem 用户信息缓存项
type userInfoCacheItem struct {
	UserInfo   *respond.UserInfo `json:"userInfo"`
	UpdateTime time.Time         `json:"updateTime"`
	ExpireTime time.Time         `json:"expireTime"`
}

// CleanExpiredMemoryCache Clean expired memory cache
func CleanExpiredMemoryCache() {
	cacheMutex.Lock()
	defer cacheMutex.Unlock()

	now := time.Now()
	memoryCache.Range(func(key, value interface{}) bool {
		// clear gift info cache
		if cacheItem, ok := value.(*cacheItem); ok {
			if now.After(cacheItem.expireTime) {
				memoryCache.Delete(key)
			}
		}
		// clear user info cache
		if userInfoCacheItem, ok := value.(*userInfoCacheItem); ok {
			if now.After(userInfoCacheItem.ExpireTime) {
				memoryCache.Delete(key)
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

		for {
			select {
			case <-ticker.C:
				CleanExpiredMemoryCache()
			}
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
		// Count memory cache items
		count := 0
		memoryCache.Range(func(key, value interface{}) bool {
			count++
			return true
		})
		status["memoryCacheCount"] = count
	}

	return status
}

// CloseCacheService Close cache service
func CloseCacheService() {
	if redisClient != nil {
		redisClient.Close()
	}
}
