package metaso

// import (
// 	"context"
// 	"encoding/json"
// 	"fmt"
// 	"manindexer/common"
// 	"manindexer/database/mongodb"
// 	"manindexer/pin"
// 	"strconv"
// 	"time"

// 	"github.com/shopspring/decimal"
// 	"go.mongodb.org/mongo-driver/bson"
// 	"go.mongodb.org/mongo-driver/bson/primitive"
// 	"go.mongodb.org/mongo-driver/mongo"
// 	"go.mongodb.org/mongo-driver/mongo/options"
// )

// func GetPevDataByMetaBlock(blockHeight int64) (pevList []PEVData, err error) {
// 	result, err := mongoClient.Collection(MetaSoPEVData).Find(context.TODO(), bson.M{"metablockheight": blockHeight})
// 	if err == nil {
// 		result.All(context.TODO(), &pevList)
// 	}
// 	return
// }
// func CountBlockPEV(blockHeight int64, block *MetaBlockChainData) (pevList []interface{}, err error) {
// 	if block.StartBlock == "" || block.EndBlock == "" {
// 		return
// 	}
// 	var startHeight, endHeight int64
// 	startHeight, err = strconv.ParseInt(block.StartBlock, 10, 64)
// 	if err != nil {
// 		return
// 	}
// 	endHeight, err = strconv.ParseInt(block.EndBlock, 10, 64)
// 	if err != nil {
// 		return
// 	}
// 	if startHeight <= 0 || endHeight <= 0 {
// 		return
// 	}
// 	chainName := ""
// 	switch block.Chain {
// 	case "Bitcoin":
// 		chainName = "btc"
// 	case "MVC":
// 		chainName = "mvc"
// 	}
// 	filter := bson.D{
// 		{Key: "chainname", Value: chainName},
// 		{Key: "genesisheight", Value: bson.D{{Key: "$gte", Value: startHeight}}},
// 		{Key: "genesisheight", Value: bson.D{{Key: "$lte", Value: endHeight}}},
// 	}
// 	results, err := mongoClient.Collection(mongodb.PinsCollection).Find(context.TODO(), filter)
// 	if err != nil {
// 		return
// 	}
// 	var pinList []*pin.PinInscription
// 	err = results.All(context.TODO(), &pinList)
// 	allowProtocols := common.Config.Statistics.AllowProtocols
// 	allowHost := common.Config.Statistics.AllowHost

// 	for _, pinNode := range pinList {
// 		if pinNode.Host == "metabitcoin.unknown" {
// 			continue
// 		}
// 		if pinNode.Host == "" {
// 			pinNode.Host = "metabitcoin.unknown"
// 		}
// 		if len(allowProtocols) >= 1 && allowProtocols[0] != "*" {
// 			if !ArrayExist(pinNode.Path, allowProtocols) {
// 				continue
// 			}
// 		}
// 		if len(allowHost) >= 1 && allowHost[0] != "*" {
// 			if !ArrayExist(pinNode.Host, allowHost) {
// 				continue
// 			}
// 		}
// 		pevs, err := CountPDV(blockHeight, block, pinNode)
// 		if err != nil {
// 			continue
// 		}
// 		for _, pev := range pevs {
// 			if pev.ToPINId == "" {
// 				continue
// 			}
// 			if pev.Host == "" || len(pev.Host) == 0 {
// 				pev.Host = "metabitcoin.unknown"
// 			}
// 			pevList = append(pevList, pev)
// 		}
// 	}
// 	if len(pevList) <= 0 {
// 		return
// 	}

// 	insertOpts := options.InsertMany().SetOrdered(false)
// 	_, err = mongoClient.Collection(MetaSoPEVData).InsertMany(context.TODO(), pevList, insertOpts)

// 	return
// }
// func getBlockHistoryValue(height int64, key string, value string) (total decimal.Decimal, err error) {
// 	filter := bson.D{{Key: "metablockheight", Value: bson.D{{Key: "$lt", Value: height}}}, {Key: "metablockheight", Value: bson.D{{Key: "$gt", Value: -1}}}}
// 	if key != "" && value != "" {
// 		filter = append(filter, bson.E{Key: key, Value: value})
// 	}
// 	match := bson.D{{Key: "$match", Value: filter}}
// 	groupStage := bson.D{
// 		{Key: "$group", Value: bson.D{
// 			{Key: "_id", Value: nil},
// 			{Key: "totalValue", Value: bson.D{{Key: "$sum", Value: "$incrementalvalue"}}},
// 		}}}
// 	cursor, err := mongoClient.Collection(MetaSoPEVData).Aggregate(context.TODO(), mongo.Pipeline{match, groupStage})
// 	if err != nil {
// 		return
// 	}
// 	defer cursor.Close(context.TODO())
// 	var results []bson.M
// 	if err = cursor.All(context.TODO(), &results); err != nil {
// 		return
// 	}
// 	if len(results) > 0 {
// 		total, _ = Decimal128ToDecimal(results[0]["totalValue"].(primitive.Decimal128))
// 	}

// 	return
// }
// func getBlockHistory(height int64) (total decimal.Decimal, err error) {
// 	filter := bson.D{{Key: "block", Value: height}}
// 	var block MetaSoBlockInfo
// 	err = mongoClient.Collection(MetaSoBlockInfoData).FindOne(context.TODO(), filter).Decode(&block)
// 	total = block.HistoryValue.Add(block.DataValue)
// 	return
// }
// func UpdateBlockValue(blockHeight int64, pevList []interface{}, blockTime int64) (err error) {
// 	if blockHeight == -1 {
// 		mongoClient.Collection(MetaSoNDVBlockData).DeleteMany(context.TODO(), bson.M{"block": -1})
// 		mongoClient.Collection(MetaSoMDVBlockData).DeleteMany(context.TODO(), bson.M{"block": -1})
// 		mongoClient.Collection(MetaSoHostAddressData).DeleteMany(context.TODO(), bson.M{"block": -1})
// 	}
// 	var hostMap = make(map[string]*MetaSoBlockNDV)
// 	var addressMap = make(map[string]*MetaSoBlockMDV)
// 	var hostAddressMap = make(map[string]*MetaSoHostAddress)
// 	//fmt.Println("pevList:", blockHeight, ">>", len(pevList))
// 	for _, item := range pevList {
// 		pev := item.(PEVData)
// 		if _, ok := hostMap[pev.Host]; ok {
// 			hostMap[pev.Host].DataValue = hostMap[pev.Host].DataValue.Add(pev.IncrementalValue)
// 			hostMap[pev.Host].PinNumber += 1
// 		} else {
// 			hostMap[pev.Host] = &MetaSoBlockNDV{DataValue: pev.IncrementalValue, Block: blockHeight, Host: pev.Host, PinNumber: 1, BlockTime: blockTime}
// 		}
// 		if _, ok := addressMap[pev.Address]; ok {
// 			addressMap[pev.Address].DataValue = addressMap[pev.Address].DataValue.Add(pev.IncrementalValue)
// 			addressMap[pev.Address].PinNumber += 1
// 			t := int64(0)
// 			if pev.Host != "metabitcoin.unknown" {
// 				t = 1
// 			}
// 			addressMap[pev.Address].PinNumberHasHost += t
// 		} else {
// 			t := int64(0)
// 			if pev.Host != "metabitcoin.unknown" {
// 				t = 1
// 			}
// 			addressMap[pev.Address] = &MetaSoBlockMDV{DataValue: pev.IncrementalValue, Block: blockHeight, Address: pev.Address, MetaId: pev.MetaId, PinNumber: 1, PinNumberHasHost: t, BlockTime: blockTime}
// 		}
// 		hostAddress := fmt.Sprintf("%s--%s", pev.Host, pev.Address)
// 		if _, ok := hostAddressMap[hostAddress]; ok {
// 			hostAddressMap[hostAddress].DataValue = hostAddressMap[hostAddress].DataValue.Add(pev.IncrementalValue)
// 			hostAddressMap[hostAddress].PinNumber += 1
// 			t := int64(0)
// 			if pev.Host != "metabitcoin.unknown" {
// 				t = 1
// 			}
// 			hostAddressMap[hostAddress].PinNumberHasHost += t
// 		} else {
// 			t := int64(0)
// 			if pev.Host != "metabitcoin.unknown" {
// 				t = 1
// 			}
// 			hostAddressMap[hostAddress] = &MetaSoHostAddress{DataValue: pev.IncrementalValue, Block: blockHeight, Address: pev.Address, MetaId: pev.MetaId, PinNumber: 1, PinNumberHasHost: t, BlockTime: blockTime, Host: pev.Host}
// 		}
// 	}

// 	var hostList []*MetaSoBlockNDV
// 	var addressList []*MetaSoBlockMDV
// 	var hostAddressList []*MetaSoHostAddress
// 	for _, value := range hostMap {
// 		value.HistoryValue, _ = getBlockHistoryValue(blockHeight, "host", value.Host)
// 		hostList = append(hostList, value)
// 		//fmt.Println(blockHeight, value.Host, value.DataValue)
// 	}
// 	for _, value := range addressMap {
// 		value.HistoryValue, _ = getBlockHistoryValue(blockHeight, "address", value.Address)
// 		addressList = append(addressList, value)
// 	}
// 	for _, value := range hostAddressMap {
// 		hostAddressList = append(hostAddressList, value)
// 	}
// 	var models []mongo.WriteModel
// 	for _, item := range hostList {
// 		filter := bson.D{{Key: "host", Value: item.Host}, {Key: "block", Value: item.Block}}
// 		update := bson.D{{Key: "$set", Value: item}}
// 		m := mongo.NewUpdateOneModel()
// 		m.SetFilter(filter).SetUpdate(update).SetUpsert(true)
// 		models = append(models, m)
// 	}
// 	bulkWriteOptions := options.BulkWrite().SetOrdered(false)
// 	mongoClient.Collection(MetaSoNDVBlockData).BulkWrite(context.Background(), models, bulkWriteOptions)

// 	var models2 []mongo.WriteModel
// 	for _, item := range addressList {
// 		filter := bson.D{{Key: "address", Value: item.Address}, {Key: "block", Value: item.Block}}
// 		update := bson.D{{Key: "$set", Value: item}}
// 		m := mongo.NewUpdateOneModel()
// 		m.SetFilter(filter).SetUpdate(update).SetUpsert(true)
// 		models2 = append(models2, m)
// 	}
// 	mongoClient.Collection(MetaSoMDVBlockData).BulkWrite(context.Background(), models2, bulkWriteOptions)

// 	var models3 []mongo.WriteModel
// 	for _, item := range hostAddressList {
// 		filter := bson.D{{Key: "address", Value: item.Address}, {Key: "block", Value: item.Block}, {Key: "host", Value: item.Host}}
// 		update := bson.D{{Key: "$set", Value: item}}
// 		m := mongo.NewUpdateOneModel()
// 		m.SetFilter(filter).SetUpdate(update).SetUpsert(true)
// 		models3 = append(models3, m)
// 	}
// 	mongoClient.Collection(MetaSoHostAddressData).BulkWrite(context.Background(), models3, bulkWriteOptions)

// 	return
// }
// func UpdateDataValue(hostMap *map[string]struct{}, addressMap *map[string]struct{}) (err error) {
// 	for host := range *hostMap {
// 		total, err := GetHostDataSum(host)
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
// 		total, err := getMetaDataSum(address)
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
// func GetHostDataSum(host string) (dataValue decimal.Decimal, err error) {
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
// func convertFloat64(value interface{}) float64 {
// 	switch v := value.(type) {
// 	case float64:
// 		return v
// 	case int32:
// 		return float64(v)
// 	case int64:
// 		return float64(v)
// 	case int:
// 		return float64(v)
// 	default:
// 		return float64(0)
// 	}
// }
// func getMetaDataSum(address string) (dataValue decimal.Decimal, err error) {
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
// func ArrayExist(key string, list []string) (exist bool) {
// 	for _, item := range list {
// 		if item == key {
// 			exist = true
// 			return
// 		}
// 	}
// 	return
// }
// func CountPDV(blockHeight int64, block *MetaBlockChainData, pinNode *pin.PinInscription) (data []PEVData, err error) {
// 	switch pinNode.Path {
// 	case "/follow":
// 		return countFollowPDV(blockHeight, block, pinNode)
// 	case "/protocols/simpledonate":
// 		return countDonatePDV(blockHeight, block, pinNode)
// 	case "/protocols/paylike":
// 		return countPayLike(blockHeight, block, pinNode)
// 	case "/protocols/paycomment":
// 		return countPaycomment(blockHeight, block, pinNode)
// 	case "/protocols/simplebuzz":
// 		return countSimplebuzz(blockHeight, block, pinNode)
// 	case "/ft/mrc20/mint":
// 		return countMrc20Mint(blockHeight, block, pinNode)
// 	default:
// 		data = createPDV(blockHeight, block, pinNode, pinNode, decimal.NewFromInt(1*8))
// 		return
// 	}
// }
// func createPDV(blockHeight int64, block *MetaBlockChainData, fromPIN *pin.PinInscription, toPIN *pin.PinInscription, value decimal.Decimal) []PEVData {
// 	startHeight, _ := strconv.ParseInt(block.StartBlock, 10, 64)
// 	endHeight, _ := strconv.ParseInt(block.EndBlock, 10, 64)
// 	lv := int64(fromPIN.PopLv)
// 	if lv <= 0 {
// 		lv = int64(1)
// 	}

// 	// dv, _ := OctalStringToDecimal(fromPIN.Pop, 4, 10000)
// 	// dvDecimal := decimal.Zero
// 	// if dv != nil {
// 	// 	dvDecimal = decimal.NewFromFloat(*dv)
// 	// }
// 	dvDecimal := decimal.Zero
// 	if blockHeight >= 0 && blockHeight <= 44 {
// 		dvDecimal = pin.GetPoPScoreV1(fromPIN.Pop, int(lv))
// 	}
// 	cut := common.Config.Mvc.PopCutNum
// 	if fromPIN.ChainName == "btc" {
// 		cut = common.Config.Btc.PopCutNum
// 	}
// 	pin.GetPoPScore(fromPIN.Pop, lv, cut)
// 	var result []PEVData
// 	data := PEVData{
// 		Host:             toPIN.Host,
// 		FromPINId:        fromPIN.Id,
// 		ToPINId:          toPIN.Id,
// 		Path:             fromPIN.Path,
// 		Address:          toPIN.CreateAddress,
// 		MetaId:           toPIN.CreateMetaId,
// 		FromChainName:    fromPIN.ChainName,
// 		ToChainName:      toPIN.ChainName,
// 		MetaBlockHeight:  blockHeight,
// 		StartBlockHeight: startHeight,
// 		EndBlockHeight:   endHeight,
// 		BlockHeight:      fromPIN.GenesisHeight,
// 		Poplv:            fromPIN.PopLv,
// 		IncrementalValue: decimal.NewFromInt(lv).Mul(value).Add(dvDecimal),
// 	}
// 	result = append(result, data)
// 	if fromPIN.Id != toPIN.Id {
// 		data2 := PEVData{
// 			Host:             fromPIN.Host,
// 			FromPINId:        fromPIN.Id,
// 			ToPINId:          fromPIN.Id,
// 			Path:             fromPIN.Path,
// 			Address:          fromPIN.Address,
// 			MetaId:           fromPIN.MetaId,
// 			FromChainName:    fromPIN.ChainName,
// 			ToChainName:      fromPIN.ChainName,
// 			MetaBlockHeight:  blockHeight,
// 			StartBlockHeight: startHeight,
// 			EndBlockHeight:   endHeight,
// 			BlockHeight:      fromPIN.GenesisHeight,
// 			Poplv:            fromPIN.PopLv,
// 			IncrementalValue: decimal.NewFromInt(lv).Mul(value).Add(dvDecimal),
// 		}
// 		result = append(result, data2)
// 	}
// 	return result
// }
// func countFollowPDV(blockHeight int64, block *MetaBlockChainData, pinNode *pin.PinInscription) (data []PEVData, err error) {
// 	metaid := string(pinNode.ContentBody)
// 	filter := bson.M{"metaid": metaid}
// 	findOptions := options.FindOne()
// 	findOptions.SetSort(bson.D{{Key: "_id", Value: 1}})
// 	var toPIN pin.PinInscription
// 	err = mongoClient.Collection(mongodb.PinsCollection).FindOne(context.TODO(), filter, findOptions).Decode(&toPIN)
// 	if err != nil {
// 		toPIN = *pinNode
// 	}
// 	data = createPDV(blockHeight, block, pinNode, &toPIN, decimal.NewFromInt(1*8))
// 	return
// }
// func getPINbyId(pinId string) (pinNode *pin.PinInscription, err error) {
// 	filter := bson.M{"id": pinId}
// 	err = mongoClient.Collection(mongodb.PinsCollection).FindOne(context.TODO(), filter, nil).Decode(&pinNode)
// 	return
// }
// func countDonatePDV(blockHeight int64, block *MetaBlockChainData, pinNode *pin.PinInscription) (data []PEVData, err error) {
// 	var dataMap map[string]interface{}
// 	err = json.Unmarshal(pinNode.ContentBody, &dataMap)
// 	if err != nil {
// 		return
// 	}
// 	toPIN, err := getPINbyId(dataMap["toPin"].(string))
// 	if err != nil {
// 		toPIN = pinNode
// 	}
// 	data = createPDV(blockHeight, block, pinNode, toPIN, decimal.NewFromInt(1*8))
// 	return
// }
// func countPayLike(blockHeight int64, block *MetaBlockChainData, pinNode *pin.PinInscription) (data []PEVData, err error) {
// 	var dataMap map[string]interface{}
// 	err = json.Unmarshal(pinNode.ContentBody, &dataMap)
// 	if err != nil {
// 		return
// 	}
// 	var toPIN *pin.PinInscription
// 	if dataMap["likeTo"].(string) == "" || dataMap["isLike"].(string) != "1" {
// 		toPIN = pinNode
// 	} else {
// 		toPIN, err = getPINbyId(dataMap["likeTo"].(string))
// 		if err != nil {
// 			toPIN = pinNode
// 		}
// 	}
// 	data = createPDV(blockHeight, block, pinNode, toPIN, decimal.NewFromInt(1*8))
// 	return
// }
// func countPaycomment(blockHeight int64, block *MetaBlockChainData, pinNode *pin.PinInscription) (data []PEVData, err error) {
// 	var dataMap map[string]interface{}
// 	err = json.Unmarshal(pinNode.ContentBody, &dataMap)
// 	if err != nil {
// 		return
// 	}
// 	var toPIN *pin.PinInscription
// 	if dataMap["commentTo"].(string) == "" {
// 		toPIN = pinNode
// 	} else {
// 		toPIN, err = getPINbyId(dataMap["commentTo"].(string))
// 		if err != nil {
// 			toPIN = pinNode
// 		}
// 	}

// 	data = createPDV(blockHeight, block, pinNode, toPIN, decimal.NewFromInt(1*8))
// 	return
// }
// func countSimplebuzz(blockHeight int64, block *MetaBlockChainData, pinNode *pin.PinInscription) (data []PEVData, err error) {
// 	var dataMap map[string]interface{}
// 	err = json.Unmarshal(pinNode.ContentBody, &dataMap)
// 	if err != nil {
// 		return
// 	}
// 	if dataMap["quotePin"] == nil || dataMap["quotePin"].(string) == "" {
// 		data = createPDV(blockHeight, block, pinNode, pinNode, decimal.NewFromInt(1*8))
// 		return
// 	}
// 	toPIN, err := getPINbyId(dataMap["quotePin"].(string))
// 	if err != nil {
// 		return
// 	}
// 	data = createPDV(blockHeight, block, pinNode, toPIN, decimal.NewFromInt(1*8))
// 	return
// }
// func countMrc20Mint(blockHeight int64, block *MetaBlockChainData, pinNode *pin.PinInscription) (data []PEVData, err error) {
// 	var dataMap map[string]interface{}
// 	err = json.Unmarshal(pinNode.ContentBody, &dataMap)
// 	if err != nil {
// 		return
// 	}
// 	if dataMap["id"].(string) == "" {
// 		return
// 	}
// 	toPIN, err := getPINbyId(dataMap["id"].(string))
// 	if err != nil {
// 		return
// 	}
// 	data = createPDV(blockHeight, block, pinNode, toPIN, decimal.NewFromInt(1*8))
// 	return
// }
