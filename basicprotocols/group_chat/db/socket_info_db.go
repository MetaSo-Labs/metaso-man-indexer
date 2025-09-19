package db

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/cockroachdb/pebble"
)

// SocketInfoDB Socket info snapshot database operations
type SocketInfoDB struct {
	pb *Pebble
}

// NewSocketInfoDB Create socket info snapshot database instance
func NewSocketInfoDB(pb *Pebble) *SocketInfoDB {
	return &SocketInfoDB{
		pb: pb,
	}
}

// SocketInfoSnapshot Socket info snapshot item
type SocketInfoSnapshot struct {
	Timestamp             int64   `json:"timestamp"`             // Snapshot timestamp
	TimeStr               string  `json:"timeStr"`               // Snapshot time string
	TotalConnections      int64   `json:"totalConnections"`      // Total connections
	ActiveConnections     int64   `json:"activeConnections"`     // Active connections
	TotalUserConnections  int64   `json:"totalUserConnections"`  // Total user connections
	ActiveUserConnections int64   `json:"activeUserConnections"` // Active user connections
	TotalMessagesSent     int64   `json:"totalMessagesSent"`     // Total messages sent
	TotalMessagesFailed   int64   `json:"totalMessagesFailed"`   // Total messages failed
	TotalMemoryUsage      int64   `json:"totalMemoryUsage"`      // Total memory usage in bytes
	AverageMemoryPerConn  int64   `json:"averageMemoryPerConn"`  // Average memory per connection in bytes
	TotalMemoryMB         float64 `json:"totalMemoryMB"`         // Total memory usage in MB
	AverageMemoryKB       float64 `json:"averageMemoryKB"`       // Average memory per connection in KB
	MemoryUsagePercent    float64 `json:"memoryUsagePercent"`    // Memory usage percentage
	MemoryLimitMB         int     `json:"memoryLimitMB"`         // Memory limit in MB
}

// SaveSocketInfoSnapshot Save socket info snapshot
func (sidb *SocketInfoDB) SaveSocketInfoSnapshot(timestamp int64, stats *SocketInfoSnapshot) error {
	// Validate timestamp
	if timestamp <= 0 {
		return fmt.Errorf("timestamp must be positive")
	}

	// Validate stats
	if stats == nil {
		return fmt.Errorf("stats cannot be nil")
	}

	// Set timestamp
	stats.Timestamp = timestamp
	stats.TimeStr = time.Unix(timestamp/1000, (timestamp%1000)*1000000).Format("2006-01-02 15:04:05")

	// Serialize data
	data, err := json.Marshal(stats)
	if err != nil {
		return fmt.Errorf("failed to marshal socket info snapshot: %v", err)
	}

	// Save to database, using timestamp as key
	key := []byte(fmt.Sprintf("%d", timestamp))
	err = Pb[TalkSocketInfoSnapshotCollection].Set(key, data, pebble.Sync)
	if err != nil {
		return fmt.Errorf("failed to save socket info snapshot: %v", err)
	}

	return nil
}

// GetSocketInfoSnapshot Get socket info snapshot by timestamp
func (sidb *SocketInfoDB) GetSocketInfoSnapshot(timestamp int64) (*SocketInfoSnapshot, error) {
	// Validate timestamp
	if timestamp <= 0 {
		return nil, fmt.Errorf("timestamp must be positive")
	}

	// Get from database
	key := []byte(fmt.Sprintf("%d", timestamp))
	data, closer, err := Pb[TalkSocketInfoSnapshotCollection].Get(key)
	if err != nil {
		if err == pebble.ErrNotFound {
			return nil, nil // Snapshot not found
		}
		return nil, fmt.Errorf("failed to get socket info snapshot: %v", err)
	}
	defer closer.Close()

	// Deserialize data
	var snapshot SocketInfoSnapshot
	err = json.Unmarshal(data, &snapshot)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal socket info snapshot: %v", err)
	}

	return &snapshot, nil
}

// DeleteSocketInfoSnapshot Delete socket info snapshot by timestamp
func (sidb *SocketInfoDB) DeleteSocketInfoSnapshot(timestamp int64) error {
	// Validate timestamp
	if timestamp <= 0 {
		return fmt.Errorf("timestamp must be positive")
	}

	// Delete from database
	key := []byte(fmt.Sprintf("%d", timestamp))
	err := Pb[TalkSocketInfoSnapshotCollection].Delete(key, pebble.Sync)
	if err != nil {
		return fmt.Errorf("failed to delete socket info snapshot: %v", err)
	}

	return nil
}

// GetAllSocketInfoSnapshots Get all socket info snapshots
func (sidb *SocketInfoDB) GetAllSocketInfoSnapshots() ([]*SocketInfoSnapshot, error) {
	var snapshots []*SocketInfoSnapshot

	// Create iterator
	iter, err := Pb[TalkSocketInfoSnapshotCollection].NewIter(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create iterator: %v", err)
	}
	defer iter.Close()

	// Iterate through all records
	for iter.First(); iter.Valid(); iter.Next() {
		// Get data
		data := iter.Value()

		// Deserialize data
		var snapshot SocketInfoSnapshot
		err = json.Unmarshal(data, &snapshot)
		if err != nil {
			// Log error but continue processing other records
			fmt.Printf("Warning: failed to unmarshal socket info snapshot for key %s: %v\n", string(iter.Key()), err)
			continue
		}

		snapshots = append(snapshots, &snapshot)
	}

	// Check iterator error
	if err = iter.Error(); err != nil {
		return nil, fmt.Errorf("iterator error: %v", err)
	}

	return snapshots, nil
}

// GetSocketInfoSnapshotsByTimeRange Get socket info snapshots within time range
func (sidb *SocketInfoDB) GetSocketInfoSnapshotsByTimeRange(startTime, endTime int64) ([]*SocketInfoSnapshot, error) {
	var snapshots []*SocketInfoSnapshot

	// Validate time range
	if startTime <= 0 || endTime <= 0 || startTime > endTime {
		return nil, fmt.Errorf("invalid time range: startTime=%d, endTime=%d", startTime, endTime)
	}

	// Create iterator
	iter, err := Pb[TalkSocketInfoSnapshotCollection].NewIter(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create iterator: %v", err)
	}
	defer iter.Close()

	// Start from the beginning of the time range
	startKey := []byte(fmt.Sprintf("%d", startTime))
	for iter.SeekGE(startKey); iter.Valid(); iter.Next() {
		// Get timestamp from key
		key := iter.Key()
		timestampStr := string(key)

		// Parse timestamp
		var timestamp int64
		if _, err := fmt.Sscanf(timestampStr, "%d", &timestamp); err != nil {
			continue
		}

		// Check if within time range
		if timestamp > endTime {
			break
		}

		// Get data
		data := iter.Value()

		// Deserialize data
		var snapshot SocketInfoSnapshot
		err = json.Unmarshal(data, &snapshot)
		if err != nil {
			// Log error but continue processing other records
			fmt.Printf("Warning: failed to unmarshal socket info snapshot for timestamp %d: %v\n", timestamp, err)
			continue
		}

		snapshots = append(snapshots, &snapshot)
	}

	// Check iterator error
	if err = iter.Error(); err != nil {
		return nil, fmt.Errorf("iterator error: %v", err)
	}

	return snapshots, nil
}

// GetLatestSocketInfoSnapshot Get the latest socket info snapshot
func (sidb *SocketInfoDB) GetLatestSocketInfoSnapshot() (*SocketInfoSnapshot, error) {
	// Create iterator
	iter, err := Pb[TalkSocketInfoSnapshotCollection].NewIter(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create iterator: %v", err)
	}
	defer iter.Close()

	// Get the last record (highest timestamp)
	var latestSnapshot *SocketInfoSnapshot
	for iter.Last(); iter.Valid(); iter.Prev() {
		// Get data
		data := iter.Value()

		// Deserialize data
		var snapshot SocketInfoSnapshot
		err = json.Unmarshal(data, &snapshot)
		if err != nil {
			// Log error but continue processing other records
			fmt.Printf("Warning: failed to unmarshal socket info snapshot for key %s: %v\n", string(iter.Key()), err)
			continue
		}

		latestSnapshot = &snapshot
		break // Get only the latest one
	}

	// Check iterator error
	if err = iter.Error(); err != nil {
		return nil, fmt.Errorf("iterator error: %v", err)
	}

	return latestSnapshot, nil
}

// GetSocketInfoSnapshotCount Get total count of socket info snapshots
func (sidb *SocketInfoDB) GetSocketInfoSnapshotCount() (int64, error) {
	count := int64(0)

	// Create iterator
	iter, err := Pb[TalkSocketInfoSnapshotCollection].NewIter(nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create iterator: %v", err)
	}
	defer iter.Close()

	// Iterate through all records and count
	for iter.First(); iter.Valid(); iter.Next() {
		count++
	}

	// Check iterator error
	if err = iter.Error(); err != nil {
		return 0, fmt.Errorf("iterator error: %v", err)
	}

	return count, nil
}

// ClearOldSocketInfoSnapshots Clear old socket info snapshots before specified timestamp
func (sidb *SocketInfoDB) ClearOldSocketInfoSnapshots(beforeTimestamp int64) (int64, error) {
	// Validate timestamp
	if beforeTimestamp <= 0 {
		return 0, fmt.Errorf("beforeTimestamp must be positive")
	}

	deletedCount := int64(0)

	// Create iterator
	iter, err := Pb[TalkSocketInfoSnapshotCollection].NewIter(nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create iterator: %v", err)
	}
	defer iter.Close()

	// Iterate through all records and delete old ones
	for iter.First(); iter.Valid(); iter.Next() {
		// Get timestamp from key
		key := iter.Key()
		timestampStr := string(key)

		// Parse timestamp
		var timestamp int64
		if _, err := fmt.Sscanf(timestampStr, "%d", &timestamp); err != nil {
			continue
		}

		// Check if timestamp is before the specified time
		if timestamp < beforeTimestamp {
			err = Pb[TalkSocketInfoSnapshotCollection].Delete(key, pebble.Sync)
			if err != nil {
				return deletedCount, fmt.Errorf("failed to delete socket info snapshot %d: %v", timestamp, err)
			}
			deletedCount++
		}
	}

	// Check iterator error
	if err = iter.Error(); err != nil {
		return deletedCount, fmt.Errorf("iterator error: %v", err)
	}

	return deletedCount, nil
}

// ClearAllSocketInfoSnapshots Clear all socket info snapshots (use with caution)
func (sidb *SocketInfoDB) ClearAllSocketInfoSnapshots() error {
	// Create iterator
	iter, err := Pb[TalkSocketInfoSnapshotCollection].NewIter(nil)
	if err != nil {
		return fmt.Errorf("failed to create iterator: %v", err)
	}
	defer iter.Close()

	// Iterate through all records and delete
	for iter.First(); iter.Valid(); iter.Next() {
		key := iter.Key()
		err = Pb[TalkSocketInfoSnapshotCollection].Delete(key, pebble.Sync)
		if err != nil {
			return fmt.Errorf("failed to delete socket info snapshot %s: %v", string(key), err)
		}
	}

	// Check iterator error
	if err = iter.Error(); err != nil {
		return fmt.Errorf("iterator error: %v", err)
	}

	return nil
}

// GetSocketInfoSnapshotStats Get socket info snapshot statistics
func (sidb *SocketInfoDB) GetSocketInfoSnapshotStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Get total count
	totalCount, err := sidb.GetSocketInfoSnapshotCount()
	if err != nil {
		return nil, fmt.Errorf("failed to get total count: %v", err)
	}
	stats["totalCount"] = totalCount

	// Get latest snapshot
	latestSnapshot, err := sidb.GetLatestSocketInfoSnapshot()
	if err != nil {
		return nil, fmt.Errorf("failed to get latest snapshot: %v", err)
	}

	if latestSnapshot != nil {
		stats["latestSnapshot"] = latestSnapshot
		stats["latestTimestamp"] = latestSnapshot.Timestamp
		stats["latestTimestampStr"] = time.Unix(latestSnapshot.Timestamp/1000, (latestSnapshot.Timestamp%1000)*1000000).Format("2006-01-02 15:04:05")
	} else {
		stats["latestSnapshot"] = nil
		stats["latestTimestamp"] = int64(0)
		stats["latestTimestampStr"] = ""
	}

	// Get snapshots from last 24 hours for trend analysis
	now := time.Now().UnixMilli()
	oneDayAgo := now - 24*60*60*1000 // 24 hours ago

	recentSnapshots, err := sidb.GetSocketInfoSnapshotsByTimeRange(oneDayAgo, now)
	if err != nil {
		stats["recentSnapshotsError"] = err.Error()
	} else {
		stats["recentSnapshotsCount"] = len(recentSnapshots)

		// Calculate averages for recent snapshots
		if len(recentSnapshots) > 0 {
			var totalConnections, activeConnections, totalMessagesSent, totalMessagesFailed int64
			var totalMemoryUsage int64

			for _, snapshot := range recentSnapshots {
				totalConnections += snapshot.TotalConnections
				activeConnections += snapshot.ActiveConnections
				totalMessagesSent += snapshot.TotalMessagesSent
				totalMessagesFailed += snapshot.TotalMessagesFailed
				totalMemoryUsage += snapshot.TotalMemoryUsage
			}

			count := int64(len(recentSnapshots))
			stats["averageTotalConnections"] = float64(totalConnections) / float64(count)
			stats["averageActiveConnections"] = float64(activeConnections) / float64(count)
			stats["averageMessagesSent"] = float64(totalMessagesSent) / float64(count)
			stats["averageMessagesFailed"] = float64(totalMessagesFailed) / float64(count)
			stats["averageMemoryUsage"] = float64(totalMemoryUsage) / float64(count)
		}
	}

	stats["lastUpdateTime"] = time.Now().UnixMilli()

	return stats, nil
}
