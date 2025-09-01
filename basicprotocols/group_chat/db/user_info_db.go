package db

import (
	"encoding/json"
	"fmt"
	"manindexer/basicprotocols/group_chat/models"
	"manindexer/basicprotocols/group_chat/protocols"
	"manindexer/pin"
	"sort"
	"strings"

	"github.com/cockroachdb/pebble"
)

// UserInfo database operations
type UserInfoDB struct {
	pb *Pebble
}

func NewUserInfoDB(pb *Pebble) *UserInfoDB {
	return &UserInfoDB{pb: pb}
}

// Save user info by address
func (udb *UserInfoDB) SaveUserInfoByAddress(address string, userInfo *models.UserInfo) error {
	data, err := json.Marshal(userInfo)
	if err != nil {
		return err
	}

	// Use address as primary key
	key := []byte(address)
	return Pb[TalkUserAddressChatPublicKeyCollection].Set(key, data, pebble.Sync)
}

// Get user info by address
func (udb *UserInfoDB) GetUserInfoByAddress(address string) (*models.UserInfo, error) {
	key := []byte(address)
	value, closer, err := Pb[TalkUserAddressChatPublicKeyCollection].Get(key)
	if err != nil {
		if err == pebble.ErrNotFound {
			return nil, nil
		}
		return nil, err
	}
	defer closer.Close()

	var userInfo models.UserInfo
	err = json.Unmarshal(value, &userInfo)
	if err != nil {
		return nil, err
	}

	return &userInfo, nil
}

// Save user info by MetaId
func (udb *UserInfoDB) SaveUserInfoByMetaId(metaId string, userInfo *models.UserInfo) error {
	data, err := json.Marshal(userInfo)
	if err != nil {
		return err
	}

	// Use metaId as primary key
	key := []byte(metaId)
	return Pb[TalkUserMetaIdChatPublicKeyCollection].Set(key, data, pebble.Sync)
}

// Get user info by MetaId
func (udb *UserInfoDB) GetUserInfoByMetaId(metaId string) (*models.UserInfo, error) {
	key := []byte(metaId)
	value, closer, err := Pb[TalkUserMetaIdChatPublicKeyCollection].Get(key)
	if err != nil {
		if err == pebble.ErrNotFound {
			return nil, nil
		}
		return nil, err
	}
	defer closer.Close()

	var userInfo models.UserInfo
	err = json.Unmarshal(value, &userInfo)
	if err != nil {
		return nil, err
	}

	return &userInfo, nil
}

// Get all user info by address (with pagination)
func (udb *UserInfoDB) GetAllUserInfoByAddress(page, size int64) ([]*models.UserInfo, error) {
	var userInfos []*models.UserInfo
	iter, err := Pb[TalkUserAddressChatPublicKeyCollection].NewIter(nil)
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

		if int64(len(userInfos)) >= size {
			break
		}

		var userInfo models.UserInfo
		err := json.Unmarshal(iter.Value(), &userInfo)
		if err != nil {
			continue
		}
		userInfos = append(userInfos, &userInfo)
	}

	return userInfos, nil
}

// Get all user info by MetaId (with pagination)
func (udb *UserInfoDB) GetAllUserInfoByMetaId(page, size int64) ([]*models.UserInfo, error) {
	var userInfos []*models.UserInfo
	iter, err := Pb[TalkUserMetaIdChatPublicKeyCollection].NewIter(nil)
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

		if int64(len(userInfos)) >= size {
			break
		}

		var userInfo models.UserInfo
		err := json.Unmarshal(iter.Value(), &userInfo)
		if err != nil {
			continue
		}
		userInfos = append(userInfos, &userInfo)
	}

	return userInfos, nil
}

// Delete user info by address
func (udb *UserInfoDB) DeleteUserInfoByAddress(address string) error {
	key := []byte(address)
	return Pb[TalkUserAddressChatPublicKeyCollection].Delete(key, pebble.Sync)
}

// Delete user info by MetaId
func (udb *UserInfoDB) DeleteUserInfoByMetaId(metaId string) error {
	key := []byte(metaId)
	return Pb[TalkUserMetaIdChatPublicKeyCollection].Delete(key, pebble.Sync)
}

// Update user info by address
func (udb *UserInfoDB) UpdateUserInfoByAddress(address string, userInfo *models.UserInfo) error {
	return udb.SaveUserInfoByAddress(address, userInfo)
}

// Update user info by MetaId
func (udb *UserInfoDB) UpdateUserInfoByMetaId(metaId string, userInfo *models.UserInfo) error {
	return udb.SaveUserInfoByMetaId(metaId, userInfo)
}

// Get user info count by address
func (udb *UserInfoDB) GetUserInfoCountByAddress() (int64, error) {
	iter, err := Pb[TalkUserAddressChatPublicKeyCollection].NewIter(nil)
	if err != nil {
		return 0, err
	}
	defer iter.Close()

	count := int64(0)
	for iter.First(); iter.Valid(); iter.Next() {
		count++
	}

	return count, nil
}

// Get user info count by MetaId
func (udb *UserInfoDB) GetUserInfoCountByMetaId() (int64, error) {
	iter, err := Pb[TalkUserMetaIdChatPublicKeyCollection].NewIter(nil)
	if err != nil {
		return 0, err
	}
	defer iter.Close()

	count := int64(0)
	for iter.First(); iter.Valid(); iter.Next() {
		count++
	}

	return count, nil
}

// Search user info by address prefix
func (udb *UserInfoDB) SearchUserInfoByAddressPrefix(prefix string, limit int) ([]*models.UserInfo, error) {
	var userInfos []*models.UserInfo
	iter, err := Pb[TalkUserAddressChatPublicKeyCollection].NewIter(&pebble.IterOptions{
		LowerBound: []byte(prefix),
		UpperBound: append([]byte(prefix), 0xff),
	})
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	count := 0
	for iter.First(); iter.Valid() && count < limit; iter.Next() {
		var userInfo models.UserInfo
		err := json.Unmarshal(iter.Value(), &userInfo)
		if err != nil {
			continue
		}
		userInfos = append(userInfos, &userInfo)
		count++
	}

	return userInfos, nil
}

// Search user info by MetaId prefix
func (udb *UserInfoDB) SearchUserInfoByMetaIdPrefix(prefix string, limit int) ([]*models.UserInfo, error) {
	var userInfos []*models.UserInfo
	iter, err := Pb[TalkUserMetaIdChatPublicKeyCollection].NewIter(&pebble.IterOptions{
		LowerBound: []byte(prefix),
		UpperBound: append([]byte(prefix), 0xff),
	})
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	count := 0
	for iter.First(); iter.Valid() && count < limit; iter.Next() {
		var userInfo models.UserInfo
		err := json.Unmarshal(iter.Value(), &userInfo)
		if err != nil {
			continue
		}
		userInfos = append(userInfos, &userInfo)
		count++
	}

	return userInfos, nil
}

// Batch save user info by address
func (udb *UserInfoDB) BatchSaveUserInfoByAddress(userInfos map[string]*models.UserInfo) error {
	batch := Pb[TalkUserAddressChatPublicKeyCollection].NewBatch()
	defer batch.Close()

	for address, userInfo := range userInfos {
		data, err := json.Marshal(userInfo)
		if err != nil {
			return err
		}
		key := []byte(address)
		batch.Set(key, data, nil)
	}

	return batch.Commit(pebble.Sync)
}

// Batch save user info by MetaId
func (udb *UserInfoDB) BatchSaveUserInfoByMetaId(userInfos map[string]*models.UserInfo) error {
	batch := Pb[TalkUserMetaIdChatPublicKeyCollection].NewBatch()
	defer batch.Close()

	for metaId, userInfo := range userInfos {
		data, err := json.Marshal(userInfo)
		if err != nil {
			return err
		}
		key := []byte(metaId)
		batch.Set(key, data, nil)
	}

	return batch.Commit(pebble.Sync)
}

// Save user info history
func (udb *UserInfoDB) SaveUserInfoHistory(key string, history []*models.UserInfo, isAddress bool) error {
	data, err := json.Marshal(history)
	if err != nil {
		return err
	}

	keyBytes := []byte(key)
	var collection string
	if isAddress {
		collection = TalkUserAddressChatPublicKeyCollection
	} else {
		collection = TalkUserMetaIdChatPublicKeyCollection
	}

	return Pb[collection].Set(keyBytes, data, pebble.Sync)
}

// Get user info history
func (udb *UserInfoDB) GetUserInfoHistory(key string, isAddress bool) ([]*models.UserInfo, error) {
	keyBytes := []byte(key)
	var collection string
	if isAddress {
		collection = TalkUserAddressChatPublicKeyCollection
	} else {
		collection = TalkUserMetaIdChatPublicKeyCollection
	}

	value, closer, err := Pb[collection].Get(keyBytes)
	if err != nil {
		if err == pebble.ErrNotFound {
			return []*models.UserInfo{}, nil
		}
		return nil, err
	}
	defer closer.Close()

	var history []*models.UserInfo
	err = json.Unmarshal(value, &history)
	if err != nil {
		return nil, err
	}

	return history, nil
}

// Process user info pin (for future use if needed)
func (udb *UserInfoDB) ProcessUserInfoPin(pin *pin.PinInscription) error {
	switch pin.Operation {
	case "create":
		if strings.HasPrefix(pin.Path, "/info/") {
			infoNode := strings.Replace(pin.Path, "/info/", "", -1)
			if strings.ToLower(infoNode) == strings.ToLower(protocols.MonitorInfoChatpubkey) {
				fmt.Printf("[UserInfoDB] ProcessUserInfoPin create: %+v\n", pin)
				return udb.processUserInfoCreate(pin)
			}
		} else {
			return nil
		}
	case "modify":
		if strings.HasPrefix(pin.Path, "/info/") {
			infoNode := strings.Replace(pin.Path, "/info/", "", -1)
			if strings.ToLower(infoNode) == strings.ToLower(protocols.MonitorInfoChatpubkey) {
				fmt.Printf("[UserInfoDB] ProcessUserInfoPin modify: %+v\n", pin)
				return udb.processUserInfoModify(pin)
			}
		} else {
			return nil
		}
	}
	return nil
}

func (udb *UserInfoDB) processUserInfoCreate(pin *pin.PinInscription) error {
	fmt.Printf("[UserInfoDB] processUserInfoCreate: %+v\n", pin)
	// Get existing history by address (use address as primary check)
	addressHistory, err := udb.GetUserInfoHistory(pin.CreateAddress, true)
	if err != nil {
		if err == pebble.ErrNotFound {
			addressHistory = []*models.UserInfo{}
		} else {
			return err
		}
	}

	// Check if there's already a create operation
	hasCreateOperation := false
	for _, entry := range addressHistory {
		if entry.Operation == "create" {
			hasCreateOperation = true
			break
		}
	}

	// Check if this PinId already exists in the history
	pinIdExists := false
	for i, entry := range addressHistory {
		if entry.ChatPublicKeyId == pin.Id {
			// If blockHeight is different, update the existing entry
			if entry.BlockHeight != pin.GenesisHeight {
				addressHistory[i].BlockHeight = pin.GenesisHeight
				addressHistory[i].Timestamp = pin.Timestamp
				addressHistory[i].ChatPublicKey = string(pin.ContentBody)
			}
			pinIdExists = true
			break
		}
	}

	// If PinId already exists and blockHeight is the same, skip processing
	if pinIdExists {
		// Sort by timestamp in descending order (newest first)
		sort.Slice(addressHistory, func(i, j int) bool {
			return addressHistory[i].Timestamp > addressHistory[j].Timestamp
		})

		// Save to both collections with the same value
		err = udb.SaveUserInfoHistory(pin.CreateAddress, addressHistory, true)
		if err != nil {
			return err
		}

		err = udb.SaveUserInfoHistory(pin.CreateMetaId, addressHistory, false)
		if err != nil {
			return err
		}

		return nil
	}

	// Create new history entry
	newEntry := &models.UserInfo{
		MetaId:          pin.CreateMetaId,
		Address:         pin.CreateAddress,
		ChatPublicKey:   string(pin.ContentBody), // Direct string conversion
		ChatPublicKeyId: pin.Id,                  // Use PinId as ChatPublicKeyId
		Timestamp:       pin.Timestamp,
		BlockHeight:     pin.GenesisHeight,
		Chain:           pin.ChainName,
		Operation:       "create",
		IsValid:         !hasCreateOperation, // If create operation already exists, set isValid to false
	}

	// Add to address history
	addressHistory = append(addressHistory, newEntry)

	// Sort by timestamp in descending order (newest first)
	sort.Slice(addressHistory, func(i, j int) bool {
		return addressHistory[i].Timestamp > addressHistory[j].Timestamp
	})

	// Save to both collections with the same value
	err = udb.SaveUserInfoHistory(pin.CreateAddress, addressHistory, true)
	if err != nil {
		return err
	}

	err = udb.SaveUserInfoHistory(pin.CreateMetaId, addressHistory, false)
	if err != nil {
		return err
	}

	return nil
}

func (udb *UserInfoDB) processUserInfoModify(pin *pin.PinInscription) error {
	// Get existing history by address (use address as primary check)
	addressHistory, err := udb.GetUserInfoHistory(pin.CreateAddress, true)
	if err != nil {
		return err
	}

	// Check if there's a create operation first
	hasCreateOperation := false
	for _, entry := range addressHistory {
		if entry.Operation == "create" {
			hasCreateOperation = true
			break
		}
	}

	// If no create operation exists, modify is invalid
	if !hasCreateOperation {
		return nil // Skip processing, modify is invalid without create
	}

	// Check if this PinId already exists in the history
	pinIdExists := false
	for i, entry := range addressHistory {
		if entry.ChatPublicKeyId == pin.Id {
			// If blockHeight is different, update the existing entry
			if entry.BlockHeight != pin.GenesisHeight {
				addressHistory[i].BlockHeight = pin.GenesisHeight
				addressHistory[i].Timestamp = pin.Timestamp
				addressHistory[i].ChatPublicKey = string(pin.ContentBody)
			}
			pinIdExists = true
			break
		}
	}

	// If PinId already exists and blockHeight is the same, skip processing
	if pinIdExists {
		// Sort by timestamp in descending order (newest first)
		sort.Slice(addressHistory, func(i, j int) bool {
			return addressHistory[i].Timestamp > addressHistory[j].Timestamp
		})

		// Save to both collections with the same value
		err = udb.SaveUserInfoHistory(pin.CreateAddress, addressHistory, true)
		if err != nil {
			return err
		}

		err = udb.SaveUserInfoHistory(pin.CreateMetaId, addressHistory, false)
		if err != nil {
			return err
		}

		return nil
	}

	// Create new history entry
	newEntry := &models.UserInfo{
		MetaId:          pin.CreateMetaId,
		Address:         pin.CreateAddress,
		ChatPublicKey:   string(pin.ContentBody), // Direct string conversion
		ChatPublicKeyId: pin.Id,                  // Use PinId as ChatPublicKeyId
		Timestamp:       pin.Timestamp,
		BlockHeight:     pin.GenesisHeight,
		Chain:           pin.ChainName,
		Operation:       "modify",
		IsValid:         true,
	}

	// Add to address history
	addressHistory = append(addressHistory, newEntry)

	// Sort by timestamp in descending order (newest first)
	sort.Slice(addressHistory, func(i, j int) bool {
		return addressHistory[i].Timestamp > addressHistory[j].Timestamp
	})

	// Save to both collections with the same value
	err = udb.SaveUserInfoHistory(pin.CreateAddress, addressHistory, true)
	if err != nil {
		return err
	}

	err = udb.SaveUserInfoHistory(pin.CreateMetaId, addressHistory, false)
	if err != nil {
		return err
	}

	return nil
}

// GetLatestValidUserInfo Get the latest valid UserInfo by address or metaId
func (udb *UserInfoDB) GetLatestValidUserInfo(key string, isAddress bool) (*models.UserInfo, error) {
	// Get user info history
	history, err := udb.GetUserInfoHistory(key, isAddress)
	if err != nil {
		return nil, err
	}
	fmt.Printf("[UserInfoDB] GetLatestValidUserInfo history: %+v\n", history)

	if len(history) == 0 {
		return nil, nil // No history found
	}

	// Find the latest valid UserInfo
	var latestValidUserInfo *models.UserInfo
	var latestTimestamp int64 = 0

	for _, userInfo := range history {
		// Check if this UserInfo is valid
		if userInfo.IsValid {
			// Check if this is the latest one
			if userInfo.Timestamp > latestTimestamp {
				latestTimestamp = userInfo.Timestamp
				latestValidUserInfo = userInfo
			}
		}
	}

	return latestValidUserInfo, nil
}

// GetLatestValidUserInfoByAddress Get the latest valid UserInfo by address
func (udb *UserInfoDB) GetLatestValidUserInfoByAddress(address string) (*models.UserInfo, error) {
	return udb.GetLatestValidUserInfo(address, true)
}

// GetLatestValidUserInfoByMetaId Get the latest valid UserInfo by metaId
func (udb *UserInfoDB) GetLatestValidUserInfoByMetaId(metaId string) (*models.UserInfo, error) {
	return udb.GetLatestValidUserInfo(metaId, false)
}
