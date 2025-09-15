package db

import (
	"encoding/json"
	"errors"
	"fmt"
	"manindexer/basicprotocols/group_chat/models"
	"manindexer/basicprotocols/group_chat/protocols"
	"manindexer/basicprotocols/group_chat/service/cache_service"
	"manindexer/pin"
	"strings"
	"sync"
	"time"

	"github.com/cockroachdb/pebble"
)

// Group search cache item
type GroupSearchCacheItem struct {
	GroupId     string `json:"groupId"`     // Group ID
	MemberCount int64  `json:"memberCount"` // Member count
	GroupName   string `json:"groupName"`   // Group name
	GroupIcon   string `json:"groupIcon"`   // Group icon
	PinId       string `json:"pinId"`       // Pin ID
	Timestamp   int64  `json:"timestamp"`   // Timestamp
}

// Group database operations
type GroupDB struct {
	pb *Pebble
	// make reference to chatDB
	cdb *ChatDB

	// Search cache related fields
	searchCache      map[string]*GroupSearchCacheItem // GroupId -> GroupSearchCacheItem
	searchCacheMutex sync.RWMutex
	cacheUpdateChan  chan struct{} // Channel to trigger cache updates
	stopCacheChan    chan struct{} // Channel to stop cache goroutine
}

func NewGroupDB(pb *Pebble, c *ChatDB) *GroupDB {
	gdb := &GroupDB{
		pb:               pb,
		cdb:              c,
		searchCache:      make(map[string]*GroupSearchCacheItem),
		searchCacheMutex: sync.RWMutex{},
		cacheUpdateChan:  make(chan struct{}, 1), // Buffered channel to avoid blocking
		stopCacheChan:    make(chan struct{}),
	}

	// Start cache update goroutine
	go gdb.startCacheUpdateGoroutine()

	return gdb
}

// Save group info
func (gdb *GroupDB) SaveGroupInfo(group *models.TalkGroupModel) error {
	// First get existing group info
	existingGroup, err := gdb.GetGroupInfoByGroupId(group.GroupId)
	if err != nil {
		return err
	}

	// If no existing data, save directly
	if existingGroup == nil {
		data, err := json.Marshal(group)
		if err != nil {
			return err
		}
		key := []byte(group.GroupId)
		err = Pb[TalkGroupInfoCollection].Set(key, data, pebble.Sync)
		if err != nil {
			return err
		}
		cache_service.SetGroupInfoToCache(group.GroupId, group)
		return nil
	}

	// Determine if update is needed
	shouldUpdate := false

	// 1. First check if pinId is the same
	if existingGroup.PinId == group.PinId {
		// pinId is the same, check if blockHeight is different
		if existingGroup.BlockHeight != group.BlockHeight {
			shouldUpdate = true
		}
	} else {
		// pinId is different, compare timestamp
		if group.Timestamp > existingGroup.Timestamp {
			shouldUpdate = true
		}
	}

	// If update is needed, save new data
	if shouldUpdate {
		data, err := json.Marshal(group)
		if err != nil {
			return err
		}
		key := []byte(group.GroupId)
		err = Pb[TalkGroupInfoCollection].Set(key, data, pebble.Sync)
		if err != nil {
			return err
		}
		cache_service.SetGroupInfoToCache(group.GroupId, group)
		return nil
	}

	// No update needed, return directly
	return nil
}

// Get group info by GroupId
func (gdb *GroupDB) GetGroupInfoByGroupId(groupId string) (*models.TalkGroupModel, error) {
	key := []byte(groupId)
	value, closer, err := Pb[TalkGroupInfoCollection].Get(key)
	if err != nil {
		if err == pebble.ErrNotFound {
			return nil, nil
		}
		return nil, err
	}
	defer closer.Close()

	var group models.TalkGroupModel
	err = json.Unmarshal(value, &group)
	if err != nil {
		return nil, err
	}

	return &group, nil
}

// Save group version info
func (gdb *GroupDB) SaveGroupVersionInfo(group *models.TalkGroupModel) error {
	data, err := json.Marshal(group)
	if err != nil {
		return err
	}

	// Use GroupId_PinId as primary key
	key1 := []byte(group.GroupId + "_" + group.PinId)
	err = Pb[TalkGroupVersionInfoCollection].Set(key1, data, pebble.Sync)
	if err != nil {
		return err
	}

	// Use PinId_GroupId as primary key
	key2 := []byte(group.PinId + "_" + group.GroupId)
	return Pb[TalkGroupVersionInfoCollection].Set(key2, data, pebble.Sync)
}

// Get group version info by GroupId and PinId
func (gdb *GroupDB) GetGroupVersionInfoByGroupIdAndPinId(groupId, pinId string) (*models.TalkGroupModel, error) {
	key := []byte(groupId + "_" + pinId)
	value, closer, err := Pb[TalkGroupVersionInfoCollection].Get(key)
	if err != nil {
		if err == pebble.ErrNotFound {
			return nil, nil
		}
		return nil, err
	}
	defer closer.Close()

	var group models.TalkGroupModel
	err = json.Unmarshal(value, &group)
	if err != nil {
		return nil, err
	}

	return &group, nil
}

// Get group version info by PinId and GroupId
func (gdb *GroupDB) GetGroupVersionInfoByPinIdAndGroupId(pinId, groupId string) (*models.TalkGroupModel, error) {
	key := []byte(pinId + "_" + groupId)
	value, closer, err := Pb[TalkGroupVersionInfoCollection].Get(key)
	if err != nil {
		if err == pebble.ErrNotFound {
			return nil, nil
		}
		return nil, err
	}
	defer closer.Close()

	var group models.TalkGroupModel
	err = json.Unmarshal(value, &group)
	if err != nil {
		return nil, err
	}

	return &group, nil
}

// Save group community association
func (gdb *GroupDB) SaveGroupCommunity(group *models.TalkGroupModel) error {
	data, err := json.Marshal(group)
	if err != nil {
		return err
	}

	// Use CommunityId_GroupId as primary key
	key := []byte(group.CommunityId + "_" + group.GroupId)
	return Pb[TalkGroupCommunityCollection].Set(key, data, pebble.Sync)
}

// Delete group community association
func (gdb *GroupDB) DeleteGroupCommunity(communityId, groupId string) error {
	key := []byte(communityId + "_" + groupId)
	return Pb[TalkGroupCommunityCollection].Delete(key, pebble.Sync)
}

// Save group channel association
func (gdb *GroupDB) SaveGroupChannel(channel *models.TalkGroupChannelModel) error {
	data, err := json.Marshal(channel)
	if err != nil {
		return err
	}

	// Use GroupId_ChannelId as primary key
	key := []byte(channel.GroupId + "_" + channel.ChannelId)
	return Pb[TalkGroupChannelCollection].Set(key, data, pebble.Sync)
}

// Get group list by community ID
func (gdb *GroupDB) GetGroupsByCommunityId(communityId string) ([]*models.TalkGroupModel, error) {
	var groups []*models.TalkGroupModel

	// Use prefix query, because key is communityId_groupId format
	prefix := []byte(communityId + "_")
	iter, err := Pb[TalkGroupCommunityCollection].NewIter(&pebble.IterOptions{
		LowerBound: prefix,
		UpperBound: append(prefix, 0xff), // Use 0xff as upper bound to ensure only query keys starting with communityId_
	})
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	for iter.First(); iter.Valid(); iter.Next() {
		var group models.TalkGroupModel
		err := json.Unmarshal(iter.Value(), &group)
		if err != nil {
			continue
		}
		groups = append(groups, &group)
	}

	return groups, nil
}

// Get group list
func (gdb *GroupDB) GetGroupList(page, size int64) ([]*models.TalkGroupModel, error) {
	var groups []*models.TalkGroupModel
	iter, err := Pb[TalkGroupInfoCollection].NewIter(nil)
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	count := int64(0)
	skip := (page - 1) * size

	for iter.First(); iter.Valid(); iter.Next() {
		if count < skip {
			count++
			continue
		}

		if int64(len(groups)) >= size {
			break
		}

		var group models.TalkGroupModel
		err := json.Unmarshal(iter.Value(), &group)
		if err != nil {
			continue
		}
		groups = append(groups, &group)
	}

	return groups, nil
}

// Delete group
func (gdb *GroupDB) DeleteGroup(groupId string) error {
	key := []byte(groupId)
	return Pb[TalkGroupInfoCollection].Delete(key, pebble.Sync)
}

// Main method to process Group
func (gdb *GroupDB) ProcessGroupPin(pin *pin.PinInscription) error {
	switch pin.Operation {
	case "create":
		path := pin.Path
		protocol := strings.Replace(path, "/protocols/", "", -1)
		if strings.ToLower(protocol) == strings.ToLower(protocols.MonitorSimpleGroupCreate) {
			return gdb.processGroupCreate(pin)
		} else if strings.ToLower(protocol) == strings.ToLower(protocols.MonitorSimpleGroupChannel) {
			return gdb.processGroupChannelCreate(pin)
		} else if strings.ToLower(protocol) == strings.ToLower(protocols.MonitorSimpleGroupJoin) {
			return gdb.processGroupJoin(pin)
		} else if strings.ToLower(protocol) == strings.ToLower(protocols.MonitorSimpleGroupRemoveUser) {
			return gdb.processGroupRemoveUser(pin)
		} else if strings.ToLower(protocol) == strings.ToLower(protocols.MonitorSimpleGroupAdmin) {
			return gdb.processGroupAdmin(pin, "")
		} else if strings.ToLower(protocol) == strings.ToLower(protocols.MonitorSimpleGroupBlock) {
			return gdb.processGroupBlock(pin, "")
		} else if strings.ToLower(protocol) == strings.ToLower(protocols.MonitorSimpleGroupWhitelist) {
			return gdb.processGroupWhitelist(pin, "")
		}
	case "modify":
		// Check ParentPath
		parentPath := pin.Path
		parentProtocol := strings.Replace(parentPath, "/protocols/", "", -1)
		if strings.ToLower(parentProtocol) == strings.ToLower(protocols.MonitorSimpleGroupCreate) {
			return gdb.processGroupModify(pin)
		} else if strings.ToLower(parentProtocol) == strings.ToLower(protocols.MonitorSimpleGroupChannel) {
			return gdb.processGroupChannelModify(pin)
		} else if strings.ToLower(parentProtocol) == strings.ToLower(protocols.MonitorSimpleGroupAdmin) {
			return gdb.processGroupAdminModify(pin)
		} else if strings.ToLower(parentProtocol) == strings.ToLower(protocols.MonitorSimpleGroupBlock) {
			return gdb.processGroupBlockModify(pin)
		} else if strings.ToLower(parentProtocol) == strings.ToLower(protocols.MonitorSimpleGroupWhitelist) {
			return gdb.processGroupWhitelistModify(pin)
		}
		return nil
	default:
		return nil // Unknown operation type, skip
	}
	return nil
}

// Process group creation
func (gdb *GroupDB) processGroupCreate(pin *pin.PinInscription) error {
	// Parse protocol data
	var simpleGroupCreate protocols.SimpleGroupCreate
	err := json.Unmarshal(pin.ContentBody, &simpleGroupCreate)
	if err != nil {
		return err
	}

	// Create group model
	group := &models.TalkGroupModel{
		// GroupId:           simpleGroupCreate.GroupId,
		GroupId:           pin.Id,
		CommunityId:       simpleGroupCreate.CommunityId,
		TxId:              pin.Id[:len(pin.Id)-2],
		PinId:             pin.Id,
		RoomName:          simpleGroupCreate.GroupName,
		RoomNote:          simpleGroupCreate.GroupNote,
		RoomIcon:          simpleGroupCreate.GroupIcon,
		RoomType:          getStringValue(simpleGroupCreate.GroupType),
		RoomStatus:        getStringValue(simpleGroupCreate.Status),
		RoomJoinType:      getStringValue(simpleGroupCreate.JoinType),
		ChatSettingType:   getInt64Value(simpleGroupCreate.ChatSettingType),
		DeleteStatus:      getInt64Value(simpleGroupCreate.DeleteStatus),
		CreateUserMetaId:  pin.CreateMetaId,
		CreateUserAddress: pin.CreateAddress,
		Chain:             pin.ChainName,
		Timestamp:         pin.Timestamp,
		BlockHeight:       pin.GenesisHeight,
	}

	// Save to version info table
	err = gdb.SaveGroupVersionInfo(group)
	if err != nil {
		return err
	}

	// Save to basic info table
	err = gdb.SaveGroupInfo(group)
	if err != nil {
		return err
	}

	if group.CommunityId != "" {
		// Save group community association
		err = gdb.SaveGroupCommunity(group)
		if err != nil {
			return err
		}
	}

	// Creator automatically joins group
	err = gdb.processCreatorAutoJoin(group, pin)
	if err != nil {
		return err
	}

	// Initialize group latest chat record
	err = gdb.initGroupLatestChat(group.GroupId, pin)
	if err != nil {
		return err
	}

	// Initialize creator's group list
	err = gdb.initMetaIdContextList(pin.CreateMetaId, group.GroupId, pin)
	if err != nil {
		return err
	}

	// Initialize creator's group join list
	err = gdb.initGroupMetaIdJoinList(pin.CreateMetaId, group.GroupId, pin)
	if err != nil {
		return err
	}

	// Trigger cache update
	gdb.triggerCacheUpdate(group.GroupId)

	return nil
}

// Process group modification
func (gdb *GroupDB) processGroupModify(pin *pin.PinInscription) error {
	// Parse protocol data
	var simpleGroupCreate protocols.SimpleGroupCreate
	err := json.Unmarshal(pin.ContentBody, &simpleGroupCreate)
	if err != nil {
		return err
	}

	// Get existing group info
	existingGroup, err := gdb.GetGroupInfoByGroupId(simpleGroupCreate.GroupId)
	if err != nil {
		return err
	}

	if existingGroup == nil {
		return errors.New("group not found in db, no modify")
		// // If group doesn't exist, process as creation
		// return gdb.processGroupCreate(pin)
	}

	if existingGroup.CreateUserAddress != pin.CreateAddress {
		return errors.New("group creator not match")
	}

	// Update group info
	existingGroup.RoomName = simpleGroupCreate.GroupName
	existingGroup.RoomNote = simpleGroupCreate.GroupNote
	existingGroup.RoomIcon = simpleGroupCreate.GroupIcon
	existingGroup.RoomType = getStringValue(simpleGroupCreate.GroupType)
	existingGroup.RoomStatus = getStringValue(simpleGroupCreate.Status)
	existingGroup.RoomJoinType = getStringValue(simpleGroupCreate.JoinType)
	existingGroup.ChatSettingType = getInt64Value(simpleGroupCreate.ChatSettingType)
	existingGroup.DeleteStatus = getInt64Value(simpleGroupCreate.DeleteStatus)
	existingGroup.TxId = pin.Id[:len(pin.Id)-2] // Remove last two characters
	existingGroup.PinId = pin.Id
	existingGroup.Timestamp = pin.Timestamp
	existingGroup.BlockHeight = pin.GenesisHeight

	// Handle changes in group-community relationship
	oldCommunityId := existingGroup.CommunityId
	newCommunityId := simpleGroupCreate.CommunityId

	// Case 1: No communityId before, now has one
	if oldCommunityId == "" && newCommunityId != "" {
		existingGroup.CommunityId = newCommunityId
		err = gdb.SaveGroupCommunity(existingGroup)
		if err != nil {
			return err
		}
	} else if oldCommunityId != "" && newCommunityId == "" {
		// Case 2: Had communityId before, now doesn't have one
		err = gdb.DeleteGroupCommunity(oldCommunityId, existingGroup.GroupId)
		if err != nil {
			return err
		}
		existingGroup.CommunityId = ""
	} else if oldCommunityId != "" && newCommunityId != "" && oldCommunityId != newCommunityId {
		// Case 3: Had communityId before, now has a different new communityId
		// Delete old association
		err = gdb.DeleteGroupCommunity(oldCommunityId, existingGroup.GroupId)
		if err != nil {
			return err
		}
		// Save new association
		existingGroup.CommunityId = newCommunityId
		err = gdb.SaveGroupCommunity(existingGroup)
		if err != nil {
			return err
		}
	} else if oldCommunityId != "" && newCommunityId != "" && oldCommunityId == newCommunityId {
		// Case 4: Had communityId before, now has the same communityId
		// Update association directly
		err = gdb.SaveGroupCommunity(existingGroup)
		if err != nil {
			return err
		}
	}

	// Save to version info table
	err = gdb.SaveGroupVersionInfo(existingGroup)
	if err != nil {
		return err
	}

	// Save to basic info table
	err = gdb.SaveGroupInfo(existingGroup)
	if err != nil {
		return err
	}

	// Trigger cache update
	gdb.triggerCacheUpdate(existingGroup.GroupId)

	return nil
}

// Process group join
func (gdb *GroupDB) processGroupJoin(pin *pin.PinInscription) error {
	// Parse protocol data
	var simpleGroupJoin protocols.SimpleGroupJoin
	err := json.Unmarshal(pin.ContentBody, &simpleGroupJoin)
	if err != nil {
		return err
	}

	// Determine join state
	var groupState models.RoomState
	if state, ok := simpleGroupJoin.State.(float64); ok {
		if state == 1 {
			groupState = models.RoomStateIn
		} else {
			groupState = models.RoomStateOut
		}
	} else {
		groupState = models.RoomStateIn // Default join
	}

	// Create group join model
	join := &models.TalkGroupJoinModel{
		GroupId:    simpleGroupJoin.GroupId,
		MetaId:     pin.CreateMetaId,
		TxId:       pin.Id[:len(pin.Id)-2], // Remove last two characters
		PinId:      pin.Id,
		Address:    pin.CreateAddress,
		GroupState: groupState,
		Referrer:   simpleGroupJoin.Referrer,
		// IsValid:      true,
		// IsNew:        true,
		BlockHeight:  pin.GenesisHeight,
		Chain:        pin.ChainName,
		ConfirmState: 0,
		Timestamp:    pin.Timestamp,
	}

	// Save group join info (save regardless of state)
	err = gdb.SaveGroupJoin(join)
	if err != nil {
		return err
	}

	// Get existing member info
	existingPerson, err := gdb.GetGroupPersonByGroupIdAndMetaId(simpleGroupJoin.GroupId, pin.MetaId)
	if err != nil {
		return err
	}

	// Handle member state changes
	if existingPerson == nil {
		// New member, decide whether to save based on state
		if groupState == models.RoomStateIn {
			// Create group member info
			person := &models.TalkGroupPerson{
				GroupId:      simpleGroupJoin.GroupId,
				MetaId:       pin.CreateMetaId,
				Address:      pin.CreateAddress,
				UserName:     "",
				UserNickName: "",
				GroupState:   groupState,
				Timestamp:    pin.Timestamp,
				PinId:        pin.Id,
				BlockHeight:  pin.GenesisHeight,
			}

			// Save group member info
			err = gdb.SaveGroupPerson(person)
			if err != nil {
				return err
			}

			// Add to user's group list
			err = gdb.addGroupToMetaIdContextList(pin.CreateMetaId, simpleGroupJoin.GroupId, pin)
			if err != nil {
				return err
			}

			// Add join record to MetaId join list
			err = gdb.addGroupJoinToMetaIdList(pin.CreateMetaId, simpleGroupJoin.GroupId, pin.Id, "join", pin, groupState, simpleGroupJoin.Referrer, "", "")
			if err != nil {
				return err
			}
		}
		// If Out state and new member, no need to save
	} else {
		// Existing member, check state changes
		if existingPerson.GroupState != groupState {
			// State has changed, need to update
			if groupState == models.RoomStateIn {
				// Changed from Out to In, save member info
				existingPerson.GroupState = groupState
				existingPerson.Timestamp = pin.Timestamp
				existingPerson.PinId = pin.Id
				existingPerson.BlockHeight = pin.GenesisHeight
				err = gdb.SaveGroupPerson(existingPerson)
				if err != nil {
					return err
				}

				// Add to user's group list
				err = gdb.addGroupToMetaIdContextList(pin.CreateMetaId, simpleGroupJoin.GroupId, pin)
				if err != nil {
					return err
				}

				// Add join record to MetaId join list
				err = gdb.addGroupJoinToMetaIdList(pin.CreateMetaId, simpleGroupJoin.GroupId, pin.Id, "join", pin, groupState, simpleGroupJoin.Referrer, "", "")
				if err != nil {
					return err
				}
			} else {
				// Changed from In to Out, delete member info
				err = gdb.DeleteGroupPerson(simpleGroupJoin.GroupId, pin.MetaId)
				if err != nil {
					return err
				}

				// Remove from user's group list
				err = gdb.removeGroupFromMetaIdContextList(pin.CreateMetaId, simpleGroupJoin.GroupId)
				if err != nil {
					return err
				}

				// Add leave record to MetaId join list (state is out)
				err = gdb.addGroupJoinToMetaIdList(pin.CreateMetaId, simpleGroupJoin.GroupId, pin.Id, "leave", pin, groupState, simpleGroupJoin.Referrer, "", "")
				if err != nil {
					return err
				}
			}
		}
		// If state hasn't changed, no need to update
	}

	return nil
}

// Process creator auto join group
func (gdb *GroupDB) processCreatorAutoJoin(group *models.TalkGroupModel, pin *pin.PinInscription) error {
	// Create group join record
	join := &models.TalkGroupJoinModel{
		GroupId:      group.GroupId,
		MetaId:       pin.CreateMetaId,
		TxId:         pin.Id[:len(pin.Id)-2], // Remove last two characters
		PinId:        pin.Id,
		Address:      pin.CreateAddress,
		GroupState:   models.RoomStateIn, // Creator joins by default
		Referrer:     "",                 // Creator has no referrer
		BlockHeight:  pin.GenesisHeight,
		Chain:        pin.ChainName,
		ConfirmState: 0,
		Timestamp:    pin.Timestamp,
	}

	// Save group join info
	err := gdb.SaveGroupJoin(join)
	if err != nil {
		return err
	}

	// Create group member info
	person := &models.TalkGroupPerson{
		GroupId:      group.GroupId,
		MetaId:       pin.CreateMetaId,
		Address:      pin.CreateAddress,
		UserName:     "",
		UserNickName: "",
		GroupState:   models.RoomStateIn, // Creator joins by default
		Timestamp:    pin.Timestamp,
		BlockHeight:  pin.GenesisHeight,
	}

	// Save group member info
	err = gdb.SaveGroupPerson(person)
	if err != nil {
		return err
	}

	// Add creator join record to MetaId join list
	err = gdb.addGroupJoinToMetaIdList(pin.CreateMetaId, group.GroupId, pin.Id, "create", pin, models.RoomStateIn, "", "", "")
	if err != nil {
		return err
	}

	return nil
}

// Initialize group latest chat record
func (gdb *GroupDB) initGroupLatestChat(groupId string, pin *pin.PinInscription) error {
	// First get existing latest chat record
	existingLatestChat, err := gdb.getGroupLatestChat(groupId)
	if err != nil {
		return err
	}

	// Create new latest chat record
	newLatestChat := &models.TalkGroupLatestChat{
		GroupId:          groupId,
		Timestamp:        pin.Timestamp,
		ChatType:         models.ChatTypeMsg, // Default to message type
		Content:          "",                 // Initially empty
		CreateAddress:    pin.CreateAddress,
		LastMessagePinId: "", // Initially empty
		MetaId:           pin.CreateMetaId,
		TxId:             pin.Id[:len(pin.Id)-2],
		Protocol:         pin.Path,
		ContentType:      "",
		Encryption:       "",
		ReplyPin:         "",
		Chain:            pin.ChainName,
		BlockHeight:      pin.GenesisHeight,
	}

	// If no existing data, save directly
	if existingLatestChat == nil {
		data, err := json.Marshal(newLatestChat)
		if err != nil {
			return err
		}
		key := []byte(groupId)
		return Pb[TalkGroupLatestChatCollection].Set(key, data, pebble.Sync)
	}

	// Determine if update is needed
	shouldUpdate := false

	// 1. First check if pinId is the same (here LastMessagePinId are all empty, so mainly compare other fields)
	if existingLatestChat.LastMessagePinId == newLatestChat.LastMessagePinId {
		// pinId is the same, check if blockHeight is different
		if existingLatestChat.BlockHeight != newLatestChat.BlockHeight {
			shouldUpdate = true
		}
	} else {
		// pinId is different, compare timestamp
		if newLatestChat.Timestamp > existingLatestChat.Timestamp {
			shouldUpdate = true
		}
	}

	// If update is needed, save new data
	if shouldUpdate {
		data, err := json.Marshal(newLatestChat)
		if err != nil {
			return err
		}
		key := []byte(groupId)
		return Pb[TalkGroupLatestChatCollection].Set(key, data, pebble.Sync)
	}

	// No update needed, return directly
	return nil
}

// Get group latest chat record (GroupDB internal method)
func (gdb *GroupDB) getGroupLatestChat(groupId string) (*models.TalkGroupLatestChat, error) {
	key := []byte(groupId)
	value, closer, err := Pb[TalkGroupLatestChatCollection].Get(key)
	if err != nil {
		if err == pebble.ErrNotFound {
			return nil, nil
		}
		return nil, err
	}
	defer closer.Close()

	var latestChat models.TalkGroupLatestChat
	err = json.Unmarshal(value, &latestChat)
	if err != nil {
		return nil, err
	}

	return &latestChat, nil
}

// Initialize user's group list
func (gdb *GroupDB) initMetaIdContextList(metaId, groupId string, pin *pin.PinInscription) error {
	//add lock
	mutex := GetMetaIdMutex(metaId)
	mutex.Lock()
	defer mutex.Unlock()

	// First get existing group list
	existingList, err := gdb.getMetaIdContextList(metaId)
	if err != nil {
		return err
	}

	// Create new group list item
	newItem := &models.MetaIdContextItem{
		GroupId:          groupId,
		Timestamp:        pin.Timestamp,
		ChatType:         models.ChatTypeMsg, // Default to message type
		Content:          "",                 // Initially empty
		CreateAddress:    pin.CreateAddress,
		LastMessagePinId: "", // Initially empty
		BlockHeight:      pin.GenesisHeight,
	}

	// If no existing data, save directly
	if existingList == nil || len(existingList.Items) == 0 {
		// Create group list
		contextList := &models.MetaIdContextList{
			MetaId: metaId,
			Items:  []*models.MetaIdContextItem{newItem},
		}

		// Serialize data
		data, err := json.Marshal(contextList)
		if err != nil {
			return err
		}

		// Use MetaId as primary key to save to TalkMetaIdContextListCollection
		key := []byte(metaId)
		return Pb[TalkMetaIdContextListCollection].Set(key, data, pebble.Sync)
	}

	// Check if group already exists
	found := false
	shouldUpdate := false
	for i, item := range existingList.Items {
		if item.GroupId == groupId {
			// Determine if update is needed
			if item.LastMessagePinId == newItem.LastMessagePinId {
				// LastMessagePinId is the same, check if blockHeight is different
				if item.BlockHeight != newItem.BlockHeight {
					shouldUpdate = true
				}
			} else {
				// LastMessagePinId is different, compare timestamp
				if newItem.Timestamp > item.Timestamp {
					shouldUpdate = true
				}
			}

			if shouldUpdate {
				// Update existing item
				existingList.Items[i] = newItem
			}
			found = true
			break
		}
	}

	// If group doesn't exist, add new item
	if !found {
		existingList.Items = append(existingList.Items, newItem)
		shouldUpdate = true
	}

	// If update is needed, save data
	if shouldUpdate {
		// Sort by timestamp in reverse order
		gdb.sortContextListByTimestamp(existingList)
		return gdb.saveMetaIdContextList(existingList)
	}

	// No update needed, return directly
	return nil
}

// Initialize user's group join list
func (gdb *GroupDB) initGroupMetaIdJoinList(metaId, groupId string, pin *pin.PinInscription) error {
	// Create initial join record item
	joinItem := &GroupMetaIdJoinItem{
		JoinPinId:     pin.Id,
		JoinType:      "create",
		JoinTimestamp: pin.Timestamp,
		GroupState:    models.RoomStateIn,
		Address:       pin.CreateAddress,
		Referrer:      "",
		BlockHeight:   pin.GenesisHeight,
		Chain:         pin.ChainName,
	}

	// Create join list
	joinList := &GroupMetaIdJoinList{
		MetaId: metaId,
		Items:  []*GroupMetaIdJoinItem{joinItem},
	}

	// Serialize data
	data, err := json.Marshal(joinList)
	if err != nil {
		return err
	}

	// key: metaId_groupId
	key := []byte(metaId + "_" + groupId)
	return Pb[TalkGroupMetaIdJoinCollection].Set(key, data, pebble.Sync)
}

// Update user's group list (add group)
func (gdb *GroupDB) addGroupToMetaIdContextList(metaId, groupId string, pin *pin.PinInscription) error {
	//add lock
	mutex := GetMetaIdMutex(metaId)
	mutex.Lock()
	defer mutex.Unlock()

	// Get existing group list
	existingList, err := gdb.getMetaIdContextList(metaId)
	if err != nil {
		return err
	}

	// Get group latest chat info from TalkGroupLatestChatCollection
	latestChat, err := gdb.getGroupLatestChat(groupId)
	if err != nil {
		return err
	}

	// Create new group list item
	newItem := &models.MetaIdContextItem{
		GroupId:          groupId,
		Timestamp:        pin.Timestamp,
		ChatType:         models.ChatTypeMsg, // Default to message type
		Content:          "",                 // Initially empty
		CreateAddress:    pin.CreateAddress,
		LastMessagePinId: "", // Initially empty
	}

	// If latest chat info is obtained, use its data
	if latestChat != nil {
		newItem.Content = latestChat.Content
		newItem.LastMessagePinId = latestChat.LastMessagePinId
		newItem.Timestamp = latestChat.Timestamp
		newItem.ChatType = latestChat.ChatType
		newItem.BlockHeight = latestChat.BlockHeight
	}

	// Check if group already exists
	found := false
	shouldUpdate := false
	for i, item := range existingList.Items {
		if item.GroupId == groupId {
			// Determine if update is needed
			if item.LastMessagePinId == newItem.LastMessagePinId {
				// LastMessagePinId is the same, check if blockHeight is different
				if item.BlockHeight != newItem.BlockHeight {
					shouldUpdate = true
				}
			} else {
				// LastMessagePinId is different, compare timestamp
				if newItem.Timestamp > item.Timestamp {
					shouldUpdate = true
				}
			}

			if shouldUpdate {
				// Update existing item
				existingList.Items[i] = newItem
			}
			found = true
			break
		}
	}

	// If group doesn't exist, add new item
	if !found {
		existingList.Items = append(existingList.Items, newItem)
		shouldUpdate = true
	}

	// If update is needed, save data
	if shouldUpdate {
		// Sort by timestamp in reverse order
		gdb.sortContextListByTimestamp(existingList)
		return gdb.saveMetaIdContextList(existingList)
	}

	// No update needed, return directly
	return nil
}

// Remove group from user's group list
func (gdb *GroupDB) removeGroupFromMetaIdContextList(metaId, groupId string) error {
	//add lock
	mutex := GetMetaIdMutex(metaId)
	mutex.Lock()
	defer mutex.Unlock()

	// Get existing group list
	existingList, err := gdb.getMetaIdContextList(metaId)
	if err != nil {
		return err
	}

	// Remove specified group
	var newItems []*models.MetaIdContextItem
	for _, item := range existingList.Items {
		if item.GroupId != groupId {
			newItems = append(newItems, item)
		}
	}

	existingList.Items = newItems

	// Save updated group list
	return gdb.saveMetaIdContextList(existingList)
}

// Get user's group list
func (gdb *GroupDB) getMetaIdContextList(metaId string) (*models.MetaIdContextList, error) {
	key := []byte(metaId)
	value, closer, err := Pb[TalkMetaIdContextListCollection].Get(key)
	if err != nil {
		if err == pebble.ErrNotFound {
			return &models.MetaIdContextList{MetaId: metaId, Items: []*models.MetaIdContextItem{}}, nil
		}
		return nil, err
	}
	defer closer.Close()

	var contextList models.MetaIdContextList
	err = json.Unmarshal(value, &contextList)
	if err != nil {
		return nil, err
	}

	return &contextList, nil
}

// Save user's group list
func (gdb *GroupDB) saveMetaIdContextList(contextList *models.MetaIdContextList) error {
	data, err := json.Marshal(contextList)
	if err != nil {
		return err
	}

	key := []byte(contextList.MetaId)
	return Pb[TalkMetaIdContextListCollection].Set(key, data, pebble.Sync)
}

// Sort group list by timestamp in reverse order
func (gdb *GroupDB) sortContextListByTimestamp(contextList *models.MetaIdContextList) {
	// Simple bubble sort, reverse order by timestamp
	for i := 0; i < len(contextList.Items)-1; i++ {
		for j := 0; j < len(contextList.Items)-1-i; j++ {
			if contextList.Items[j].Timestamp < contextList.Items[j+1].Timestamp {
				contextList.Items[j], contextList.Items[j+1] = contextList.Items[j+1], contextList.Items[j]
			}
		}
	}
}

// Helper function: get string value
func getStringValue(v interface{}) string {
	if v == nil {
		return ""
	}
	if str, ok := v.(string); ok {
		return str
	}
	return ""
}

// Helper function: get int64 value
func getInt64Value(v interface{}) int64 {
	if v == nil {
		return 0
	}
	switch val := v.(type) {
	case float64:
		return int64(val)
	case int64:
		return val
	case int:
		return int64(val)
	default:
		return 0
	}
}

// Save group join info
func (gdb *GroupDB) SaveGroupJoin(join *models.TalkGroupJoinModel) error {
	data, err := json.Marshal(join)
	if err != nil {
		return err
	}

	// Use GroupId_PinId as primary key
	key1 := []byte(join.GroupId + "_" + join.PinId)
	err = Pb[TalkGroupJoinCollection].Set(key1, data, pebble.Sync)
	if err != nil {
		return err
	}

	// Use PinId_GroupId as primary key
	key2 := []byte(join.PinId + "_" + join.GroupId)
	return Pb[TalkGroupJoinCollection].Set(key2, data, pebble.Sync)
}

// Get join info by group ID and PinId
func (gdb *GroupDB) GetGroupJoinByGroupIdAndPinId(groupId, pinId string) (*models.TalkGroupJoinModel, error) {
	key := []byte(groupId + "_" + pinId)
	value, closer, err := Pb[TalkGroupJoinCollection].Get(key)
	if err != nil {
		if err == pebble.ErrNotFound {
			return nil, nil
		}
		return nil, err
	}
	defer closer.Close()

	var join models.TalkGroupJoinModel
	err = json.Unmarshal(value, &join)
	if err != nil {
		return nil, err
	}

	return &join, nil
}

// Get group member list
func (gdb *GroupDB) GetGroupMembers(groupId string) ([]*models.TalkGroupJoinModel, error) {
	var members []*models.TalkGroupJoinModel

	// Use prefix query, because key is groupId_pinId format
	prefix := []byte(groupId + "_")
	iter, err := Pb[TalkGroupPersonCollection].NewIter(&pebble.IterOptions{
		// iter, err := Pb[TalkGroupJoinCollection].NewIter(&pebble.IterOptions{
		LowerBound: prefix,
		UpperBound: append(prefix, 0xff), // Use 0xff as upper bound to ensure only query keys starting with groupId_
	})
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	for iter.First(); iter.Valid(); iter.Next() {
		var join models.TalkGroupJoinModel
		err := json.Unmarshal(iter.Value(), &join)
		if err != nil {
			continue
		}
		// Only return members in the group
		if join.GroupState == models.RoomStateIn {
			members = append(members, &join)
		}
	}

	return members, nil
}

// Get group member list from TalkGroupPersonListCollection
func (gdb *GroupDB) GetGroupMembersFromList(groupId string) ([]*models.TalkGroupPerson, error) {
	// Get group person list from TalkGroupPersonListCollection
	personList, err := gdb.GetGroupPersonListFromCollection(groupId)
	if err != nil {
		return nil, err
	}

	if personList == nil {
		return []*models.TalkGroupPerson{}, nil
	}

	// Filter only members who are in the group (GroupState == RoomStateIn)
	var members []*models.TalkGroupPerson
	for _, person := range personList.Persons {
		if person.GroupState == models.RoomStateIn {
			members = append(members, person)
		}
	}

	gdb.sortGroupPersonListByTimestamp(members)

	return members, nil
}

// Get group member list (with pagination support)
func (gdb *GroupDB) GetGroupMembersWithPagination(groupId string, cursor, size int64) ([]*models.TalkGroupJoinModel, int64, error) {
	var members []*models.TalkGroupJoinModel
	var total int64 = 0

	// Use prefix query, because key is groupId_pinId format
	prefix := []byte(groupId + "_")
	iter, err := Pb[TalkGroupPersonCollection].NewIter(&pebble.IterOptions{
		LowerBound: prefix,
		UpperBound: append(prefix, 0xff), // Use 0xff as upper bound to ensure only query keys starting with groupId_
	})
	if err != nil {
		return nil, 0, err
	}
	defer iter.Close()

	// First calculate total count
	for iter.First(); iter.Valid(); iter.Next() {
		var join models.TalkGroupJoinModel
		err := json.Unmarshal(iter.Value(), &join)
		if err != nil {
			continue
		}
		// Only count members in the group
		if join.GroupState == models.RoomStateIn {
			total++
		}
	}

	// Restart iteration for pagination
	iter, err = Pb[TalkGroupPersonCollection].NewIter(&pebble.IterOptions{
		LowerBound: prefix,
		UpperBound: append(prefix, 0xff),
	})
	if err != nil {
		return nil, 0, err
	}
	defer iter.Close()

	count := int64(0)
	skip := cursor

	for iter.First(); iter.Valid(); iter.Next() {
		var join models.TalkGroupJoinModel
		err := json.Unmarshal(iter.Value(), &join)
		if err != nil {
			continue
		}
		// Only return members in the group
		if join.GroupState == models.RoomStateIn {
			if count < skip {
				count++
				continue
			}

			if int64(len(members)) >= size {
				break
			}

			members = append(members, &join)
		}
	}

	return members, total, nil
}

// Get group member count
func (gdb *GroupDB) GetGroupMemberCount(groupId string) (int64, error) {
	var count int64 = 0

	// Use prefix query, because key is groupId_pinId format
	prefix := []byte(groupId + "_")
	iter, err := Pb[TalkGroupPersonCollection].NewIter(&pebble.IterOptions{
		LowerBound: prefix,
		UpperBound: append(prefix, 0xff), // Use 0xff as upper bound to ensure only query keys starting with groupId_
	})
	if err != nil {
		return 0, err
	}
	defer iter.Close()

	for iter.First(); iter.Valid(); iter.Next() {
		var join models.TalkGroupJoinModel
		err := json.Unmarshal(iter.Value(), &join)
		if err != nil {
			continue
		}
		// Only count members in the group
		if join.GroupState == models.RoomStateIn {
			count++
		}
	}

	return count, nil
}

// Get group member count V2 - from TalkGroupPersonListCollection
func (gdb *GroupDB) GetGroupMemberCountFromList(groupId string) (int64, error) {
	// Get group person list from TalkGroupPersonListCollection
	personList, err := gdb.GetGroupPersonListFromCollection(groupId)
	if err != nil {
		return 0, err
	}

	if personList == nil {
		return 0, nil
	}

	// Count only members who are in the group (GroupState == RoomStateIn)
	var count int64 = 0
	for _, person := range personList.Persons {
		if person.GroupState == models.RoomStateIn {
			count++
		}
	}

	return count, nil
}

// Save group member info
func (gdb *GroupDB) SaveGroupPerson(person *models.TalkGroupPerson) error {
	data, err := json.Marshal(person)
	if err != nil {
		return err
	}

	// Use GroupId_MetaId as primary key
	key1 := []byte(person.GroupId + "_" + person.MetaId)
	err = Pb[TalkGroupPersonCollection].Set(key1, data, pebble.Sync)
	if err != nil {
		return err
	}

	// Use MetaId_GroupId as primary key
	key2 := []byte(person.MetaId + "_" + person.GroupId)
	err = Pb[TalkGroupPersonCollection].Set(key2, data, pebble.Sync)
	if err != nil {
		return err
	}

	// Update group person list
	err = gdb.updateGroupPersonList(person.GroupId)
	if err != nil {
		fmt.Println("updateGroupPersonList error", err)
		// continue
	}

	return nil
}

// Delete group member info
func (gdb *GroupDB) DeleteGroupPerson(groupId, metaId string) error {
	// Use GroupId_MetaId as primary key
	key1 := []byte(groupId + "_" + metaId)
	err := Pb[TalkGroupPersonCollection].Delete(key1, pebble.Sync)
	if err != nil {
		return err
	}

	// Use MetaId_GroupId as primary key
	key2 := []byte(metaId + "_" + groupId)
	err = Pb[TalkGroupPersonCollection].Delete(key2, pebble.Sync)
	if err != nil {
		return err
	}

	// Update group person list
	err = gdb.updateGroupPersonList(groupId)
	if err != nil {
		fmt.Println("updateGroupPersonList error", err)
		// continue
	}

	// Delete from cache as well (updateGroupPersonList will update cache with new data)
	// cache_service.DeleteGroupMemberListFromCache(groupId) // Not needed since updateGroupPersonList updates cache

	return nil
}

// Get member info by group ID and MetaId
func (gdb *GroupDB) GetGroupPersonByGroupIdAndMetaId(groupId, metaId string) (*models.TalkGroupPerson, error) {
	// Construct GroupId_MetaId
	key := []byte(groupId + "_" + metaId)
	value, closer, err := Pb[TalkGroupPersonCollection].Get(key)
	if err != nil {
		if err == pebble.ErrNotFound {
			return nil, nil
		}
		return nil, err
	}
	defer closer.Close()

	var person models.TalkGroupPerson
	err = json.Unmarshal(value, &person)
	if err != nil {
		return nil, err
	}

	return &person, nil
}

// Update group person list in TalkGroupPersonListCollection
func (gdb *GroupDB) updateGroupPersonList(groupId string) error {
	// Get all group members for this group
	persons, err := gdb.GetGroupPersonList(groupId)
	if err != nil {
		return err
	}

	// Sort persons by timestamp in descending order (newest first)
	gdb.sortGroupPersonListByTimestamp(persons)

	// Create group person list model
	personList := &models.TalkGroupPersonList{
		GroupId: groupId,
		Persons: persons,
		Total:   int64(len(persons)),
	}

	// Serialize data
	data, err := json.Marshal(personList)
	if err != nil {
		return err
	}

	// Save to TalkGroupPersonListCollection using groupId as key
	key := []byte(groupId)
	err = Pb[TalkGroupPersonListCollection].Set(key, data, pebble.Sync)
	if err != nil {
		return err
	}

	// Update cache with the new data
	cache_service.SetGroupMemberListToCache(groupId, personList)

	return nil
}

// Get group person list from TalkGroupPersonListCollection
func (gdb *GroupDB) GetGroupPersonListFromCollection(groupId string) (*models.TalkGroupPersonList, error) {
	key := []byte(groupId)
	value, closer, err := Pb[TalkGroupPersonListCollection].Get(key)
	if err != nil {
		if err == pebble.ErrNotFound {
			return nil, nil
		}
		return nil, err
	}
	defer closer.Close()

	var personList models.TalkGroupPersonList
	err = json.Unmarshal(value, &personList)
	if err != nil {
		return nil, err
	}

	return &personList, nil
}

// Sort group person list by timestamp in descending order (oldest first)
func (gdb *GroupDB) sortGroupPersonListByTimestamp(persons []*models.TalkGroupPerson) {
	// Simple bubble sort, reverse order by timestamp
	for i := 0; i < len(persons)-1; i++ {
		for j := 0; j < len(persons)-1-i; j++ {
			if persons[j].Timestamp > persons[j+1].Timestamp {
				persons[j], persons[j+1] = persons[j+1], persons[j]
			}
		}
	}
}

// Get member info by MetaId and group ID
func (gdb *GroupDB) GetGroupPersonByMetaIdAndGroupId(metaId, groupId string) (*models.TalkGroupPerson, error) {
	// Construct MetaId_GroupId
	key := []byte(metaId + "_" + groupId)
	value, closer, err := Pb[TalkGroupPersonCollection].Get(key)
	if err != nil {
		if err == pebble.ErrNotFound {
			return nil, nil
		}
		return nil, err
	}
	defer closer.Close()

	var person models.TalkGroupPerson
	err = json.Unmarshal(value, &person)
	if err != nil {
		return nil, err
	}

	return &person, nil
}

// Get group member list
func (gdb *GroupDB) GetGroupPersonList(groupId string) ([]*models.TalkGroupPerson, error) {
	var persons []*models.TalkGroupPerson

	// Use prefix query, because key is groupId_metaId format
	prefix := []byte(groupId + "_")
	iter, err := Pb[TalkGroupPersonCollection].NewIter(&pebble.IterOptions{
		LowerBound: prefix,
		UpperBound: append(prefix, 0xff), // Use 0xff as upper bound to ensure only query keys starting with groupId_
	})
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	for iter.First(); iter.Valid(); iter.Next() {
		var person models.TalkGroupPerson
		err := json.Unmarshal(iter.Value(), &person)
		if err != nil {
			continue
		}
		// Only return members in the group
		if person.GroupState == models.RoomStateIn {
			persons = append(persons, &person)
		}
	}

	return persons, nil
}

// Get group list joined by user based on MetaId
func (gdb *GroupDB) GetGroupListByMetaId(metaId string, cursor, size int64) ([]*models.TalkGroupModel, error) {
	var groups []*models.TalkGroupModel

	// Use prefix query to get all groups joined by this MetaId
	// Key format is MetaId_GroupId in TalkGroupPersonCollection
	prefix := []byte(metaId + "_")
	iter, err := Pb[TalkGroupPersonCollection].NewIter(&pebble.IterOptions{
		LowerBound: prefix,
		UpperBound: append(prefix, 0xff), // Use 0xff as upper bound to ensure only query keys starting with metaId_
	})
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	count := int64(0)
	skip := cursor

	for iter.First(); iter.Valid(); iter.Next() {
		var person models.TalkGroupPerson
		err := json.Unmarshal(iter.Value(), &person)
		if err != nil {
			continue
		}

		// Only return groups where user is in the group
		if person.GroupState == models.RoomStateIn {
			if count < skip {
				count++
				continue
			}

			if int64(len(groups)) >= size {
				break
			}

			// Get group info by GroupId
			group, err := gdb.GetGroupInfoByGroupId(person.GroupId)
			if err != nil {
				continue
			}
			if group != nil {
				groups = append(groups, group)
			}
		}
	}

	return groups, nil
}

// Group MetaId join record item
type GroupMetaIdJoinItem struct {
	JoinPinId     string           `json:"joinPinId"`     // Join PinId
	JoinType      string           `json:"joinType"`      // Join type: create, join
	JoinTimestamp int64            `json:"joinTimestamp"` // Join timestamp
	GroupState    models.RoomState `json:"groupState"`    // Group state: 1-in, -1-out
	Address       string           `json:"address"`       // User address
	Referrer      string           `json:"referrer"`      // Referrer
	BlockHeight   int64            `json:"blockHeight"`   // Block height
	Chain         string           `json:"chain"`         // Chain type
	ByMetaId      string           `json:"byMetaId"`      // By MetaId
	ByAddress     string           `json:"byAddress"`     // By Address
}

// Group MetaId join list
type GroupMetaIdJoinList struct {
	MetaId string                 `json:"metaId"` // User MetaId
	Items  []*GroupMetaIdJoinItem `json:"items"`  // Join record list
}

// Get user's group join list
func (gdb *GroupDB) getGroupMetaIdJoinList(metaId string) (*GroupMetaIdJoinList, error) {
	key := []byte(metaId)
	value, closer, err := Pb[TalkGroupMetaIdJoinCollection].Get(key)
	if err != nil {
		if err == pebble.ErrNotFound {
			return &GroupMetaIdJoinList{MetaId: metaId, Items: []*GroupMetaIdJoinItem{}}, nil
		}
		return nil, err
	}
	defer closer.Close()

	var joinList GroupMetaIdJoinList
	err = json.Unmarshal(value, &joinList)
	if err != nil {
		return nil, err
	}

	return &joinList, nil
}

// Save user's group join list
func (gdb *GroupDB) saveGroupMetaIdJoinList(joinList *GroupMetaIdJoinList, groupId string) error {
	data, err := json.Marshal(joinList)
	if err != nil {
		return err
	}

	//key: metaId_groupId
	key := []byte(joinList.MetaId + "_" + groupId)
	return Pb[TalkGroupMetaIdJoinCollection].Set(key, data, pebble.Sync)
}

// Add group join record to user's join list
func (gdb *GroupDB) addGroupJoinToMetaIdList(
	metaId, groupId, pinId, joinType string,
	pin *pin.PinInscription, groupState models.RoomState, referrer string,
	byMetaId, byAddress string) error {
	// Add lock for TalkGroupMetaIdJoinCollection operations
	mutex := GetGroupMetaIdJoinMutex(metaId)
	mutex.Lock()
	defer mutex.Unlock()

	// Get existing join list
	existingList, err := gdb.getGroupMetaIdJoinList(metaId)
	if err != nil {
		return err
	}

	// Create new join record item
	newItem := &GroupMetaIdJoinItem{
		JoinPinId:     pinId,
		JoinType:      joinType,
		JoinTimestamp: pin.Timestamp,
		GroupState:    groupState,
		Address:       pin.CreateAddress,
		Referrer:      referrer,
		BlockHeight:   pin.GenesisHeight,
		Chain:         pin.ChainName,
		ByMetaId:      byMetaId,
		ByAddress:     byAddress,
	}

	// Check if group record already exists
	found := false
	for i, item := range existingList.Items {
		// Determine if already exists by JoinPinId (because each join has different PinId)
		if item.JoinPinId == pinId {
			// Update existing item
			existingList.Items[i] = newItem
			found = true
			break
		}
	}

	// If not found, add new item
	if !found {
		existingList.Items = append(existingList.Items, newItem)
	}

	// Sort by timestamp in reverse order
	gdb.sortGroupJoinListByTimestamp(existingList)

	// Save updated join list
	return gdb.saveGroupMetaIdJoinList(existingList, groupId)
}

// Sort group join list by timestamp in reverse order
func (gdb *GroupDB) sortGroupJoinListByTimestamp(joinList *GroupMetaIdJoinList) {
	// Simple bubble sort, reverse order by timestamp
	for i := 0; i < len(joinList.Items)-1; i++ {
		for j := 0; j < len(joinList.Items)-1-i; j++ {
			if joinList.Items[j].JoinTimestamp < joinList.Items[j+1].JoinTimestamp {
				joinList.Items[j], joinList.Items[j+1] = joinList.Items[j+1], joinList.Items[j]
			}
		}
	}
}

// Process group remove user
func (gdb *GroupDB) processGroupRemoveUser(pin *pin.PinInscription) error {
	// Parse protocol data
	var simpleGroupRemoveUser protocols.SimpleGroupRemoveUser
	err := json.Unmarshal(pin.ContentBody, &simpleGroupRemoveUser)
	if err != nil {
		return err
	}

	// Get group info to verify creator
	group, err := gdb.GetGroupInfoByGroupId(simpleGroupRemoveUser.GroupId)
	if err != nil {
		return err
	}
	if group == nil {
		return errors.New("group not found")
	}

	// Verify that the user initiating the removal is the group creator
	if group.CreateUserAddress != pin.CreateAddress {
		return errors.New("only group creator can remove users")
	}

	if simpleGroupRemoveUser.RemoveMetaid == group.CreateUserMetaId {
		return errors.New("cannot remove group creator")
	}

	// Get the address of the user being removed
	removeUserAddress, _ := gdb.getUserAddressByMetaId(simpleGroupRemoveUser.RemoveMetaid)

	// Create remove user model
	removeUser := &models.TalkGroupRemoveUserModel{
		GroupId:         simpleGroupRemoveUser.GroupId,
		RemoveMetaId:    simpleGroupRemoveUser.RemoveMetaid,
		RemoveAddress:   removeUserAddress,
		RemoveReason:    simpleGroupRemoveUser.Reason,
		RemoveByMetaId:  pin.CreateMetaId,
		RemoveByAddress: pin.CreateAddress,
		TxId:            pin.Id[:len(pin.Id)-2], // Remove last two characters
		PinId:           pin.Id,
		Chain:           pin.ChainName,
		BlockHeight:     pin.GenesisHeight,
		ConfirmState:    0,
		Timestamp:       pin.Timestamp,
	}

	// Save remove user record
	err = gdb.SaveGroupRemoveUser(removeUser)
	if err != nil {
		return err
	}

	// Remove user from group member list
	err = gdb.DeleteGroupPerson(simpleGroupRemoveUser.GroupId, simpleGroupRemoveUser.RemoveMetaid)
	if err != nil {
		return err
	}

	// Remove from user's group list
	err = gdb.removeGroupFromMetaIdContextList(simpleGroupRemoveUser.RemoveMetaid, simpleGroupRemoveUser.GroupId)
	if err != nil {
		return err
	}

	// Add remove record to user's join list
	err = gdb.addGroupJoinToMetaIdList(
		simpleGroupRemoveUser.RemoveMetaid, simpleGroupRemoveUser.GroupId, pin.Id, "remove",
		pin, models.RoomStateOut, "",
		pin.CreateMetaId, pin.CreateAddress,
	)
	if err != nil {
		return err
	}

	// Generate system message for user removal
	err = gdb.generateRemoveUserSystemMessage(
		simpleGroupRemoveUser.GroupId,
		simpleGroupRemoveUser.RemoveMetaid,
		removeUserAddress,
		simpleGroupRemoveUser.Reason,
		pin.CreateMetaId,
		pin.CreateAddress,
		pin,
	)
	if err != nil {
		return err
	}

	return nil
}

// Save group remove user record
func (gdb *GroupDB) SaveGroupRemoveUser(removeUser *models.TalkGroupRemoveUserModel) error {
	data, err := json.Marshal(removeUser)
	if err != nil {
		return err
	}

	// Use GroupId_PinId as primary key
	key1 := []byte(removeUser.GroupId + "_" + removeUser.PinId)
	err = Pb[TalkGroupRemoveUserCollection].Set(key1, data, pebble.Sync)
	if err != nil {
		return err
	}

	// Use PinId_GroupId as primary key
	key2 := []byte(removeUser.PinId + "_" + removeUser.GroupId)
	return Pb[TalkGroupRemoveUserCollection].Set(key2, data, pebble.Sync)
}

// Get remove user record by GroupId and PinId
func (gdb *GroupDB) GetGroupRemoveUserByGroupIdAndPinId(groupId, pinId string) (*models.TalkGroupRemoveUserModel, error) {
	key := []byte(groupId + "_" + pinId)
	value, closer, err := Pb[TalkGroupRemoveUserCollection].Get(key)
	if err != nil {
		if err == pebble.ErrNotFound {
			return nil, nil
		}
		return nil, err
	}
	defer closer.Close()

	var removeUser models.TalkGroupRemoveUserModel
	err = json.Unmarshal(value, &removeUser)
	if err != nil {
		return nil, err
	}

	return &removeUser, nil
}

// Get remove user record by PinId and GroupId
func (gdb *GroupDB) GetGroupRemoveUserByPinIdAndGroupId(pinId, groupId string) (*models.TalkGroupRemoveUserModel, error) {
	key := []byte(pinId + "_" + groupId)
	value, closer, err := Pb[TalkGroupRemoveUserCollection].Get(key)
	if err != nil {
		if err == pebble.ErrNotFound {
			return nil, nil
		}
		return nil, err
	}
	defer closer.Close()

	var removeUser models.TalkGroupRemoveUserModel
	err = json.Unmarshal(value, &removeUser)
	if err != nil {
		return nil, err
	}

	return &removeUser, nil
}

// Get user address by MetaId (helper method)
func (gdb *GroupDB) getUserAddressByMetaId(metaId string) (string, error) {
	// This is a helper method to get user address by MetaId
	// In a real implementation, you might want to query a user database
	// For now, we'll return an empty string as placeholder
	// You can implement this based on your user management system
	return "", nil
}

// Generate system message for user removal
func (gdb *GroupDB) generateRemoveUserSystemMessage(
	groupId string,
	removeMetaId string,
	removeAddress string,
	removeReason string,
	removeByMetaId string,
	removeByAddress string,
	pin *pin.PinInscription) error {

	// make reference to chatDB
	cdb := gdb.cdb
	if cdb == nil {
		return errors.New("chatDB is nil")
	}

	// Create a unique PinId for the system message
	systemPinId := pin.Id

	// Create system message content
	content := fmt.Sprintf("User {%s} was removed from the group", removeMetaId)
	if removeReason != "" {
		content += fmt.Sprintf(" (Reason: [%s])", removeReason)
	}

	// Create system chat message
	systemChat := &models.TalkGroupChatV3{
		GroupId:     groupId,
		TxId:        pin.Id[:len(pin.Id)-2], // Remove last two characters
		PinId:       systemPinId,
		MetaId:      removeByMetaId,  // Use the remover's MetaId as sender
		Address:     removeByAddress, // Use the remover's address as sender
		Protocol:    pin.Path,
		Content:     content,
		ContentType: "text/plain",
		Encryption:  "",
		ChatType:    models.ChatTypeRemove,    // Use the new remove type
		InsideIndex: models.ChatInsideIndexIn, // System message is always "in"
		ReplyPin:    "",
		Timestamp:   pin.Timestamp,
		Chain:       pin.ChainName,
		BlockHeight: pin.GenesisHeight,
		Index:       -1, // Default index
	}

	// Save system message to database
	err := cdb.SaveChat(systemChat)
	if err != nil {
		return err
	}

	// Save timestamp index with state
	isGoEnqueue, err := cdb.SaveChatTimestampWithState(systemChat)
	if err != nil {
		return err
	}

	// Enqueue message for asynchronous processing
	if isGoEnqueue {
		err = cdb.EnqueueChatMessage(systemChat)
		if err != nil {
			return err
		}
	}

	return nil
}

// Process group admin setting
func (gdb *GroupDB) processGroupAdmin(pin *pin.PinInscription, operation string) error {
	// Parse protocol data
	var simpleGroupAdmin protocols.SimpleGroupAdmin
	err := json.Unmarshal(pin.ContentBody, &simpleGroupAdmin)
	if err != nil {
		return err
	}

	// Get group info to verify creator
	group, err := gdb.GetGroupInfoByGroupId(simpleGroupAdmin.GroupId)
	if err != nil {
		return err
	}
	if group == nil {
		return errors.New("group not found")
	}

	// Verify that the user initiating the admin setting is the group creator
	if group.CreateUserAddress != pin.CreateAddress {
		return errors.New("only group creator can set admins")
	}

	if operation == "" {
		operation = "create"
	}
	// Add admin record to group admin list
	err = gdb.addGroupAdminToGroupList(simpleGroupAdmin.GroupId, pin.Id, operation, pin, simpleGroupAdmin.Admins)
	if err != nil {
		return err
	}

	return nil
}

// Process group admin modify
func (gdb *GroupDB) processGroupAdminModify(pin *pin.PinInscription) error {
	return gdb.processGroupAdmin(pin, "modify")
}

// Process group block setting
func (gdb *GroupDB) processGroupBlock(pin *pin.PinInscription, operation string) error {
	// Parse protocol data
	var simpleGroupBlock protocols.SimpleGroupBlock
	err := json.Unmarshal(pin.ContentBody, &simpleGroupBlock)
	if err != nil {
		return err
	}

	// Get group info to verify creator or admin
	group, err := gdb.GetGroupInfoByGroupId(simpleGroupBlock.GroupId)
	if err != nil {
		return err
	}
	if group == nil {
		return errors.New("group not found")
	}

	// Verify that the user initiating the block setting is the group creator or admin
	if group.CreateUserAddress != pin.CreateAddress {
		// Check if user is admin
		isAdmin, err := gdb.IsUserAdmin(simpleGroupBlock.GroupId, pin.CreateMetaId, pin.Timestamp)
		if err != nil {
			return err
		}
		if !isAdmin {
			return errors.New("only group creator or admin can set block list")
		}
	}

	if operation == "" {
		operation = "create"
	}
	// Add block record to group block list
	err = gdb.addGroupBlockToGroupList(simpleGroupBlock.GroupId, pin.Id, operation, pin, simpleGroupBlock.Users)
	if err != nil {
		return err
	}

	return nil
}

// Process group block modify
func (gdb *GroupDB) processGroupBlockModify(pin *pin.PinInscription) error {
	return gdb.processGroupBlock(pin, "modify")
}

// Process group whitelist setting
func (gdb *GroupDB) processGroupWhitelist(pin *pin.PinInscription, operation string) error {
	// Parse protocol data
	var simpleGroupWhitelist protocols.SimpleGroupWhitelist
	err := json.Unmarshal(pin.ContentBody, &simpleGroupWhitelist)
	if err != nil {
		return err
	}

	// Get group info to verify creator or admin
	group, err := gdb.GetGroupInfoByGroupId(simpleGroupWhitelist.GroupId)
	if err != nil {
		return err
	}
	if group == nil {
		return errors.New("group not found")
	}

	// Verify that the user initiating the whitelist setting is the group creator or admin
	if group.CreateUserAddress != pin.CreateAddress {
		// Check if user is admin
		isAdmin, err := gdb.IsUserAdmin(simpleGroupWhitelist.GroupId, pin.CreateMetaId, pin.Timestamp)
		if err != nil {
			return err
		}
		if !isAdmin {
			return errors.New("only group creator or admin can set whitelist")
		}
	}

	if operation == "" {
		operation = "create"
	}
	// Add whitelist record to group whitelist list
	err = gdb.addGroupWhitelistToGroupList(simpleGroupWhitelist.GroupId, pin.Id, operation, pin, simpleGroupWhitelist.Users)
	if err != nil {
		return err
	}

	return nil
}

// Process group whitelist modify
func (gdb *GroupDB) processGroupWhitelistModify(pin *pin.PinInscription) error {
	return gdb.processGroupWhitelist(pin, "modify")
}

// Get group admin list
func (gdb *GroupDB) getGroupAdminList(groupId string) (*models.GroupAdminList, error) {
	key := []byte(groupId)
	value, closer, err := Pb[TalkGroupAdminCollection].Get(key)
	if err != nil {
		if err == pebble.ErrNotFound {
			return &models.GroupAdminList{GroupId: groupId, Items: []*models.GroupAdminItem{}}, nil
		}
		return nil, err
	}
	defer closer.Close()

	var adminList models.GroupAdminList
	err = json.Unmarshal(value, &adminList)
	if err != nil {
		return nil, err
	}

	return &adminList, nil
}

// Save group admin list
func (gdb *GroupDB) saveGroupAdminList(adminList *models.GroupAdminList) error {
	data, err := json.Marshal(adminList)
	if err != nil {
		return err
	}

	key := []byte(adminList.GroupId)
	return Pb[TalkGroupAdminCollection].Set(key, data, pebble.Sync)
}

// Add group admin record to group admin list
func (gdb *GroupDB) addGroupAdminToGroupList(
	groupId, pinId, adminType string,
	pin *pin.PinInscription, admins []string) error {
	// Add lock for TalkGroupAdminCollection operations
	mutex := GetGroupAdminMutex(groupId)
	mutex.Lock()
	defer mutex.Unlock()

	// Get existing admin list
	existingList, err := gdb.getGroupAdminList(groupId)
	if err != nil {
		return err
	}

	// Create new admin record item
	newItem := &models.GroupAdminItem{
		AdminPinId:     pinId,
		AdminType:      adminType,
		AdminTimestamp: pin.Timestamp,
		Admins:         admins,
		SetByMetaId:    pin.CreateMetaId,
		SetByAddress:   pin.CreateAddress,
		BlockHeight:    pin.GenesisHeight,
		Chain:          pin.ChainName,
	}

	// Check if admin record already exists
	found := false
	for i, item := range existingList.Items {
		// Determine if already exists by AdminPinId (because each admin has different PinId)
		if item.AdminPinId == pinId {
			// Update existing item
			existingList.Items[i] = newItem
			found = true
			break
		}
	}

	// If not found, add new item
	if !found {
		existingList.Items = append(existingList.Items, newItem)
	}

	// Sort by timestamp in reverse order
	gdb.sortGroupAdminListByTimestamp(existingList)

	// Save updated admin list
	return gdb.saveGroupAdminList(existingList)
}

// Sort group admin list by timestamp in ascending order
func (gdb *GroupDB) sortGroupAdminListByTimestamp(adminList *models.GroupAdminList) {
	// Simple bubble sort, ascending order by timestamp
	for i := 0; i < len(adminList.Items)-1; i++ {
		for j := 0; j < len(adminList.Items)-1-i; j++ {
			if adminList.Items[j].AdminTimestamp > adminList.Items[j+1].AdminTimestamp {
				adminList.Items[j], adminList.Items[j+1] = adminList.Items[j+1], adminList.Items[j]
			}
		}
	}
}

// Get group block list
func (gdb *GroupDB) getGroupBlockList(groupId string) (*models.GroupBlockList, error) {
	key := []byte(groupId)
	value, closer, err := Pb[TalkGroupBlockCollection].Get(key)
	if err != nil {
		if err == pebble.ErrNotFound {
			return &models.GroupBlockList{GroupId: groupId, Items: []*models.GroupBlockItem{}}, nil
		}
		return nil, err
	}
	defer closer.Close()

	var blockList models.GroupBlockList
	err = json.Unmarshal(value, &blockList)
	if err != nil {
		return nil, err
	}

	return &blockList, nil
}

// Save group block list
func (gdb *GroupDB) saveGroupBlockList(blockList *models.GroupBlockList) error {
	data, err := json.Marshal(blockList)
	if err != nil {
		return err
	}

	key := []byte(blockList.GroupId)
	return Pb[TalkGroupBlockCollection].Set(key, data, pebble.Sync)
}

// Add group block record to group block list
func (gdb *GroupDB) addGroupBlockToGroupList(
	groupId, pinId, blockType string,
	pin *pin.PinInscription, blockedUsers []string) error {
	// Add lock for TalkGroupBlockCollection operations
	mutex := GetGroupBlockMutex(groupId)
	mutex.Lock()
	defer mutex.Unlock()

	// Get existing block list
	existingList, err := gdb.getGroupBlockList(groupId)
	if err != nil {
		return err
	}

	// Create new block record item
	newItem := &models.GroupBlockItem{
		BlockPinId:     pinId,
		BlockType:      blockType,
		BlockTimestamp: pin.Timestamp,
		BlockedUsers:   blockedUsers,
		SetByMetaId:    pin.CreateMetaId,
		SetByAddress:   pin.CreateAddress,
		BlockHeight:    pin.GenesisHeight,
		Chain:          pin.ChainName,
	}

	// Check if block record already exists
	found := false
	for i, item := range existingList.Items {
		// Determine if already exists by BlockPinId (because each block has different PinId)
		if item.BlockPinId == pinId {
			// Update existing item
			existingList.Items[i] = newItem
			found = true
			break
		}
	}

	// If not found, add new item
	if !found {
		existingList.Items = append(existingList.Items, newItem)
	}

	// Sort by timestamp in reverse order
	gdb.sortGroupBlockListByTimestamp(existingList)

	// Save updated block list
	return gdb.saveGroupBlockList(existingList)
}

// Sort group block list by timestamp in ascending order
func (gdb *GroupDB) sortGroupBlockListByTimestamp(blockList *models.GroupBlockList) {
	// Simple bubble sort, ascending order by timestamp
	for i := 0; i < len(blockList.Items)-1; i++ {
		for j := 0; j < len(blockList.Items)-1-i; j++ {
			if blockList.Items[j].BlockTimestamp > blockList.Items[j+1].BlockTimestamp {
				blockList.Items[j], blockList.Items[j+1] = blockList.Items[j+1], blockList.Items[j]
			}
		}
	}
}

// Get group whitelist list
func (gdb *GroupDB) getGroupWhitelistList(groupId string) (*models.GroupWhitelistList, error) {
	key := []byte(groupId)
	value, closer, err := Pb[TalkGroupWhitelistCollection].Get(key)
	if err != nil {
		if err == pebble.ErrNotFound {
			return &models.GroupWhitelistList{GroupId: groupId, Items: []*models.GroupWhitelistItem{}}, nil
		}
		return nil, err
	}
	defer closer.Close()

	var whitelistList models.GroupWhitelistList
	err = json.Unmarshal(value, &whitelistList)
	if err != nil {
		return nil, err
	}

	return &whitelistList, nil
}

// Save group whitelist list
func (gdb *GroupDB) saveGroupWhitelistList(whitelistList *models.GroupWhitelistList) error {
	data, err := json.Marshal(whitelistList)
	if err != nil {
		return err
	}

	key := []byte(whitelistList.GroupId)
	return Pb[TalkGroupWhitelistCollection].Set(key, data, pebble.Sync)
}

// Add group whitelist record to group whitelist list
func (gdb *GroupDB) addGroupWhitelistToGroupList(
	groupId, pinId, whitelistType string,
	pin *pin.PinInscription, whitelistUsers []string) error {
	// Add lock for TalkGroupWhitelistCollection operations
	mutex := GetGroupWhitelistMutex(groupId)
	mutex.Lock()
	defer mutex.Unlock()

	// Get existing whitelist list
	existingList, err := gdb.getGroupWhitelistList(groupId)
	if err != nil {
		return err
	}

	// Create new whitelist record item
	newItem := &models.GroupWhitelistItem{
		WhitelistPinId:     pinId,
		WhitelistType:      whitelistType,
		WhitelistTimestamp: pin.Timestamp,
		WhitelistUsers:     whitelistUsers,
		SetByMetaId:        pin.CreateMetaId,
		SetByAddress:       pin.CreateAddress,
		BlockHeight:        pin.GenesisHeight,
		Chain:              pin.ChainName,
	}

	// Check if whitelist record already exists
	found := false
	for i, item := range existingList.Items {
		// Determine if already exists by WhitelistPinId (because each whitelist has different PinId)
		if item.WhitelistPinId == pinId {
			// Update existing item
			existingList.Items[i] = newItem
			found = true
			break
		}
	}

	// If not found, add new item
	if !found {
		existingList.Items = append(existingList.Items, newItem)
	}

	// Sort by timestamp in reverse order
	gdb.sortGroupWhitelistListByTimestamp(existingList)

	// Save updated whitelist list
	return gdb.saveGroupWhitelistList(existingList)
}

// Sort group whitelist list by timestamp in ascending order
func (gdb *GroupDB) sortGroupWhitelistListByTimestamp(whitelistList *models.GroupWhitelistList) {
	// Simple bubble sort, ascending order by timestamp
	for i := 0; i < len(whitelistList.Items)-1; i++ {
		for j := 0; j < len(whitelistList.Items)-1-i; j++ {
			if whitelistList.Items[j].WhitelistTimestamp > whitelistList.Items[j+1].WhitelistTimestamp {
				whitelistList.Items[j], whitelistList.Items[j+1] = whitelistList.Items[j+1], whitelistList.Items[j]
			}
		}
	}
}

// Check if user is admin of the group at specific timestamp
func (gdb *GroupDB) IsUserAdmin(groupId, metaId string, pinTimestamp int64) (bool, error) {
	// Get group admin list
	adminList, err := gdb.getGroupAdminList(groupId)
	if err != nil {
		return false, err
	}
	if adminList == nil || len(adminList.Items) == 0 {
		return false, nil
	}

	// Data is already sorted by timestamp in ascending order
	// Find the admin record that was effective at the given timestamp
	var effectiveAdminItem *models.GroupAdminItem
	for _, item := range adminList.Items {
		// Find the latest admin record that was set before or at the pinTimestamp
		if item.AdminTimestamp <= pinTimestamp {
			effectiveAdminItem = item
		} else {
			// Since data is sorted by timestamp, we can break here
			break
		}
	}

	// If no effective admin record found, user is not admin
	if effectiveAdminItem == nil {
		return false, nil
	}

	// Check if metaId is in the effective admin list
	for _, adminMetaId := range effectiveAdminItem.Admins {
		if adminMetaId == metaId {
			return true, nil
		}
	}

	return false, nil
}

// Check if user is blocked in the group at specific timestamp
func (gdb *GroupDB) IsUserBlock(groupId, metaId string, pinTimestamp int64) (bool, error) {
	// Get group block list
	blockList, err := gdb.getGroupBlockList(groupId)
	if err != nil {
		return false, err
	}
	if blockList == nil || len(blockList.Items) == 0 {
		return false, nil
	}

	// Data is already sorted by timestamp in ascending order
	// Find the block record that was effective at the given timestamp
	var effectiveBlockItem *models.GroupBlockItem
	for _, item := range blockList.Items {
		// Find the latest block record that was set before or at the pinTimestamp
		if item.BlockTimestamp <= pinTimestamp {
			effectiveBlockItem = item
		} else {
			// Since data is sorted by timestamp, we can break here
			break
		}
	}

	// If no effective block record found, user is not blocked
	if effectiveBlockItem == nil {
		return false, nil
	}

	// Check if metaId is in the effective block list
	for _, blockedMetaId := range effectiveBlockItem.BlockedUsers {
		if blockedMetaId == metaId {
			return true, nil
		}
	}

	return false, nil
}

// Check if user is whitelisted in the group at specific timestamp
func (gdb *GroupDB) IsUserWhitelist(groupId, metaId string, pinTimestamp int64) (bool, error) {
	// Get group whitelist list
	whitelistList, err := gdb.getGroupWhitelistList(groupId)
	if err != nil {
		return false, err
	}
	if whitelistList == nil || len(whitelistList.Items) == 0 {
		return false, nil
	}

	// Data is already sorted by timestamp in ascending order
	// Find the whitelist record that was effective at the given timestamp
	var effectiveWhitelistItem *models.GroupWhitelistItem
	for _, item := range whitelistList.Items {
		// Find the latest whitelist record that was set before or at the pinTimestamp
		if item.WhitelistTimestamp <= pinTimestamp {
			effectiveWhitelistItem = item
		} else {
			// Since data is sorted by timestamp, we can break here
			break
		}
	}

	// If no effective whitelist record found, user is not whitelisted
	if effectiveWhitelistItem == nil {
		return false, nil
	}

	// Check if metaId is in the effective whitelist
	for _, whitelistMetaId := range effectiveWhitelistItem.WhitelistUsers {
		if whitelistMetaId == metaId {
			return true, nil
		}
	}

	return false, nil
}

// startCacheUpdateGoroutine starts the cache update goroutine
func (gdb *GroupDB) startCacheUpdateGoroutine() {
	// Execute initial cache update immediately on startup
	fmt.Printf("[GroupDB] Starting initial cache update...\n")
	gdb.updateSearchCache()
	fmt.Printf("[GroupDB] Initial cache update completed\n")

	ticker := time.NewTicker(30 * time.Second) // Update every 30 seconds
	defer ticker.Stop()

	// Flag to track if update is in progress
	isUpdating := false

	for {
		select {
		case <-ticker.C:

			if GlobalIsStop {
				fmt.Printf("[GroupDB] Cache update goroutine is stopped, skipping this cycle\n")
				continue
			}

			// Periodic update - only if not already updating
			if !isUpdating {
				isUpdating = true
				go func() {
					gdb.updateSearchCache()
					isUpdating = false
				}()
			} else {
				fmt.Printf("[GroupDB] Skipping periodic update - previous update still in progress\n")
			}
		case <-gdb.cacheUpdateChan:

			if GlobalIsStop {
				fmt.Printf("[GroupDB] Cache update goroutine is stopped, skipping this cycle\n")
				continue
			}

			// Manual update triggered - only if not already updating
			if !isUpdating {
				isUpdating = true
				go func() {
					gdb.updateSearchCache()
					isUpdating = false
				}()
			} else {
				fmt.Printf("[GroupDB] Skipping manual update - previous update still in progress\n")
			}
		case <-gdb.stopCacheChan:
			// Stop goroutine
			return
		}
	}
}

// updateSearchCache updates the search cache from TalkGroupInfoCollection
func (gdb *GroupDB) updateSearchCache() {
	t := time.Now().UnixMilli()
	// Create a new temporary cache first
	newCache := make(map[string]*GroupSearchCacheItem)

	// Iterate through all groups in TalkGroupInfoCollection
	iter, err := Pb[TalkGroupInfoCollection].NewIter(nil)
	if err != nil {
		fmt.Printf("[GroupDB] Failed to create iterator for search cache update: %v\n", err)
		return
	}
	defer iter.Close()

	for iter.First(); iter.Valid(); iter.Next() {
		var group models.TalkGroupModel
		err := json.Unmarshal(iter.Value(), &group)
		if err != nil {
			continue
		}

		// Add to new cache
		cacheItem := &GroupSearchCacheItem{
			GroupId:   group.GroupId,
			GroupName: group.RoomName,
			GroupIcon: group.RoomIcon,
			PinId:     group.PinId,
			Timestamp: group.Timestamp,
		}
		newCache[group.GroupId] = cacheItem
	}

	// Only after successfully building the new cache, replace the old one
	gdb.searchCacheMutex.Lock()
	gdb.searchCache = newCache
	gdb.searchCacheMutex.Unlock()

	fmt.Printf("[GroupDB] Search cache updated, total groups: %d, time: %d\n", len(newCache), time.Now().UnixMilli()-t)
}

// triggerCacheUpdate triggers a manual cache update for a specific group
func (gdb *GroupDB) triggerCacheUpdate(groupId string) {
	// Get the latest group info from database
	group, err := gdb.GetGroupInfoByGroupId(groupId)
	if err != nil || group == nil {
		return
	}

	// Update only this specific group in cache
	gdb.searchCacheMutex.Lock()
	defer gdb.searchCacheMutex.Unlock()

	// Create or update the cache item for this group
	cacheItem := &GroupSearchCacheItem{
		GroupId:   group.GroupId,
		GroupName: group.RoomName,
		GroupIcon: group.RoomIcon,
		PinId:     group.PinId,
		Timestamp: group.Timestamp,
	}
	gdb.searchCache[groupId] = cacheItem

	fmt.Printf("[GroupDB] Cache updated for group: %s\n", groupId)
}

// SearchGroups searches groups by name or ID using fuzzy search
func (gdb *GroupDB) SearchGroups(query string, limit int, memberCountMin int) ([]*GroupSearchCacheItem, error) {
	if query == "" {
		return nil, errors.New("search query cannot be empty")
	}

	if limit <= 0 {
		limit = 20 // Default limit
	}

	t := time.Now().UnixMilli()
	// Create a temporary copy of the cache to avoid blocking during search
	gdb.searchCacheMutex.RLock()
	cacheCopy := make(map[string]*GroupSearchCacheItem, len(gdb.searchCache))
	for k, v := range gdb.searchCache {
		cacheCopy[k] = v
	}
	gdb.searchCacheMutex.RUnlock()
	fmt.Printf("[GroupDB] Search groups from cache, time: %d\n", time.Now().UnixMilli()-t)

	var results []*GroupSearchCacheItem
	queryLower := strings.ToLower(query)

	// Search through the temporary copy
	for _, item := range cacheCopy {

		if memberCountMin > 0 {
			memberCount, err := gdb.GetGroupMemberCountFromList(item.GroupId)
			if err != nil {
				continue
			}
			if memberCount <= int64(memberCountMin) {
				continue
			}
			item.MemberCount = memberCount
		}
		// Check if group name or ID contains the query (case-insensitive)
		if strings.Contains(strings.ToLower(item.GroupName), queryLower) ||
			strings.Contains(strings.ToLower(item.GroupId), queryLower) {
			results = append(results, item)

			// Check limit
			if len(results) >= limit {
				break
			}
		}
	}

	// Sort results by timestamp (newest first)
	// Simple bubble sort for small result sets
	for i := 0; i < len(results)-1; i++ {
		for j := 0; j < len(results)-1-i; j++ {
			if results[j].Timestamp < results[j+1].Timestamp {
				results[j], results[j+1] = results[j+1], results[j]
			}
		}
	}

	return results, nil
}

// GetSearchCacheStats returns search cache statistics
func (gdb *GroupDB) GetSearchCacheStats() map[string]interface{} {
	gdb.searchCacheMutex.RLock()
	totalGroups := len(gdb.searchCache)
	gdb.searchCacheMutex.RUnlock()

	return map[string]interface{}{
		"totalGroups": totalGroups,
		"lastUpdate":  time.Now().Unix(),
		"searchCache": gdb.searchCache,
	}
}

// StopCacheGoroutine stops the cache update goroutine
func (gdb *GroupDB) StopCacheGoroutine() {
	close(gdb.stopCacheChan)
}

// ==================== Channel Related Methods ====================

// Save channel info
func (gdb *GroupDB) SaveChannelInfo(channel *models.TalkGroupChannelModel) error {
	// First get existing channel info
	existingChannel, err := gdb.GetChannelInfoByChannelId(channel.ChannelId)
	if err != nil {
		return err
	}

	// If no existing data, save directly
	if existingChannel == nil {
		data, err := json.Marshal(channel)
		if err != nil {
			return err
		}
		key := []byte(channel.ChannelId)
		err = Pb[TalkGroupChannelInfoCollection].Set(key, data, pebble.Sync)
		if err != nil {
			return err
		}
		return nil
	}

	// Determine if update is needed
	shouldUpdate := false

	// 1. First check if pinId is the same
	if existingChannel.PinId == channel.PinId {
		// pinId is the same, check if blockHeight is different
		if existingChannel.BlockHeight != channel.BlockHeight {
			shouldUpdate = true
		}
	} else {
		// pinId is different, compare timestamp
		if channel.Timestamp > existingChannel.Timestamp {
			shouldUpdate = true
		}
	}

	// If update is needed, save new data
	if shouldUpdate {
		data, err := json.Marshal(channel)
		if err != nil {
			return err
		}
		key := []byte(channel.ChannelId)
		err = Pb[TalkGroupChannelInfoCollection].Set(key, data, pebble.Sync)
		if err != nil {
			return err
		}
		return nil
	}

	// No update needed, return directly
	return nil
}

// Get channel info by ChannelId
func (gdb *GroupDB) GetChannelInfoByChannelId(channelId string) (*models.TalkGroupChannelModel, error) {
	key := []byte(channelId)
	value, closer, err := Pb[TalkGroupChannelInfoCollection].Get(key)
	if err != nil {
		if err == pebble.ErrNotFound {
			return nil, nil
		}
		return nil, err
	}
	defer closer.Close()

	var channel models.TalkGroupChannelModel
	err = json.Unmarshal(value, &channel)
	if err != nil {
		return nil, err
	}

	return &channel, nil
}

// Save channel version info
func (gdb *GroupDB) SaveChannelVersionInfo(channel *models.TalkGroupChannelModel) error {
	data, err := json.Marshal(channel)
	if err != nil {
		return err
	}

	// Use ChannelId_PinId as primary key
	key1 := []byte(channel.ChannelId + "_" + channel.PinId)
	err = Pb[TalkGroupChannelVersionInfoCollection].Set(key1, data, pebble.Sync)
	if err != nil {
		return err
	}

	// Use PinId_ChannelId as primary key
	key2 := []byte(channel.PinId + "_" + channel.ChannelId)
	return Pb[TalkGroupChannelVersionInfoCollection].Set(key2, data, pebble.Sync)
}

// Get channel version info by ChannelId and PinId
func (gdb *GroupDB) GetChannelVersionInfoByChannelIdAndPinId(channelId, pinId string) (*models.TalkGroupChannelModel, error) {
	key := []byte(channelId + "_" + pinId)
	value, closer, err := Pb[TalkGroupChannelVersionInfoCollection].Get(key)
	if err != nil {
		if err == pebble.ErrNotFound {
			return nil, nil
		}
		return nil, err
	}
	defer closer.Close()

	var channel models.TalkGroupChannelModel
	err = json.Unmarshal(value, &channel)
	if err != nil {
		return nil, err
	}

	return &channel, nil
}

// Get channel version info by PinId and ChannelId
func (gdb *GroupDB) GetChannelVersionInfoByPinIdAndChannelId(pinId, channelId string) (*models.TalkGroupChannelModel, error) {
	key := []byte(pinId + "_" + channelId)
	value, closer, err := Pb[TalkGroupChannelVersionInfoCollection].Get(key)
	if err != nil {
		if err == pebble.ErrNotFound {
			return nil, nil
		}
		return nil, err
	}
	defer closer.Close()

	var channel models.TalkGroupChannelModel
	err = json.Unmarshal(value, &channel)
	if err != nil {
		return nil, err
	}

	return &channel, nil
}

// Get channels by group ID from TalkGroupChannelCollection
func (gdb *GroupDB) GetChannelsByGroupId(groupId string) ([]*models.TalkGroupChannelModel, error) {
	var channels []*models.TalkGroupChannelModel

	// Use prefix query, because key is groupId_channelId format
	prefix := []byte(groupId + "_")
	iter, err := Pb[TalkGroupChannelCollection].NewIter(&pebble.IterOptions{
		LowerBound: prefix,
		UpperBound: append(prefix, 0xff), // Use 0xff as upper bound to ensure only query keys starting with groupId_
	})
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	for iter.First(); iter.Valid(); iter.Next() {
		var channel models.TalkGroupChannelModel
		err := json.Unmarshal(iter.Value(), &channel)
		if err != nil {
			continue
		}
		channels = append(channels, &channel)
	}

	return channels, nil
}

// Delete channel
func (gdb *GroupDB) DeleteChannel(channelId string) error {
	key := []byte(channelId)
	return Pb[TalkGroupChannelInfoCollection].Delete(key, pebble.Sync)
}

// Process channel creation
func (gdb *GroupDB) processGroupChannelCreate(pin *pin.PinInscription) error {
	// Parse protocol data
	var simpleGroupChannel protocols.SimpleGroupChannel
	err := json.Unmarshal(pin.ContentBody, &simpleGroupChannel)
	if err != nil {
		return err
	}

	// Verify that the group exists and user has permission
	group, err := gdb.GetGroupInfoByGroupId(simpleGroupChannel.GroupId)
	if err != nil {
		return err
	}
	if group == nil {
		return errors.New("group not found")
	}

	// Verify that the user initiating the channel creation is the group creator or admin
	if group.CreateUserAddress != pin.CreateAddress {
		// Check if user is admin
		// isAdmin, err := gdb.IsUserAdmin(simpleGroupChannel.GroupId, pin.CreateMetaId, pin.Timestamp)
		// if err != nil {
		// 	return err
		// }
		// if !isAdmin {
		// 	return errors.New("only group creator or admin can create channels")
		// }
		return errors.New("only group creator can create channels")
	}

	// Create channel model
	channel := &models.TalkGroupChannelModel{
		ChannelId:         pin.Id,
		GroupId:           simpleGroupChannel.GroupId,
		TxId:              pin.Id[:len(pin.Id)-2],
		PinId:             pin.Id,
		ChannelName:       simpleGroupChannel.ChannelName,
		ChannelNote:       simpleGroupChannel.ChannelNote,
		ChannelIcon:       simpleGroupChannel.ChannelIcon,
		ChannelType:       simpleGroupChannel.ChannelType,
		CreateUserMetaId:  pin.CreateMetaId,
		CreateUserAddress: pin.CreateAddress,
		Chain:             pin.ChainName,
		DeleteStatus:      0, // Default to normal
		Timestamp:         pin.Timestamp,
		BlockHeight:       pin.GenesisHeight,
	}

	// Save to version info table
	err = gdb.SaveChannelVersionInfo(channel)
	if err != nil {
		return err
	}

	// Save to basic info table
	err = gdb.SaveChannelInfo(channel)
	if err != nil {
		return err
	}

	// Save group channel association
	err = gdb.SaveGroupChannel(channel)
	if err != nil {
		return err
	}

	return nil
}

// Process channel modification
func (gdb *GroupDB) processGroupChannelModify(pin *pin.PinInscription) error {
	// Parse protocol data
	var simpleGroupChannel protocols.SimpleGroupChannel
	err := json.Unmarshal(pin.ContentBody, &simpleGroupChannel)
	if err != nil {
		return err
	}

	// Get existing channel info
	existingChannel, err := gdb.GetChannelInfoByChannelId(simpleGroupChannel.ChannelId)
	if err != nil {
		return err
	}

	if existingChannel == nil {
		return errors.New("channel not found in db, no modify")
	}

	// Verify that the user initiating the modification is the channel creator or group admin
	if existingChannel.CreateUserAddress != pin.CreateAddress {
		// Check if user is group admin
		// isAdmin, err := gdb.IsUserAdmin(existingChannel.GroupId, pin.CreateMetaId, pin.Timestamp)
		// if err != nil {
		// 	return err
		// }
		// if !isAdmin {
		// 	return errors.New("only channel creator or group admin can modify channels")
		// }
		return errors.New("only channel creator can modify channels")
	}

	// Update channel info
	existingChannel.ChannelName = simpleGroupChannel.ChannelName
	existingChannel.ChannelNote = simpleGroupChannel.ChannelNote
	existingChannel.ChannelIcon = simpleGroupChannel.ChannelIcon
	existingChannel.ChannelType = simpleGroupChannel.ChannelType
	existingChannel.TxId = pin.Id[:len(pin.Id)-2] // Remove last two characters
	existingChannel.PinId = pin.Id
	existingChannel.Timestamp = pin.Timestamp
	existingChannel.BlockHeight = pin.GenesisHeight

	// Save to version info table
	err = gdb.SaveChannelVersionInfo(existingChannel)
	if err != nil {
		return err
	}

	// Save to basic info table
	err = gdb.SaveChannelInfo(existingChannel)
	if err != nil {
		return err
	}

	return nil
}
