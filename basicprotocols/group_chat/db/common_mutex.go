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
