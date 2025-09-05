package cache_service

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"manindexer/basicprotocols/group_chat/api/respond"

	"github.com/go-redis/redis/v8"
)

// UserInfoCache user info cache
type UserInfoCache struct {
	memoryCache sync.Map
	ttl         time.Duration
	// independent locks for different address/metaId
	userLocks sync.Map // map[string]*sync.RWMutex
}

// userInfoCacheItem user info cache item
type userInfoCacheItem struct {
	UserInfo   *respond.UserInfo `json:"userInfo"`
	UpdateTime time.Time         `json:"updateTime"`
	ExpireTime time.Time         `json:"expireTime"`
}

var (
	// user info cache instance
	userInfoCache *UserInfoCache
)

// InitUserInfoCache initialize user info cache
func InitUserInfoCache(ttl time.Duration) {
	userInfoCache = &UserInfoCache{
		ttl: ttl,
	}
}

// getUserLock get lock for specified key
func (uic *UserInfoCache) getUserLock(key string) *sync.RWMutex {
	// try to get existing lock
	if lockInterface, exists := uic.userLocks.Load(key); exists {
		return lockInterface.(*sync.RWMutex)
	}

	// create new lock
	lock := &sync.RWMutex{}

	// use LoadOrStore to ensure only one goroutine can create the lock
	if actualLock, loaded := uic.userLocks.LoadOrStore(key, lock); loaded {
		return actualLock.(*sync.RWMutex)
	}

	return lock
}

// GetCacheUserInfo get user info cache
func GetCacheUserInfo(address string) (*respond.UserInfo, error) {
	if !initialized {
		return nil, fmt.Errorf("cache service not initialized")
	}

	if useRedis && redisClient != nil {
		return getRedisUserInfo(address)
	} else {
		return getMemoryUserInfo(address)
	}
}

// GetCacheUserInfoWithTime get user info cache with update time
func GetCacheUserInfoWithTime(address string) (*respond.UserInfo, time.Time, error) {
	if !initialized {
		return nil, time.Time{}, fmt.Errorf("cache service not initialized")
	}

	if useRedis && redisClient != nil {
		return getRedisUserInfoWithTime(address)
	} else {
		return getMemoryUserInfoWithTime(address)
	}
}

// SetCacheUserInfo set user info cache
func SetCacheUserInfo(address string, userInfo *respond.UserInfo) (bool, error) {
	if !initialized {
		return false, fmt.Errorf("cache service not initialized")
	}

	if useRedis && redisClient != nil {
		return setRedisUserInfo(address, userInfo)
	} else {
		return setMemoryUserInfo(address, userInfo)
	}
}

// GetCacheUserInfoByMetaId get user info cache by MetaId
func GetCacheUserInfoByMetaId(metaId string) (*respond.UserInfo, error) {
	if !initialized {
		return nil, fmt.Errorf("cache service not initialized")
	}

	if useRedis && redisClient != nil {
		return getRedisUserInfoByMetaId(metaId)
	} else {
		return getMemoryUserInfoByMetaId(metaId)
	}
}

// GetCacheUserInfoByMetaIdWithTime get user info cache by MetaId with update time
func GetCacheUserInfoByMetaIdWithTime(metaId string) (*respond.UserInfo, time.Time, error) {
	if !initialized {
		return nil, time.Time{}, fmt.Errorf("cache service not initialized")
	}

	if useRedis && redisClient != nil {
		return getRedisUserInfoByMetaIdWithTime(metaId)
	} else {
		return getMemoryUserInfoByMetaIdWithTime(metaId)
	}
}

// SetCacheUserInfoByMetaId set user info cache by MetaId
func SetCacheUserInfoByMetaId(metaId string, userInfo *respond.UserInfo) (bool, error) {
	if !initialized {
		return false, fmt.Errorf("cache service not initialized")
	}

	if useRedis && redisClient != nil {
		return setRedisUserInfoByMetaId(metaId, userInfo)
	} else {
		return setMemoryUserInfoByMetaId(metaId, userInfo)
	}
}

// getRedisUserInfo get user info from Redis
func getRedisUserInfo(address string) (*respond.UserInfo, error) {
	key := fmt.Sprintf("userinfo:%s", address)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := redisClient.Get(ctx, key).Result()
	if err == redis.Nil {
		// Key does not exist
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("Redis get failed: %v", err)
	}

	// need to deserialize JSON to userInfoCacheItem struct
	// since Redis stores JSON string, need to parse
	var cacheItem userInfoCacheItem
	if err := json.Unmarshal([]byte(result), &cacheItem); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cache item: %v", err)
	}

	// check if expired
	if time.Now().After(cacheItem.ExpireTime) {
		// expired, delete cache
		redisClient.Del(ctx, key)
		return nil, nil
	}

	return cacheItem.UserInfo, nil
}

// getRedisUserInfoWithTime get user info from Redis with update time
func getRedisUserInfoWithTime(address string) (*respond.UserInfo, time.Time, error) {
	key := fmt.Sprintf("userinfo:%s", address)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := redisClient.Get(ctx, key).Result()
	if err == redis.Nil {
		// Key does not exist
		return nil, time.Time{}, nil
	} else if err != nil {
		return nil, time.Time{}, fmt.Errorf("Redis get failed: %v", err)
	}

	// deserialize JSON to userInfoCacheItem struct
	var cacheItem userInfoCacheItem
	if err := json.Unmarshal([]byte(result), &cacheItem); err != nil {
		return nil, time.Time{}, fmt.Errorf("failed to unmarshal cache item: %v", err)
	}

	// check if expired
	if time.Now().After(cacheItem.ExpireTime) {
		// expired, delete cache
		redisClient.Del(ctx, key)
		return nil, time.Time{}, nil
	}

	return cacheItem.UserInfo, cacheItem.UpdateTime, nil
}

// GetAllCachedUserInfoAddresses Get all cached user info addresses
func GetAllCachedUserInfoAddresses() []string {
	if !initialized {
		return []string{}
	}

	if useRedis && redisClient != nil {
		return getAllRedisUserInfoAddresses()
	} else {
		return getAllMemoryUserInfoAddresses()
	}
}

// GetAllCachedUserInfoMetaIds Get all cached user info metaIds
func GetAllCachedUserInfoMetaIds() []string {
	if !initialized {
		return []string{}
	}

	if useRedis && redisClient != nil {
		return getAllRedisUserInfoMetaIds()
	} else {
		return getAllMemoryUserInfoMetaIds()
	}
}

// getAllRedisUserInfoAddresses Get all user info addresses from Redis
func getAllRedisUserInfoAddresses() []string {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Use SCAN to get all keys with pattern "userinfo:*"
	var addresses []string
	iter := redisClient.Scan(ctx, 0, "userinfo:*", 100).Iterator()

	for iter.Next(ctx) {
		key := iter.Val()
		// Extract address from key (remove "userinfo:" prefix)
		if len(key) > 9 { // "userinfo:" is 9 characters
			address := key[9:]
			addresses = append(addresses, address)
		}
	}

	if err := iter.Err(); err != nil {
		fmt.Printf("Error scanning Redis keys: %v\n", err)
	}

	return addresses
}

// getAllMemoryUserInfoAddresses Get all user info addresses from memory
func getAllMemoryUserInfoAddresses() []string {
	var addresses []string
	userInfoCache.memoryCache.Range(func(key, value interface{}) bool {
		if keyStr, ok := key.(string); ok {
			// Check if this is a user info cache key
			if len(keyStr) > 9 && keyStr[:9] == "userinfo:" {
				address := keyStr[9:]
				addresses = append(addresses, address)
			}
		}
		return true
	})

	return addresses
}

// getMemoryUserInfoWithTime get user info from memory with update time
func getMemoryUserInfoWithTime(address string) (*respond.UserInfo, time.Time, error) {
	key := fmt.Sprintf("userinfo:%s", address)

	// use lock for specific key
	lock := userInfoCache.getUserLock(key)
	lock.RLock()
	defer lock.RUnlock()

	if value, ok := userInfoCache.memoryCache.Load(key); ok {
		if cacheItem, ok := value.(*userInfoCacheItem); ok {
			// check if expired
			if time.Now().Before(cacheItem.ExpireTime) {
				return cacheItem.UserInfo, cacheItem.UpdateTime, nil
			} else {
				// expired, delete it
				userInfoCache.memoryCache.Delete(key)
			}
		}
	}

	return nil, time.Time{}, nil
}

// setRedisUserInfo set user info to Redis
func setRedisUserInfo(address string, userInfo *respond.UserInfo) (bool, error) {
	key := fmt.Sprintf("userinfo:%s", address)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// create cache item
	cacheItem := &userInfoCacheItem{
		UserInfo:   userInfo,
		UpdateTime: time.Now(),
		ExpireTime: time.Now().Add(expireTime),
	}

	// serialize to JSON
	jsonData, err := json.Marshal(cacheItem)
	if err != nil {
		return false, fmt.Errorf("failed to marshal cache item: %v", err)
	}

	// set cache
	err = redisClient.Set(ctx, key, string(jsonData), expireTime).Err()
	if err != nil {
		return false, fmt.Errorf("Redis set failed: %v", err)
	}

	return true, nil
}

// getMemoryUserInfo get user info from memory
func getMemoryUserInfo(address string) (*respond.UserInfo, error) {
	key := fmt.Sprintf("userinfo:%s", address)

	// use lock for specific key
	lock := userInfoCache.getUserLock(key)
	lock.RLock()
	defer lock.RUnlock()

	if value, ok := userInfoCache.memoryCache.Load(key); ok {
		if cacheItem, ok := value.(*userInfoCacheItem); ok {
			// check if expired
			if time.Now().Before(cacheItem.ExpireTime) {
				return cacheItem.UserInfo, nil
			} else {
				// expired, delete it
				userInfoCache.memoryCache.Delete(key)
			}
		}
	}

	return nil, nil
}

// setMemoryUserInfo set user info to memory
func setMemoryUserInfo(address string, userInfo *respond.UserInfo) (bool, error) {
	key := fmt.Sprintf("userinfo:%s", address)

	// use lock for specific key
	lock := userInfoCache.getUserLock(key)
	lock.Lock()
	defer lock.Unlock()

	// create cache item
	cacheItem := &userInfoCacheItem{
		UserInfo:   userInfo,
		UpdateTime: time.Now(),
		ExpireTime: time.Now().Add(userInfoCache.ttl),
	}

	userInfoCache.memoryCache.Store(key, cacheItem)
	return true, nil
}

// getRedisUserInfoByMetaId get user info from Redis by MetaId
func getRedisUserInfoByMetaId(metaId string) (*respond.UserInfo, error) {
	key := fmt.Sprintf("userinfo_metaid:%s", metaId)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := redisClient.Get(ctx, key).Result()
	if err == redis.Nil {
		// Key does not exist
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("Redis get failed: %v", err)
	}

	// deserialize JSON to userInfoCacheItem struct
	var cacheItem userInfoCacheItem
	if err := json.Unmarshal([]byte(result), &cacheItem); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cache item: %v", err)
	}

	// check if expired
	if time.Now().After(cacheItem.ExpireTime) {
		// expired, delete cache
		redisClient.Del(ctx, key)
		return nil, nil
	}

	return cacheItem.UserInfo, nil
}

// getRedisUserInfoByMetaIdWithTime get user info from Redis by MetaId with update time
func getRedisUserInfoByMetaIdWithTime(metaId string) (*respond.UserInfo, time.Time, error) {
	key := fmt.Sprintf("userinfo_metaid:%s", metaId)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := redisClient.Get(ctx, key).Result()
	if err == redis.Nil {
		// Key does not exist
		return nil, time.Time{}, nil
	} else if err != nil {
		return nil, time.Time{}, fmt.Errorf("Redis get failed: %v", err)
	}

	// deserialize JSON to userInfoCacheItem struct
	var cacheItem userInfoCacheItem
	if err := json.Unmarshal([]byte(result), &cacheItem); err != nil {
		return nil, time.Time{}, fmt.Errorf("failed to unmarshal cache item: %v", err)
	}

	// check if expired
	if time.Now().After(cacheItem.ExpireTime) {
		// expired, delete cache
		redisClient.Del(ctx, key)
		return nil, time.Time{}, nil
	}

	return cacheItem.UserInfo, cacheItem.UpdateTime, nil
}

// getMemoryUserInfoByMetaId get user info from memory by MetaId
func getMemoryUserInfoByMetaId(metaId string) (*respond.UserInfo, error) {
	key := fmt.Sprintf("userinfo_metaid:%s", metaId)

	// use lock for specific key
	lock := userInfoCache.getUserLock(key)
	lock.RLock()
	defer lock.RUnlock()

	if value, ok := userInfoCache.memoryCache.Load(key); ok {
		if cacheItem, ok := value.(*userInfoCacheItem); ok {
			// check if expired
			if time.Now().Before(cacheItem.ExpireTime) {
				return cacheItem.UserInfo, nil
			} else {
				// expired, delete it
				userInfoCache.memoryCache.Delete(key)
			}
		}
	}

	return nil, nil
}

// getMemoryUserInfoByMetaIdWithTime get user info from memory by MetaId with update time
func getMemoryUserInfoByMetaIdWithTime(metaId string) (*respond.UserInfo, time.Time, error) {
	key := fmt.Sprintf("userinfo_metaid:%s", metaId)

	// use lock for specific key
	lock := userInfoCache.getUserLock(key)
	lock.RLock()
	defer lock.RUnlock()

	if value, ok := userInfoCache.memoryCache.Load(key); ok {
		if cacheItem, ok := value.(*userInfoCacheItem); ok {
			// check if expired
			if time.Now().Before(cacheItem.ExpireTime) {
				return cacheItem.UserInfo, cacheItem.UpdateTime, nil
			} else {
				// expired, delete it
				userInfoCache.memoryCache.Delete(key)
			}
		}
	}

	return nil, time.Time{}, nil
}

// setRedisUserInfoByMetaId set user info to Redis by MetaId
func setRedisUserInfoByMetaId(metaId string, userInfo *respond.UserInfo) (bool, error) {
	key := fmt.Sprintf("userinfo_metaid:%s", metaId)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// create cache item
	cacheItem := &userInfoCacheItem{
		UserInfo:   userInfo,
		UpdateTime: time.Now(),
		ExpireTime: time.Now().Add(expireTime),
	}

	// serialize to JSON
	jsonData, err := json.Marshal(cacheItem)
	if err != nil {
		return false, fmt.Errorf("failed to marshal cache item: %v", err)
	}

	// set cache
	err = redisClient.Set(ctx, key, string(jsonData), expireTime).Err()
	if err != nil {
		return false, fmt.Errorf("Redis set failed: %v", err)
	}

	return true, nil
}

// setMemoryUserInfoByMetaId set user info to memory by MetaId
func setMemoryUserInfoByMetaId(metaId string, userInfo *respond.UserInfo) (bool, error) {
	key := fmt.Sprintf("userinfo_metaid:%s", metaId)

	// use lock for specific key
	lock := userInfoCache.getUserLock(key)
	lock.Lock()
	defer lock.Unlock()

	// create cache item
	cacheItem := &userInfoCacheItem{
		UserInfo:   userInfo,
		UpdateTime: time.Now(),
		ExpireTime: time.Now().Add(userInfoCache.ttl),
	}

	userInfoCache.memoryCache.Store(key, cacheItem)
	return true, nil
}

// getAllRedisUserInfoMetaIds Get all user info metaIds from Redis
func getAllRedisUserInfoMetaIds() []string {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Use SCAN to get all keys with pattern "userinfo_metaid:*"
	var metaIds []string
	iter := redisClient.Scan(ctx, 0, "userinfo_metaid:*", 100).Iterator()

	for iter.Next(ctx) {
		key := iter.Val()
		// Extract metaId from key (remove "userinfo_metaid:" prefix)
		if len(key) > 16 { // "userinfo_metaid:" is 16 characters
			metaId := key[16:]
			metaIds = append(metaIds, metaId)
		}
	}

	if err := iter.Err(); err != nil {
		fmt.Printf("Error scanning Redis keys: %v\n", err)
	}

	return metaIds
}

// getAllMemoryUserInfoMetaIds Get all user info metaIds from memory
func getAllMemoryUserInfoMetaIds() []string {
	var metaIds []string
	userInfoCache.memoryCache.Range(func(key, value interface{}) bool {
		if keyStr, ok := key.(string); ok {
			// Check if this is a user info metaId cache key
			if len(keyStr) > 16 && keyStr[:16] == "userinfo_metaid:" {
				metaId := keyStr[16:]
				metaIds = append(metaIds, metaId)
			}
		}
		return true
	})

	return metaIds
}

// CleanExpiredUserInfoCache clean expired user info cache
func CleanExpiredUserInfoCache() {
	now := time.Now()
	userInfoCache.memoryCache.Range(func(key, value interface{}) bool {
		// clean user info cache
		if cacheItem, ok := value.(*userInfoCacheItem); ok {
			if now.After(cacheItem.ExpireTime) {
				userInfoCache.memoryCache.Delete(key)
			}
		}
		return true
	})
}
