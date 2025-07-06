package man

import (
	"context"
	"manindexer/common"
	"manindexer/database/mongodb"
	"manindexer/pin"
	"time"

	"github.com/bytedance/sonic"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var notifcationPath = map[string]bool{
	"/follow":                 true,
	"/protocols/simpledonate": true,
	"/protocols/paylike":      true,
	"/protocols/paycomment":   true,
}

func handNotifcation(pinNode *pin.PinInscription) {
	if !common.ModuleExist("metaso_notifcation") {
		return
	}
	if _, ok := notifcationPath[pinNode.Path]; !ok {
		return
	}
	toPIN := getNotifcationToAddress(pinNode)
	if toPIN.Id == "" {
		return
	}
	notifcationData := pin.NotifcationData{
		NotifcationId:   time.Now().UnixNano(),
		NotifcationType: pinNode.Path,
		FromPinId:       pinNode.Id,
		FromAddress:     pinNode.Address,
		NotifcationPin:  toPIN.Id,
		NotifcationTime: time.Now().Unix(),
	}
	// Save the notification data to DB
	content, err := sonic.Marshal(notifcationData)
	if err != nil {
		return
	}
	PebbleStore.Database.SetNotifcation(toPIN.Address, content)
}

func getNotifcationToAddress(pinNode *pin.PinInscription) (toPIN pin.PinInscription) {
	switch pinNode.Path {
	case "/follow":
		toPIN, _ = getFollowPin(pinNode)
	case "/protocols/simpledonate":
		toPIN, _ = getDonatePin(pinNode)
	case "/protocols/paylike":
		toPIN, _ = getPayLikePin(pinNode)
	case "/protocols/paycomment":
		toPIN, _ = getPaycommentPin(pinNode)
	}
	return
}
func getPINbyId(pinId string) (pinNode pin.PinInscription, err error) {
	pinNode, err = PebbleStore.Database.GetPinInscriptionByKey(pinId)
	switch err {
	case nil:
		return
	case mongo.ErrNoDocuments:
		pinNode, err = PebbleStore.Database.GetMempoolPin(pinId)
	}
	return
}
func getFollowPin(pinNode *pin.PinInscription) (toPIN pin.PinInscription, err error) {
	metaid := string(pinNode.ContentBody)
	filter := bson.M{"metaid": metaid}
	findOptions := options.FindOne()
	findOptions.SetSort(bson.D{{Key: "_id", Value: 1}})
	var info pin.MetaIdInfo
	err = mongodb.Client.Collection(mongodb.MetaIdInfoCollection).FindOne(context.TODO(), filter, findOptions).Decode(&info)
	if err != nil && err == mongo.ErrNoDocuments {
		err = mongodb.Client.Collection(mongodb.MempoolPinsCollection).FindOne(context.TODO(), filter, findOptions).Decode(&toPIN)
		return
	} else {
		toPIN = pin.PinInscription{
			Id:      pinNode.Id,
			Address: info.Address,
		}
	}
	return
}
func getDonatePin(pinNode *pin.PinInscription) (toPIN pin.PinInscription, err error) {
	var dataMap map[string]interface{}
	err = sonic.Unmarshal(pinNode.ContentBody, &dataMap)
	if err != nil {
		return
	}
	return getPINbyId(dataMap["toPin"].(string))
}
func getPayLikePin(pinNode *pin.PinInscription) (toPIN pin.PinInscription, err error) {
	var dataMap map[string]interface{}
	err = sonic.Unmarshal(pinNode.ContentBody, &dataMap)
	if err != nil {
		return
	}
	if dataMap["likeTo"].(string) == "" || dataMap["isLike"].(string) != "1" {
		return
	} else {
		return getPINbyId(dataMap["likeTo"].(string))
	}
}
func getPaycommentPin(pinNode *pin.PinInscription) (toPIN pin.PinInscription, err error) {
	var dataMap map[string]interface{}
	err = sonic.Unmarshal(pinNode.ContentBody, &dataMap)
	if err != nil {
		return
	}
	if dataMap["commentTo"].(string) == "" {
		return
	} else {
		return getPINbyId(dataMap["commentTo"].(string))
	}
}
