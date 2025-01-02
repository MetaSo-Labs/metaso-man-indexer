package metaso

import (
	"context"
	"encoding/json"
	"manindexer/common"
	"manindexer/database/mongodb"
	"manindexer/pin"
	"strconv"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func CountBlockPEV(blockHeight int64, block *MetaBlockChainData) (pevList []interface{}, err error) {
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
	filter := bson.D{
		{Key: "chainname", Value: chainName},
		{Key: "genesisheight", Value: bson.D{{Key: "$gte", Value: startHeight}}},
		{Key: "genesisheight", Value: bson.D{{Key: "$lte", Value: endHeight}}},
	}
	results, err := mongoClient.Collection(mongodb.PinsCollection).Find(context.TODO(), filter)
	if err != nil {
		return
	}
	var pinList []*pin.PinInscription
	err = results.All(context.TODO(), &pinList)
	//fmt.Println("count pinList:", chainName, startHeight, endHeight, len(pinList))
	//var pevList []interface{}
	allowProtocols := common.Config.Statistics.AllowProtocols
	allowHost := common.Config.Statistics.AllowHost

	for _, pinNode := range pinList {
		if pinNode.Host == "metabitcoin.unknown" {
			continue
		}
		if pinNode.Host == "" {
			pinNode.Host = "metabitcoin.unknown"
		}
		if len(allowProtocols) >= 1 && allowProtocols[0] != "*" {
			if !ArrayExist(pinNode.Path, allowProtocols) {
				continue
			}
		}
		if len(allowHost) >= 1 && allowHost[0] != "*" {
			if !ArrayExist(pinNode.Host, allowHost) {
				continue
			}
		}
		pev, err := countPDV(blockHeight, block, pinNode)
		if err != nil {
			continue
		}
		if pev.ToPINId == "" {
			continue
		}
		if pev.Host == "" || len(pev.Host) == 0 {
			pev.Host = "metabitcoin.unknown"
		}

		pevList = append(pevList, pev)

	}
	if len(pevList) <= 0 {
		return
	}

	insertOpts := options.InsertMany().SetOrdered(false)
	_, err = mongoClient.Collection(MetaSoPEVData).InsertMany(context.TODO(), pevList, insertOpts)

	return
}
func getBlockHistoryValue(height int64, key string, value string) (total float64, err error) {
	filter := bson.D{{Key: "metablockheight", Value: bson.D{{Key: "$lt", Value: height}}}}
	if key != "" && value != "" {
		filter = append(filter, bson.E{Key: key, Value: value})
	}
	match := bson.D{{Key: "$match", Value: filter}}
	groupStage := bson.D{
		{Key: "$group", Value: bson.D{
			{Key: "_id", Value: nil},
			{Key: "totalValue", Value: bson.D{{Key: "$sum", Value: "$incrementalvalue"}}},
		}}}
	cursor, err := mongoClient.Collection(MetaSoPEVData).Aggregate(context.TODO(), mongo.Pipeline{match, groupStage})
	if err != nil {
		return
	}
	defer cursor.Close(context.TODO())
	var results []bson.M
	if err = cursor.All(context.TODO(), &results); err != nil {
		return
	}
	if len(results) > 0 {
		total = convertFloat64(results[0]["totalValue"])
	}

	return
}
func UpdateBlcokValue(blockHeight int64, pevList []interface{}) (err error) {
	var hostMap = make(map[string]*MetaSoBlockNDV)
	var addressMap = make(map[string]*MetaSoBlockMDV)
	//fmt.Println("pevList:", blockHeight, ">>", len(pevList))
	for _, item := range pevList {
		pev := item.(PEVData)
		if _, ok := hostMap[pev.Host]; ok {
			hostMap[pev.Host].DataValue += pev.IncrementalValue
			hostMap[pev.Host].PinNumber += 1
		} else {
			hostMap[pev.Host] = &MetaSoBlockNDV{DataValue: pev.IncrementalValue, Block: blockHeight, Host: pev.Host, PinNumber: 1}
		}
		if _, ok := addressMap[pev.Address]; ok {
			addressMap[pev.Address].DataValue += pev.IncrementalValue
			addressMap[pev.Address].PinNumber += 1
			t := int64(0)
			if pev.Host != "metabitcoin.unknown" {
				t = 1
			}
			addressMap[pev.Address].PinNumberHasHost += t
		} else {
			t := int64(0)
			if pev.Host != "metabitcoin.unknown" {
				t = 1
			}
			addressMap[pev.Address] = &MetaSoBlockMDV{DataValue: pev.IncrementalValue, Block: blockHeight, Address: pev.Address, MetaId: pev.MetaId, PinNumber: 1, PinNumberHasHost: t}
		}
	}

	var hostList []*MetaSoBlockNDV
	var addressList []*MetaSoBlockMDV
	for _, value := range hostMap {
		value.HistoryValue, _ = getBlockHistoryValue(blockHeight, "host", value.Host)
		hostList = append(hostList, value)
		//fmt.Println(value.Host, value.DataValue)
	}
	for _, value := range addressMap {
		value.HistoryValue, _ = getBlockHistoryValue(blockHeight, "address", value.Address)
		addressList = append(addressList, value)
	}
	var models []mongo.WriteModel
	for _, item := range hostList {
		filter := bson.D{{Key: "host", Value: item.Host}, {Key: "block", Value: item.Block}}
		update := bson.D{{Key: "$set", Value: item}}
		m := mongo.NewUpdateOneModel()
		m.SetFilter(filter).SetUpdate(update).SetUpsert(true)
		models = append(models, m)
	}
	bulkWriteOptions := options.BulkWrite().SetOrdered(false)
	mongoClient.Collection(MetaSoNDVBlockData).BulkWrite(context.Background(), models, bulkWriteOptions)

	var models2 []mongo.WriteModel
	for _, item := range addressList {
		filter := bson.D{{Key: "address", Value: item.Address}, {Key: "block", Value: item.Block}}
		update := bson.D{{Key: "$set", Value: item}}
		m := mongo.NewUpdateOneModel()
		m.SetFilter(filter).SetUpdate(update).SetUpsert(true)
		models2 = append(models2, m)
	}
	mongoClient.Collection(MetaSoMDVBlockData).BulkWrite(context.Background(), models2, bulkWriteOptions)
	return
}
func UpdateDataValue(hostMap *map[string]struct{}, addressMap *map[string]struct{}) (err error) {
	for host := range *hostMap {
		total, err := getHostDataSum(host)
		if err == nil && total > 0 {
			data := MetaSoNDV{
				Host:      host,
				DataValue: total,
			}
			mongoClient.Collection(MetaSoNDVData).UpdateOne(context.TODO(), bson.M{"host": host}, bson.M{"$set": data}, options.Update().SetUpsert(true))
		}
	}
	for address := range *addressMap {
		total, err := getMetaDataSum(address)
		if err == nil && total > 0 {
			data := MetaSoMDV{
				MetaId:    common.GetMetaIdByAddress(address),
				Address:   address,
				DataValue: total,
			}
			mongoClient.Collection(MetaSoMDVData).UpdateOne(context.TODO(), bson.M{"address": address}, bson.M{"$set": data}, options.Update().SetUpsert(true))
			time.Sleep(time.Millisecond * 100)
		}
	}
	return
}
func getHostDataSum(host string) (dataValue float64, err error) {
	filter := bson.D{{Key: "host", Value: host}}
	match := bson.D{{Key: "$match", Value: filter}}
	groupStage := bson.D{
		{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$host"},
			{Key: "totalValue", Value: bson.D{{Key: "$sum", Value: "$incrementalvalue"}}},
		}}}
	cursor, err := mongoClient.Collection(MetaSoPEVData).Aggregate(context.TODO(), mongo.Pipeline{match, groupStage})

	//cursor, err := mongoClient.Collection(MetaSoPEVData).Aggregate(context.TODO(), pipeline)
	if err != nil {
		return
	}
	defer cursor.Close(context.TODO())
	var results []bson.M
	if err = cursor.All(context.TODO(), &results); err != nil {
		return
	}
	for _, result := range results {
		if result["_id"] == host {
			dataValue = convertFloat64(result["totalValue"])
			break
		}
	}
	return
}
func convertFloat64(value interface{}) float64 {
	switch v := value.(type) {
	case float64:
		return v
	case int32:
		return float64(v)
	case int64:
		return float64(v)
	case int:
		return float64(v)
	default:
		return float64(0)
	}
}
func getMetaDataSum(address string) (dataValue float64, err error) {
	pipeline := bson.A{
		bson.D{
			{Key: "$match", Value: bson.D{
				{Key: "address", Value: address},
			}},
		},
		bson.D{
			{Key: "$group", Value: bson.D{
				{Key: "_id", Value: "$address"},
				{Key: "totalValue", Value: bson.D{
					{Key: "$sum", Value: "$incrementalvalue"},
				}},
			}},
		},
	}
	cursor, err := mongoClient.Collection(MetaSoPEVData).Aggregate(context.TODO(), pipeline)
	if err != nil {
		return
	}
	defer cursor.Close(context.TODO())
	var results []bson.M
	if err = cursor.All(context.TODO(), &results); err != nil {
		return
	}
	for _, result := range results {
		if result["_id"] == address {
			dataValue = convertFloat64(result["totalValue"])
			break
		}
	}
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
func countPDV(blockHeight int64, block *MetaBlockChainData, pinNode *pin.PinInscription) (data PEVData, err error) {
	switch pinNode.Path {
	case "/follow":
		return countFollowPDV(blockHeight, block, pinNode)
	case "/protocols/simpledonate":
		return countDonatePDV(blockHeight, block, pinNode)
	case "/protocols/paylike":
		return countPayLike(blockHeight, block, pinNode)
	case "/protocols/paycomment":
		return countPaycomment(blockHeight, block, pinNode)
	case "/protocols/simplebuzz":
		return countSimplebuzz(blockHeight, block, pinNode)
	case "/ft/mrc20/mint":
		return countMrc20Mint(blockHeight, block, pinNode)
	default:
		data = createPDV(blockHeight, block, pinNode, pinNode, 1.0)
		return
	}
}
func createPDV(blockHeight int64, block *MetaBlockChainData, fromPIN *pin.PinInscription, toPIN *pin.PinInscription, value float64) PEVData {
	startHeight, _ := strconv.ParseInt(block.StartBlock, 10, 64)
	endHeight, _ := strconv.ParseInt(block.EndBlock, 10, 64)
	return PEVData{
		Host:             toPIN.Host,
		FromPINId:        fromPIN.Id,
		ToPINId:          toPIN.Id,
		Path:             fromPIN.Path,
		Address:          toPIN.Address,
		MetaId:           toPIN.MetaId,
		FromChainName:    fromPIN.ChainName,
		ToChainName:      toPIN.ChainName,
		MetaBlockHeight:  blockHeight,
		StartBlockHeight: startHeight,
		EndBlockHeight:   endHeight,
		BlockHeight:      fromPIN.GenesisHeight,
		IncrementalValue: float64(value),
	}
}
func countFollowPDV(blockHeight int64, block *MetaBlockChainData, pinNode *pin.PinInscription) (data PEVData, err error) {
	metaid := string(pinNode.ContentBody)
	filter := bson.M{"metaid": metaid}
	findOptions := options.FindOne()
	findOptions.SetSort(bson.D{{Key: "_id", Value: 1}})
	var toPIN pin.PinInscription
	err = mongoClient.Collection(mongodb.PinsCollection).FindOne(context.TODO(), filter, findOptions).Decode(&toPIN)
	if err != nil {
		return
	}
	data = createPDV(blockHeight, block, pinNode, &toPIN, 1.0)
	return
}
func getPINbyId(pinId string) (pinNode *pin.PinInscription, err error) {
	filter := bson.M{"id": pinId}
	err = mongoClient.Collection(mongodb.PinsCollection).FindOne(context.TODO(), filter, nil).Decode(&pinNode)
	return
}
func countDonatePDV(blockHeight int64, block *MetaBlockChainData, pinNode *pin.PinInscription) (data PEVData, err error) {
	var dataMap map[string]interface{}
	err = json.Unmarshal(pinNode.ContentBody, &dataMap)
	if err != nil {
		return
	}
	toPIN, err := getPINbyId(dataMap["toPin"].(string))
	if err != nil {
		return
	}
	data = createPDV(blockHeight, block, pinNode, toPIN, 1.0)
	return
}
func countPayLike(blockHeight int64, block *MetaBlockChainData, pinNode *pin.PinInscription) (data PEVData, err error) {
	var dataMap map[string]interface{}
	err = json.Unmarshal(pinNode.ContentBody, &dataMap)
	if err != nil {
		return
	}
	if dataMap["likeTo"].(string) == "" || dataMap["isLike"].(string) != "1" {
		return
	}
	toPIN, err := getPINbyId(dataMap["likeTo"].(string))
	if err != nil {
		return
	}
	data = createPDV(blockHeight, block, pinNode, toPIN, 1.0)
	return
}
func countPaycomment(blockHeight int64, block *MetaBlockChainData, pinNode *pin.PinInscription) (data PEVData, err error) {
	var dataMap map[string]interface{}
	err = json.Unmarshal(pinNode.ContentBody, &dataMap)
	if err != nil {
		return
	}
	if dataMap["commentTo"].(string) == "" {
		return
	}
	toPIN, err := getPINbyId(dataMap["commentTo"].(string))
	if err != nil {
		return
	}
	data = createPDV(blockHeight, block, pinNode, toPIN, 1.0)
	return
}
func countSimplebuzz(blockHeight int64, block *MetaBlockChainData, pinNode *pin.PinInscription) (data PEVData, err error) {
	var dataMap map[string]interface{}
	err = json.Unmarshal(pinNode.ContentBody, &dataMap)
	if err != nil {
		return
	}
	if dataMap["quotePin"] == nil || dataMap["quotePin"].(string) == "" {
		data = createPDV(blockHeight, block, pinNode, pinNode, 1.0)
		return
	}
	toPIN, err := getPINbyId(dataMap["quotePin"].(string))
	if err != nil {
		return
	}
	data = createPDV(blockHeight, block, pinNode, toPIN, 1.0)
	return
}
func countMrc20Mint(blockHeight int64, block *MetaBlockChainData, pinNode *pin.PinInscription) (data PEVData, err error) {
	var dataMap map[string]interface{}
	err = json.Unmarshal(pinNode.ContentBody, &dataMap)
	if err != nil {
		return
	}
	if dataMap["id"].(string) == "" {
		return
	}
	toPIN, err := getPINbyId(dataMap["id"].(string))
	if err != nil {
		return
	}
	data = createPDV(blockHeight, block, pinNode, toPIN, 1.0)
	return
}
