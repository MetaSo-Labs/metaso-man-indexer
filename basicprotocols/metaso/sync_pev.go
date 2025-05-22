package metaso

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"manindexer/common"
	"manindexer/database/mongodb"
	"manindexer/man"
	"net/http"
	"strconv"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func (metaso *MetaSo) SyncPEVTest(height int64) {
	metaBlock := getMetaBlock(height)
	var err error
	for _, chain := range metaBlock.Chains {
		endBlock := int64(0)
		maxBlock := int64(0)
		if chain.Chain == "Bitcoin" {
			maxBlock = man.MaxHeight["btc"]
		} else if chain.Chain == "MVC" {
			maxBlock = man.MaxHeight["mvc"]
		}
		if chain.EndBlock == "" {
			continue
		}
		endBlock, err = strconv.ParseInt(chain.EndBlock, 10, 64)
		if err != nil || endBlock <= 0 {
			log.Println(">>", chain.Chain, err, endBlock, maxBlock)
			return
		}
	}

	var totalPevList []interface{}
	for _, chain := range metaBlock.Chains {
		pevList, err := CountBlockPEV(metaBlock.MetablockHeight, &chain)
		fmt.Println(err)
		if len(pevList) > 0 {
			totalPevList = append(totalPevList, pevList...)
		}
	}
	fmt.Println("len:", len(totalPevList))
	hostMap := make(map[string]struct{})
	addressMap := make(map[string]struct{})
	blockInfoData := &MetaSoBlockInfo{Block: metaBlock.MetablockHeight, MetaBlock: *metaBlock}
	for _, item := range totalPevList {
		pev := item.(PEVData)
		hostMap[pev.Host] = struct{}{}
		addressMap[pev.Address] = struct{}{}
		blockInfoData.DataValue = blockInfoData.DataValue.Add(pev.IncrementalValue)
		fmt.Println("pev.IncrementalValue:", pev.IncrementalValue)
		blockInfoData.PinNumber += 1
		if pev.Host != "metabitcoin.unknown" {
			blockInfoData.PinNumberHasHost += 1
		}
	}
	blockInfoData.AddressNumber = int64(len(addressMap))
	blockInfoData.HostNumber = int64(len(hostMap))
	if metaBlock.MetablockHeight > 0 {
		blockInfoData.HistoryValue, _ = getBlockHistory(metaBlock.MetablockHeight - 1)
	}
	mongoClient.Collection(MetaSoBlockInfoData).UpdateOne(context.TODO(), bson.M{"block": metaBlock.MetablockHeight}, bson.M{"$set": blockInfoData}, options.Update().SetUpsert(true))
	UpdateBlockValue(metaBlock.MetablockHeight, totalPevList, metaBlock.Timestamp)
	fmt.Printf("%+v", hostMap)
	UpdateDataValue(&hostMap, &addressMap)
	log.Println("count metaBlock:", metaBlock.MetablockHeight)
	//mongodb.UpdateSyncLastNumber("metablock", metaBlock.MetablockHeight)
}
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

	var err error
	for _, chain := range metaBlock.Chains {
		endBlock := int64(0)
		maxBlock := int64(0)
		if chain.Chain == "Bitcoin" {
			maxBlock = man.MaxHeight["btc"]
		} else if chain.Chain == "MVC" {
			maxBlock = man.MaxHeight["mvc"]
		}
		if chain.EndBlock == "" {
			continue
		}
		endBlock, err = strconv.ParseInt(chain.EndBlock, 10, 64)
		if err != nil || endBlock <= 0 || endBlock > maxBlock {
			//if err != nil || endBlock <= 0 {
			log.Println(">>", chain.Chain, err, endBlock, maxBlock)
			return
		}
	}
	mongoClient.Collection(MetaSoPEVData).DeleteMany(context.TODO(), bson.M{"metablockheight": -1})
	var totalPevList []interface{}
	for _, chain := range metaBlock.Chains {
		pevList, err := CountBlockPEV(metaBlock.MetablockHeight, &chain)
		log.Println(err, chain.Chain, len(pevList))
		if len(pevList) > 0 {
			totalPevList = append(totalPevList, pevList...)
		}
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
	if metaBlock.MetablockHeight > 0 {
		blockInfoData.HistoryValue, _ = getBlockHistory(metaBlock.MetablockHeight - 1)
	}
	mongoClient.Collection(MetaSoBlockInfoData).UpdateOne(context.TODO(), bson.M{"block": metaBlock.MetablockHeight}, bson.M{"$set": blockInfoData}, options.Update().SetUpsert(true))

	err = UpdateBlockValue(metaBlock.MetablockHeight, totalPevList, metaBlock.Timestamp)
	if err != nil {
		log.Println("UpdateBlockValue:", err)
	}
	err = UpdateDataValue(&hostMap, &addressMap)
	if err != nil {
		log.Println("UpdateDataValue:", err)
	}
	log.Println("count metaBlock:", metaBlock.MetablockHeight)
	mongodb.UpdateSyncLastNumber("metablock", metaBlock.MetablockHeight)

}

func (metaso *MetaSo) SyncPendingPEV() {
	if man.IsSync {
		return
	}
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
	//log.Println("last metaBlock:", lastMetaBlock.MetablockHeight)
	//mongoClient.Collection(MetaSoPEVData).DeleteMany(context.TODO(), bson.M{"metablockheight": -1})
	//btc := bitcoin.BitcoinChain{}
	//mvc := microvisionchain.MicroVisionChain{}
	//btcLastBlockHeight := btc.GetBestHeight()
	btcLastBlockHeight, _ := mongodb.GetSyncLastNumber("btcChainSyncHeight")

	btcBeginBlockHeight := int64(0)
	mvcLastBlockHeight := int64(0)
	mvcBeginBlockHeight := int64(0)
	if man.ChainAdapter["mvc"] != nil {
		//mvcLastBlockHeight = mvc.GetBestHeight()
		mvcLastBlockHeight, _ = mongodb.GetSyncLastNumber("mvcChainSyncHeight")
	}
	btcPendingPevHeight, _ := mongodb.GetSyncLastNumber("btcPendingPevHeight")
	mvcPendingPevHeight, _ := mongodb.GetSyncLastNumber("mvcPendingPevHeight")
	if btcPendingPevHeight >= btcLastBlockHeight && mvcPendingPevHeight >= mvcLastBlockHeight {
		return
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
	btcSyncHeight := btcBeginBlockHeight
	mvcSyncHeight := mvcBeginBlockHeight
	if btcPendingPevHeight > 0 {
		btcSyncHeight = btcPendingPevHeight + 1
	}
	if mvcPendingPevHeight > 0 {
		mvcSyncHeight = mvcPendingPevHeight + 1
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
	syncBlock := &MetaBlockData{
		Header:          "",
		PreHeader:       lastMetaBlock.Header,
		MetablockHeight: -1,
		Chains: []MetaBlockChainData{
			{
				Chain:      "Bitcoin",
				StartBlock: strconv.FormatInt(btcSyncHeight, 10),
				EndBlock:   strconv.FormatInt(btcLastBlockHeight, 10),
			},
			{
				Chain:      "MVC",
				StartBlock: strconv.FormatInt(mvcSyncHeight, 10),
				EndBlock:   strconv.FormatInt(mvcLastBlockHeight, 10),
			},
		},
	}
	log.Println("btcPendingPevHeight:", btcPendingPevHeight, "btcLastBlockHeight:", btcLastBlockHeight, "mvcPendingPevHeight:", mvcPendingPevHeight, "mvcLastBlockHeight:", mvcLastBlockHeight)
	log.Println("syncPendingPev metaBlock:", lastMetaBlock.MetablockHeight)
	for _, chain := range syncBlock.Chains {
		// pevList, _ := CountBlockPEV(pendingBlock.MetablockHeight, &chain)
		// if len(pevList) > 0 {
		// 	totalPevList = append(totalPevList, pevList...)
		// }
		CountBlockPEV(syncBlock.MetablockHeight, &chain)
		if chain.Chain == "MVC" {
			mongodb.UpdateSyncLastNumber("mvcPendingPevHeight", mvcLastBlockHeight)
		}
		if chain.Chain == "Bitcoin" {
			mongodb.UpdateSyncLastNumber("btcPendingPevHeight", btcLastBlockHeight)
		}
	}
	hostMap := make(map[string]struct{})
	addressMap := make(map[string]struct{})
	blockInfoData := &MetaSoBlockInfo{Block: pendingBlock.MetablockHeight, MetaBlock: *pendingBlock}
	totalPevList, err := GetPevDataByMetaBlock(-1)
	var totalPevList2 []interface{}
	if err != nil {
		log.Println("GetPevDataByMetaBlock:", err)
		return
	}
	for _, pev := range totalPevList {
		hostMap[pev.Host] = struct{}{}
		addressMap[pev.Address] = struct{}{}
		blockInfoData.DataValue = blockInfoData.DataValue.Add(pev.IncrementalValue)
		blockInfoData.PinNumber += 1
		if pev.Host != "metabitcoin.unknown" {
			blockInfoData.PinNumberHasHost += 1
		}
		totalPevList2 = append(totalPevList2, pev)
	}
	blockInfoData.AddressNumber = int64(len(addressMap))
	blockInfoData.HostNumber = int64(len(hostMap))
	//blockInfoData.HistoryValue, _ = getBlockHistoryValue(metaBlock.MetablockHeight, "", "")
	mongoClient.Collection(MetaSoBlockInfoData).UpdateOne(context.TODO(), bson.M{"block": pendingBlock.MetablockHeight}, bson.M{"$set": blockInfoData}, options.Update().SetUpsert(true))

	UpdateBlockValue(pendingBlock.MetablockHeight, totalPevList2, pendingBlock.Timestamp)
	UpdateDataValue(&hostMap, &addressMap)
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
	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	resp, err := client.Get(url)
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
	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	resp, err := client.Get(url)
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
