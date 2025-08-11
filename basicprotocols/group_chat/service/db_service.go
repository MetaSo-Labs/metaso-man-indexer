package service

import (
	"encoding/json"
	"fmt"
	"manindexer/basicprotocols/group_chat/db"

	"github.com/cockroachdb/pebble"
)

type DbService struct{}

// 通用查询方法 - 根据前缀查询数据
func (s *DbService) QueryByPrefix(collectionName, prefix string, limit int) ([]map[string]interface{}, error) {
	dbInstance, exists := db.Pb[collectionName]
	if !exists {
		return nil, fmt.Errorf("数据库 %s 不存在", collectionName)
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

		// 尝试解析JSON
		var jsonData interface{}
		if err := json.Unmarshal(iter.Value(), &jsonData); err != nil {
			// 如果不是JSON，直接使用字符串
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

// 通用查询方法 - 根据key查询单条数据
func (s *DbService) QueryByKey(collectionName, key string) (map[string]interface{}, error) {
	dbInstance, exists := db.Pb[collectionName]
	if !exists {
		return nil, fmt.Errorf("数据库 %s 不存在", collectionName)
	}

	value, closer, err := dbInstance.Get([]byte(key))
	if err != nil {
		return nil, fmt.Errorf("查询失败: %v", err)
	}
	defer closer.Close()

	// 尝试解析JSON
	var jsonData interface{}
	if err := json.Unmarshal(value, &jsonData); err != nil {
		// 如果不是JSON，直接使用字符串
		jsonData = string(value)
	}

	return map[string]interface{}{
		"key":   key,
		"value": jsonData,
	}, nil
}

// 通用查询方法 - 获取所有数据（限制数量）
func (s *DbService) QueryAll(collectionName string, limit int) ([]map[string]interface{}, error) {
	dbInstance, exists := db.Pb[collectionName]
	if !exists {
		return nil, fmt.Errorf("数据库 %s 不存在", collectionName)
	}

	var results []map[string]interface{}
	iter, _ := dbInstance.NewIter(nil)
	defer iter.Close()

	count := 0
	for iter.First(); iter.Valid() && count < limit; iter.Next() {
		key := string(iter.Key())
		value := string(iter.Value())

		// 尝试解析JSON
		var jsonData interface{}
		if err := json.Unmarshal(iter.Value(), &jsonData); err != nil {
			// 如果不是JSON，直接使用字符串
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

// 社区相关查询方法

// 查询社区信息
func (s *DbService) QueryCommunityInfo(communityId string) (map[string]interface{}, error) {
	return s.QueryByKey(db.TalkCommunityInfoCollection, communityId)
}

// 查询所有社区信息
func (s *DbService) QueryAllCommunityInfo(limit int) ([]map[string]interface{}, error) {
	return s.QueryAll(db.TalkCommunityInfoCollection, limit)
}

// 查询社区版本信息
func (s *DbService) QueryCommunityVersionInfo(communityId string, limit int) ([]map[string]interface{}, error) {
	prefix := communityId + "_"
	return s.QueryByPrefix(db.TalkCommunityVersionInfoCollection, prefix, limit)
}

// 查询社区地址信息
func (s *DbService) QueryCommunityAddress(communityId string, limit int) ([]map[string]interface{}, error) {
	prefix := communityId + "_"
	return s.QueryByPrefix(db.TalkCommunityAddressCollection, prefix, limit)
}

// 查询社区加入记录
func (s *DbService) QueryCommunityJoin(communityId string, limit int) ([]map[string]interface{}, error) {
	prefix := communityId + "_"
	return s.QueryByPrefix(db.TalkCommunityJoinCollection, prefix, limit)
}

// 查询社区成员
func (s *DbService) QueryCommunityPerson(communityId string, limit int) ([]map[string]interface{}, error) {
	prefix := communityId + "_"
	return s.QueryByPrefix(db.TalkCommunityPersonCollection, prefix, limit)
}

// 群组相关查询方法

// 查询群组信息
func (s *DbService) QueryGroupInfo(groupId string) (map[string]interface{}, error) {
	return s.QueryByKey(db.TalkGroupInfoCollection, groupId)
}

// 查询所有群组信息
func (s *DbService) QueryAllGroupInfo(limit int) ([]map[string]interface{}, error) {
	return s.QueryAll(db.TalkGroupInfoCollection, limit)
}

// 查询群组版本信息
func (s *DbService) QueryGroupVersionInfo(groupId string, limit int) ([]map[string]interface{}, error) {
	prefix := groupId + "_"
	return s.QueryByPrefix(db.TalkGroupVersionInfoCollection, prefix, limit)
}

// 查询群组社区关联
func (s *DbService) QueryGroupCommunity(communityId string, limit int) ([]map[string]interface{}, error) {
	prefix := communityId + "_"
	return s.QueryByPrefix(db.TalkGroupCommunityCollection, prefix, limit)
}

// 查询群组加入记录
func (s *DbService) QueryGroupJoin(groupId string, limit int) ([]map[string]interface{}, error) {
	prefix := groupId + "_"
	return s.QueryByPrefix(db.TalkGroupJoinCollection, prefix, limit)
}

// 查询群组成员
func (s *DbService) QueryGroupPerson(groupId string, limit int) ([]map[string]interface{}, error) {
	prefix := groupId + "_"
	return s.QueryByPrefix(db.TalkGroupPersonCollection, prefix, limit)
}

// 用户群列表相关查询方法

// 查询用户的群列表
func (s *DbService) QueryMetaIdContextList(metaId string) (map[string]interface{}, error) {
	return s.QueryByKey(db.TalkMetaIdContextListCollection, metaId)
}

// 查询所有用户的群列表
func (s *DbService) QueryAllMetaIdContextList(limit int) ([]map[string]interface{}, error) {
	return s.QueryAll(db.TalkMetaIdContextListCollection, limit)
}

// 消息队列相关查询方法

// 查询聊天队列
func (s *DbService) QueryGroupChatQueue(limit int) ([]map[string]interface{}, error) {
	return s.QueryAll(db.TalkGroupChatQueueCollection, limit)
}

// 聊天相关查询方法

// 查询群聊消息
func (s *DbService) QueryGroupChatPin(pinId string) (map[string]interface{}, error) {
	return s.QueryByKey(db.TalkGroupChatPinCollection, pinId)
}

// 查询群聊消息（按时间戳范围）
func (s *DbService) QueryGroupChatByTimestamp(groupId string, startTime, endTime int64, limit int) ([]map[string]interface{}, error) {
	prefix := groupId + "_"
	return s.QueryByPrefix(db.TalkGroupChatTimestampCollection, prefix, limit)
}

// 查询红包消息
func (s *DbService) QueryRedEnvelopePin(pinId string) (map[string]interface{}, error) {
	return s.QueryByKey(db.TalkGroupRedEnvelopePinCollection, pinId)
}

// 查询所有红包消息
func (s *DbService) QueryAllRedEnvelopePin(limit int) ([]map[string]interface{}, error) {
	return s.QueryAll(db.TalkGroupRedEnvelopePinCollection, limit)
}

// 查询抢红包记录
func (s *DbService) QueryOpenRedEnvelopePin(pinId string) (map[string]interface{}, error) {
	return s.QueryByKey(db.TalkGroupOpenRedEnvelopePinCollection, pinId)
}

// 查询所有抢红包记录
func (s *DbService) QueryAllOpenRedEnvelopePin(limit int) ([]map[string]interface{}, error) {
	return s.QueryAll(db.TalkGroupOpenRedEnvelopePinCollection, limit)
}

// 查询剩余红包
func (s *DbService) QueryResidueRedEnvelopePin(pinId string) (map[string]interface{}, error) {
	return s.QueryByKey(db.TalkGroupResidueRedEnvelopePinCollection, pinId)
}

// 查询所有剩余红包
func (s *DbService) QueryAllResidueRedEnvelopePin(limit int) ([]map[string]interface{}, error) {
	return s.QueryAll(db.TalkGroupResidueRedEnvelopePinCollection, limit)
}

// 统计相关方法

// 获取数据库统计信息
func (s *DbService) GetDatabaseStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	for collectionName, dbInstance := range db.Pb {
		// 获取数据库大小
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

// 获取指定数据库的记录数量
func (s *DbService) GetCollectionCount(collectionName string) (int, error) {
	dbInstance, exists := db.Pb[collectionName]
	if !exists {
		return 0, fmt.Errorf("数据库 %s 不存在", collectionName)
	}

	iter, _ := dbInstance.NewIter(nil)
	defer iter.Close()

	count := 0
	for iter.First(); iter.Valid(); iter.Next() {
		count++
	}

	return count, nil
}

// 获取所有可用的数据库名称
func (s *DbService) GetAvailableCollections() []string {
	var collections []string
	for collectionName := range db.Pb {
		collections = append(collections, collectionName)
	}
	return collections
}
