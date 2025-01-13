package metaso

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"manindexer/adapter/bitcoin"
	"manindexer/adapter/microvisionchain"
	"manindexer/common"
	"manindexer/database/mongodb"
	"manindexer/man"
	"net/http"
	"strconv"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func (metaso *MetaSo) syncPEV() {
	if common.Config.Statistics.MetaChainHost == "" || common.Config.Statistics.AllowHost == nil || common.Config.Statistics.AllowProtocols == nil {
		return
	}
	metaBlock, _ := metaso.getLastMetaBlock(1)
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
	blockInfoData := &MetaSoBlockInfo{Block: metaBlock.MetablockHeight, MetaBlock: *metaBlock}
	for _, item := range totalPevList {
		pev := item.(PEVData)
		hostMap[pev.Host] = struct{}{}
		addressMap[pev.Address] = struct{}{}
		blockInfoData.DataValue = blockInfoData.DataValue.Add(pev.IncrementalValue)
		blockInfoData.PinNumber += 1
		if pev.Host != "metabitcoin.unknown" {
			blockInfoData.PinNumberHasHost += 1
		}
	}
	blockInfoData.AddressNumber = int64(len(addressMap))
	blockInfoData.HostNumber = int64(len(hostMap))
	blockInfoData.HistoryValue, _ = getBlockHistoryValue(metaBlock.MetablockHeight, "", "")
	mongoClient.Collection(MetaSoBlockInfoData).UpdateOne(context.TODO(), bson.M{"block": metaBlock.MetablockHeight}, bson.M{"$set": blockInfoData}, options.Update().SetUpsert(true))

	go UpdateBlockValue(metaBlock.MetablockHeight, totalPevList, metaBlock.Timestamp)
	go UpdateDataValue(&hostMap, &addressMap)

}

func (metaso *MetaSo) syncPendingPEV() {
	if common.Config.Statistics.MetaChainHost == "" || common.Config.Statistics.AllowHost == nil || common.Config.Statistics.AllowProtocols == nil {
		return
	}
	lastMetaBlock, _ := metaso.getLastMetaBlock(0)
	if lastMetaBlock == nil {
		return
	}
	if lastMetaBlock.Header == "" {
		return
	}
	log.Println("last metaBlock:", lastMetaBlock.MetablockHeight)
	mongoClient.Collection(MetaSoPEVData).DeleteMany(context.TODO(), bson.M{"metablockheight": -1})
	btc := bitcoin.BitcoinChain{}
	mvc := microvisionchain.MicroVisionChain{}
	btcLastBlockHeight := btc.GetBestHeight()
	btcBeginBlockHeight := btcLastBlockHeight
	mvcLastBlockHeight := int64(0)
	mvcBeginBlockHeight := int64(0)
	if man.ChainAdapter["mvc"] != nil {
		mvcLastBlockHeight = mvc.GetBestHeight()
		mvcBeginBlockHeight = mvcLastBlockHeight
	}

	for _, c := range lastMetaBlock.Chains {
		if c.Chain == "Bitcoin" {
			btcBeginBlockHeight, _ = strconv.ParseInt(c.PreEndBlock, 10, 64)
			btcBeginBlockHeight += 1
		}
		if c.Chain == "MVC" {
			mvcBeginBlockHeight, _ = strconv.ParseInt(c.PreEndBlock, 10, 64)
			mvcBeginBlockHeight += 1
		}
	}
	pendingBlock := &MetaBlockData{
		Header:          "",
		PreHeader:       lastMetaBlock.Header,
		MetablockHeight: -1,
		Chains: []MetaBlockChainData{
			{
				Chain:      "Bitcoin",
				StartBlock: strconv.FormatInt(btcBeginBlockHeight, 10),
				EndBlock:   strconv.FormatInt(btcLastBlockHeight, 10),
			},
			{
				Chain:      "MVC",
				StartBlock: strconv.FormatInt(mvcBeginBlockHeight, 10),
				EndBlock:   strconv.FormatInt(mvcLastBlockHeight, 10),
			},
		},
	}

	var totalPevList []interface{}
	for _, chain := range pendingBlock.Chains {
		pevList, _ := CountBlockPEV(pendingBlock.MetablockHeight, &chain)
		totalPevList = append(totalPevList, pevList...)
	}
	hostMap := make(map[string]struct{})
	addressMap := make(map[string]struct{})
	blockInfoData := &MetaSoBlockInfo{Block: pendingBlock.MetablockHeight, MetaBlock: *pendingBlock}
	for _, item := range totalPevList {
		pev := item.(PEVData)
		hostMap[pev.Host] = struct{}{}
		addressMap[pev.Address] = struct{}{}
		blockInfoData.DataValue = blockInfoData.DataValue.Add(pev.IncrementalValue)
		blockInfoData.PinNumber += 1
		if pev.Host != "metabitcoin.unknown" {
			blockInfoData.PinNumberHasHost += 1
		}
	}
	blockInfoData.AddressNumber = int64(len(addressMap))
	blockInfoData.HostNumber = int64(len(hostMap))
	//blockInfoData.HistoryValue, _ = getBlockHistoryValue(metaBlock.MetablockHeight, "", "")
	mongoClient.Collection(MetaSoBlockInfoData).UpdateOne(context.TODO(), bson.M{"block": pendingBlock.MetablockHeight}, bson.M{"$set": blockInfoData}, options.Update().SetUpsert(true))

	go UpdateBlockValue(pendingBlock.MetablockHeight, totalPevList, pendingBlock.Timestamp)
	//go UpdateDataValue(&hostMap, &addressMap)
}

func (metaso *MetaSo) getLastMetaBlock(addNum int64) (metaBlock *MetaBlockData, err error) {
	localHeight, err := mongodb.GetSyncLastNumber("metablock")
	if err != nil {
		return
	}
	metaBlock = getMetaBlock(localHeight + addNum)
	//fmt.Println("metaBlock:", metaBlock)
	return
}

type metaBlockRes struct {
	Code     int           `json:"code"`
	Data     MetaBlockData `json:"data"`
	Messsage string        `json:"messsage"`
}
type lastMetaBlockRes struct {
	Code     int               `json:"code"`
	Data     LastMetaBlockData `json:"data"`
	Messsage string            `json:"messsage"`
}

func getMetaBlock(height int64) (metaBlock *MetaBlockData) {
	url := fmt.Sprintf("%s/api/block/info?number=%d", common.Config.Statistics.MetaChainHost, height)
	resp, err := http.Get(url)
	if err != nil {
		//fmt.Println("Error making GET request:", err)
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
func getLastMetaBlock() (info *LastMetaBlockData) {
	url := fmt.Sprintf("%s/api/block/latest", common.Config.Statistics.MetaChainHost)
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
	var data lastMetaBlockRes
	err = json.Unmarshal(body, &data)
	if err != nil {
		fmt.Println("Error unmarshalling JSON:", err)
		return
	}
	info = &data.Data
	return
}
