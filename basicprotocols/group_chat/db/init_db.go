package db

import (
	"fmt"
	"log"
	"manindexer/common"
	"os"
	"path/filepath"

	"github.com/cockroachdb/pebble"
)

const (
	// 社区相关数据库
	TalkCommunityVersionInfoCollection string = "talk_community_version_info" // key: communityId_pinId 和 pinId_communityId
	TalkCommunityInfoCollection        string = "talk_community_info"         // key: communityId
	TalkCommunityAddressCollection     string = "talk_community_address"      // key: communityId_address 和 address_communityId

	TalkCommunityJoinCollection   string = "talk_community_join"   // key: communityId_pinId
	TalkCommunityPersonCollection string = "talk_community_person" // key: communityId_metaId 和 metaId_communityId

	// 群组相关数据库
	TalkGroupInfoCollection        string = "talk_group_info"         // key: groupId
	TalkGroupVersionInfoCollection string = "talk_group_version_info" // key: groupId_pinId 和 pinId_groupId
	TalkGroupCommunityCollection   string = "talk_group_community"    // key: communityId_groupId

	TalkGroupMetaIdJoinCollection string = "talk_group_metaid_join" // key: metaId_groupId, value: []{joinPinId, joinType, joinTimestamp}
	TalkGroupJoinCollection       string = "talk_group_join"        // key: groupId_pinId 和 pinId_groupId
	TalkGroupPersonCollection     string = "talk_group_person"      // key: groupId_metaId 和 metaId_groupId

	TalkGroupLatestChatCollection   string = "talk_group_latest_chat"    // key: groupId，value: {groupId, timestamp, chatType, content, createAddress}
	TalkMetaIdContextListCollection string = "talk_meta_id_context_list" // key: metaId，value: []{groupId, timestamp, chatType, content, createAddress}

	// 消息队列相关数据库
	TalkGroupChatQueueCollection string = "talk_group_chat_queue" // key: timestamp_pinId，value: chat消息数据
	// TalkGroupChatQueueProcessingCollection string = "talk_group_chat_queue_processing" // key: pinId，value: 处理状态

	// 聊天相关数据库
	TalkGroupChatPinCollection               string = "talk_group_chat_pin"                 // key: pinId
	TalkGroupRedEnvelopePinCollection        string = "talk_group_red_envelope_pin"         // key: pinId
	TalkGroupOpenRedEnvelopePinCollection    string = "talk_group_open_red_envelope_pin"    // key: pinId
	TalkGroupResidueRedEnvelopePinCollection string = "talk_group_residue_red_envelope_pin" // key: pinId
	TalkGroupChatTimestampCollection         string = "talk_group_chat_timestamp"           // key: groupId_timestamp，value: pinId_chatType_timestamp
	TalkGroupChatTimestampOutCollection      string = "talk_group_chat_timestamp_out"       // key: groupId_timestamp，value: pinId_chatType_timestamp

	//私聊
	TalkPrivateChatPinCollection          string = "talk_private_chat_pin"           // key: pinId
	TalkPrivateChatTimestampCollection    string = "talk_private_chat_timestamp"     // key: selfMetaId_otherMetaId_timestamp和otherMetaId_selfMetaId_timestamp，value: pinId_chatType_timestamp
	TalkPrivateChatTimestampOutCollection string = "talk_private_chat_timestamp_out" // key: selfMetaId_otherMetaId_timestamp和otherMetaId_selfMetaId_timestamp，value: pinId_chatType_timestamp
	TalkPrivateChatQueueCollection        string = "talk_private_chat_queue"         // key: timestamp_pinId，value: chat消息数据
)

type Pebble struct{}
type Logger struct{}

func (ml Logger) Infof(format string, args ...interface{}) {}
func (ml Logger) Fatalf(format string, args ...interface{}) {
	log.Println(format, args)
}
func (ml Logger) Errorf(format string, args ...interface{}) {
	log.Println(format, args)
}

var Pb map[string]*pebble.DB

func (pb *Pebble) InitDatabase() error {
	Pb = make(map[string]*pebble.DB, 10)

	// 初始化社区相关数据库
	err := open(TalkCommunityVersionInfoCollection)
	if err != nil {
		return fmt.Errorf("Pebble %s init error: %v", TalkCommunityVersionInfoCollection, err)
	}
	err = open(TalkCommunityInfoCollection)
	if err != nil {
		return fmt.Errorf("Pebble %s init error: %v", TalkCommunityInfoCollection, err)
	}
	err = open(TalkCommunityAddressCollection)
	if err != nil {
		return fmt.Errorf("Pebble %s init error: %v", TalkCommunityAddressCollection, err)
	}
	err = open(TalkCommunityJoinCollection)
	if err != nil {
		return fmt.Errorf("Pebble %s init error: %v", TalkCommunityJoinCollection, err)
	}
	err = open(TalkCommunityPersonCollection)
	if err != nil {
		return fmt.Errorf("Pebble %s init error: %v", TalkCommunityPersonCollection, err)
	}

	// 初始化群组相关数据库
	err = open(TalkGroupInfoCollection)
	if err != nil {
		return fmt.Errorf("Pebble %s init error: %v", TalkGroupInfoCollection, err)
	}
	err = open(TalkGroupVersionInfoCollection)
	if err != nil {
		return fmt.Errorf("Pebble %s init error: %v", TalkGroupVersionInfoCollection, err)
	}
	err = open(TalkGroupCommunityCollection)
	if err != nil {
		return fmt.Errorf("Pebble %s init error: %v", TalkGroupCommunityCollection, err)
	}
	err = open(TalkGroupJoinCollection)
	if err != nil {
		return fmt.Errorf("Pebble %s init error: %v", TalkGroupJoinCollection, err)
	}
	err = open(TalkGroupPersonCollection)
	if err != nil {
		return fmt.Errorf("Pebble %s init error: %v", TalkGroupPersonCollection, err)
	}

	// 初始化群组MetaId加入数据库
	err = open(TalkGroupMetaIdJoinCollection)
	if err != nil {
		return fmt.Errorf("Pebble %s init error: %v", TalkGroupMetaIdJoinCollection, err)
	}

	// 初始化用户群列表数据库
	err = open(TalkMetaIdContextListCollection)
	if err != nil {
		return fmt.Errorf("Pebble %s init error: %v", TalkMetaIdContextListCollection, err)
	}

	// 初始化群组最新聊天数据库
	err = open(TalkGroupLatestChatCollection)
	if err != nil {
		return fmt.Errorf("Pebble %s init error: %v", TalkGroupLatestChatCollection, err)
	}

	// 初始化消息队列数据库
	err = open(TalkGroupChatQueueCollection)
	if err != nil {
		return fmt.Errorf("Pebble %s init error: %v", TalkGroupChatQueueCollection, err)
	}
	// err = open(TalkGroupChatQueueProcessingCollection)
	// if err != nil {
	// 	return fmt.Errorf("Pebble %s init error: %v", TalkGroupChatQueueProcessingCollection, err)
	// }

	// 初始化聊天相关数据库
	err = open(TalkGroupChatPinCollection)
	if err != nil {
		return fmt.Errorf("Pebble %s init error: %v", TalkGroupChatPinCollection, err)
	}
	err = open(TalkGroupRedEnvelopePinCollection)
	if err != nil {
		return fmt.Errorf("Pebble %s init error: %v", TalkGroupRedEnvelopePinCollection, err)
	}
	err = open(TalkGroupOpenRedEnvelopePinCollection)
	if err != nil {
		return fmt.Errorf("Pebble %s init error: %v", TalkGroupOpenRedEnvelopePinCollection, err)
	}
	err = open(TalkGroupResidueRedEnvelopePinCollection)
	if err != nil {
		return fmt.Errorf("Pebble %s init error: %v", TalkGroupResidueRedEnvelopePinCollection, err)
	}
	err = open(TalkGroupChatTimestampCollection)
	if err != nil {
		return fmt.Errorf("Pebble %s init error: %v", TalkGroupChatTimestampCollection, err)
	}
	err = open(TalkGroupChatTimestampOutCollection)
	if err != nil {
		return fmt.Errorf("Pebble %s init error: %v", TalkGroupChatTimestampOutCollection, err)
	}
	return nil
}

func open(dbName string) (err error) {
	lg := Logger{}

	// 设置默认数据库路径
	var dbPath string
	if common.Config != nil && common.Config.Pebble.Dir != "" {
		dbPath = common.Config.Pebble.Dir
	} else {
		// 使用默认路径
		dbPath = "./data"
	}

	dbPath = filepath.Join(dbPath, "group_chat_data")
	err = os.MkdirAll(dbPath, 0755)
	if err != nil {
		return fmt.Errorf("failed to create directory %s: %v", dbPath, err)
	}
	var db *pebble.DB
	db, err = pebble.Open(dbPath+"/"+dbName, &pebble.Options{Logger: lg})
	if err != nil {
		log.Printf("Pebble %s init error: %v\n", dbName, err)
	} else {
		Pb[dbName] = db
		log.Printf("Pebble %s init success\n", dbName)
	}
	return
}

// 关闭所有数据库连接
func (pb *Pebble) CloseAll() {
	for name, db := range Pb {
		if err := db.Close(); err != nil {
			log.Printf("Close database %s error: %v\n", name, err)
		} else {
			log.Printf("Close database %s success\n", name)
		}
	}
}
