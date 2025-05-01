package mongodb

import (
	"context"
	"manindexer/pin"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func FixNullMetaIdPinId() (err error) {
	filter := bson.M{"pinid": nil}
	find, err := mongoClient.Collection(MetaIdInfoCollection).Find(context.TODO(), filter)
	if err != nil {
		return
	}
	var result []*pin.MetaIdDataValue
	err = find.All(context.TODO(), &result)
	for _, item := range result {
		updateMetaidPinId(item)
		time.Sleep(time.Millisecond * 100)
	}
	return
}
func updateMetaidPinId(item *pin.MetaIdDataValue) (err error) {
	filter := bson.M{"createaddress": item.Address}
	findOptions := options.FindOne()
	findOptions.SetSort(bson.D{{Key: "_id", Value: 1}})
	var pinNode pin.PinInscription
	err = mongoClient.Collection(PinsCollection).FindOne(context.TODO(), filter, findOptions).Decode(&pinNode)
	if err != nil {
		return
	}
	filter2 := bson.M{"address": item.Address}
	_, err = mongoClient.Collection(MetaIdInfoCollection).UpdateOne(context.TODO(), filter2, bson.M{"$set": bson.M{"pinid": pinNode.Id}})
	return
}

func GetPin(pinId string) (pinNode pin.PinInscription, err error) {
	filter := bson.M{"id": pinId}
	findOptions := options.FindOne()
	err = mongoClient.Collection(PinsCollection).FindOne(context.TODO(), filter, findOptions).Decode(&pinNode)
	return
}
