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
	data, err := json.Marshal(group)
	if err != nil {
		return err
	}

	// 使用 GroupId 作为主键
	key := []byte(group.GroupId)
	return Pb[TalkGroupInfoCollection].Set(key, data, pebble.Sync)
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
		GroupId:           simpleGroupCreate.GroupId,
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
		Timestamp:         pin.Timestamp,
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
		GroupId:      simpleGroupJoin.GroupId,
		MetaId:       pin.CreateMetaId,
		TxId:         pin.Id[:len(pin.Id)-2], //截取掉后两位
		PinId:        pin.Id,
		Address:      pin.CreateAddress,
		GroupState:   groupState,
		Referrer:     simpleGroupJoin.Referrer,
		IsValid:      true,
		IsNew:        true,
		BlockHeight:  pin.GenesisHeight,
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
			}

			// 保存群组成员信息
			err = gdb.SaveGroupPerson(person)
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
				err = gdb.SaveGroupPerson(existingPerson)
				if err != nil {
					return err
				}
			} else {
				// 从 In 变为 Out，删除成员信息
				err = gdb.DeleteGroupPerson(simpleGroupJoin.GroupId, pin.MetaId)
				if err != nil {
					return err
				}
			}
		}
		// 如果状态没有变化，不需要更新
	}

	return nil
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
