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

// LuckyBagCache Lucky bag cache structure
type LuckyBagCache struct {
	memoryCache sync.Map // Memory cache map[string]interface{}
	ttl         time.Duration
	// Independent locks for different lucky bags
	luckyBagLocks sync.Map // map[string]*sync.RWMutex
	// Statistics
	hitCount  int64 // Cache hit count
	missCount int64 // Cache miss count
}

// LuckyBagCacheItem Lucky bag cache item
type LuckyBagCacheItem struct {
	LuckyBag     *models.TalkGroupLuckyBagV3 `json:"luckyBag"`
	UpdateTime   time.Time                   `json:"updateTime"`
	ExpireTime   time.Time                   `json:"expireTime"`
	IsDirty      bool                        `json:"isDirty"`      // Whether there are unsynchronized modifications
	LastSyncTime time.Time                   `json:"lastSyncTime"` // Last sync time
}

// OpenLuckyBagListCacheItem Open lucky bag list cache item
type OpenLuckyBagListCacheItem struct {
	OpenList     *models.OpenLuckyBagList `json:"openList"`
	UpdateTime   time.Time                `json:"updateTime"`
	ExpireTime   time.Time                `json:"expireTime"`
	IsDirty      bool                     `json:"isDirty"`
	LastSyncTime time.Time                `json:"lastSyncTime"`
}

// ResidueLuckyBagListCacheItem Residue lucky bag list cache item
type ResidueLuckyBagListCacheItem struct {
	ResidueList  *models.ResidueLuckyBagList `json:"residueList"`
	UpdateTime   time.Time                   `json:"updateTime"`
	ExpireTime   time.Time                   `json:"expireTime"`
	IsDirty      bool                        `json:"isDirty"`
	LastSyncTime time.Time                   `json:"lastSyncTime"`
}

// OpenLuckyBagCacheItem Open lucky bag detail cache item
type OpenLuckyBagCacheItem struct {
	OpenLuckyBag *models.TalkGroupOpenLuckyBagV3 `json:"openLuckyBag"`
	UpdateTime   time.Time                       `json:"updateTime"`
	ExpireTime   time.Time                       `json:"expireTime"`
	IsDirty      bool                            `json:"isDirty"`
	LastSyncTime time.Time                       `json:"lastSyncTime"`
}

// ResidueLuckyBagCacheItem Residue lucky bag detail cache item
type ResidueLuckyBagCacheItem struct {
	ResidueLuckyBag *models.TalkGroupResidueLuckyBagV3 `json:"residueLuckyBag"`
	UpdateTime      time.Time                          `json:"updateTime"`
	ExpireTime      time.Time                          `json:"expireTime"`
	IsDirty         bool                               `json:"isDirty"`
	LastSyncTime    time.Time                          `json:"lastSyncTime"`
}

var (
	// Lucky bag cache instances grouped by group
	luckyBagCacheMap sync.Map // map[string]*LuckyBagCache
	// Open lucky bag list cache instances grouped by group
	openLuckyBagListCacheMap sync.Map // map[string]*LuckyBagCache
	// Residue lucky bag list cache instances grouped by group
	residueLuckyBagListCacheMap sync.Map // map[string]*LuckyBagCache
	// Open lucky bag detail cache instances grouped by group
	openLuckyBagCacheMap sync.Map // map[string]*LuckyBagCache
	// Residue lucky bag detail cache instances grouped by group
	residueLuckyBagCacheMap sync.Map // map[string]*LuckyBagCache

	// Sync queue
	syncQueue chan syncTask
	// Number of sync worker goroutines
	syncWorkers = 5
)

// syncTask Sync task
type syncTask struct {
	Type      string      `json:"type"`      // "luckyBag", "openList", "residueList", "openLuckyBag", "residueLuckyBag"
	Key       string      `json:"key"`       // Cache key
	Data      interface{} `json:"data"`      // Data to sync
	Timestamp time.Time   `json:"timestamp"` // Task timestamp
}

// InitLuckyBagCache Initialize lucky bag cache
func InitLuckyBagCache(ttl time.Duration) {
	// Initialize sync queue
	syncQueue = make(chan syncTask, 10000) // Buffer 10000 tasks

	// Start sync worker goroutines
	for i := 0; i < syncWorkers; i++ {
		go syncWorker()
	}

	// Start periodic sync goroutine
	go periodicSync()

	log.Printf("LuckyBag cache initialized with TTL: %v", ttl)
}

// getLuckyBagCache Get lucky bag cache for specified group
func getLuckyBagCache(groupId string) *LuckyBagCache {
	if cache, ok := luckyBagCacheMap.Load(groupId); ok {
		return cache.(*LuckyBagCache)
	}

	// Create new cache instance
	newCache := &LuckyBagCache{
		ttl: 10 * time.Minute, // Default 10 minutes TTL
	}

	if actualCache, loaded := luckyBagCacheMap.LoadOrStore(groupId, newCache); loaded {
		return actualCache.(*LuckyBagCache)
	}

	return newCache
}

// getOpenLuckyBagListCache Get open lucky bag list cache for specified group
func getOpenLuckyBagListCache(groupId string) *LuckyBagCache {
	if cache, ok := openLuckyBagListCacheMap.Load(groupId); ok {
		return cache.(*LuckyBagCache)
	}

	// Create new cache instance
	newCache := &LuckyBagCache{
		ttl: 10 * time.Minute, // Default 10 minutes TTL
	}

	if actualCache, loaded := openLuckyBagListCacheMap.LoadOrStore(groupId, newCache); loaded {
		return actualCache.(*LuckyBagCache)
	}

	return newCache
}

// getResidueLuckyBagListCache Get residue lucky bag list cache for specified group
func getResidueLuckyBagListCache(groupId string) *LuckyBagCache {
	if cache, ok := residueLuckyBagListCacheMap.Load(groupId); ok {
		return cache.(*LuckyBagCache)
	}

	// Create new cache instance
	newCache := &LuckyBagCache{
		ttl: 10 * time.Minute, // Default 10 minutes TTL
	}

	if actualCache, loaded := residueLuckyBagListCacheMap.LoadOrStore(groupId, newCache); loaded {
		return actualCache.(*LuckyBagCache)
	}

	return newCache
}

// getOpenLuckyBagCache Get open lucky bag detail cache for specified group
func getOpenLuckyBagCache(groupId string) *LuckyBagCache {
	if cache, ok := openLuckyBagCacheMap.Load(groupId); ok {
		return cache.(*LuckyBagCache)
	}

	// Create new cache instance
	newCache := &LuckyBagCache{
		ttl: 10 * time.Minute, // Default 10 minutes TTL
	}

	if actualCache, loaded := openLuckyBagCacheMap.LoadOrStore(groupId, newCache); loaded {
		return actualCache.(*LuckyBagCache)
	}

	return newCache
}

// getResidueLuckyBagCache Get residue lucky bag detail cache for specified group
func getResidueLuckyBagCache(groupId string) *LuckyBagCache {
	if cache, ok := residueLuckyBagCacheMap.Load(groupId); ok {
		return cache.(*LuckyBagCache)
	}

	// Create new cache instance
	newCache := &LuckyBagCache{
		ttl: 10 * time.Minute, // Default 10 minutes TTL
	}

	if actualCache, loaded := residueLuckyBagCacheMap.LoadOrStore(groupId, newCache); loaded {
		return actualCache.(*LuckyBagCache)
	}

	return newCache
}

// getLuckyBagLock Get lock for specified lucky bag
func (lbc *LuckyBagCache) getLuckyBagLock(key string) *sync.RWMutex {
	// Try to get existing lock
	if lockInterface, exists := lbc.luckyBagLocks.Load(key); exists {
		return lockInterface.(*sync.RWMutex)
	}

	// Create new lock
	lock := &sync.RWMutex{}

	// Use LoadOrStore to ensure only one goroutine can create the lock
	if actualLock, loaded := lbc.luckyBagLocks.LoadOrStore(key, lock); loaded {
		return actualLock.(*sync.RWMutex)
	}

	return lock
}

// GetCacheLuckyBag Get lucky bag cache
func GetCacheLuckyBag(groupId, pinId string) (*models.TalkGroupLuckyBagV3, error) {
	if !initialized {
		return nil, fmt.Errorf("cache service not initialized")
	}

	if useRedis && redisClient != nil {
		return getRedisLuckyBag(pinId)
	} else {
		return getMemoryLuckyBag(groupId, pinId)
	}
}

// SetCacheLuckyBag Set lucky bag cache
func SetCacheLuckyBag(luckyBag *models.TalkGroupLuckyBagV3) (bool, error) {
	if !initialized {
		return false, fmt.Errorf("cache service not initialized")
	}

	if useRedis && redisClient != nil {
		return setRedisLuckyBag(luckyBag)
	} else {
		return setMemoryLuckyBag(luckyBag)
	}
}

// GetCacheOpenLuckyBagList Get open lucky bag list cache
func GetCacheOpenLuckyBagList(groupId, luckyBagPinId string) (*models.OpenLuckyBagList, error) {
	if !initialized {
		return nil, fmt.Errorf("cache service not initialized")
	}

	if useRedis && redisClient != nil {
		return getRedisOpenLuckyBagList(luckyBagPinId)
	} else {
		return getMemoryOpenLuckyBagList(groupId, luckyBagPinId)
	}
}

// SetCacheOpenLuckyBagList Set open lucky bag list cache
func SetCacheOpenLuckyBagList(groupId, luckyBagPinId string, openList *models.OpenLuckyBagList) (bool, error) {
	if !initialized {
		return false, fmt.Errorf("cache service not initialized")
	}

	if useRedis && redisClient != nil {
		return setRedisOpenLuckyBagList(luckyBagPinId, openList)
	} else {
		return setMemoryOpenLuckyBagList(groupId, luckyBagPinId, openList)
	}
}

// GetCacheResidueLuckyBagList Get residue lucky bag list cache
func GetCacheResidueLuckyBagList(groupId, luckyBagPinId string) (*models.ResidueLuckyBagList, error) {
	if !initialized {
		return nil, fmt.Errorf("cache service not initialized")
	}

	if useRedis && redisClient != nil {
		return getRedisResidueLuckyBagList(luckyBagPinId)
	} else {
		return getMemoryResidueLuckyBagList(groupId, luckyBagPinId)
	}
}

// SetCacheResidueLuckyBagList Set residue lucky bag list cache
func SetCacheResidueLuckyBagList(groupId, luckyBagPinId string, residueList *models.ResidueLuckyBagList) (bool, error) {
	if !initialized {
		return false, fmt.Errorf("cache service not initialized")
	}

	if useRedis && redisClient != nil {
		return setRedisResidueLuckyBagList(luckyBagPinId, residueList)
	} else {
		return setMemoryResidueLuckyBagList(groupId, luckyBagPinId, residueList)
	}
}

// GetCacheOpenLuckyBag Get open lucky bag detail cache
func GetCacheOpenLuckyBag(groupId, openPinId string) (*models.TalkGroupOpenLuckyBagV3, error) {
	if !initialized {
		return nil, fmt.Errorf("cache service not initialized")
	}

	if useRedis && redisClient != nil {
		return getRedisOpenLuckyBag(openPinId)
	} else {
		return getMemoryOpenLuckyBag(groupId, openPinId)
	}
}

// SetCacheOpenLuckyBag Set open lucky bag detail cache
func SetCacheOpenLuckyBag(groupId, openPinId string, openLuckyBag *models.TalkGroupOpenLuckyBagV3) (bool, error) {
	if !initialized {
		return false, fmt.Errorf("cache service not initialized")
	}

	if useRedis && redisClient != nil {
		return setRedisOpenLuckyBag(openPinId, openLuckyBag)
	} else {
		return setMemoryOpenLuckyBag(groupId, openPinId, openLuckyBag)
	}
}

// GetCacheResidueLuckyBag Get residue lucky bag detail cache
func GetCacheResidueLuckyBag(groupId, residuePinId string) (*models.TalkGroupResidueLuckyBagV3, error) {
	if !initialized {
		return nil, fmt.Errorf("cache service not initialized")
	}

	if useRedis && redisClient != nil {
		return getRedisResidueLuckyBag(residuePinId)
	} else {
		return getMemoryResidueLuckyBag(groupId, residuePinId)
	}
}

// SetCacheResidueLuckyBag Set residue lucky bag detail cache
func SetCacheResidueLuckyBag(groupId, residuePinId string, residueLuckyBag *models.TalkGroupResidueLuckyBagV3) (bool, error) {
	if !initialized {
		return false, fmt.Errorf("cache service not initialized")
	}

	if useRedis && redisClient != nil {
		return setRedisResidueLuckyBag(residuePinId, residueLuckyBag)
	} else {
		return setMemoryResidueLuckyBag(groupId, residuePinId, residueLuckyBag)
	}
}

// ========== Memory Cache Implementation ==========

// getMemoryLuckyBag Get lucky bag from memory
func getMemoryLuckyBag(groupId, pinId string) (*models.TalkGroupLuckyBagV3, error) {
	cache := getLuckyBagCache(groupId)

	// Use lucky bag level lock
	luckyBagLock := cache.getLuckyBagLock(pinId)
	luckyBagLock.RLock()
	defer luckyBagLock.RUnlock()

	// Use sync.Map Load method
	if value, exists := cache.memoryCache.Load(pinId); exists {
		if cacheItem, ok := value.(*LuckyBagCacheItem); ok {
			// Check if expired
			if time.Now().Before(cacheItem.ExpireTime) {
				// Cache hit
				cache.hitCount++
				return cacheItem.LuckyBag, nil
			} else {
				// Expired, delete it (need to upgrade to write lock)
				luckyBagLock.RUnlock()
				luckyBagLock.Lock()
				defer luckyBagLock.Unlock()

				cache.memoryCache.Delete(pinId)
			}
		}
	}

	// Cache miss
	cache.missCount++
	return nil, nil
}

// setMemoryLuckyBag Set memory lucky bag cache
func setMemoryLuckyBag(luckyBag *models.TalkGroupLuckyBagV3) (bool, error) {
	cache := getLuckyBagCache(luckyBag.GroupId)

	// Use lucky bag level lock
	luckyBagLock := cache.getLuckyBagLock(luckyBag.PinId)
	luckyBagLock.Lock()
	defer luckyBagLock.Unlock()

	now := time.Now()
	cacheItem := &LuckyBagCacheItem{
		LuckyBag:     luckyBag,
		UpdateTime:   now,
		ExpireTime:   now.Add(cache.ttl),
		IsDirty:      true,
		LastSyncTime: now,
	}

	// Use sync.Map Store method
	cache.memoryCache.Store(luckyBag.PinId, cacheItem)

	// Add to sync queue
	select {
	case syncQueue <- syncTask{
		Type:      "luckyBag",
		Key:       luckyBag.PinId,
		Data:      luckyBag,
		Timestamp: now,
	}:
	default:
		log.Printf("Sync queue is full, dropping lucky bag sync task for %s", luckyBag.PinId)
	}

	return true, nil
}

// getMemoryOpenLuckyBagList Get open lucky bag list from memory
func getMemoryOpenLuckyBagList(groupId, luckyBagPinId string) (*models.OpenLuckyBagList, error) {
	cache := getOpenLuckyBagListCache(groupId)

	// Use lucky bag level lock
	luckyBagLock := cache.getLuckyBagLock(luckyBagPinId)
	luckyBagLock.RLock()
	defer luckyBagLock.RUnlock()

	// Use sync.Map Load method
	if value, exists := cache.memoryCache.Load(luckyBagPinId); exists {
		if cacheItem, ok := value.(*OpenLuckyBagListCacheItem); ok {
			// Check if expired
			if time.Now().Before(cacheItem.ExpireTime) {
				// Cache hit
				cache.hitCount++
				return cacheItem.OpenList, nil
			} else {
				// Expired, delete it (need to upgrade to write lock)
				luckyBagLock.RUnlock()
				luckyBagLock.Lock()
				defer luckyBagLock.Unlock()

				cache.memoryCache.Delete(luckyBagPinId)
			}
		}
	}

	// Cache miss
	cache.missCount++
	return nil, nil
}

// setMemoryOpenLuckyBagList Set memory open lucky bag list cache
func setMemoryOpenLuckyBagList(groupId, luckyBagPinId string, openList *models.OpenLuckyBagList) (bool, error) {
	cache := getOpenLuckyBagListCache(groupId)

	// Use lucky bag level lock
	luckyBagLock := cache.getLuckyBagLock(luckyBagPinId)
	luckyBagLock.Lock()
	defer luckyBagLock.Unlock()

	now := time.Now()
	cacheItem := &OpenLuckyBagListCacheItem{
		OpenList:     openList,
		UpdateTime:   now,
		ExpireTime:   now.Add(cache.ttl),
		IsDirty:      true,
		LastSyncTime: now,
	}

	// Use sync.Map Store method
	cache.memoryCache.Store(luckyBagPinId, cacheItem)

	// Add to sync queue
	select {
	case syncQueue <- syncTask{
		Type:      "openList",
		Key:       luckyBagPinId,
		Data:      openList,
		Timestamp: now,
	}:
	default:
		log.Printf("Sync queue is full, dropping open list sync task for %s", luckyBagPinId)
	}

	return true, nil
}

// getMemoryResidueLuckyBagList Get residue lucky bag list from memory
func getMemoryResidueLuckyBagList(groupId, luckyBagPinId string) (*models.ResidueLuckyBagList, error) {
	cache := getResidueLuckyBagListCache(groupId)

	// Use lucky bag level lock
	luckyBagLock := cache.getLuckyBagLock(luckyBagPinId)
	luckyBagLock.RLock()
	defer luckyBagLock.RUnlock()

	// Use sync.Map Load method
	if value, exists := cache.memoryCache.Load(luckyBagPinId); exists {
		if cacheItem, ok := value.(*ResidueLuckyBagListCacheItem); ok {
			// Check if expired
			if time.Now().Before(cacheItem.ExpireTime) {
				// Cache hit
				cache.hitCount++
				return cacheItem.ResidueList, nil
			} else {
				// Expired, delete it (need to upgrade to write lock)
				luckyBagLock.RUnlock()
				luckyBagLock.Lock()
				defer luckyBagLock.Unlock()

				cache.memoryCache.Delete(luckyBagPinId)
			}
		}
	}

	// Cache miss
	cache.missCount++
	return nil, nil
}

// setMemoryResidueLuckyBagList Set memory residue lucky bag list cache
func setMemoryResidueLuckyBagList(groupId, luckyBagPinId string, residueList *models.ResidueLuckyBagList) (bool, error) {
	cache := getResidueLuckyBagListCache(groupId)

	// Use lucky bag level lock
	luckyBagLock := cache.getLuckyBagLock(luckyBagPinId)
	luckyBagLock.Lock()
	defer luckyBagLock.Unlock()

	now := time.Now()
	cacheItem := &ResidueLuckyBagListCacheItem{
		ResidueList:  residueList,
		UpdateTime:   now,
		ExpireTime:   now.Add(cache.ttl),
		IsDirty:      true,
		LastSyncTime: now,
	}

	// Use sync.Map Store method
	cache.memoryCache.Store(luckyBagPinId, cacheItem)

	// Add to sync queue
	select {
	case syncQueue <- syncTask{
		Type:      "residueList",
		Key:       luckyBagPinId,
		Data:      residueList,
		Timestamp: now,
	}:
	default:
		log.Printf("Sync queue is full, dropping residue list sync task for %s", luckyBagPinId)
	}

	return true, nil
}

// getMemoryOpenLuckyBag Get open lucky bag detail from memory
func getMemoryOpenLuckyBag(groupId, openPinId string) (*models.TalkGroupOpenLuckyBagV3, error) {
	cache := getOpenLuckyBagCache(groupId)

	// Use lucky bag level lock
	luckyBagLock := cache.getLuckyBagLock(openPinId)
	luckyBagLock.RLock()
	defer luckyBagLock.RUnlock()

	// Use sync.Map Load method
	if value, exists := cache.memoryCache.Load(openPinId); exists {
		if cacheItem, ok := value.(*OpenLuckyBagCacheItem); ok {
			// Check if expired
			if time.Now().Before(cacheItem.ExpireTime) {
				// Cache hit
				cache.hitCount++
				return cacheItem.OpenLuckyBag, nil
			} else {
				// Expired, delete it (need to upgrade to write lock)
				luckyBagLock.RUnlock()
				luckyBagLock.Lock()
				defer luckyBagLock.Unlock()

				cache.memoryCache.Delete(openPinId)
			}
		}
	}

	// Cache miss
	cache.missCount++
	return nil, nil
}

// setMemoryOpenLuckyBag Set memory open lucky bag detail cache
func setMemoryOpenLuckyBag(groupId, openPinId string, openLuckyBag *models.TalkGroupOpenLuckyBagV3) (bool, error) {
	cache := getOpenLuckyBagCache(groupId)

	// Use lucky bag level lock
	luckyBagLock := cache.getLuckyBagLock(openPinId)
	luckyBagLock.Lock()
	defer luckyBagLock.Unlock()

	now := time.Now()
	cacheItem := &OpenLuckyBagCacheItem{
		OpenLuckyBag: openLuckyBag,
		UpdateTime:   now,
		ExpireTime:   now.Add(cache.ttl),
		IsDirty:      true,
		LastSyncTime: now,
	}

	// Use sync.Map Store method
	cache.memoryCache.Store(openPinId, cacheItem)

	// Add to sync queue
	select {
	case syncQueue <- syncTask{
		Type:      "openLuckyBag",
		Key:       openPinId,
		Data:      openLuckyBag,
		Timestamp: now,
	}:
	default:
		log.Printf("Sync queue is full, dropping open lucky bag sync task for %s", openPinId)
	}

	return true, nil
}

// getMemoryResidueLuckyBag Get residue lucky bag detail from memory
func getMemoryResidueLuckyBag(groupId, residuePinId string) (*models.TalkGroupResidueLuckyBagV3, error) {
	cache := getResidueLuckyBagCache(groupId)

	// Use lucky bag level lock
	luckyBagLock := cache.getLuckyBagLock(residuePinId)
	luckyBagLock.RLock()
	defer luckyBagLock.RUnlock()

	// Use sync.Map Load method
	if value, exists := cache.memoryCache.Load(residuePinId); exists {
		if cacheItem, ok := value.(*ResidueLuckyBagCacheItem); ok {
			// Check if expired
			if time.Now().Before(cacheItem.ExpireTime) {
				// Cache hit
				cache.hitCount++
				return cacheItem.ResidueLuckyBag, nil
			} else {
				// Expired, delete it (need to upgrade to write lock)
				luckyBagLock.RUnlock()
				luckyBagLock.Lock()
				defer luckyBagLock.Unlock()

				cache.memoryCache.Delete(residuePinId)
			}
		}
	}

	// Cache miss
	cache.missCount++
	return nil, nil
}

// setMemoryResidueLuckyBag Set memory residue lucky bag detail cache
func setMemoryResidueLuckyBag(groupId, residuePinId string, residueLuckyBag *models.TalkGroupResidueLuckyBagV3) (bool, error) {
	cache := getResidueLuckyBagCache(groupId)

	// Use lucky bag level lock
	luckyBagLock := cache.getLuckyBagLock(residuePinId)
	luckyBagLock.Lock()
	defer luckyBagLock.Unlock()

	now := time.Now()
	cacheItem := &ResidueLuckyBagCacheItem{
		ResidueLuckyBag: residueLuckyBag,
		UpdateTime:      now,
		ExpireTime:      now.Add(cache.ttl),
		IsDirty:         true,
		LastSyncTime:    now,
	}

	// Use sync.Map Store method
	cache.memoryCache.Store(residuePinId, cacheItem)

	// Add to sync queue
	select {
	case syncQueue <- syncTask{
		Type:      "residueLuckyBag",
		Key:       residuePinId,
		Data:      residueLuckyBag,
		Timestamp: now,
	}:
	default:
		log.Printf("Sync queue is full, dropping residue lucky bag sync task for %s", residuePinId)
	}

	return true, nil
}

// ========== Redis Cache Implementation ==========

// getRedisLuckyBag Get lucky bag from Redis
func getRedisLuckyBag(pinId string) (*models.TalkGroupLuckyBagV3, error) {
	key := fmt.Sprintf("luckybag:%s", pinId)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := redisClient.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("Redis get failed: %v", err)
	}

	var cacheItem LuckyBagCacheItem
	if err := json.Unmarshal([]byte(result), &cacheItem); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cache item: %v", err)
	}

	// Check if expired
	if time.Now().After(cacheItem.ExpireTime) {
		redisClient.Del(ctx, key)
		return nil, nil
	}

	return cacheItem.LuckyBag, nil
}

// setRedisLuckyBag Set Redis lucky bag cache
func setRedisLuckyBag(luckyBag *models.TalkGroupLuckyBagV3) (bool, error) {
	key := fmt.Sprintf("luckybag:%s", luckyBag.PinId)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	now := time.Now()
	cacheItem := &LuckyBagCacheItem{
		LuckyBag:     luckyBag,
		UpdateTime:   now,
		ExpireTime:   now.Add(10 * time.Minute), // Default 10 minutes TTL
		IsDirty:      true,
		LastSyncTime: now,
	}

	data, err := json.Marshal(cacheItem)
	if err != nil {
		return false, fmt.Errorf("failed to marshal cache item: %v", err)
	}

	err = redisClient.Set(ctx, key, data, 10*time.Minute).Err()
	if err != nil {
		return false, fmt.Errorf("Redis set failed: %v", err)
	}

	// Add to sync queue
	select {
	case syncQueue <- syncTask{
		Type:      "luckyBag",
		Key:       luckyBag.PinId,
		Data:      luckyBag,
		Timestamp: now,
	}:
	default:
		log.Printf("Sync queue is full, dropping lucky bag sync task for %s", luckyBag.PinId)
	}

	return true, nil
}

// getRedisOpenLuckyBagList Get open lucky bag list from Redis
func getRedisOpenLuckyBagList(luckyBagPinId string) (*models.OpenLuckyBagList, error) {
	key := fmt.Sprintf("openlist:%s", luckyBagPinId)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := redisClient.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("Redis get failed: %v", err)
	}

	var cacheItem OpenLuckyBagListCacheItem
	if err := json.Unmarshal([]byte(result), &cacheItem); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cache item: %v", err)
	}

	// Check if expired
	if time.Now().After(cacheItem.ExpireTime) {
		redisClient.Del(ctx, key)
		return nil, nil
	}

	return cacheItem.OpenList, nil
}

// setRedisOpenLuckyBagList Set Redis open lucky bag list cache
func setRedisOpenLuckyBagList(luckyBagPinId string, openList *models.OpenLuckyBagList) (bool, error) {
	key := fmt.Sprintf("openlist:%s", luckyBagPinId)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	now := time.Now()
	cacheItem := &OpenLuckyBagListCacheItem{
		OpenList:     openList,
		UpdateTime:   now,
		ExpireTime:   now.Add(10 * time.Minute), // Default 10 minutes TTL
		IsDirty:      true,
		LastSyncTime: now,
	}

	data, err := json.Marshal(cacheItem)
	if err != nil {
		return false, fmt.Errorf("failed to marshal cache item: %v", err)
	}

	err = redisClient.Set(ctx, key, data, 10*time.Minute).Err()
	if err != nil {
		return false, fmt.Errorf("Redis set failed: %v", err)
	}

	// Add to sync queue
	select {
	case syncQueue <- syncTask{
		Type:      "openList",
		Key:       luckyBagPinId,
		Data:      openList,
		Timestamp: now,
	}:
	default:
		log.Printf("Sync queue is full, dropping open list sync task for %s", luckyBagPinId)
	}

	return true, nil
}

// getRedisResidueLuckyBagList Get residue lucky bag list from Redis
func getRedisResidueLuckyBagList(luckyBagPinId string) (*models.ResidueLuckyBagList, error) {
	key := fmt.Sprintf("residuelist:%s", luckyBagPinId)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := redisClient.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("Redis get failed: %v", err)
	}

	var cacheItem ResidueLuckyBagListCacheItem
	if err := json.Unmarshal([]byte(result), &cacheItem); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cache item: %v", err)
	}

	// Check if expired
	if time.Now().After(cacheItem.ExpireTime) {
		redisClient.Del(ctx, key)
		return nil, nil
	}

	return cacheItem.ResidueList, nil
}

// setRedisResidueLuckyBagList Set Redis residue lucky bag list cache
func setRedisResidueLuckyBagList(luckyBagPinId string, residueList *models.ResidueLuckyBagList) (bool, error) {
	key := fmt.Sprintf("residuelist:%s", luckyBagPinId)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	now := time.Now()
	cacheItem := &ResidueLuckyBagListCacheItem{
		ResidueList:  residueList,
		UpdateTime:   now,
		ExpireTime:   now.Add(10 * time.Minute), // Default 10 minutes TTL
		IsDirty:      true,
		LastSyncTime: now,
	}

	data, err := json.Marshal(cacheItem)
	if err != nil {
		return false, fmt.Errorf("failed to marshal cache item: %v", err)
	}

	err = redisClient.Set(ctx, key, data, 10*time.Minute).Err()
	if err != nil {
		return false, fmt.Errorf("Redis set failed: %v", err)
	}

	// Add to sync queue
	select {
	case syncQueue <- syncTask{
		Type:      "residueList",
		Key:       luckyBagPinId,
		Data:      residueList,
		Timestamp: now,
	}:
	default:
		log.Printf("Sync queue is full, dropping residue list sync task for %s", luckyBagPinId)
	}

	return true, nil
}

// getRedisOpenLuckyBag Get open lucky bag detail from Redis
func getRedisOpenLuckyBag(openPinId string) (*models.TalkGroupOpenLuckyBagV3, error) {
	key := fmt.Sprintf("openluckybag:%s", openPinId)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := redisClient.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("Redis get failed: %v", err)
	}

	var cacheItem OpenLuckyBagCacheItem
	if err := json.Unmarshal([]byte(result), &cacheItem); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cache item: %v", err)
	}

	// Check if expired
	if time.Now().After(cacheItem.ExpireTime) {
		redisClient.Del(ctx, key)
		return nil, nil
	}

	return cacheItem.OpenLuckyBag, nil
}

// setRedisOpenLuckyBag Set Redis open lucky bag detail cache
func setRedisOpenLuckyBag(openPinId string, openLuckyBag *models.TalkGroupOpenLuckyBagV3) (bool, error) {
	key := fmt.Sprintf("openluckybag:%s", openPinId)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	now := time.Now()
	cacheItem := &OpenLuckyBagCacheItem{
		OpenLuckyBag: openLuckyBag,
		UpdateTime:   now,
		ExpireTime:   now.Add(10 * time.Minute), // Default 10 minutes TTL
		IsDirty:      true,
		LastSyncTime: now,
	}

	data, err := json.Marshal(cacheItem)
	if err != nil {
		return false, fmt.Errorf("failed to marshal cache item: %v", err)
	}

	err = redisClient.Set(ctx, key, data, 10*time.Minute).Err()
	if err != nil {
		return false, fmt.Errorf("Redis set failed: %v", err)
	}

	// Add to sync queue
	select {
	case syncQueue <- syncTask{
		Type:      "openLuckyBag",
		Key:       openPinId,
		Data:      openLuckyBag,
		Timestamp: now,
	}:
	default:
		log.Printf("Sync queue is full, dropping open lucky bag sync task for %s", openPinId)
	}

	return true, nil
}

// getRedisResidueLuckyBag Get residue lucky bag detail from Redis
func getRedisResidueLuckyBag(residuePinId string) (*models.TalkGroupResidueLuckyBagV3, error) {
	key := fmt.Sprintf("residueluckybag:%s", residuePinId)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := redisClient.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("Redis get failed: %v", err)
	}

	var cacheItem ResidueLuckyBagCacheItem
	if err := json.Unmarshal([]byte(result), &cacheItem); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cache item: %v", err)
	}

	// Check if expired
	if time.Now().After(cacheItem.ExpireTime) {
		redisClient.Del(ctx, key)
		return nil, nil
	}

	return cacheItem.ResidueLuckyBag, nil
}

// setRedisResidueLuckyBag Set Redis residue lucky bag detail cache
func setRedisResidueLuckyBag(residuePinId string, residueLuckyBag *models.TalkGroupResidueLuckyBagV3) (bool, error) {
	key := fmt.Sprintf("residueluckybag:%s", residuePinId)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	now := time.Now()
	cacheItem := &ResidueLuckyBagCacheItem{
		ResidueLuckyBag: residueLuckyBag,
		UpdateTime:      now,
		ExpireTime:      now.Add(10 * time.Minute), // Default 10 minutes TTL
		IsDirty:         true,
		LastSyncTime:    now,
	}

	data, err := json.Marshal(cacheItem)
	if err != nil {
		return false, fmt.Errorf("failed to marshal cache item: %v", err)
	}

	err = redisClient.Set(ctx, key, data, 10*time.Minute).Err()
	if err != nil {
		return false, fmt.Errorf("Redis set failed: %v", err)
	}

	// Add to sync queue
	select {
	case syncQueue <- syncTask{
		Type:      "residueLuckyBag",
		Key:       residuePinId,
		Data:      residueLuckyBag,
		Timestamp: now,
	}:
	default:
		log.Printf("Sync queue is full, dropping residue lucky bag sync task for %s", residuePinId)
	}

	return true, nil
}

// ========== Sync Mechanism ==========

// syncWorker Sync worker goroutine
func syncWorker() {
	for task := range syncQueue {
		err := syncToPebble(task)
		if err != nil {
			log.Printf("Failed to sync task %+v to Pebble: %v", task, err)
		}
	}
}

// syncToPebble Sync data to Pebble database
func syncToPebble(task syncTask) error {
	switch task.Type {
	case "luckyBag":
		if luckyBag, ok := task.Data.(*models.TalkGroupLuckyBagV3); ok {
			// Need to implement save logic based on actual situation
			// Since db package access is needed, log temporarily here
			log.Printf("Syncing lucky bag for %s", task.Key)
			_ = luckyBag // Avoid unused variable warning
		}
	case "openList":
		if openList, ok := task.Data.(*models.OpenLuckyBagList); ok {
			// Need to implement save logic based on actual situation
			// Since OpenLuckyBagList saving is complex, special handling may be needed
			log.Printf("Syncing open list for %s", task.Key)
			_ = openList // Avoid unused variable warning
		}
	case "residueList":
		if residueList, ok := task.Data.(*models.ResidueLuckyBagList); ok {
			// Need to implement save logic based on actual situation
			log.Printf("Syncing residue list for %s", task.Key)
			_ = residueList // Avoid unused variable warning
		}
	case "openLuckyBag":
		if openLuckyBag, ok := task.Data.(*models.TalkGroupOpenLuckyBagV3); ok {
			// Need to implement save logic based on actual situation
			log.Printf("Syncing open lucky bag for %s", task.Key)
			_ = openLuckyBag // Avoid unused variable warning
		}
	case "residueLuckyBag":
		if residueLuckyBag, ok := task.Data.(*models.TalkGroupResidueLuckyBagV3); ok {
			// Need to implement save logic based on actual situation
			log.Printf("Syncing residue lucky bag for %s", task.Key)
			_ = residueLuckyBag // Avoid unused variable warning
		}
	}
	return nil
}

// periodicSync Periodically sync expiring cache
func periodicSync() {
	ticker := time.NewTicker(5 * time.Minute) // Check every 5 minutes
	defer ticker.Stop()

	for range ticker.C {
		// Sync expiring lucky bag cache
		syncExpiringLuckyBags()
		// Sync expiring open list cache
		syncExpiringOpenLists()
		// Sync expiring residue list cache
		syncExpiringResidueLists()
		// Sync expiring open lucky bag detail cache
		syncExpiringOpenLuckyBags()
		// Sync expiring residue lucky bag detail cache
		syncExpiringResidueLuckyBags()
	}
}

// syncExpiringLuckyBags Sync expiring lucky bag cache
func syncExpiringLuckyBags() {
	now := time.Now()
	expireThreshold := 2 * time.Minute // Sync 2 minutes in advance

	// Iterate through all group lucky bag caches
	luckyBagCacheMap.Range(func(groupIdInterface, cacheInterface interface{}) bool {
		groupId := groupIdInterface.(string)
		cache := cacheInterface.(*LuckyBagCache)

		// Create copy to avoid holding lock for long time
		itemsToSync := make(map[string]*LuckyBagCacheItem)
		cache.memoryCache.Range(func(keyInterface, valueInterface interface{}) bool {
			key := keyInterface.(string)
			value := valueInterface
			if cacheItem, ok := value.(*LuckyBagCacheItem); ok {
				// Check if about to expire and not synced
				if cacheItem.IsDirty && now.Add(expireThreshold).After(cacheItem.ExpireTime) {
					itemsToSync[key] = cacheItem
				}
			}
			return true
		})

		// Process items that need sync
		for key, cacheItem := range itemsToSync {
			select {
			case syncQueue <- syncTask{
				Type:      "luckyBag",
				Key:       key,
				Data:      cacheItem.LuckyBag,
				Timestamp: now,
			}:
				// Mark as synced
				if itemInterface, ok := cache.memoryCache.Load(key); ok {
					if item, ok := itemInterface.(*LuckyBagCacheItem); ok {
						item.IsDirty = false
						item.LastSyncTime = now
					}
				}
			default:
				log.Printf("Sync queue is full, cannot sync expiring lucky bag %s in group %s", key, groupId)
			}
		}

		return true // Continue iteration
	})
}

// syncExpiringOpenLists Sync expiring open list cache
func syncExpiringOpenLists() {
	now := time.Now()
	expireThreshold := 2 * time.Minute

	// Iterate through all group open list caches
	openLuckyBagListCacheMap.Range(func(groupIdInterface, cacheInterface interface{}) bool {
		groupId := groupIdInterface.(string)
		cache := cacheInterface.(*LuckyBagCache)

		// Create copy to avoid holding lock for long time
		itemsToSync := make(map[string]*OpenLuckyBagListCacheItem)
		cache.memoryCache.Range(func(keyInterface, valueInterface interface{}) bool {
			key := keyInterface.(string)
			value := valueInterface
			if cacheItem, ok := value.(*OpenLuckyBagListCacheItem); ok {
				if cacheItem.IsDirty && now.Add(expireThreshold).After(cacheItem.ExpireTime) {
					itemsToSync[key] = cacheItem
				}
			}
			return true
		})

		// Process items that need sync
		for key, cacheItem := range itemsToSync {
			select {
			case syncQueue <- syncTask{
				Type:      "openList",
				Key:       key,
				Data:      cacheItem.OpenList,
				Timestamp: now,
			}:
				// Mark as synced
				if itemInterface, ok := cache.memoryCache.Load(key); ok {
					if item, ok := itemInterface.(*OpenLuckyBagListCacheItem); ok {
						item.IsDirty = false
						item.LastSyncTime = now
					}
				}
			default:
				log.Printf("Sync queue is full, cannot sync expiring open list %s in group %s", key, groupId)
			}
		}

		return true // Continue iteration
	})
}

// syncExpiringResidueLists Sync expiring residue list cache
func syncExpiringResidueLists() {
	now := time.Now()
	expireThreshold := 2 * time.Minute

	// Iterate through all group residue list caches
	residueLuckyBagListCacheMap.Range(func(groupIdInterface, cacheInterface interface{}) bool {
		groupId := groupIdInterface.(string)
		cache := cacheInterface.(*LuckyBagCache)

		// Create copy to avoid holding lock for long time
		itemsToSync := make(map[string]*ResidueLuckyBagListCacheItem)
		cache.memoryCache.Range(func(keyInterface, valueInterface interface{}) bool {
			key := keyInterface.(string)
			value := valueInterface
			if cacheItem, ok := value.(*ResidueLuckyBagListCacheItem); ok {
				if cacheItem.IsDirty && now.Add(expireThreshold).After(cacheItem.ExpireTime) {
					itemsToSync[key] = cacheItem
				}
			}
			return true
		})

		// Process items that need sync
		for key, cacheItem := range itemsToSync {
			select {
			case syncQueue <- syncTask{
				Type:      "residueList",
				Key:       key,
				Data:      cacheItem.ResidueList,
				Timestamp: now,
			}:
				// Mark as synced
				if itemInterface, ok := cache.memoryCache.Load(key); ok {
					if item, ok := itemInterface.(*ResidueLuckyBagListCacheItem); ok {
						item.IsDirty = false
						item.LastSyncTime = now
					}
				}
			default:
				log.Printf("Sync queue is full, cannot sync expiring residue list %s in group %s", key, groupId)
			}
		}

		return true // Continue iteration
	})
}

// syncExpiringOpenLuckyBags Sync expiring open lucky bag detail cache
func syncExpiringOpenLuckyBags() {
	now := time.Now()
	expireThreshold := 2 * time.Minute

	// Iterate through all group open lucky bag detail caches
	openLuckyBagCacheMap.Range(func(groupIdInterface, cacheInterface interface{}) bool {
		groupId := groupIdInterface.(string)
		cache := cacheInterface.(*LuckyBagCache)

		// Create copy to avoid holding lock for long time
		itemsToSync := make(map[string]*OpenLuckyBagCacheItem)
		cache.memoryCache.Range(func(keyInterface, valueInterface interface{}) bool {
			key := keyInterface.(string)
			value := valueInterface
			if cacheItem, ok := value.(*OpenLuckyBagCacheItem); ok {
				if cacheItem.IsDirty && now.Add(expireThreshold).After(cacheItem.ExpireTime) {
					itemsToSync[key] = cacheItem
				}
			}
			return true
		})

		// Process items that need sync
		for key, cacheItem := range itemsToSync {
			select {
			case syncQueue <- syncTask{
				Type:      "openLuckyBag",
				Key:       key,
				Data:      cacheItem.OpenLuckyBag,
				Timestamp: now,
			}:
				// Mark as synced
				if itemInterface, ok := cache.memoryCache.Load(key); ok {
					if item, ok := itemInterface.(*OpenLuckyBagCacheItem); ok {
						item.IsDirty = false
						item.LastSyncTime = now
					}
				}
			default:
				log.Printf("Sync queue is full, cannot sync expiring open lucky bag %s in group %s", key, groupId)
			}
		}

		return true // Continue iteration
	})
}

// syncExpiringResidueLuckyBags Sync expiring residue lucky bag detail cache
func syncExpiringResidueLuckyBags() {
	now := time.Now()
	expireThreshold := 2 * time.Minute

	// Iterate through all group residue lucky bag detail caches
	residueLuckyBagCacheMap.Range(func(groupIdInterface, cacheInterface interface{}) bool {
		groupId := groupIdInterface.(string)
		cache := cacheInterface.(*LuckyBagCache)

		// Create copy to avoid holding lock for long time
		itemsToSync := make(map[string]*ResidueLuckyBagCacheItem)
		cache.memoryCache.Range(func(keyInterface, valueInterface interface{}) bool {
			key := keyInterface.(string)
			value := valueInterface
			if cacheItem, ok := value.(*ResidueLuckyBagCacheItem); ok {
				if cacheItem.IsDirty && now.Add(expireThreshold).After(cacheItem.ExpireTime) {
					itemsToSync[key] = cacheItem
				}
			}
			return true
		})

		// Process items that need sync
		for key, cacheItem := range itemsToSync {
			select {
			case syncQueue <- syncTask{
				Type:      "residueLuckyBag",
				Key:       key,
				Data:      cacheItem.ResidueLuckyBag,
				Timestamp: now,
			}:
				// Mark as synced
				if itemInterface, ok := cache.memoryCache.Load(key); ok {
					if item, ok := itemInterface.(*ResidueLuckyBagCacheItem); ok {
						item.IsDirty = false
						item.LastSyncTime = now
					}
				}
			default:
				log.Printf("Sync queue is full, cannot sync expiring residue lucky bag %s in group %s", key, groupId)
			}
		}

		return true // Continue iteration
	})
}

// ========== Cache Statistics and Monitoring ==========

// GetCacheStats Get cache statistics
func GetCacheStats() map[string]interface{} {
	stats := make(map[string]interface{})

	// Lucky bag cache statistics
	var totalLuckyBagHitCount, totalLuckyBagMissCount int64
	var totalLuckyBagSize int
	luckyBagCacheMap.Range(func(groupIdInterface, cacheInterface interface{}) bool {
		cache := cacheInterface.(*LuckyBagCache)
		totalLuckyBagHitCount += cache.hitCount
		totalLuckyBagMissCount += cache.missCount
		cache.memoryCache.Range(func(key, value interface{}) bool {
			totalLuckyBagSize++
			return true
		})
		return true
	})

	luckyBagHitRate := float64(0)
	if totalLuckyBagHitCount+totalLuckyBagMissCount > 0 {
		luckyBagHitRate = float64(totalLuckyBagHitCount) / float64(totalLuckyBagHitCount+totalLuckyBagMissCount) * 100
	}
	stats["luckyBag"] = map[string]interface{}{
		"hitCount":  totalLuckyBagHitCount,
		"missCount": totalLuckyBagMissCount,
		"hitRate":   fmt.Sprintf("%.2f%%", luckyBagHitRate),
		"size":      totalLuckyBagSize,
	}

	// Open list cache statistics
	var totalOpenListHitCount, totalOpenListMissCount int64
	var totalOpenListSize int
	openLuckyBagListCacheMap.Range(func(groupIdInterface, cacheInterface interface{}) bool {
		cache := cacheInterface.(*LuckyBagCache)
		totalOpenListHitCount += cache.hitCount
		totalOpenListMissCount += cache.missCount
		cache.memoryCache.Range(func(key, value interface{}) bool {
			totalOpenListSize++
			return true
		})
		return true
	})

	openListHitRate := float64(0)
	if totalOpenListHitCount+totalOpenListMissCount > 0 {
		openListHitRate = float64(totalOpenListHitCount) / float64(totalOpenListHitCount+totalOpenListMissCount) * 100
	}
	stats["openList"] = map[string]interface{}{
		"hitCount":  totalOpenListHitCount,
		"missCount": totalOpenListMissCount,
		"hitRate":   fmt.Sprintf("%.2f%%", openListHitRate),
		"size":      totalOpenListSize,
	}

	// Residue list cache statistics
	var totalResidueListHitCount, totalResidueListMissCount int64
	var totalResidueListSize int
	residueLuckyBagListCacheMap.Range(func(groupIdInterface, cacheInterface interface{}) bool {
		cache := cacheInterface.(*LuckyBagCache)
		totalResidueListHitCount += cache.hitCount
		totalResidueListMissCount += cache.missCount
		cache.memoryCache.Range(func(key, value interface{}) bool {
			totalResidueListSize++
			return true
		})
		return true
	})

	residueListHitRate := float64(0)
	if totalResidueListHitCount+totalResidueListMissCount > 0 {
		residueListHitRate = float64(totalResidueListHitCount) / float64(totalResidueListHitCount+totalResidueListMissCount) * 100
	}
	stats["residueList"] = map[string]interface{}{
		"hitCount":  totalResidueListHitCount,
		"missCount": totalResidueListMissCount,
		"hitRate":   fmt.Sprintf("%.2f%%", residueListHitRate),
		"size":      totalResidueListSize,
	}

	// Open lucky bag detail cache statistics
	var totalOpenLuckyBagHitCount, totalOpenLuckyBagMissCount int64
	var totalOpenLuckyBagSize int
	openLuckyBagCacheMap.Range(func(groupIdInterface, cacheInterface interface{}) bool {
		cache := cacheInterface.(*LuckyBagCache)
		totalOpenLuckyBagHitCount += cache.hitCount
		totalOpenLuckyBagMissCount += cache.missCount
		cache.memoryCache.Range(func(key, value interface{}) bool {
			totalOpenLuckyBagSize++
			return true
		})
		return true
	})

	openLuckyBagHitRate := float64(0)
	if totalOpenLuckyBagHitCount+totalOpenLuckyBagMissCount > 0 {
		openLuckyBagHitRate = float64(totalOpenLuckyBagHitCount) / float64(totalOpenLuckyBagHitCount+totalOpenLuckyBagMissCount) * 100
	}
	stats["openLuckyBag"] = map[string]interface{}{
		"hitCount":  totalOpenLuckyBagHitCount,
		"missCount": totalOpenLuckyBagMissCount,
		"hitRate":   fmt.Sprintf("%.2f%%", openLuckyBagHitRate),
		"size":      totalOpenLuckyBagSize,
	}

	// Residue lucky bag detail cache statistics
	var totalResidueLuckyBagHitCount, totalResidueLuckyBagMissCount int64
	var totalResidueLuckyBagSize int
	residueLuckyBagCacheMap.Range(func(groupIdInterface, cacheInterface interface{}) bool {
		cache := cacheInterface.(*LuckyBagCache)
		totalResidueLuckyBagHitCount += cache.hitCount
		totalResidueLuckyBagMissCount += cache.missCount
		cache.memoryCache.Range(func(key, value interface{}) bool {
			totalResidueLuckyBagSize++
			return true
		})
		return true
	})

	residueLuckyBagHitRate := float64(0)
	if totalResidueLuckyBagHitCount+totalResidueLuckyBagMissCount > 0 {
		residueLuckyBagHitRate = float64(totalResidueLuckyBagHitCount) / float64(totalResidueLuckyBagHitCount+totalResidueLuckyBagMissCount) * 100
	}
	stats["residueLuckyBag"] = map[string]interface{}{
		"hitCount":  totalResidueLuckyBagHitCount,
		"missCount": totalResidueLuckyBagMissCount,
		"hitRate":   fmt.Sprintf("%.2f%%", residueLuckyBagHitRate),
		"size":      totalResidueLuckyBagSize,
	}

	// Sync queue statistics
	stats["syncQueue"] = map[string]interface{}{
		"length": len(syncQueue),
		"cap":    cap(syncQueue),
	}

	return stats
}

// ClearCacheStats Clear cache statistics
func ClearCacheStats() {
	// Clear lucky bag cache statistics
	luckyBagCacheMap.Range(func(groupIdInterface, cacheInterface interface{}) bool {
		cache := cacheInterface.(*LuckyBagCache)
		cache.hitCount = 0
		cache.missCount = 0
		return true
	})

	// Clear open list cache statistics
	openLuckyBagListCacheMap.Range(func(groupIdInterface, cacheInterface interface{}) bool {
		cache := cacheInterface.(*LuckyBagCache)
		cache.hitCount = 0
		cache.missCount = 0
		return true
	})

	// Clear residue list cache statistics
	residueLuckyBagListCacheMap.Range(func(groupIdInterface, cacheInterface interface{}) bool {
		cache := cacheInterface.(*LuckyBagCache)
		cache.hitCount = 0
		cache.missCount = 0
		return true
	})

	// Clear open lucky bag detail cache statistics
	openLuckyBagCacheMap.Range(func(groupIdInterface, cacheInterface interface{}) bool {
		cache := cacheInterface.(*LuckyBagCache)
		cache.hitCount = 0
		cache.missCount = 0
		return true
	})

	// Clear residue lucky bag detail cache statistics
	residueLuckyBagCacheMap.Range(func(groupIdInterface, cacheInterface interface{}) bool {
		cache := cacheInterface.(*LuckyBagCache)
		cache.hitCount = 0
		cache.missCount = 0
		return true
	})
}

// ClearAllCache Clear all cache
func ClearAllCache() {
	// Clear lucky bag cache
	luckyBagCacheMap.Range(func(groupIdInterface, cacheInterface interface{}) bool {
		cache := cacheInterface.(*LuckyBagCache)
		cache.memoryCache.Range(func(key, value interface{}) bool {
			cache.memoryCache.Delete(key)
			return true
		})
		cache.hitCount = 0
		cache.missCount = 0
		return true
	})

	// Clear open list cache
	openLuckyBagListCacheMap.Range(func(groupIdInterface, cacheInterface interface{}) bool {
		cache := cacheInterface.(*LuckyBagCache)
		cache.memoryCache.Range(func(key, value interface{}) bool {
			cache.memoryCache.Delete(key)
			return true
		})
		cache.hitCount = 0
		cache.missCount = 0
		return true
	})

	// Clear residue list cache
	residueLuckyBagListCacheMap.Range(func(groupIdInterface, cacheInterface interface{}) bool {
		cache := cacheInterface.(*LuckyBagCache)
		cache.memoryCache.Range(func(key, value interface{}) bool {
			cache.memoryCache.Delete(key)
			return true
		})
		cache.hitCount = 0
		cache.missCount = 0
		return true
	})

	// Clear open lucky bag detail cache
	openLuckyBagCacheMap.Range(func(groupIdInterface, cacheInterface interface{}) bool {
		cache := cacheInterface.(*LuckyBagCache)
		cache.memoryCache.Range(func(key, value interface{}) bool {
			cache.memoryCache.Delete(key)
			return true
		})
		cache.hitCount = 0
		cache.missCount = 0
		return true
	})

	// Clear residue lucky bag detail cache
	residueLuckyBagCacheMap.Range(func(groupIdInterface, cacheInterface interface{}) bool {
		cache := cacheInterface.(*LuckyBagCache)
		cache.memoryCache.Range(func(key, value interface{}) bool {
			cache.memoryCache.Delete(key)
			return true
		})
		cache.hitCount = 0
		cache.missCount = 0
		return true
	})

	log.Printf("All lucky bag caches cleared")
}
