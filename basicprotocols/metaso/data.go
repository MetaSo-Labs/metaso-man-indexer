package metaso

import (
	"context"
	"fmt"
	"log"
	"manindexer/common"
	"manindexer/database/mongodb"
	"path"
	"time"

	"github.com/yanyiwu/gojieba"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	_typeList = []string{"metaid", "host", "pinid"}
	jiebax    *gojieba.Jieba
)

func (metaso *MetaSo) Synchronization() {

	//fixHost()
	//fixStatistics()
	dictDir := "./jieba_dict"
	jiebaPath := path.Join(dictDir, "jieba.dict.utf8")
	hmmPath := path.Join(dictDir, "hmm_model.utf8")
	userPath := path.Join(dictDir, "user.dict.utf8")
	idfPath := path.Join(dictDir, "idf.utf8")
	stopPath := path.Join(dictDir, "stop_words.utf8")
	jiebax = gojieba.NewJieba(jiebaPath, hmmPath, userPath, idfPath, stopPath)

	defer jiebax.Free()
	for {
		metaso.synchTweet()
		metaso.synchTweetLike()
		metaso.synchMeatsoDonate()
		metaso.synchTweetComment()
		metaso.syncHostData()
		metaso.syncMrc20TickData()
		metaso.synchMempoolData()
		time.Sleep(time.Second * 10)
	}
}
func (metaso *MetaSo) SyncPEV() (err error) {
	//fixStatistics()
	//for {
	// if !man.FirstCompleted {
	// 	time.Sleep(time.Minute * 1)
	// 	log.Println("waiting for first completed...")
	// 	continue
	// }
	// PEV 从独立的程序统计，先注释
	//metaso.syncPEV()
	metaso.SyncPendingPEV()
	//	time.Sleep(time.Second * 10)
	//}
	return
}
func fixHost() {
	fixed, _ := mongodb.GetSyncLastNumber("fixhost")
	if fixed == -1 {
		mongoClient.Collection(TweetCollection).DeleteMany(context.TODO(), bson.D{})
		mongoClient.Collection("sync_lastid_log").DeleteOne(context.TODO(), bson.M{"key": "tweet"})
	}
	mongodb.UpdateSyncLastNumber("fixhost", 1)
}
func fixStatistics() {
	fixed, _ := mongodb.GetSyncLastNumber("fixstatistics")
	fixedTarger := int64(23)
	if fixed != fixedTarger {
		mongoClient.Collection(MetaSoPEVData).DeleteMany(context.TODO(), bson.D{})
		mongoClient.Collection(MetaSoMDVData).DeleteMany(context.TODO(), bson.D{})
		mongoClient.Collection(MetaSoNDVData).DeleteMany(context.TODO(), bson.D{})
		mongoClient.Collection(MetaSoMDVBlockData).DeleteMany(context.TODO(), bson.D{})
		mongoClient.Collection(MetaSoNDVBlockData).DeleteMany(context.TODO(), bson.D{})
		mongoClient.Collection(MetaSoBlockInfoData).DeleteMany(context.TODO(), bson.D{})
		mongoClient.Collection(MetaSoHostAddressData).DeleteMany(context.TODO(), bson.D{})
		mongoClient.Collection(TweetCollection).DeleteMany(context.TODO(), bson.D{})
		mongoClient.Collection("sync_lastid_log").DeleteOne(context.TODO(), bson.M{"key": "metablock"})
		mongoClient.Collection("sync_lastid_log").DeleteOne(context.TODO(), bson.M{"key": "tweet"})
		// if fixedTarger == 22 {
		// 	for i := 892312; i <= 894039; i++ {
		// 		man.DoIndexerRun("btc", int64(i), true)
		// 		fmt.Println("btc reindex", i)
		// 	}
		// 	for i := 117006; i <= 118681; i++ {
		// 		man.DoIndexerRun("mvc", int64(i), true)
		// 		fmt.Println("mvc reindex", i)
		// 	}
		// }
	}
	mongodb.UpdateSyncLastNumber("fixstatistics", fixedTarger)
}
func (metaso *MetaSo) SyncPendingPEVF() (err error) {
	for {
		metaso.SyncPendingPEV()
		time.Sleep(time.Minute * 2)
	}
}
func (metaso *MetaSo) SynchBlockedSettings() (err error) {
	for {
		metaso.updateCollection("isintblocked", 3, "blocked", TweetCollection)
		metaso.updateCollection("isintblocked", 3, "blocked", mongodb.MempoolPinsCollection)
		metaso.SaveSynchBlockedSetting()
		metaso.updateCollection("isintrecommended", 2, "isrecommended", TweetCollection)
		metaso.updateCollection("isintrecommended", 2, "isrecommended", mongodb.MempoolPinsCollection)
		metaso.SaveRecommendedAuthor()
		time.Sleep(time.Minute * 3)
	}
}

func (metaso *MetaSo) updateCollection(flag string, sn int64, key string, collection string) (err error) {
	isint, _ := mongodb.GetSyncLastNumber(flag)
	if isint == sn {
		return
	}
	batchSize := 1000
	filter := bson.M{key: bson.M{"$exists": false}}
	for {
		cursor, err := mongoClient.Collection(collection).Find(
			context.TODO(),
			filter,
			options.Find().SetProjection(bson.M{"_id": 1}).SetLimit(int64(batchSize)),
		)
		defer cursor.Close(context.TODO())
		if err != nil {
			fmt.Println(err)
			return err
		}
		var ids []primitive.ObjectID
		for cursor.Next(context.TODO()) {
			var doc bson.M
			if err := cursor.Decode(&doc); err != nil {
				return err
			}
			if id, ok := doc["_id"].(primitive.ObjectID); ok {
				ids = append(ids, id)
			}
		}
		if len(ids) == 0 {
			break
		}
		cursor.Close(context.TODO())
		var updates []mongo.WriteModel
		for _, id := range ids {
			update := mongo.NewUpdateOneModel().
				SetFilter(bson.M{"_id": id}).
				SetUpdate(bson.M{"$set": bson.M{key: false}})
			updates = append(updates, update)
		}
		_, err = mongoClient.Collection(collection).BulkWrite(context.TODO(), updates)
		if err != nil {
			log.Printf("batchUpdate %s fail: %v", collection, err)
			return err
		}
	}

	mongodb.UpdateSyncLastNumber(flag, sn)

	return
}
func (metaso *MetaSo) SaveRecommendedAuthor() (err error) {
	list, _, err := metaso.GetRecommendedAuthors(context.Background(), 0, 1000)
	if err != nil {
		return
	}
	for _, item := range list {
		common.RecommendedAuthor[item.AuthorID] = struct{}{}
	}
	for k, _ := range common.RecommendedAuthor {
		fmt.Println(k)
	}
	return
}
func (metaso *MetaSo) SaveSynchBlockedSetting() (err error) {
	for _, tp := range _typeList {
		list1, _, err1 := getBlockedList(tp, 0, 10000)
		if err1 == nil {
			for _, item := range list1 {
				key := fmt.Sprintf("%s_%s", tp, item.BlockedContent)
				common.BlockedData[key] = struct{}{}
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
		if doc.Path == "/protocols/simplebuzz" {
			doc.Keywords = jiebax.Cut(string(doc.ContentBody), true)
		}
		//check blocked
		hostKey := fmt.Sprintf("host_%s", doc.Host)
		metaidKey := fmt.Sprintf("metaid_%s", doc.CreateMetaId)
		pinidKey := fmt.Sprintf("pinid_%s", doc.Id)
		if _, ok := common.BlockedData[hostKey]; ok {
			doc.Blocked = true
		}
		if _, ok := common.BlockedData[metaidKey]; ok {
			doc.Blocked = true
		}
		if _, ok := common.BlockedData[pinidKey]; ok {
			doc.Blocked = true
		}
		if _, ok := common.RecommendedAuthor[doc.Address]; ok {
			doc.IsRecommended = true
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
