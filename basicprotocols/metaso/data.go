package metaso

import (
	"context"
	"fmt"
	"manindexer/common"
	"manindexer/database/mongodb"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	BlockedData map[string]struct{}
	_typeList   = []string{"metaid", "host", "pinid"}
)

func (metaso *MetaSo) Synchronization() {
	BlockedData = map[string]struct{}{}
	fixHost()
	for {
		metaso.synchTweet()
		metaso.synchTweetLike()
		metaso.synchMeatsoDonate()
		metaso.synchTweetComment()
		metaso.syncHostData()
		metaso.syncMrc20TickData()
		metaso.synchMempoolData()
		time.Sleep(time.Second * 3)
	}
}
func (metaso *MetaSo) SyncPEV() (err error) {
	for {
		metaso.syncPEV()
		time.Sleep(time.Second * 10)
	}
}
func fixHost() {
	fixed, _ := mongodb.GetSyncLastNumber("fixhost")
	if fixed == -1 {
		mongoClient.Collection(TweetCollection).DeleteMany(context.TODO(), bson.D{})
		mongoClient.Collection("sync_lastid_log").DeleteOne(context.TODO(), bson.M{"key": "tweet"})
	}
	mongodb.UpdateSyncLastNumber("fixhost", 1)
}
func (metaso *MetaSo) SyncPendingPEVF() (err error) {
	for {
		metaso.syncPendingPEV()
		time.Sleep(time.Minute * 5)
	}
}
func (metaso *MetaSo) SynchBlockedSettings() (err error) {
	for {
		metaso.synchBlockedSettings()
		time.Sleep(time.Minute * 3)
	}
}
func (metaso *MetaSo) synchBlockedSettings() (err error) {
	BlockedData = map[string]struct{}{}
	for _, tp := range _typeList {
		list1, _, err1 := getBlockedList(tp, 0, 10000)
		if err1 == nil {
			for _, item := range list1 {
				key := fmt.Sprintf("%s_%s", tp, item.BlockedContent)
				BlockedData[key] = struct{}{}
			}
		}
	}
	return
}
func (metaso *MetaSo) synchTweet() (err error) {
	last, err := mongodb.GetSyncLastId("tweet")
	if err != nil {
		return
	}
	var pinList []*Tweet
	// filter := bson.D{
	// 	{Key: "path", Value: "/protocols/simplebuzz"},
	// }
	filter := DataFilter
	if last != primitive.NilObjectID {
		filter = append(filter, bson.E{Key: "_id", Value: bson.D{{Key: "$gt", Value: last}}})
	}
	findOptions := options.Find()
	findOptions.SetSort(bson.D{{Key: "_id", Value: 1}})
	findOptions.SetLimit(500)
	result, err := mongoClient.Collection(mongodb.PinsCollection).Find(context.TODO(), filter)
	if err != nil {
		return
	}
	result.All(context.TODO(), &pinList)
	if len(pinList) <= 0 {
		return
	}

	var insertDocs []interface{}
	var lastId primitive.ObjectID
	onlyHost := common.Config.MetaSo.OnlyHost
	for _, doc := range pinList {
		if onlyHost != "" && doc.Host != onlyHost {
			continue
		}
		insertDocs = append(insertDocs, doc)
		if mongodb.CompareObjectIDs(doc.MogoID, lastId) > 0 {
			lastId = doc.MogoID
		}
	}
	insertOpts := options.InsertMany().SetOrdered(false)
	_, err1 := mongoClient.Collection(TweetCollection).InsertMany(context.TODO(), insertDocs, insertOpts)
	if err1 != nil {
		err = err1
		return
	}
	mongodb.UpdateSyncLastIdLog("tweet", lastId)
	return
}
