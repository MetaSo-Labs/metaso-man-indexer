package service

import (
	"encoding/json"
	"fmt"
	"manindexer/basicprotocols/group_chat/db"
	"manindexer/basicprotocols/group_chat/models"
	"manindexer/basicprotocols/group_chat/service/common_service"
	"strconv"
	"strings"
	"time"

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

// Query open lucky bag list by lucky bag PinId
func QueryOpenLuckyBagList(luckyBagPinId string) (map[string]interface{}, error) {
	return QueryByKey(db.TalkGroupOpenLuckyBagListCollection, luckyBagPinId)
}

// GetDetailedOpenLuckyBagList gets detailed open lucky bag list with grab state, user info, and lucky bag details
func GetDetailedOpenLuckyBagList(luckyBagPinId string) (map[string]interface{}, error) {
	// Get lucky bag details first
	luckyBag, err := chatDB.GetLuckyBagByPinId(luckyBagPinId)
	if err != nil {
		return nil, fmt.Errorf("failed to get lucky bag: %v", err)
	}
	if luckyBag == nil {
		return nil, fmt.Errorf("lucky bag not found: %s", luckyBagPinId)
	}

	// Get open lucky bag list
	openList, err := chatDB.GetOpenLuckyBagList(luckyBagPinId)
	if err != nil {
		return nil, fmt.Errorf("failed to get open lucky bag list: %v", err)
	}

	// Get residue lucky bag list
	residueList, err := chatDB.GetResidueLuckyBagList(luckyBagPinId)
	if err != nil {
		return nil, fmt.Errorf("failed to get residue lucky bag list: %v", err)
	}

	// Build detailed response
	response := map[string]interface{}{
		"luckyBagPinId": luckyBagPinId,
		"luckyBagInfo": map[string]interface{}{
			"txId":             luckyBag.TxId,
			"metaId":           luckyBag.MetaId,
			"address":          luckyBag.Address,
			"amount":           luckyBag.Amount,
			"count":            luckyBag.Count,
			"subId":            luckyBag.SubId,
			"code":             luckyBag.Code,
			"createTimeStr":    luckyBag.CreateTimeStr,
			"domain":           luckyBag.Domain,
			"luckyBagAddress":  luckyBag.LuckyBagAddress,
			"genType":          luckyBag.GenType,
			"genState":         luckyBag.GenState,
			"validCount":       luckyBag.ValidCount,
			"payListCount":     len(luckyBag.PayList),
			"luckyBagVouts":    luckyBag.LuckyBagVouts,
			"payList":          luckyBag.PayList,
			"errPayList":       luckyBag.ErrPayList,
			"errLuckyBagVouts": luckyBag.ErrLuckyBagVouts,
			"content":          luckyBag.Content,
			"img":              luckyBag.Img,
			"imgType":          luckyBag.ImgType,
			"type":             luckyBag.Type,
			"timestamp":        luckyBag.Timestamp,
			"chain":            luckyBag.Chain,
			"state":            luckyBag.State,
		},
		"totalCount":   len(openList.Items) + len(residueList.Items),
		"openCount":    len(openList.Items),
		"residueCount": len(residueList.Items),
		"items":        []map[string]interface{}{},
	}

	// Process each open lucky bag item
	for _, item := range openList.Items {
		// Get detailed open lucky bag info
		openLuckyBag, err := chatDB.GetOpenLuckyBagByPinId(item.OpenPinId)
		if err != nil {
			// Skip this item if we can't get details
			continue
		}

		// Get user info
		userInfo := common_service.FetchMetaIDUserInfo(item.CreateAddress)

		// Build detailed item
		detailedItem := map[string]interface{}{
			"openPinId":        item.OpenPinId,
			"groupId":          item.GroupId,
			"timestamp":        item.Timestamp,
			"createMetaId":     item.CreateMetaId,
			"createAddress":    item.CreateAddress,
			"luckyBagOutIndex": item.LuckyBagOutIndex,
			"userInfo":         userInfo,
		}

		// Add grab state details if available
		if openLuckyBag != nil {
			detailedItem["grabState"] = openLuckyBag.GrabState
			detailedItem["grabTxId"] = openLuckyBag.GrabTxId
			detailedItem["grabMsg"] = openLuckyBag.GrabMsg
			detailedItem["amount"] = openLuckyBag.Amount
			detailedItem["index"] = openLuckyBag.Index
			detailedItem["isWithdraw"] = openLuckyBag.IsWithdraw
			detailedItem["grabTimestamp"] = openLuckyBag.Timestamp
		}

		response["items"] = append(response["items"].([]map[string]interface{}), detailedItem)
	}

	// Process each residue lucky bag item
	for _, item := range residueList.Items {
		// Get detailed residue lucky bag info
		residueLuckyBag, err := chatDB.GetResidueLuckyBagByLuckyBagPinId(item.ResiduePinId)
		if err != nil {
			// Skip this item if we can't get details
			continue
		}

		// Get user info
		userInfo := common_service.FetchMetaIDUserInfo(item.CreateAddress)

		// Build detailed item
		detailedItem := map[string]interface{}{
			"residuePinId":         item.ResiduePinId,
			"groupId":              item.GroupId,
			"timestamp":            item.Timestamp,
			"createMetaId":         item.CreateMetaId,
			"createAddress":        item.CreateAddress,
			"luckyBagOutIndexList": item.LuckyBagOutIndexList,
			"userInfo":             userInfo,
			"type":                 "residue", // Mark as residue type
		}

		// Add reclaim state details if available
		if residueLuckyBag != nil {
			detailedItem["reclaimState"] = residueLuckyBag.ReclaimState
			detailedItem["reclaimTxId"] = residueLuckyBag.ReclaimTxId
			detailedItem["reclaimMsg"] = residueLuckyBag.ReclaimMsg
			detailedItem["amount"] = residueLuckyBag.Amount
			detailedItem["usedList"] = residueLuckyBag.UsedList
			detailedItem["reclaimTimestamp"] = residueLuckyBag.Timestamp
		}

		response["items"] = append(response["items"].([]map[string]interface{}), detailedItem)
	}

	return response, nil
}

// QueryMetaIdJoinList gets MetaId join list by metaId
func QueryMetaIdJoinList(metaId string) (map[string]interface{}, error) {
	// Query database directly using prefix
	prefix := metaId + "_"
	results, err := QueryByPrefix(db.TalkGroupMetaIdJoinCollection, prefix, 1000) // Use large limit to get all records
	if err != nil {
		return nil, fmt.Errorf("failed to query MetaId join list: %v", err)
	}

	// Build response
	response := map[string]interface{}{
		"metaId": metaId,
		"items":  []map[string]interface{}{},
	}

	// Process each result
	for _, result := range results {
		// Parse the join list data
		if value, ok := result["value"]; ok {
			if joinListData, ok := value.(map[string]interface{}); ok {
				if items, ok := joinListData["items"].([]interface{}); ok {
					for _, item := range items {
						if joinItem, ok := item.(map[string]interface{}); ok {
							// Get user info for the join record
							address := ""
							if addr, ok := joinItem["address"].(string); ok {
								address = addr
							}
							userInfo := common_service.FetchMetaIDUserInfo(address)

							// Build detailed item
							detailedItem := map[string]interface{}{
								"joinPinId":     joinItem["joinPinId"],
								"joinType":      joinItem["joinType"],
								"joinTimestamp": joinItem["joinTimestamp"],
								"groupState":    joinItem["groupState"],
								"address":       address,
								"referrer":      joinItem["referrer"],
								"blockHeight":   joinItem["blockHeight"],
								"chain":         joinItem["chain"],
								"userInfo":      userInfo,
							}

							response["items"] = append(response["items"].([]map[string]interface{}), detailedItem)
						}
					}
				}
			}
		}
	}

	// Add total count
	response["totalCount"] = len(response["items"].([]map[string]interface{}))

	return response, nil
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

// GetLuckyBagLockStats Get comprehensive statistics about lucky bag locks
func GetLuckyBagLockStats() (map[string]interface{}, error) {
	// Get lock statistics from ChatDB
	stats := chatDB.GetLuckyBagLockStats()
	return stats, nil
}

// GetGroupChatIndexList Get TalkGroupChatIndexCollection list with cursor pagination and reverse order
func GetGroupChatIndexList(cursor, size int, groupId string) (map[string]interface{}, error) {
	if cursor < 0 {
		cursor = 0
	}
	if size <= 0 {
		size = 20
	}

	dbInstance, exists := db.Pb[db.TalkGroupChatIndexCollection]
	if !exists {
		return nil, fmt.Errorf("database %s does not exist", db.TalkGroupChatIndexCollection)
	}

	var results []map[string]interface{}
	var iter *pebble.Iterator
	var err error

	// If groupId is provided, use prefix filtering
	if groupId != "" {
		prefix := groupId + "_"
		iter, err = dbInstance.NewIter(&pebble.IterOptions{
			LowerBound: []byte(prefix),
			UpperBound: []byte(prefix + string([]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff})),
		})
	} else {
		iter, err = dbInstance.NewIter(nil)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create iterator: %v", err)
	}
	defer iter.Close()

	// Calculate skip count
	skip := cursor
	count := 0
	total := 0

	// First, count total records
	if groupId != "" {
		// Count records with prefix
		for iter.First(); iter.Valid(); iter.Next() {
			total++
		}
	} else {
		// Count all records
		for iter.First(); iter.Valid(); iter.Next() {
			total++
		}
	}

	// Then, get records in reverse order with cursor pagination
	for iter.Last(); iter.Valid(); iter.Prev() {
		if count < skip {
			count++
			continue
		}

		if len(results) >= size {
			break
		}

		key := string(iter.Key())
		value := string(iter.Value())

		// Parse key: groupId_index (with zero-padding)
		keyParts := strings.Split(key, "_")
		parsedGroupId := ""
		index := ""
		if len(keyParts) >= 2 {
			parsedGroupId = keyParts[0]
			// Remove leading zeros and get the actual index
			indexStr := strings.TrimLeft(keyParts[1], "0")
			if indexStr == "" {
				indexStr = "0" // If all zeros, treat as 0
			}
			index = indexStr
		}

		// Parse value: pinId_chatType_timestamp_isSet
		valueParts := strings.Split(value, "_")
		pinId := ""
		chatType := ""
		timestamp := ""
		isSet := ""
		if len(valueParts) >= 4 {
			pinId = valueParts[0]
			chatType = valueParts[1]
			timestamp = valueParts[2]
			isSet = valueParts[3]
		}

		results = append(results, map[string]interface{}{
			"key":       key,
			"value":     value,
			"groupId":   parsedGroupId,
			"index":     index,
			"pinId":     pinId,
			"chatType":  chatType,
			"timestamp": timestamp,
			"isSet":     isSet,
		})
	}

	return map[string]interface{}{
		"total":   total,
		"cursor":  cursor,
		"size":    size,
		"results": results,
	}, nil
}

// GetGroupChannelChatIndexList Get TalkGroupChannelChatIndexCollection list with cursor pagination and reverse order
func GetGroupChannelChatIndexList(cursor, size int, channelId string) (map[string]interface{}, error) {
	if cursor < 0 {
		cursor = 0
	}
	if size <= 0 {
		size = 20
	}

	dbInstance, exists := db.Pb[db.TalkGroupChannelChatIndexCollection]
	if !exists {
		return nil, fmt.Errorf("database %s does not exist", db.TalkGroupChannelChatIndexCollection)
	}

	var results []map[string]interface{}
	var iter *pebble.Iterator
	var err error

	// If groupId is provided, use prefix filtering
	if channelId != "" {
		prefix := channelId + "_"
		iter, err = dbInstance.NewIter(&pebble.IterOptions{
			LowerBound: []byte(prefix),
			UpperBound: []byte(prefix + string([]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff})),
		})
	} else {
		iter, err = dbInstance.NewIter(nil)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create iterator: %v", err)
	}
	defer iter.Close()

	// Calculate skip count
	skip := cursor
	count := 0
	total := 0

	// First, count total records
	if channelId != "" {
		// Count records with prefix
		for iter.First(); iter.Valid(); iter.Next() {
			total++
		}
	} else {
		// Count all records
		for iter.First(); iter.Valid(); iter.Next() {
			total++
		}
	}

	// Then, get records in reverse order with cursor pagination
	for iter.Last(); iter.Valid(); iter.Prev() {
		if count < skip {
			count++
			continue
		}

		if len(results) >= size {
			break
		}

		key := string(iter.Key())
		value := string(iter.Value())

		// Parse key: groupId_index (with zero-padding)
		keyParts := strings.Split(key, "_")
		parsedChannelId := ""
		index := ""
		if len(keyParts) >= 2 {
			parsedChannelId = keyParts[0]
			// Remove leading zeros and get the actual index
			indexStr := strings.TrimLeft(keyParts[1], "0")
			if indexStr == "" {
				indexStr = "0" // If all zeros, treat as 0
			}
			index = indexStr
		}

		// Parse value: pinId_chatType_timestamp_isSet
		valueParts := strings.Split(value, "_")
		pinId := ""
		chatType := ""
		timestamp := ""
		isSet := ""
		if len(valueParts) >= 4 {
			pinId = valueParts[0]
			chatType = valueParts[1]
			timestamp = valueParts[2]
			isSet = valueParts[3]
		}

		results = append(results, map[string]interface{}{
			"key":       key,
			"value":     value,
			"channelId": parsedChannelId,
			"index":     index,
			"pinId":     pinId,
			"chatType":  chatType,
			"timestamp": timestamp,
			"isSet":     isSet,
		})
	}

	return map[string]interface{}{
		"total":   total,
		"cursor":  cursor,
		"size":    size,
		"results": results,
	}, nil
}

// GetPrivateChatIndexList Get TalkPrivateChatIndexCollection list with cursor pagination and reverse order
func GetPrivateChatIndexList(cursor, size int, fromTo string) (map[string]interface{}, error) {
	if cursor < 0 {
		cursor = 0
	}
	if size <= 0 {
		size = 20
	}

	dbInstance, exists := db.Pb[db.TalkPrivateChatIndexCollection]
	if !exists {
		return nil, fmt.Errorf("database %s does not exist", db.TalkPrivateChatIndexCollection)
	}

	var results []map[string]interface{}
	var iter *pebble.Iterator
	var err error

	// If fromTo is provided, use prefix filtering
	if fromTo != "" {
		prefix := fromTo + "_"
		iter, err = dbInstance.NewIter(&pebble.IterOptions{
			LowerBound: []byte(prefix),
			UpperBound: []byte(prefix + string([]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff})),
		})
	} else {
		iter, err = dbInstance.NewIter(nil)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create iterator: %v", err)
	}
	defer iter.Close()

	// Calculate skip count
	skip := cursor
	count := 0
	total := 0

	// First, count total records
	if fromTo != "" {
		// Count records with prefix
		for iter.First(); iter.Valid(); iter.Next() {
			total++
		}
	} else {
		// Count all records
		for iter.First(); iter.Valid(); iter.Next() {
			total++
		}
	}

	// Then, get records in reverse order with cursor pagination
	for iter.Last(); iter.Valid(); iter.Prev() {
		if count < skip {
			count++
			continue
		}

		if len(results) >= size {
			break
		}

		key := string(iter.Key())
		value := string(iter.Value())

		// Parse key: fromMetaId_toMetaId_index (with zero-padding)
		keyParts := strings.Split(key, "_")
		fromMetaId := ""
		toMetaId := ""
		index := ""
		if len(keyParts) >= 3 {
			fromMetaId = keyParts[0]
			toMetaId = keyParts[1]
			// Remove leading zeros and get the actual index
			indexStr := strings.TrimLeft(keyParts[2], "0")
			if indexStr == "" {
				indexStr = "0" // If all zeros, treat as 0
			}
			index = indexStr
		}

		// Parse value: pinId_chatType_timestamp_isSet
		valueParts := strings.Split(value, "_")
		pinId := ""
		chatType := ""
		timestamp := ""
		isSet := ""
		if len(valueParts) >= 4 {
			pinId = valueParts[0]
			chatType = valueParts[1]
			timestamp = valueParts[2]
			isSet = valueParts[3]
		}

		results = append(results, map[string]interface{}{
			"key":        key,
			"value":      value,
			"fromMetaId": fromMetaId,
			"toMetaId":   toMetaId,
			"index":      index,
			"pinId":      pinId,
			"chatType":   chatType,
			"timestamp":  timestamp,
			"isSet":      isSet,
		})
	}

	return map[string]interface{}{
		"total":   total,
		"cursor":  cursor,
		"size":    size,
		"results": results,
	}, nil
}

// GetGroupChatIndexKeys Get TalkGroupChatIndexCollection key list with cursor pagination
func GetGroupChatIndexKeys(cursor, size int, groupId string) (map[string]interface{}, error) {
	if cursor < 0 {
		cursor = 0
	}
	if size <= 0 {
		size = 20
	}

	dbInstance, exists := db.Pb[db.TalkGroupChatIndexCollection]
	if !exists {
		return nil, fmt.Errorf("database %s does not exist", db.TalkGroupChatIndexCollection)
	}

	var results []string
	var iter *pebble.Iterator
	var err error

	// If groupId is provided, use prefix filtering
	if groupId != "" {
		prefix := groupId + "_"
		iter, err = dbInstance.NewIter(&pebble.IterOptions{
			LowerBound: []byte(prefix),
			UpperBound: []byte(prefix + string([]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff})),
		})
	} else {
		iter, err = dbInstance.NewIter(nil)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create iterator: %v", err)
	}
	defer iter.Close()

	// Calculate skip count
	skip := cursor
	count := 0
	total := 0

	// First, count total records
	if groupId != "" {
		// Count records with prefix
		for iter.First(); iter.Valid(); iter.Next() {
			total++
		}
	} else {
		// Count all records
		for iter.First(); iter.Valid(); iter.Next() {
			total++
		}
	}

	// Then, get keys in reverse order with cursor pagination
	for iter.Last(); iter.Valid(); iter.Prev() {
		if count < skip {
			count++
			continue
		}

		if len(results) >= size {
			break
		}

		key := string(iter.Key())
		// Process the key to remove zero-padding for better readability
		keyParts := strings.Split(key, "_")
		if len(keyParts) >= 2 {
			// Remove leading zeros from the index part
			indexStr := strings.TrimLeft(keyParts[1], "0")
			if indexStr == "" {
				indexStr = "0" // If all zeros, treat as 0
			}
			processedKey := keyParts[0] + "_" + indexStr
			results = append(results, processedKey)
		} else {
			results = append(results, key)
		}
	}

	return map[string]interface{}{
		"total":  total,
		"cursor": cursor,
		"size":   size,
		"keys":   results,
	}, nil
}

// GetGroupChannelChatIndexKeys Get TalkGroupChannelChatIndexCollection key list with cursor pagination
func GetGroupChannelChatIndexKeys(cursor, size int, channelId string) (map[string]interface{}, error) {
	if cursor < 0 {
		cursor = 0
	}
	if size <= 0 {
		size = 20
	}

	dbInstance, exists := db.Pb[db.TalkGroupChannelChatIndexCollection]
	if !exists {
		return nil, fmt.Errorf("database %s does not exist", db.TalkGroupChannelChatIndexCollection)
	}

	var results []string
	var iter *pebble.Iterator
	var err error

	// If groupId is provided, use prefix filtering
	if channelId != "" {
		prefix := channelId + "_"
		iter, err = dbInstance.NewIter(&pebble.IterOptions{
			LowerBound: []byte(prefix),
			UpperBound: []byte(prefix + string([]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff})),
		})
	} else {
		iter, err = dbInstance.NewIter(nil)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create iterator: %v", err)
	}
	defer iter.Close()

	// Calculate skip count
	skip := cursor
	count := 0
	total := 0

	// First, count total records
	if channelId != "" {
		// Count records with prefix
		for iter.First(); iter.Valid(); iter.Next() {
			total++
		}
	} else {
		// Count all records
		for iter.First(); iter.Valid(); iter.Next() {
			total++
		}
	}

	// Then, get keys in reverse order with cursor pagination
	for iter.Last(); iter.Valid(); iter.Prev() {
		if count < skip {
			count++
			continue
		}

		if len(results) >= size {
			break
		}

		key := string(iter.Key())
		// Process the key to remove zero-padding for better readability
		keyParts := strings.Split(key, "_")
		if len(keyParts) >= 2 {
			// Remove leading zeros from the index part
			indexStr := strings.TrimLeft(keyParts[1], "0")
			if indexStr == "" {
				indexStr = "0" // If all zeros, treat as 0
			}
			processedKey := keyParts[0] + "_" + indexStr
			results = append(results, processedKey)
		} else {
			results = append(results, key)
		}
	}

	return map[string]interface{}{
		"total":  total,
		"cursor": cursor,
		"size":   size,
		"keys":   results,
	}, nil
}

// GetGroupChatTimestamp2OutList Get TalkGroupChatTimestamp2OutCollection list with cursor pagination and reverse order
func GetGroupChatTimestamp2OutList(cursor, size int, groupId string) (map[string]interface{}, error) {
	if cursor < 0 {
		cursor = 0
	}
	if size <= 0 {
		size = 20
	}

	dbInstance, exists := db.Pb[db.TalkGroupChatTimestamp2OutCollection]
	if !exists {
		return nil, fmt.Errorf("database %s does not exist", db.TalkGroupChatTimestamp2OutCollection)
	}

	var results []map[string]interface{}
	var iter *pebble.Iterator
	var err error

	// If groupId is provided, use prefix filtering
	if groupId != "" {
		prefix := groupId + "_"
		iter, err = dbInstance.NewIter(&pebble.IterOptions{
			LowerBound: []byte(prefix),
			UpperBound: []byte(prefix + string([]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff})),
		})
	} else {
		iter, err = dbInstance.NewIter(nil)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create iterator: %v", err)
	}
	defer iter.Close()

	// Calculate skip count
	skip := cursor
	count := 0
	total := 0

	// First, count total records
	if groupId != "" {
		// Count records with prefix
		for iter.First(); iter.Valid(); iter.Next() {
			total++
		}
	} else {
		// Count all records
		for iter.First(); iter.Valid(); iter.Next() {
			total++
		}
	}

	// Then, get records in reverse order with cursor pagination
	for iter.Last(); iter.Valid(); iter.Prev() {
		if count < skip {
			count++
			continue
		}

		if len(results) >= size {
			break
		}

		key := string(iter.Key())
		value := string(iter.Value())

		results = append(results, map[string]interface{}{
			"key":   key,
			"value": value,
		})
	}

	return map[string]interface{}{
		"total":   total,
		"cursor":  cursor,
		"size":    size,
		"results": results,
	}, nil
}

// GetGroupChatTimestamp2OutList Get TalkGroupChannelChatTimestamp2OutCollection list with cursor pagination and reverse order
func GetGroupChannelChatTimestampOutList(cursor, size int, channelId string) (map[string]interface{}, error) {
	if cursor < 0 {
		cursor = 0
	}
	if size <= 0 {
		size = 20
	}

	dbInstance, exists := db.Pb[db.TalkGroupChannelChatTimestamp2OutCollection]
	if !exists {
		return nil, fmt.Errorf("database %s does not exist", db.TalkGroupChannelChatTimestamp2OutCollection)
	}

	var results []map[string]interface{}
	var iter *pebble.Iterator
	var err error

	// If groupId is provided, use prefix filtering
	if channelId != "" {
		prefix := channelId + "_"
		iter, err = dbInstance.NewIter(&pebble.IterOptions{
			LowerBound: []byte(prefix),
			UpperBound: []byte(prefix + string([]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff})),
		})
	} else {
		iter, err = dbInstance.NewIter(nil)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create iterator: %v", err)
	}
	defer iter.Close()

	// Calculate skip count
	skip := cursor
	count := 0
	total := 0

	// First, count total records
	if channelId != "" {
		// Count records with prefix
		for iter.First(); iter.Valid(); iter.Next() {
			total++
		}
	} else {
		// Count all records
		for iter.First(); iter.Valid(); iter.Next() {
			total++
		}
	}

	// Then, get records in reverse order with cursor pagination
	for iter.Last(); iter.Valid(); iter.Prev() {
		if count < skip {
			count++
			continue
		}

		if len(results) >= size {
			break
		}

		key := string(iter.Key())
		value := string(iter.Value())

		results = append(results, map[string]interface{}{
			"key":   key,
			"value": value,
		})
	}

	return map[string]interface{}{
		"total":   total,
		"cursor":  cursor,
		"size":    size,
		"results": results,
	}, nil
}

// GetLuckyBagCollectionList get lucky bag collection list with pagination
func GetLuckyBagCollectionList(collectionName string, cursor int, size int) (map[string]interface{}, error) {
	dbInstance, exists := db.Pb[collectionName]
	if !exists {
		return nil, fmt.Errorf("database %s does not exist", collectionName)
	}

	// Validate collection name
	validCollections := []string{
		db.TalkGroupLuckyBagPinPendingCollection,
		db.TalkGroupLuckyBagPinCompletedCollection,
		db.TalkGroupLuckyBagPinTimeoutResidueCollection,
		db.TalkGroupLuckyBagPinErrPendingCollection,
		db.TalkGroupLuckyBagPinErrTimeoutResidueCollection,
	}

	isValid := false
	for _, validCollection := range validCollections {
		if collectionName == validCollection {
			isValid = true
			break
		}
	}

	if !isValid {
		return nil, fmt.Errorf("invalid collection name: %s", collectionName)
	}

	// Set default values
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
	iter, err := dbInstance.NewIter(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create iterator: %v", err)
	}
	defer iter.Close()

	count := 0
	skipCount := 0

	// Iterate through all keys
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

	// Calculate next cursor
	nextCursor := cursor + count
	if count < size {
		nextCursor = -1 // No more data
	}

	result := map[string]interface{}{
		"collection": collectionName,
		"cursor":     cursor,
		"size":       size,
		"nextCursor": nextCursor,
		"count":      count,
		"data":       results,
	}

	return result, nil
}

// GetLuckyBagQueueList get lucky bag queue collection list with pagination
func GetLuckyBagQueueList(collectionName string, cursor int, size int) (map[string]interface{}, error) {
	dbInstance, exists := db.Pb[collectionName]
	if !exists {
		return nil, fmt.Errorf("database %s does not exist", collectionName)
	}

	// Validate collection name
	validCollections := []string{
		db.TalkGroupOpenLuckyBagQueueCollection,
		db.TalkGroupResidueLuckyBagQueueCollection,
	}

	isValid := false
	for _, validCollection := range validCollections {
		if collectionName == validCollection {
			isValid = true
			break
		}
	}

	if !isValid {
		return nil, fmt.Errorf("invalid collection name: %s", collectionName)
	}

	// Set default values
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
	iter, err := dbInstance.NewIter(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create iterator: %v", err)
	}
	defer iter.Close()

	count := 0
	skipCount := 0

	// Iterate through all keys
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

	// Calculate next cursor
	nextCursor := cursor + count
	if count < size {
		nextCursor = -1 // No more data
	}

	result := map[string]interface{}{
		"collection": collectionName,
		"cursor":     cursor,
		"size":       size,
		"nextCursor": nextCursor,
		"count":      count,
		"data":       results,
	}

	return result, nil
}

// GetResidueLuckyBagByPinId get residue lucky bag data by specific pinId
func GetResidueLuckyBagByPinId(pinId string) (map[string]interface{}, error) {
	// Query by pinId from TalkGroupResidueLuckyBagPinCollection
	value, closer, err := db.Pb[db.TalkGroupResidueLuckyBagPinCollection].Get([]byte(pinId))
	if err != nil {
		if err == pebble.ErrNotFound {
			return map[string]interface{}{
				"pinId":   pinId,
				"found":   false,
				"message": "Residue lucky bag not found",
			}, nil
		}
		return nil, fmt.Errorf("query failed: %v", err)
	}
	defer closer.Close()

	// Try to parse JSON
	var jsonData interface{}
	if err := json.Unmarshal(value, &jsonData); err != nil {
		// If not JSON, use string directly
		jsonData = string(value)
	}

	result := map[string]interface{}{
		"pinId": pinId,
		"found": true,
		"value": jsonData,
	}

	return result, nil
}

// GetResidueLuckyBagList get residue lucky bag collection list with pagination
func GetResidueLuckyBagList(cursor int, size int) (map[string]interface{}, error) {
	// Set default values
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
	iter, err := db.Pb[db.TalkGroupResidueLuckyBagPinCollection].NewIter(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create iterator: %v", err)
	}
	defer iter.Close()

	count := 0
	skipCount := 0

	// Iterate through all keys
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

	// Calculate next cursor
	nextCursor := cursor + count
	if count < size {
		nextCursor = -1 // No more data
	}

	result := map[string]interface{}{
		"collection": db.TalkGroupResidueLuckyBagPinCollection,
		"cursor":     cursor,
		"size":       size,
		"nextCursor": nextCursor,
		"count":      count,
		"data":       results,
	}

	return result, nil
}

// GetPrivateChatTimestampList get private chat timestamp collection list with pagination
func GetPrivateChatTimestampList(from, to string, cursor int, size int) (map[string]interface{}, error) {
	// Set default values
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

	// Construct prefix for filtering
	fromToPrefix := from + "_" + to + "_"

	// Create iterator with prefix bounds for efficient querying
	iter, err := db.Pb[db.TalkPrivateChatTimestampCollection].NewIter(&pebble.IterOptions{
		LowerBound: []byte(fromToPrefix),
		UpperBound: []byte(fromToPrefix + string([]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff})),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create iterator: %v", err)
	}
	defer iter.Close()

	count := 0
	skipCount := 0

	// Iterate through keys in reverse order starting from the prefix
	for iter.Last(); iter.Valid(); iter.Prev() {
		key := string(iter.Key())

		// Skip until cursor
		if skipCount < cursor {
			skipCount++
			continue
		}

		// Check if we've reached the size limit
		if count >= size {
			break
		}

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

	// Calculate next cursor
	nextCursor := cursor + count
	if count < size {
		nextCursor = -1 // No more data
	}

	result := map[string]interface{}{
		"collection": db.TalkPrivateChatTimestampCollection,
		"from":       from,
		"to":         to,
		"cursor":     cursor,
		"size":       size,
		"nextCursor": nextCursor,
		"count":      count,
		"data":       results,
	}

	return result, nil
}

// GetLuckyBagCollectionByPinId get lucky bag collection data by specific pinId
func GetLuckyBagCollectionByPinId(collectionName, pinId string) (map[string]interface{}, error) {
	dbInstance, exists := db.Pb[collectionName]
	if !exists {
		return nil, fmt.Errorf("database %s does not exist", collectionName)
	}

	// Validate collection name
	validCollections := []string{
		db.TalkGroupLuckyBagPinPendingCollection,
		db.TalkGroupLuckyBagPinCompletedCollection,
		db.TalkGroupLuckyBagPinTimeoutResidueCollection,
		db.TalkGroupLuckyBagPinErrPendingCollection,
		db.TalkGroupLuckyBagPinErrTimeoutResidueCollection,
	}

	isValid := false
	for _, validCollection := range validCollections {
		if collectionName == validCollection {
			isValid = true
			break
		}
	}

	if !isValid {
		return nil, fmt.Errorf("invalid collection name: %s", collectionName)
	}

	// Query by pinId
	value, closer, err := dbInstance.Get([]byte(pinId))
	if err != nil {
		if err == pebble.ErrNotFound {
			return map[string]interface{}{
				"collection": collectionName,
				"pinId":      pinId,
				"found":      false,
				"message":    "PinId not found in collection",
			}, nil
		}
		return nil, fmt.Errorf("query failed: %v", err)
	}
	defer closer.Close()

	// Try to parse JSON
	var jsonData interface{}
	if err := json.Unmarshal(value, &jsonData); err != nil {
		// If not JSON, use string directly
		jsonData = string(value)
	}

	result := map[string]interface{}{
		"collection": collectionName,
		"pinId":      pinId,
		"found":      true,
		"key":        pinId,
		"value":      jsonData,
	}

	return result, nil
}

// GetMetaIdContextListByMetaId Get TalkMetaIdContextListCollection data by metaId
func GetMetaIdContextListByMetaId(metaId string) (map[string]interface{}, error) {
	if metaId == "" {
		return nil, fmt.Errorf("metaId parameter cannot be empty")
	}

	// Query by metaId from TalkMetaIdContextListCollection
	value, closer, err := db.Pb[db.TalkMetaIdContextListCollection].Get([]byte(metaId))
	if err != nil {
		if err == pebble.ErrNotFound {
			return map[string]interface{}{
				"metaId":  metaId,
				"found":   false,
				"message": "MetaId context list not found",
			}, nil
		}
		return nil, fmt.Errorf("query failed: %v", err)
	}
	defer closer.Close()

	// Try to parse JSON
	var jsonData interface{}
	if err := json.Unmarshal(value, &jsonData); err != nil {
		// If not JSON, use string directly
		jsonData = string(value)
	}

	result := map[string]interface{}{
		"metaId": metaId,
		"found":  true,
		"value":  jsonData,
	}

	return result, nil
}

// GetPrivateChatTimestampOutList get private chat timestamp out collection list with pagination
func GetPrivateChatTimestampOutList(from, to string, cursor int, size int) (map[string]interface{}, error) {
	// Set default values
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

	// Construct prefix for filtering
	fromToPrefix := from + "_" + to + "_"

	// Create iterator with prefix bounds for efficient querying
	iter, err := db.Pb[db.TalkPrivateChatTimestampOutCollection].NewIter(&pebble.IterOptions{
		LowerBound: []byte(fromToPrefix),
		UpperBound: []byte(fromToPrefix + string([]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff})),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create iterator: %v", err)
	}
	defer iter.Close()

	count := 0
	skipCount := 0

	// Iterate through keys in reverse order starting from the prefix
	for iter.Last(); iter.Valid(); iter.Prev() {
		key := string(iter.Key())

		// Skip until cursor
		if skipCount < cursor {
			skipCount++
			continue
		}

		// Check if we've reached the size limit
		if count >= size {
			break
		}

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

	// Calculate next cursor
	nextCursor := cursor + count
	if count < size {
		nextCursor = -1 // No more data
	}

	result := map[string]interface{}{
		"collection": db.TalkPrivateChatTimestampOutCollection,
		"from":       from,
		"to":         to,
		"cursor":     cursor,
		"size":       size,
		"nextCursor": nextCursor,
		"count":      count,
		"data":       results,
	}

	return result, nil
}

// GetGroupPersonListCollection Get TalkGroupPersonListCollection data with pagination
// Returns key (groupId) and value (total member count) list
func GetGroupPersonListCollection(cursor, size int) (map[string]interface{}, error) {
	if size <= 0 {
		size = 20 // Default size
	}
	if cursor < 0 {
		cursor = 0 // Default cursor
	}

	var results []map[string]interface{}

	// Create iterator for TalkGroupPersonListCollection
	iter, err := db.Pb[db.TalkGroupPersonListCollection].NewIter(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create iterator: %v", err)
	}
	defer iter.Close()

	count := 0
	skipCount := 0
	total := 0

	// Iterate through all keys
	for iter.First(); iter.Valid(); iter.Next() {
		key := string(iter.Key())
		total++

		// Skip until cursor
		if skipCount < cursor {
			skipCount++
			continue
		}

		// Check if we've reached the size limit
		if count >= size {
			break
		}

		// Parse the value to get member count
		var personList struct {
			GroupId string        `json:"groupId"`
			Persons []interface{} `json:"persons"`
		}

		if err := json.Unmarshal(iter.Value(), &personList); err != nil {
			// If parsing fails, use 0 as member count
			results = append(results, map[string]interface{}{
				"groupId":     key,
				"memberCount": 0,
			})
		} else {
			// Count active members (GroupState == 1)
			activeMemberCount := 0
			for _, person := range personList.Persons {
				if personMap, ok := person.(map[string]interface{}); ok {
					if groupState, exists := personMap["groupState"]; exists {
						if state, ok := groupState.(float64); ok && state == 1 {
							activeMemberCount++
						}
					}
				}
			}

			results = append(results, map[string]interface{}{
				"groupId":     key,
				"memberCount": activeMemberCount,
			})
		}
		count++
	}

	// Calculate next cursor
	nextCursor := cursor + count
	if count < size {
		nextCursor = -1 // No more data
	}

	result := map[string]interface{}{
		"collection": db.TalkGroupPersonListCollection,
		"cursor":     cursor,
		"size":       size,
		"nextCursor": nextCursor,
		"count":      count,
		"data":       results,
		"total":      total,
	}

	return result, nil
}

// GetLuckyBagErrorCollectionKeys Get keys from lucky bag error collections with pagination
func GetLuckyBagErrorCollectionKeys(collectionName string, cursor int, size int) (map[string]interface{}, error) {
	if size <= 0 {
		size = 20
	}
	if cursor < 0 {
		cursor = 0
	}

	// Validate collection name
	if collectionName != db.TalkGroupOpenLuckyBagErrCollection && collectionName != db.TalkGroupResidueLuckyBagErrCollection {
		return nil, fmt.Errorf("invalid collection name: %s", collectionName)
	}

	// Get collection
	collection, exists := db.Pb[collectionName]
	if !exists {
		return nil, fmt.Errorf("collection %s not found", collectionName)
	}

	// Get total count
	totalCount, err := collection.EstimateDiskUsage(nil, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to estimate disk usage: %v", err)
	}

	// Get keys with pagination
	var keys []string
	var nextCursor int

	iter, err := collection.NewIter(&pebble.IterOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to create iterator: %v", err)
	}
	defer iter.Close()

	// Skip to cursor position
	skipped := 0
	for iter.First(); iter.Valid() && skipped < cursor; iter.Next() {
		skipped++
	}

	// Collect keys
	for iter.Valid() && len(keys) < size {
		key := string(iter.Key())
		keys = append(keys, key)
		iter.Next()
	}

	// Check if there are more items
	hasMore := iter.Valid()
	if hasMore {
		nextCursor = cursor + size
	}

	return map[string]interface{}{
		"collection": collectionName,
		"keys":       keys,
		"total":      totalCount,
		"cursor":     cursor,
		"size":       size,
		"hasMore":    hasMore,
		"nextCursor": nextCursor,
	}, nil
}

// GetLuckyBagPinByPinId Get lucky bag pin data by pinId from specified collection
func GetLuckyBagPinByPinId(collectionName, pinId string) (map[string]interface{}, error) {
	// Validate collection name
	if collectionName != db.TalkGroupOpenLuckyBagPinCollection && collectionName != db.TalkGroupResidueLuckyBagPinCollection {
		return nil, fmt.Errorf("invalid collection name: %s", collectionName)
	}

	// Get collection
	collection, exists := db.Pb[collectionName]
	if !exists {
		return nil, fmt.Errorf("collection %s not found", collectionName)
	}

	// Get value by key
	value, closer, err := collection.Get([]byte(pinId))
	if err != nil {
		if err == pebble.ErrNotFound {
			return nil, fmt.Errorf("pinId %s not found in collection %s", pinId, collectionName)
		}
		return nil, fmt.Errorf("failed to get value: %v", err)
	}
	defer closer.Close()

	// Parse JSON value
	var data map[string]interface{}
	err = json.Unmarshal(value, &data)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %v", err)
	}

	return map[string]interface{}{
		"collection": collectionName,
		"pinId":      pinId,
		"data":       data,
	}, nil
}

// GetLuckyBagCodeAddressKeyFromCompleted Get lucky bag code address key from completed collection
func GetLuckyBagCodeAddressKeyFromCompleted(code, address string) (map[string]interface{}, error) {
	// Validate input parameters
	if code == "" || address == "" {
		return nil, fmt.Errorf("code and address cannot be empty")
	}

	// Construct key: code_address
	key := code + "_" + address

	// Get collection
	collection, exists := db.Pb[db.TalkGroupLuckyBagCodeAddressKeyCompletedCollection]
	if !exists {
		return nil, fmt.Errorf("collection %s not found", db.TalkGroupLuckyBagCodeAddressKeyCompletedCollection)
	}

	// Get value by key
	value, closer, err := collection.Get([]byte(key))
	if err != nil {
		if err == pebble.ErrNotFound {
			return nil, fmt.Errorf("lucky bag code address key not found for code %s and address %s", code, address)
		}
		return nil, fmt.Errorf("failed to get code address key from completed collection: %v", err)
	}
	defer closer.Close()

	// Parse JSON value
	var codeAddressKey struct {
		Key             string `json:"key"`
		Code            string `json:"code"`
		LuckyBagAddress string `json:"luckyBagAddress"`
		Timestamp       int64  `json:"timestamp"`
	}
	err = json.Unmarshal(value, &codeAddressKey)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal code address key: %v", err)
	}

	// Return data without the private key for security
	result := map[string]interface{}{
		"code":            codeAddressKey.Code,
		"luckyBagAddress": codeAddressKey.LuckyBagAddress,
		"timestamp":       codeAddressKey.Timestamp,
	}

	return result, nil
}

// RetryFailedLuckyBagOperation Retry failed lucky bag operation by pinId
func RetryFailedLuckyBagOperation(pinId string) (map[string]interface{}, error) {
	// Validate input parameter
	if pinId == "" {
		return nil, fmt.Errorf("pinId cannot be empty")
	}

	// First try to get from TalkGroupOpenLuckyBagErrCollection
	openErrCollection, exists := db.Pb[db.TalkGroupOpenLuckyBagErrCollection]
	if exists {
		value, closer, err := openErrCollection.Get([]byte(pinId))
		if err == nil {
			defer closer.Close()
			luckyBagPinId := string(value)

			// Get the original open lucky bag record
			openCollection, exists := db.Pb[db.TalkGroupOpenLuckyBagPinCollection]
			if exists {
				openValue, openCloser, err := openCollection.Get([]byte(pinId))
				if err == nil {
					defer openCloser.Close()

					// Parse the open lucky bag record
					var openLuckyBag models.TalkGroupOpenLuckyBagV3
					err = json.Unmarshal(openValue, &openLuckyBag)
					if err == nil {
						// Update grabState to 1 (centralized open)
						openLuckyBag.GrabState = 1
						openLuckyBag.RetryCount = 0

						// Save updated record
						updatedData, err := json.Marshal(openLuckyBag)
						if err == nil {
							err = openCollection.Set([]byte(pinId), updatedData, pebble.Sync)
							if err == nil {
								// Requeue to TalkGroupOpenLuckyBagQueueCollection
								err = requeueOpenLuckyBag(&openLuckyBag)
								if err == nil {
									// Remove from error collection
									openErrCollection.Delete([]byte(pinId), pebble.Sync)

									return map[string]interface{}{
										"type":          "open",
										"pinId":         pinId,
										"luckyBagPinId": luckyBagPinId,
										"message":       "Successfully requeued open lucky bag operation",
									}, nil
								} else {
									return nil, fmt.Errorf("failed to requeue open lucky bag: %v", err)
								}
							} else {
								return nil, fmt.Errorf("failed to save updated data to open collection: %v", err)
							}
						} else {
							return nil, fmt.Errorf("failed to marshal updated data: %v", err)
						}
					}
				} else {
					return nil, fmt.Errorf("failed to get value from open collection: %v", err)
				}
			}
		} else {
			return nil, fmt.Errorf("failed to get value from open error collection: %v", err)
		}
	}

	// If not found in open error collection, try TalkGroupResidueLuckyBagErrCollection
	residueErrCollection, exists := db.Pb[db.TalkGroupResidueLuckyBagErrCollection]
	if exists {
		value, closer, err := residueErrCollection.Get([]byte(pinId))
		if err == nil {
			defer closer.Close()
			luckyBagPinId := string(value)

			// Get the original residue lucky bag record
			residueCollection, exists := db.Pb[db.TalkGroupResidueLuckyBagPinCollection]
			if exists {
				residueValue, residueCloser, err := residueCollection.Get([]byte(pinId))
				if err == nil {
					defer residueCloser.Close()

					// Parse the residue lucky bag record
					var residueLuckyBag models.TalkGroupResidueLuckyBagV3
					err = json.Unmarshal(residueValue, &residueLuckyBag)
					if err == nil {
						// Update reclaimState to 1 (centralized reclaim)
						residueLuckyBag.ReclaimState = 1
						residueLuckyBag.RetryCount = 0

						// Save updated record
						updatedData, err := json.Marshal(residueLuckyBag)
						if err == nil {
							err = residueCollection.Set([]byte(pinId), updatedData, pebble.Sync)
							if err == nil {
								// Requeue to TalkGroupResidueLuckyBagQueueCollection
								err = requeueResidueLuckyBag(&residueLuckyBag)
								if err == nil {
									// Remove from error collection
									residueErrCollection.Delete([]byte(pinId), pebble.Sync)

									return map[string]interface{}{
										"type":          "residue",
										"pinId":         pinId,
										"luckyBagPinId": luckyBagPinId,
										"message":       "Successfully requeued residue lucky bag operation",
									}, nil
								} else {
									return nil, fmt.Errorf("failed to requeue residue lucky bag: %v", err)
								}
							} else {
								return nil, fmt.Errorf("failed to save updated data to residue collection: %v", err)
							}
						} else {
							return nil, fmt.Errorf("failed to marshal updated data: %v", err)
						}
					} else {
						return nil, fmt.Errorf("failed to unmarshal residue lucky bag record: %v", err)
					}
				} else {
					return nil, fmt.Errorf("failed to get value from residue collection: %v", err)
				}
			}
		} else {
			return nil, fmt.Errorf("failed to get value from residue error collection: %v", err)
		}
	}

	return nil, fmt.Errorf("pinId %s not found in any error collection", pinId)
}

// requeueOpenLuckyBag requeue open lucky bag to queue collection
func requeueOpenLuckyBag(openLuckyBag *models.TalkGroupOpenLuckyBagV3) error {
	// Create queue message
	queueMessage := map[string]interface{}{
		"pinId":        openLuckyBag.PinId,
		"openLuckyBag": openLuckyBag,
		"timestamp":    time.Now().Unix(),
		"retryCount":   0,
		"status":       "pending",
	}

	// Marshal queue message
	data, err := json.Marshal(queueMessage)
	if err != nil {
		return fmt.Errorf("failed to marshal queue message: %v", err)
	}

	// Use timestamp_pinId as key
	key := fmt.Sprintf("%d_%s", time.Now().Unix(), openLuckyBag.PinId)

	// Get queue collection
	queueCollection, exists := db.Pb[db.TalkGroupOpenLuckyBagQueueCollection]
	if !exists {
		return fmt.Errorf("queue collection %s not found", db.TalkGroupOpenLuckyBagQueueCollection)
	}

	// Save to queue
	return queueCollection.Set([]byte(key), data, pebble.Sync)
}

// requeueResidueLuckyBag requeue residue lucky bag to queue collection
func requeueResidueLuckyBag(residueLuckyBag *models.TalkGroupResidueLuckyBagV3) error {
	// Create queue message
	queueMessage := map[string]interface{}{
		"pinId":           residueLuckyBag.PinId,
		"residueLuckyBag": residueLuckyBag,
		"timestamp":       time.Now().Unix(),
		"retryCount":      0,
		"status":          "pending",
	}

	// Marshal queue message
	data, err := json.Marshal(queueMessage)
	if err != nil {
		return fmt.Errorf("failed to marshal queue message: %v", err)
	}

	// Use timestamp_pinId as key
	key := fmt.Sprintf("%d_%s", time.Now().Unix(), residueLuckyBag.PinId)

	// Get queue collection
	queueCollection, exists := db.Pb[db.TalkGroupResidueLuckyBagQueueCollection]
	if !exists {
		return fmt.Errorf("queue collection %s not found", db.TalkGroupResidueLuckyBagQueueCollection)
	}

	// Save to queue
	return queueCollection.Set([]byte(key), data, pebble.Sync)
}

// RetryFailedLuckyBagOperationsByLuckyBagId Retry failed lucky bag operations by lucky bag ID
func RetryFailedLuckyBagOperationsByLuckyBagId(luckyBagId string) (map[string]interface{}, error) {
	// Validate input parameter
	if luckyBagId == "" {
		return nil, fmt.Errorf("luckyBagId cannot be empty")
	}

	// Get error collection
	openErrCollection, exists := db.Pb[db.TalkGroupOpenLuckyBagErrCollection]
	if !exists {
		return nil, fmt.Errorf("error collection %s not found", db.TalkGroupOpenLuckyBagErrCollection)
	}

	// Get open lucky bag pin collection to match lucky bag ID
	openPinCollection, exists := db.Pb[db.TalkGroupOpenLuckyBagPinCollection]
	if !exists {
		return nil, fmt.Errorf("open lucky bag pin collection %s not found", db.TalkGroupOpenLuckyBagPinCollection)
	}

	var retryResults []map[string]interface{}
	var retryErrors []string

	// Iterate through error collection to find matching lucky bag IDs
	iter, err := openErrCollection.NewIter(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create iterator: %v", err)
	}
	defer iter.Close()

	for iter.First(); iter.Valid(); iter.Next() {
		errorPinId := string(iter.Key())

		// Get the open lucky bag record to check if it belongs to the target lucky bag
		openValue, openCloser, err := openPinCollection.Get([]byte(errorPinId))
		if err == nil {
			defer openCloser.Close()

			// Parse the open lucky bag record
			var openLuckyBag models.TalkGroupOpenLuckyBagV3
			err = json.Unmarshal(openValue, &openLuckyBag)
			if err == nil && openLuckyBag.LuckyBagPinId == luckyBagId {
				// This record matches the target lucky bag ID, retry it
				result, err := RetryFailedLuckyBagOperation(errorPinId)
				if err != nil {
					retryErrors = append(retryErrors, fmt.Sprintf("Failed to retry pinId %s: %v", errorPinId, err))
				} else {
					retryResults = append(retryResults, result)
				}
			}
		} else {
			fmt.Printf("Failed to get value from open collection: %v", err)
		}
	}

	// Build response
	response := map[string]interface{}{
		"luckyBagId":   luckyBagId,
		"totalFound":   len(retryResults) + len(retryErrors),
		"successCount": len(retryResults),
		"errorCount":   len(retryErrors),
		"retryResults": retryResults,
		"retryErrors":  retryErrors,
	}

	if len(retryErrors) > 0 {
		response["message"] = fmt.Sprintf("Retried %d operations, %d succeeded, %d failed", len(retryResults)+len(retryErrors), len(retryResults), len(retryErrors))
	} else {
		response["message"] = fmt.Sprintf("Successfully retried %d operations", len(retryResults))
	}

	return response, nil
}

// Group admin, block, and whitelist related query methods

// QueryGroupAdminCollection Get TalkGroupAdminCollection data with pagination
func QueryGroupAdminCollection(cursor, size int) (map[string]interface{}, error) {
	if size <= 0 {
		size = 20 // Default size
	}
	if cursor < 0 {
		cursor = 0 // Default cursor
	}

	var results []map[string]interface{}

	// Create iterator for TalkGroupAdminCollection
	iter, err := db.Pb[db.TalkGroupAdminCollection].NewIter(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create iterator: %v", err)
	}
	defer iter.Close()

	count := 0
	skipCount := 0
	total := 0

	// Iterate through all keys
	for iter.First(); iter.Valid(); iter.Next() {
		key := string(iter.Key())
		total++

		// Skip until cursor
		if skipCount < cursor {
			skipCount++
			continue
		}

		// Check if we've reached the size limit
		if count >= size {
			break
		}

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

	// Calculate next cursor
	nextCursor := cursor + count
	if count < size {
		nextCursor = -1 // No more data
	}

	result := map[string]interface{}{
		"collection": db.TalkGroupAdminCollection,
		"cursor":     cursor,
		"size":       size,
		"nextCursor": nextCursor,
		"count":      count,
		"data":       results,
		"total":      total,
	}

	return result, nil
}

// QueryGroupBlockCollection Get TalkGroupBlockCollection data with pagination
func QueryGroupBlockCollection(cursor, size int) (map[string]interface{}, error) {
	if size <= 0 {
		size = 20 // Default size
	}
	if cursor < 0 {
		cursor = 0 // Default cursor
	}

	var results []map[string]interface{}

	// Create iterator for TalkGroupBlockCollection
	iter, err := db.Pb[db.TalkGroupBlockCollection].NewIter(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create iterator: %v", err)
	}
	defer iter.Close()

	count := 0
	skipCount := 0
	total := 0

	// Iterate through all keys
	for iter.First(); iter.Valid(); iter.Next() {
		key := string(iter.Key())
		total++

		// Skip until cursor
		if skipCount < cursor {
			skipCount++
			continue
		}

		// Check if we've reached the size limit
		if count >= size {
			break
		}

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

	// Calculate next cursor
	nextCursor := cursor + count
	if count < size {
		nextCursor = -1 // No more data
	}

	result := map[string]interface{}{
		"collection": db.TalkGroupBlockCollection,
		"cursor":     cursor,
		"size":       size,
		"nextCursor": nextCursor,
		"count":      count,
		"data":       results,
		"total":      total,
	}

	return result, nil
}

// QueryGroupWhitelistCollection Get TalkGroupWhitelistCollection data with pagination
func QueryGroupWhitelistCollection(cursor, size int) (map[string]interface{}, error) {
	if size <= 0 {
		size = 20 // Default size
	}
	if cursor < 0 {
		cursor = 0 // Default cursor
	}

	var results []map[string]interface{}

	// Create iterator for TalkGroupWhitelistCollection
	iter, err := db.Pb[db.TalkGroupWhitelistCollection].NewIter(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create iterator: %v", err)
	}
	defer iter.Close()

	count := 0
	skipCount := 0
	total := 0

	// Iterate through all keys
	for iter.First(); iter.Valid(); iter.Next() {
		key := string(iter.Key())
		total++

		// Skip until cursor
		if skipCount < cursor {
			skipCount++
			continue
		}

		// Check if we've reached the size limit
		if count >= size {
			break
		}

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

	// Calculate next cursor
	nextCursor := cursor + count
	if count < size {
		nextCursor = -1 // No more data
	}

	result := map[string]interface{}{
		"collection": db.TalkGroupWhitelistCollection,
		"cursor":     cursor,
		"size":       size,
		"nextCursor": nextCursor,
		"count":      count,
		"data":       results,
		"total":      total,
	}

	return result, nil
}

// QueryGroupAdminByGroupId Get TalkGroupAdminCollection data by groupId
func QueryGroupAdminByGroupId(groupId string) (map[string]interface{}, error) {
	if groupId == "" {
		return nil, fmt.Errorf("groupId parameter cannot be empty")
	}

	// Query by groupId from TalkGroupAdminCollection
	value, closer, err := db.Pb[db.TalkGroupAdminCollection].Get([]byte(groupId))
	if err != nil {
		if err == pebble.ErrNotFound {
			return map[string]interface{}{
				"groupId": groupId,
				"found":   false,
				"message": "Group admin data not found",
			}, nil
		}
		return nil, fmt.Errorf("query failed: %v", err)
	}
	defer closer.Close()

	// Try to parse JSON
	var jsonData interface{}
	if err := json.Unmarshal(value, &jsonData); err != nil {
		// If not JSON, use string directly
		jsonData = string(value)
	}

	result := map[string]interface{}{
		"groupId": groupId,
		"found":   true,
		"value":   jsonData,
	}

	return result, nil
}

// QueryGroupBlockByGroupId Get TalkGroupBlockCollection data by groupId
func QueryGroupBlockByGroupId(groupId string) (map[string]interface{}, error) {
	if groupId == "" {
		return nil, fmt.Errorf("groupId parameter cannot be empty")
	}

	// Query by groupId from TalkGroupBlockCollection
	value, closer, err := db.Pb[db.TalkGroupBlockCollection].Get([]byte(groupId))
	if err != nil {
		if err == pebble.ErrNotFound {
			return map[string]interface{}{
				"groupId": groupId,
				"found":   false,
				"message": "Group block data not found",
			}, nil
		}
		return nil, fmt.Errorf("query failed: %v", err)
	}
	defer closer.Close()

	// Try to parse JSON
	var jsonData interface{}
	if err := json.Unmarshal(value, &jsonData); err != nil {
		// If not JSON, use string directly
		jsonData = string(value)
	}

	result := map[string]interface{}{
		"groupId": groupId,
		"found":   true,
		"value":   jsonData,
	}

	return result, nil
}

// QueryGroupWhitelistByGroupId Get TalkGroupWhitelistCollection data by groupId
func QueryGroupWhitelistByGroupId(groupId string) (map[string]interface{}, error) {
	if groupId == "" {
		return nil, fmt.Errorf("groupId parameter cannot be empty")
	}

	// Query by groupId from TalkGroupWhitelistCollection
	value, closer, err := db.Pb[db.TalkGroupWhitelistCollection].Get([]byte(groupId))
	if err != nil {
		if err == pebble.ErrNotFound {
			return map[string]interface{}{
				"groupId": groupId,
				"found":   false,
				"message": "Group whitelist data not found",
			}, nil
		}
		return nil, fmt.Errorf("query failed: %v", err)
	}
	defer closer.Close()

	// Try to parse JSON
	var jsonData interface{}
	if err := json.Unmarshal(value, &jsonData); err != nil {
		// If not JSON, use string directly
		jsonData = string(value)
	}

	result := map[string]interface{}{
		"groupId": groupId,
		"found":   true,
		"value":   jsonData,
	}

	return result, nil
}
