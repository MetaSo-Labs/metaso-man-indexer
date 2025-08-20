package metaso

import (
	"context"
	"encoding/json"
	"manindexer/database/mongodb"
	"manindexer/pin"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type TweetWithLike struct {
	Tweet
	Like    []string `json:"like"`
	Donate  []string `json:"donate"`
	Blocked bool     `json:"blocked"`
}

func textSearch(lastId string, size int64, key string) (listData []*TweetWithLike, total int64, err error) {
	if len(key) <= 0 {
		return
	}
	var list []*Tweet
	filter := bson.D{{Key: "$text", Value: bson.D{{Key: "$search", Value: key}}}}
	totalFilter := bson.D{{Key: "$text", Value: bson.D{{Key: "$search", Value: key}}}}
	if lastId != "" {
		var objectId primitive.ObjectID
		objectId, err = primitive.ObjectIDFromHex(lastId)
		if err != nil {
			return
		}
		filter = append(filter, bson.E{Key: "_id", Value: bson.D{{Key: "$lt", Value: objectId}}})
	}

	findOptions := options.Find()
	findOptions.SetSort(bson.D{{Key: "timestamp", Value: -1}})
	findOptions.SetLimit(size)
	result, err := mongoClient.Collection(TweetCollection).Find(context.TODO(), filter, findOptions)
	if err != nil {
		return
	}
	err = result.All(context.TODO(), &list)
	if err == mongo.ErrNoDocuments {
		err = nil
	}
	var pinIdList []string
	for _, item := range list {
		item.Content = string(item.ContentBody)
		item.ContentBody = nil
		pinIdList = append(pinIdList, item.Id)
	}

	checkMap := make(map[string]*TweetWithLike, len(list))
	for _, item := range list {
		checkMap[item.Id] = &TweetWithLike{Tweet: *item, Like: []string{}, Donate: []string{}}
	}
	likeMap, err := batchGetPayLike(pinIdList)
	if err == nil {
		for _, item := range list {
			if v, ok := likeMap[item.Id]; ok {
				checkMap[item.Id].Like = v
			}
		}
	}
	donateMap, err := batchGetSimpleDonat(pinIdList)
	if err == nil {
		for _, item := range list {
			if v, ok := donateMap[item.Id]; ok {
				checkMap[item.Id].Donate = v
			}
		}
	}
	for _, item := range list {
		if v, ok := checkMap[item.Id]; ok {
			listData = append(listData, v)
		}
	}
	total, err = mongoClient.Collection(TweetCollection).CountDocuments(context.TODO(), totalFilter)

	return
}
func getNewest(lastId string, size int64, listType string, metaid string, followed string) (listData []*TweetWithLike, total int64, err error) {
	var list []*Tweet
	filter := bson.D{{Key: "blocked", Value: false}}
	totalFilter := bson.D{{Key: "blocked", Value: false}}
	if lastId != "" {
		var objectId primitive.ObjectID
		objectId, err = primitive.ObjectIDFromHex(lastId)
		if err != nil {
			return
		}
		filter = append(filter, bson.E{Key: "_id", Value: bson.D{{Key: "$lt", Value: objectId}}})
	}
	if metaid != "" && followed == "1" {
		followList, err1 := getAddressFollowing(metaid)
		if err1 != nil || len(followList) == 0 {
			err = nil
			return
		}
		totalFilter = append(totalFilter, bson.E{Key: "createmetaid", Value: bson.D{{Key: "$in", Value: followList}}})
		filter = append(filter, bson.E{Key: "createmetaid", Value: bson.D{{Key: "$in", Value: followList}}})
	} else if metaid != "" && followed == "" {
		filter = append(filter, bson.E{Key: "createmetaid", Value: metaid})
		totalFilter = append(totalFilter, bson.E{Key: "createmetaid", Value: metaid})
	}
	if listType == "hot" {
		now := time.Now()
		twentyFourHoursAgo := now.Add(-24 * time.Hour)
		filter = append(filter, bson.E{
			Key: "timestamp",
			Value: bson.D{
				{Key: "$gt", Value: twentyFourHoursAgo.Unix()},
				{Key: "$lt", Value: now.Unix()},
			},
		})
		totalFilter = append(totalFilter, bson.E{
			Key: "timestamp",
			Value: bson.D{
				{Key: "$gt", Value: twentyFourHoursAgo.Unix()},
				{Key: "$lt", Value: now.Unix()},
			},
		})
	}
	findOptions := options.Find()
	findOptions.SetSort(bson.D{{Key: listType, Value: -1}})
	findOptions.SetLimit(size)
	result, err := mongoClient.Collection(BuzzView).Find(context.TODO(), filter, findOptions)
	if err != nil {
		return
	}
	err = result.All(context.TODO(), &list)
	if err == mongo.ErrNoDocuments {
		err = nil
	}
	var pinIdList []string
	for _, item := range list {
		item.Content = string(item.ContentBody)
		item.ContentBody = nil
		pinIdList = append(pinIdList, item.Id)
	}

	mempoolList, err := getBuzzMempoolCount(pinIdList)
	if err == nil {
		for _, item := range list {
			for _, data := range mempoolList {
				if item.Id == data.Target && data.Path == "/protocols/paylike" {
					item.LikeCount += 1
				}
				if item.Id == data.Target && data.Path == "/protocols/paycomment" {
					item.CommentCount += 1
				}
				if item.Id == data.Target && data.Path == "/protocols/simpledonate" {
					item.DonateCount += 1
				}
			}
		}
	}
	checkMap := make(map[string]*TweetWithLike, len(list))
	for _, item := range list {
		checkMap[item.Id] = &TweetWithLike{Tweet: *item, Like: []string{}, Donate: []string{}}
	}
	likeMap, err := batchGetPayLike(pinIdList)
	if err == nil {
		for _, item := range list {
			if v, ok := likeMap[item.Id]; ok {
				checkMap[item.Id].Like = v
			}
		}
	}
	donateMap, err := batchGetSimpleDonat(pinIdList)
	if err == nil {
		for _, item := range list {
			if v, ok := donateMap[item.Id]; ok {
				checkMap[item.Id].Donate = v
			}
		}
	}
	for _, item := range list {
		if v, ok := checkMap[item.Id]; ok {
			listData = append(listData, v)
		}
	}
	total, err = mongoClient.Collection(BuzzView).CountDocuments(context.TODO(), totalFilter)

	return
}

func getBuzzMempoolCount(pinIdList []string) (mempoolData []MempoolData, err error) {
	filter := bson.D{{Key: "target", Value: bson.D{{Key: "$in", Value: pinIdList}}}}
	resultMempool, err := mongoClient.Collection(MetaSoMempoolCollection).Find(context.TODO(), filter)
	if err != nil {
		return
	}
	resultMempool.All(context.TODO(), &mempoolData)
	return
}

func getAddressFollowing(metaid string) (list []string, err error) {
	filterA := bson.M{"followmetaid": metaid, "status": true}
	result, err := mongoClient.Collection(mongodb.FollowCollection).Find(context.TODO(), filterA)
	if err != nil {
		return
	}
	var followData []*pin.FollowData //pin.FollowData
	err = result.All(context.TODO(), &followData)
	for _, item := range followData {
		list = append(list, item.MetaId)
	}
	return
}
func batchGetPayLike(pinIdList []string) (list map[string][]string, err error) {
	list = make(map[string][]string)
	filter1 := bson.D{{Key: "liketopinid", Value: bson.D{{Key: "$in", Value: pinIdList}}}}
	result, err := mongoClient.Collection(TweetLikeCollection).Find(context.TODO(), filter1)
	var likeList []*TweetLike
	if err == nil {
		result.All(context.TODO(), &likeList)
	}
	for _, like := range likeList {
		if like.IsLike != "1" {
			continue
		}
		list[like.LikeToPinId] = append(list[like.LikeToPinId], like.CreateMetaid)
	}
	//mempool
	filter2 := bson.D{{Key: "target", Value: bson.D{{Key: "$in", Value: pinIdList}}}, {Key: "path", Value: "/protocols/paylike"}}
	resultMempool, err := mongoClient.Collection(MetaSoMempoolCollection).Find(context.TODO(), filter2)
	if err == nil {
		var mempoolData []MempoolData
		resultMempool.All(context.TODO(), &mempoolData)
		for _, data := range mempoolData {
			if data.IsCancel == 1 {
				if v, ok := list[data.Target]; ok {
					list[data.Target] = deleteSlice(v, data.CreateMetaId)
				}
			} else {
				list[data.Target] = append(list[data.Target], data.CreateMetaId)
			}
		}
	}
	return
}
func batchGetSimpleDonat(pinIdList []string) (list map[string][]string, err error) {
	list = make(map[string][]string)
	filter1 := bson.D{{Key: "topin", Value: bson.D{{Key: "$in", Value: pinIdList}}}}
	result, err := mongoClient.Collection(MetaSoDonateData).Find(context.TODO(), filter1)
	var donatList []*MetasoDonate
	if err == nil {
		result.All(context.TODO(), &donatList)
	}
	for _, donat := range donatList {
		list[donat.ToPin] = append(list[donat.ToPin], donat.CreateMetaid)
	}
	//mempool
	filter2 := bson.D{{Key: "target", Value: bson.D{{Key: "$in", Value: pinIdList}}}, {Key: "path", Value: "/protocols/simpledonate"}}
	resultMempool, err := mongoClient.Collection(MetaSoMempoolCollection).Find(context.TODO(), filter2)
	if err == nil {
		var mempoolData []MempoolData
		resultMempool.All(context.TODO(), &mempoolData)
		for _, data := range mempoolData {
			list[data.Target] = append(list[data.Target], data.CreateMetaId)
		}
	}
	return
}
func deleteSlice(s []string, elem string) []string {
	r := s[:0]
	for _, v := range s {
		if v != elem {
			r = append(r, v)
		}
	}
	return r
}

type CommentsList struct {
	PinId         string `json:"pinId"`
	ChainName     string `json:"chainName"`
	CreateAddress string `json:"createAddress"`
	CreateMetaid  string `json:"CreateMetaid"`
	Content       string `json:"content"`
	Timestamp     int64  `json:"timestamp"`
	LikeNum       int64  `json:"likeNum"`
	CommentNum    int64  `json:"commentNum"`
}

func getCommentsList(pinId string) (comments []*CommentsList, err error) {
	var commentsList []*TweetComment
	filter2 := bson.D{{Key: "commentpinid", Value: pinId}}
	result, err := mongoClient.Collection(TweetCommentCollection).Find(context.TODO(), filter2)
	if err == nil {
		result.All(context.TODO(), &commentsList)
	}
	if len(commentsList) > 0 {
		var idList []string
		for _, c := range commentsList {
			idList = append(idList, c.PinId)
		}
		filter := bson.D{{Key: "id", Value: bson.D{{Key: "$in", Value: idList}}}}
		findOptions := options.Find().SetProjection(bson.D{
			{Key: "id", Value: 1},
			{Key: "likecount", Value: 1},
			{Key: "commentcount", Value: 1},
			{Key: "_id", Value: 0}, // 如果不需要返回 MongoDB 的 `_id` 字段
		})
		result, err := mongoClient.Collection(BuzzView).Find(context.TODO(), filter, findOptions)
		if err == nil {
			var tweetList []*Tweet
			result.All(context.TODO(), &tweetList)
			tweetMap := make(map[string]*Tweet, len(tweetList))
			for _, t := range tweetList {
				tweetMap[t.Id] = t
			}

			for _, c := range commentsList {
				if t, ok := tweetMap[c.PinId]; ok {
					comments = append(comments, &CommentsList{
						PinId:         c.PinId,
						ChainName:     c.ChainName,
						CreateAddress: c.CreateAddress,
						CreateMetaid:  c.CreateMetaid,
						Content:       c.Content,
						Timestamp:     c.Timestamp,
						LikeNum:       int64(t.LikeCount),
						CommentNum:    int64(t.CommentCount),
					})
				}
			}
		}
	}
	return
}
func getInfo(pinId string) (tweet *Tweet, comments []*CommentsList, like []*TweetLike, donates []*MetasoDonate, err error) {
	filter := bson.D{{Key: "id", Value: pinId}}
	err = mongoClient.Collection(BuzzView).FindOne(context.TODO(), filter, nil).Decode(&tweet)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			err = nil
		}
		return
	}
	tweet.Content = string(tweet.ContentBody)
	tweet.ContentBody = nil

	comments, _ = getCommentsList(pinId)

	filter3 := bson.D{{Key: "liketopinid", Value: pinId}}
	result2, err := mongoClient.Collection(TweetLikeCollection).Find(context.TODO(), filter3)
	if err == nil {
		result2.All(context.TODO(), &like)
	}

	filter5 := bson.D{{Key: "topin", Value: pinId}}
	result5, err := mongoClient.Collection(MetaSoDonateData).Find(context.TODO(), filter5)
	if err == nil {
		result5.All(context.TODO(), &donates)
	}

	//mempool
	filter4 := bson.D{{Key: "target", Value: pinId}}
	resultMempool, err := mongoClient.Collection(MetaSoMempoolCollection).Find(context.TODO(), filter4)
	if err != nil {
		return
	}
	var mempoolData []MempoolData
	resultMempool.All(context.TODO(), &mempoolData)
	for _, data := range mempoolData {
		if data.Path == "/protocols/paylike" {
			var likeData TweetLike
			err := json.Unmarshal([]byte(data.Content), &likeData)
			if err == nil {
				like = append(like, &likeData)
				tweet.LikeCount += 1
			}
		} else if data.Path == "/protocols/paycomment" {
			var commentData TweetComment
			err := json.Unmarshal([]byte(data.Content), &commentData)
			if err == nil {
				comments = append(comments, &CommentsList{
					PinId:         commentData.PinId,
					ChainName:     commentData.ChainName,
					CreateAddress: commentData.CreateAddress,
					CreateMetaid:  commentData.CreateMetaid,
					Content:       commentData.Content,
					Timestamp:     commentData.Timestamp,
					LikeNum:       0,
					CommentNum:    0,
				})
				tweet.CommentCount += 1
			}
		} else if data.Path == "/protocols/simpledonate" {
			var donatedata MetasoDonate
			err := json.Unmarshal([]byte(data.Content), &donatedata)
			if err == nil {
				donates = append(donates, &donatedata)
				tweet.DonateCount += 1
			}
		}
	}
	return
}
func getBlockInfo(height int64, host string, cursor int64, size int64, orderby string) (list []*HostData, err error) {
	var filter primitive.D
	if height > 0 {
		filter = bson.D{{Key: "blockHeight", Value: height}}
	} else {
		filter = bson.D{{Key: "host", Value: host}}
	}
	if orderby == "" {
		orderby = "txCount"
	}
	findOptions := options.Find()
	findOptions.SetSort(bson.D{{Key: orderby, Value: -1}})
	findOptions.SetSkip(cursor).SetLimit(size)
	result, err := mongoClient.Collection(HostDataCollection).Find(context.TODO(), filter, findOptions)
	if err != nil {
		return
	}
	err = result.All(context.TODO(), &list)
	if err == mongo.ErrNoDocuments {
		err = nil
	}
	return
}
func getBlockNDV(height int64, host string, cursor int64, size int64, orderby string) (list []*MetaSoBlockNDV, err error) {
	var filter primitive.D
	if height > 0 {
		filter = bson.D{{Key: "block", Value: height}}
	}
	if host != "" {
		filter = bson.D{{Key: "host", Value: host}}
	}
	findOptions := options.Find()
	findOptions.SetSort(bson.D{{Key: "datavalue", Value: -1}})
	findOptions.SetSkip(cursor).SetLimit(size)
	result, err := mongoClient.Collection(MetaSoNDVBlockData).Find(context.TODO(), filter, findOptions)
	if err != nil {
		return
	}
	err = result.All(context.TODO(), &list)
	if err == mongo.ErrNoDocuments {
		err = nil
	}
	return
}
func getNdvPageList(host string, pagecursor int64, size int64, orderby string) (list []*MetaSoNDV, err error) {
	// filter := bson.D{}
	// if host != "" {
	// 	filter = bson.D{{Key: "host", Value: host}}
	// }
	// findOptions := options.Find()
	// findOptions.SetSort(bson.D{{Key: "datavalue", Value: -1}})
	// findOptions.SetSkip(cursor).SetLimit(size)
	// result, err := mongoClient.Collection(MetaSoNDVData).Find(context.TODO(), filter, findOptions)
	// if err != nil {
	// 	return
	// }
	// err = result.All(context.TODO(), &list)
	// if err == mongo.ErrNoDocuments {
	// 	err = nil
	// }
	if host == "" {
		return
	}
	pipeline := mongo.Pipeline{
		{{"$match", bson.D{{"host", host}}}},
		{{"$group", bson.D{
			{"_id", nil},
			{"totalValue", bson.D{{"$sum", "$datavalue"}}},
		}}},
	}

	cursor, err := mongoClient.Collection(MetaSoNDVBlockData).Aggregate(context.TODO(), pipeline)
	if err != nil {
		return
	}
	defer cursor.Close(context.TODO())

	var results []bson.M
	if err = cursor.All(context.TODO(), &results); err != nil {
		return
	}
	if len(results) > 0 {
		v, _ := Decimal128ToDecimal(results[0]["totalValue"].(primitive.Decimal128))
		data := &MetaSoNDV{
			Host:      host,
			DataValue: v,
		}
		list = append(list, data)
	}
	return
}
func getMdvPageList(address string, cursor int64, size int64, orderby string) (list []*MetaSoMDV, err error) {
	filter := bson.D{}
	if address != "" {
		filter = bson.D{{Key: "address", Value: address}}
	}
	findOptions := options.Find()
	findOptions.SetSort(bson.D{{Key: "datavalue", Value: -1}})
	findOptions.SetSkip(cursor).SetLimit(size)
	result, err := mongoClient.Collection(MetaSoMDVData).Find(context.TODO(), filter, findOptions)
	if err != nil {
		return
	}
	err = result.All(context.TODO(), &list)
	if err == mongo.ErrNoDocuments {
		err = nil
	}
	return
}
func getBlockMDV(height int64, address string, cursor int64, size int64, orderby string) (list []*MetaSoBlockMDV, err error) {
	var filter primitive.D
	if height > 0 {
		filter = bson.D{{Key: "block", Value: height}}
	}
	if address != "" {
		filter = bson.D{{Key: "address", Value: address}}
	}
	findOptions := options.Find()
	findOptions.SetSort(bson.D{{Key: "datavalue", Value: -1}})
	findOptions.SetSkip(cursor).SetLimit(size)
	result, err := mongoClient.Collection(MetaSoMDVBlockData).Find(context.TODO(), filter, findOptions)
	if err != nil {
		return
	}
	err = result.All(context.TODO(), &list)
	if err == mongo.ErrNoDocuments {
		err = nil
	}
	return
}
func getTickByAddress(address string, tickType string) (list []*Mrc20DeployInfo, err error) {
	filter := bson.D{{Key: "address", Value: address}}
	if tickType == "idcoins" {
		filter = append(filter, bson.E{Key: "idcoin", Value: 1})
	}
	findOptions := options.Find()
	findOptions.SetSort(bson.D{{Key: "tick", Value: 1}})
	result, err := mongoClient.Collection(MetasoTickCollection).Find(context.TODO(), filter, findOptions)
	if err != nil {
		return
	}
	err = result.All(context.TODO(), &list)
	if err == mongo.ErrNoDocuments {
		err = nil
	}
	return
}
func getMempoolFollow(metaid string) (list []*string, err error) {
	filter := bson.D{{Key: "target", Value: metaid}, {Key: "path", Value: "/follow"}}
	resultMempool, err := mongoClient.Collection(MetaSoMempoolCollection).Find(context.TODO(), filter)
	if err != nil {
		return
	}
	var mempoolData []MempoolData
	resultMempool.All(context.TODO(), &mempoolData)
	for _, data := range mempoolData {
		list = append(list, &data.Content)
	}
	return
}

func getBlockedList(blockType string, cursor int64, size int64) (list []*BlockedSetting, total int64, err error) {
	filter := bson.D{{Key: "blockedtype", Value: blockType}}
	findOptions := options.Find()
	findOptions.SetSkip(cursor).SetLimit(size)
	result, err := mongoClient.Collection(BlockedSettingData).Find(context.TODO(), filter, findOptions)
	if err != nil {
		return
	}
	err = result.All(context.TODO(), &list)
	if err == mongo.ErrNoDocuments {
		err = nil
	}
	total, _ = mongoClient.Collection(BlockedSettingData).CountDocuments(context.TODO(), filter)
	return
}

func addBlockedList(blockType string, blockContent string, originalContent string) (err error) {
	_, err = mongoClient.Collection(BlockedSettingData).InsertOne(context.TODO(), BlockedSetting{BlockedType: blockType, BlockedContent: blockContent, Timestamp: time.Now().Unix(), OriginalContent: originalContent})
	return
}
func deleteBlockedList(blockType string, blockContent string) (err error) {
	filter := bson.D{{Key: "blockedtype", Value: blockType}, {Key: "blockedcontent", Value: blockContent}}
	_, err = mongoClient.Collection(BlockedSettingData).DeleteOne(context.TODO(), filter)
	return
}
