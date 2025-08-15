package service

import (
	"encoding/json"
	"fmt"
	"manindexer/basicprotocols/group_chat/db"

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
