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
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	BlockedData map[string]struct{}
	_typeList   = []string{"metaid", "host", "pinid"}
	jiebax      *gojieba.Jieba
)

func (metaso *MetaSo) Synchronization() {
	BlockedData = map[string]struct{}{}
	fixHost()
	//fixStatistics()
	dictDir := "./jieba_dict"
	jiebaPath := path.Join(dictDir, "jieba.dict.utf8")
	hmmPath := path.Join(dictDir, "hmm_model.utf8")
	userPath := path.Join(dictDir, "user.dict.utf8")
	idfPath := path.Join(dictDir, "idf.utf8")
	stopPath := path.Join(dictDir, "stop_words.utf8")
	jiebax = gojieba.NewJieba(jiebaPath, hmmPath, userPath, idfPath, stopPath)

	defer jiebax.Free()
	// for {
	// 	metaso.synchTweet()
	// 	metaso.synchTweetLike()
	// 	metaso.synchMeatsoDonate()
	// 	metaso.synchTweetComment()
	// 	metaso.syncHostData()
	// 	metaso.syncMrc20TickData()
	// 	metaso.synchMempoolData()
	// 	time.Sleep(time.Second * 3)
	// }
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if err := metaso.synchTweet(); err != nil {
				log.Printf("Error synching tweets: %v\n", err)
			}
			if err := metaso.synchTweetLike(); err != nil {
				log.Printf("Error synching tweet likes: %v\n", err)
			}
			if err := metaso.synchMeatsoDonate(); err != nil {
				log.Printf("Error synching Meatso donations: %v\n", err)
			}
			if err := metaso.synchTweetComment(); err != nil {
				log.Printf("Error synching tweet comments: %v\n", err)
			}
			if err := metaso.syncHostData(); err != nil {
				log.Printf("Error syncing host data: %v\n", err)
			}
			if err := metaso.syncMrc20TickData(); err != nil {
				log.Printf("Error syncing MRC20 tick data: %v\n", err)
			}
			if err := metaso.synchMempoolData(); err != nil {
				log.Printf("Error synching mempool data: %v\n", err)
			}
		case <-ctx.Done():
			log.Println("Shutting down...")
			return
		}
	}
}
func (metaso *MetaSo) SyncPEV() (err error) {
	fixStatistics()
	// for {
	// 	metaso.syncPendingPEV()
	// 	metaso.syncPEV()
	// 	time.Sleep(time.Second * 5)
	// }
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if err := metaso.syncPendingPEV(); err != nil {
				log.Printf("Error syncPendingPEV: %v\n", err)
			}
			if err := metaso.syncPEV(); err != nil {
				log.Printf("Error syncPEV: %v\n", err)
			}
		case <-ctx.Done():
			log.Println("Shutting down...")
			return
		}
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
func fixStatistics() {
	fixed, _ := mongodb.GetSyncLastNumber("fixstatistics")
	fixedTarger := int64(22)
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
		metaso.syncPendingPEV()
		time.Sleep(time.Minute * 2)
	}
}
func (metaso *MetaSo) SynchBlockedSettings() (err error) {
	// for {
	// 	metaso.synchBlockedSettings()
	// 	time.Sleep(time.Minute * 3)
	// }
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if err := metaso.synchBlockedSettings(); err != nil {
				log.Printf("Error synchBlockedSettings: %v\n", err)
			}
		case <-ctx.Done():
			log.Println("Shutting down...")
			return
		}
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
		if doc.Path == "/protocols/simplebuzz" {
			doc.Keywords = jiebax.Cut(string(doc.ContentBody), true)
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
