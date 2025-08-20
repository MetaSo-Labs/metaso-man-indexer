package man

import (
	"context"
	"log"
	"manindexer/common"
	"manindexer/database/mongodb"
	"manindexer/pin"
	"regexp"
	"strings"
	"time"

	"github.com/bytedance/sonic"
	"github.com/cockroachdb/pebble"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var notifcationPath = map[string]bool{
	"/follow":                 true,
	"/protocols/simpledonate": true,
	"/protocols/paylike":      true,
	"/protocols/paycomment":   true,
	"/protocols/simplebuzz":   true,
}

func handNotifcation(pinNode *pin.PinInscription) {
	if !common.ModuleExist("metaso_notifcation") {
		return
	}
	if _, ok := common.NotifcationBlackedHost[pinNode.Host]; ok {
		return
	}
	if _, ok := notifcationPath[pinNode.Path]; !ok {
		return
	}
	toPINList := getNotifcationToAddress(pinNode)
	if len(toPINList) == 0 {
		return
	}
	for _, toPIN := range toPINList {
		notifcationType := pinNode.Path
		if toPIN.Path == "Mention" {
			notifcationType = "Mention"
		}
		notifcationData := pin.NotifcationData{
			NotifcationId:   time.Now().UnixMilli(),
			NotifcationType: notifcationType,
			FromPinId:       pinNode.Id,
			FromAddress:     pinNode.Address,
			FromPinHost:     pinNode.Host,
			FromPinChain:    pinNode.ChainName,
			NotifcationPin:  toPIN.Id,
			NotifcationTime: time.Now().Unix(),
			NotifcationHost: toPIN.Host,
		}
		// Save the notification data to DB
		content, err := sonic.Marshal(notifcationData)
		if err != nil {
			return
		}
		if toPIN.Path == "Mention" {
			log.Printf("==> handNotifcation to %s, content: %s", toPIN.Address, content)
		}
		PebbleStore.Database.SetNotifcation(toPIN.Address, content)
		PebbleStore.Database.CleanUpNotifcation(toPIN.Address)
	}
}

func getNotifcationToAddress(pinNode *pin.PinInscription) (toPIN []pin.PinInscription) {
	switch pinNode.Path {
	case "/follow":
		toPIN, _ = getFollowPin(pinNode)
	case "/protocols/simpledonate":
		toPIN, _ = getDonatePin(pinNode)
	case "/protocols/paylike":
		toPIN, _ = getPayLikePin(pinNode)
	case "/protocols/paycomment":
		toPIN, _ = getPaycommentPin(pinNode)
	case "/protocols/simplebuzz":
		toPIN, _ = getRepostPin(pinNode)
		toPIN2, _ := getAtIdCoinPin(pinNode)
		if len(toPIN2) > 0 {
			toPIN = append(toPIN, toPIN2...)
		}
	}
	return
}
func getPINbyId(pinId string) (pinNode pin.PinInscription, err error) {
	pinNode, err = PebbleStore.Database.GetPinInscriptionByKey(pinId)
	switch err {
	case nil:
		return
	case pebble.ErrNotFound:
		pinNode, err = PebbleStore.Database.GetMempoolPin(pinId)
	}
	return
}
func getFollowPin(pinNode *pin.PinInscription) (toPIN []pin.PinInscription, err error) {
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
		toPIN = []pin.PinInscription{
			{
				Id:      pinNode.Id,
				Address: info.Address,
			},
		}
	}
	return
}
func getDonatePin(pinNode *pin.PinInscription) (toPIN []pin.PinInscription, err error) {
	var dataMap map[string]interface{}
	err = sonic.Unmarshal(pinNode.ContentBody, &dataMap)
	if err != nil {
		return
	}
	to, _ := getPINbyId(dataMap["toPin"].(string))
	return []pin.PinInscription{to}, nil
}
func getPayLikePin(pinNode *pin.PinInscription) (toPIN []pin.PinInscription, err error) {
	var dataMap map[string]interface{}
	err = sonic.Unmarshal(pinNode.ContentBody, &dataMap)
	if err != nil {
		return
	}
	if _, ok := dataMap["likeTo"]; !ok {
		return
	}
	if _, ok := dataMap["isLike"]; !ok {
		return
	}
	if dataMap["likeTo"].(string) == "" || dataMap["isLike"].(string) != "1" {
		return
	} else {
		toPINItem, err1 := getPINbyId(dataMap["likeTo"].(string))
		if err1 == nil {
			toPIN = []pin.PinInscription{toPINItem}
		}
		return
	}
}
func getPaycommentPin(pinNode *pin.PinInscription) (toPIN []pin.PinInscription, err error) {
	var dataMap map[string]interface{}
	err = sonic.Unmarshal(pinNode.ContentBody, &dataMap)
	if err != nil {
		return
	}
	if _, ok := dataMap["commentTo"]; !ok {
		return
	} else {
		if dataMap["commentTo"] == nil || dataMap["commentTo"].(string) == "" {
			return
		}
		toPINItem, err1 := getPINbyId(dataMap["commentTo"].(string))
		if err1 == nil {
			toPIN = []pin.PinInscription{toPINItem}
		}
		return
	}
}
func getRepostPin(pinNode *pin.PinInscription) (toPIN []pin.PinInscription, err error) {
	var dataMap map[string]interface{}
	err = sonic.Unmarshal(pinNode.ContentBody, &dataMap)
	if err != nil {
		return
	}
	if _, ok := dataMap["quotePin"]; !ok {
		return
	} else {
		if dataMap["quotePin"] == nil || dataMap["quotePin"].(string) == "" {
			return
		}
		toPINItem, err1 := getPINbyId(dataMap["quotePin"].(string))
		if err1 == nil {
			toPIN = []pin.PinInscription{toPINItem}
		}
		return
	}
}
func getAtIdCoinPin(pinNode *pin.PinInscription) (toPIN []pin.PinInscription, err error) {
	content := string(pinNode.ContentBody)
	list := ExtractAtList(content)
	if len(list) <= 0 {
		return
	}
	log.Println("==>getAtIdCoinPin list:", list)
	for _, atId := range list {
		key := strings.ToLower(atId)
		if address, ok := common.IDCOINS[key]; ok {
			toPINItem := pin.PinInscription{
				Id:        pinNode.Id,
				Address:   address,
				Host:      pinNode.Host,
				ChainName: pinNode.ChainName,
				Path:      "Mention",
			}
			toPIN = append(toPIN, toPINItem)
			log.Println("==>Mention to :", address)
		}
	}
	return
}
func ExtractAtList(content string) []string {
	re := regexp.MustCompile(`@(\S+?)\s`)
	matches := re.FindAllStringSubmatch(content, -1)
	var atList []string
	for _, m := range matches {
		if len(m) > 1 {
			atList = append(atList, m[1])
		}
	}
	return atList
}
