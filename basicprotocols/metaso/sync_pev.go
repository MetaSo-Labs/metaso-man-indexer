package metaso

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"manindexer/common"
	"manindexer/database/mongodb"
	"net/http"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func (metaso *MetaSo) syncPEV() {
	if common.Config.Statistics.MetaChainHost == "" || common.Config.Statistics.AllowHost == nil || common.Config.Statistics.AllowProtocols == nil {
		return
	}
	metaBlock, _ := metaso.getLastMetaBlock()
	if metaBlock == nil {
		return
	}
	if metaBlock.Header == "" {
		return
	}
	log.Println("count metaBlock:", metaBlock.MetablockHeight)
	mongodb.UpdateSyncLastNumber("metablock", metaBlock.MetablockHeight)
	var totalPevList []interface{}
	for _, chain := range metaBlock.Chains {
		pevList, _ := CountBlockPEV(metaBlock.MetablockHeight, &chain)
		totalPevList = append(totalPevList, pevList...)
	}
	hostMap := make(map[string]struct{})
	addressMap := make(map[string]struct{})
	blockInfoData := &MetaSoBlockInfo{Block: metaBlock.MetablockHeight}
	for _, item := range totalPevList {
		pev := item.(PEVData)
		hostMap[pev.Host] = struct{}{}
		addressMap[pev.Address] = struct{}{}
		blockInfoData.DataValue += pev.IncrementalValue
		blockInfoData.PinNumber += 1
		if pev.Host != "metabitcoin.unknown" {
			blockInfoData.PinNumberHasHost += 1
		}
	}
	blockInfoData.AddressNumber = int64(len(addressMap))
	blockInfoData.HostNumber = int64(len(hostMap))
	blockInfoData.HistoryValue, _ = getBlockHistoryValue(metaBlock.MetablockHeight, "", "")
	mongoClient.Collection(MetaSoBlockInfoData).UpdateOne(context.TODO(), bson.M{"block": metaBlock.MetablockHeight}, bson.M{"$set": blockInfoData}, options.Update().SetUpsert(true))

	go UpdateBlcokValue(metaBlock.MetablockHeight, totalPevList)
	go UpdateDataValue(&hostMap, &addressMap)

}
func (metaso *MetaSo) getLastMetaBlock() (metaBlock *MetaBlockData, err error) {
	localHeight, err := mongodb.GetSyncLastNumber("metablock")
	if err != nil {
		return
	}
	metaBlock = getMetaBlock(localHeight + 1)
	//fmt.Println("metaBlock:", metaBlock)
	return
}

type metaBlockRes struct {
	Code     int           `json:"code"`
	Data     MetaBlockData `json:"data"`
	Messsage string        `json:"messsage"`
}

func getMetaBlock(height int64) (metaBlock *MetaBlockData) {
	url := fmt.Sprintf("%s/api/block/info?number=%d", common.Config.Statistics.MetaChainHost, height)
	resp, err := http.Get(url)
	if err != nil {
		fmt.Println("Error making GET request:", err)
		return
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return
	}
	var data metaBlockRes
	err = json.Unmarshal(body, &data)
	if err != nil {
		fmt.Println("Error unmarshalling JSON:", err)
		return
	}
	metaBlock = &data.Data
	return
}
