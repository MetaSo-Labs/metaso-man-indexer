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

// func (metaso *MetaSo) SyncPEVTest(height int64) {
// 	metaBlock := getMetaBlock(height)
// 	var err error
// 	for _, chain := range metaBlock.Chains {
// 		endBlock := int64(0)
// 		maxBlock := int64(0)
// 		if chain.Chain == "Bitcoin" {
// 			maxBlock = man.MaxHeight["btc"]
// 		} else if chain.Chain == "MVC" {
// 			maxBlock = man.MaxHeight["mvc"]
// 		}
// 		if chain.EndBlock == "" {
// 			continue
// 		}
// 		endBlock, err = strconv.ParseInt(chain.EndBlock, 10, 64)
// 		if err != nil || endBlock <= 0 {
// 			log.Println(">>", chain.Chain, err, endBlock, maxBlock)
// 			return
// 		}
// 	}

//		var totalPevList []interface{}
//		for _, chain := range metaBlock.Chains {
//			pevList, err := CountBlockPEV(metaBlock.MetablockHeight, &chain)
//			fmt.Println(err)
//			if len(pevList) > 0 {
//				totalPevList = append(totalPevList, pevList...)
//			}
//		}
//		fmt.Println("len:", len(totalPevList))
//		hostMap := make(map[string]struct{})
//		addressMap := make(map[string]struct{})
//		blockInfoData := &MetaSoBlockInfo{Block: metaBlock.MetablockHeight, MetaBlock: *metaBlock}
//		for _, item := range totalPevList {
//			pev := item.(PEVData)
//			hostMap[pev.Host] = struct{}{}
//			addressMap[pev.Address] = struct{}{}
//			blockInfoData.DataValue = blockInfoData.DataValue.Add(pev.IncrementalValue)
//			fmt.Println("pev.IncrementalValue:", pev.IncrementalValue)
//			blockInfoData.PinNumber += 1
//			if pev.Host != "metabitcoin.unknown" {
//				blockInfoData.PinNumberHasHost += 1
//			}
//		}
//		blockInfoData.AddressNumber = int64(len(addressMap))
//		blockInfoData.HostNumber = int64(len(hostMap))
//		if metaBlock.MetablockHeight > 0 {
//			blockInfoData.HistoryValue, _ = getBlockHistory(metaBlock.MetablockHeight - 1)
//		}
//		mongoClient.Collection(MetaSoBlockInfoData).UpdateOne(context.TODO(), bson.M{"block": metaBlock.MetablockHeight}, bson.M{"$set": blockInfoData}, options.Update().SetUpsert(true))
//		UpdateBlockValue(metaBlock.MetablockHeight, totalPevList, metaBlock.Timestamp)
//		fmt.Printf("%+v", hostMap)
//		UpdateDataValue(&hostMap, &addressMap)
//		log.Println("count metaBlock:", metaBlock.MetablockHeight)
//		//mongodb.UpdateSyncLastNumber("metablock", metaBlock.MetablockHeight)
//	}
var pb PevPebbledb

func PebblePevInit() {
	pb = PevPebbledb{}
	err := pb.NewDataBase("./pev_data_pebble")
	if err != nil {
		log.Println("PevPebbledb pev_data_pebble error:", err)
		return
	}
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
	log.Println("===>Begin syncPEV metaBlock:", metaBlock.MetablockHeight)
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
	// pebbledb不需要删除，直接set即可
	//mongoClient.Collection(MetaSoPEVData).DeleteMany(context.TODO(), bson.M{"metablockheight": -1})
	//var totalPevList []interface{}
	//var totalPevList []PEVData
	blockInfoData := &MetaSoBlockInfo{Block: metaBlock.MetablockHeight, MetaBlock: *metaBlock}
	lastData := &PevHandle{
		BlockInfoData:  blockInfoData,
		HostMap:        make(map[string]*MetaSoBlockNDV),
		AddressMap:     make(map[string]*MetaSoBlockMDV),
		HostAddressMap: make(map[string]*MetaSoHostAddress),
	}
	for _, chain := range metaBlock.Chains {
		err := pb.CountBlockPEV(metaBlock.MetablockHeight, &chain, lastData, metaBlock.Timestamp)
		// log.Println(err, chain.Chain, len(pevList))
		// if len(pevList) > 0 {
		// 	totalPevList = append(totalPevList, pevList...)
		// }
		if err != nil {
			log.Println("CountBlockPEV ERR:", err, chain.Chain, "metablock:", metaBlock.MetablockHeight)
		}
	}
	// hostMap := make(map[string]struct{})
	// addressMap := make(map[string]struct{})

	// for _, pev := range totalPevList {
	// 	//pev := item.(PEVData)
	// 	hostMap[pev.Host] = struct{}{}
	// 	addressMap[pev.Address] = struct{}{}
	// 	blockInfoData.DataValue = blockInfoData.DataValue.Add(pev.IncrementalValue)
	// 	blockInfoData.PinNumber += 1
	// 	if pev.Host != "metabitcoin.unknown" {
	// 		blockInfoData.PinNumberHasHost += 1
	// 	}
	// }
	blockInfoData.AddressNumber = int64(len(lastData.AddressMap))
	blockInfoData.HostNumber = int64(len(lastData.HostMap))
	if metaBlock.MetablockHeight > 0 {
		blockInfoData.HistoryValue, _ = pb.getBlockHistory(metaBlock.MetablockHeight - 1)
	}
	mongoClient.Collection(MetaSoBlockInfoData).UpdateOne(context.TODO(), bson.M{"block": metaBlock.MetablockHeight}, bson.M{"$set": blockInfoData}, options.Update().SetUpsert(true))
	err = pb.UpdateBlockValue(metaBlock.MetablockHeight, lastData, metaBlock.Timestamp)
	if err != nil {
		log.Println("UpdateBlockValue:", err)
		return
	}

	// err = pb.SaveBlockPevData(metaBlock.MetablockHeight, &totalPevList)
	// if err != nil {
	// 	return
	// }
	log.Println("count metaBlock:", metaBlock.MetablockHeight)
	ClearMemPool()
	mongodb.UpdateSyncLastNumber("metablock", metaBlock.MetablockHeight)
}
func ClearMemPool() {
	_, err := mongoClient.Collection(MetaSoMDVBlockData).DeleteMany(context.TODO(), bson.M{"block": -1})
	if err != nil {
		log.Println("clear MetaSoMDVBlockData mempool err:", err)
	}
	_, err = mongoClient.Collection(MetaSoNDVBlockData).DeleteMany(context.TODO(), bson.M{"block": -1})
	if err != nil {
		log.Println("clear MetaSoNDVBlockData mempool err:", err)
	}
	_, err = mongoClient.Collection(MetaSoHostAddressData).DeleteMany(context.TODO(), bson.M{"block": -1})
	if err != nil {
		log.Println("clear MetaSoNDVBlockData mempool err:", err)
	}
}
func (metaso *MetaSo) SyncPendingPEV() {
	// if man.IsSync {
	// 	return
	// }
	if common.Config.Statistics.MetaChainHost == "" || common.Config.Statistics.AllowHost == nil || common.Config.Statistics.AllowProtocols == nil {
		log.Println("SyncPendingPEV Config Check Err")
		return
	}
	localHeight, err := mongodb.GetSyncLastNumber("metablock")
	if err != nil {
		log.Println("GetSyncLastNumber metaso:", err)
		return
	}
	lastBlock := getLastMetaBlock()
	if lastBlock == nil {
		log.Println("getLastMetaBlock is nil")
		return
	}
	if lastBlock.LastNumber > localHeight {
		log.Println("SyncPendingPEV:lastBlock.LastNumber != localHeight")
		return
	}
	// metaBlock, _ := metaso.getLastMetaBlock(1)
	// if metaBlock != nil {
	// 	log.Println("SyncPendingPEV:wait sync latest", metaBlock.MetablockHeight)
	// 	return
	// }
	lastMetaBlock, _ := metaso.getLastMetaBlock(0)
	if lastMetaBlock == nil {
		log.Println("SyncPendingPEV:getLastMetaBlock  nil")
		return
	}
	if lastMetaBlock.Header == "" {
		return
	}

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
	log.Println("===>Begin syncPendingPev metaBlock:", lastMetaBlock.MetablockHeight)
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
	blockInfoData := &MetaSoBlockInfo{Block: pendingBlock.MetablockHeight, MetaBlock: *pendingBlock}

	lastData := &PevHandle{
		BlockInfoData:  blockInfoData,
		HostMap:        make(map[string]*MetaSoBlockNDV),
		AddressMap:     make(map[string]*MetaSoBlockMDV),
		HostAddressMap: make(map[string]*MetaSoHostAddress),
	}
	for _, chain := range syncBlock.Chains {
		// pevList, _ := CountBlockPEV(pendingBlock.MetablockHeight, &chain)
		// if len(pevList) > 0 {
		// 	totalPevList = append(totalPevList, pevList...)
		// }
		//pb.CountBlockPEV(syncBlock.MetablockHeight, &chain)
		err := pb.CountBlockPEV(-1, &chain, lastData, time.Now().Unix())
		if err != nil {
			log.Println("CountBlockPEV ERR:", err, chain.Chain, "metablock:", -1)
		}
		if chain.Chain == "MVC" {
			mongodb.UpdateSyncLastNumber("mvcPendingPevHeight", mvcLastBlockHeight)
		}
		if chain.Chain == "Bitcoin" {
			mongodb.UpdateSyncLastNumber("btcPendingPevHeight", btcLastBlockHeight)
		}
	}
	// hostMap := make(map[string]struct{})
	// addressMap := make(map[string]struct{})
	//blockInfoData := &MetaSoBlockInfo{Block: pendingBlock.MetablockHeight, MetaBlock: *pendingBlock}
	// totalPevList, err := pb.GetPevDataByMetaBlock(-1)
	// var totalPevList2 []PEVData
	// if err != nil {
	// 	log.Println("GetPevDataByMetaBlock:", err)
	// 	return
	// }
	// for _, pev := range totalPevList {
	// 	hostMap[pev.Host] = struct{}{}
	// 	addressMap[pev.Address] = struct{}{}
	// 	blockInfoData.DataValue = blockInfoData.DataValue.Add(pev.IncrementalValue)
	// 	blockInfoData.PinNumber += 1
	// 	if pev.Host != "metabitcoin.unknown" {
	// 		blockInfoData.PinNumberHasHost += 1
	// 	}
	// 	totalPevList2 = append(totalPevList2, pev)
	// }

	blockInfoData.AddressNumber = int64(len(lastData.AddressMap))
	blockInfoData.HostNumber = int64(len(lastData.HostMap))
	//blockInfoData.HistoryValue, _ = getBlockHistoryValue(metaBlock.MetablockHeight, "", "")
	update := bson.D{
		{Key: "$inc", Value: bson.D{
			{Key: "addressnumber", Value: lastData.BlockInfoData.AddressNumber},
			{Key: "hostnumber", Value: lastData.BlockInfoData.HostNumber},
			{Key: "datavalue", Value: lastData.BlockInfoData.DataValue},
			{Key: "pinnumber", Value: lastData.BlockInfoData.PinNumber},
		}},
		{Key: "$setOnInsert", Value: blockInfoData},
	}

	mongoClient.Collection(MetaSoBlockInfoData).UpdateOne(context.TODO(), bson.M{"block": pendingBlock.MetablockHeight}, update, options.Update().SetUpsert(true))

	pb.UpdateBlockValue(pendingBlock.MetablockHeight, lastData, pendingBlock.Timestamp)
	//pb.UpdateDataValue(&hostMap, &addressMap)
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
