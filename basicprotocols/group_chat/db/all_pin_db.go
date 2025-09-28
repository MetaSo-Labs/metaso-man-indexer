package db

import (
	"fmt"
	"log"

	"github.com/cockroachdb/pebble"
)

// IsPinSynced Check if a pin has been synced
// pinId: The pin ID to check
// Returns: true if synced, false if not synced, error if database error
func IsPinSynced(pinId string) (bool, error) {
	if pinId == "" {
		return false, fmt.Errorf("pinId cannot be empty")
	}

	// Get value from database
	value, closer, err := Pb[TalkAllPinCollection].Get([]byte(pinId))
	if err != nil {
		if err == pebble.ErrNotFound {
			// Pin not found, means not synced
			return false, nil
		}
		return false, fmt.Errorf("failed to get pin sync status: %v", err)
	}
	defer closer.Close()

	// Parse the value to check if it's marked as synced
	isResync := string(value)
	if isResync == "true" || isResync == "1" {
		return true, nil
	}

	return false, nil
}

// MarkPinAsSynced Mark a pin as synced
// pinId: The pin ID to mark as synced
// synced: Whether this is a synced operation
// Returns: error if database operation fails
func MarkPinAsSynced(pinId string, synced bool) error {
	if pinId == "" {
		return fmt.Errorf("pinId cannot be empty")
	}

	// Convert boolean to string
	var value string
	if synced {
		value = "true"
	} else {
		value = "false"
	}

	// Save to database
	err := Pb[TalkAllPinCollection].Set([]byte(pinId), []byte(value), pebble.Sync)
	if err != nil {
		return fmt.Errorf("failed to mark pin as synced: %v", err)
	}

	log.Printf("[ALL_PIN]Marked pin %s as synced (synced: %v)", pinId, synced)
	return nil
}

// GetPinSyncStatus Get detailed sync status of a pin
// pinId: The pin ID to check
// Returns: sync status information
func GetPinSyncStatus(pinId string) (map[string]interface{}, error) {
	if pinId == "" {
		return nil, fmt.Errorf("pinId cannot be empty")
	}

	// Get value from database
	value, closer, err := Pb[TalkAllPinCollection].Get([]byte(pinId))
	if err != nil {
		if err == pebble.ErrNotFound {
			return map[string]interface{}{
				"pinId":    pinId,
				"isSynced": false,
				"isResync": false,
				"found":    false,
				"message":  "Pin not found in sync database",
			}, nil
		}
		return nil, fmt.Errorf("failed to get pin sync status: %v", err)
	}
	defer closer.Close()

	// Parse the value
	isResyncStr := string(value)
	isResync := isResyncStr == "true" || isResyncStr == "1"

	return map[string]interface{}{
		"pinId":    pinId,
		"isSynced": true,
		"isResync": isResync,
		"found":    true,
		"rawValue": isResyncStr,
	}, nil
}

// GetAllSyncedPins Get all synced pins with pagination
// cursor: Starting position for pagination
// size: Number of items to return
// Returns: paginated list of synced pins
func GetAllSyncedPins(cursor, size int) (map[string]interface{}, error) {
	if cursor < 0 {
		cursor = 0
	}
	if size <= 0 {
		size = 20
	}
	if size > 100 {
		size = 100
	}

	var results []map[string]interface{}
	iter, err := Pb[TalkAllPinCollection].NewIter(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create iterator: %v", err)
	}
	defer iter.Close()

	count := 0
	skipCount := 0
	total := 0

	// First, count total records
	for iter.First(); iter.Valid(); iter.Next() {
		total++
	}

	// Then, get records with pagination
	for iter.First(); iter.Valid(); iter.Next() {
		// Skip until cursor
		if skipCount < cursor {
			skipCount++
			continue
		}

		// Check if we've reached the size limit
		if count >= size {
			break
		}

		pinId := string(iter.Key())
		value := string(iter.Value())
		isResync := value == "true" || value == "1"

		results = append(results, map[string]interface{}{
			"pinId":    pinId,
			"isResync": isResync,
			"value":    value,
		})
		count++
	}

	// Calculate next cursor
	nextCursor := cursor + count
	if count < size {
		nextCursor = -1 // No more data
	}

	return map[string]interface{}{
		"collection": TalkAllPinCollection,
		"cursor":     cursor,
		"size":       size,
		"nextCursor": nextCursor,
		"count":      count,
		"total":      total,
		"data":       results,
	}, nil
}
