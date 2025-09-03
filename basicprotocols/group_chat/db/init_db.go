package db

import (
	"fmt"
	"log"
	"manindexer/basicprotocols/group_chat/models"
	"manindexer/common"
	"os"
	"path/filepath"

	"github.com/cockroachdb/pebble"
)

const (
	// Community related databases
	TalkCommunityVersionInfoCollection string = "talk_community_version_info" // key: communityId_pinId and pinId_communityId
	TalkCommunityInfoCollection        string = "talk_community_info"         // key: communityId
	TalkCommunityAddressCollection     string = "talk_community_address"      // key: communityId_address and address_communityId

	TalkCommunityJoinCollection   string = "talk_community_join"   // key: communityId_pinId
	TalkCommunityPersonCollection string = "talk_community_person" // key: communityId_metaId and metaId_communityId

	// Group related databases
	TalkGroupInfoCollection        string = "talk_group_info"         // key: groupId
	TalkGroupVersionInfoCollection string = "talk_group_version_info" // key: groupId_pinId and pinId_groupId
	TalkGroupCommunityCollection   string = "talk_group_community"    // key: communityId_groupId

	TalkGroupRemoveUserCollection string = "talk_group_remove_user" // key: groupId_pinId and pinId_groupId
	TalkGroupMetaIdJoinCollection string = "talk_group_metaid_join" // key: metaId_groupId, value: []{joinPinId, joinType, joinTimestamp}
	TalkGroupJoinCollection       string = "talk_group_join"        // key: groupId_pinId and pinId_groupId
	TalkGroupPersonCollection     string = "talk_group_person"      // key: groupId_metaId and metaId_groupId

	TalkGroupLatestChatCollection   string = "talk_group_latest_chat"    // key: groupId，value: {groupId, timestamp, chatType, content, createAddress}
	TalkMetaIdContextListCollection string = "talk_meta_id_context_list" // key: metaId，value: []{groupId, timestamp, chatType, content, createAddress}

	// Message queue related databases
	TalkGroupChatQueueCollection            string = "talk_group_chat_queue"              // key: timestamp_pinId，value: chat message data
	TalkGroupOpenLuckyBagQueueCollection    string = "talk_group_open_lucky_bag_queue"    // key: timestamp_pinId，value:
	TalkGroupResidueLuckyBagQueueCollection string = "talk_group_residue_lucky_bag_queue" // key: timestamp_pinId，value:
	// TalkGroupChatQueueProcessingCollection string = "talk_group_chat_queue_processing" // key: pinId，value: processing status

	// Chat related databases
	TalkGroupChatPinCollection             string = "talk_group_chat_pin"               // key: pinId
	TalkGroupLuckyBagPinCollection         string = "talk_group_lucky_bag_pin"          // key: pinId
	TalkGroupOpenLuckyBagPinCollection     string = "talk_group_open_lucky_bag_pin"     // key: pinId
	TalkGroupResidueLuckyBagPinCollection  string = "talk_group_residue_lucky_bag_pin"  // key: pinId
	TalkGroupOpenLuckyBagListCollection    string = "talk_group_open_lucky_bag_list"    // key: luckyBagPinId，value: []{openPinId, groupId, timestamp, createAddress}
	TalkGroupResidueLuckyBagListCollection string = "talk_group_residue_lucky_bag_list" // key: luckyBagPinId，value: []{residuePinId, groupId, timestamp, createAddress}
	TalkGroupChatTimestampCollection       string = "talk_group_chat_timestamp"         // key: groupId_timestamp，value: pinId_chatType_timestamp
	TalkGroupChatTimestampOutCollection    string = "talk_group_chat_timestamp_out"     // key: groupId_timestamp，value: pinId_chatType_timestamp
	TalkGroupChatTimestamp2Collection      string = "talk_group_chat_timestamp_2"       // key: groupId_timestamp+number(6)，value: pinId_chatType_timestamp_number
	TalkGroupChatTimestamp2OutCollection   string = "talk_group_chat_timestamp_out_2"   // key: groupId_timestamp+number(6)，value: pinId_chatType_timestamp_number
	TalkGroupChatIndexCollection           string = "talk_group_chat_index"             // key: groupId_index，value: pinId_chatType_timestamp_isSet

	// Lucky bag residue related databases
	TalkGroupLuckyBagPinPendingCollection              string = "talk_group_lucky_bag_pin_pending"                // key: pinId，value: luckyBagPinId
	TalkGroupLuckyBagPinCompletedCollection            string = "talk_group_lucky_bag_pin_completed"              // key: pinId，value: luckyBagPinId
	TalkGroupLuckyBagPinTimeoutResidueCollection       string = "talk_group_lucky_bag_pin_timeout_residue"        // key: pinId，value: luckyBagPinId
	TalkGroupLuckyBagPinErrPendingCollection           string = "talk_group_lucky_bag_pin_err_pending"            // key: pinId，value: luckyBagPinId
	TalkGroupLuckyBagPinErrTimeoutResidueCollection    string = "talk_group_lucky_bag_pin_err_timeout_residue"    // key: pinId，value: luckyBagPinId
	TalkGroupLuckyBagCodeAddressKeyCollection          string = "talk_group_lucky_bag_code_address_key"           //key: code_address, value: {key, code, luckyBagAddress, timestamp}
	TalkGroupLuckyBagCodeAddressKeyCompletedCollection string = "talk_group_lucky_bag_code_address_key_completed" //key: code_address, value: {key, code, luckyBagAddress, timestamp}

	// Private chat
	TalkPrivateChatPinCollection          string = "talk_private_chat_pin"           // key: pinId
	TalkPrivateChatTimestampCollection    string = "talk_private_chat_timestamp"     // key: from_to_timestamp+number(6) and to_from_timestamp+number(6)，value: pinId_chatType_timestamp_number
	TalkPrivateChatTimestampOutCollection string = "talk_private_chat_timestamp_out" // key: from_to_timestamp+number(6) and to_from_timestamp+number(6)，value: pinId_chatType_timestamp_number
	TalkPrivateChatQueueCollection        string = "talk_private_chat_queue"         // key: timestamp_pinId，value: chat message data
	TalkPrivateChatIndexCollection        string = "talk_private_chat_index"         // key: from_to_index，value: pinId_chatType_timestamp_isSet

	// Version info
	TalkVersionInfoCollection string = "talk_version_info" // key: version，value: version

	// User info
	TalkUserAddressChatPublicKeyCollection string = "talk_user_address_chat_public_key" // key: address，value: []{chatPublicKey, chatPublicKeyId, timestamp, blockHeight, chain}
	TalkUserMetaIdChatPublicKeyCollection  string = "talk_user_metaid_chat_public_key"  // key: metaId，value: []{chatPublicKey, chatPublicKeyId, timestamp, blockHeight, chain}

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

	// Initialize community related databases
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

	// Initialize group related databases
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
	err = open(TalkGroupRemoveUserCollection)
	if err != nil {
		return fmt.Errorf("Pebble %s init error: %v", TalkGroupRemoveUserCollection, err)
	}

	// Initialize group MetaId join database
	err = open(TalkGroupMetaIdJoinCollection)
	if err != nil {
		return fmt.Errorf("Pebble %s init error: %v", TalkGroupMetaIdJoinCollection, err)
	}

	// Initialize user group list database
	err = open(TalkMetaIdContextListCollection)
	if err != nil {
		return fmt.Errorf("Pebble %s init error: %v", TalkMetaIdContextListCollection, err)
	}

	// Initialize group latest chat database
	err = open(TalkGroupLatestChatCollection)
	if err != nil {
		return fmt.Errorf("Pebble %s init error: %v", TalkGroupLatestChatCollection, err)
	}

	// Initialize message queue databases
	err = open(TalkGroupChatQueueCollection)
	if err != nil {
		return fmt.Errorf("Pebble %s init error: %v", TalkGroupChatQueueCollection, err)
	}
	// Initialize grab lucky bag queue database
	err = open(TalkGroupOpenLuckyBagQueueCollection)
	if err != nil {
		return fmt.Errorf("Pebble %s init error: %v", TalkGroupOpenLuckyBagQueueCollection, err)
	}
	err = open(TalkGroupResidueLuckyBagQueueCollection)
	if err != nil {
		return fmt.Errorf("Pebble %s init error: %v", TalkGroupResidueLuckyBagQueueCollection, err)
	}
	// err = open(TalkGroupChatQueueProcessingCollection)
	// if err != nil {
	// 	return fmt.Errorf("Pebble %s init error: %v", TalkGroupChatQueueProcessingCollection, err)
	// }

	// Initialize chat related databases
	err = open(TalkGroupChatPinCollection)
	if err != nil {
		return fmt.Errorf("Pebble %s init error: %v", TalkGroupChatPinCollection, err)
	}
	err = open(TalkGroupLuckyBagPinCollection)
	if err != nil {
		return fmt.Errorf("Pebble %s init error: %v", TalkGroupLuckyBagPinCollection, err)
	}
	err = open(TalkGroupOpenLuckyBagPinCollection)
	if err != nil {
		return fmt.Errorf("Pebble %s init error: %v", TalkGroupOpenLuckyBagPinCollection, err)
	}
	err = open(TalkGroupResidueLuckyBagPinCollection)
	if err != nil {
		return fmt.Errorf("Pebble %s init error: %v", TalkGroupResidueLuckyBagPinCollection, err)
	}
	err = open(TalkGroupOpenLuckyBagListCollection)
	if err != nil {
		return fmt.Errorf("Pebble %s init error: %v", TalkGroupOpenLuckyBagListCollection, err)
	}
	err = open(TalkGroupResidueLuckyBagListCollection)
	if err != nil {
		return fmt.Errorf("Pebble %s init error: %v", TalkGroupResidueLuckyBagListCollection, err)
	}
	err = open(TalkGroupChatTimestampCollection)
	if err != nil {
		return fmt.Errorf("Pebble %s init error: %v", TalkGroupChatTimestampCollection, err)
	}
	err = open(TalkGroupChatTimestampOutCollection)
	if err != nil {
		return fmt.Errorf("Pebble %s init error: %v", TalkGroupChatTimestampOutCollection, err)
	}
	err = open(TalkGroupChatTimestamp2Collection)
	if err != nil {
		return fmt.Errorf("Pebble %s init error: %v", TalkGroupChatTimestamp2Collection, err)
	}
	err = open(TalkGroupChatTimestamp2OutCollection)
	if err != nil {
		return fmt.Errorf("Pebble %s init error: %v", TalkGroupChatTimestamp2OutCollection, err)
	}
	err = open(TalkGroupChatIndexCollection)
	if err != nil {
		return fmt.Errorf("Pebble %s init error: %v", TalkGroupChatIndexCollection, err)
	}

	// Initialize lucky bag residue related databases
	err = open(TalkGroupLuckyBagPinPendingCollection)
	if err != nil {
		return fmt.Errorf("Pebble %s init error: %v", TalkGroupLuckyBagPinPendingCollection, err)
	}
	err = open(TalkGroupLuckyBagPinCompletedCollection)
	if err != nil {
		return fmt.Errorf("Pebble %s init error: %v", TalkGroupLuckyBagPinCompletedCollection, err)
	}
	err = open(TalkGroupLuckyBagPinTimeoutResidueCollection)
	if err != nil {
		return fmt.Errorf("Pebble %s init error: %v", TalkGroupLuckyBagPinTimeoutResidueCollection, err)
	}
	err = open(TalkGroupLuckyBagPinErrPendingCollection)
	if err != nil {
		return fmt.Errorf("Pebble %s init error: %v", TalkGroupLuckyBagPinErrPendingCollection, err)
	}
	err = open(TalkGroupLuckyBagPinErrTimeoutResidueCollection)
	if err != nil {
		return fmt.Errorf("Pebble %s init error: %v", TalkGroupLuckyBagPinErrTimeoutResidueCollection, err)
	}
	err = open(TalkGroupLuckyBagCodeAddressKeyCollection)
	if err != nil {
		return fmt.Errorf("Pebble %s init error: %v", TalkGroupLuckyBagCodeAddressKeyCollection, err)
	}
	err = open(TalkGroupLuckyBagCodeAddressKeyCompletedCollection)
	if err != nil {
		return fmt.Errorf("Pebble %s init error: %v", TalkGroupLuckyBagCodeAddressKeyCompletedCollection, err)
	}

	// Initialize private chat related databases
	err = open(TalkPrivateChatPinCollection)
	if err != nil {
		return fmt.Errorf("Pebble %s init error: %v", TalkPrivateChatPinCollection, err)
	}
	err = open(TalkPrivateChatTimestampCollection)
	if err != nil {
		return fmt.Errorf("Pebble %s init error: %v", TalkPrivateChatTimestampCollection, err)
	}
	err = open(TalkPrivateChatTimestampOutCollection)
	if err != nil {
		return fmt.Errorf("Pebble %s init error: %v", TalkPrivateChatTimestampOutCollection, err)
	}
	err = open(TalkPrivateChatQueueCollection)
	if err != nil {
		return fmt.Errorf("Pebble %s init error: %v", TalkPrivateChatQueueCollection, err)
	}
	err = open(TalkPrivateChatIndexCollection)
	if err != nil {
		return fmt.Errorf("Pebble %s init error: %v", TalkPrivateChatIndexCollection, err)
	}
	// Initialize version info database
	err = open(TalkVersionInfoCollection)
	if err != nil {
		return fmt.Errorf("Pebble %s init error: %v", TalkVersionInfoCollection, err)
	}

	// Initialize user info database
	err = open(TalkUserAddressChatPublicKeyCollection)
	if err != nil {
		return fmt.Errorf("Pebble %s init error: %v", TalkUserAddressChatPublicKeyCollection, err)
	}

	err = open(TalkUserMetaIdChatPublicKeyCollection)
	if err != nil {
		return fmt.Errorf("Pebble %s init error: %v", TalkUserMetaIdChatPublicKeyCollection, err)
	}

	err = CheckAndMigrateDatabase()
	if err != nil {
		return fmt.Errorf("Pebble %s migrate error: %v", TalkVersionInfoCollection, err)
	}

	// Initialize backup system
	err = InitBackupDB()
	if err != nil {
		return fmt.Errorf("Pebble backup system init error: %v", err)
	}

	return nil
}

func open(dbName string) (err error) {
	lg := Logger{}

	// Set default database path
	var dbPath string
	if common.Config != nil && common.Config.Pebble.Dir != "" {
		dbPath = common.Config.Pebble.Dir
	} else {
		// Use default path
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

// Close all database connections
func (pb *Pebble) CloseAll() {
	for name, db := range Pb {
		if err := db.Close(); err != nil {
			log.Printf("Close database %s error: %v\n", name, err)
		} else {
			log.Printf("Close database %s success\n", name)
		}
	}
}

var (
	handleGroupChatItem   func(chat *models.TalkGroupChatV3) error
	handlePrivateChatItem func(chat *models.TalkPrivateChatV3) error
)

func SetHandleGroupChatItem(handle func(chat *models.TalkGroupChatV3) error) {
	handleGroupChatItem = handle
}

func dealGroupChatItem(chat *models.TalkGroupChatV3) error {
	if handleGroupChatItem != nil {
		return handleGroupChatItem(chat)
	}
	return nil
}

func SetHandlePrivateChatItem(handle func(chat *models.TalkPrivateChatV3) error) {
	handlePrivateChatItem = handle
}

func dealPrivateChatItem(chat *models.TalkPrivateChatV3) error {
	if handlePrivateChatItem != nil {
		return handlePrivateChatItem(chat)
	}
	return nil
}
