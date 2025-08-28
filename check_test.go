package main

import (
	"fmt"
	"manindexer/common"
	"manindexer/man"
	blockcheck "manindexer/man/block_check"
	"manindexer/pin"
	"testing"

	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/wire"
)

func TestCheckUserInfo(t *testing.T) {
	common.InitConfig("./config.toml")
	common.InitSyncDB()
	common.SaveToDictDB("test", []byte("test_user"))
	v, err := common.LoadFromDictDB("test")
	fmt.Println(err, string(v))
	v, err = common.LoadFromDictDB("test1")
	fmt.Println(err, string(v), v == nil)
}
func TestSaveUserInfo(t *testing.T) {
	common.InitConfig("./config.toml")
	man.InitAdapter("mvc", "mongo", "0", "1")
	// path := man.GetModifyPath("1399be2eac0c45a95a1c863670297dff743a076af74dbf94f66aafddbcd65298i0")
	// fmt.Println(path)
	//fmt.Println(common.Chain)
	// checkChains := []blockcheck.CheckChain{
	// 	{ChainName: "mvc", From: 135167, To: 135167},
	// }
	//bestHeight := man.ChainAdapter["mvc"].GetBestHeight()
	//fmt.Println(bestHeight)
	//blockcheck.DoCheck(checkChains)

	tx, err := man.ChainAdapter["mvc"].GetTransaction("7fb097e28badfbcc3916fe12464fdce30b677909d972d2980716a2e650864125")
	if err != nil {
		t.Fatal(err)
	}
	pins := man.IndexerAdapter["mvc"].CatchPinsByTx(tx.(*btcutil.Tx).MsgTx(), 0, 0, "", "", 0)
	if len(pins) == 0 {
		t.Fatal("No pins found")
	}
	userMap := make(map[string]*pin.MetaIdInfo)
	for _, pinNode := range pins {
		blockcheck.CheckUserInfoPath(pinNode, &userMap)
	}
}

func TestPopLv(t *testing.T) {
	common.InitConfig("./config.toml")
	man.InitAdapter("btc", "mongo", "0", "1")
	block, err := man.ChainAdapter["btc"].GetBlock(910845)
	if err != nil {
		t.Fatal(err)
	}
	b := block.(*wire.MsgBlock)
	fmt.Println("MerkleRoot:", b.Header.MerkleRoot)
	id := "0ab97a467da18c7047c76ae1db84547092c18cafa59f3f075e229a26f5f69d6di0"
	fmt.Println("PINID:", id)
	hash := b.Header.BlockHash()
	fmt.Println("BlockHash:", hash.String())
	pop, _ := common.GenPop(id, b.Header.MerkleRoot.String(), hash.String())
	fmt.Println("POP:", pop)
	popLv, _ := pin.PopLevelCount("btc", pop)
	fmt.Println("POP Level:", popLv)
}
