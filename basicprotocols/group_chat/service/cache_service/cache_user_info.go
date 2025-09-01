package cache_service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"manindexer/basicprotocols/group_chat/api/respond"

	"github.com/go-redis/redis/v8"
)

// GetCacheUserInfo 获取用户信息缓存
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

// GetCacheUserInfoWithTime 获取用户信息缓存，同时返回更新时间
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

// SetCacheUserInfo 设置用户信息缓存
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

// GetCacheUserInfoByMetaId 根据MetaId获取用户信息缓存
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

// GetCacheUserInfoByMetaIdWithTime 根据MetaId获取用户信息缓存，同时返回更新时间
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

// SetCacheUserInfoByMetaId 根据MetaId设置用户信息缓存
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

// getRedisUserInfo 从Redis获取用户信息
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

	// 这里需要反序列化JSON到userInfoCacheItem结构体
	// 由于Redis存储的是JSON字符串，需要解析
	var cacheItem userInfoCacheItem
	if err := json.Unmarshal([]byte(result), &cacheItem); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cache item: %v", err)
	}

	// 检查是否过期
	if time.Now().After(cacheItem.ExpireTime) {
		// 过期了，删除缓存
		redisClient.Del(ctx, key)
		return nil, nil
	}

	return cacheItem.UserInfo, nil
}

// getRedisUserInfoWithTime 从Redis获取用户信息，同时返回更新时间
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

	// 反序列化JSON到userInfoCacheItem结构体
	var cacheItem userInfoCacheItem
	if err := json.Unmarshal([]byte(result), &cacheItem); err != nil {
		return nil, time.Time{}, fmt.Errorf("failed to unmarshal cache item: %v", err)
	}

	// 检查是否过期
	if time.Now().After(cacheItem.ExpireTime) {
		// 过期了，删除缓存
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
	cacheMutex.RLock()
	defer cacheMutex.RUnlock()

	var addresses []string
	memoryCache.Range(func(key, value interface{}) bool {
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

// getMemoryUserInfoWithTime 从内存获取用户信息，同时返回更新时间
func getMemoryUserInfoWithTime(address string) (*respond.UserInfo, time.Time, error) {
	key := fmt.Sprintf("userinfo:%s", address)

	cacheMutex.RLock()
	defer cacheMutex.RUnlock()

	if value, ok := memoryCache.Load(key); ok {
		if cacheItem, ok := value.(*userInfoCacheItem); ok {
			// 检查是否过期
			if time.Now().Before(cacheItem.ExpireTime) {
				return cacheItem.UserInfo, cacheItem.UpdateTime, nil
			} else {
				// 过期了，删除它
				memoryCache.Delete(key)
			}
		}
	}

	return nil, time.Time{}, nil
}

// setRedisUserInfo 设置用户信息到Redis
func setRedisUserInfo(address string, userInfo *respond.UserInfo) (bool, error) {
	key := fmt.Sprintf("userinfo:%s", address)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 创建缓存项
	cacheItem := &userInfoCacheItem{
		UserInfo:   userInfo,
		UpdateTime: time.Now(),
		ExpireTime: time.Now().Add(expireTime),
	}

	// 序列化为JSON
	jsonData, err := json.Marshal(cacheItem)
	if err != nil {
		return false, fmt.Errorf("failed to marshal cache item: %v", err)
	}

	// 设置缓存
	err = redisClient.Set(ctx, key, string(jsonData), expireTime).Err()
	if err != nil {
		return false, fmt.Errorf("Redis set failed: %v", err)
	}

	return true, nil
}

// getMemoryUserInfo 从内存获取用户信息
func getMemoryUserInfo(address string) (*respond.UserInfo, error) {
	key := fmt.Sprintf("userinfo:%s", address)

	cacheMutex.RLock()
	defer cacheMutex.RUnlock()

	if value, ok := memoryCache.Load(key); ok {
		if cacheItem, ok := value.(*userInfoCacheItem); ok {
			// 检查是否过期
			if time.Now().Before(cacheItem.ExpireTime) {
				return cacheItem.UserInfo, nil
			} else {
				// 过期了，删除它
				memoryCache.Delete(key)
			}
		}
	}

	return nil, nil
}

// setMemoryUserInfo 设置用户信息到内存
func setMemoryUserInfo(address string, userInfo *respond.UserInfo) (bool, error) {
	key := fmt.Sprintf("userinfo:%s", address)

	cacheMutex.Lock()
	defer cacheMutex.Unlock()

	// 创建缓存项
	cacheItem := &userInfoCacheItem{
		UserInfo:   userInfo,
		UpdateTime: time.Now(),
		ExpireTime: time.Now().Add(expireTime),
	}

	memoryCache.Store(key, cacheItem)
	return true, nil
}

// getRedisUserInfoByMetaId 从Redis根据MetaId获取用户信息
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

	// 反序列化JSON到userInfoCacheItem结构体
	var cacheItem userInfoCacheItem
	if err := json.Unmarshal([]byte(result), &cacheItem); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cache item: %v", err)
	}

	// 检查是否过期
	if time.Now().After(cacheItem.ExpireTime) {
		// 过期了，删除缓存
		redisClient.Del(ctx, key)
		return nil, nil
	}

	return cacheItem.UserInfo, nil
}

// getRedisUserInfoByMetaIdWithTime 从Redis根据MetaId获取用户信息，同时返回更新时间
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

	// 反序列化JSON到userInfoCacheItem结构体
	var cacheItem userInfoCacheItem
	if err := json.Unmarshal([]byte(result), &cacheItem); err != nil {
		return nil, time.Time{}, fmt.Errorf("failed to unmarshal cache item: %v", err)
	}

	// 检查是否过期
	if time.Now().After(cacheItem.ExpireTime) {
		// 过期了，删除缓存
		redisClient.Del(ctx, key)
		return nil, time.Time{}, nil
	}

	return cacheItem.UserInfo, cacheItem.UpdateTime, nil
}

// getMemoryUserInfoByMetaId 从内存根据MetaId获取用户信息
func getMemoryUserInfoByMetaId(metaId string) (*respond.UserInfo, error) {
	key := fmt.Sprintf("userinfo_metaid:%s", metaId)

	cacheMutex.RLock()
	defer cacheMutex.RUnlock()

	if value, ok := memoryCache.Load(key); ok {
		if cacheItem, ok := value.(*userInfoCacheItem); ok {
			// 检查是否过期
			if time.Now().Before(cacheItem.ExpireTime) {
				return cacheItem.UserInfo, nil
			} else {
				// 过期了，删除它
				memoryCache.Delete(key)
			}
		}
	}

	return nil, nil
}

// getMemoryUserInfoByMetaIdWithTime 从内存根据MetaId获取用户信息，同时返回更新时间
func getMemoryUserInfoByMetaIdWithTime(metaId string) (*respond.UserInfo, time.Time, error) {
	key := fmt.Sprintf("userinfo_metaid:%s", metaId)

	cacheMutex.RLock()
	defer cacheMutex.RUnlock()

	if value, ok := memoryCache.Load(key); ok {
		if cacheItem, ok := value.(*userInfoCacheItem); ok {
			// 检查是否过期
			if time.Now().Before(cacheItem.ExpireTime) {
				return cacheItem.UserInfo, cacheItem.UpdateTime, nil
			} else {
				// 过期了，删除它
				memoryCache.Delete(key)
			}
		}
	}

	return nil, time.Time{}, nil
}

// setRedisUserInfoByMetaId 根据MetaId设置用户信息到Redis
func setRedisUserInfoByMetaId(metaId string, userInfo *respond.UserInfo) (bool, error) {
	key := fmt.Sprintf("userinfo_metaid:%s", metaId)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 创建缓存项
	cacheItem := &userInfoCacheItem{
		UserInfo:   userInfo,
		UpdateTime: time.Now(),
		ExpireTime: time.Now().Add(expireTime),
	}

	// 序列化为JSON
	jsonData, err := json.Marshal(cacheItem)
	if err != nil {
		return false, fmt.Errorf("failed to marshal cache item: %v", err)
	}

	// 设置缓存
	err = redisClient.Set(ctx, key, string(jsonData), expireTime).Err()
	if err != nil {
		return false, fmt.Errorf("Redis set failed: %v", err)
	}

	return true, nil
}

// setMemoryUserInfoByMetaId 根据MetaId设置用户信息到内存
func setMemoryUserInfoByMetaId(metaId string, userInfo *respond.UserInfo) (bool, error) {
	key := fmt.Sprintf("userinfo_metaid:%s", metaId)

	cacheMutex.Lock()
	defer cacheMutex.Unlock()

	// 创建缓存项
	cacheItem := &userInfoCacheItem{
		UserInfo:   userInfo,
		UpdateTime: time.Now(),
		ExpireTime: time.Now().Add(expireTime),
	}

	memoryCache.Store(key, cacheItem)
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
	cacheMutex.RLock()
	defer cacheMutex.RUnlock()

	var metaIds []string
	memoryCache.Range(func(key, value interface{}) bool {
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
