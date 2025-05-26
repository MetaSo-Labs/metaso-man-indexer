package pebblestore

import (
	"fmt"
	"manindexer/common"
	"manindexer/pin"
	"strconv"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/cockroachdb/pebble/v2"
)

func (db *Database) GetPinListByIdList(outputList []string, batchSize int, replace bool) (transferCheck []*pin.PinInscription, err error) {
	num := len(transferCheck)
	for i := 0; i < num; i += batchSize {
		end := i + batchSize
		if end > num {
			end = num
		}
		vals := db.BatchGetPinListByKeys(outputList[i:end], replace)
		for _, val := range vals {
			var pinNode pin.PinInscription
			err := sonic.Unmarshal(val, &pinNode)
			if err == nil {
				transferCheck = append(transferCheck, &pinNode)
			}
		}
	}
	return
}
func (db *Database) UpdateTransferPin(trasferMap map[string]*pin.PinTransferInfo) (err error) {
	var updateLit []pin.PinInscription
	for id, info := range trasferMap {
		val, err := db.GetPinByKey(id)
		if err != nil {
			continue
		}
		var pinNode pin.PinInscription
		err = sonic.Unmarshal(val, &pinNode)
		if err != nil {
			continue
		}
		pinNode.IsTransfered = true
		pinNode.Address = info.Address
		pinNode.MetaId = common.GetMetaIdByAddress(info.Address)
		pinNode.Location = info.Location
		pinNode.Offset = info.Offset
		pinNode.Output = info.Output
		pinNode.OutputValue = info.OutputValue
		updateLit = append(updateLit, pinNode)
	}
	if len(updateLit) > 0 {
		err = db.BatchInsertPins(updateLit)
	}
	return
}
func (db *Database) BatchUpdatePins(pins []*pin.PinInscription) (err error) {
	for _, oldPin := range pins {
		if oldPin.OriginalId == "" || oldPin.Status == 0 {
			continue
		}
		dbshard := db.getShard(oldPin.Id)
		val, closer, err := dbshard.Get([]byte(oldPin.Id))
		if err == nil {
			var newPin pin.PinInscription
			err := sonic.Unmarshal(val, &newPin)
			if err == nil {
				newPin.Status = oldPin.Status
			}
			newVal, err := sonic.Marshal(newPin)
			if err == nil {
				dbshard.Set([]byte(newPin.Id), newVal, pebble.Sync)
			}
			closer.Close()
		}
	}
	return
}
func (db *Database) SetAllPins(height int64, pinList []interface{}, batchSize int) (err error) {
	num := len(pinList)
	if num <= 0 {
		return
	}
	list := make([]pin.PinInscription, 0, num)
	keys := make([]string, 0, num)
	//key是 path_blockTime_chainName_height,value是[]pinId
	pathMap := make(map[string][]string)
	// AddressDB: 按地址存储的PIN ID列表，key是address转换后的metaid,value是[]pinId&path&outputValue
	addressMap := make(map[string][]string)
	for _, item := range pinList {
		p := item.(*pin.PinInscription)
		if p == nil {
			continue
		}
		list = append(list, *p)
		keys = append(keys, p.Id)
		if p.Path != "" {
			k := common.ConcatBytesOptimized([]string{p.Path, "&", fmt.Sprint(p.Timestamp), "&", p.ChainName, "&", fmt.Sprint(p.GenesisHeight)}, "")
			pathMap[k] = append(pathMap[k], p.Id)
		}
		if p.MetaId != "" {
			v := common.ConcatBytesOptimized([]string{p.Id, "&", p.Path, "&", fmt.Sprint(p.OutputValue)}, "")
			addressMap[p.MetaId] = append(addressMap[p.MetaId], v)
		}
	}

	blockTime := list[0].Timestamp
	chainName := list[0].ChainName
	for i := 0; i < num; i += batchSize {
		end := i + batchSize
		if end > num {
			end = num
		}
		err = db.BatchInsertPins(list[i:end])
		if err != nil {
			fmt.Printf("插入区块%d第%d~%d条失败: %v\n", height, i, end, err)
		}
	}
	list = list[:0]
	//InsertPageInfo
	page := PageInfo{
		ChainName:   chainName,
		BlockTime:   blockTime,
		BlockHeight: height,
		Type:        "pin",
		Num:         num,
		Keys:        keys,
	}
	err = db.InsertPageInfo(db.PagesDB, page)
	if err != nil {
		fmt.Printf("InsertPageInfo err: %v", err)
	}
	if len(pathMap) > 0 {
		pathData := make(map[string]string)
		for k, v := range pathMap {
			pathData[k] = "," + strings.Join(v, ",")
		}
		db.BatchInsertPathPins(pathData)
	}
	if len(addressMap) > 0 {
		addressData := make(map[string]string)
		for k, v := range addressMap {
			addressData[k] = "," + strings.Join(v, ",")
		}
		db.BatchMergeAddressData(addressData)
	}
	return
}
func (db *Database) CountSet(key string, value int64) (err error) {
	return db.CountDB.Set([]byte(key), []byte(strconv.FormatInt(value, 10)), pebble.Sync)
}
func (db *Database) CountAdd(key string, value int64) (err error) {
	val, closer, err := db.CountDB.Get([]byte(key))
	if err != nil {
		if err == pebble.ErrNotFound {
			db.CountSet(key, value)
			return
		} else {
			return
		}
	}
	defer closer.Close()
	old, err := strconv.ParseInt(string(val), 10, 64)
	if err != nil {
		return
	}
	return db.CountDB.Set([]byte(key), []byte(strconv.FormatInt(old+value, 10)), pebble.Sync)
}
