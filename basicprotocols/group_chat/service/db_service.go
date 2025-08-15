package service

import (
	"encoding/json"
	"fmt"
	"manindexer/basicprotocols/group_chat/db"

	"github.com/cockroachdb/pebble"
)

type DbService struct{}

// Generic query method - Query data by prefix
func (s *DbService) QueryByPrefix(collectionName, prefix string, limit int) ([]map[string]interface{}, error) {
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

// Generic query method - Query single data by key
func (s *DbService) QueryByKey(collectionName, key string) (map[string]interface{}, error) {
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
func (s *DbService) QueryAll(collectionName string, limit int) ([]map[string]interface{}, error) {
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
func (s *DbService) QueryCommunityInfo(communityId string) (map[string]interface{}, error) {
	return s.QueryByKey(db.TalkCommunityInfoCollection, communityId)
}

// Query all community info
func (s *DbService) QueryAllCommunityInfo(limit int) ([]map[string]interface{}, error) {
	return s.QueryAll(db.TalkCommunityInfoCollection, limit)
}

// Query community version info
func (s *DbService) QueryCommunityVersionInfo(communityId string, limit int) ([]map[string]interface{}, error) {
	prefix := communityId + "_"
	return s.QueryByPrefix(db.TalkCommunityVersionInfoCollection, prefix, limit)
}

// Query community address info
func (s *DbService) QueryCommunityAddress(communityId string, limit int) ([]map[string]interface{}, error) {
	prefix := communityId + "_"
	return s.QueryByPrefix(db.TalkCommunityAddressCollection, prefix, limit)
}

// Query community join records
func (s *DbService) QueryCommunityJoin(communityId string, limit int) ([]map[string]interface{}, error) {
	prefix := communityId + "_"
	return s.QueryByPrefix(db.TalkCommunityJoinCollection, prefix, limit)
}

// Query community members
func (s *DbService) QueryCommunityPerson(communityId string, limit int) ([]map[string]interface{}, error) {
	prefix := communityId + "_"
	return s.QueryByPrefix(db.TalkCommunityPersonCollection, prefix, limit)
}

// Group-related query methods

// Query group info
func (s *DbService) QueryGroupInfo(groupId string) (map[string]interface{}, error) {
	return s.QueryByKey(db.TalkGroupInfoCollection, groupId)
}

// Query all group info
func (s *DbService) QueryAllGroupInfo(limit int) ([]map[string]interface{}, error) {
	return s.QueryAll(db.TalkGroupInfoCollection, limit)
}

// Query group version info
func (s *DbService) QueryGroupVersionInfo(groupId string, limit int) ([]map[string]interface{}, error) {
	prefix := groupId + "_"
	return s.QueryByPrefix(db.TalkGroupVersionInfoCollection, prefix, limit)
}

// Query group community association
func (s *DbService) QueryGroupCommunity(communityId string, limit int) ([]map[string]interface{}, error) {
	prefix := communityId + "_"
	return s.QueryByPrefix(db.TalkGroupCommunityCollection, prefix, limit)
}

// Query group join records
func (s *DbService) QueryGroupJoin(groupId string, limit int) ([]map[string]interface{}, error) {
	prefix := groupId + "_"
	return s.QueryByPrefix(db.TalkGroupJoinCollection, prefix, limit)
}

// Query group members
func (s *DbService) QueryGroupPerson(groupId string, limit int) ([]map[string]interface{}, error) {
	prefix := groupId + "_"
	return s.QueryByPrefix(db.TalkGroupPersonCollection, prefix, limit)
}

// User group list related query methods

// Query user's group list
func (s *DbService) QueryMetaIdContextList(metaId string) (map[string]interface{}, error) {
	return s.QueryByKey(db.TalkMetaIdContextListCollection, metaId)
}

// Query all users' group lists
func (s *DbService) QueryAllMetaIdContextList(limit int) ([]map[string]interface{}, error) {
	return s.QueryAll(db.TalkMetaIdContextListCollection, limit)
}

// Message queue related query methods

// Query chat queue
func (s *DbService) QueryGroupChatQueue(limit int) ([]map[string]interface{}, error) {
	return s.QueryAll(db.TalkGroupChatQueueCollection, limit)
}

// Chat related query methods

// Query group chat message
func (s *DbService) QueryGroupChatPin(pinId string) (map[string]interface{}, error) {
	return s.QueryByKey(db.TalkGroupChatPinCollection, pinId)
}

// Query group chat message (by timestamp range)
func (s *DbService) QueryGroupChatByTimestamp(groupId string, startTime, endTime int64, limit int) ([]map[string]interface{}, error) {
	prefix := groupId + "_"
	return s.QueryByPrefix(db.TalkGroupChatTimestampCollection, prefix, limit)
}

// Query lucky bag message
func (s *DbService) QueryRedEnvelopePin(pinId string) (map[string]interface{}, error) {
	return s.QueryByKey(db.TalkGroupLuckyBagPinCollection, pinId)
}

// Query all lucky bag messages
func (s *DbService) QueryAllRedEnvelopePin(limit int) ([]map[string]interface{}, error) {
	return s.QueryAll(db.TalkGroupLuckyBagPinCollection, limit)
}

// Query grab lucky bag records
func (s *DbService) QueryOpenRedEnvelopePin(pinId string) (map[string]interface{}, error) {
	return s.QueryByKey(db.TalkGroupOpenLuckyBagPinCollection, pinId)
}

// Query all grab lucky bag records
func (s *DbService) QueryAllOpenRedEnvelopePin(limit int) ([]map[string]interface{}, error) {
	return s.QueryAll(db.TalkGroupOpenLuckyBagPinCollection, limit)
}

// Query remaining lucky bag
func (s *DbService) QueryResidueRedEnvelopePin(pinId string) (map[string]interface{}, error) {
	return s.QueryByKey(db.TalkGroupResidueLuckyBagPinCollection, pinId)
}

// Query all remaining lucky bags
func (s *DbService) QueryAllResidueRedEnvelopePin(limit int) ([]map[string]interface{}, error) {
	return s.QueryAll(db.TalkGroupResidueLuckyBagPinCollection, limit)
}

// Statistics related methods

// Get database statistics
func (s *DbService) GetDatabaseStats() (map[string]interface{}, error) {
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
func (s *DbService) GetCollectionCount(collectionName string) (int, error) {
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
func (s *DbService) GetAvailableCollections() []string {
	var collections []string
	for collectionName := range db.Pb {
		collections = append(collections, collectionName)
	}
	return collections
}
