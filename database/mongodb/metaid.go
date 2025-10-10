package mongodb

import (
	"context"
	"fmt"
	"manindexer/common"
	"manindexer/pin"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func (mg *Mongodb) GetMaxMetaIdNumber() (number int64) {
	findOp := options.FindOne()
	findOp.SetSort(bson.D{{Key: "number", Value: -1}})
	var info pin.MetaIdInfo
	err := mongoClient.Collection(MetaIdInfoCollection).FindOne(context.TODO(), bson.D{}, findOp).Decode(&info)
	if err != nil && err == mongo.ErrNoDocuments {
		err = nil
		number = 1
		return
	}
	number = info.Number + 1
	return
}

func (mg *Mongodb) GetMetaIdInfo(address string, mempool bool, metaid string) (info *pin.MetaIdInfo, unconfirmed string, err error) {
	filter := bson.D{{Key: "address", Value: address}}
	if metaid != "" {
		filter = bson.D{{Key: "metaid", Value: metaid}}
	}
	var mempoolInfo pin.MetaIdInfo
	if mempool {
		if metaid != "" {
			mempoolInfo, _ = findMetaIdInfoInMempool("metaid", metaid)
		}
		if address != "" {
			mempoolInfo, _ = findMetaIdInfoInMempool("address", address)
		}
	}
	var unconfirmedList []string
	err = mongoClient.Collection(MetaIdInfoCollection).FindOne(context.TODO(), filter).Decode(&info)
	if err == mongo.ErrNoDocuments {
		err = nil
		if mempoolInfo == (pin.MetaIdInfo{}) {
			return
		}
		info = &mempoolInfo
	}
	if mempool && mempoolInfo != (pin.MetaIdInfo{}) {
		if mempoolInfo.Number == -1 {
			unconfirmedList = append(unconfirmedList, "number")
		}
		if mempoolInfo.Avatar != "" && mempoolInfo.AvatarId != "" {
			info.Avatar = "/content/" + mempoolInfo.AvatarId
			unconfirmedList = append(unconfirmedList, "avatar")
		}
		if mempoolInfo.Name != "" {
			info.Name = mempoolInfo.Name
			unconfirmedList = append(unconfirmedList, "name")
		}
		if mempoolInfo.Bio != "" {
			info.Bio = mempoolInfo.Bio
			unconfirmedList = append(unconfirmedList, "bio")
		}
		if mempoolInfo.Background != "" {
			info.Background = mempoolInfo.Background
			unconfirmedList = append(unconfirmedList, "background")
		}
		if mempoolInfo.ChatPubKey != "" {
			info.ChatPubKey = mempoolInfo.ChatPubKey
			unconfirmedList = append(unconfirmedList, "chatpubkey")
		}
	}
	if len(unconfirmedList) > 0 {
		unconfirmed = strings.Join(unconfirmedList, ",")
	}
	if info.AvatarId != "" {
		info.Avatar = "/content/" + info.AvatarId
	}
	info.MetaId = common.GetMetaIdByAddress(info.Address)
	return
}
func findMetaIdInfoInMempool(key string, value string) (info pin.MetaIdInfo, err error) {
	result, err := mongoClient.Collection(MempoolPinsCollection).Find(context.TODO(), bson.M{key: value})
	if err != nil {
		return
	}
	var pins []pin.PinInscription
	err = result.All(context.TODO(), &pins)
	if err != nil {
		return
	}
	for _, pin := range pins {
		if pin.OriginalPath == "/info/name" {
			info.Name = string(pin.ContentBody)
		} else if pin.OriginalPath == "/info/avatar" {
			info.Avatar = fmt.Sprintf("/content/%s", pin.Id)
		} else if pin.OriginalPath == "/info/nft-avatar" {
			info.Avatar = fmt.Sprintf("/content/%s", pin.Id)
		} else if pin.OriginalPath == "/info/bid" {
			info.Bio = string(pin.ContentBody)
		} else if pin.Path == "/info/background" {
			info.Background = fmt.Sprintf("/content/%s", pin.Id)
		}
	}
	return
}
func (mg *Mongodb) BatchUpsertMetaIdInfo(infoList map[string]*pin.MetaIdInfo) (err error) {
	//bT := time.Now()
	var models []mongo.WriteModel
	for _, info := range infoList {
		filter := bson.D{{Key: "address", Value: info.Address}}
		var updateInfo bson.D
		/*
			update := bson.D{{Key: "$set", Value: bson.D{
				{Key: "mumber", Value: info.Number},
				{Key: "roottxid", Value: info.RootTxId},
				{Key: "name", Value: info.Name},
				{Key: "address", Value: info.Address},
				{Key: "avatar", Value: info.Avatar},
				{Key: "bio", Value: info.Bio},
				{Key: "soulbondtoken", Value: info.SoulbondToken},
			}},
			}
		*/
		if info.Number > 0 {
			updateInfo = append(updateInfo, bson.E{Key: "number", Value: info.Number})
		}
		//if info.MetaId != "" {
		updateInfo = append(updateInfo, bson.E{Key: "metaid", Value: common.GetMetaIdByAddress(info.Address)})
		//}
		if info.Name != "" {
			updateInfo = append(updateInfo, bson.E{Key: "name", Value: info.Name})
		}
		if info.NameId != "" {
			updateInfo = append(updateInfo, bson.E{Key: "nameid", Value: info.NameId})
		}
		if info.Address != "" {
			updateInfo = append(updateInfo, bson.E{Key: "address", Value: info.Address})
		}
		if info.ChainName != "" {
			updateInfo = append(updateInfo, bson.E{Key: "chainname", Value: info.ChainName})
		}
		if len(info.Avatar) > 0 {
			updateInfo = append(updateInfo, bson.E{Key: "avatar", Value: info.Avatar})
		}
		if len(info.AvatarId) > 0 {
			updateInfo = append(updateInfo, bson.E{Key: "avatarid", Value: info.AvatarId})
		}
		if len(info.NftAvatar) > 0 {
			updateInfo = append(updateInfo, bson.E{Key: "nftavatar", Value: info.NftAvatar})
		}
		if len(info.NftAvatarId) > 0 {
			updateInfo = append(updateInfo, bson.E{Key: "nftavatarid", Value: info.NftAvatarId})
		}
		if len(info.Bio) > 0 {
			updateInfo = append(updateInfo, bson.E{Key: "bio", Value: info.Bio})
		}
		if len(info.BioId) > 0 {
			updateInfo = append(updateInfo, bson.E{Key: "bioid", Value: info.BioId})
		}
		if len(info.SoulbondToken) > 0 {
			updateInfo = append(updateInfo, bson.E{Key: "soulbondtoken", Value: info.SoulbondToken})
		}
		if info.Background != "" {
			updateInfo = append(updateInfo, bson.E{Key: "background", Value: info.Background})
		}
		if info.ChatPubKey != "" {
			updateInfo = append(updateInfo, bson.E{Key: "chatpubkey", Value: info.ChatPubKey})
		}
		updateInfo = append(updateInfo, bson.E{Key: "lastupdate", Value: time.Now().Unix()})
		update := bson.D{{Key: "$set", Value: updateInfo}}
		m := mongo.NewUpdateOneModel()
		m.SetFilter(filter).SetUpdate(update).SetUpsert(true)
		models = append(models, m)
	}
	bulkWriteOptions := options.BulkWrite().SetOrdered(false)
	_, err = mongoClient.Collection(MetaIdInfoCollection).BulkWrite(context.Background(), models, bulkWriteOptions)
	//eT := time.Since(bT)
	//fmt.Println("BatchUpsertMetaIdInfo time: ", eT)
	return
}
func BatchGetMetaIdInfo(lastupdate int64, limit int) (infoList map[string]*pin.MetaIdInfo, err error) {
	filter := bson.M{"lastupdate": bson.M{"$gt": lastupdate}}
	opts := options.Find().SetSort(bson.D{{Key: "lastupdate", Value: -1}}).SetLimit(int64(limit))
	result, err := mongoClient.Collection(MetaIdInfoCollection).Find(context.TODO(), filter, opts)
	if err != nil {
		return
	}
	var pins []pin.MetaIdInfo
	err = result.All(context.TODO(), &pins)
	if err != nil {
		return
	}
	infoList = make(map[string]*pin.MetaIdInfo)
	for _, pin := range pins {
		infoList[pin.Address] = &pin
	}
	return
}
func UpdateAllMetaId() error {
	ctx := context.TODO()
	collection := mongoClient.Collection(MetaIdInfoCollection)

	// 批量处理，防止内存溢出
	batchSize := int64(1000)
	var lastID primitive.ObjectID

	for {
		// 分批查询
		filter := bson.M{}
		if !lastID.IsZero() {
			filter["_id"] = bson.M{"$gt": lastID}
		}
		opts := options.Find().SetSort(bson.D{{Key: "_id", Value: 1}}).SetLimit(batchSize)
		cursor, err := collection.Find(ctx, filter, opts)
		if err != nil {
			return err
		}
		var docs []struct {
			ID      primitive.ObjectID `bson:"_id"`
			Address string             `bson:"address"`
		}
		if err := cursor.All(ctx, &docs); err != nil {
			return err
		}
		if len(docs) == 0 {
			break
		}

		var models []mongo.WriteModel
		for _, doc := range docs {
			newMetaId := common.GetMetaIdByAddress(doc.Address)
			update := bson.M{"$set": bson.M{"metaid": newMetaId}}
			model := mongo.NewUpdateOneModel().
				SetFilter(bson.M{"_id": doc.ID}).
				SetUpdate(update)
			models = append(models, model)
			lastID = doc.ID
		}
		if len(models) > 0 {
			_, err := collection.BulkWrite(ctx, models)
			if err != nil {
				return err
			}
		}
		if int64(len(docs)) < batchSize {
			break
		}
	}
	return nil
}
func FetchMetaIdInfoBatch(lastID string, batchSize int64) (pins []*pin.MetaIdInfo, nextLastID string, err error) {
	// 构建过滤条件
	filter := bson.M{}
	if lastID != "" {
		objectID, err := primitive.ObjectIDFromHex(lastID)
		if err != nil {
			return nil, "", fmt.Errorf("invalid lastID: %v", err)
		}
		filter["_id"] = bson.M{"$gt": objectID}
	}

	// 设置查询选项
	opts := options.Find().SetSort(bson.D{{Key: "_id", Value: 1}}).SetLimit(batchSize)

	// 查询数据
	cursor, err := mongoClient.Collection(MetaIdInfoCollection).Find(context.TODO(), filter, opts)
	if err != nil {
		return nil, "", err
	}

	// 手动解析结果
	for cursor.Next(context.TODO()) {
		var raw bson.M
		if err := cursor.Decode(&raw); err != nil {
			return nil, "", err
		}

		// 提取 _id
		id := raw["_id"].(primitive.ObjectID)

		// 将其他字段映射到 MetaIdInfo
		var pin pin.MetaIdInfo
		bsonBytes, _ := bson.Marshal(raw)
		bson.Unmarshal(bsonBytes, &pin)

		pins = append(pins, &pin)
		nextLastID = id.Hex() // 更新 nextLastID
	}

	return pins, nextLastID, nil
}
func addPDV(pins []interface{}) error {
	var models []mongo.WriteModel
	for _, p := range pins {
		pinNode := p.(*pin.PinInscription)
		filter := bson.D{{Key: "metaid", Value: pinNode.MetaId}}
		updateInfo := bson.M{"$inc": bson.M{"pdv": pinNode.DataValue}}
		m := mongo.NewUpdateOneModel()
		m.SetFilter(filter).SetUpdate(updateInfo).SetUpsert(true)
		models = append(models, m)
	}
	bulkWriteOptions := options.BulkWrite().SetOrdered(false)
	_, err := mongoClient.Collection(MetaIdInfoCollection).BulkWrite(context.Background(), models, bulkWriteOptions)
	return err
}
func addFDV(pins []interface{}) (err error) {
	for _, p := range pins {
		pinNode := p.(*pin.PinInscription)
		addSingleFDV(pinNode.MetaId, pinNode.DataValue)
	}
	return
}
func addSingleFDV(metaId string, value int) (err error) {
	//get follow
	filter := bson.M{"followmetaid": metaId, "status": true}
	result, err := mongoClient.Collection(FollowCollection).Find(context.TODO(), filter)
	if err != nil {
		return
	}
	var followData []*pin.FollowData //pin.FollowData
	err = result.All(context.TODO(), &followData)
	if err != nil {
		return
	}
	if len(followData) <= 0 {
		return
	}
	var models []mongo.WriteModel
	for _, f := range followData {
		filter := bson.D{{Key: "metaid", Value: f.MetaId}}
		updateInfo := bson.M{"$inc": bson.M{"fdv": value}}
		m := mongo.NewUpdateOneModel()
		m.SetFilter(filter).SetUpdate(updateInfo).SetUpsert(true)
		models = append(models, m)
	}
	bulkWriteOptions := options.BulkWrite().SetOrdered(false)
	_, err = mongoClient.Collection(MetaIdInfoCollection).BulkWrite(context.Background(), models, bulkWriteOptions)
	return err
}
func addFollowFDV(metaId string, follower string, action string) (err error) {
	var info pin.MetaIdInfo
	filter := bson.M{"metaid": follower}
	err = mongoClient.Collection(MetaIdInfoCollection).FindOne(context.TODO(), filter).Decode(&info)
	if err != nil {
		return
	}
	filter = bson.M{"metaid": metaId}
	value := info.Pdv
	if action == "unfollow" {
		value = value * -1
	}
	update := bson.M{"$inc": bson.M{"fdv": value}}
	_, err = mongoClient.Collection(MetaIdInfoCollection).UpdateOne(context.TODO(), filter, update)
	return
}
func (mg *Mongodb) GetMetaIdPageList(page int64, size int64, order string) (pins []*pin.MetaIdInfo, err error) {
	cursor := (page - 1) * size
	if order == "" {
		order = "number"
	}
	opts := options.Find().SetSort(bson.D{{Key: order, Value: -1}}).SetSkip(cursor).SetLimit(size)
	result, err := mongoClient.Collection(MetaIdInfoCollection).Find(context.TODO(), bson.M{}, opts)
	if err != nil {
		return
	}
	err = result.All(context.TODO(), &pins)
	return
}
func (mg *Mongodb) BatchUpsertMetaIdInfoAddition(infoList []*pin.MetaIdInfoAdditional) (err error) {
	var models []mongo.WriteModel
	for _, info := range infoList {
		filter := bson.D{{Key: "metaid", Value: info.MetaId}, {Key: "infokey", Value: info.InfoKey}}
		var updateInfo bson.D
		updateInfo = append(updateInfo, bson.E{Key: "infoValue", Value: info.InfoValue})
		updateInfo = append(updateInfo, bson.E{Key: "pinid", Value: info.PinId})
		update := bson.D{{Key: "$set", Value: updateInfo}}
		m := mongo.NewUpdateOneModel()
		m.SetFilter(filter).SetUpdate(update).SetUpsert(true)
		models = append(models, m)
	}

	bulkWriteOptions := options.BulkWrite().SetOrdered(false)
	_, err = mongoClient.Collection(InfoCollection).BulkWrite(context.Background(), models, bulkWriteOptions)
	return
}
func batchUpdateFollowCount(list map[string]int) (err error) {
	var models []mongo.WriteModel
	for metaid, cnt := range list {
		filter := bson.D{{Key: "metaid", Value: metaid}}
		var updateInfo bson.D
		updateInfo = append(updateInfo, bson.E{Key: "followcount", Value: cnt})

		update := bson.D{{Key: "$inc", Value: updateInfo}}
		m := mongo.NewUpdateOneModel()
		m.SetFilter(filter).SetUpdate(update)
		models = append(models, m)
	}
	bulkWriteOptions := options.BulkWrite().SetOrdered(false)
	_, err = mongoClient.Collection(MetaIdInfoCollection).BulkWrite(context.Background(), models, bulkWriteOptions)

	return
}
func (mg *Mongodb) GetDataValueByMetaIdList(list []string) (result []*pin.MetaIdDataValue, err error) {
	filter := bson.M{"$or": bson.A{bson.M{"address": bson.M{"$in": list}}, bson.M{"metaid": bson.M{"$in": list}}}}
	find, err := mongoClient.Collection(MetaIdInfoCollection).Find(context.TODO(), filter)
	if err != nil {
		return
	}
	err = find.All(context.TODO(), &result)
	return
}
