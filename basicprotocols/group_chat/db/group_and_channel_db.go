package db

import (
	"encoding/json"
	"errors"
	"manindexer/basicprotocols/group_chat/models"
	"manindexer/basicprotocols/group_chat/protocols"
	"manindexer/pin"
	"strings"

	"github.com/cockroachdb/pebble"
)

// Group database operations
type GroupDB struct {
	pb *Pebble
}

func NewGroupDB(pb *Pebble) *GroupDB {
	return &GroupDB{pb: pb}
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
		return Pb[TalkGroupInfoCollection].Set(key, data, pebble.Sync)
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
		return Pb[TalkGroupInfoCollection].Set(key, data, pebble.Sync)
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
		} else if strings.ToLower(protocol) == strings.ToLower(protocols.MonitorSimpleGroupJoin) {
			return gdb.processGroupJoin(pin)
		}
	case "modify":
		// Check ParentPath
		parentPath := pin.Path
		parentProtocol := strings.Replace(parentPath, "/protocols/", "", -1)
		if strings.ToLower(parentProtocol) == strings.ToLower(protocols.MonitorSimpleGroupCreate) {
			return gdb.processGroupModify(pin)
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
			err = gdb.addGroupJoinToMetaIdList(pin.CreateMetaId, simpleGroupJoin.GroupId, pin.Id, "join", pin, groupState, simpleGroupJoin.Referrer)
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
				err = gdb.addGroupJoinToMetaIdList(pin.CreateMetaId, simpleGroupJoin.GroupId, pin.Id, "join", pin, groupState, simpleGroupJoin.Referrer)
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
				err = gdb.addGroupJoinToMetaIdList(pin.CreateMetaId, simpleGroupJoin.GroupId, pin.Id, "leave", pin, groupState, simpleGroupJoin.Referrer)
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
	err = gdb.addGroupJoinToMetaIdList(pin.CreateMetaId, group.GroupId, pin.Id, "create", pin, models.RoomStateIn, "")
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
	return Pb[TalkGroupPersonCollection].Set(key2, data, pebble.Sync)
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
	return Pb[TalkGroupPersonCollection].Delete(key2, pebble.Sync)
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

	// Get user's group join list
	joinList, err := gdb.getGroupMetaIdJoinList(metaId)
	if err != nil {
		return nil, err
	}

	// Calculate pagination
	start := cursor
	end := cursor + size

	// Get group info
	for i, item := range joinList.Items {
		if int64(i) < start {
			continue
		}
		if int64(i) >= end {
			break
		}

		// Only return records in the group
		if item.GroupState == models.RoomStateIn {
			group, err := gdb.GetGroupInfoByGroupId(item.JoinPinId)
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
func (gdb *GroupDB) addGroupJoinToMetaIdList(metaId, groupId, pinId, joinType string, pin *pin.PinInscription, groupState models.RoomState, referrer string) error {
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
