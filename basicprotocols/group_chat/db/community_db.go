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

// Community database operations
type CommunityDB struct {
	pb *Pebble
}

func NewCommunityDB(pb *Pebble) *CommunityDB {
	return &CommunityDB{pb: pb}
}

// Save community version info
func (cdb *CommunityDB) SaveCommunityVersionInfo(community *models.TalkCommunityModel) error {
	data, err := json.Marshal(community)
	if err != nil {
		return err
	}

	// Use CommunityId_PinId as primary key
	key := []byte(community.CommunityId + "_" + community.PinId)
	return Pb[TalkCommunityVersionInfoCollection].Set(key, data, pebble.Sync)
}

// Get community version info by CommunityId and PinId
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

// Save community info
func (cdb *CommunityDB) SaveCommunityInfo(community *models.TalkCommunityModel) error {
	data, err := json.Marshal(community)
	if err != nil {
		return err
	}

	// Use CommunityId as primary key
	key := []byte(community.CommunityId)
	return Pb[TalkCommunityInfoCollection].Set(key, data, pebble.Sync)
}

// Get community info by CommunityId
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

// Get community list
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

// Delete community
func (cdb *CommunityDB) DeleteCommunity(communityId string) error {
	key := []byte(communityId)
	return Pb[TalkCommunityInfoCollection].Delete(key, pebble.Sync)
}

// Save community join info
func (cdb *CommunityDB) SaveCommunityJoin(join *models.TalkCommunityJoinModel) error {
	data, err := json.Marshal(join)
	if err != nil {
		return err
	}

	// Use CommunityId_PinId as primary key
	key := []byte(join.CommunityId + "_" + join.PinId)
	return Pb[TalkCommunityJoinCollection].Set(key, data, pebble.Sync)
}

// Get join info by community ID and PinId
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

// Get community member list
func (cdb *CommunityDB) GetCommunityMembers(communityId string) ([]*models.TalkCommunityJoinModel, error) {
	var members []*models.TalkCommunityJoinModel

	// Use prefix query, because key is communityId_pinId format
	prefix := []byte(communityId + "_")
	// iter, err := Pb[TalkCommunityJoinCollection].NewIter(&pebble.IterOptions{
	iter, err := Pb[TalkCommunityPersonCollection].NewIter(&pebble.IterOptions{
		LowerBound: prefix,
		UpperBound: append(prefix, 0xff), // Use 0xff as upper bound to ensure only query keys starting with communityId_
	})
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
		// Only return members in the community
		if join.CommunityState == models.RoomStateIn {
			members = append(members, &join)
		}
	}

	return members, nil
}

// Save community member info
func (cdb *CommunityDB) SaveCommunityPerson(person *models.TalkCommunityPerson) error {
	data, err := json.Marshal(person)
	if err != nil {
		return err
	}

	// Use CommunityId_MetaId as primary key
	key := []byte(person.CommunityId + "_" + person.MetaId)
	return Pb[TalkCommunityPersonCollection].Set(key, data, pebble.Sync)
}

// Get member info by community ID and MetaId
func (cdb *CommunityDB) GetCommunityPersonByCommunityIdAndMetaId(communityId, metaId string) (*models.TalkCommunityPerson, error) {
	// Construct CommunityId_MetaId
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

// Get community member list
func (cdb *CommunityDB) GetCommunityPersonList(communityId string) ([]*models.TalkCommunityPerson, error) {
	var persons []*models.TalkCommunityPerson

	// Use prefix query, because key is communityId_metaId format
	prefix := []byte(communityId + "_")
	iter, err := Pb[TalkCommunityPersonCollection].NewIter(&pebble.IterOptions{
		LowerBound: prefix,
		UpperBound: append(prefix, 0xff), // Use 0xff as upper bound to ensure only query keys starting with communityId_
	})
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
		// Only return members in the community
		if person.CommunityState == models.RoomStateIn {
			persons = append(persons, &person)
		}
	}

	return persons, nil
}

// Save community address association
func (cdb *CommunityDB) SaveCommunityAddress(communityId, address string, data []byte) error {
	// Use CommunityId_Address as primary key
	key := []byte(communityId + "_" + address)
	return Pb[TalkCommunityAddressCollection].Set(key, data, pebble.Sync)
}

// Get address association info by community ID and address
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

// Main method to process Community
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
		// Check ParentPath
		parentPath := pin.Path
		parentProtocol := strings.Replace(parentPath, "/protocols/", "", -1)
		if strings.ToLower(parentProtocol) == strings.ToLower(protocols.MonitorSimpleCommunity) {
			return cdb.processCommunityModify(pin)
		}
		return nil
	default:
		return nil // Unknown operation type, skip
	}
	return nil
}

// Process community creation
func (cdb *CommunityDB) processCommunityCreate(pin *pin.PinInscription) error {
	isSynced, err := IsPinSynced(pin.Id)
	if err != nil {
		return err
	}
	if isSynced {
		// Already synced, skip processing
		return nil
	}

	// Parse protocol data
	var simpleCommunity protocols.SimpleCommunity
	err = json.Unmarshal(pin.ContentBody, &simpleCommunity)
	if err != nil {
		return err
	}

	// Create community model
	community := &models.TalkCommunityModel{
		// CommunityId: simpleCommunity.CommunityId,
		CommunityId: pin.Id,
		MetaId:      pin.CreateMetaId,
		TxId:        pin.Id[:len(pin.Id)-2],
		PinId:       pin.Id,
		Address:     pin.CreateAddress,
		// PublicKey:   pin.CreateAddress,
		Name:        simpleCommunity.Name,
		Description: simpleCommunity.Description,
		Cover:       simpleCommunity.Cover,
		Icon:        simpleCommunity.Icon,
		MetaName:    simpleCommunity.MetaName,
		MetaNameNft: simpleCommunity.MetaNameNft,
		Admins:      simpleCommunity.Admins,
		Reserved:    simpleCommunity.Reserved,
		Chain:       pin.ChainName,
		BlockHeight: pin.GenesisHeight,
		Timestamp:   pin.Timestamp,
	}

	// Save to version info table
	err = cdb.SaveCommunityVersionInfo(community)
	if err != nil {
		return err
	}

	// Save to basic info table
	err = cdb.SaveCommunityInfo(community)
	if err != nil {
		return err
	}

	// Save address association
	addressData, _ := json.Marshal(community)
	err = cdb.SaveCommunityAddress(community.CommunityId, pin.Address, addressData)
	if err != nil {
		return err
	}

	// Mark pin as synced
	err = MarkPinAsSynced(pin.Id, true)
	if err != nil {
		return err
	}

	return nil
}

// Process community join
func (cdb *CommunityDB) processCommunityJoin(pin *pin.PinInscription) error {
	isSynced, err := IsPinSynced(pin.Id)
	if err != nil {
		return err
	}
	if isSynced {
		// Already synced, skip processing
		return nil
	}

	// Parse protocol data
	var simpleCommunityJoin protocols.SimpleCommunityJoin
	err = json.Unmarshal(pin.ContentBody, &simpleCommunityJoin)
	if err != nil {
		return err
	}

	// Determine join state
	var communityState models.RoomState
	if state, ok := simpleCommunityJoin.State.(float64); ok {
		if state == 1 {
			communityState = models.RoomStateIn
		} else {
			communityState = models.RoomStateOut
		}
	} else {
		communityState = models.RoomStateIn // Default join
	}

	// Create community join model
	join := &models.TalkCommunityJoinModel{
		CommunityId: simpleCommunityJoin.CommunityId,
		MetaId:      pin.MetaId,
		TxId:        pin.Id[:len(pin.Id)-2],
		PinId:       pin.Id,
		Address:     pin.Address,
		// PublicKey:      pin.CreateAddress,
		CommunityState: communityState,
		// IsValid:        true,
		// IsNew:          true,
		Chain:       pin.ChainName,
		BlockHeight: pin.GenesisHeight,
		Timestamp:   pin.Timestamp,
	}

	// Save community join info
	err = cdb.SaveCommunityJoin(join)
	if err != nil {
		return err
	}

	// Create community member info
	person := &models.TalkCommunityPerson{
		CommunityId:    simpleCommunityJoin.CommunityId,
		MetaId:         pin.MetaId,
		UserName:       pin.MetaId, // Use MetaId as username
		UserNickName:   pin.MetaId,
		CommunityState: communityState,
		Timestamp:      pin.Timestamp,
	}

	// Save community member info
	err = cdb.SaveCommunityPerson(person)
	if err != nil {
		return err
	}

	// Mark pin as synced
	err = MarkPinAsSynced(pin.Id, true)
	if err != nil {
		return err
	}

	return nil
}

// Process community modification
func (cdb *CommunityDB) processCommunityModify(pin *pin.PinInscription) error {
	isSynced, err := IsPinSynced(pin.Id)
	if err != nil {
		return err
	}
	if isSynced {
		// Already synced, skip processing
		return nil
	}

	// Parse protocol data
	var simpleCommunity protocols.SimpleCommunity
	err = json.Unmarshal(pin.ContentBody, &simpleCommunity)
	if err != nil {
		return err
	}

	// Get existing community info
	existingCommunity, err := cdb.GetCommunityInfoByCommunityId(simpleCommunity.CommunityId)
	if err != nil {
		return err
	}

	if existingCommunity == nil {
		// If community doesn't exist, process as creation
		return cdb.processCommunityCreate(pin)
	}

	if existingCommunity.Address != pin.CreateAddress {
		return errors.New("community creator not match")
	}

	// Update community info
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

	// Save to version info table
	err = cdb.SaveCommunityVersionInfo(existingCommunity)
	if err != nil {
		return err
	}

	// Save to basic info table
	err = cdb.SaveCommunityInfo(existingCommunity)
	if err != nil {
		return err
	}

	// Mark pin as synced
	err = MarkPinAsSynced(pin.Id, true)
	if err != nil {
		return err
	}

	return nil
}
