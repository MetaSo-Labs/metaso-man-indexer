package service

import (
	"fmt"
	"log"
	"manindexer/adapter"
	"manindexer/adapter/bitcoin"
	"manindexer/adapter/microvisionchain"
	"manindexer/blockfile"
	"manindexer/pin"
	"sync"
	"time"
)

var (
	syncServiceInstance *SyncService
	syncServiceOnce     sync.Once
)

// ProcessGroupChatPinFunc Function type for processing group chat pins
type ProcessGroupChatPinFunc func(pin *pin.PinInscription, tx interface{}, isResync bool) error

type SyncService struct {
	// Sync status
	isRunning    bool
	currentStats *SyncStats
	mu           sync.RWMutex
	cancelChan   chan struct{}
	// Callback function for processing pins
	processPinFunc ProcessGroupChatPinFunc
}

type SyncStats struct {
	TotalPins     int       `json:"total_pins"`
	ProcessedPins int       `json:"processed_pins"`
	SuccessPins   int       `json:"success_pins"`
	FailedPins    int       `json:"failed_pins"`
	StartTime     time.Time `json:"start_time"`
	EndTime       time.Time `json:"end_time"`
	IsCompleted   bool      `json:"is_completed"`
	ErrorMessage  string    `json:"error_message,omitempty"`
}

// NewSyncService Create new sync service instance
func NewSyncService() *SyncService {
	return &SyncService{
		isRunning: false,
		currentStats: &SyncStats{
			TotalPins:     0,
			ProcessedPins: 0,
			SuccessPins:   0,
			FailedPins:    0,
			IsCompleted:   false,
		},
		cancelChan: make(chan struct{}),
	}
}

// SetProcessPinFunc Set callback function for processing pins
func (s *SyncService) SetProcessPinFunc(fn ProcessGroupChatPinFunc) {
	s.processPinFunc = fn
}

// GetSyncService Get sync service singleton instance
func GetSyncService() *SyncService {
	syncServiceOnce.Do(func() {
		syncServiceInstance = NewSyncService()
	})
	return syncServiceInstance
}

// SetSyncService Set sync service instance (for dependency injection)
func SetSyncService(service *SyncService) {
	syncServiceInstance = service
}

// isBlockFileDbInitialized Check if BlockFileDb is initialized
func isBlockFileDbInitialized() bool {
	// Check if time_index collection exists in PbMap
	_, ok := blockfile.PbMap["time_index"]
	return ok
}

// SyncPinsByTimeRangeWithBatch Sync pin data by timestamp range with batch processing
// startTs, endTs: Start and end timestamps
// host, protocol: Filter conditions, can be nil for no filtering
// batchDurationSeconds: Duration of each batch in seconds, recommend 3600 seconds (1 hour)
func (s *SyncService) SyncPinsByTimeRangeWithBatch(startTs, endTs int64, host, protocol []string, batchDurationSeconds int64) error {
	// Check if BlockFileDb is initialized
	if !isBlockFileDbInitialized() {
		return fmt.Errorf("BlockFileDb not initialized, please call blockfile.InitBlockFileDb() first")
	}

	s.mu.Lock()
	if s.isRunning {
		s.mu.Unlock()
		return fmt.Errorf("sync service is already running, please wait for completion")
	}

	s.isRunning = true
	s.currentStats = &SyncStats{
		TotalPins:     0,
		ProcessedPins: 0,
		SuccessPins:   0,
		FailedPins:    0,
		StartTime:     time.Now(),
		IsCompleted:   false,
	}
	// Recreate cancel channel
	s.cancelChan = make(chan struct{})
	s.mu.Unlock()

	log.Printf("Starting sync for time range [%d, %d] pin data, batch size: %d seconds", startTs, endTs, batchDurationSeconds)

	// Async batch processing
	go s.processPinsInBatches(startTs, endTs, host, protocol, batchDurationSeconds)

	return nil
}

// processPinsInBatches Process pin data in batches
func (s *SyncService) processPinsInBatches(startTs, endTs int64, host, protocol []string, batchDurationSeconds int64) {
	defer func() {
		s.updateStats(func(stats *SyncStats) {
			stats.IsCompleted = true
			stats.EndTime = time.Now()
		})
		s.mu.Lock()
		s.isRunning = false
		s.mu.Unlock()
		log.Printf("Batch sync completed: total %d, success %d, failed %d",
			s.currentStats.TotalPins, s.currentStats.SuccessPins, s.currentStats.FailedPins)
	}()

	// Calculate total batches
	totalDuration := endTs - startTs
	totalBatches := (totalDuration + batchDurationSeconds - 1) / batchDurationSeconds // Round up

	log.Printf("Time range %d seconds, will be split into %d batches", totalDuration, totalBatches)

	// Process in batches
	for batchNum := int64(0); batchNum < totalBatches; batchNum++ {
		// Check if cancelled
		select {
		case <-s.cancelChan:
			log.Printf("Sync cancelled, stopping at batch %d/%d", batchNum+1, totalBatches)
			return
		default:
		}

		// Calculate current batch time range
		batchStartTs := startTs + batchNum*batchDurationSeconds
		batchEndTs := batchStartTs + batchDurationSeconds
		if batchEndTs > endTs {
			batchEndTs = endTs
		}

		log.Printf("Processing batch %d/%d, time range [%d, %d]", batchNum+1, totalBatches, batchStartTs, batchEndTs)

		// Get current batch pins
		pins, err := blockfile.QueryPinsByTimeRange(batchStartTs, batchEndTs, host, protocol)
		if err != nil {
			log.Printf("Batch %d query failed: %v", batchNum+1, err)
			s.updateStats(func(stats *SyncStats) {
				stats.ErrorMessage = fmt.Sprintf("Batch %d query failed: %v", batchNum+1, err)
			})
			continue
		}

		log.Printf("Batch %d retrieved %d pins", batchNum+1, len(pins))

		// Update total count
		s.updateStats(func(stats *SyncStats) {
			stats.TotalPins += len(pins)
		})

		// Process current batch pins
		s.processBatchPins(pins, batchNum+1, totalBatches)

		// Brief rest between batches to avoid too frequent database operations
		time.Sleep(100 * time.Millisecond)
	}
}

// processBatchPins Process a batch of pins
func (s *SyncService) processBatchPins(pins []pin.PinInscription, batchNum, totalBatches int64) {
	if len(pins) == 0 {
		return
	}

	// Process pins concurrently with controlled concurrency
	maxConcurrency := 10
	semaphore := make(chan struct{}, maxConcurrency)
	var wg sync.WaitGroup

	for _, pinNode := range pins {
		// Check if cancelled
		select {
		case <-s.cancelChan:
			log.Printf("Sync cancelled, stopping batch %d processing", batchNum)
			wg.Wait() // Wait for currently processing pins to complete
			return
		default:
		}

		wg.Add(1)
		go func(pin pin.PinInscription) {
			defer wg.Done()

			// Check if cancelled again
			select {
			case <-s.cancelChan:
				log.Printf("Sync cancelled, skipping pin [%s] %s", pin.ChainName, pin.Path)
				return
			default:
			}

			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			var err error
			// Call pin processing callback function
			if s.processPinFunc != nil {
				err = s.processPinFunc(&pin, nil, true)
			} else {
				err = fmt.Errorf("pin processing callback function not set")
			}

			// Update statistics
			s.updateStats(func(stats *SyncStats) {
				stats.ProcessedPins++
				if err != nil {
					stats.FailedPins++
					log.Printf("Failed to process pin [%s] %s: %v", pin.ChainName, pin.Path, err)
				} else {
					stats.SuccessPins++
				}
			})

			// Print progress every 50 processed pins
			if s.currentStats.ProcessedPins%50 == 0 {
				log.Printf("Batch %d/%d processing progress: %d/%d (%.1f%%)",
					batchNum, totalBatches,
					s.currentStats.ProcessedPins, s.currentStats.TotalPins,
					float64(s.currentStats.ProcessedPins)/float64(s.currentStats.TotalPins)*100)
			}
		}(pinNode)
	}

	// Wait for all goroutines in current batch to complete
	wg.Wait()
	log.Printf("Batch %d/%d completed, current total progress: %d/%d",
		batchNum, totalBatches, s.currentStats.ProcessedPins, s.currentStats.TotalPins)
}

// updateStats Helper method for updating statistics
func (s *SyncService) updateStats(updateFunc func(*SyncStats)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	updateFunc(s.currentStats)
}

// GetSyncStats Get current sync status and progress
func (s *SyncService) GetSyncStats() *SyncStats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Return a copy of statistics
	statsCopy := *s.currentStats
	return &statsCopy
}

// IsRunning Check if sync service is running
func (s *SyncService) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.isRunning
}

// StopSync Stop sync service (if running)
func (s *SyncService) StopSync() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.isRunning {
		return fmt.Errorf("sync service is not running")
	}

	// Send cancel signal
	close(s.cancelChan)
	log.Println("Received stop sync request, cancelling sync...")
	return nil
}

// BlockHeightResult Block height query result
type BlockHeightResult struct {
	ChainName string `json:"chainName"` // Chain name (mvc or btc)
	Height    int64  `json:"height"`    // Block height
	Timestamp int64  `json:"timestamp"` // Block timestamp
}

// GetBlockHeightByTimestamp Find block height by timestamp
// timestamp: Target timestamp
// Returns block heights closest to the timestamp on MVC and BTC chains
func GetBlockHeightByTimestamp(timestamp int64) ([]BlockHeightResult, error) {
	var results []BlockHeightResult

	// Initialize MVC chain
	mvcChain := &microvisionchain.MicroVisionChain{}
	mvcChain.InitChain()

	// Initialize BTC chain
	btcChain := &bitcoin.BitcoinChain{}
	btcChain.InitChain()

	// Find MVC chain block height
	mvcHeight, err := findBlockHeightByTimestamp(mvcChain, timestamp)
	if err != nil {
		log.Printf("Failed to find MVC block height: %v", err)
	} else {
		results = append(results, BlockHeightResult{
			ChainName: "mvc",
			Height:    mvcHeight.Height,
			Timestamp: mvcHeight.Timestamp,
		})
	}

	// Find BTC chain block height
	btcHeight, err := findBlockHeightByTimestamp(btcChain, timestamp)
	if err != nil {
		log.Printf("Failed to find BTC block height: %v", err)
	} else {
		results = append(results, BlockHeightResult{
			ChainName: "btc",
			Height:    btcHeight.Height,
			Timestamp: btcHeight.Timestamp,
		})
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("unable to find blocks corresponding to timestamp %d on any chain", timestamp)
	}

	return results, nil
}

// findBlockHeightByTimestamp Use binary search to find block height closest to target timestamp
func findBlockHeightByTimestamp(chain adapter.Chain, targetTimestamp int64) (*BlockHeightResult, error) {
	// Get current best block height
	bestHeight := chain.GetBestHeight()
	if bestHeight <= 0 {
		return nil, fmt.Errorf("unable to get best block height")
	}

	// Get initial block height
	initialHeight := chain.GetInitialHeight()
	if initialHeight < 0 {
		initialHeight = 0
	}

	left := initialHeight
	right := bestHeight
	var closestHeight int64
	var closestTimestamp int64
	var minDiff int64 = -1

	// Binary search
	for left <= right {
		mid := (left + right) / 2

		// Get middle block timestamp
		timestamp, err := chain.GetBlockTime(mid)
		if err != nil {
			log.Printf("Failed to get block %d timestamp: %v", mid, err)
			// If failed, try adjacent block
			if mid > left {
				mid = mid - 1
				timestamp, err = chain.GetBlockTime(mid)
			}
			if err != nil {
				// If still failed, adjust search range
				right = mid - 1
				continue
			}
		}

		// Calculate time difference
		diff := timestamp - targetTimestamp
		if diff < 0 {
			diff = -diff
		}

		// Update closest block
		if minDiff == -1 || diff < minDiff {
			minDiff = diff
			closestHeight = mid
			closestTimestamp = timestamp
		}

		// Adjust search range
		if timestamp < targetTimestamp {
			left = mid + 1
		} else if timestamp > targetTimestamp {
			right = mid - 1
		} else {
			// Found exact match
			closestHeight = mid
			closestTimestamp = timestamp
			break
		}
	}

	if minDiff == -1 {
		return nil, fmt.Errorf("unable to find suitable block")
	}

	return &BlockHeightResult{
		Height:    closestHeight,
		Timestamp: closestTimestamp,
	}, nil
}
