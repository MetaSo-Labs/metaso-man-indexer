package metaso

import (
	"context"
	"fmt"
	"log"
	"manindexer/common"
	"manindexer/common/mongo_util"
	"manindexer/database/mongodb"
	"reflect"
	"time"

	"github.com/shopspring/decimal"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

type Mongodb struct{}

var (
	mongoClient *mongo.Database
)

const (
	TweetCollection         string = "metaso_tweet"
	TweetCountCollection    string = "metaso_tweet_count"
	TweetLikeCollection     string = "metaso_tweet_like"
	TweetCommentCollection  string = "metaso_sync_comment"
	BuzzView                string = "buzzview"
	HostDataCollection      string = "host_data"
	MetasoTickCollection    string = "metaso_tick"
	MetaSoMempoolCollection string = "metaso_mempool"
	MetaSoPEVData           string = "metaso_pevdata"
	MetaSoMDVData           string = "metaso_mdvdata"
	MetaSoNDVData           string = "metaso_ndvdata"
	MetaSoMDVBlockData      string = "metaso_block_mdvdata"
	MetaSoNDVBlockData      string = "metaso_block_ndvdata"
	MetaSoBlockInfoData     string = "metaso_block_info"
	MetaSoHostAddressData   string = "metaso_host_address"
	MetaSoDonateData        string = "metaso_donate_data"
	BlockedSettingData      string = "metaso_blocked_settings"
	RecommendedAuthors      string = "metaso_recommended_authors"
)

var DataFilter = bson.D{
	{Key: "$or", Value: bson.A{
		bson.D{{Key: "path", Value: "/protocols/simplebuzz"}},
		bson.D{{Key: "path", Value: "/protocols/banana"}},
		bson.D{{Key: "path", Value: "/protocols/paybuzz"}},
		bson.D{{Key: "path", Value: "/protocols/subscribebuzz"}},
	}},
}

func ConnectMongoDb() {
	mg := common.Config.MongoDb
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(mg.TimeOut))
	defer cancel()
	o := options.Client().ApplyURI(mg.MongoURI)
	o.SetMaxPoolSize(uint64(mg.PoolSize))
	o.SetRegistry(bson.NewRegistryBuilder().
		RegisterDecoder(reflect.TypeOf(decimal.Decimal{}), mongo_util.Decimal{}).
		RegisterEncoder(reflect.TypeOf(decimal.Decimal{}), mongo_util.Decimal{}).
		Build())
	client, err := mongo.Connect(ctx, o)
	if err != nil {
		log.Panic("ConnectToDB", err)
		return
	}
	if err = client.Ping(context.Background(), readpref.Primary()); err != nil {
		log.Panic("ConnectToDB", err)
		return
	}
	mongoClient = client.Database(mg.DbName)
	createIndex(mongoClient)
	createBuzzView()
}
func Decimal128ToDecimal(d primitive.Decimal128) (decimal.Decimal, error) {
	decimalStr := d.String()
	result, err := decimal.NewFromString(decimalStr)
	if err != nil {
		return decimal.Zero, fmt.Errorf("Decimal128ToDecimal faild: %v", err)
	}
	return result, nil
}
func createIndex(mongoClient *mongo.Database) {
	//Tweet
	mongo_util.CreateIndexIfNotExists(mongoClient, TweetCollection, "pinid_1", bson.D{{Key: "id", Value: 1}}, true)
	mongo_util.CreateIndexIfNotExists(mongoClient, TweetCollection, "output_1", bson.D{{Key: "output", Value: 1}}, false)
	mongo_util.CreateIndexIfNotExists(mongoClient, TweetCollection, "path_1", bson.D{{Key: "path", Value: 1}}, false)
	mongo_util.CreateIndexIfNotExists(mongoClient, TweetCollection, "chainname_1", bson.D{{Key: "chainname", Value: 1}}, false)
	mongo_util.CreateIndexIfNotExists(mongoClient, TweetCollection, "timestamp_1", bson.D{{Key: "timestamp", Value: 1}}, false)
	mongo_util.CreateIndexIfNotExists(mongoClient, TweetCollection, "metaid_1", bson.D{{Key: "metaid", Value: 1}}, false)
	mongo_util.CreateIndexIfNotExists(mongoClient, TweetCollection, "creatormetaid_1", bson.D{{Key: "creatormetaid", Value: 1}}, false)
	mongo_util.CreateIndexIfNotExists(mongoClient, TweetCollection, "number_1", bson.D{{Key: "number", Value: 1}}, false)
	mongo_util.CreateIndexIfNotExists(mongoClient, TweetCollection, "operation_1", bson.D{{Key: "operation", Value: 1}}, false)
	mongo_util.CreateIndexIfNotExists(mongoClient, TweetCollection, "blocked_1", bson.D{{Key: "blocked", Value: 1}}, false)
	mongo_util.CreateTextIndexIfNotExists(mongoClient, TweetCollection, "tweet_text_1", []string{"keywords"})
	mongo_util.CreateIndexIfNotExists(mongoClient, TweetCollection, "isrecommended_1", bson.D{{Key: "isrecommended", Value: 1}}, false)
	//payLike
	mongo_util.CreateIndexIfNotExists(mongoClient, TweetLikeCollection, "pinid_1", bson.D{{Key: "pinid", Value: 1}}, true)
	mongo_util.CreateIndexIfNotExists(mongoClient, TweetLikeCollection, "liketopinid_1", bson.D{{Key: "liketopinid", Value: 1}}, false)
	mongo_util.CreateIndexIfNotExists(mongoClient, TweetLikeCollection, "createaddress_1", bson.D{{Key: "createaddress", Value: 1}}, false)
	mongo_util.CreateIndexIfNotExists(mongoClient, TweetLikeCollection, "createmetaid_1", bson.D{{Key: "createmetaid", Value: 1}}, false)
	mongo_util.CreateIndexIfNotExists(mongoClient, TweetLikeCollection, "islike_1", bson.D{{Key: "islike", Value: 1}}, false)
	mongo_util.CreateIndexIfNotExists(mongoClient, TweetLikeCollection, "timestamp_1", bson.D{{Key: "timestamp", Value: 1}}, false)
	//comment
	mongo_util.CreateIndexIfNotExists(mongoClient, TweetCommentCollection, "pinid_1", bson.D{{Key: "pinid", Value: 1}}, true)
	mongo_util.CreateIndexIfNotExists(mongoClient, TweetCommentCollection, "commentpinid_1", bson.D{{Key: "commentpinid", Value: 1}}, false)
	mongo_util.CreateIndexIfNotExists(mongoClient, TweetCommentCollection, "createaddress_1", bson.D{{Key: "createaddress", Value: 1}}, false)
	mongo_util.CreateIndexIfNotExists(mongoClient, TweetCommentCollection, "createmetaid_1", bson.D{{Key: "createmetaid", Value: 1}}, false)
	mongo_util.CreateIndexIfNotExists(mongoClient, TweetCommentCollection, "islike_1", bson.D{{Key: "islike", Value: 1}}, false)
	mongo_util.CreateIndexIfNotExists(mongoClient, TweetCommentCollection, "timestamp_1", bson.D{{Key: "timestamp", Value: 1}}, false)
	//hostData
	mongo_util.CreateIndexIfNotExists(mongoClient, HostDataCollection, "host_height_1", bson.D{{Key: "host", Value: 1}, {Key: "blockHeight", Value: 1}}, true)
	mongo_util.CreateIndexIfNotExists(mongoClient, HostDataCollection, "host_1", bson.D{{Key: "host", Value: 1}}, false)
	mongo_util.CreateIndexIfNotExists(mongoClient, HostDataCollection, "height_1", bson.D{{Key: "blockHeight", Value: 1}}, false)
	//MetasoTickCollection
	mongo_util.CreateIndexWithFilterIfNotExists(mongoClient, MetasoTickCollection, "idcoin_address_1", bson.D{{Key: "address", Value: 1}, {Key: "idcoin", Value: 1}}, true, bson.D{{Key: "idcoin", Value: 1}})
	mongo_util.CreateIndexIfNotExists(mongoClient, MetasoTickCollection, "address_1", bson.D{{Key: "address", Value: 1}}, false)
	mongo_util.CreateIndexIfNotExists(mongoClient, MetasoTickCollection, "idconin_1", bson.D{{Key: "idconin", Value: 1}}, false)
	//MetaSoMempoolCollection
	mongo_util.CreateIndexIfNotExists(mongoClient, MetaSoMempoolCollection, "pinid_1", bson.D{{Key: "pinid", Value: 1}}, true)
	mongo_util.CreateIndexIfNotExists(mongoClient, MetaSoMempoolCollection, "target_1", bson.D{{Key: "target", Value: 1}}, false)
	//MetaSoPEVData
	mongo_util.DeleteIndex(mongoClient, MetaSoPEVData, "frompinid_1")
	mongo_util.DeleteIndex(mongoClient, MetaSoPEVData, "frompinid_metablockheight_1")
	mongo_util.CreateIndexIfNotExists(mongoClient, MetaSoPEVData, "frompinid_topinid_metablockheight_1", bson.D{{Key: "frompinid", Value: 1}, {Key: "topinid", Value: 1}, {Key: "metablockheight", Value: 1}}, true)
	mongo_util.CreateIndexIfNotExists(mongoClient, MetaSoPEVData, "host_1", bson.D{{Key: "host", Value: 1}}, false)
	mongo_util.CreateIndexIfNotExists(mongoClient, MetaSoPEVData, "host_metablockheight_1", bson.D{{Key: "host", Value: 1}, {Key: "metablockheight", Value: 1}}, false)
	mongo_util.CreateIndexIfNotExists(mongoClient, MetaSoPEVData, "address_1", bson.D{{Key: "address", Value: 1}}, false)
	mongo_util.CreateIndexIfNotExists(mongoClient, MetaSoPEVData, "address_metablockheight_1", bson.D{{Key: "address", Value: 1}, {Key: "metablockheight", Value: 1}}, false)
	mongo_util.CreateIndexIfNotExists(mongoClient, MetaSoPEVData, "address_host_1", bson.D{{Key: "address", Value: 1}, {Key: "host", Value: 1}}, false)
	mongo_util.CreateIndexIfNotExists(mongoClient, MetaSoPEVData, "host_address_metablockheight_1", bson.D{{Key: "host", Value: 1}, {Key: "address", Value: 1}, {Key: "metablockheight", Value: 1}}, false)
	//MetaSoMDVData
	mongo_util.CreateIndexIfNotExists(mongoClient, MetaSoMDVData, "metaid_1", bson.D{{Key: "metaid", Value: 1}}, true)
	mongo_util.CreateIndexIfNotExists(mongoClient, MetaSoMDVData, "address_1", bson.D{{Key: "address", Value: 1}}, true)
	//MetaSoNDVData
	mongo_util.CreateIndexIfNotExists(mongoClient, MetaSoNDVData, "host_1", bson.D{{Key: "host", Value: 1}}, true)
	//MetaSoMDVBlockData
	mongo_util.CreateIndexIfNotExists(mongoClient, MetaSoMDVBlockData, "address_block_1", bson.D{{Key: "address", Value: 1}, {Key: "block", Value: 1}}, true)
	mongo_util.CreateIndexIfNotExists(mongoClient, MetaSoMDVBlockData, "metaid_block_1", bson.D{{Key: "metaid", Value: 1}, {Key: "block", Value: 1}}, true)
	//MetaSoNDVBlockData
	mongo_util.CreateIndexIfNotExists(mongoClient, MetaSoNDVBlockData, "host_block_1", bson.D{{Key: "host", Value: 1}, {Key: "block", Value: 1}}, true)
	mongo_util.CreateIndexIfNotExists(mongoClient, MetaSoNDVBlockData, "host_1", bson.D{{Key: "host", Value: 1}}, false)
	//MetaSoBlockInfoData
	mongo_util.CreateIndexIfNotExists(mongoClient, MetaSoBlockInfoData, "block_1", bson.D{{Key: "block", Value: 1}}, true)
	//MetaSoHostAddressData
	mongo_util.CreateIndexIfNotExists(mongoClient, MetaSoHostAddressData, "host_address_block_1", bson.D{{Key: "host", Value: 1}, {Key: "address", Value: 1}, {Key: "block", Value: 1}}, true)
	//MetaSoDonateData
	mongo_util.CreateIndexIfNotExists(mongoClient, MetaSoDonateData, "pinid_1", bson.D{{Key: "pinid", Value: 1}}, true)
	mongo_util.CreateIndexIfNotExists(mongoClient, MetaSoDonateData, "topin_1", bson.D{{Key: "topin", Value: 1}}, false)
	mongo_util.CreateIndexIfNotExists(mongoClient, MetaSoDonateData, "createaddress_1", bson.D{{Key: "createaddress", Value: 1}}, false)
	mongo_util.CreateIndexIfNotExists(mongoClient, MetaSoDonateData, "toaddress_1", bson.D{{Key: "toaddress", Value: 1}}, false)
	//BlockedSettingData
	mongo_util.CreateIndexIfNotExists(mongoClient, BlockedSettingData, "blockedtype_1", bson.D{{Key: "blockedtype", Value: 1}}, false)
	mongo_util.CreateIndexIfNotExists(mongoClient, BlockedSettingData, "blockedtype_blockedcontent_1", bson.D{{Key: "blockedtype", Value: 1}, {Key: "blockedcontent", Value: 1}}, true)
	//RecommendedAuthors
	mongo_util.CreateIndexIfNotExists(mongoClient, RecommendedAuthors, "author_id_1", bson.D{{Key: "author_id", Value: 1}}, true)
}
func createBuzzView() {
	views, err := mongoClient.ListCollectionNames(context.Background(), bson.M{"name": BuzzView})
	if err != nil {
		return
	}
	if len(views) == 0 {
		mongoClient.CreateView(
			context.Background(),
			BuzzView,
			TweetCollection,
			bson.A{
				bson.D{{Key: "$unionWith", Value: bson.D{
					{Key: "coll", Value: mongodb.MempoolPinsCollection},
					{Key: "pipeline", Value: mongo.Pipeline{
						{{Key: "$match", Value: DataFilter}},
					}},
				}}},
			},
		)
	}
}
