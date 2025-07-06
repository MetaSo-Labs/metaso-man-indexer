package metaso

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"manindexer/common"
	"manindexer/database/mongodb"
	"manindexer/man"
	"manindexer/pin"
	"os"
	"strconv"
	"time"

	"github.com/bytedance/sonic"
	"github.com/cockroachdb/pebble"
	"github.com/shopspring/decimal"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type PevPebbledb struct {
	Database *Database
}
type Database struct {
	PevDataDB  *pebble.DB
	PevCountDB *pebble.DB
}
type customLogger struct{}

func (l *customLogger) Infof(format string, args ...interface{})  {}
func (l *customLogger) Fatalf(format string, args ...interface{}) {}
func (l *customLogger) Errorf(format string, args ...interface{}) {}

var noopLogger = &customLogger{}

func (pb *PevPebbledb) NewDataBase(basePath string) error {
	os.MkdirAll(fmt.Sprintf("%s/data", basePath), 0755)
	pevDataDB, err := pebble.Open(fmt.Sprintf("%s/data/db", basePath), &pebble.Options{Logger: noopLogger})
	if err != nil {
		return err
	}
	os.MkdirAll(fmt.Sprintf("%s/count", basePath), 0755)
	pevCountDB, err := pebble.Open(fmt.Sprintf("%s/count/db", basePath), &pebble.Options{Logger: noopLogger})
	if err != nil {
		return err
	}
	pb.Database = &Database{PevDataDB: pevDataDB, PevCountDB: pevCountDB}
	return nil
}

// Close 关闭所有数据库
func (idx *Database) Close() error {
	idx.PevDataDB.Close()
	idx.PevCountDB.Close()
	return nil
}

func (pb *PevPebbledb) GetPevDataByMetaBlock(blockHeight int64) (pevList []PEVData, err error) {
	key := fmt.Sprintf("block%d", blockHeight)
	val, closer, err := pb.Database.PevDataDB.Get([]byte(key))
	if err != nil {
		return
	}
	defer closer.Close()
	err = sonic.Unmarshal(val, &pevList)
	return
}
func (pb *PevPebbledb) SaveBlockPevData(blockHeight int64, pevList *[]PEVData) (err error) {
	key := fmt.Sprintf("block%d_%d", blockHeight, time.Now().Unix())
	val, err := sonic.Marshal(pevList)
	if err != nil {
		return fmt.Errorf("failed to marshal pevList: %w", err)
	}
	return pb.Database.PevDataDB.Set([]byte(key), val, pebble.Sync)
}

func (pb *PevPebbledb) CountBlockPEV(blockHeight int64, block *MetaBlockChainData, data *PevHandle, metaBlockTime int64) (err error) {
	if block.StartBlock == "" || block.EndBlock == "" {
		return
	}
	var startHeight, endHeight int64
	startHeight, err = strconv.ParseInt(block.StartBlock, 10, 64)
	if err != nil {
		return
	}
	endHeight, err = strconv.ParseInt(block.EndBlock, 10, 64)
	if err != nil {
		return
	}
	if startHeight <= 0 || endHeight <= 0 {
		return
	}
	chainName := ""
	switch block.Chain {
	case "Bitcoin":
		chainName = "btc"
	case "MVC":
		chainName = "mvc"
	}
	ch := make(chan []pin.PinInscription)
	go func() {
		err := man.PebbleStore.Database.GetMetaBlockData(blockHeight, startHeight, endHeight, chainName, 10000, ch)
		close(ch)
		if err != nil {
			log.Println("Error getting meta block data:", err)
		}
	}()
	for batch := range ch {
		HandlePevSlice(batch, data, block, blockHeight, metaBlockTime, false)
		log.Println("Processed batch of pins for block", blockHeight, "chain", chainName, "batch size:", len(batch))
	}
	//pinList, err := man.PebbleStore.Database.GetMetaBlockData(startHeight, endHeight, chainName)

	// if len(pevList) <= 0 {
	// 	return
	// }
	// pevKey := fmt.Sprintf("%s_%d",chainName,blockHeight)
	// pevData,err := sonic.Marshal()
	// pb.Database.PevDataDB.Set([]byte(pevKey),)
	return
}
func (pb *PevPebbledb) getBlockNdvHistoryValue(height int64, host string) (total decimal.Decimal, err error) {
	if height <= 0 {
		total = decimal.Zero
		return
	}
	var item MetaSoBlockNDV
	err = mongoClient.Collection(MetaSoNDVBlockData).FindOne(context.TODO(), bson.D{{Key: "host", Value: host}, {Key: "block", Value: height - 1}}).Decode(&item)
	if err != nil {
		total = decimal.Zero
		return
	}
	total = item.DataValue.Add(item.HistoryValue)
	return
}
func (pb *PevPebbledb) getBlockMdvHistoryValue(height int64, address string) (total decimal.Decimal, err error) {
	if height <= 0 {
		total = decimal.Zero
		return
	}
	var item MetaSoBlockMDV
	err = mongoClient.Collection(MetaSoMDVBlockData).FindOne(context.TODO(), bson.D{{Key: "address", Value: address}, {Key: "block", Value: height - 1}}).Decode(&item)
	if err != nil {
		total = decimal.Zero
		return
	}
	total = item.DataValue.Add(item.HistoryValue)
	return
}

func (pb *PevPebbledb) getBlockHistory(height int64) (total decimal.Decimal, err error) {
	filter := bson.D{{Key: "block", Value: height}}
	var block MetaSoBlockInfo
	err = mongoClient.Collection(MetaSoBlockInfoData).FindOne(context.TODO(), filter).Decode(&block)
	total = block.HistoryValue.Add(block.DataValue)
	return
}
func (pb *PevPebbledb) UpdateBlockValue(blockHeight int64, lastData *PevHandle, blockTime int64) (err error) {
	// var hostMap = make(map[string]*MetaSoBlockNDV)
	// var addressMap = make(map[string]*MetaSoBlockMDV)
	// var hostAddressMap = make(map[string]*MetaSoHostAddress)
	// //fmt.Println("pevList:", blockHeight, ">>", len(pevList))
	// for _, pev := range *pevList {
	// 	if _, ok := hostMap[pev.Host]; ok {
	// 		hostMap[pev.Host].DataValue = hostMap[pev.Host].DataValue.Add(pev.IncrementalValue)
	// 		hostMap[pev.Host].PinNumber += 1
	// 	} else {
	// 		hostMap[pev.Host] = &MetaSoBlockNDV{DataValue: pev.IncrementalValue, Block: blockHeight, Host: pev.Host, PinNumber: 1, BlockTime: blockTime}
	// 	}
	// 	if _, ok := addressMap[pev.Address]; ok {
	// 		addressMap[pev.Address].DataValue = addressMap[pev.Address].DataValue.Add(pev.IncrementalValue)
	// 		addressMap[pev.Address].PinNumber += 1
	// 		t := int64(0)
	// 		if pev.Host != "metabitcoin.unknown" {
	// 			t = 1
	// 		}
	// 		addressMap[pev.Address].PinNumberHasHost += t
	// 	} else {
	// 		t := int64(0)
	// 		if pev.Host != "metabitcoin.unknown" {
	// 			t = 1
	// 		}
	// 		addressMap[pev.Address] = &MetaSoBlockMDV{DataValue: pev.IncrementalValue, Block: blockHeight, Address: pev.Address, MetaId: pev.MetaId, PinNumber: 1, PinNumberHasHost: t, BlockTime: blockTime}
	// 	}
	// 	hostAddress := fmt.Sprintf("%s--%s", pev.Host, pev.Address)
	// 	if _, ok := hostAddressMap[hostAddress]; ok {
	// 		hostAddressMap[hostAddress].DataValue = hostAddressMap[hostAddress].DataValue.Add(pev.IncrementalValue)
	// 		hostAddressMap[hostAddress].PinNumber += 1
	// 		t := int64(0)
	// 		if pev.Host != "metabitcoin.unknown" {
	// 			t = 1
	// 		}
	// 		hostAddressMap[hostAddress].PinNumberHasHost += t
	// 	} else {
	// 		t := int64(0)
	// 		if pev.Host != "metabitcoin.unknown" {
	// 			t = 1
	// 		}
	// 		hostAddressMap[hostAddress] = &MetaSoHostAddress{DataValue: pev.IncrementalValue, Block: blockHeight, Address: pev.Address, MetaId: pev.MetaId, PinNumber: 1, PinNumberHasHost: t, BlockTime: blockTime, Host: pev.Host}
	// 	}
	// }

	var hostList []*MetaSoBlockNDV
	var addressList []*MetaSoBlockMDV
	var hostAddressList []*MetaSoHostAddress
	for _, value := range lastData.HostMap {
		value.HistoryValue, _ = pb.getBlockNdvHistoryValue(blockHeight, value.Host)
		hostList = append(hostList, value)
		//fmt.Println(blockHeight, value.Host, value.DataValue)
	}
	for _, value := range lastData.AddressMap {
		value.HistoryValue, _ = pb.getBlockMdvHistoryValue(blockHeight, value.Address)
		addressList = append(addressList, value)
	}
	for _, value := range lastData.HostAddressMap {
		hostAddressList = append(hostAddressList, value)
	}
	var models []mongo.WriteModel
	for _, item := range hostList {
		filter := bson.D{{Key: "host", Value: item.Host}, {Key: "block", Value: item.Block}}
		//update := bson.D{{Key: "$set", Value: item}}
		update := bson.D{
			{Key: "$inc", Value: bson.D{
				{Key: "datavalue", Value: item.DataValue},
				{Key: "pinnumber", Value: item.PinNumber},
			}},
			{Key: "$setOnInsert", Value: bson.D{
				{Key: "host", Value: item.Host},
				{Key: "block", Value: item.Block},
				{Key: "blocktime", Value: item.BlockTime},
				{Key: "historyvalue", Value: item.HistoryValue},
			}},
		}
		m := mongo.NewUpdateOneModel()
		m.SetFilter(filter).SetUpdate(update).SetUpsert(true)
		models = append(models, m)
	}
	bulkWriteOptions := options.BulkWrite().SetOrdered(false)
	_, err = mongoClient.Collection(MetaSoNDVBlockData).BulkWrite(context.Background(), models, bulkWriteOptions)
	log.Println("MetaSoNDVBlockData bulk write error:", err)

	var models2 []mongo.WriteModel
	for _, item := range addressList {
		filter := bson.D{{Key: "address", Value: item.Address}, {Key: "block", Value: item.Block}}
		//update := bson.D{{Key: "$set", Value: item}}
		update := bson.D{
			{Key: "$inc", Value: bson.D{
				{Key: "datavalue", Value: item.DataValue},
				{Key: "pinnumber", Value: item.PinNumber},
			}},
			{Key: "$setOnInsert", Value: bson.D{
				{Key: "address", Value: item.Address},
				{Key: "metaid", Value: item.MetaId},
				{Key: "block", Value: item.Block},
				{Key: "blocktime", Value: item.BlockTime},
				{Key: "historyvalue", Value: item.HistoryValue},
				{Key: "pinnumberhashost", Value: item.PinNumberHasHost},
			}},
		}
		m := mongo.NewUpdateOneModel()
		m.SetFilter(filter).SetUpdate(update).SetUpsert(true)
		models2 = append(models2, m)
	}
	_, err = mongoClient.Collection(MetaSoMDVBlockData).BulkWrite(context.Background(), models2, bulkWriteOptions)
	log.Println("MetaSoMDVBlockData bulk write error:", err)

	var models3 []mongo.WriteModel
	for _, item := range hostAddressList {
		filter := bson.D{{Key: "address", Value: item.Address}, {Key: "block", Value: item.Block}, {Key: "host", Value: item.Host}}
		//update := bson.D{{Key: "$set", Value: item}}
		update := bson.D{
			{Key: "$inc", Value: bson.D{
				{Key: "datavalue", Value: item.DataValue},
				{Key: "pinnumber", Value: item.PinNumber},
			}},
			{Key: "$setOnInsert", Value: bson.D{
				{Key: "host", Value: item.Host},
				{Key: "address", Value: item.Address},
				{Key: "metaid", Value: item.MetaId},
				{Key: "block", Value: item.Block},
				{Key: "blocktime", Value: item.BlockTime},
				{Key: "historyvalue", Value: item.HistoryValue},
				{Key: "pinnumberhashost", Value: item.PinNumberHasHost},
			}},
		}
		m := mongo.NewUpdateOneModel()
		m.SetFilter(filter).SetUpdate(update).SetUpsert(true)
		models3 = append(models3, m)
	}
	_, err = mongoClient.Collection(MetaSoHostAddressData).BulkWrite(context.Background(), models3, bulkWriteOptions)
	log.Println("MetaSoHostAddressData bulk write error:", err)
	return
}

// func (pb *PevPebbledb) UpdateDataValue(hostMap *map[string]struct{}, addressMap *map[string]struct{}) (err error) {
// 	for host := range *hostMap {
// 		total, err := pb.GetHostDataSum(host)
// 		//fmt.Println(err, host, total)
// 		if err == nil && total.Cmp(decimal.Zero) >= 1 {
// 			data := MetaSoNDV{
// 				Host:      host,
// 				DataValue: total,
// 			}
// 			mongoClient.Collection(MetaSoNDVData).UpdateOne(context.TODO(), bson.M{"host": host}, bson.M{"$set": data}, options.Update().SetUpsert(true))
// 		}
// 	}
// 	for address := range *addressMap {
// 		total, err := pb.getMetaDataSum(address)
// 		if err == nil && total.Cmp(decimal.Zero) >= 1 {
// 			data := MetaSoMDV{
// 				MetaId:    common.GetMetaIdByAddress(address),
// 				Address:   address,
// 				DataValue: total,
// 			}
// 			mongoClient.Collection(MetaSoMDVData).UpdateOne(context.TODO(), bson.M{"address": address}, bson.M{"$set": data}, options.Update().SetUpsert(true))
// 			time.Sleep(time.Millisecond * 100)
// 		}
// 	}
// 	return
// }
// func (pb *PevPebbledb) GetHostDataSum(host string) (dataValue decimal.Decimal, err error) {
// 	filter := bson.D{{Key: "host", Value: host}}
// 	match := bson.D{{Key: "$match", Value: filter}}
// 	groupStage := bson.D{
// 		{Key: "$group", Value: bson.D{
// 			{Key: "_id", Value: "$host"},
// 			{Key: "totalValue", Value: bson.D{{Key: "$sum", Value: "$incrementalvalue"}}},
// 		}}}
// 	cursor, err := mongoClient.Collection(MetaSoPEVData).Aggregate(context.TODO(), mongo.Pipeline{match, groupStage})

// 	//cursor, err := mongoClient.Collection(MetaSoPEVData).Aggregate(context.TODO(), pipeline)
// 	if err != nil {
// 		return
// 	}
// 	defer cursor.Close(context.TODO())
// 	var results []bson.M
// 	if err = cursor.All(context.TODO(), &results); err != nil {
// 		return
// 	}
// 	for _, result := range results {
// 		if result["_id"] == host {
// 			dataValue, _ = Decimal128ToDecimal(result["totalValue"].(primitive.Decimal128))
// 			break
// 		}
// 	}
// 	return
// }

// func (pb *PevPebbledb) getMetaDataSum(address string) (dataValue decimal.Decimal, err error) {
// 	pipeline := bson.A{
// 		bson.D{
// 			{Key: "$match", Value: bson.D{
// 				{Key: "address", Value: address},
// 			}},
// 		},
// 		bson.D{
// 			{Key: "$group", Value: bson.D{
// 				{Key: "_id", Value: "$address"},
// 				{Key: "totalValue", Value: bson.D{
// 					{Key: "$sum", Value: "$incrementalvalue"},
// 				}},
// 			}},
// 		},
// 	}
// 	cursor, err := mongoClient.Collection(MetaSoPEVData).Aggregate(context.TODO(), pipeline)
// 	if err != nil {
// 		return
// 	}
// 	defer cursor.Close(context.TODO())
// 	var results []bson.M
// 	if err = cursor.All(context.TODO(), &results); err != nil {
// 		return
// 	}
// 	for _, result := range results {
// 		if result["_id"] == address {
// 			dataValue, _ = Decimal128ToDecimal(result["totalValue"].(primitive.Decimal128))
// 			break
// 		}
// 	}
// 	return
// }

func (pb *PevPebbledb) CountPDV(blockHeight int64, block *MetaBlockChainData, pinNode *pin.PinInscription) (data []PEVData, err error) {
	switch pinNode.Path {
	case "/follow":
		return pb.countFollowPDV(blockHeight, block, pinNode)
	case "/protocols/simpledonate":
		return pb.countDonatePDV(blockHeight, block, pinNode)
	case "/protocols/paylike":
		return pb.countPayLike(blockHeight, block, pinNode)
	case "/protocols/paycomment":
		return pb.countPaycomment(blockHeight, block, pinNode)
	case "/protocols/simplebuzz":
		return pb.countSimplebuzz(blockHeight, block, pinNode)
	case "/ft/mrc20/mint":
		return pb.countMrc20Mint(blockHeight, block, pinNode)
	default:
		data = pb.createPDV(blockHeight, block, pinNode, pinNode, decimal.NewFromInt(1*8))
		return
	}
}
func (pb *PevPebbledb) createPDV(blockHeight int64, block *MetaBlockChainData, fromPIN *pin.PinInscription, toPIN *pin.PinInscription, value decimal.Decimal) []PEVData {
	startHeight, _ := strconv.ParseInt(block.StartBlock, 10, 64)
	endHeight, _ := strconv.ParseInt(block.EndBlock, 10, 64)
	// lv := int64(fromPIN.PopLv)
	// if lv <= 0 {
	// 	lv = int64(1)
	// }
	// dv, _ := OctalStringToDecimal(fromPIN.Pop, 4, 10000)
	// dvDecimal := decimal.Zero
	// if dv != nil {
	// 	dvDecimal = decimal.NewFromFloat(*dv)
	// }
	cut := common.Config.Mvc.PopCutNum
	if fromPIN.ChainName == "btc" {
		cut = common.Config.Btc.PopCutNum
	}
	dvDecimal := pin.GetPoPScore(fromPIN.Pop, int64(fromPIN.PopLv), cut)
	if blockHeight >= 0 && blockHeight <= 44 {
		//dvDecimal = fromPIN.PoPScoreV1
		//dvDecimal = decimal.NewFromInt(lv).Mul(value).Add(fromPIN.PoPScoreV1)
		dvDecimal = pin.GetPoPScoreV1(fromPIN.Pop, fromPIN.PopLv)
	}
	var result []PEVData
	data := PEVData{
		Host:             toPIN.Host,
		FromPINId:        fromPIN.Id,
		ToPINId:          toPIN.Id,
		Path:             fromPIN.Path,
		Address:          toPIN.CreateAddress,
		MetaId:           toPIN.CreateMetaId,
		FromChainName:    fromPIN.ChainName,
		ToChainName:      toPIN.ChainName,
		MetaBlockHeight:  blockHeight,
		StartBlockHeight: startHeight,
		EndBlockHeight:   endHeight,
		BlockHeight:      fromPIN.GenesisHeight,
		Poplv:            fromPIN.PopLv,
		IncrementalValue: dvDecimal,
	}
	result = append(result, data)
	if fromPIN.Id != toPIN.Id {
		data2 := PEVData{
			Host:             fromPIN.Host,
			FromPINId:        fromPIN.Id,
			ToPINId:          fromPIN.Id,
			Path:             fromPIN.Path,
			Address:          fromPIN.Address,
			MetaId:           fromPIN.MetaId,
			FromChainName:    fromPIN.ChainName,
			ToChainName:      fromPIN.ChainName,
			MetaBlockHeight:  blockHeight,
			StartBlockHeight: startHeight,
			EndBlockHeight:   endHeight,
			BlockHeight:      fromPIN.GenesisHeight,
			Poplv:            fromPIN.PopLv,
			IncrementalValue: dvDecimal,
		}
		result = append(result, data2)
	}
	return result
}
func (pb *PevPebbledb) countFollowPDV(blockHeight int64, block *MetaBlockChainData, pinNode *pin.PinInscription) (data []PEVData, err error) {
	metaid := string(pinNode.ContentBody)
	filter := bson.M{"metaid": metaid}
	findOptions := options.FindOne()
	findOptions.SetSort(bson.D{{Key: "_id", Value: 1}})
	var toPIN pin.PinInscription
	err = mongoClient.Collection(mongodb.PinsCollection).FindOne(context.TODO(), filter, findOptions).Decode(&toPIN)
	if err != nil {
		toPIN = *pinNode
	}
	data = pb.createPDV(blockHeight, block, pinNode, &toPIN, decimal.NewFromInt(1*8))
	return
}
func (pb *PevPebbledb) getPINbyId(pinId string) (pinNode *pin.PinInscription, err error) {
	result, err := man.PebbleStore.Database.GetPinInscriptionByKey(pinId)
	if err == nil {
		pinNode = &result
	}
	return
}
func (pb *PevPebbledb) countDonatePDV(blockHeight int64, block *MetaBlockChainData, pinNode *pin.PinInscription) (data []PEVData, err error) {
	var dataMap map[string]interface{}
	err = json.Unmarshal(pinNode.ContentBody, &dataMap)
	if err != nil {
		return
	}
	toPIN, err := pb.getPINbyId(dataMap["toPin"].(string))
	if err != nil {
		toPIN = pinNode
	}
	data = pb.createPDV(blockHeight, block, pinNode, toPIN, decimal.NewFromInt(1*8))
	return
}
func (pb *PevPebbledb) countPayLike(blockHeight int64, block *MetaBlockChainData, pinNode *pin.PinInscription) (data []PEVData, err error) {
	var dataMap map[string]interface{}
	err = json.Unmarshal(pinNode.ContentBody, &dataMap)
	if err != nil {
		return
	}
	var toPIN *pin.PinInscription
	if dataMap["likeTo"].(string) == "" || dataMap["isLike"].(string) != "1" {
		toPIN = pinNode
	} else {
		toPIN, err = pb.getPINbyId(dataMap["likeTo"].(string))
		if err != nil {
			toPIN = pinNode
		}
	}
	data = pb.createPDV(blockHeight, block, pinNode, toPIN, decimal.NewFromInt(1*8))
	return
}
func (pb *PevPebbledb) countPaycomment(blockHeight int64, block *MetaBlockChainData, pinNode *pin.PinInscription) (data []PEVData, err error) {
	var dataMap map[string]interface{}
	err = json.Unmarshal(pinNode.ContentBody, &dataMap)
	if err != nil {
		return
	}
	var toPIN *pin.PinInscription
	if dataMap["commentTo"].(string) == "" {
		toPIN = pinNode
	} else {
		toPIN, err = pb.getPINbyId(dataMap["commentTo"].(string))
		if err != nil {
			toPIN = pinNode
		}
	}

	data = pb.createPDV(blockHeight, block, pinNode, toPIN, decimal.NewFromInt(1*8))
	return
}
func (pb *PevPebbledb) countSimplebuzz(blockHeight int64, block *MetaBlockChainData, pinNode *pin.PinInscription) (data []PEVData, err error) {
	var dataMap map[string]interface{}
	err = json.Unmarshal(pinNode.ContentBody, &dataMap)
	if err != nil {
		return
	}
	if dataMap["quotePin"] == nil || dataMap["quotePin"].(string) == "" {
		data = pb.createPDV(blockHeight, block, pinNode, pinNode, decimal.NewFromInt(1*8))
		return
	}
	toPIN, err := pb.getPINbyId(dataMap["quotePin"].(string))
	if err != nil {
		return
	}
	data = pb.createPDV(blockHeight, block, pinNode, toPIN, decimal.NewFromInt(1*8))
	return
}
func (pb *PevPebbledb) countMrc20Mint(blockHeight int64, block *MetaBlockChainData, pinNode *pin.PinInscription) (data []PEVData, err error) {
	var dataMap map[string]interface{}
	err = json.Unmarshal(pinNode.ContentBody, &dataMap)
	if err != nil {
		return
	}
	if dataMap["id"].(string) == "" {
		return
	}
	toPIN, err := pb.getPINbyId(dataMap["id"].(string))
	if err != nil {
		return
	}
	data = pb.createPDV(blockHeight, block, pinNode, toPIN, decimal.NewFromInt(1*8))
	return
}

func ArrayExist(key string, list []string) (exist bool) {
	for _, item := range list {
		if item == key {
			exist = true
			return
		}
	}
	return
}
