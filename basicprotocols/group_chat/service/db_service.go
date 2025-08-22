package service

import (
	"encoding/json"
	"fmt"
	"manindexer/basicprotocols/group_chat/db"
	"strconv"

	"github.com/cockroachdb/pebble"
)

// Generic query method - Query data by prefix
func QueryByPrefix(collectionName, prefix string, limit int) ([]map[string]interface{}, error) {
	return QueryByPrefixWithOrder(collectionName, prefix, limit, false)
}

// Generic query method - Query data by prefix in reverse order
func QueryByPrefixReverse(collectionName, prefix string, limit int) ([]map[string]interface{}, error) {
	return QueryByPrefixWithOrder(collectionName, prefix, limit, true)
}

// Generic query method - Query data by prefix with order control
func QueryByPrefixWithOrder(collectionName, prefix string, limit int, reverse bool) ([]map[string]interface{}, error) {
	dbInstance, exists := db.Pb[collectionName]
	if !exists {
		return nil, fmt.Errorf("database %s does not exist", collectionName)
	}

	var results []map[string]interface{}
	iter, _ := dbInstance.NewIter(&pebble.IterOptions{
		LowerBound: []byte(prefix),
		UpperBound: []byte(prefix + "\xff"),
	})
	defer iter.Close()

	count := 0

	if reverse {
		// Reverse order: start from last and go backwards
		for iter.Last(); iter.Valid() && count < limit; iter.Prev() {
			key := string(iter.Key())
			value := string(iter.Value())

			// Try to parse JSON
			var jsonData interface{}
			if err := json.Unmarshal(iter.Value(), &jsonData); err != nil {
				// If not JSON, use string directly
				jsonData = value
			}

			results = append(results, map[string]interface{}{
				"key":   key,
				"value": jsonData,
			})
			count++
		}
	} else {
		// Forward order: start from first and go forwards
		for iter.First(); iter.Valid() && count < limit; iter.Next() {
			key := string(iter.Key())
			value := string(iter.Value())

			// Try to parse JSON
			var jsonData interface{}
			if err := json.Unmarshal(iter.Value(), &jsonData); err != nil {
				// If not JSON, use string directly
				jsonData = value
			}

			results = append(results, map[string]interface{}{
				"key":   key,
				"value": jsonData,
			})
			count++
		}
	}

	return results, nil
}

// Generic query method - Query single data by key
func QueryByKey(collectionName, key string) (map[string]interface{}, error) {
	dbInstance, exists := db.Pb[collectionName]
	if !exists {
		return nil, fmt.Errorf("database %s does not exist", collectionName)
	}

	value, closer, err := dbInstance.Get([]byte(key))
	if err != nil {
		return nil, fmt.Errorf("query failed: %v", err)
	}
	defer closer.Close()

	// Try to parse JSON
	var jsonData interface{}
	if err := json.Unmarshal(value, &jsonData); err != nil {
		// If not JSON, use string directly
		jsonData = string(value)
	}

	return map[string]interface{}{
		"key":   key,
		"value": jsonData,
	}, nil
}

// Generic query method - Get all data (with limit)
func QueryAll(collectionName string, limit int) ([]map[string]interface{}, error) {
	dbInstance, exists := db.Pb[collectionName]
	if !exists {
		return nil, fmt.Errorf("database %s does not exist", collectionName)
	}

	var results []map[string]interface{}
	iter, _ := dbInstance.NewIter(nil)
	defer iter.Close()

	count := 0
	for iter.First(); iter.Valid() && count < limit; iter.Next() {
		key := string(iter.Key())
		value := string(iter.Value())

		// Try to parse JSON
		var jsonData interface{}
		if err := json.Unmarshal(iter.Value(), &jsonData); err != nil {
			// If not JSON, use string directly
			jsonData = value
		}

		results = append(results, map[string]interface{}{
			"key":   key,
			"value": jsonData,
		})
		count++
	}

	return results, nil
}

// Community-related query methods

// Query community info
func QueryCommunityInfo(communityId string) (map[string]interface{}, error) {
	return QueryByKey(db.TalkCommunityInfoCollection, communityId)
}

// Query all community info
func QueryAllCommunityInfo(limit int) ([]map[string]interface{}, error) {
	return QueryAll(db.TalkCommunityInfoCollection, limit)
}

// Query community version info
func QueryCommunityVersionInfo(communityId string, limit int) ([]map[string]interface{}, error) {
	prefix := communityId + "_"
	return QueryByPrefix(db.TalkCommunityVersionInfoCollection, prefix, limit)
}

// Query community address info
func QueryCommunityAddress(communityId string, limit int) ([]map[string]interface{}, error) {
	prefix := communityId + "_"
	return QueryByPrefix(db.TalkCommunityAddressCollection, prefix, limit)
}

// Query community join records
func QueryCommunityJoin(communityId string, limit int) ([]map[string]interface{}, error) {
	prefix := communityId + "_"
	return QueryByPrefix(db.TalkCommunityJoinCollection, prefix, limit)
}

// Query community members
func QueryCommunityPerson(communityId string, limit int) ([]map[string]interface{}, error) {
	prefix := communityId + "_"
	return QueryByPrefix(db.TalkCommunityPersonCollection, prefix, limit)
}

// Group-related query methods

// Query group info
func QueryGroupInfo(groupId string) (map[string]interface{}, error) {
	return QueryByKey(db.TalkGroupInfoCollection, groupId)
}

// Query all group info
func QueryAllGroupInfo(limit int) ([]map[string]interface{}, error) {
	return QueryAll(db.TalkGroupInfoCollection, limit)
}

// Query group version info
func QueryGroupVersionInfo(groupId string, limit int) ([]map[string]interface{}, error) {
	prefix := groupId + "_"
	return QueryByPrefix(db.TalkGroupVersionInfoCollection, prefix, limit)
}

// Query group community association
func QueryGroupCommunity(communityId string, limit int) ([]map[string]interface{}, error) {
	prefix := communityId + "_"
	return QueryByPrefix(db.TalkGroupCommunityCollection, prefix, limit)
}

// Query group join records
func QueryGroupJoin(groupId string, limit int) ([]map[string]interface{}, error) {
	prefix := groupId + "_"
	return QueryByPrefix(db.TalkGroupJoinCollection, prefix, limit)
}

// Query group members
func QueryGroupPerson(groupId string, limit int) ([]map[string]interface{}, error) {
	prefix := groupId + "_"
	return QueryByPrefix(db.TalkGroupPersonCollection, prefix, limit)
}

// User group list related query methods

// Query user's group list
func QueryMetaIdContextList(metaId string) (map[string]interface{}, error) {
	return QueryByKey(db.TalkMetaIdContextListCollection, metaId)
}

// Query all users' group lists
func QueryAllMetaIdContextList(limit int) ([]map[string]interface{}, error) {
	return QueryAll(db.TalkMetaIdContextListCollection, limit)
}

// Message queue related query methods

// Query chat queue
func QueryGroupChatQueue(limit int) ([]map[string]interface{}, error) {
	return QueryAll(db.TalkGroupChatQueueCollection, limit)
}

// Chat related query methods

// Query group chat message
func QueryGroupChatPin(pinId string) (map[string]interface{}, error) {
	return QueryByKey(db.TalkGroupChatPinCollection, pinId)
}

// Query group chat message (by timestamp range)
func QueryGroupChatByTimestamp(groupId string, startTime, endTime int64, limit int) ([]map[string]interface{}, error) {
	prefix := groupId + "_"
	return QueryByPrefixReverse(db.TalkGroupChatTimestampCollection, prefix, limit)
}

// Query lucky bag message
func QueryRedEnvelopePin(pinId string) (map[string]interface{}, error) {
	return QueryByKey(db.TalkGroupLuckyBagPinCollection, pinId)
}

// Query all lucky bag messages
func QueryAllRedEnvelopePin(limit int) ([]map[string]interface{}, error) {
	return QueryAll(db.TalkGroupLuckyBagPinCollection, limit)
}

// Query grab lucky bag records
func QueryOpenRedEnvelopePin(pinId string) (map[string]interface{}, error) {
	return QueryByKey(db.TalkGroupOpenLuckyBagPinCollection, pinId)
}

// Query all grab lucky bag records
func QueryAllOpenRedEnvelopePin(limit int) ([]map[string]interface{}, error) {
	return QueryAll(db.TalkGroupOpenLuckyBagPinCollection, limit)
}

// Query remaining lucky bag
func QueryResidueRedEnvelopePin(pinId string) (map[string]interface{}, error) {
	return QueryByKey(db.TalkGroupResidueLuckyBagPinCollection, pinId)
}

// Query all remaining lucky bags
func QueryAllResidueRedEnvelopePin(limit int) ([]map[string]interface{}, error) {
	return QueryAll(db.TalkGroupResidueLuckyBagPinCollection, limit)
}

// Statistics related methods

// Get database statistics
func GetDatabaseStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	for collectionName, dbInstance := range db.Pb {
		// Get database size
		iter, _ := dbInstance.NewIter(nil)
		defer iter.Close()

		count := 0
		for iter.First(); iter.Valid(); iter.Next() {
			count++
		}

		stats[collectionName] = map[string]interface{}{
			"record_count": count,
		}
	}

	return stats, nil
}

// Get record count for specified database
func GetCollectionCount(collectionName string) (int, error) {
	dbInstance, exists := db.Pb[collectionName]
	if !exists {
		return 0, fmt.Errorf("database %s does not exist", collectionName)
	}

	iter, _ := dbInstance.NewIter(nil)
	defer iter.Close()

	count := 0
	for iter.First(); iter.Valid(); iter.Next() {
		count++
	}

	return count, nil
}

// Get all available database names
func GetAvailableCollections() []string {
	var collections []string
	for collectionName := range db.Pb {
		collections = append(collections, collectionName)
	}
	return collections
}

// LuckyBagStatistics represents statistics for lucky bag data
type LuckyBagStatistics struct {
	GroupId             string  `json:"groupId"`
	StartTime           int64   `json:"startTime"`
	EndTime             int64   `json:"endTime"`
	LuckyBagCount       int     `json:"luckyBagCount"`       // Number of lucky bags sent
	OpenLuckyBagCount   int     `json:"openLuckyBagCount"`   // Number of opened lucky bags
	TotalAmount         float64 `json:"totalAmount"`         // Total amount of all lucky bags
	TotalOpenedAmount   float64 `json:"totalOpenedAmount"`   // Total amount of opened lucky bags
	UniqueOpeners       int     `json:"uniqueOpeners"`       // Number of unique users who opened lucky bags
	OpenRate            float64 `json:"openRate"`            // Open rate (opened/total)
	AverageAmount       float64 `json:"averageAmount"`       // Average amount per lucky bag
	AverageOpenedAmount float64 `json:"averageOpenedAmount"` // Average amount per opened lucky bag
}

// LuckyBagStatisticsByGroup represents statistics for lucky bag data grouped by group
type LuckyBagStatisticsByGroup struct {
	StartTime          int64                          `json:"startTime"`
	EndTime            int64                          `json:"endTime"`
	TotalLuckyBagCount int                            `json:"totalLuckyBagCount"` // Total number of lucky bags sent across all groups
	TotalOpenCount     int                            `json:"totalOpenCount"`     // Total number of opened lucky bags across all groups
	TotalAmount        float64                        `json:"totalAmount"`        // Total amount of all lucky bags across all groups
	TotalOpenedAmount  float64                        `json:"totalOpenedAmount"`  // Total amount of opened lucky bags across all groups
	TotalUniqueOpeners int                            `json:"totalUniqueOpeners"` // Total number of unique users who opened lucky bags across all groups
	GroupStats         map[string]*LuckyBagStatistics `json:"groupStats"`         // Statistics by group ID
}

// GetLuckyBagStatisticsByGroupAndTimeRange Get lucky bag statistics for a specific group or all groups within a time range
func GetLuckyBagStatisticsByGroupAndTimeRange(groupId string, startTime, endTime int64) (interface{}, error) {
	// If groupId is empty, return statistics for all groups
	if groupId == "" {
		return getLuckyBagStatisticsForAllGroups(startTime, endTime)
	}

	// Return statistics for specific group
	return getLuckyBagStatisticsForSpecificGroup(groupId, startTime, endTime)
}

// getLuckyBagStatisticsForSpecificGroup Get lucky bag statistics for a specific group within a time range
func getLuckyBagStatisticsForSpecificGroup(groupId string, startTime, endTime int64) (*LuckyBagStatistics, error) {
	stats := &LuckyBagStatistics{
		GroupId:   groupId,
		StartTime: startTime,
		EndTime:   endTime,
	}

	// Get lucky bag database instance
	luckyBagDB, exists := db.Pb[db.TalkGroupLuckyBagPinCollection]
	if !exists {
		return nil, fmt.Errorf("database %s does not exist", db.TalkGroupLuckyBagPinCollection)
	}

	// Get open lucky bag database instance
	openLuckyBagDB, exists := db.Pb[db.TalkGroupOpenLuckyBagPinCollection]
	if !exists {
		return nil, fmt.Errorf("database %s does not exist", db.TalkGroupOpenLuckyBagPinCollection)
	}

	// Count lucky bags in time range
	luckyBagIter, err := luckyBagDB.NewIter(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create lucky bag iterator: %v", err)
	}
	defer luckyBagIter.Close()

	// Count open lucky bags in time range
	openLuckyBagIter, err := openLuckyBagDB.NewIter(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create open lucky bag iterator: %v", err)
	}
	defer openLuckyBagIter.Close()

	// Track unique openers
	uniqueOpeners := make(map[string]bool)
	totalAmount := 0.0
	totalOpenedAmount := 0.0

	// Iterate through lucky bags
	for luckyBagIter.First(); luckyBagIter.Valid(); luckyBagIter.Next() {
		var luckyBag map[string]interface{}
		if err := json.Unmarshal(luckyBagIter.Value(), &luckyBag); err != nil {
			continue
		}

		// Check if it belongs to the specified group
		if luckyBag["groupId"] != groupId {
			continue
		}

		// Check timestamp range
		timestamp, ok := luckyBag["timestamp"].(float64)
		if !ok {
			continue
		}

		if int64(timestamp) < startTime || int64(timestamp) > endTime {
			continue
		}

		// Count this lucky bag
		stats.LuckyBagCount++

		// Calculate total amount
		if amount, ok := luckyBag["amount"].(string); ok {
			if amountFloat, err := strconv.ParseFloat(amount, 64); err == nil {
				totalAmount += amountFloat
			}
		}
	}

	// Iterate through open lucky bags
	for openLuckyBagIter.First(); openLuckyBagIter.Valid(); openLuckyBagIter.Next() {
		var openLuckyBag map[string]interface{}
		if err := json.Unmarshal(openLuckyBagIter.Value(), &openLuckyBag); err != nil {
			continue
		}

		// Check if it belongs to the specified group
		if openLuckyBag["groupId"] != groupId {
			continue
		}

		// Check timestamp range
		timestamp, ok := openLuckyBag["timestamp"].(float64)
		if !ok {
			continue
		}

		if int64(timestamp) < startTime || int64(timestamp) > endTime {
			continue
		}

		// Count this open lucky bag
		stats.OpenLuckyBagCount++

		// Track unique openers
		if metaId, ok := openLuckyBag["metaId"].(string); ok {
			uniqueOpeners[metaId] = true
		}

		// Calculate total opened amount
		if amount, ok := openLuckyBag["amount"].(string); ok {
			if amountFloat, err := strconv.ParseFloat(amount, 64); err == nil {
				totalOpenedAmount += amountFloat
			}
		}
	}

	// Set calculated values
	stats.TotalAmount = totalAmount
	stats.TotalOpenedAmount = totalOpenedAmount
	stats.UniqueOpeners = len(uniqueOpeners)

	// Calculate derived statistics
	if stats.LuckyBagCount > 0 {
		stats.OpenRate = float64(stats.OpenLuckyBagCount) / float64(stats.LuckyBagCount)
		stats.AverageAmount = totalAmount / float64(stats.LuckyBagCount)
	}

	if stats.OpenLuckyBagCount > 0 {
		stats.AverageOpenedAmount = totalOpenedAmount / float64(stats.OpenLuckyBagCount)
	}

	return stats, nil
}

// getLuckyBagStatisticsForAllGroups Get lucky bag statistics for all groups within a time range
func getLuckyBagStatisticsForAllGroups(startTime, endTime int64) (*LuckyBagStatisticsByGroup, error) {
	stats := &LuckyBagStatisticsByGroup{
		StartTime:  startTime,
		EndTime:    endTime,
		GroupStats: make(map[string]*LuckyBagStatistics),
	}

	// Get lucky bag database instance
	luckyBagDB, exists := db.Pb[db.TalkGroupLuckyBagPinCollection]
	if !exists {
		return nil, fmt.Errorf("database %s does not exist", db.TalkGroupLuckyBagPinCollection)
	}

	// Get open lucky bag database instance
	openLuckyBagDB, exists := db.Pb[db.TalkGroupOpenLuckyBagPinCollection]
	if !exists {
		return nil, fmt.Errorf("database %s does not exist", db.TalkGroupOpenLuckyBagPinCollection)
	}

	// Count lucky bags in time range
	luckyBagIter, err := luckyBagDB.NewIter(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create lucky bag iterator: %v", err)
	}
	defer luckyBagIter.Close()

	// Count open lucky bags in time range
	openLuckyBagIter, err := openLuckyBagDB.NewIter(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create open lucky bag iterator: %v", err)
	}
	defer openLuckyBagIter.Close()

	// Track unique openers across all groups
	allUniqueOpeners := make(map[string]bool)
	totalAmount := 0.0
	totalOpenedAmount := 0.0

	// Group statistics tracking
	groupStats := make(map[string]*LuckyBagStatistics)
	groupUniqueOpeners := make(map[string]map[string]bool)

	// Iterate through lucky bags
	for luckyBagIter.First(); luckyBagIter.Valid(); luckyBagIter.Next() {
		var luckyBag map[string]interface{}
		if err := json.Unmarshal(luckyBagIter.Value(), &luckyBag); err != nil {
			continue
		}

		// Check timestamp range
		timestamp, ok := luckyBag["timestamp"].(float64)
		if !ok {
			continue
		}

		if int64(timestamp) < startTime || int64(timestamp) > endTime {
			continue
		}

		// Get group ID
		groupId, ok := luckyBag["groupId"].(string)
		if !ok || groupId == "" {
			continue
		}

		// Initialize group stats if not exists
		if groupStats[groupId] == nil {
			groupStats[groupId] = &LuckyBagStatistics{
				GroupId:   groupId,
				StartTime: startTime,
				EndTime:   endTime,
			}
			groupUniqueOpeners[groupId] = make(map[string]bool)
		}

		// Count this lucky bag
		groupStats[groupId].LuckyBagCount++
		stats.TotalLuckyBagCount++

		// Calculate total amount
		if amount, ok := luckyBag["amount"].(string); ok {
			if amountFloat, err := strconv.ParseFloat(amount, 64); err == nil {
				groupStats[groupId].TotalAmount += amountFloat
				totalAmount += amountFloat
			}
		}
	}

	// Iterate through open lucky bags
	for openLuckyBagIter.First(); openLuckyBagIter.Valid(); openLuckyBagIter.Next() {
		var openLuckyBag map[string]interface{}
		if err := json.Unmarshal(openLuckyBagIter.Value(), &openLuckyBag); err != nil {
			continue
		}

		// Check timestamp range
		timestamp, ok := openLuckyBag["timestamp"].(float64)
		if !ok {
			continue
		}

		if int64(timestamp) < startTime || int64(timestamp) > endTime {
			continue
		}

		// Get group ID
		groupId, ok := openLuckyBag["groupId"].(string)
		if !ok || groupId == "" {
			continue
		}

		// Initialize group stats if not exists
		if groupStats[groupId] == nil {
			groupStats[groupId] = &LuckyBagStatistics{
				GroupId:   groupId,
				StartTime: startTime,
				EndTime:   endTime,
			}
			groupUniqueOpeners[groupId] = make(map[string]bool)
		}

		// Count this open lucky bag
		groupStats[groupId].OpenLuckyBagCount++
		stats.TotalOpenCount++

		// Track unique openers
		if metaId, ok := openLuckyBag["metaId"].(string); ok {
			groupUniqueOpeners[groupId][metaId] = true
			allUniqueOpeners[metaId] = true
		}

		// Calculate total opened amount
		if amount, ok := openLuckyBag["amount"].(string); ok {
			if amountFloat, err := strconv.ParseFloat(amount, 64); err == nil {
				groupStats[groupId].TotalOpenedAmount += amountFloat
				totalOpenedAmount += amountFloat
			}
		}
	}

	// Set calculated values for each group
	for groupId, groupStat := range groupStats {
		groupStat.UniqueOpeners = len(groupUniqueOpeners[groupId])

		// Calculate derived statistics for each group
		if groupStat.LuckyBagCount > 0 {
			groupStat.OpenRate = float64(groupStat.OpenLuckyBagCount) / float64(groupStat.LuckyBagCount)
			groupStat.AverageAmount = groupStat.TotalAmount / float64(groupStat.LuckyBagCount)
		}

		if groupStat.OpenLuckyBagCount > 0 {
			groupStat.AverageOpenedAmount = groupStat.TotalOpenedAmount / float64(groupStat.OpenLuckyBagCount)
		}
	}

	// Set overall calculated values
	stats.TotalAmount = totalAmount
	stats.TotalOpenedAmount = totalOpenedAmount
	stats.TotalUniqueOpeners = len(allUniqueOpeners)
	stats.GroupStats = groupStats

	return stats, nil
}
