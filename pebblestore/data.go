package pebblestore

import (
	"fmt"
	"manindexer/common"
	"manindexer/pin"
	"strconv"
	"strings"
	"time"

	"github.com/bytedance/sonic"
	"github.com/cockroachdb/pebble/v2"
)

func (db *Database) GetPinListByIdList(outputList []string, batchSize int, replace bool) (transferCheck []*pin.PinInscription, err error) {
	// num := len(transferCheck)
	// for i := 0; i < num; i += batchSize {
	// 	end := i + batchSize
	// 	if end > num {
	// 		end = num
	// 	}
	// 	vals := db.BatchGetPinListByKeys(outputList[i:end], replace)
	// 	for _, val := range vals {
	// 		var pinNode pin.PinInscription
	// 		err := sonic.Unmarshal(val, &pinNode)
	// 		if err == nil {
	// 			transferCheck = append(transferCheck, &pinNode)
	// 		}
	// 	}
	// }
	vals := db.BatchGetPinListByKeys(outputList, replace)
	for _, val := range vals {
		var pinNode pin.PinInscription
		err := sonic.Unmarshal(val, &pinNode)
		if err == nil {
			transferCheck = append(transferCheck, &pinNode)
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
	pinSortkeys := make([]string, 0, num)
	//key是 path_blockTime_chainName_height,value是[]pinId
	pathMap := make(map[string][]string)
	// AddressDB: 按地址存储的PIN ID列表，key是address转换后的metaid,value是[]pinId&path&outputValue
	addressMap := make(map[string][]string)
	first := pinList[0].(*pin.PinInscription)
	chainName := first.ChainName
	blockTime := first.Timestamp
	//fixedHeight := common.ConcatBytesOptimized([]string{fmt.Sprintf("%010d", height), "&", chainName}, "")
	publicKeyStr := common.ConcatBytesOptimized([]string{fmt.Sprintf("%010d", blockTime), "&", chainName, "&", fmt.Sprintf("%010d", height)}, "")
	for _, item := range pinList {
		p := item.(*pin.PinInscription)
		if p == nil {
			continue
		}
		list = append(list, *p)
		keys = append(keys, p.Id)
		sortKey := common.ConcatBytesOptimized([]string{publicKeyStr, "&", p.Id}, "")
		pinSortkeys = append(pinSortkeys, sortKey)
		if p.Path != "" {
			k := common.ConcatBytesOptimized([]string{p.Path, "&", publicKeyStr}, "")
			pathMap[k] = append(pathMap[k], p.Id)
		}
		if p.MetaId != "" {
			v := common.ConcatBytesOptimized([]string{p.Id, "&", p.Path, "&", fmt.Sprint(p.OutputValue)}, "")
			addressMap[p.MetaId] = append(addressMap[p.MetaId], v)
		}
	}
	// for i := 0; i < num; i += batchSize {
	// 	end := i + batchSize
	// 	if end > num {
	// 		end = num
	// 	}
	// 	err = db.BatchInsertPins(list[i:end])
	// 	if err != nil {
	// 		fmt.Printf("插入区块%d第%d~%d条失败: %v\n", height, i, end, err)
	// 	}
	// }
	st := time.Now()
	err = db.BatchInsertPins(list)
	fmt.Println("  >BatchInsertPins:", time.Since(st))
	if err != nil {
		fmt.Printf("插入区块PIN%d失败: %v\n", height, err)
	}
	list = list[:0]
	//Insert Pins sort
	st = time.Now()
	db.InsertPinSort(db.PinSort, pinSortkeys)
	fmt.Println("  >InsertPinSort:", time.Since(st))
	st = time.Now()
	pinSortkeys = pinSortkeys[:0]
	db.InsertBlockTxs(publicKeyStr, strings.Join(keys, ","))
	fmt.Println("  >InsertBlockTxs:", time.Since(st))
	st = time.Now()

	keys = keys[:0]
	if len(pathMap) > 0 {
		pathData := make(map[string]string)
		for k, v := range pathMap {
			pathData[k] = "," + strings.Join(v, ",")
		}
		db.BatchInsertPathPins(pathData)
	}
	fmt.Println("  >BatchInsertPathPins:", time.Since(st))
	st = time.Now()
	if len(addressMap) > 0 {
		addressData := make(map[string]string)
		for k, v := range addressMap {
			addressData[k] = "," + strings.Join(v, ",")
		}
		db.BatchMergeAddressData(addressData)
		fmt.Println("  >BatchMergeAddressData:", time.Since(st))
	}
	return
}
func (db *Database) CountSet(key string, value int64) (err error) {
	return db.CountDB.Set([]byte(key), []byte(strconv.FormatInt(value, 10)), pebble.Sync)
}
func (db *Database) CountAdd(key string, value int64) error {
	val, closer, err := db.CountDB.Get([]byte(key))
	if err == pebble.ErrNotFound {
		return db.CountDB.Set([]byte(key), []byte(strconv.FormatInt(value, 10)), pebble.Sync)
	} else if err != nil {
		return err
	}
	old, err := strconv.ParseInt(string(val), 10, 64)
	closer.Close()
	if err != nil {
		return err
	}
	return db.CountDB.Set([]byte(key), []byte(strconv.FormatInt(old+value, 10)), pebble.Sync)
}

// func (db *Database) CountAdd(key string, value int64) (err error) {
// 	val, closer, err := db.CountDB.Get([]byte(key))
// 	if err != nil {
// 		if err == pebble.ErrNotFound {
// 			db.CountSet(key, value)
// 			return
// 		} else {
// 			return
// 		}
// 	}
// 	defer closer.Close()
// 	old, err := strconv.ParseInt(string(val), 10, 64)
// 	if err != nil {
// 		return
// 	}
// 	return db.CountDB.Set([]byte(key), []byte(strconv.FormatInt(old+value, 10)), pebble.Sync)
// }
