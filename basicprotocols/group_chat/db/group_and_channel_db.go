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

// 群组数据库操作
type GroupDB struct {
	pb *Pebble
}

func NewGroupDB(pb *Pebble) *GroupDB {
	return &GroupDB{pb: pb}
}

// 保存群组信息
func (gdb *GroupDB) SaveGroupInfo(group *models.TalkGroupModel) error {
	// 先获取现有的群组信息
	existingGroup, err := gdb.GetGroupInfoByGroupId(group.GroupId)
	if err != nil {
		return err
	}

	// 如果不存在现有数据，直接保存
	if existingGroup == nil {
		data, err := json.Marshal(group)
		if err != nil {
			return err
		}
		key := []byte(group.GroupId)
		return Pb[TalkGroupInfoCollection].Set(key, data, pebble.Sync)
	}

	// 判断是否需要更新
	shouldUpdate := false

	// 1. 先判断pinId是否一样
	if existingGroup.PinId == group.PinId {
		// pinId一样，检查blockHeight是否不一样
		if existingGroup.BlockHeight != group.BlockHeight {
			shouldUpdate = true
		}
	} else {
		// pinId不一样，比较timestamp
		if group.Timestamp > existingGroup.Timestamp {
			shouldUpdate = true
		}
	}

	// 如果需要更新，则保存新数据
	if shouldUpdate {
		data, err := json.Marshal(group)
		if err != nil {
			return err
		}
		key := []byte(group.GroupId)
		return Pb[TalkGroupInfoCollection].Set(key, data, pebble.Sync)
	}

	// 不需要更新，直接返回
	return nil
}

// 根据GroupId获取群组信息
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

// 保存群组版本信息
func (gdb *GroupDB) SaveGroupVersionInfo(group *models.TalkGroupModel) error {
	data, err := json.Marshal(group)
	if err != nil {
		return err
	}

	// 使用 GroupId_PinId 作为主键
	key1 := []byte(group.GroupId + "_" + group.PinId)
	err = Pb[TalkGroupVersionInfoCollection].Set(key1, data, pebble.Sync)
	if err != nil {
		return err
	}

	// 使用 PinId_GroupId 作为主键
	key2 := []byte(group.PinId + "_" + group.GroupId)
	return Pb[TalkGroupVersionInfoCollection].Set(key2, data, pebble.Sync)
}

// 根据GroupId和PinId获取群组版本信息
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

// 根据PinId和GroupId获取群组版本信息
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

// 保存群组社区关联
func (gdb *GroupDB) SaveGroupCommunity(group *models.TalkGroupModel) error {
	data, err := json.Marshal(group)
	if err != nil {
		return err
	}

	// 使用 CommunityId_GroupId 作为主键
	key := []byte(group.CommunityId + "_" + group.GroupId)
	return Pb[TalkGroupCommunityCollection].Set(key, data, pebble.Sync)
}

// 删除群组社区关联
func (gdb *GroupDB) DeleteGroupCommunity(communityId, groupId string) error {
	key := []byte(communityId + "_" + groupId)
	return Pb[TalkGroupCommunityCollection].Delete(key, pebble.Sync)
}

// 根据社区ID获取群组列表
func (gdb *GroupDB) GetGroupsByCommunityId(communityId string) ([]*models.TalkGroupModel, error) {
	var groups []*models.TalkGroupModel
	iter, err := Pb[TalkGroupCommunityCollection].NewIter(nil)
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
		if group.CommunityId == communityId {
			groups = append(groups, &group)
		}
	}

	return groups, nil
}

// 获取群组列表
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

// 删除群组
func (gdb *GroupDB) DeleteGroup(groupId string) error {
	key := []byte(groupId)
	return Pb[TalkGroupInfoCollection].Delete(key, pebble.Sync)
}

// 总的处理 Group 方法
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
		//检查ParentPath
		parentPath := pin.Path
		parentProtocol := strings.Replace(parentPath, "/protocols/", "", -1)
		if strings.ToLower(parentProtocol) == strings.ToLower(protocols.MonitorSimpleGroupCreate) {
			return gdb.processGroupModify(pin)
		}
		return nil
	default:
		return nil // 未知操作类型，跳过
	}
	return nil
}

// 处理群组创建
func (gdb *GroupDB) processGroupCreate(pin *pin.PinInscription) error {
	// 解析协议数据
	var simpleGroupCreate protocols.SimpleGroupCreate
	err := json.Unmarshal(pin.ContentBody, &simpleGroupCreate)
	if err != nil {
		return err
	}

	// 创建群组模型
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

	// 保存到版本信息表
	err = gdb.SaveGroupVersionInfo(group)
	if err != nil {
		return err
	}

	// 保存到基本信息表
	err = gdb.SaveGroupInfo(group)
	if err != nil {
		return err
	}

	if group.CommunityId != "" {
		// 保存群组社区关联
		err = gdb.SaveGroupCommunity(group)
		if err != nil {
			return err
		}
	}

	// 创建者自动加入群组
	err = gdb.processCreatorAutoJoin(group, pin)
	if err != nil {
		return err
	}

	// 初始化群组最新聊天记录
	err = gdb.initGroupLatestChat(group.GroupId, pin)
	if err != nil {
		return err
	}

	// 初始化创建者的群列表
	err = gdb.initMetaIdContextList(pin.CreateMetaId, group.GroupId, pin)
	if err != nil {
		return err
	}

	// 初始化创建者的群组加入列表
	err = gdb.initGroupMetaIdJoinList(pin.CreateMetaId, group.GroupId, pin)
	if err != nil {
		return err
	}

	return nil
}

// 处理群组修改
func (gdb *GroupDB) processGroupModify(pin *pin.PinInscription) error {
	// 解析协议数据
	var simpleGroupCreate protocols.SimpleGroupCreate
	err := json.Unmarshal(pin.ContentBody, &simpleGroupCreate)
	if err != nil {
		return err
	}

	// 获取现有群组信息
	existingGroup, err := gdb.GetGroupInfoByGroupId(simpleGroupCreate.GroupId)
	if err != nil {
		return err
	}

	if existingGroup == nil {
		return errors.New("group not found in db, no modify")
		// // 如果群组不存在，按创建处理
		// return gdb.processGroupCreate(pin)
	}

	if existingGroup.CreateUserAddress != pin.CreateAddress {
		return errors.New("group creator not match")
	}

	// 更新群组信息
	existingGroup.RoomName = simpleGroupCreate.GroupName
	existingGroup.RoomNote = simpleGroupCreate.GroupNote
	existingGroup.RoomType = getStringValue(simpleGroupCreate.GroupType)
	existingGroup.RoomStatus = getStringValue(simpleGroupCreate.Status)
	existingGroup.RoomJoinType = getStringValue(simpleGroupCreate.JoinType)
	existingGroup.ChatSettingType = getInt64Value(simpleGroupCreate.ChatSettingType)
	existingGroup.DeleteStatus = getInt64Value(simpleGroupCreate.DeleteStatus)
	existingGroup.TxId = pin.Id[:len(pin.Id)-2] //截取掉后两位
	existingGroup.PinId = pin.Id
	existingGroup.Timestamp = pin.Timestamp
	existingGroup.BlockHeight = pin.GenesisHeight

	// 处理群组与社区的关系变化
	oldCommunityId := existingGroup.CommunityId
	newCommunityId := simpleGroupCreate.CommunityId

	// 情况一：原来没有communityId，现在有了
	if oldCommunityId == "" && newCommunityId != "" {
		existingGroup.CommunityId = newCommunityId
		err = gdb.SaveGroupCommunity(existingGroup)
		if err != nil {
			return err
		}
	} else if oldCommunityId != "" && newCommunityId == "" {
		// 情况二：原来有communityId，现在没有了
		err = gdb.DeleteGroupCommunity(oldCommunityId, existingGroup.GroupId)
		if err != nil {
			return err
		}
		existingGroup.CommunityId = ""
	} else if oldCommunityId != "" && newCommunityId != "" && oldCommunityId != newCommunityId {
		// 情况三：原来有communityId，现在有新的不同的communityId
		// 删除旧的关联
		err = gdb.DeleteGroupCommunity(oldCommunityId, existingGroup.GroupId)
		if err != nil {
			return err
		}
		// 保存新的关联
		existingGroup.CommunityId = newCommunityId
		err = gdb.SaveGroupCommunity(existingGroup)
		if err != nil {
			return err
		}
	} else if oldCommunityId != "" && newCommunityId != "" && oldCommunityId == newCommunityId {
		// 情况四：原来有communityId，现在有相同的communityId
		// 直接更新关联
		err = gdb.SaveGroupCommunity(existingGroup)
		if err != nil {
			return err
		}
	}

	// 保存到版本信息表
	err = gdb.SaveGroupVersionInfo(existingGroup)
	if err != nil {
		return err
	}

	// 保存到基本信息表
	err = gdb.SaveGroupInfo(existingGroup)
	if err != nil {
		return err
	}

	return nil
}

// 处理群组加入
func (gdb *GroupDB) processGroupJoin(pin *pin.PinInscription) error {
	// 解析协议数据
	var simpleGroupJoin protocols.SimpleGroupJoin
	err := json.Unmarshal(pin.ContentBody, &simpleGroupJoin)
	if err != nil {
		return err
	}

	// 确定加入状态
	var groupState models.RoomState
	if state, ok := simpleGroupJoin.State.(float64); ok {
		if state == 1 {
			groupState = models.RoomStateIn
		} else {
			groupState = models.RoomStateOut
		}
	} else {
		groupState = models.RoomStateIn // 默认加入
	}

	// 创建群组加入模型
	join := &models.TalkGroupJoinModel{
		GroupId:    simpleGroupJoin.GroupId,
		MetaId:     pin.CreateMetaId,
		TxId:       pin.Id[:len(pin.Id)-2], //截取掉后两位
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

	// 保存群组加入信息（无论状态如何都要保存）
	err = gdb.SaveGroupJoin(join)
	if err != nil {
		return err
	}

	// 获取现有的成员信息
	existingPerson, err := gdb.GetGroupPersonByGroupIdAndMetaId(simpleGroupJoin.GroupId, pin.MetaId)
	if err != nil {
		return err
	}

	// 处理成员状态变化
	if existingPerson == nil {
		// 新成员，根据状态决定是否保存
		if groupState == models.RoomStateIn {
			// 创建群组成员信息
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

			// 保存群组成员信息
			err = gdb.SaveGroupPerson(person)
			if err != nil {
				return err
			}

			// 添加到用户的群列表
			err = gdb.addGroupToMetaIdContextList(pin.CreateMetaId, simpleGroupJoin.GroupId, pin)
			if err != nil {
				return err
			}

			// 添加加入记录到MetaId加入列表
			err = gdb.addGroupJoinToMetaIdList(pin.CreateMetaId, simpleGroupJoin.GroupId, pin.Id, "join", pin, groupState, simpleGroupJoin.Referrer)
			if err != nil {
				return err
			}
		}
		// 如果是 Out 状态且是新成员，不需要保存
	} else {
		// 现有成员，检查状态变化
		if existingPerson.GroupState != groupState {
			// 状态发生变化，需要更新
			if groupState == models.RoomStateIn {
				// 从 Out 变为 In，保存成员信息
				existingPerson.GroupState = groupState
				existingPerson.Timestamp = pin.Timestamp
				existingPerson.PinId = pin.Id
				existingPerson.BlockHeight = pin.GenesisHeight
				err = gdb.SaveGroupPerson(existingPerson)
				if err != nil {
					return err
				}

				// 添加到用户的群列表
				err = gdb.addGroupToMetaIdContextList(pin.CreateMetaId, simpleGroupJoin.GroupId, pin)
				if err != nil {
					return err
				}

				// 添加加入记录到MetaId加入列表
				err = gdb.addGroupJoinToMetaIdList(pin.CreateMetaId, simpleGroupJoin.GroupId, pin.Id, "join", pin, groupState, simpleGroupJoin.Referrer)
				if err != nil {
					return err
				}
			} else {
				// 从 In 变为 Out，删除成员信息
				err = gdb.DeleteGroupPerson(simpleGroupJoin.GroupId, pin.MetaId)
				if err != nil {
					return err
				}

				// 从用户的群列表中移除
				err = gdb.removeGroupFromMetaIdContextList(pin.CreateMetaId, simpleGroupJoin.GroupId)
				if err != nil {
					return err
				}

				// 添加退出记录到MetaId加入列表（状态为out）
				err = gdb.addGroupJoinToMetaIdList(pin.CreateMetaId, simpleGroupJoin.GroupId, pin.Id, "leave", pin, groupState, simpleGroupJoin.Referrer)
				if err != nil {
					return err
				}
			}
		}
		// 如果状态没有变化，不需要更新
	}

	return nil
}

// 处理创建者自动加入群组
func (gdb *GroupDB) processCreatorAutoJoin(group *models.TalkGroupModel, pin *pin.PinInscription) error {
	// 创建群组加入记录
	join := &models.TalkGroupJoinModel{
		GroupId:      group.GroupId,
		MetaId:       pin.CreateMetaId,
		TxId:         pin.Id[:len(pin.Id)-2], // 截取掉后两位
		PinId:        pin.Id,
		Address:      pin.CreateAddress,
		GroupState:   models.RoomStateIn, // 创建者默认加入
		Referrer:     "",                 // 创建者没有推荐人
		BlockHeight:  pin.GenesisHeight,
		Chain:        pin.ChainName,
		ConfirmState: 0,
		Timestamp:    pin.Timestamp,
	}

	// 保存群组加入信息
	err := gdb.SaveGroupJoin(join)
	if err != nil {
		return err
	}

	// 创建群组成员信息
	person := &models.TalkGroupPerson{
		GroupId:      group.GroupId,
		MetaId:       pin.CreateMetaId,
		Address:      pin.CreateAddress,
		UserName:     "",
		UserNickName: "",
		GroupState:   models.RoomStateIn, // 创建者默认加入
		Timestamp:    pin.Timestamp,
		BlockHeight:  pin.GenesisHeight,
	}

	// 保存群组成员信息
	err = gdb.SaveGroupPerson(person)
	if err != nil {
		return err
	}

	// 添加创建者加入记录到MetaId加入列表
	err = gdb.addGroupJoinToMetaIdList(pin.CreateMetaId, group.GroupId, pin.Id, "create", pin, models.RoomStateIn, "")
	if err != nil {
		return err
	}

	return nil
}

// 初始化群组最新聊天记录
func (gdb *GroupDB) initGroupLatestChat(groupId string, pin *pin.PinInscription) error {
	// 先获取现有的最新聊天记录
	existingLatestChat, err := gdb.getGroupLatestChat(groupId)
	if err != nil {
		return err
	}

	// 创建新的最新聊天记录
	newLatestChat := &models.TalkGroupLatestChat{
		GroupId:          groupId,
		Timestamp:        pin.Timestamp,
		ChatType:         models.ChatTypeMsg, // 默认为消息类型
		Content:          "",                 // 初始为空
		CreateAddress:    pin.CreateAddress,
		LastMessagePinId: "", // 初始为空
		MetaId:           pin.CreateMetaId,
		TxId:             pin.Id[:len(pin.Id)-2],
		Protocol:         pin.Path,
		ContentType:      "",
		Encryption:       "",
		ReplyPin:         "",
		Chain:            pin.ChainName,
		BlockHeight:      pin.GenesisHeight,
	}

	// 如果不存在现有数据，直接保存
	if existingLatestChat == nil {
		data, err := json.Marshal(newLatestChat)
		if err != nil {
			return err
		}
		key := []byte(groupId)
		return Pb[TalkGroupLatestChatCollection].Set(key, data, pebble.Sync)
	}

	// 判断是否需要更新
	shouldUpdate := false

	// 1. 先判断pinId是否一样（这里LastMessagePinId都是空的，所以主要比较其他字段）
	if existingLatestChat.LastMessagePinId == newLatestChat.LastMessagePinId {
		// pinId一样，检查blockHeight是否不一样
		if existingLatestChat.BlockHeight != newLatestChat.BlockHeight {
			shouldUpdate = true
		}
	} else {
		// pinId不一样，比较timestamp
		if newLatestChat.Timestamp > existingLatestChat.Timestamp {
			shouldUpdate = true
		}
	}

	// 如果需要更新，则保存新数据
	if shouldUpdate {
		data, err := json.Marshal(newLatestChat)
		if err != nil {
			return err
		}
		key := []byte(groupId)
		return Pb[TalkGroupLatestChatCollection].Set(key, data, pebble.Sync)
	}

	// 不需要更新，直接返回
	return nil
}

// 获取群组最新聊天记录（GroupDB内部方法）
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

// 初始化用户的群列表
func (gdb *GroupDB) initMetaIdContextList(metaId, groupId string, pin *pin.PinInscription) error {
	// 先获取现有的群列表
	existingList, err := gdb.getMetaIdContextList(metaId)
	if err != nil {
		return err
	}

	// 创建新的群列表项
	newItem := &models.MetaIdContextItem{
		GroupId:          groupId,
		Timestamp:        pin.Timestamp,
		ChatType:         models.ChatTypeMsg, // 默认为消息类型
		Content:          "",                 // 初始为空
		CreateAddress:    pin.CreateAddress,
		LastMessagePinId: "", // 初始为空
		BlockHeight:      pin.GenesisHeight,
	}

	// 如果不存在现有数据，直接保存
	if existingList == nil || len(existingList.Items) == 0 {
		// 创建群列表
		contextList := &models.MetaIdContextList{
			MetaId: metaId,
			Items:  []*models.MetaIdContextItem{newItem},
		}

		// 序列化数据
		data, err := json.Marshal(contextList)
		if err != nil {
			return err
		}

		// 使用 MetaId 作为主键保存到 TalkMetaIdContextListCollection
		key := []byte(metaId)
		return Pb[TalkMetaIdContextListCollection].Set(key, data, pebble.Sync)
	}

	// 检查是否已存在该群组
	found := false
	shouldUpdate := false
	for i, item := range existingList.Items {
		if item.GroupId == groupId {
			// 判断是否需要更新
			if item.LastMessagePinId == newItem.LastMessagePinId {
				// LastMessagePinId一样，检查blockHeight是否不一样
				if item.BlockHeight != newItem.BlockHeight {
					shouldUpdate = true
				}
			} else {
				// LastMessagePinId不一样，比较timestamp
				if newItem.Timestamp > item.Timestamp {
					shouldUpdate = true
				}
			}

			if shouldUpdate {
				// 更新现有项
				existingList.Items[i] = newItem
			}
			found = true
			break
		}
	}

	// 如果不存在该群组，添加新项
	if !found {
		existingList.Items = append(existingList.Items, newItem)
		shouldUpdate = true
	}

	// 如果需要更新，保存数据
	if shouldUpdate {
		// 按时间戳倒序排序
		gdb.sortContextListByTimestamp(existingList)
		return gdb.saveMetaIdContextList(existingList)
	}

	// 不需要更新，直接返回
	return nil
}

// 初始化用户的群组加入列表
func (gdb *GroupDB) initGroupMetaIdJoinList(metaId, groupId string, pin *pin.PinInscription) error {
	// 创建初始的加入记录项
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

	// 创建加入列表
	joinList := &GroupMetaIdJoinList{
		MetaId: metaId,
		Items:  []*GroupMetaIdJoinItem{joinItem},
	}

	// 序列化数据
	data, err := json.Marshal(joinList)
	if err != nil {
		return err
	}

	// key: metaId_groupId
	key := []byte(metaId + "_" + groupId)
	return Pb[TalkGroupMetaIdJoinCollection].Set(key, data, pebble.Sync)
}

// 更新用户的群列表（添加群组）
func (gdb *GroupDB) addGroupToMetaIdContextList(metaId, groupId string, pin *pin.PinInscription) error {
	// 获取现有的群列表
	existingList, err := gdb.getMetaIdContextList(metaId)
	if err != nil {
		return err
	}

	// 从TalkGroupLatestChatCollection获取群组最新聊天信息
	latestChat, err := gdb.getGroupLatestChat(groupId)
	if err != nil {
		return err
	}

	// 创建新的群列表项
	newItem := &models.MetaIdContextItem{
		GroupId:          groupId,
		Timestamp:        pin.Timestamp,
		ChatType:         models.ChatTypeMsg, // 默认为消息类型
		Content:          "",                 // 初始为空
		CreateAddress:    pin.CreateAddress,
		LastMessagePinId: "", // 初始为空
	}

	// 如果获取到了最新聊天信息，使用其数据
	if latestChat != nil {
		newItem.Content = latestChat.Content
		newItem.LastMessagePinId = latestChat.LastMessagePinId
		newItem.Timestamp = latestChat.Timestamp
		newItem.ChatType = latestChat.ChatType
		newItem.BlockHeight = latestChat.BlockHeight
	}

	// 检查是否已存在该群组
	found := false
	shouldUpdate := false
	for i, item := range existingList.Items {
		if item.GroupId == groupId {
			// 判断是否需要更新
			if item.LastMessagePinId == newItem.LastMessagePinId {
				// LastMessagePinId一样，检查blockHeight是否不一样
				if item.BlockHeight != newItem.BlockHeight {
					shouldUpdate = true
				}
			} else {
				// LastMessagePinId不一样，比较timestamp
				if newItem.Timestamp > item.Timestamp {
					shouldUpdate = true
				}
			}

			if shouldUpdate {
				// 更新现有项
				existingList.Items[i] = newItem
			}
			found = true
			break
		}
	}

	// 如果不存在该群组，添加新项
	if !found {
		existingList.Items = append(existingList.Items, newItem)
		shouldUpdate = true
	}

	// 如果需要更新，保存数据
	if shouldUpdate {
		// 按时间戳倒序排序
		gdb.sortContextListByTimestamp(existingList)
		return gdb.saveMetaIdContextList(existingList)
	}

	// 不需要更新，直接返回
	return nil
}

// 从用户的群列表中移除群组
func (gdb *GroupDB) removeGroupFromMetaIdContextList(metaId, groupId string) error {
	// 获取现有的群列表
	existingList, err := gdb.getMetaIdContextList(metaId)
	if err != nil {
		return err
	}

	// 移除指定群组
	var newItems []*models.MetaIdContextItem
	for _, item := range existingList.Items {
		if item.GroupId != groupId {
			newItems = append(newItems, item)
		}
	}

	existingList.Items = newItems

	// 保存更新后的群列表
	return gdb.saveMetaIdContextList(existingList)
}

// 获取用户的群列表
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

// 保存用户的群列表
func (gdb *GroupDB) saveMetaIdContextList(contextList *models.MetaIdContextList) error {
	data, err := json.Marshal(contextList)
	if err != nil {
		return err
	}

	key := []byte(contextList.MetaId)
	return Pb[TalkMetaIdContextListCollection].Set(key, data, pebble.Sync)
}

// 按时间戳倒序排序群列表
func (gdb *GroupDB) sortContextListByTimestamp(contextList *models.MetaIdContextList) {
	// 简单的冒泡排序，按时间戳倒序
	for i := 0; i < len(contextList.Items)-1; i++ {
		for j := 0; j < len(contextList.Items)-1-i; j++ {
			if contextList.Items[j].Timestamp < contextList.Items[j+1].Timestamp {
				contextList.Items[j], contextList.Items[j+1] = contextList.Items[j+1], contextList.Items[j]
			}
		}
	}
}

// 辅助函数：获取字符串值
func getStringValue(v interface{}) string {
	if v == nil {
		return ""
	}
	if str, ok := v.(string); ok {
		return str
	}
	return ""
}

// 辅助函数：获取int64值
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

// 保存群组加入信息
func (gdb *GroupDB) SaveGroupJoin(join *models.TalkGroupJoinModel) error {
	data, err := json.Marshal(join)
	if err != nil {
		return err
	}

	// 使用 GroupId_PinId 作为主键
	key1 := []byte(join.GroupId + "_" + join.PinId)
	err = Pb[TalkGroupJoinCollection].Set(key1, data, pebble.Sync)
	if err != nil {
		return err
	}

	// 使用 PinId_GroupId 作为主键
	key2 := []byte(join.PinId + "_" + join.GroupId)
	return Pb[TalkGroupJoinCollection].Set(key2, data, pebble.Sync)
}

// 根据群组ID和PinId获取加入信息
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

// 获取群组成员列表
func (gdb *GroupDB) GetGroupMembers(groupId string) ([]*models.TalkGroupJoinModel, error) {
	var members []*models.TalkGroupJoinModel
	iter, err := Pb[TalkGroupJoinCollection].NewIter(nil)
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
		if join.GroupId == groupId && join.GroupState == models.RoomStateIn {
			members = append(members, &join)
		}
	}

	return members, nil
}

// 保存群组成员信息
func (gdb *GroupDB) SaveGroupPerson(person *models.TalkGroupPerson) error {
	data, err := json.Marshal(person)
	if err != nil {
		return err
	}

	// 使用 GroupId_MetaId 作为主键
	key1 := []byte(person.GroupId + "_" + person.MetaId)
	err = Pb[TalkGroupPersonCollection].Set(key1, data, pebble.Sync)
	if err != nil {
		return err
	}

	// 使用 MetaId_GroupId 作为主键
	key2 := []byte(person.MetaId + "_" + person.GroupId)
	return Pb[TalkGroupPersonCollection].Set(key2, data, pebble.Sync)
}

// 删除群组成员信息
func (gdb *GroupDB) DeleteGroupPerson(groupId, metaId string) error {
	// 使用 GroupId_MetaId 作为主键
	key1 := []byte(groupId + "_" + metaId)
	err := Pb[TalkGroupPersonCollection].Delete(key1, pebble.Sync)
	if err != nil {
		return err
	}

	// 使用 MetaId_GroupId 作为主键
	key2 := []byte(metaId + "_" + groupId)
	return Pb[TalkGroupPersonCollection].Delete(key2, pebble.Sync)
}

// 根据群组ID和MetaId获取成员信息
func (gdb *GroupDB) GetGroupPersonByGroupIdAndMetaId(groupId, metaId string) (*models.TalkGroupPerson, error) {
	// 构造 GroupId_MetaId
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

// 根据MetaId和GroupId获取成员信息
func (gdb *GroupDB) GetGroupPersonByMetaIdAndGroupId(metaId, groupId string) (*models.TalkGroupPerson, error) {
	// 构造 MetaId_GroupId
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

// 获取群组成员列表
func (gdb *GroupDB) GetGroupPersonList(groupId string) ([]*models.TalkGroupPerson, error) {
	var persons []*models.TalkGroupPerson
	iter, err := Pb[TalkGroupPersonCollection].NewIter(nil)
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
		if person.GroupId == groupId && person.GroupState == models.RoomStateIn {
			persons = append(persons, &person)
		}
	}

	return persons, nil
}

// 群组MetaId加入记录项
type GroupMetaIdJoinItem struct {
	JoinPinId     string           `json:"joinPinId"`     // 加入的PinId
	JoinType      string           `json:"joinType"`      // 加入类型：create, join
	JoinTimestamp int64            `json:"joinTimestamp"` // 加入时间戳
	GroupState    models.RoomState `json:"groupState"`    // 群组状态：1-in, -1-out
	Address       string           `json:"address"`       // 用户地址
	Referrer      string           `json:"referrer"`      // 推荐人
	BlockHeight   int64            `json:"blockHeight"`   // 区块高度
	Chain         string           `json:"chain"`         // 链类型
}

// 群组MetaId加入列表
type GroupMetaIdJoinList struct {
	MetaId string                 `json:"metaId"` // 用户MetaId
	Items  []*GroupMetaIdJoinItem `json:"items"`  // 加入记录列表
}

// 获取用户的群组加入列表
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

// 保存用户的群组加入列表
func (gdb *GroupDB) saveGroupMetaIdJoinList(joinList *GroupMetaIdJoinList, groupId string) error {
	data, err := json.Marshal(joinList)
	if err != nil {
		return err
	}

	//key: metaId_groupId
	key := []byte(joinList.MetaId + "_" + groupId)
	return Pb[TalkGroupMetaIdJoinCollection].Set(key, data, pebble.Sync)
}

// 添加群组加入记录到用户的加入列表
func (gdb *GroupDB) addGroupJoinToMetaIdList(metaId, groupId, pinId, joinType string, pin *pin.PinInscription, groupState models.RoomState, referrer string) error {
	// 获取现有的加入列表
	existingList, err := gdb.getGroupMetaIdJoinList(metaId)
	if err != nil {
		return err
	}

	// 创建新的加入记录项
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

	// 检查是否已存在该群组的记录
	found := false
	for i, item := range existingList.Items {
		// 通过JoinPinId来判断是否已存在（因为每次加入都有不同的PinId）
		if item.JoinPinId == pinId {
			// 更新现有项
			existingList.Items[i] = newItem
			found = true
			break
		}
	}

	// 如果不存在，添加新项
	if !found {
		existingList.Items = append(existingList.Items, newItem)
	}

	// 按时间戳倒序排序
	gdb.sortGroupJoinListByTimestamp(existingList)

	// 保存更新后的加入列表
	return gdb.saveGroupMetaIdJoinList(existingList, groupId)
}

// 按时间戳倒序排序群组加入列表
func (gdb *GroupDB) sortGroupJoinListByTimestamp(joinList *GroupMetaIdJoinList) {
	// 简单的冒泡排序，按时间戳倒序
	for i := 0; i < len(joinList.Items)-1; i++ {
		for j := 0; j < len(joinList.Items)-1-i; j++ {
			if joinList.Items[j].JoinTimestamp < joinList.Items[j+1].JoinTimestamp {
				joinList.Items[j], joinList.Items[j+1] = joinList.Items[j+1], joinList.Items[j]
			}
		}
	}
}
