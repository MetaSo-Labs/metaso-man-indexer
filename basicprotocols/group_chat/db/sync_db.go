package db

import (
	"encoding/json"
	"fmt"
	"log"
	"manindexer/adapter"
	"manindexer/basicprotocols/group_chat/protocols"
	"manindexer/blockfile"
	"manindexer/common"
	"manindexer/pin"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cockroachdb/pebble"
)

// SyncDB Sync info database operations
type SyncDB struct {
	pb *Pebble
}

// NewSyncDB Create sync info database instance
func NewSyncDB(pb *Pebble) *SyncDB {
	return &SyncDB{
		pb: pb,
	}
}

// SyncInfo Sync info for time-based synchronization across chains
type SyncInfo struct {
	Timestamp              int64     `json:"timestamp"`              // Sync timestamp (seconds)
	LastSyncTime           int64     `json:"lastSyncTime"`           // Last synced timestamp (seconds)
	TargetSyncTime         int64     `json:"targetSyncTime"`         // Target synced timestamp (seconds)
	CurrentTime            int64     `json:"currentTime"`            // Current timestamp (seconds)
	LastSyncTimeRange      int64     `json:"lastSyncTimeRange"`      // Last synced time range end (seconds)
	FirstPinTime           int64     `json:"firstPinTime"`           // First pin timestamp (seconds)
	LastProcessedPinTime   int64     `json:"lastProcessedPinTime"`   // Last processed pin timestamp for resume (seconds)
	TotalPins              int       `json:"totalPins"`              // Total pins processed
	ProcessedPins          int       `json:"processedPins"`          // Processed pins count
	SuccessPins            int       `json:"successPins"`            // Successfully processed pins
	FailedPins             int       `json:"failedPins"`             // Failed pins count
	IsAutoSync             bool      `json:"isAutoSync"`             // Whether auto sync is enabled
	ErrorMessage           string    `json:"errorMessage"`           // Error message if any
	LastSyncStartTime      time.Time `json:"lastSyncStartTime"`      // Last sync start time
	LastSyncEndTime        time.Time `json:"lastSyncEndTime"`        // Last sync end time
	SyncInterval           int64     `json:"syncInterval"`           // Sync check interval in seconds
	BatchSize              int       `json:"batchSize"`              // Batch size for processing
	RetryCount             int       `json:"retryCount"`             // Retry count for failed syncs
	TimeRangeSize          int64     `json:"timeRangeSize"`          // Time range size in seconds for each sync batch
	FristTimeSyncStartTime int64     `json:"fristTimeSyncStartTime"` // First time sync start time
	FristTimeSyncEndTime   int64     `json:"fristTimeSyncEndTime"`   // First time sync end time

	LastIsCompleted     bool  `json:"lastIsCompleted"`     // Last is completed
	LastIsCompletedTime int64 `json:"lastIsCompletedTime"` // Last is completed time

	// Block height tracking for each chain
	LastProcessedBlockHeight map[string]int64 `json:"lastProcessedBlockHeight"` // Last processed block height for each chain
	CurrentBlockHeight       map[string]int64 `json:"currentBlockHeight"`       // Current block height for each chain
	MaxProcessedBlockHeight  map[string]int64 `json:"maxProcessedBlockHeight"`  // Maximum processed block height for each chain
}

// SyncDBService Sync database service with continuous monitoring
type SyncDBService struct {
	syncDB     *SyncDB
	isRunning  bool
	mu         sync.RWMutex
	cancelChan chan struct{}
	// Callback function for processing pins
	processPinFunc func(pin *pin.PinInscription, tx interface{}, isResync bool) error
	// Chain adapters
	chainAdapter map[string]adapter.Chain

	// Auto sync settings
	syncInterval  int64 // Sync check interval in seconds
	batchSize     int   // Batch size for processing pins
	timeRangeSize int64 // Time range size in seconds for each sync batch

	// Sync hosts and protocols
	hosts     []string
	protocols []string

	// Global sync completion status
	isSyncCompleted bool // Global flag indicating if sync is completed

	isGroupChatIndexingCompleted   bool // Global flag indicating if group chat indexing is completed
	groupChatIndexingRemaining     int  // Global flag indicating if group chat indexing is completed
	isPrivateChatIndexingCompleted bool // Global flag indicating if private chat indexing is completed
	privateChatIndexingRemaining   int  // Global flag indicating if private chat indexing is completed

	// In-memory sync status (not persisted to database)
	isSyncing bool // Whether currently syncing (memory only)
}

// NewSyncDBService Create new sync database service instance
func NewSyncDBService(pb *Pebble, adapter map[string]adapter.Chain) *SyncDBService {
	log.Printf("[SYNC]Hosts: %v", common.Config.SyncHost)
	log.Printf("[SYNC]Protocols: %v", protocols.ProtocolList)

	service := &SyncDBService{
		syncDB:        NewSyncDB(pb),
		isRunning:     false,
		cancelChan:    make(chan struct{}),
		chainAdapter:  adapter,
		syncInterval:  30,   // Default 30 seconds
		batchSize:     1000, // Default batch size
		timeRangeSize: 3600, // Default 1 hour time range
		hosts:         common.Config.SyncHost,
		protocols:     protocols.ProtocolList,
	}

	// Check last sync completion status on initialization
	service.checkLastSyncCompletionOnInit()

	return service
}

// SetProcessPinFunc Set callback function for processing pins
func (s *SyncDBService) SetProcessPinFunc(fn func(pin *pin.PinInscription, tx interface{}, isResync bool) error) {
	s.processPinFunc = fn
}

// checkLastSyncCompletionOnInit Check last sync completion status on initialization
func (s *SyncDBService) checkLastSyncCompletionOnInit() {
	// Get current sync info
	syncInfo, err := s.syncDB.GetSyncInfo()
	if err != nil {
		log.Printf("[SYNC]Failed to get sync info during initialization: %v", err)
		return
	}

	if syncInfo == nil {
		log.Printf("[SYNC]No sync info found during initialization")
		return
	}

	if common.Config.GroupChat.IsMempoolDataWaitingResync {
		// Check if last sync was completed and within 10 minutes
		if syncInfo.LastIsCompleted && syncInfo.LastIsCompletedTime > 0 {
			currentTime := time.Now().Unix()
			timeDiff := currentTime - syncInfo.LastIsCompletedTime

			// 10 minutes = 600 seconds
			if timeDiff <= 600 {
				s.isSyncCompleted = true
				log.Printf("[SYNC]Last sync was completed within 10 minutes (time diff: %d seconds), setting isSyncCompleted=true", timeDiff)
			} else {
				log.Printf("[SYNC]Last sync completion was %d seconds ago (more than 10 minutes), keeping isSyncCompleted=false", timeDiff)
			}
		} else {
			log.Printf("[SYNC]Last sync was not completed or no completion time recorded, keeping isSyncCompleted=false")
		}
	} else {
		s.isSyncCompleted = true
	}

	// Check queue indexing status
	// s.updateQueueIndexingStatus()
}

// SetSyncInterval Set sync check interval
func (s *SyncDBService) SetSyncInterval(intervalSeconds int64) {
	s.syncInterval = intervalSeconds
}

// updateQueueIndexingStatus Update queue indexing completion status (safe concurrent access)
func (s *SyncDBService) updateQueueIndexingStatus() {
	// Check group chat queue status with timeout protection
	groupChatRemaining := s.getGroupChatQueuePendingCount()
	if groupChatRemaining <= 0 {
		s.isGroupChatIndexingCompleted = true
		log.Printf("[SYNC]Group chat indexing completed or no data, remaining: %d", groupChatRemaining)
	} else {
		log.Printf("[SYNC]Group chat indexing in progress, approximately %d messages", groupChatRemaining)
	}
	s.groupChatIndexingRemaining = groupChatRemaining

	// Check private chat queue status with timeout protection
	privateChatRemaining := s.getPrivateChatQueuePendingCount()
	if privateChatRemaining <= 0 {
		s.isPrivateChatIndexingCompleted = true
		log.Printf("[SYNC]Private chat indexing completed or no data, remaining: %d", privateChatRemaining)
	} else {
		log.Printf("[SYNC]Private chat indexing in progress, approximately %d messages", privateChatRemaining)
	}
	s.privateChatIndexingRemaining = privateChatRemaining
}

// getGroupChatQueuePendingCount Get approximate pending count from group chat queue (safe concurrent read)
func (s *SyncDBService) getGroupChatQueuePendingCount() int {
	// Check if collection exists
	if _, exists := Pb[TalkGroupChatQueueCollection]; !exists {
		log.Printf("[SYNC]Group chat queue collection does not exist")
		return -1
	}

	// Use a simple estimation approach that's safe for concurrent access
	// Since we can't easily get metrics, use a limited iteration approach
	return s.getSimpleQueueCount(TalkGroupChatQueueCollection, 10) // Limit to 500 for safety
}

// getPrivateChatQueuePendingCount Get approximate pending count from private chat queue (safe concurrent read)
func (s *SyncDBService) getPrivateChatQueuePendingCount() int {
	// Check if collection exists
	if _, exists := Pb[TalkPrivateChatQueueCollection]; !exists {
		log.Printf("[SYNC]Private chat queue collection does not exist")
		return -1
	}

	// Use a simple estimation approach that's safe for concurrent access
	// Since we can't easily get metrics, use a limited iteration approach
	return s.getSimpleQueueCount(TalkPrivateChatQueueCollection, 10) // Limit to 500 for safety
}

// getSimpleQueueCount Get simple queue count with iteration limit for safety
func (s *SyncDBService) getSimpleQueueCount(collection string, maxCount int) int {
	iter, err := Pb[collection].NewIter(nil)
	if err != nil {
		log.Printf("[SYNC]Failed to create iterator for %s: %v", collection, err)
		return -1
	}
	defer iter.Close()

	count := 0
	for iter.First(); iter.Valid() && count < maxCount; iter.Next() {
		count++
	}

	return count
}

// SetBatchSize Set batch size for processing
func (s *SyncDBService) SetBatchSize(size int) {
	s.batchSize = size
}

// SetTimeRangeSize Set time range size for each sync batch
func (s *SyncDBService) SetTimeRangeSize(sizeSeconds int64) {
	s.timeRangeSize = sizeSeconds
}

// SaveSyncInfo Save sync info for time-based synchronization
func (sdb *SyncDB) SaveSyncInfo(syncInfo *SyncInfo) error {
	// Validate sync info
	if syncInfo == nil {
		return fmt.Errorf("sync info cannot be nil")
	}

	// Set timestamp
	syncInfo.Timestamp = time.Now().Unix()

	// Serialize data
	data, err := json.Marshal(syncInfo)
	if err != nil {
		return fmt.Errorf("failed to marshal sync info: %v", err)
	}

	// Save to database, using fixed key for time-based sync
	key := []byte("sync_time_based")
	err = Pb[TalkSyncInfoCollection].Set(key, data, pebble.Sync)
	if err != nil {
		return fmt.Errorf("failed to save sync info: %v", err)
	}

	log.Printf("[SYNC]Saved sync info: lastSyncTime=%d, currentTime=%d, processedPins=%d",
		syncInfo.LastSyncTime, syncInfo.CurrentTime, syncInfo.ProcessedPins)

	return nil
}

// GetSyncInfo Get sync info for time-based synchronization
func (sdb *SyncDB) GetSyncInfo() (*SyncInfo, error) {
	// Get from database
	key := []byte("sync_time_based")
	data, closer, err := Pb[TalkSyncInfoCollection].Get(key)
	if err != nil {
		if err == pebble.ErrNotFound {
			return nil, nil // Sync info not found
		}
		return nil, fmt.Errorf("failed to get sync info: %v", err)
	}
	defer closer.Close()

	// Deserialize data
	var syncInfo SyncInfo
	err = json.Unmarshal(data, &syncInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal sync info: %v", err)
	}

	return &syncInfo, nil
}

// GetAllSyncInfo Get sync info (now returns single sync info for time-based sync)
func (sdb *SyncDB) GetAllSyncInfo() (map[string]*SyncInfo, error) {
	syncInfo, err := sdb.GetSyncInfo()
	if err != nil {
		return nil, err
	}

	syncInfoMap := make(map[string]*SyncInfo)
	if syncInfo != nil {
		syncInfoMap["time_based"] = syncInfo
	}

	return syncInfoMap, nil
}

// DeleteSyncInfo Delete sync info for time-based synchronization
func (sdb *SyncDB) DeleteSyncInfo() error {
	// Delete from database
	key := []byte("sync_time_based")
	err := Pb[TalkSyncInfoCollection].Delete(key, pebble.Sync)
	if err != nil {
		return fmt.Errorf("failed to delete sync info: %v", err)
	}

	log.Printf("[SYNC]Deleted sync info")
	return nil
}

// StartAutoSync Start automatic sync monitoring service
func (s *SyncDBService) StartAutoSync() error {
	// Check if BlockFileDb is initialized
	if !s.isBlockFileDbInitialized() {
		log.Printf("[SYNC]BlockFileDb not initialized, please call blockfile.InitBlockFileDb() first")
		return fmt.Errorf("BlockFileDb not initialized, please call blockfile.InitBlockFileDb() first")
	}

	s.mu.Lock()
	if s.isRunning {
		s.mu.Unlock()
		return fmt.Errorf("auto sync service is already running")
	}
	s.isRunning = true
	s.mu.Unlock()

	// Recreate cancel channel
	close(s.cancelChan)
	s.cancelChan = make(chan struct{})

	log.Printf("[SYNC]Starting auto sync service with interval %d seconds", s.syncInterval)

	// Start auto sync goroutine
	go s.autoSyncLoop()

	return nil
}

// StopAutoSync Stop automatic sync monitoring service
func (s *SyncDBService) StopAutoSync() error {
	s.mu.Lock()
	if !s.isRunning {
		s.mu.Unlock()
		return fmt.Errorf("auto sync service is not running")
	}
	s.mu.Unlock()

	// Send cancel signal
	close(s.cancelChan)
	log.Println("[SYNC]Stopping auto sync service...")

	return nil
}

// IsRunning Check if auto sync service is running
func (s *SyncDBService) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.isRunning
}

// autoSyncLoop Main auto sync loop
func (s *SyncDBService) autoSyncLoop() {
	defer func() {
		s.mu.Lock()
		s.isRunning = false
		s.mu.Unlock()
		log.Println("[SYNC]Auto sync service stopped")
	}()

	ticker := time.NewTicker(time.Duration(s.syncInterval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.cancelChan:
			log.Println("[SYNC]Auto sync service cancelled")
			return
		case <-ticker.C:
			// Check sync status for time-based synchronization
			s.checkAndSyncByTime()
		}
	}
}

// checkAndSyncByTime Check and sync by time range across all chains
func (s *SyncDBService) checkAndSyncByTime() {
	log.Printf("[SYNC]checkAndSyncByTime started")

	// Get current sync info
	syncInfo, err := s.syncDB.GetSyncInfo()
	if err != nil {
		log.Printf("[SYNC]Failed to get sync info: %v", err)
		return
	}

	// Initialize sync info if not exists
	if syncInfo == nil {
		syncInfo = &SyncInfo{
			LastSyncTime:             0,
			TargetSyncTime:           0,
			FirstPinTime:             0,
			LastProcessedPinTime:     0,
			IsAutoSync:               true,
			SyncInterval:             s.syncInterval,
			BatchSize:                s.batchSize,
			TimeRangeSize:            s.timeRangeSize,
			RetryCount:               0,
			FristTimeSyncStartTime:   0,
			FristTimeSyncEndTime:     0,
			LastIsCompleted:          false,
			LastIsCompletedTime:      0,
			LastProcessedBlockHeight: make(map[string]int64),
			CurrentBlockHeight:       make(map[string]int64),
			MaxProcessedBlockHeight:  make(map[string]int64),
		}
	}

	// Get current time (use seconds for consistency with pin timestamps)
	currentTime := time.Now().Unix()
	syncInfo.CurrentTime = currentTime
	syncInfo.Timestamp = currentTime
	syncInfo.TargetSyncTime = currentTime

	// Check if we need to sync - use last processed pin time for resume
	startTime := syncInfo.LastProcessedPinTime
	if startTime == 0 {
		startTime = syncInfo.LastSyncTime
	}

	// If startTime is still 0, get the first pin timestamp from blockfile
	if startTime == 0 {
		firstPinTime, err := s.getFirstPinTimestamp()
		if err != nil {
			log.Printf("[SYNC]Failed to get first pin timestamp: %v", err)
			return
		}
		if firstPinTime > 0 {
			startTime = firstPinTime
			syncInfo.LastSyncTime = firstPinTime
			syncInfo.FirstPinTime = firstPinTime
			syncInfo.LastProcessedPinTime = firstPinTime
			log.Printf("[SYNC]Using first pin timestamp as start time: %d", firstPinTime)
		} else {
			log.Printf("[SYNC]No pins found in blockfile, skipping sync")
			return
		}
	}

	// Check if all chains are up to date before proceeding with sync
	log.Printf("[SYNC]Checking if all chains are up to date...")
	if !s.checkAllChainsUpToDate() {
		log.Printf("[SYNC]Some chains are not up to date, skipping sync")
		s.syncDB.SaveSyncInfo(syncInfo)
		return
	}
	log.Printf("[SYNC]All chains are up to date, proceeding with sync check")

	log.Printf("[SYNC]Time check: currentTime=%d, startTime=%d", currentTime, startTime)
	if currentTime <= startTime {
		// No new time range, update last sync time
		log.Printf("[SYNC]No new time range, skipping sync (currentTime <= startTime)")
		s.syncDB.SaveSyncInfo(syncInfo)
		return
	}

	// Check if already syncing (use in-memory status)
	log.Printf("[SYNC]Checking sync status: isSyncing=%v", s.isSyncing)
	if s.isSyncing {
		log.Printf("[SYNC]Sync is already in progress, skipping")
		return
	}

	// Start syncing new time range
	log.Printf("[SYNC]Syncing from time %d to %d", startTime, currentTime)
	s.syncByTimeRange(syncInfo, startTime, currentTime)
}

// syncByTimeRange Sync pins by time range across all chains
func (s *SyncDBService) syncByTimeRange(syncInfo *SyncInfo, startTime, targetTime int64) {
	// Mark as syncing (in-memory only)
	s.isSyncing = true
	syncInfo.LastSyncStartTime = time.Now()

	// Record first time sync start time if not already set
	if syncInfo.FristTimeSyncStartTime == 0 {
		syncInfo.FristTimeSyncStartTime = time.Now().Unix()
		log.Printf("[SYNC]Recording first time sync start time: %d", syncInfo.FristTimeSyncStartTime)
	}

	syncInfo.ErrorMessage = ""
	s.syncDB.SaveSyncInfo(syncInfo)

	// Process time ranges in batches
	totalProcessed := 0
	totalSuccess := 0
	totalFailed := 0

	// Use time range size in seconds (no conversion needed)
	timeRangeSeconds := s.timeRangeSize

	for currentStartTime := startTime; currentStartTime < targetTime; currentStartTime += timeRangeSeconds {
		// Check if cancelled
		select {
		case <-s.cancelChan:
			log.Printf("[SYNC]Sync cancelled at time %d", currentStartTime)
			s.isSyncing = false // Use in-memory status

			// Record first time sync end time if start time was recorded and end time not set yet
			// if syncInfo.FristTimeSyncStartTime > 0 && syncInfo.FristTimeSyncEndTime == 0 {
			// 	syncInfo.FristTimeSyncEndTime = time.Now().Unix()
			// 	log.Printf("[SYNC]Recording first time sync end time (cancelled): %d", syncInfo.FristTimeSyncEndTime)
			// }

			s.syncDB.SaveSyncInfo(syncInfo)
			return
		default:
		}

		// Calculate batch end time
		currentEndTime := currentStartTime + timeRangeSeconds
		if currentEndTime > targetTime {
			currentEndTime = targetTime
		}

		// Get pins for this time range
		pins, err := blockfile.QueryPinsByTimeRange(currentStartTime, currentEndTime, s.hosts, s.protocols)
		if err != nil {
			log.Printf("[SYNC]Failed to query pins for time range %d-%d: %v", currentStartTime, currentEndTime, err)
			syncInfo.RetryCount++
			continue
		}

		var (
			btcHeight int64
			mvcHeight int64
		)
		for _, pin := range pins {
			if pin.ChainName == "btc" {
				btcHeight = pin.GenesisHeight
			}
			if pin.ChainName == "mvc" {
				mvcHeight = pin.GenesisHeight
			}
		}

		//打印当前时间戳范围和pins数量
		log.Printf("[SYNC]Current time range %d-%d, %s-%s, found %d pins, btcHeight=%d, mvcHeight=%d", currentStartTime, currentEndTime, timeFormat(currentStartTime), timeFormat(currentEndTime), len(pins), btcHeight, mvcHeight)

		// Process pins
		processed, success, failed := s.processPins(pins, currentStartTime, currentEndTime, syncInfo)
		totalProcessed += processed
		totalSuccess += success
		totalFailed += failed

		// Update last processed pin time for resume capability
		if len(pins) > 0 {
			// Find the latest pin timestamp in this batch
			var latestPinTime int64
			for _, pin := range pins {
				if pin.Timestamp > latestPinTime {
					latestPinTime = pin.Timestamp
				}
			}
			if latestPinTime > syncInfo.LastProcessedPinTime {
				syncInfo.LastProcessedPinTime = latestPinTime
			}
		}

		// Update sync info
		syncInfo.LastSyncTime = currentEndTime
		syncInfo.TotalPins += processed
		syncInfo.ProcessedPins += processed
		syncInfo.SuccessPins += success
		syncInfo.FailedPins += failed
		s.syncDB.SaveSyncInfo(syncInfo)

		log.Printf("[SYNC]Synced time range %d-%d, processed %d pins (%d success, %d failed)",
			currentStartTime, currentEndTime, processed, success, failed)

		// Brief pause between batches
		time.Sleep(100 * time.Millisecond)
	}

	// Update current block height for each chain
	s.updateCurrentBlockHeights(syncInfo)

	// Check if sync is completed
	isCompleted := s.checkSyncCompletion(syncInfo)
	if isCompleted {
		// Record first time sync end time if start time was recorded and end time not set yet
		if syncInfo.FristTimeSyncStartTime > 0 && syncInfo.FristTimeSyncEndTime == 0 {
			syncInfo.FristTimeSyncEndTime = time.Now().Unix()
			log.Printf("[SYNC]Recording first time sync end time: %d", syncInfo.FristTimeSyncEndTime)
		}

		// Record last completion status and time
		syncInfo.LastIsCompleted = true
		syncInfo.LastIsCompletedTime = time.Now().Unix()
		log.Printf("[SYNC]Recording sync completion: LastIsCompleted=true, LastIsCompletedTime=%d", syncInfo.LastIsCompletedTime)

		// once sync is completed, set sync completed status
		s.SetSyncCompleted(isCompleted)
	} else {
		// Record last completion status as false
		syncInfo.LastIsCompleted = false
		// syncInfo.LastIsCompletedTime = time.Now().Unix()
		log.Printf("[SYNC]Recording sync completion: LastIsCompleted=false, LastIsCompletedTime=%d", syncInfo.LastIsCompletedTime)
	}

	// Mark sync as completed (in-memory only)
	s.isSyncing = false
	syncInfo.LastSyncEndTime = time.Now()

	syncInfo.RetryCount = 0
	s.syncDB.SaveSyncInfo(syncInfo)

	log.Printf("[SYNC]Sync completed, total processed %d pins (%d success, %d failed), sync status: %v",
		totalProcessed, totalSuccess, totalFailed, isCompleted)
}

// processPins Process a batch of pins by time range sequentially
func (s *SyncDBService) processPins(pins []pin.PinInscription, startTime, endTime int64, syncInfo *SyncInfo) (processed, success, failed int) {
	if len(pins) == 0 {
		return 0, 0, 0
	}

	// Process pins sequentially
	for i, pinNode := range pins {
		// Check if cancelled
		select {
		case <-s.cancelChan:
			log.Printf("[SYNC]Pin processing cancelled at index %d", i)
			return
		default:
		}

		// Call pin processing callback function
		if s.processPinFunc != nil {
			err := s.processPinFunc(&pinNode, nil, true)
			if err != nil {
				log.Printf("[SYNC]Failed to process pin [%s] %s: %v", pinNode.ChainName, pinNode.Path, err)
				failed++
			} else {
				success++
			}
		}

		// Update block height tracking for this chain
		if syncInfo != nil {
			chainName := pinNode.ChainName
			blockHeight := pinNode.GenesisHeight

			// Initialize maps if nil
			if syncInfo.LastProcessedBlockHeight == nil {
				syncInfo.LastProcessedBlockHeight = make(map[string]int64)
			}
			if syncInfo.MaxProcessedBlockHeight == nil {
				syncInfo.MaxProcessedBlockHeight = make(map[string]int64)
			}

			// Update last processed block height for this chain
			syncInfo.LastProcessedBlockHeight[chainName] = blockHeight

			// Update max processed block height for this chain
			if currentMax, exists := syncInfo.MaxProcessedBlockHeight[chainName]; !exists || blockHeight > currentMax {
				syncInfo.MaxProcessedBlockHeight[chainName] = blockHeight
			}
		}

		processed++

		// Log progress every 10 pins
		if processed%10 == 0 {
			log.Printf("[SYNC]Processed %d pins, success: %d, failed: %d", processed, success, failed)
		}
	}

	return processed, success, failed
}

// updateCurrentBlockHeights Update current block height for each chain
func (s *SyncDBService) updateCurrentBlockHeights(syncInfo *SyncInfo) {
	if syncInfo == nil {
		return
	}

	// Initialize CurrentBlockHeight map if nil
	if syncInfo.CurrentBlockHeight == nil {
		syncInfo.CurrentBlockHeight = make(map[string]int64)
	}

	// Get current block height for each chain
	for chainName, adapter := range s.chainAdapter {
		if adapter != nil {
			// Get current block height from chain adapter using GetBestHeight()
			currentHeight := adapter.GetBestHeight()
			if currentHeight > 0 {
				// Update current block height
				syncInfo.CurrentBlockHeight[chainName] = currentHeight
				log.Printf("[SYNC]Updated current block height for chain %s: %d", chainName, currentHeight)
			} else {
				log.Printf("[SYNC]Failed to get current block height for chain %s: returned %d", chainName, currentHeight)
			}
		}
	}
}

func timeFormat(timestamp int64) string {
	return time.Unix(timestamp, 0).Format("2006-01-02 15:04:05")
}

// checkSyncCompletion Check if sync is completed based on all chains block height with 5 block tolerance
func (s *SyncDBService) checkSyncCompletion(syncInfo *SyncInfo) bool {
	if syncInfo == nil {
		return false
	}

	// Check all chains block height
	for chainName, adapter := range s.chainAdapter {
		if adapter == nil {
			log.Printf("[SYNC]Chain %s adapter not found, skipping", chainName)
			continue
		}

		// Get current block height from chain adapter
		currentHeight := adapter.GetBestHeight()
		if currentHeight <= 0 {
			log.Printf("[SYNC]Chain %s current height is invalid: %d", chainName, currentHeight)
			return false
		}

		// Get the latest block height from PebbleDB (what we need to sync to)
		latestBlockHeight, err := s.getLatestBlockHeightFromPebble(chainName)
		if err != nil {
			log.Printf("[SYNC]Failed to get latest block height for chain %s from PebbleDB: %v", chainName, err)
			return false
		}

		log.Printf("[SYNC]Chain %s - current: %d, latest in PebbleDB: %d", chainName, currentHeight, latestBlockHeight)

		// Check if we have processed up to the latest block height
		lastProcessedHeight, exists := syncInfo.LastProcessedBlockHeight[chainName]
		if !exists {
			log.Printf("[SYNC]Chain %s last processed height not found", chainName)
			return false
		}

		// Check if all three heights are equal: currentHeight == latestBlockHeight == lastProcessedHeight
		if currentHeight != latestBlockHeight || latestBlockHeight != lastProcessedHeight || currentHeight != lastProcessedHeight {
			log.Printf("[SYNC]Chain %s sync not completed: current=%d, latest=%d, processed=%d (not all equal)",
				chainName, currentHeight, latestBlockHeight, lastProcessedHeight)
			return false
		}

		log.Printf("[SYNC]Chain %s sync completed: current=%d, latest=%d, processed=%d (all equal)",
			chainName, currentHeight, latestBlockHeight, lastProcessedHeight)
	}

	// Check if there are no new pins to process
	// Get current time and check if there are any new pins
	// currentTime := time.Now().Unix()
	startTime := syncInfo.LastSyncTime
	timeRangeSeconds := s.timeRangeSize

	// Check for new pins in the next time range
	newPins, err := blockfile.QueryPinsByTimeRange(startTime, startTime+timeRangeSeconds, s.hosts, s.protocols)
	if err != nil {
		log.Printf("[SYNC]Failed to check for new pins: %v", err)
		return false
	}

	// If there are new pins, sync is not completed
	if len(newPins) > 0 {
		log.Printf("[SYNC]Found %d new pins, sync not completed", len(newPins))
		return false
	}

	log.Printf("[SYNC]Sync completed: all chains up to date and no new pins")
	return true
}

// IsSyncCompleted Get global sync completion status
func (s *SyncDBService) IsSyncCompleted() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.isSyncCompleted
}

// SetSyncCompleted Set global sync completion status
func (s *SyncDBService) SetSyncCompleted(completed bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.isSyncCompleted = completed
	log.Printf("[SYNC]Sync completion status set to: %v", completed)
}

// GetSyncInfo Get current sync info for time-based synchronization
func (s *SyncDBService) GetSyncInfo() (*SyncInfo, error) {
	return s.syncDB.GetSyncInfo()
}

// GetAllSyncInfo Get sync info for all chains
func (s *SyncDBService) GetAllSyncInfo() (map[string]*SyncInfo, error) {
	return s.syncDB.GetAllSyncInfo()
}

// GetSyncStats Get current sync statistics for time-based synchronization
func (s *SyncDBService) GetSyncStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Get sync info
	syncInfo, err := s.syncDB.GetSyncInfo()
	if err != nil {
		return nil, fmt.Errorf("failed to get sync info: %v", err)
	}

	stats["syncInfo"] = syncInfo
	stats["isRunning"] = s.IsRunning()
	stats["isSyncCompleted"] = s.IsSyncCompleted()
	stats["syncInterval"] = s.syncInterval
	stats["batchSize"] = s.batchSize
	stats["timeRangeSize"] = s.timeRangeSize

	s.updateQueueIndexingStatus()

	if syncInfo != nil {
		stats["totalPins"] = syncInfo.TotalPins
		stats["totalProcessed"] = syncInfo.ProcessedPins
		stats["totalSuccess"] = syncInfo.SuccessPins
		stats["totalFailed"] = syncInfo.FailedPins
		stats["isSyncing"] = s.isSyncing // Use in-memory status
		stats["isAutoSync"] = syncInfo.IsAutoSync
		stats["lastSyncTime"] = syncInfo.LastSyncTime
		stats["currentTime"] = syncInfo.CurrentTime
		stats["lastProcessedBlockHeight"] = syncInfo.LastProcessedBlockHeight
		stats["currentBlockHeight"] = syncInfo.CurrentBlockHeight
		stats["maxProcessedBlockHeight"] = syncInfo.MaxProcessedBlockHeight
		stats["firstPinTimeStr"] = time.Unix(syncInfo.FirstPinTime, 0).Format("2006-01-02 15:04:05")
		stats["lastProcessedPinTimeStr"] = time.Unix(syncInfo.LastProcessedPinTime, 0).Format("2006-01-02 15:04:05")
		stats["lastSyncStartTime"] = syncInfo.LastSyncStartTime.Format("2006-01-02 15:04:05")
		stats["lastSyncEndTime"] = syncInfo.LastSyncEndTime.Format("2006-01-02 15:04:05")

		// Calculate total sync duration
		var totalSyncDuration int64
		if !syncInfo.LastSyncStartTime.IsZero() && !syncInfo.LastSyncEndTime.IsZero() {
			totalSyncDuration = syncInfo.LastSyncEndTime.Sub(syncInfo.LastSyncStartTime).Milliseconds()
		}
		stats["totalSyncDurationMs"] = totalSyncDuration
		stats["totalSyncDurationStr"] = formatDuration(totalSyncDuration)

		// Calculate first time sync duration
		var firstTimeSyncDuration int64
		if syncInfo.FristTimeSyncStartTime > 0 && syncInfo.FristTimeSyncEndTime > 0 {
			firstTimeSyncDuration = (syncInfo.FristTimeSyncEndTime - syncInfo.FristTimeSyncStartTime) * 1000 // Convert seconds to milliseconds
		}
		stats["firstTimeSyncStartTime"] = syncInfo.FristTimeSyncStartTime
		stats["firstTimeSyncEndTime"] = syncInfo.FristTimeSyncEndTime
		stats["firstTimeSyncStartTimeStr"] = time.Unix(syncInfo.FristTimeSyncStartTime, 0).Format("2006-01-02 15:04:05")
		stats["firstTimeSyncEndTimeStr"] = time.Unix(syncInfo.FristTimeSyncEndTime, 0).Format("2006-01-02 15:04:05")
		stats["firstTimeSyncDurationMs"] = firstTimeSyncDuration
		stats["firstTimeSyncDurationStr"] = formatDuration(firstTimeSyncDuration)

		// Add last completion status and time
		stats["lastIsCompleted"] = syncInfo.LastIsCompleted
		stats["lastIsCompletedTime"] = syncInfo.LastIsCompletedTime
		stats["lastIsCompletedTimeStr"] = time.Unix(syncInfo.LastIsCompletedTime, 0).Format("2006-01-02 15:04:05")

		// Add queue indexing status
		// s.updateQueueIndexingStatus() // Update status before returning
		stats["isGroupChatIndexingCompleted"] = s.isGroupChatIndexingCompleted
		stats["groupChatIndexingRemaining"] = s.groupChatIndexingRemaining
		stats["isPrivateChatIndexingCompleted"] = s.isPrivateChatIndexingCompleted
		stats["privateChatIndexingRemaining"] = s.privateChatIndexingRemaining

		if syncInfo.TotalPins > 0 {
			stats["overallProgress"] = float64(syncInfo.ProcessedPins) / float64(syncInfo.TotalPins) * 100
		} else {
			stats["overallProgress"] = 0.0
		}
	} else {
		stats["totalPins"] = 0
		stats["totalProcessed"] = 0
		stats["totalSuccess"] = 0
		stats["totalFailed"] = 0
		stats["isSyncing"] = false
		stats["isAutoSync"] = false
		stats["lastSyncTime"] = 0
		stats["currentTime"] = time.Now().Unix()
		stats["overallProgress"] = 0.0
		stats["lastProcessedBlockHeight"] = make(map[string]int64)
		stats["currentBlockHeight"] = make(map[string]int64)
		stats["maxProcessedBlockHeight"] = make(map[string]int64)
		stats["lastSyncStartTime"] = ""
		stats["lastSyncEndTime"] = ""
		stats["totalSyncDurationMs"] = int64(0)
		stats["totalSyncDurationStr"] = "0ms"
		stats["firstTimeSyncStartTime"] = int64(0)
		stats["firstTimeSyncEndTime"] = int64(0)
		stats["firstTimeSyncStartTimeStr"] = ""
		stats["firstTimeSyncEndTimeStr"] = ""
		stats["firstTimeSyncDurationMs"] = int64(0)
		stats["firstTimeSyncDurationStr"] = "0ms"
		stats["lastIsCompleted"] = false
		stats["lastIsCompletedTime"] = int64(0)
		stats["lastIsCompletedTimeStr"] = ""

		// Add default queue indexing status
		// s.updateQueueIndexingStatus() // Update status before returning
		stats["isGroupChatIndexingCompleted"] = s.isGroupChatIndexingCompleted
		stats["groupChatIndexingRemaining"] = s.groupChatIndexingRemaining
		stats["isPrivateChatIndexingCompleted"] = s.isPrivateChatIndexingCompleted
		stats["privateChatIndexingRemaining"] = s.privateChatIndexingRemaining
	}

	stats["lastUpdateTime"] = time.Now().Unix()

	return stats, nil
}

// ResetSyncInfo Reset sync info for time-based synchronization
func (s *SyncDBService) ResetSyncInfo(startTime int64) error {
	syncInfo := &SyncInfo{
		LastSyncTime:             startTime,
		LastProcessedPinTime:     startTime,
		CurrentTime:              startTime,
		IsAutoSync:               true,
		SyncInterval:             s.syncInterval,
		BatchSize:                s.batchSize,
		TimeRangeSize:            s.timeRangeSize,
		RetryCount:               0,
		Timestamp:                time.Now().Unix(),
		LastProcessedBlockHeight: make(map[string]int64),
		CurrentBlockHeight:       make(map[string]int64),
		MaxProcessedBlockHeight:  make(map[string]int64),
	}

	// Reset sync completion status and in-memory sync status
	s.SetSyncCompleted(false)
	s.isSyncing = false

	return s.syncDB.SaveSyncInfo(syncInfo)
}

// isBlockFileDbInitialized Check if BlockFileDb is initialized
func (s *SyncDBService) isBlockFileDbInitialized() bool {
	// Check if time_index collection exists in PbMap
	_, ok := blockfile.PbMap["time_index"]
	return ok
}

// getFirstPinTimestamp Get the timestamp of the first pin from blockfile
func (s *SyncDBService) getFirstPinTimestamp() (int64, error) {
	// Check if blockfile is initialized
	if !s.isBlockFileDbInitialized() {
		return 0, fmt.Errorf("blockfile not initialized")
	}

	// Get time_index collection from blockfile
	timeIndexDB, exists := blockfile.PbMap["time_index"]
	if !exists {
		return 0, fmt.Errorf("time_index collection not found in blockfile")
	}

	// Create iterator to find the first (earliest) timestamp
	iter, err := timeIndexDB.NewIter(nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create iterator: %v", err)
	}
	defer iter.Close()

	// Get the first key (earliest timestamp)
	if !iter.First() {
		// No data found
		return 0, nil
	}

	// Parse the timestamp from the key
	// The key format is typically timestamp_pinId or similar
	key := string(iter.Key())

	// Try to extract timestamp from the key
	// Assuming the key format is timestamp_pinId or timestamp+pinId
	var timestamp int64
	if len(key) > 0 {
		// Try to parse the first part as timestamp
		// Look for underscore separator
		parts := strings.Split(key, "_")
		if len(parts) > 0 {
			if ts, err := strconv.ParseInt(parts[0], 10, 64); err == nil {
				timestamp = ts
			}
		}

		// If no underscore found, try to parse the entire key as timestamp
		if timestamp == 0 {
			if ts, err := strconv.ParseInt(key, 10, 64); err == nil {
				timestamp = ts
			}
		}
	}

	if timestamp == 0 {
		return 0, fmt.Errorf("failed to parse timestamp from key: %s", key)
	}

	log.Printf("[SYNC]Found first pin timestamp: %d", timestamp)
	return timestamp, nil
}

// getLatestBlockHeightFromPebble Get the latest block height for a chain from PebbleDB
func (s *SyncDBService) getLatestBlockHeightFromPebble(chainName string) (int64, error) {
	// Use PebbleGetData to get the latest block height from meta collection
	data, err := blockfile.PebbleGetData("meta", chainName+"_lastheight")
	if err != nil {
		if err == pebble.ErrNotFound {
			// If not found, return 0 (no data synced yet)
			return 0, nil
		}
		return 0, fmt.Errorf("failed to get latest block height for chain %s: %v", chainName, err)
	}

	// Parse the block height from the data
	height, err := strconv.ParseInt(string(data), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse block height for chain %s: %v", chainName, err)
	}

	return height, nil
}

// checkAllChainsUpToDate Check if all chains are up to date by comparing PebbleDB height with node height
func (s *SyncDBService) checkAllChainsUpToDate() bool {
	for chainName, adapter := range s.chainAdapter {
		if adapter == nil {
			log.Printf("[SYNC]Chain %s adapter not found, skipping", chainName)
			continue
		}

		// Get current block height from chain adapter
		currentHeight := adapter.GetBestHeight()
		if currentHeight <= 0 {
			log.Printf("[SYNC]Chain %s current height is invalid: %d", chainName, currentHeight)
			return false
		}

		// Get the latest block height from PebbleDB using PebbleGetData
		pebbleHeight, err := s.getPebbleHeight(chainName)
		if err != nil {
			log.Printf("[SYNC]Failed to get PebbleDB height for chain %s: %v", chainName, err)
			return false
		}

		log.Printf("[SYNC]Chain %s - node height: %d, PebbleDB height: %d", chainName, currentHeight, pebbleHeight)

		// Check if PebbleDB height matches node height
		if pebbleHeight != currentHeight {
			log.Printf("[SYNC]Chain %s not up to date: PebbleDB height %d != node height %d", chainName, pebbleHeight, currentHeight)
			return false
		} else {
			log.Printf("[SYNC]Chain %s is up to date: PebbleDB height %d == node height %d", chainName, pebbleHeight, currentHeight)
		}
	}

	log.Printf("[SYNC]All chains are up to date")
	return true
}

// getPebbleHeight Get the latest block height for a chain from PebbleDB using PebbleGetData
func (s *SyncDBService) getPebbleHeight(chainName string) (int64, error) {
	// Use PebbleGetData to get the latest block height from meta collection
	data, err := blockfile.PebbleGetData("meta", chainName+"_lastheight")
	if err != nil {
		if err == pebble.ErrNotFound {
			// If not found, return 0 (no data synced yet)
			return 0, nil
		}
		return 0, fmt.Errorf("failed to get PebbleDB height for chain %s: %v", chainName, err)
	}

	// Parse the block height from the data
	height, err := strconv.ParseInt(string(data), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse PebbleDB height for chain %s: %v", chainName, err)
	}

	return height, nil
}

// formatDuration Format duration in milliseconds to human readable string
func formatDuration(durationMs int64) string {
	if durationMs <= 0 {
		return "0ms"
	}

	seconds := durationMs / 1000
	minutes := seconds / 60
	hours := minutes / 60
	days := hours / 24

	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm %ds", days, hours%24, minutes%60, seconds%60)
	} else if hours > 0 {
		return fmt.Sprintf("%dh %dm %ds", hours, minutes%60, seconds%60)
	} else if minutes > 0 {
		return fmt.Sprintf("%dm %ds", minutes, seconds%60)
	} else {
		return fmt.Sprintf("%ds %dms", seconds, durationMs%1000)
	}
}

// GetPinsCountByTimeRange Get pins count by time range
func (s *SyncDBService) GetPinsCountByTimeRange(startTime, endTime int64) (map[string]interface{}, error) {
	// Validate time range
	if startTime <= 0 || endTime <= 0 {
		return nil, fmt.Errorf("startTime and endTime must be positive")
	}
	if startTime >= endTime {
		return nil, fmt.Errorf("startTime must be less than endTime")
	}

	// Query pins by time range
	pins, err := blockfile.QueryPinsByTimeRange(startTime, endTime, s.hosts, s.protocols)
	if err != nil {
		return nil, fmt.Errorf("failed to query pins by time range: %v", err)
	}

	// Count pins by chain
	chainCounts := make(map[string]int)
	for _, pin := range pins {
		chainCounts[pin.ChainName]++
	}

	// Build response
	result := map[string]interface{}{
		"startTime":   startTime,
		"endTime":     endTime,
		"totalCount":  len(pins),
		"chainCounts": chainCounts,
		"hosts":       s.hosts,
		"protocols":   s.protocols,
	}

	log.Printf("[SYNC]GetPinsCountByTimeRange: %+v", result)

	log.Printf("[SYNC]GetPinsCountByTimeRange: startTime=%d, endTime=%d, totalCount=%d",
		startTime, endTime, len(pins))

	return result, nil
}

// CheckPinExistsByChainAndHeight Check if a pin exists by chain name, block height and pinId
func (s *SyncDBService) CheckPinExistsByChainAndHeight(chainName string, blockHeight int64, pinId string) (bool, *pin.PinInscription, error) {
	// Validate input parameters
	if chainName == "" || pinId == "" || blockHeight <= 0 {
		return false, nil, fmt.Errorf("invalid parameters: chainName, pinId and blockHeight must be provided")
	}

	log.Printf("[SYNC]Checking pin existence: chain=%s, height=%d, pinId=%s",
		chainName, blockHeight, pinId)

	// Use GetPinsByChainAndHeight to get all pins for the specific chain and height
	pins, err := blockfile.GetPinsByChainAndHeight(chainName, blockHeight, nil, nil)
	if err != nil {
		return false, nil, fmt.Errorf("failed to get pins by chain and height: %v", err)
	}

	// Search for the specific pin
	for _, pin := range pins {
		// Check if this pin matches our criteria
		if pin.Id == pinId {
			log.Printf("[SYNC]Found matching pin: chain=%s, height=%d, pinId=%s, timestamp=%d",
				pin.ChainName, pin.GenesisHeight, pin.Id, pin.Timestamp)

			return true, &pin, nil
		}
	}

	log.Printf("[SYNC]Pin not found: chain=%s, height=%d, pinId=%s", chainName, blockHeight, pinId)
	return false, nil, nil
}

// GetPinIdsByChainAndHeight Get pin IDs by chain name and block height
func (s *SyncDBService) GetPinIdsByChainAndHeight(chainName string, blockHeight int64) (map[string]interface{}, error) {
	// Validate input parameters
	if chainName == "" || blockHeight <= 0 {
		return nil, fmt.Errorf("invalid parameters: chainName and blockHeight must be provided")
	}

	log.Printf("[SYNC]Getting pin IDs: chain=%s, height=%d",
		chainName, blockHeight)

	// Use GetPinsByChainAndHeight to get all pins for the specific chain and height
	pins, err := blockfile.GetPinsByChainAndHeight(chainName, blockHeight, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get pins by chain and height: %v", err)
	}

	// Collect pin IDs
	var pinIds []string
	for _, pin := range pins {
		pinIds = append(pinIds, pin.Id)
		log.Printf("[SYNC]Found matching pin: chain=%s, height=%d, pinId=%s, timestamp=%d",
			pin.ChainName, pin.GenesisHeight, pin.Id, pin.Timestamp)
	}

	resp := map[string]interface{}{
		"chainName":   chainName,
		"blockHeight": blockHeight,
		"pinIds":      pinIds,
		"pinsTotal":   len(pins),
	}

	log.Printf("[SYNC]Found %d pins for chain=%s, height=%d", len(pinIds), chainName, blockHeight)
	return resp, nil
}
