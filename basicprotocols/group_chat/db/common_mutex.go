package db

import (
	"log"
	"sync"
	"sync/atomic"
	"time"
)

// MetaIdMutexItem represents a mutex item for a specific MetaId
type MetaIdMutexItem struct {
	mutex       *sync.Mutex
	lastUsed    time.Time
	accessCount int64
}

// CommonMutexManager manages mutexes for MetaId-based operations
type CommonMutexManager struct {
	metaIdMutexMap sync.Map
	cleanupRunning int32 // Atomic flag to track if cleanup is running
}

// Global instance of the mutex manager
var GlobalMetaIdMutexManager *CommonMutexManager
var once sync.Once

// NewCommonMutexManager creates a new mutex manager
func NewCommonMutexManager() *CommonMutexManager {
	manager := &CommonMutexManager{}

	// Start cleanup goroutine
	go manager.startCleanupGoroutine()

	return manager
}

// GetGlobalMetaIdMutexManager returns the singleton instance of the mutex manager
func GetGlobalMetaIdMutexManager() *CommonMutexManager {
	once.Do(func() {
		GlobalMetaIdMutexManager = NewCommonMutexManager()
	})
	return GlobalMetaIdMutexManager
}

// GetMetaIdMutex gets or creates a mutex for a specific MetaId
func (cmm *CommonMutexManager) GetMetaIdMutex(metaId string) *sync.Mutex {
	// Try to get existing lock from sync.Map
	if value, exists := cmm.metaIdMutexMap.Load(metaId); exists {
		if item, ok := value.(*MetaIdMutexItem); ok {
			// Update access statistics
			item.lastUsed = time.Now()
			item.accessCount++
			return item.mutex
		}
	}

	// If not exists, create new lock item
	newItem := &MetaIdMutexItem{
		mutex:       &sync.Mutex{},
		lastUsed:    time.Now(),
		accessCount: 1,
	}

	// Use LoadOrStore to ensure atomicity, avoid duplicate creation
	if value, loaded := cmm.metaIdMutexMap.LoadOrStore(metaId, newItem); loaded {
		// If already exists, return existing lock and update statistics
		if item, ok := value.(*MetaIdMutexItem); ok {
			item.lastUsed = time.Now()
			item.accessCount++
			return item.mutex
		}
	}

	// Return newly created lock
	return newItem.mutex
}

// startCleanupGoroutine starts the cleanup goroutine
func (cmm *CommonMutexManager) startCleanupGoroutine() {
	ticker := time.NewTicker(5 * time.Minute) // Every 5 minutes
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Check if cleanup is already running, if so, skip this iteration
			if atomic.CompareAndSwapInt32(&cmm.cleanupRunning, 0, 1) {
				// Set flag to indicate cleanup is running
				cmm.cleanupUnusedMutexes()
				// Reset flag when cleanup is done
				atomic.StoreInt32(&cmm.cleanupRunning, 0)
			} else {
				// Cleanup is already running, skip this iteration
				log.Printf("Cleanup already running, skipping this iteration")
			}
		}
	}
}

// cleanupUnusedMutexes cleans up unused mutexes
func (cmm *CommonMutexManager) cleanupUnusedMutexes() {
	now := time.Now()
	cleanupThreshold := 30 * time.Minute // 30 minutes not used to clean up

	var keysToDelete []string

	// Traverse all mutexes, find mutexes to clean up
	cmm.metaIdMutexMap.Range(func(key, value interface{}) bool {
		if item, ok := value.(*MetaIdMutexItem); ok {
			// Check if it exceeds the cleanup threshold
			if now.Sub(item.lastUsed) > cleanupThreshold {
				keysToDelete = append(keysToDelete, key.(string))
			}
		}
		return true
	})

	// Delete unused mutexes
	for _, key := range keysToDelete {
		cmm.metaIdMutexMap.Delete(key)
	}

	if len(keysToDelete) > 0 {
		log.Printf("Cleaned up %d unused MetaId mutexes", len(keysToDelete))
	}
}

// CleanupMetaIdMutex manually cleans up a specific MetaId mutex
func (cmm *CommonMutexManager) CleanupMetaIdMutex(metaId string) bool {
	// Check if the mutex is being used
	if value, exists := cmm.metaIdMutexMap.Load(metaId); exists {
		if item, ok := value.(*MetaIdMutexItem); ok {
			// If the mutex is being used (a goroutine holds the lock), it cannot be deleted
			// Here we use a simple heuristic: if there has been access in the last 5 minutes, it will not be deleted
			if time.Since(item.lastUsed) < 5*time.Minute {
				return false // The mutex is still being used
			}
		}
	}

	// Delete mutex
	cmm.metaIdMutexMap.Delete(metaId)
	return true
}

// GetMetaIdMutexStats gets mutex statistics
func (cmm *CommonMutexManager) GetMetaIdMutexStats() map[string]interface{} {
	stats := make(map[string]interface{})
	totalMutexes := 0
	activeMutexes := 0
	now := time.Now()

	cmm.metaIdMutexMap.Range(func(key, value interface{}) bool {
		totalMutexes++
		if item, ok := value.(*MetaIdMutexItem); ok {
			// If there has been access in the last 5 minutes, it is considered active
			if now.Sub(item.lastUsed) < 5*time.Minute {
				activeMutexes++
			}
		}
		return true
	})

	stats["totalMutexes"] = totalMutexes
	stats["activeMutexes"] = activeMutexes
	stats["inactiveMutexes"] = totalMutexes - activeMutexes

	return stats
}

// Convenience function to get MetaId mutex
func GetMetaIdMutex(metaId string) *sync.Mutex {
	manager := GetGlobalMetaIdMutexManager()
	if manager == nil {
		// Fallback: create a temporary mutex if manager is nil (should not happen in normal cases)
		log.Printf("Warning: GlobalMetaIdMutexManager is nil, creating temporary mutex for MetaId: %s", metaId)
		return &sync.Mutex{}
	}
	return manager.GetMetaIdMutex(metaId)
}

// GroupMetaIdJoinMutexItem represents a mutex item for a specific MetaId's group join operations
type GroupMetaIdJoinMutexItem struct {
	mutex       *sync.Mutex
	lastUsed    time.Time
	accessCount int64
}

// GroupMetaIdJoinMutexManager manages mutexes for MetaId group join operations
type GroupMetaIdJoinMutexManager struct {
	metaIdJoinMutexMap sync.Map
	cleanupRunning     int32 // Atomic flag to track if cleanup is running
}

// Global instance of the group join mutex manager
var GlobalGroupMetaIdJoinMutexManager *GroupMetaIdJoinMutexManager
var groupJoinOnce sync.Once

// NewGroupMetaIdJoinMutexManager creates a new group join mutex manager
func NewGroupMetaIdJoinMutexManager() *GroupMetaIdJoinMutexManager {
	manager := &GroupMetaIdJoinMutexManager{}

	// Start cleanup goroutine
	go manager.startCleanupGoroutine()

	return manager
}

// GetGlobalGroupMetaIdJoinMutexManager returns the singleton instance of the group join mutex manager
func GetGlobalGroupMetaIdJoinMutexManager() *GroupMetaIdJoinMutexManager {
	groupJoinOnce.Do(func() {
		GlobalGroupMetaIdJoinMutexManager = NewGroupMetaIdJoinMutexManager()
	})
	return GlobalGroupMetaIdJoinMutexManager
}

// GetGroupMetaIdJoinMutex gets or creates a mutex for a specific MetaId's group join operations
func (gmjmm *GroupMetaIdJoinMutexManager) GetGroupMetaIdJoinMutex(metaId string) *sync.Mutex {
	// Try to get existing lock from sync.Map
	if value, exists := gmjmm.metaIdJoinMutexMap.Load(metaId); exists {
		if item, ok := value.(*GroupMetaIdJoinMutexItem); ok {
			// Update access statistics
			item.lastUsed = time.Now()
			item.accessCount++
			return item.mutex
		}
	}

	// If not exists, create new lock item
	newItem := &GroupMetaIdJoinMutexItem{
		mutex:       &sync.Mutex{},
		lastUsed:    time.Now(),
		accessCount: 1,
	}

	// Use LoadOrStore to ensure atomicity, avoid duplicate creation
	if value, loaded := gmjmm.metaIdJoinMutexMap.LoadOrStore(metaId, newItem); loaded {
		// If already exists, return existing lock and update statistics
		if item, ok := value.(*GroupMetaIdJoinMutexItem); ok {
			item.lastUsed = time.Now()
			item.accessCount++
			return item.mutex
		}
	}

	// Return newly created lock
	return newItem.mutex
}

// startCleanupGoroutine starts the cleanup goroutine for group join mutexes
func (gmjmm *GroupMetaIdJoinMutexManager) startCleanupGoroutine() {
	ticker := time.NewTicker(5 * time.Minute) // Every 5 minutes
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Check if cleanup is already running, if so, skip this iteration
			if atomic.CompareAndSwapInt32(&gmjmm.cleanupRunning, 0, 1) {
				// Set flag to indicate cleanup is running
				gmjmm.cleanupUnusedGroupJoinMutexes()
				// Reset flag when cleanup is done
				atomic.StoreInt32(&gmjmm.cleanupRunning, 0)
			} else {
				// Cleanup is already running, skip this iteration
				log.Printf("Group join cleanup already running, skipping this iteration")
			}
		}
	}
}

// cleanupUnusedGroupJoinMutexes cleans up unused group join mutexes
func (gmjmm *GroupMetaIdJoinMutexManager) cleanupUnusedGroupJoinMutexes() {
	now := time.Now()
	cleanupThreshold := 30 * time.Minute // 30 minutes not used to clean up

	var keysToDelete []string

	// Traverse all mutexes, find mutexes to clean up
	gmjmm.metaIdJoinMutexMap.Range(func(key, value interface{}) bool {
		if item, ok := value.(*GroupMetaIdJoinMutexItem); ok {
			// Check if it exceeds the cleanup threshold
			if now.Sub(item.lastUsed) > cleanupThreshold {
				keysToDelete = append(keysToDelete, key.(string))
			}
		}
		return true
	})

	// Delete unused mutexes
	for _, key := range keysToDelete {
		gmjmm.metaIdJoinMutexMap.Delete(key)
	}

	if len(keysToDelete) > 0 {
		log.Printf("Cleaned up %d unused GroupMetaIdJoin mutexes", len(keysToDelete))
	}
}

// CleanupGroupMetaIdJoinMutex manually cleans up a specific MetaId group join mutex
func (gmjmm *GroupMetaIdJoinMutexManager) CleanupGroupMetaIdJoinMutex(metaId string) bool {
	// Check if the mutex is being used
	if value, exists := gmjmm.metaIdJoinMutexMap.Load(metaId); exists {
		if item, ok := value.(*GroupMetaIdJoinMutexItem); ok {
			// If the mutex is being used (a goroutine holds the lock), it cannot be deleted
			// Here we use a simple heuristic: if there has been access in the last 5 minutes, it will not be deleted
			if time.Since(item.lastUsed) < 5*time.Minute {
				return false // The mutex is still being used
			}
		}
	}

	// Delete mutex
	gmjmm.metaIdJoinMutexMap.Delete(metaId)
	return true
}

// GetGroupMetaIdJoinMutexStats gets group join mutex statistics
func (gmjmm *GroupMetaIdJoinMutexManager) GetGroupMetaIdJoinMutexStats() map[string]interface{} {
	stats := make(map[string]interface{})
	totalMutexes := 0
	activeMutexes := 0
	now := time.Now()

	gmjmm.metaIdJoinMutexMap.Range(func(key, value interface{}) bool {
		totalMutexes++
		if item, ok := value.(*GroupMetaIdJoinMutexItem); ok {
			// If there has been access in the last 5 minutes, it is considered active
			if now.Sub(item.lastUsed) < 5*time.Minute {
				activeMutexes++
			}
		}
		return true
	})

	stats["totalGroupJoinMutexes"] = totalMutexes
	stats["activeGroupJoinMutexes"] = activeMutexes
	stats["inactiveGroupJoinMutexes"] = totalMutexes - activeMutexes

	return stats
}

// Convenience function to get GroupMetaIdJoin mutex
func GetGroupMetaIdJoinMutex(metaId string) *sync.Mutex {
	manager := GetGlobalGroupMetaIdJoinMutexManager()
	if manager == nil {
		// Fallback: create a temporary mutex if manager is nil (should not happen in normal cases)
		log.Printf("Warning: GlobalGroupMetaIdJoinMutexManager is nil, creating temporary mutex for MetaId: %s", metaId)
		return &sync.Mutex{}
	}
	return manager.GetGroupMetaIdJoinMutex(metaId)
}

// GroupAdminMutexItem represents a mutex item for a specific GroupId's admin operations
type GroupAdminMutexItem struct {
	mutex       *sync.Mutex
	lastUsed    time.Time
	accessCount int64
}

// GroupAdminMutexManager manages mutexes for GroupId admin operations
type GroupAdminMutexManager struct {
	groupAdminMutexMap sync.Map
	cleanupRunning     int32 // Atomic flag to track if cleanup is running
}

// Global instance of the group admin mutex manager
var GlobalGroupAdminMutexManager *GroupAdminMutexManager
var groupAdminOnce sync.Once

// NewGroupAdminMutexManager creates a new group admin mutex manager
func NewGroupAdminMutexManager() *GroupAdminMutexManager {
	manager := &GroupAdminMutexManager{}

	// Start cleanup goroutine
	go manager.startCleanupGoroutine()

	return manager
}

// GetGlobalGroupAdminMutexManager returns the singleton instance of the group admin mutex manager
func GetGlobalGroupAdminMutexManager() *GroupAdminMutexManager {
	groupAdminOnce.Do(func() {
		GlobalGroupAdminMutexManager = NewGroupAdminMutexManager()
	})
	return GlobalGroupAdminMutexManager
}

// GetGroupAdminMutex gets or creates a mutex for a specific GroupId's admin operations
func (gamm *GroupAdminMutexManager) GetGroupAdminMutex(groupId string) *sync.Mutex {
	// Try to get existing lock from sync.Map
	if value, exists := gamm.groupAdminMutexMap.Load(groupId); exists {
		if item, ok := value.(*GroupAdminMutexItem); ok {
			// Update access statistics
			item.lastUsed = time.Now()
			item.accessCount++
			return item.mutex
		}
	}

	// If not exists, create new lock item
	newItem := &GroupAdminMutexItem{
		mutex:       &sync.Mutex{},
		lastUsed:    time.Now(),
		accessCount: 1,
	}

	// Use LoadOrStore to ensure atomicity, avoid duplicate creation
	if value, loaded := gamm.groupAdminMutexMap.LoadOrStore(groupId, newItem); loaded {
		// If already exists, return existing lock and update statistics
		if item, ok := value.(*GroupAdminMutexItem); ok {
			item.lastUsed = time.Now()
			item.accessCount++
			return item.mutex
		}
	}

	// Return newly created lock
	return newItem.mutex
}

// startCleanupGoroutine starts the cleanup goroutine for group admin mutexes
func (gamm *GroupAdminMutexManager) startCleanupGoroutine() {
	ticker := time.NewTicker(5 * time.Minute) // Every 5 minutes
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Check if cleanup is already running, if so, skip this iteration
			if atomic.CompareAndSwapInt32(&gamm.cleanupRunning, 0, 1) {
				// Set flag to indicate cleanup is running
				gamm.cleanupUnusedGroupAdminMutexes()
				// Reset flag when cleanup is done
				atomic.StoreInt32(&gamm.cleanupRunning, 0)
			} else {
				// Cleanup is already running, skip this iteration
				log.Printf("Group admin cleanup already running, skipping this iteration")
			}
		}
	}
}

// cleanupUnusedGroupAdminMutexes cleans up unused group admin mutexes
func (gamm *GroupAdminMutexManager) cleanupUnusedGroupAdminMutexes() {
	now := time.Now()
	cleanupThreshold := 30 * time.Minute // 30 minutes not used to clean up

	var keysToDelete []string

	// Traverse all mutexes, find mutexes to clean up
	gamm.groupAdminMutexMap.Range(func(key, value interface{}) bool {
		if item, ok := value.(*GroupAdminMutexItem); ok {
			// Check if it exceeds the cleanup threshold
			if now.Sub(item.lastUsed) > cleanupThreshold {
				keysToDelete = append(keysToDelete, key.(string))
			}
		}
		return true
	})

	// Delete unused mutexes
	for _, key := range keysToDelete {
		gamm.groupAdminMutexMap.Delete(key)
	}

	if len(keysToDelete) > 0 {
		log.Printf("Cleaned up %d unused GroupAdmin mutexes", len(keysToDelete))
	}
}

// Convenience function to get GroupAdmin mutex
func GetGroupAdminMutex(groupId string) *sync.Mutex {
	manager := GetGlobalGroupAdminMutexManager()
	if manager == nil {
		// Fallback: create a temporary mutex if manager is nil (should not happen in normal cases)
		log.Printf("Warning: GlobalGroupAdminMutexManager is nil, creating temporary mutex for GroupId: %s", groupId)
		return &sync.Mutex{}
	}
	return manager.GetGroupAdminMutex(groupId)
}

// GroupBlockMutexItem represents a mutex item for a specific GroupId's block operations
type GroupBlockMutexItem struct {
	mutex       *sync.Mutex
	lastUsed    time.Time
	accessCount int64
}

// GroupBlockMutexManager manages mutexes for GroupId block operations
type GroupBlockMutexManager struct {
	groupBlockMutexMap sync.Map
	cleanupRunning     int32 // Atomic flag to track if cleanup is running
}

// Global instance of the group block mutex manager
var GlobalGroupBlockMutexManager *GroupBlockMutexManager
var groupBlockOnce sync.Once

// NewGroupBlockMutexManager creates a new group block mutex manager
func NewGroupBlockMutexManager() *GroupBlockMutexManager {
	manager := &GroupBlockMutexManager{}

	// Start cleanup goroutine
	go manager.startCleanupGoroutine()

	return manager
}

// GetGlobalGroupBlockMutexManager returns the singleton instance of the group block mutex manager
func GetGlobalGroupBlockMutexManager() *GroupBlockMutexManager {
	groupBlockOnce.Do(func() {
		GlobalGroupBlockMutexManager = NewGroupBlockMutexManager()
	})
	return GlobalGroupBlockMutexManager
}

// GetGroupBlockMutex gets or creates a mutex for a specific GroupId's block operations
func (gbmm *GroupBlockMutexManager) GetGroupBlockMutex(groupId string) *sync.Mutex {
	// Try to get existing lock from sync.Map
	if value, exists := gbmm.groupBlockMutexMap.Load(groupId); exists {
		if item, ok := value.(*GroupBlockMutexItem); ok {
			// Update access statistics
			item.lastUsed = time.Now()
			item.accessCount++
			return item.mutex
		}
	}

	// If not exists, create new lock item
	newItem := &GroupBlockMutexItem{
		mutex:       &sync.Mutex{},
		lastUsed:    time.Now(),
		accessCount: 1,
	}

	// Use LoadOrStore to ensure atomicity, avoid duplicate creation
	if value, loaded := gbmm.groupBlockMutexMap.LoadOrStore(groupId, newItem); loaded {
		// If already exists, return existing lock and update statistics
		if item, ok := value.(*GroupBlockMutexItem); ok {
			item.lastUsed = time.Now()
			item.accessCount++
			return item.mutex
		}
	}

	// Return newly created lock
	return newItem.mutex
}

// startCleanupGoroutine starts the cleanup goroutine for group block mutexes
func (gbmm *GroupBlockMutexManager) startCleanupGoroutine() {
	ticker := time.NewTicker(5 * time.Minute) // Every 5 minutes
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Check if cleanup is already running, if so, skip this iteration
			if atomic.CompareAndSwapInt32(&gbmm.cleanupRunning, 0, 1) {
				// Set flag to indicate cleanup is running
				gbmm.cleanupUnusedGroupBlockMutexes()
				// Reset flag when cleanup is done
				atomic.StoreInt32(&gbmm.cleanupRunning, 0)
			} else {
				// Cleanup is already running, skip this iteration
				log.Printf("Group block cleanup already running, skipping this iteration")
			}
		}
	}
}

// cleanupUnusedGroupBlockMutexes cleans up unused group block mutexes
func (gbmm *GroupBlockMutexManager) cleanupUnusedGroupBlockMutexes() {
	now := time.Now()
	cleanupThreshold := 30 * time.Minute // 30 minutes not used to clean up

	var keysToDelete []string

	// Traverse all mutexes, find mutexes to clean up
	gbmm.groupBlockMutexMap.Range(func(key, value interface{}) bool {
		if item, ok := value.(*GroupBlockMutexItem); ok {
			// Check if it exceeds the cleanup threshold
			if now.Sub(item.lastUsed) > cleanupThreshold {
				keysToDelete = append(keysToDelete, key.(string))
			}
		}
		return true
	})

	// Delete unused mutexes
	for _, key := range keysToDelete {
		gbmm.groupBlockMutexMap.Delete(key)
	}

	if len(keysToDelete) > 0 {
		log.Printf("Cleaned up %d unused GroupBlock mutexes", len(keysToDelete))
	}
}

// Convenience function to get GroupBlock mutex
func GetGroupBlockMutex(groupId string) *sync.Mutex {
	manager := GetGlobalGroupBlockMutexManager()
	if manager == nil {
		// Fallback: create a temporary mutex if manager is nil (should not happen in normal cases)
		log.Printf("Warning: GlobalGroupBlockMutexManager is nil, creating temporary mutex for GroupId: %s", groupId)
		return &sync.Mutex{}
	}
	return manager.GetGroupBlockMutex(groupId)
}

// GroupWhitelistMutexItem represents a mutex item for a specific GroupId's whitelist operations
type GroupWhitelistMutexItem struct {
	mutex       *sync.Mutex
	lastUsed    time.Time
	accessCount int64
}

// GroupWhitelistMutexManager manages mutexes for GroupId whitelist operations
type GroupWhitelistMutexManager struct {
	groupWhitelistMutexMap sync.Map
	cleanupRunning         int32 // Atomic flag to track if cleanup is running
}

// Global instance of the group whitelist mutex manager
var GlobalGroupWhitelistMutexManager *GroupWhitelistMutexManager
var groupWhitelistOnce sync.Once

// NewGroupWhitelistMutexManager creates a new group whitelist mutex manager
func NewGroupWhitelistMutexManager() *GroupWhitelistMutexManager {
	manager := &GroupWhitelistMutexManager{}

	// Start cleanup goroutine
	go manager.startCleanupGoroutine()

	return manager
}

// GetGlobalGroupWhitelistMutexManager returns the singleton instance of the group whitelist mutex manager
func GetGlobalGroupWhitelistMutexManager() *GroupWhitelistMutexManager {
	groupWhitelistOnce.Do(func() {
		GlobalGroupWhitelistMutexManager = NewGroupWhitelistMutexManager()
	})
	return GlobalGroupWhitelistMutexManager
}

// GetGroupWhitelistMutex gets or creates a mutex for a specific GroupId's whitelist operations
func (gwmm *GroupWhitelistMutexManager) GetGroupWhitelistMutex(groupId string) *sync.Mutex {
	// Try to get existing lock from sync.Map
	if value, exists := gwmm.groupWhitelistMutexMap.Load(groupId); exists {
		if item, ok := value.(*GroupWhitelistMutexItem); ok {
			// Update access statistics
			item.lastUsed = time.Now()
			item.accessCount++
			return item.mutex
		}
	}

	// If not exists, create new lock item
	newItem := &GroupWhitelistMutexItem{
		mutex:       &sync.Mutex{},
		lastUsed:    time.Now(),
		accessCount: 1,
	}

	// Use LoadOrStore to ensure atomicity, avoid duplicate creation
	if value, loaded := gwmm.groupWhitelistMutexMap.LoadOrStore(groupId, newItem); loaded {
		// If already exists, return existing lock and update statistics
		if item, ok := value.(*GroupWhitelistMutexItem); ok {
			item.lastUsed = time.Now()
			item.accessCount++
			return item.mutex
		}
	}

	// Return newly created lock
	return newItem.mutex
}

// startCleanupGoroutine starts the cleanup goroutine for group whitelist mutexes
func (gwmm *GroupWhitelistMutexManager) startCleanupGoroutine() {
	ticker := time.NewTicker(5 * time.Minute) // Every 5 minutes
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Check if cleanup is already running, if so, skip this iteration
			if atomic.CompareAndSwapInt32(&gwmm.cleanupRunning, 0, 1) {
				// Set flag to indicate cleanup is running
				gwmm.cleanupUnusedGroupWhitelistMutexes()
				// Reset flag when cleanup is done
				atomic.StoreInt32(&gwmm.cleanupRunning, 0)
			} else {
				// Cleanup is already running, skip this iteration
				log.Printf("Group whitelist cleanup already running, skipping this iteration")
			}
		}
	}
}

// cleanupUnusedGroupWhitelistMutexes cleans up unused group whitelist mutexes
func (gwmm *GroupWhitelistMutexManager) cleanupUnusedGroupWhitelistMutexes() {
	now := time.Now()
	cleanupThreshold := 30 * time.Minute // 30 minutes not used to clean up

	var keysToDelete []string

	// Traverse all mutexes, find mutexes to clean up
	gwmm.groupWhitelistMutexMap.Range(func(key, value interface{}) bool {
		if item, ok := value.(*GroupWhitelistMutexItem); ok {
			// Check if it exceeds the cleanup threshold
			if now.Sub(item.lastUsed) > cleanupThreshold {
				keysToDelete = append(keysToDelete, key.(string))
			}
		}
		return true
	})

	// Delete unused mutexes
	for _, key := range keysToDelete {
		gwmm.groupWhitelistMutexMap.Delete(key)
	}

	if len(keysToDelete) > 0 {
		log.Printf("Cleaned up %d unused GroupWhitelist mutexes", len(keysToDelete))
	}
}

// Convenience function to get GroupWhitelist mutex
func GetGroupWhitelistMutex(groupId string) *sync.Mutex {
	manager := GetGlobalGroupWhitelistMutexManager()
	if manager == nil {
		// Fallback: create a temporary mutex if manager is nil (should not happen in normal cases)
		log.Printf("Warning: GlobalGroupWhitelistMutexManager is nil, creating temporary mutex for GroupId: %s", groupId)
		return &sync.Mutex{}
	}
	return manager.GetGroupWhitelistMutex(groupId)
}
