package pebblestore

import (
	"fmt"
	"manindexer/common"
	"manindexer/pin"
	"strconv"
	"strings"
	"time"

	"github.com/bytedance/sonic"
	"github.com/cockroachdb/pebble"
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
func (db *Database) SetAllPins_BAK(height int64, pinList []interface{}, batchSize int) (err error) {
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

func (db *Database) SetAllPins(height int64, pinList []interface{}, batchSize int) (err error) {
	num := len(pinList)
	if num <= 0 {
		return
	}
	first := pinList[0].(*pin.PinInscription)
	chainName := first.ChainName
	blockTime := first.Timestamp
	publicKeyStr := common.ConcatBytesOptimized([]string{fmt.Sprintf("%010d", blockTime), "&", chainName, "&", fmt.Sprintf("%010d", height)}, "")
	keys := make([]string, 0, num)
	pinSortkeys := make([]string, 0, num)
	for i := 0; i < num; i += batchSize {
		end := i + batchSize
		if end > num {
			end = num
		}
		batch := pinList[i:end]

		// 处理本批数据
		list := make([]pin.PinInscription, 0, len(batch))
		pathMap := make(map[string][]string)
		addressMap := make(map[string][]string)

		for _, item := range batch {
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

		// 批量插入/处理
		if len(list) > 0 {
			st := time.Now()
			err = db.BatchInsertPins(list)
			if err != nil {
				fmt.Printf("插入区块PIN失败: %v\n", err)
			}
			fmt.Println("  >BatchInsertPins:", time.Since(st))
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
		// 本批处理完后，keys等会被GC回收
		list = list[:0]
		pathMap = make(map[string][]string)
		addressMap = make(map[string][]string)
	}
	db.InsertPinSort(db.PinSort, pinSortkeys)
	db.InsertBlockTxs(publicKeyStr, strings.Join(keys, ","))
	keys = nil
	pinSortkeys = nil
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

func (db *Database) SetMempool(key string, value []byte) error {
	return db.PinsMempoolDb.Set([]byte(key), value, pebble.Sync)
}
func (db *Database) GetMempool(key string) ([]byte, error) {
	result, closer, err := db.PinsMempoolDb.Get([]byte(key))
	if err != nil {
		if err == pebble.ErrNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("GetMempool error: %v", err)
	}
	defer closer.Close()
	return result, nil
}
func (db *Database) GetMempoolPin(key string) (pinNode pin.PinInscription, err error) {
	result, err := db.GetMempool(key)
	if err != nil {
		return
	}
	err = sonic.Unmarshal(result, &pinNode)
	return
}
func (db *Database) DeleteMempool(key string) error {
	return db.PinsMempoolDb.Delete([]byte(key), pebble.Sync)
}
func (db *Database) BatchDeleteMempool(key []string) error {
	batch := db.PinsMempoolDb.NewBatch()
	for _, v := range key {
		batch.Delete([]byte(v), nil)
	}
	if err := batch.Commit(pebble.Sync); err != nil {
		batch.Close()
		return err
	}
	batch.Close()
	return nil
}
func (db *Database) SetNotifcation(key string, value []byte) error {
	sep := []byte("@*@")
	return db.NotifcationDb.Merge([]byte(key), append(value, sep...), pebble.Sync)
}
func (db *Database) GetNotifcation(key string) ([]byte, error) {
	result, closer, err := db.NotifcationDb.Get([]byte(key))
	if err != nil {
		if err == pebble.ErrNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("GetNotifcation error: %v", err)
	}
	defer closer.Close()
	return result, nil
}
func (db *Database) DeleteNotifcation(key string) error {
	return db.NotifcationDb.Delete([]byte(key), pebble.Sync)
}

func (db *Database) CleanUpNotifcation(key string) error {
	result, err := db.GetNotifcation(key)
	if err != nil {
		return err
	}

	// 分割数据
	arr := strings.Split(string(result), "@*@")

	// 如果数据大于300条，滚动删除
	if len(arr) > 300 {
		remaining := arr[len(arr)-200:] // 保留最后200条
		newValue := strings.Join(remaining, "@*@")
		return db.NotifcationDb.Set([]byte(key), []byte(newValue), pebble.Sync)
	}

	return nil
}
