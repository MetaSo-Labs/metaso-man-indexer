package metaso

import (
	"context"
	"time"

	"github.com/shopspring/decimal"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type metaBlockHostInfo struct {
	MetaBlockHeight       int64           `json:"metaBlockHeight"`
	MetaBlockHash         string          `json:"metaBlockHash"`
	MetaBlockTime         int64           `json:"metaBlockTime"`
	PreviousMetaBlockHash string          `json:"previousMetaBlockHash"`
	StartBlock            string          `json:"startBlock"`
	EndBlock              string          `json:"endBlock"`
	Pins                  int64           `json:"pins"`
	PinsInHost            int64           `json:"pinsInHost"`
	MdvValue              decimal.Decimal `json:"mdvValue"`
	MdvDeltaValue         decimal.Decimal `json:"mdvDeltaValue"`
	Total                 int64           `json:"total"`
}
type metaBlockHostItem struct {
	Host          string          `json:"host"`
	Pins          int64           `json:"pins"`
	MdvValue      decimal.Decimal `json:"mdvValue"`
	MdvDeltaValue decimal.Decimal `json:"mdvDeltaValue"`
	BlockHeight   int64           `json:"blockHeight"`
	BlockTime     int64           `json:"blockTime"`
}
type metaBlockAddressItem struct {
	MetaId        string          `json:"metaId"`
	Address       string          `json:"address"`
	Pins          int64           `json:"pins"`
	PinsInHost    int64           `json:"pinsInHost"`
	MdvValue      decimal.Decimal `json:"mdvValue"`
	MdvDeltaValue decimal.Decimal `json:"mdvDeltaValue"`
	BlockHeight   int64           `json:"blockHeight"`
	BlockTime     int64           `json:"blockTime"`
}

func getBlockNDVPageList(height, cursor, size int64) (info metaBlockHostInfo, list []*metaBlockHostItem, err error) {
	filter := bson.D{{Key: "block", Value: height}}
	var block MetaSoBlockInfo
	mongoClient.Collection(MetaSoBlockInfoData).FindOne(context.TODO(), filter).Decode(&block)

	info.MetaBlockHeight = block.MetaBlock.MetablockHeight
	info.MetaBlockHash = block.MetaBlock.Header
	info.MetaBlockTime = block.MetaBlock.Timestamp
	info.PreviousMetaBlockHash = block.MetaBlock.PreHeader
	info.Pins = block.PinNumber
	info.MdvValue = block.DataValue.Add(block.HistoryValue)
	info.MdvDeltaValue = block.DataValue
	info.Total = block.HostNumber
	for _, chain := range block.MetaBlock.Chains {
		if chain.Chain == "Bitcoin" {
			info.StartBlock = chain.StartBlock
			info.EndBlock = chain.EndBlock
		}
	}

	findOptions := options.Find()
	findOptions.SetSort(bson.D{{Key: "datavalue", Value: -1}})
	findOptions.SetSkip(cursor).SetLimit(size)
	result, err := mongoClient.Collection(MetaSoNDVBlockData).Find(context.TODO(), filter, findOptions)
	if err != nil {
		return
	}
	var ndvList []*MetaSoBlockNDV
	err = result.All(context.TODO(), &ndvList)
	if err == mongo.ErrNoDocuments {
		err = nil
		return
	}
	for _, ndv := range ndvList {
		item := &metaBlockHostItem{
			Host:          ndv.Host,
			MdvDeltaValue: ndv.DataValue,
			MdvValue:      ndv.DataValue.Add(ndv.HistoryValue),
			Pins:          ndv.PinNumber,
		}
		list = append(list, item)
	}
	return
}
func getHostValuePageList(heightBegin, heightEnd, timeBegin, timeEnd int64, host string, cursor, size int64) (list []*metaBlockHostItem, total int64, err error) {
	filter := bson.D{}
	if heightBegin >= -1 && heightEnd >= -1 {
		filter = append(filter, bson.E{Key: "block", Value: bson.D{{Key: "$gte", Value: heightBegin}, {Key: "$lte", Value: heightEnd}}})
	}
	if timeBegin >= 0 && timeEnd > timeBegin {
		filter = append(filter, bson.E{Key: "blocktime", Value: bson.D{{Key: "$gte", Value: timeBegin}, {Key: "$lte", Value: timeEnd}}})
	}
	if host != "" {
		filter = append(filter, bson.E{Key: "host", Value: host})
	}

	findOptions := options.Find()
	findOptions.SetSort(bson.D{{Key: "datavalue", Value: -1}})
	findOptions.SetSkip(cursor).SetLimit(size)
	result, err := mongoClient.Collection(MetaSoNDVBlockData).Find(context.TODO(), filter, findOptions)
	if err != nil {
		return
	}
	var ndvList []*MetaSoBlockNDV
	err = result.All(context.TODO(), &ndvList)
	if err == mongo.ErrNoDocuments {
		err = nil
		return
	}
	for _, ndv := range ndvList {
		item := &metaBlockHostItem{
			Host:          ndv.Host,
			MdvDeltaValue: ndv.DataValue,
			MdvValue:      ndv.DataValue.Add(ndv.HistoryValue),
			Pins:          ndv.PinNumber,
			BlockHeight:   ndv.Block,
			BlockTime:     ndv.BlockTime,
		}
		list = append(list, item)
	}
	total, err = mongoClient.Collection(MetaSoNDVBlockData).CountDocuments(context.TODO(), filter)
	return
}

type hostAddressValueRes struct {
	Address   interface{} `json:"address"`
	DataValue interface{} `json:"dataValue"`
}

func getHostAddressValuePageList(heightBegin, heightEnd, timeBegin, timeEnd int64, host string, cursor, size int64) (list []*hostAddressValueRes, total interface{}, err error) {
	if host == "" {
		host = "metabitcoin.unknown"
	}
	filter := bson.D{}
	if heightBegin >= -1 && heightEnd >= -1 {
		filter = append(filter, bson.E{Key: "block", Value: bson.D{{Key: "$gte", Value: heightBegin}, {Key: "$lte", Value: heightEnd}}})
	}
	if timeBegin > 0 && timeEnd > 0 {
		filter = append(filter, bson.E{Key: "blocktime", Value: bson.D{{Key: "$gte", Value: timeBegin}, {Key: "$lte", Value: timeEnd}}})
	}
	if host != "" {
		filter = append(filter, bson.E{Key: "host", Value: host})
	}

	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: filter}},
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$address"},
			{Key: "totalValue", Value: bson.D{{Key: "$sum", Value: "$datavalue"}}},
		}}},
		{{Key: "$sort", Value: bson.D{{Key: "totalValue", Value: -1}}}},
		{{Key: "$facet", Value: bson.D{
			{Key: "data", Value: bson.A{
				bson.D{{Key: "$skip", Value: cursor}},
				bson.D{{Key: "$limit", Value: size}},
			}},
			{Key: "total", Value: bson.A{
				bson.D{{Key: "$count", Value: "total"}},
			}},
		}}},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()

	dbcursor, err := mongoClient.Collection(MetaSoHostAddressData).Aggregate(ctx, pipeline)
	if err != nil {
		return
	}
	defer dbcursor.Close(ctx)
	var results []bson.M
	if err = dbcursor.All(ctx, &results); err != nil {
		return
	}
	if len(results) <= 0 {
		return
	}
	data := results[0]["data"].(bson.A)
	dbtotal := results[0]["total"].(bson.A)
	if len(dbtotal) > 0 {
		total = dbtotal[0].(bson.M)["total"]
	}
	for _, item := range data {
		list = append(list, &hostAddressValueRes{Address: item.(bson.M)["_id"], DataValue: item.(bson.M)["totalValue"]})
	}
	return
}
func getHostAddressValue(heightBegin, heightEnd, timeBegin, timeEnd int64, host string, address string, cursor, size int64) (list []*MetaSoHostAddress, total int64, err error) {
	if address == "" {
		return
	}
	if host == "" {
		host = "metabitcoin.unknown"
	}
	filter := bson.D{}
	filter = append(filter, bson.E{Key: "address", Value: address})
	if heightBegin >= -1 && heightEnd >= -1 {
		filter = append(filter, bson.E{Key: "block", Value: bson.D{{Key: "$gte", Value: heightBegin}, {Key: "$lte", Value: heightEnd}}})
	}
	if timeBegin > 0 && timeEnd > 0 {
		filter = append(filter, bson.E{Key: "blocktime", Value: bson.D{{Key: "$gte", Value: timeBegin}, {Key: "$lte", Value: timeEnd}}})
	}
	if host != "" {
		filter = append(filter, bson.E{Key: "host", Value: host})
	}

	findOptions := options.Find()
	findOptions.SetSort(bson.D{{Key: "datavalue", Value: -1}})
	findOptions.SetSkip(cursor).SetLimit(size)
	result, err := mongoClient.Collection(MetaSoHostAddressData).Find(context.TODO(), filter, findOptions)
	if err != nil {
		return
	}

	err = result.All(context.TODO(), &list)
	if err == mongo.ErrNoDocuments {
		err = nil
		return
	}
	total, err = mongoClient.Collection(MetaSoHostAddressData).CountDocuments(context.TODO(), filter)
	return
}
func getBlockMDVPageList(height, cursor, size int64) (info metaBlockHostInfo, list []*metaBlockAddressItem, err error) {
	filter := bson.D{{Key: "block", Value: height}}
	var block MetaSoBlockInfo
	mongoClient.Collection(MetaSoBlockInfoData).FindOne(context.TODO(), filter).Decode(&block)

	info.MetaBlockHeight = block.MetaBlock.MetablockHeight
	info.MetaBlockHash = block.MetaBlock.Header
	info.MetaBlockTime = block.MetaBlock.Timestamp
	info.PreviousMetaBlockHash = block.MetaBlock.PreHeader
	info.Pins = block.PinNumber
	info.MdvValue = block.DataValue.Add(block.HistoryValue)
	info.MdvDeltaValue = block.DataValue
	info.Total = block.AddressNumber
	info.PinsInHost = block.PinNumberHasHost
	for _, chain := range block.MetaBlock.Chains {
		if chain.Chain == "Bitcoin" {
			info.StartBlock = chain.StartBlock
			info.EndBlock = chain.EndBlock
		}
	}

	findOptions := options.Find()
	findOptions.SetSort(bson.D{{Key: "datavalue", Value: -1}})
	findOptions.SetSkip(cursor).SetLimit(size)
	result, err := mongoClient.Collection(MetaSoMDVBlockData).Find(context.TODO(), filter, findOptions)
	if err != nil {
		return
	}
	var mdvList []*MetaSoBlockMDV
	err = result.All(context.TODO(), &mdvList)
	if err == mongo.ErrNoDocuments {
		err = nil
		return
	}
	for _, mdv := range mdvList {
		item := &metaBlockAddressItem{
			MetaId:        mdv.MetaId,
			Address:       mdv.Address,
			MdvDeltaValue: mdv.DataValue,
			MdvValue:      mdv.DataValue.Add(mdv.HistoryValue),
			Pins:          mdv.PinNumber,
			PinsInHost:    mdv.PinNumberHasHost,
		}
		list = append(list, item)
	}
	return
}
