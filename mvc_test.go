package main

import (
	"fmt"
	"log"
	"manindexer/adapter/microvisionchain"
	"manindexer/basicprotocols/metaso"
	"manindexer/common"
	"manindexer/man"
	"manindexer/pin"
	"testing"
	"time"

	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/wire"
)

func TestMvcCatchPinsByTx(t *testing.T) {
	man.InitAdapter("mvc", "mongo", "1", "1")
	txid := "c90955c83ac07fbf4351ede0381becd4e47922e25c4f9f3dc6c11339a1ed360f"
	txResult, err := man.ChainAdapter["mvc"].GetTransaction(txid)
	fmt.Println(err)
	tx := txResult.(*btcutil.Tx)
	fmt.Println(tx.Hash().String())
	index := microvisionchain.Indexer{
		ChainParams: &chaincfg.TestNet3Params,
		PopCutNum:   common.Config.Mvc.PopCutNum,
		DbAdapter:   &man.DbAdapter,
	}
	hash := txid
	list := index.CatchPinsByTx(tx.MsgTx(), 91722, 0, hash, "", 0)
	fmt.Println(list)
}
func TestCatchMvcData(t *testing.T) {
	common.InitConfig("")
	man.InitAdapter("mvc", "mongo", "1", "1")
	//from := 2870989
	//to := 2870990
	// for i := from; i <= to; i++ {
	// 	man.DoIndexerRun("btc", int64(i))
	// }
	//man.DoIndexerRun("mvc", int64(101608), false)
	h := man.ChainAdapter["mvc"].GetBestHeight()
	fmt.Println(h)

}
func TestMvcGetSaveData(t *testing.T) {
	common.InitConfig("./config_mvc.toml")
	man.InitAdapter("mvc", "mongo", "1", "1")
	pinList, _, _, _, _, _, _, _, _, err := man.PebbleStore.GetSaveData("mvc", 120000)
	for _, pinNode := range pinList {
		p := pinNode.(*pin.PinInscription)
		fmt.Println(p.Id, p.PopLv, p.PoPScore)
	}
	fmt.Println(err, len(pinList))
}
func TestGetBestHeight(t *testing.T) {
	common.InitConfig("")
	man.InitAdapter("mvc", "mongo", "1", "1")
	bestHeight := man.ChainAdapter["mvc"].GetBestHeight()
	fmt.Println(bestHeight)
}
func TestMvcGetBlock(t *testing.T) {
	common.InitConfig("./config_mvc.toml")
	man.InitAdapter("mvc", "mongo", "0", "1")
	//122654
	blockMsg, err := man.ChainAdapter["mvc"].GetBlock(123014)
	block := blockMsg.(*wire.MsgBlock)
	for i, tx := range block.Transactions {
		for _, in := range tx.TxIn {
			fmt.Println(i, in.PreviousOutPoint.Hash.String())
		}
		for _, in := range tx.TxIn {
			fmt.Println(i, in.PreviousOutPoint.Hash.String())
		}
	}
	fmt.Println(err)
	fmt.Println(len(block.Transactions))
}
func TestMvcDoIndexerRun(t *testing.T) {
	common.InitConfig("./config_mvc.toml")
	man.InitAdapter("mvc", "mongo", "0", "1")
	startTime := time.Now()
	//122999
	//122654
	//120000
	//120941 61936 Pins
	//122314 210000 Pins
	//122563 350094 Pins
	man.PebbleStore.DoIndexerRun("mvc", 120671, false)
	elapsed := time.Since(startTime)
	fmt.Printf("执行耗时: %s\n", elapsed)
}
func TestMvcPebble(t *testing.T) {
	common.InitConfig("./config_mvc.toml")
	man.InitAdapter("mvc", "mongo", "0", "1")
	pinNode, err := man.PebbleStore.GetPinById("a28bcbf40a2307283ae2580874bc6ec95c88582f1ca800e92eeb4cb34959dcb6i0")
	fmt.Println(err)
	fmt.Println(pinNode)
}
func TestCountBlockPEV(t *testing.T) {
	common.InitConfig("./config_mvc.toml")
	man.InitAdapter("mvc", "mongo", "0", "1")
	pb := metaso.PevPebbledb{}
	metaso.PebblePevInit()
	metaso.ConnectMongoDb()
	var err error
	err = pb.NewDataBase("../pev_data_pebble")
	fmt.Println(err)
	block := &metaso.MetaBlockChainData{
		Chain:       "MVC",
		PreEndBlock: "120669",
		StartBlock:  "120670",
		EndBlock:    "120671",
	}
	blockInfoData := &metaso.MetaSoBlockInfo{
		Block:     1,
		BlockTime: 1672531200, // Example block time
	}

	lastData := &metaso.PevHandle{
		BlockInfoData:  blockInfoData,
		HostMap:        make(map[string]*metaso.MetaSoBlockNDV),
		AddressMap:     make(map[string]*metaso.MetaSoBlockMDV),
		HostAddressMap: make(map[string]*metaso.MetaSoHostAddress),
	}

	err = pb.CountBlockPEV(51, block, lastData, 1672531200)
	fmt.Println(err)
	// for _, pinNode := range pinList {
	// 	p := pinNode.(metaso.PEVData)
	// 	fmt.Println(p.Poplv, p.IncrementalValue)
	// }
	err = pb.UpdateBlockValue(51, lastData, 1672531200)
	if err != nil {
		log.Println("UpdateBlockValue:", err)
		return
	}
}
func TestGetPoPScore(t *testing.T) {
	pop := "00000000000000000000000000033334636114634740160621766746103710301536702156407500462500210217703025774636164673653463655020276177464072614410456650342634434412222501133440"
	fmt.Println("pop:", pop)
	popLv := 6
	score := pin.GetPoPScore(pop, int64(popLv), 21)
	fmt.Println("Lv6 PoP Score:", score)
	popLv = 5
	score = pin.GetPoPScore(pop, int64(popLv), 21)
	fmt.Println("Lv5 PoP Score:", score)
	popLv = 4
	score = pin.GetPoPScore(pop, int64(popLv), 21)
	fmt.Println("Lv4 PoP Score:", score)
}
