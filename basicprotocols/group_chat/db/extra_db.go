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

	pinIdStr := strings.Split(pin.Id, "i")
	if len(pinIdStr) < 2 {
		return fmt.Errorf("invalid pin id: %s", pin.Id)
	}
	t := pinIdStr[0]

	// Create lucky bag extra information model
	extra := &models.TalkGroupLuckyBagV3Extra{
		TxId:            t,
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

// SaveLuckyBagManualExtraGas saves lucky bag manual extra gas information
func (edb *ExtraDB) SaveLuckyBagManualExtraGas(extra *models.TalkGroupLuckyBagV3ManualExtraGas) error {
	data, err := json.Marshal(extra)
	if err != nil {
		return fmt.Errorf("marshal lucky bag manual extra gas failed: %v", err)
	}

	// Use LuckyBagTxId as primary key
	key := []byte(extra.LuckyBagPinId)
	err = Pb[TalkGroupLuckyBagManualExtraGasCollection].Set(key, data, pebble.Sync)
	if err != nil {
		return fmt.Errorf("save lucky bag manual extra gas to db failed: %v", err)
	}

	log.Printf("SaveLuckyBagManualExtraGas success, luckyBagPinId: %s, txId: %s", extra.LuckyBagPinId, extra.TxId)
	return nil
}

// GetLuckyBagManualExtraGasByLuckyBagTxId gets lucky bag manual extra gas information by LuckyBagTxId
func (edb *ExtraDB) GetLuckyBagManualExtraGasByLuckyBagPinId(luckyBagPinId string) (*models.TalkGroupLuckyBagV3ManualExtraGas, error) {
	key := []byte(luckyBagPinId)
	value, closer, err := Pb[TalkGroupLuckyBagManualExtraGasCollection].Get(key)
	if err != nil {
		if err == pebble.ErrNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("get lucky bag manual extra gas from db failed: %v", err)
	}
	defer closer.Close()

	var extra models.TalkGroupLuckyBagV3ManualExtraGas
	err = json.Unmarshal(value, &extra)
	if err != nil {
		return nil, fmt.Errorf("unmarshal lucky bag manual extra gas failed: %v", err)
	}

	return &extra, nil
}

// DeleteLuckyBagManualExtraGas deletes lucky bag manual extra gas information
func (edb *ExtraDB) DeleteLuckyBagManualExtraGas(luckyBagTxId string) error {
	key := []byte(luckyBagTxId)
	err := Pb[TalkGroupLuckyBagManualExtraGasCollection].Delete(key, pebble.Sync)
	if err != nil {
		return fmt.Errorf("delete lucky bag manual extra gas from db failed: %v", err)
	}
	return nil
}

// getLuckyBagByPinId gets lucky bag information by pinId
func (edb *ExtraDB) GetLuckyBagByPinId(pinId string) (*models.TalkGroupLuckyBagV3, error) {
	key := []byte(pinId)
	value, closer, err := Pb[TalkGroupLuckyBagPinCollection].Get(key)
	if err != nil {
		if err == pebble.ErrNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("get lucky bag by pinId failed: %v", err)
	}
	defer closer.Close()

	var luckyBag models.TalkGroupLuckyBagV3
	err = json.Unmarshal(value, &luckyBag)
	if err != nil {
		return nil, fmt.Errorf("unmarshal lucky bag failed: %v", err)
	}

	return &luckyBag, nil
}

// buildGasOutputs builds gas outputs based on payList count
func (edb *ExtraDB) BuildGasOutputs(payList []*models.ProInfoPayList, perAmount uint64) []*models.LuckyBagGasOutput {
	gasOutputs := make([]*models.LuckyBagGasOutput, 0, len(payList))

	// Calculate amount per output
	for i, payInfo := range payList {

		gasOutput := &models.LuckyBagGasOutput{
			GasAmount:  perAmount,
			GasAddress: payInfo.GasAddress,
			GasIndex:   int64(i),
		}
		gasOutputs = append(gasOutputs, gasOutput)
	}

	return gasOutputs
}

// // GetAllLuckyBagManualExtraGases gets all lucky bag manual extra gas information
// func (edb *ExtraDB) GetAllLuckyBagManualExtraGases() ([]*models.TalkGroupLuckyBagV3ManualExtraGas, error) {
// 	iter, err := Pb[TalkGroupLuckyBagManualExtraGasCollection].NewIter(nil)
// 	if err != nil {
// 		return nil, fmt.Errorf("create iterator failed: %v", err)
// 	}
// 	defer iter.Close()

// 	var extras []*models.TalkGroupLuckyBagV3ManualExtraGas
// 	for iter.First(); iter.Valid(); iter.Next() {
// 		var extra models.TalkGroupLuckyBagV3ManualExtraGas
// 		err := json.Unmarshal(iter.Value(), &extra)
// 		if err != nil {
// 			log.Printf("unmarshal lucky bag manual extra gas failed: %v", err)
// 			continue
// 		}
// 		extras = append(extras, &extra)
// 	}

// 	if err := iter.Error(); err != nil {
// 		return nil, fmt.Errorf("iterate lucky bag manual extra gases failed: %v", err)
// 	}

// 	return extras, nil
// }

// CountLuckyBagManualExtraGases counts total number of lucky bag manual extra gas information
func (edb *ExtraDB) CountLuckyBagManualExtraGases() (int64, error) {
	iter, err := Pb[TalkGroupLuckyBagManualExtraGasCollection].NewIter(nil)
	if err != nil {
		return 0, fmt.Errorf("create iterator failed: %v", err)
	}
	defer iter.Close()

	count := int64(0)
	for iter.First(); iter.Valid(); iter.Next() {
		count++
	}

	if err := iter.Error(); err != nil {
		return 0, fmt.Errorf("count lucky bag manual extra gases failed: %v", err)
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
