package db

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"manindexer/basicprotocols/group_chat/models"
	"manindexer/basicprotocols/group_chat/protocols"
	"manindexer/common"
	"manindexer/common/metacontract_util"
	"manindexer/pin"
	"strings"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/wire"
	"github.com/cockroachdb/pebble"
)

// ExtraDB handles extra group chat related data
type ExtraDB struct {
	pb *Pebble
}

// NewExtraDB creates a new ExtraDB instance
func NewExtraDB(pb *Pebble) *ExtraDB {
	return &ExtraDB{pb: pb}
}

// SaveLuckyBagExtra saves lucky bag extra FT information
func (edb *ExtraDB) SaveLuckyBagExtra(extra *models.TalkGroupLuckyBagV3Extra) error {
	data, err := json.Marshal(extra)
	if err != nil {
		return fmt.Errorf("marshal lucky bag extra failed: %v", err)
	}

	// Use TxId as primary key
	key := []byte(extra.TxId)
	err = Pb[TalkGroupLuckyBagExtraFtCollection].Set(key, data, pebble.Sync)
	if err != nil {
		return fmt.Errorf("save lucky bag extra to db failed: %v", err)
	}

	log.Printf("SaveLuckyBagExtra success, txId: %s, groupId: %s", extra.TxId, extra.GroupId)
	return nil
}

// GetLuckyBagExtraByTxId gets lucky bag extra FT information by TxId
func (edb *ExtraDB) GetLuckyBagExtraByTxId(txId string) (*models.TalkGroupLuckyBagV3Extra, error) {
	key := []byte(txId)
	value, closer, err := Pb[TalkGroupLuckyBagExtraFtCollection].Get(key)
	if err != nil {
		if err == pebble.ErrNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("get lucky bag extra from db failed: %v", err)
	}
	defer closer.Close()

	var extra models.TalkGroupLuckyBagV3Extra
	err = json.Unmarshal(value, &extra)
	if err != nil {
		return nil, fmt.Errorf("unmarshal lucky bag extra failed: %v", err)
	}

	return &extra, nil
}

// DeleteLuckyBagExtra deletes lucky bag extra FT information
func (edb *ExtraDB) DeleteLuckyBagExtra(txId string) error {
	key := []byte(txId)
	err := Pb[TalkGroupLuckyBagExtraFtCollection].Delete(key, pebble.Sync)
	if err != nil {
		return fmt.Errorf("delete lucky bag extra from db failed: %v", err)
	}
	return nil
}

// processGroupLuckyBagExtra processes group lucky bag extra FT information
func (edb *ExtraDB) processGroupLuckyBagExtra(pin *pin.PinInscription, txData *wire.MsgTx, isResync bool) error {
	txId := pin.Id[:len(pin.Id)-2]

	// Check if this TxId has already been saved
	existingExtra, err := edb.GetLuckyBagExtraByTxId(txId)
	if err != nil {
		return fmt.Errorf("get existing lucky bag extra failed: %v", err)
	}

	if existingExtra != nil {
		// Already exists, skip processing
		log.Printf("Lucky bag extra already exists, txId: %s", txId)
		return nil
	}

	// Check if already synced
	isSynced, err := IsPinSynced(pin.Id)
	if err != nil {
		return fmt.Errorf("check pin sync status failed: %v", err)
	}
	if isSynced {
		// Already synced, skip processing
		log.Printf("Lucky bag extra already synced, pinId: %s", pin.Id)
		return nil
	}

	// Parse protocol data
	var simpleLuckyBagExtra protocols.SimpleGroupLuckyBagExtra
	err = json.Unmarshal(pin.ContentBody, &simpleLuckyBagExtra)
	if err != nil {
		return fmt.Errorf("unmarshal lucky bag extra protocol failed: %v", err)
	}

	// Convert TokenOutputs
	var (
		tokenOutputs []*models.LuckyBagFtOutput = make([]*models.LuckyBagFtOutput, 0)
		codehash     string
		genesis      string
		genesisId    string
		sensibleId   string           // GenesisTx outpoint
		name         string           // FT name
		symbol       string           // FT symbol
		decimal      uint8            // FT decimal
		netParams    *chaincfg.Params = &chaincfg.MainNetParams
	)
	if common.TestNet == "1" {
		netParams = &chaincfg.TestNet3Params
	} else if common.TestNet == "2" {
		netParams = &chaincfg.RegressionNetParams
	}

	if txData != nil {
		for i, vout := range txData.TxOut {
			ftUtxoInfo, contractTypeStr, err := ParseContractFtInfo(hex.EncodeToString(vout.PkScript), netParams)
			if err != nil {
				return fmt.Errorf("parse contract ft info failed: %v", err)
			}
			if contractTypeStr != "ft" {
				continue
			}
			codehash = ftUtxoInfo.CodeHash
			genesis = ftUtxoInfo.Genesis
			genesisId = ftUtxoInfo.GenesisId
			sensibleId = ftUtxoInfo.SensibleId
			name = ftUtxoInfo.Name
			symbol = ftUtxoInfo.Symbol
			decimal = ftUtxoInfo.Decimal
			tokenOutputs = append(tokenOutputs, &models.LuckyBagFtOutput{
				TokenAmount:  ftUtxoInfo.Amount,
				TokenAddress: ftUtxoInfo.Address,
				Index:        int64(i),
			})
		}
	}

	// Create lucky bag extra information model
	extra := &models.TalkGroupLuckyBagV3Extra{
		TxId:            pin.Id[:len(pin.Id)-2],
		PinId:           pin.Id,
		SubId:           simpleLuckyBagExtra.SubId,
		GroupId:         simpleLuckyBagExtra.GroupId,
		ChannelId:       simpleLuckyBagExtra.ChannelId,
		Code:            simpleLuckyBagExtra.Code,
		CreateTime:      simpleLuckyBagExtra.CreateTime,
		Domain:          simpleLuckyBagExtra.Domain,
		LuckyBagAddress: simpleLuckyBagExtra.LuckyBagAddress,
		Codehash:        codehash,
		Genesis:         genesis,
		GenesisId:       genesisId,
		SensibleId:      sensibleId,
		Name:            name,
		Symbol:          symbol,
		Decimal:         decimal,
		Type:            simpleLuckyBagExtra.Type,
		TokenOutputs:    tokenOutputs,
	}

	// Save to database
	err = edb.SaveLuckyBagExtra(extra)
	if err != nil {
		return fmt.Errorf("save lucky bag extra failed: %v", err)
	}

	// Mark as synced
	err = MarkPinAsSynced(pin.Id, true)
	if err != nil {
		log.Printf("mark pin as synced failed, pinId: %s, error: %v", pin.Id, err)
		// Don't return error because data has been saved successfully
	}

	log.Printf("processGroupLuckyBagExtra success, pinId: %s, groupId: %s, code: %s", pin.Id, extra.GroupId, extra.Code)
	return nil
}

// ProcessGroupLuckyBagExtra public processing method (for external calls)
func (edb *ExtraDB) ProcessGroupLuckyBagExtra(pin *pin.PinInscription, tx interface{}, isResync bool) error {
	var txData *wire.MsgTx
	path := pin.Path
	protocol := strings.Replace(path, "/protocols/", "", -1)
	if strings.EqualFold(protocol, protocols.MonitorSimpleGroupLuckyBagExtra) {
		if tx != nil {
			// Determine tx type, if it's wire.MsgTx or *wire.MsgTx, convert to *wire.MsgTx
			switch txType := tx.(type) {
			case *wire.MsgTx:
				txData = txType
			case wire.MsgTx:
				txData = &txType
			default:
				// Other types, try direct conversion
				if msgTx, ok := tx.(*wire.MsgTx); ok {
					txData = msgTx
				} else if msgTx, ok := tx.(wire.MsgTx); ok {
					txData = &msgTx
				}
			}
		}
	}

	return edb.processGroupLuckyBagExtra(pin, txData, isResync)
}

// GetAllLuckyBagExtras gets all lucky bag extra FT information (for debugging or export)
func (edb *ExtraDB) GetAllLuckyBagExtras() ([]*models.TalkGroupLuckyBagV3Extra, error) {
	iter, err := Pb[TalkGroupLuckyBagExtraFtCollection].NewIter(nil)
	if err != nil {
		return nil, fmt.Errorf("create iterator failed: %v", err)
	}
	defer iter.Close()

	var extras []*models.TalkGroupLuckyBagV3Extra
	for iter.First(); iter.Valid(); iter.Next() {
		var extra models.TalkGroupLuckyBagV3Extra
		err := json.Unmarshal(iter.Value(), &extra)
		if err != nil {
			log.Printf("unmarshal lucky bag extra failed: %v", err)
			continue
		}
		extras = append(extras, &extra)
	}

	if err := iter.Error(); err != nil {
		return nil, fmt.Errorf("iterate lucky bag extras failed: %v", err)
	}

	return extras, nil
}

// GetLuckyBagExtrasByGroupId gets all lucky bag extra FT information by GroupId
func (edb *ExtraDB) GetLuckyBagExtrasByGroupId(groupId string) ([]*models.TalkGroupLuckyBagV3Extra, error) {
	iter, err := Pb[TalkGroupLuckyBagExtraFtCollection].NewIter(nil)
	if err != nil {
		return nil, fmt.Errorf("create iterator failed: %v", err)
	}
	defer iter.Close()

	var extras []*models.TalkGroupLuckyBagV3Extra
	for iter.First(); iter.Valid(); iter.Next() {
		var extra models.TalkGroupLuckyBagV3Extra
		err := json.Unmarshal(iter.Value(), &extra)
		if err != nil {
			log.Printf("unmarshal lucky bag extra failed: %v", err)
			continue
		}

		if extra.GroupId == groupId {
			extras = append(extras, &extra)
		}
	}

	if err := iter.Error(); err != nil {
		return nil, fmt.Errorf("iterate lucky bag extras failed: %v", err)
	}

	return extras, nil
}

// GetLuckyBagExtrasByCode gets lucky bag extra FT information by Code
func (edb *ExtraDB) GetLuckyBagExtrasByCode(code string) ([]*models.TalkGroupLuckyBagV3Extra, error) {
	iter, err := Pb[TalkGroupLuckyBagExtraFtCollection].NewIter(nil)
	if err != nil {
		return nil, fmt.Errorf("create iterator failed: %v", err)
	}
	defer iter.Close()

	var extras []*models.TalkGroupLuckyBagV3Extra
	for iter.First(); iter.Valid(); iter.Next() {
		var extra models.TalkGroupLuckyBagV3Extra
		err := json.Unmarshal(iter.Value(), &extra)
		if err != nil {
			log.Printf("unmarshal lucky bag extra failed: %v", err)
			continue
		}

		if extra.Code == code {
			extras = append(extras, &extra)
		}
	}

	if err := iter.Error(); err != nil {
		return nil, fmt.Errorf("iterate lucky bag extras failed: %v", err)
	}

	return extras, nil
}

// CountLuckyBagExtras counts total number of lucky bag extra FT information
func (edb *ExtraDB) CountLuckyBagExtras() (int64, error) {
	iter, err := Pb[TalkGroupLuckyBagExtraFtCollection].NewIter(nil)
	if err != nil {
		return 0, fmt.Errorf("create iterator failed: %v", err)
	}
	defer iter.Close()

	count := int64(0)
	for iter.First(); iter.Valid(); iter.Next() {
		count++
	}

	if err := iter.Error(); err != nil {
		return 0, fmt.Errorf("count lucky bag extras failed: %v", err)
	}

	return count, nil
}

// ParseFtInfo parses FT information from script
func ParseContractFtInfo(scriptHex string, params *chaincfg.Params) (*metacontract_util.FTUtxoInfo, string, error) {
	scriptBytes, err := hex.DecodeString(scriptHex)
	if err != nil {
		return nil, "", err
	}

	contractTypeStr := ""
	contractType := metacontract_util.GetContractType(scriptBytes)
	if contractType == metacontract_util.ContractTypeFT {
		contractTypeStr = "ft"
		ftUtxoInfo, err := metacontract_util.ExtractFTUtxoInfo(scriptBytes, params)
		if err != nil {
			return nil, "", err
		}
		return ftUtxoInfo, contractTypeStr, nil
	} else {
		contractTypeStr = "unknown"
		return nil, contractTypeStr, nil
	}
}
