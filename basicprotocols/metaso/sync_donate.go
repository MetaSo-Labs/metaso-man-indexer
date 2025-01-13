package metaso

import (
	"context"
	"encoding/json"
	"manindexer/common"
	"manindexer/database/mongodb"

	"github.com/shopspring/decimal"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func (metaso *MetaSo) getLastDonate() (pinList []*Tweet, err error) {
	last, err := mongodb.GetSyncLastId("metasodonate")
	if err != nil {
		return
	}
	filter := bson.D{
		{Key: "path", Value: "/protocols/simpledonate"},
	}
	if last != primitive.NilObjectID {
		filter = append(filter, bson.E{Key: "_id", Value: bson.D{{Key: "$gt", Value: last}}})
	}
	findOptions := options.Find()
	findOptions.SetSort(bson.D{{Key: "_id", Value: 1}})
	findOptions.SetLimit(500)
	result, err := mongoClient.Collection(mongodb.PinsCollection).Find(context.TODO(), filter, findOptions)
	if err != nil {
		return
	}
	err = result.All(context.TODO(), &pinList)
	return
}

type DonateProtocols struct {
	CreateTime string `json:"createTime"`
	To         string `json:"to"`
	CoinType   string `json:"coinType"`
	Amount     string `json:"amount"`
	ToPin      string `json:"toPin"`
	Message    string `json:"message"`
}

func (metaso *MetaSo) getSynchMeatsoDonate(pinList []*Tweet) (dataList []*MetasoDonate, err error) {
	var pinIdList []string
	pinMap := make(map[string][]*MetasoDonate)
	for _, pinNode := range pinList {
		var pinDonate DonateProtocols
		err := json.Unmarshal(pinNode.ContentBody, &pinDonate)
		if err != nil {
			continue
		}
		amt, err := decimal.NewFromString(pinDonate.Amount)
		if err != nil {
			continue
		}
		pinIdList = append(pinIdList, pinDonate.ToPin)
		donate := MetasoDonate{
			PinId:         pinNode.Id,
			PinNumber:     pinNode.Number,
			ChainName:     pinNode.ChainName,
			CreateAddress: pinNode.Address,
			CreateMetaid:  common.GetMetaIdByAddress(pinNode.Address),
			Timestamp:     pinNode.Timestamp,
			CreateTime:    pinDonate.CreateTime,
			ToAddress:     pinDonate.To,
			CoinType:      pinDonate.CoinType,
			Amount:        amt,
			ToPin:         pinDonate.ToPin,
			Message:       pinDonate.Message,
		}
		pinMap[pinDonate.ToPin] = append(pinMap[pinDonate.ToPin], &donate)
	}
	filter2 := bson.M{"id": bson.M{"$in": pinIdList}}
	result2, err := mongoClient.Collection(TweetCollection).Find(context.TODO(), filter2)
	if err != nil {
		return
	}
	var list []Tweet
	err = result2.All(context.TODO(), &list)
	if err != nil {
		return
	}
	for _, item := range list {
		dataList = append(dataList, pinMap[item.Id]...)
	}
	return
}

func (metaso *MetaSo) synchMeatsoDonate() (err error) {
	pinList, err := metaso.getLastDonate()

	if len(pinList) <= 0 {
		return
	}
	var lastId primitive.ObjectID
	for _, pinNode := range pinList {
		if mongodb.CompareObjectIDs(pinNode.MogoID, lastId) > 0 {
			lastId = pinNode.MogoID
		}
	}
	mongodb.UpdateSyncLastIdLog("metasodonate", lastId)
	list, err := metaso.getSynchMeatsoDonate(pinList)
	if len(list) <= 0 {
		return
	}
	var dataList []interface{}
	cntMap := make(map[string]int)
	for _, item := range list {

		dataList = append(dataList, item)
		cntMap[item.ToPin] += 1
	}
	metaso.updateSynchMeatsoDonate(dataList, cntMap)
	return
}
func (metaso *MetaSo) updateSynchMeatsoDonate(dataList []interface{}, cntMap map[string]int) (err error) {
	if len(dataList) > 0 {
		insertOpts := options.InsertMany().SetOrdered(false)
		_, err1 := mongoClient.Collection(MetaSoDonateData).InsertMany(context.TODO(), dataList, insertOpts)
		if err1 != nil {
			err = err1
			return
		}
	}

	if len(cntMap) > 0 {
		var models []mongo.WriteModel
		for pinid, cnt := range cntMap {
			filter := bson.D{{Key: "id", Value: pinid}}
			var updateInfo bson.D
			updateInfo = append(updateInfo, bson.E{Key: "donatecount", Value: cnt})
			update := bson.D{{Key: "$inc", Value: updateInfo}}
			m := mongo.NewUpdateOneModel()
			m.SetFilter(filter).SetUpdate(update)
			models = append(models, m)
		}
		bulkWriteOptions := options.BulkWrite().SetOrdered(false)
		mongoClient.Collection(TweetCollection).BulkWrite(context.Background(), models, bulkWriteOptions)
	}
	return
}
