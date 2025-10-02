package main

import (
	"encoding/json"
	"fmt"
	"log"
	"manindexer/common"
	"manindexer/database/mongodb"
	"manindexer/man"
	"manindexer/pebblestore"
	"manindexer/pin"
	"testing"
)

func TestPinPageList(t *testing.T) {
	common.InitConfig("./config_mvc.toml")
	man.InitAdapter("mvc", "mongo", "0", "1")
	list, nextId, err := man.PebbleStore.PinPageList(0, 1, "")
	fmt.Println(err, len(list), nextId)
	cnt, err := mongodb.CountMetaid()
	fmt.Println(err, cnt)
	//fmt.Println(list, nextId)
}
func TestQueryPageBlock(t *testing.T) {
	common.InitConfig("./config_mvc.toml")
	man.InitAdapter("mvc", "mongo", "0", "1")
	q := pebblestore.PageQuery{Type: "pin", Page: 0, Size: 2, LastId: ""}
	list, err := man.PebbleStore.QueryPageBlock(q)
	fmt.Println(err)
	fmt.Println(list)
}
func TestGetMvcBlockFee(t *testing.T) {
	common.InitConfig("./config_mvc.toml")
	man.InitAdapter("mvc", "mongo", "0", "1")
	feeRateInfo := make(map[string]int, 2728)
	pebblestore.GetMvcBlockFee(126573, &feeRateInfo)
	fmt.Println(feeRateInfo)
}
func TestGetBlockData(t *testing.T) {
	common.InitConfig("./config_regtest.toml")
	man.InitAdapter("btc", "mongo", "2", "1")
	//blockdata.GetData("btc", 432)
	err := man.SaveBlockFileFromChain("btc", 430)
	fmt.Println("SaveBlockFile:", err)
	// 从文件加载区块数据
	loadedData, err := man.LoadFBlockPart("btc", 430, 0)
	if err != nil {
		log.Fatalf("加载区块失败: %v", err)
	}
	fmt.Println("loaded len:", len(loadedData))
	for _, v := range loadedData {
		var pinNode pin.PinInscription
		err := json.Unmarshal(v, &pinNode)
		if err == nil {
			fmt.Println(pinNode.Id)
		}
	}

}
