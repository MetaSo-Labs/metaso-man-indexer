package db

import (
	"encoding/json"
	"manindexer/basicprotocols/group_chat/models"
	"manindexer/basicprotocols/group_chat/protocols"
	"manindexer/pin"
	"strings"

	"github.com/cockroachdb/pebble"
)

// 社区数据库操作
type CommunityDB struct {
	pb *Pebble
}

func NewCommunityDB(pb *Pebble) *CommunityDB {
	return &CommunityDB{pb: pb}
}

// 保存社区版本信息
func (cdb *CommunityDB) SaveCommunityVersionInfo(community *models.TalkCommunityModel) error {
	data, err := json.Marshal(community)
	if err != nil {
		return err
	}

	// 使用 CommunityId_PinId 作为主键
	key := []byte(community.CommunityId + "_" + community.PinId)
	return Pb[TalkCommunityVersionInfoCollection].Set(key, data, pebble.Sync)
}

// 根据CommunityId和PinId获取社区版本信息
func (cdb *CommunityDB) GetCommunityVersionInfoByCommunityIdAndPinId(communityId, pinId string) (*models.TalkCommunityModel, error) {
	key := []byte(communityId + "_" + pinId)
	value, closer, err := Pb[TalkCommunityVersionInfoCollection].Get(key)
	if err != nil {
		if err == pebble.ErrNotFound {
			return nil, nil
		}
		return nil, err
	}
	defer closer.Close()

	var community models.TalkCommunityModel
	err = json.Unmarshal(value, &community)
	if err != nil {
		return nil, err
	}

	return &community, nil
}

// 保存社区信息
func (cdb *CommunityDB) SaveCommunityInfo(community *models.TalkCommunityModel) error {
	data, err := json.Marshal(community)
	if err != nil {
		return err
	}

	// 使用 CommunityId 作为主键
	key := []byte(community.CommunityId)
	return Pb[TalkCommunityInfoCollection].Set(key, data, pebble.Sync)
}

// 根据CommunityId获取社区信息
func (cdb *CommunityDB) GetCommunityInfoByCommunityId(communityId string) (*models.TalkCommunityModel, error) {
	key := []byte(communityId)
	value, closer, err := Pb[TalkCommunityInfoCollection].Get(key)
	if err != nil {
		if err == pebble.ErrNotFound {
			return nil, nil
		}
		return nil, err
	}
	defer closer.Close()

	var community models.TalkCommunityModel
	err = json.Unmarshal(value, &community)
	if err != nil {
		return nil, err
	}

	return &community, nil
}

// 获取社区列表
func (cdb *CommunityDB) GetCommunityList(page, size int64) ([]*models.TalkCommunityModel, error) {
	var communities []*models.TalkCommunityModel
	iter, err := Pb[TalkCommunityInfoCollection].NewIter(nil)
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

		if int64(len(communities)) >= size {
			break
		}

		var community models.TalkCommunityModel
		err := json.Unmarshal(iter.Value(), &community)
		if err != nil {
			continue
		}
		communities = append(communities, &community)
	}

	return communities, nil
}

// 删除社区
func (cdb *CommunityDB) DeleteCommunity(communityId string) error {
	key := []byte(communityId)
	return Pb[TalkCommunityInfoCollection].Delete(key, pebble.Sync)
}

// 保存社区加入信息
func (cdb *CommunityDB) SaveCommunityJoin(join *models.TalkCommunityJoinModel) error {
	data, err := json.Marshal(join)
	if err != nil {
		return err
	}

	// 使用 CommunityId_PinId 作为主键
	key := []byte(join.CommunityId + "_" + join.PinId)
	return Pb[TalkCommunityJoinCollection].Set(key, data, pebble.Sync)
}

// 根据社区ID和PinId获取加入信息
func (cdb *CommunityDB) GetCommunityJoinByCommunityIdAndPinId(communityId, pinId string) (*models.TalkCommunityJoinModel, error) {
	key := []byte(communityId + "_" + pinId)
	value, closer, err := Pb[TalkCommunityJoinCollection].Get(key)
	if err != nil {
		if err == pebble.ErrNotFound {
			return nil, nil
		}
		return nil, err
	}
	defer closer.Close()

	var join models.TalkCommunityJoinModel
	err = json.Unmarshal(value, &join)
	if err != nil {
		return nil, err
	}

	return &join, nil
}

// 获取社区成员列表
func (cdb *CommunityDB) GetCommunityMembers(communityId string) ([]*models.TalkCommunityJoinModel, error) {
	var members []*models.TalkCommunityJoinModel
	iter, err := Pb[TalkCommunityJoinCollection].NewIter(nil)
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	for iter.First(); iter.Valid(); iter.Next() {
		var join models.TalkCommunityJoinModel
		err := json.Unmarshal(iter.Value(), &join)
		if err != nil {
			continue
		}
		if join.CommunityId == communityId && join.CommunityState == models.RoomStateIn {
			members = append(members, &join)
		}
	}

	return members, nil
}

// 保存社区成员信息
func (cdb *CommunityDB) SaveCommunityPerson(person *models.TalkCommunityPerson) error {
	data, err := json.Marshal(person)
	if err != nil {
		return err
	}

	// 使用 CommunityId_MetaId 作为主键
	key := []byte(person.CommunityId + "_" + person.MetaId)
	return Pb[TalkCommunityPersonCollection].Set(key, data, pebble.Sync)
}

// 根据社区ID和MetaId获取成员信息
func (cdb *CommunityDB) GetCommunityPersonByCommunityIdAndMetaId(communityId, metaId string) (*models.TalkCommunityPerson, error) {
	// 构造 CommunityId_MetaId
	key := []byte(communityId + "_" + metaId)
	value, closer, err := Pb[TalkCommunityPersonCollection].Get(key)
	if err != nil {
		if err == pebble.ErrNotFound {
			return nil, nil
		}
		return nil, err
	}
	defer closer.Close()

	var person models.TalkCommunityPerson
	err = json.Unmarshal(value, &person)
	if err != nil {
		return nil, err
	}

	return &person, nil
}

// 获取社区成员列表
func (cdb *CommunityDB) GetCommunityPersonList(communityId string) ([]*models.TalkCommunityPerson, error) {
	var persons []*models.TalkCommunityPerson
	iter, err := Pb[TalkCommunityPersonCollection].NewIter(nil)
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	for iter.First(); iter.Valid(); iter.Next() {
		var person models.TalkCommunityPerson
		err := json.Unmarshal(iter.Value(), &person)
		if err != nil {
			continue
		}
		if person.CommunityId == communityId && person.CommunityState == models.RoomStateIn {
			persons = append(persons, &person)
		}
	}

	return persons, nil
}

// 保存社区地址关联
func (cdb *CommunityDB) SaveCommunityAddress(communityId, address string, data []byte) error {
	// 使用 CommunityId_Address 作为主键
	key := []byte(communityId + "_" + address)
	return Pb[TalkCommunityAddressCollection].Set(key, data, pebble.Sync)
}

// 根据社区ID和地址获取地址关联信息
func (cdb *CommunityDB) GetCommunityAddressByCommunityIdAndAddress(communityId, address string) ([]byte, error) {
	key := []byte(communityId + "_" + address)
	value, closer, err := Pb[TalkCommunityAddressCollection].Get(key)
	if err != nil {
		if err == pebble.ErrNotFound {
			return nil, nil
		}
		return nil, err
	}
	defer closer.Close()

	return value, nil
}

// 总的处理 Community 方法
func (cdb *CommunityDB) ProcessCommunityPin(pin *pin.PinInscription) error {
	switch pin.Operation {
	case "create":
		path := pin.Path
		protocol := strings.Replace(path, "/protocols/", "", -1)
		if strings.ToLower(protocol) == strings.ToLower(protocols.MonitorSimpleCommunity) {
			return cdb.processCommunityCreate(pin)
		} else if strings.ToLower(protocol) == strings.ToLower(protocols.MonitorSimpleCommunityJoin) {
			return cdb.processCommunityJoin(pin)
		}
	case "modify":
		//检查ParentPath
		parentPath := pin.Path
		parentProtocol := strings.Replace(parentPath, "/protocols/", "", -1)
		if strings.ToLower(parentProtocol) == strings.ToLower(protocols.MonitorSimpleCommunity) {
			return cdb.processCommunityModify(pin)
		}
		return nil
	default:
		return nil // 未知操作类型，跳过
	}
	return nil
}

// 处理社区创建
func (cdb *CommunityDB) processCommunityCreate(pin *pin.PinInscription) error {
	// 解析协议数据
	var simpleCommunity protocols.SimpleCommunity
	err := json.Unmarshal(pin.ContentBody, &simpleCommunity)
	if err != nil {
		return err
	}

	// 创建社区模型
	community := &models.TalkCommunityModel{
		CommunityId: simpleCommunity.CommunityId,
		MetaId:      pin.MetaId,
		TxId:        pin.Id[:len(pin.Id)-2],
		PinId:       pin.Id,
		Address:     pin.Address,
		// PublicKey:   pin.CreateAddress,
		Name:        simpleCommunity.Name,
		Description: simpleCommunity.Description,
		Cover:       simpleCommunity.Cover,
		Icon:        simpleCommunity.Icon,
		MetaName:    simpleCommunity.MetaName,
		MetaNameNft: simpleCommunity.MetaNameNft,
		Admins:      simpleCommunity.Admins,
		Reserved:    simpleCommunity.Reserved,
		BlockHeight: pin.GenesisHeight,
		Timestamp:   pin.Timestamp,
	}

	// 保存到版本信息表
	err = cdb.SaveCommunityVersionInfo(community)
	if err != nil {
		return err
	}

	// 保存到基本信息表
	err = cdb.SaveCommunityInfo(community)
	if err != nil {
		return err
	}

	// 保存地址关联
	addressData, _ := json.Marshal(community)
	err = cdb.SaveCommunityAddress(community.CommunityId, pin.Address, addressData)
	if err != nil {
		return err
	}

	return nil
}

// 处理社区加入
func (cdb *CommunityDB) processCommunityJoin(pin *pin.PinInscription) error {
	// 解析协议数据
	var simpleCommunityJoin protocols.SimpleCommunityJoin
	err := json.Unmarshal(pin.ContentBody, &simpleCommunityJoin)
	if err != nil {
		return err
	}

	// 确定加入状态
	var communityState models.RoomState
	if state, ok := simpleCommunityJoin.State.(float64); ok {
		if state == 1 {
			communityState = models.RoomStateIn
		} else {
			communityState = models.RoomStateOut
		}
	} else {
		communityState = models.RoomStateIn // 默认加入
	}

	// 创建社区加入模型
	join := &models.TalkCommunityJoinModel{
		CommunityId: simpleCommunityJoin.CommunityId,
		MetaId:      pin.MetaId,
		TxId:        pin.Id[:len(pin.Id)-2],
		PinId:       pin.Id,
		Address:     pin.Address,
		// PublicKey:      pin.CreateAddress,
		CommunityState: communityState,
		IsValid:        true,
		IsNew:          true,
		BlockHeight:    pin.GenesisHeight,
		Timestamp:      pin.Timestamp,
	}

	// 保存社区加入信息
	err = cdb.SaveCommunityJoin(join)
	if err != nil {
		return err
	}

	// 创建社区成员信息
	person := &models.TalkCommunityPerson{
		CommunityId:    simpleCommunityJoin.CommunityId,
		MetaId:         pin.MetaId,
		UserName:       pin.MetaId, // 使用 MetaId 作为用户名
		UserNickName:   pin.MetaId,
		CommunityState: communityState,
		Timestamp:      pin.Timestamp,
	}

	// 保存社区成员信息
	err = cdb.SaveCommunityPerson(person)
	if err != nil {
		return err
	}

	return nil
}

// 处理社区修改
func (cdb *CommunityDB) processCommunityModify(pin *pin.PinInscription) error {
	// 解析协议数据
	var simpleCommunity protocols.SimpleCommunity
	err := json.Unmarshal(pin.ContentBody, &simpleCommunity)
	if err != nil {
		return err
	}

	// 获取现有社区信息
	existingCommunity, err := cdb.GetCommunityInfoByCommunityId(simpleCommunity.CommunityId)
	if err != nil {
		return err
	}

	if existingCommunity == nil {
		// 如果社区不存在，按创建处理
		return cdb.processCommunityCreate(pin)
	}

	// 更新社区信息
	existingCommunity.Name = simpleCommunity.Name
	existingCommunity.Description = simpleCommunity.Description
	existingCommunity.Cover = simpleCommunity.Cover
	existingCommunity.Icon = simpleCommunity.Icon
	existingCommunity.MetaName = simpleCommunity.MetaName
	existingCommunity.MetaNameNft = simpleCommunity.MetaNameNft
	existingCommunity.Admins = simpleCommunity.Admins
	existingCommunity.Reserved = simpleCommunity.Reserved
	existingCommunity.TxId = pin.Id[:len(pin.Id)-2]
	existingCommunity.PinId = pin.Id
	existingCommunity.Timestamp = pin.Timestamp

	// 保存到版本信息表
	err = cdb.SaveCommunityVersionInfo(existingCommunity)
	if err != nil {
		return err
	}

	// 保存到基本信息表
	err = cdb.SaveCommunityInfo(existingCommunity)
	if err != nil {
		return err
	}

	return nil
}
