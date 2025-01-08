package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"manindexer/adapter/bitcoin"
	"manindexer/basicprotocols/metaaccess"
	"manindexer/basicprotocols/metaso"
	"manindexer/basicprotocols/mrc721"
	"manindexer/common"
	"manindexer/database"
	"manindexer/database/mongodb"
	"manindexer/man"
	"manindexer/pin"
	"math/big"
	"net/url"
	"strconv"
	"testing"

	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/wire"
)

func TestGetBlock(t *testing.T) {
	common.InitConfig()
	chain := &bitcoin.BitcoinChain{}
	block, err := chain.GetBlock(1)
	fmt.Println(err)
	b := block.(*wire.MsgBlock)
	fmt.Println(b.Header.BlockHash().String())

	txret, err := chain.GetTransaction("798a14129d9697906908046998431ee9e97293bc6c5e8d9d3418f1d944913608")
	fmt.Println(err)
	tx := txret.(*btcutil.Tx)
	fmt.Println("HasWitness", tx.HasWitness())
	for _, out := range tx.MsgTx().TxOut {
		fmt.Println(out.Value)
	}

	indexer := &bitcoin.Indexer{ChainParams: &chaincfg.TestNet3Params}
	pins := indexer.CatchPinsByTx(tx.MsgTx(), 123, 11123232, "", "", 0)
	fmt.Println(len(pins))
	for _, pin := range pins {
		fmt.Println("----------------")
		fmt.Printf("%+v\n", pin)
		//fmt.Println("-----------------\ncontent:", string(inscription.Pin.ContentBody))
		//fmt.Println(hex.EncodeToString(inscription.Pin.ContentBody))
	}
}
func TestGetPin(t *testing.T) {
	common.InitConfig()
	man.InitAdapter("btc", "mongo", "1", "1")
	txId := "95abc6fda259c4700d19897d4fd2f2b686504f7fa0a3bb224af743e0473df64a"
	chain := &bitcoin.BitcoinChain{}
	txret, err := chain.GetTransaction(txId)
	if err != nil {
		return
	}
	tx := txret.(*btcutil.Tx)
	fmt.Println("HasWitness", tx.HasWitness())
	indexer := &bitcoin.Indexer{ChainParams: &chaincfg.TestNet3Params}
	pins := indexer.CatchPinsByTx(tx.MsgTx(), 0, 0, "", "", 0)
	fmt.Println(pins)
	for _, pin := range pins {
		fmt.Println(string(pin.ContentBody))
	}
}
func TestAddMempoolPin(t *testing.T) {
	dbAdapter := &mongodb.Mongodb{}
	pin, err := dbAdapter.GetPinByNumberOrId("2")
	fmt.Println(err, pin.Address)
	err = dbAdapter.AddMempoolPin(pin)
	fmt.Println(err)
}
func TestDelMempoolPin(t *testing.T) {
	common.InitConfig()
	man.InitAdapter("btc", "mongo", "1", "1")
	man.DeleteMempoolData(2572919, "btc")
}
func TestConfig(t *testing.T) {
	config := common.Config
	fmt.Println(config.Protocols)
	decimals, err := strconv.ParseInt("", 10, 64)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(decimals)
}

func TestGetDbPin(t *testing.T) {
	common.InitConfig()
	man.InitAdapter("btc", "mongo", "1", "1")
	p, err := man.DbAdapter.GetPinByNumberOrId("999")
	fmt.Println(err)
	//fmt.Println(string(p.ContentBody))
	//contentType := common.DetectContentType(&p.ContentBody)
	//fmt.Println(contentType)
	standardEncoded := base64.StdEncoding.EncodeToString(p.ContentBody)
	fmt.Println(standardEncoded)
}
func TestMongoGeneratorFind(t *testing.T) {
	jsonData := `
	{"collection":"pins","action":"sum","filterRelation":"or","field":["number"],
	"filter":[{"operator":"=","key":"number","value":1},{"operator":"=","key":"number","value":2}],
	"cursor":0,"limit":1,"sort":["number","desc"]
	}
	`
	var g database.Generator
	err := json.Unmarshal([]byte(jsonData), &g)
	fmt.Println(err)
	fmt.Println(g.Action)
	mg := mongodb.Mongodb{}
	ret, err := mg.GeneratorFind(g)
	fmt.Println(err, len(ret))
	if err == nil {
		jsonStr, err1 := json.Marshal(ret)
		if err1 != nil {
			fmt.Println("Error marshalling JSON:", err)
		}
		fmt.Println(string(jsonStr))
	}
}
func TestGetSaveData(t *testing.T) {
	common.InitConfig()
	man.InitAdapter("btc", "mongo", "1", "1")
	pinList, _, _, _, _, mrc20List, _, _, _, _, err := man.GetSaveData("btc", 2868996)
	fmt.Println(err, len(pinList), len(mrc20List))
	// var testList []*pin.PinInscription
	// for _, mrc20 := range mrc20List {
	// 	if mrc20.GenesisTransaction == "3f7f5a5b31b97df8d8c568b649ce8e8f38f39db714a8f52ac104b6d2dd889d45" {
	// 		testList = append(testList, mrc20)
	// 	}
	// }
	//man.Mrc20Handle(testList)
	//man.Mrc20Handle(mrc20List)
}
func TestCatchData(t *testing.T) {
	common.InitConfig()
	man.InitAdapter("btc", "mongo", "1", "1")
	//from := 2870989
	//to := 2870990
	// for i := from; i <= to; i++ {
	// 	man.DoIndexerRun("btc", int64(i))
	// }
	man.DoIndexerRun("btc", int64(2873530))

}
func TestHash(t *testing.T) {
	common.InitConfig()
	add := "tb1qtjqupfjej6a9wu94g374fvnlq6ks9v4am7hwtz"
	h := common.GetMetaIdByAddress(add)
	fmt.Println(add)
	fmt.Println(h)
}
func TestGetOwner(t *testing.T) {
	common.InitConfig()
	man.InitAdapter("btc", "mongo", "1", "1")
	//txResult, err := man.ChainAdapter.GetTransaction("d8373e66a6852331c667c94bdccdac94b4908b7ca47b35a00d90a76ae29eb015")
	//fmt.Println(err)
	//tx := txResult.(*btcutil.Tx)
	//inpitId := "8fb1a5154b013f1efaae82a922e03419d6d765006812e6cf32e7b8709971a8c7:0"
	//man.IndexerAdapter.GetOWnerAddress()
	// index := bitcoin.Indexer{
	// 	ChainParams: &chaincfg.TestNet3Params,
	// 	PopCutNum:   common.Config.Btc.PopCutNum,
	// 	DbAdapter:   &man.DbAdapter,
	// }
	// info, err := index.GetOWnerAddress(inpitId, tx.MsgTx())
	// fmt.Println(err)
	// fmt.Printf("%+v", info)
	// list, err := index.TransferCheck(tx.MsgTx())
	// fmt.Println(err)
	// for _, l := range list {
	// 	fmt.Printf("%+v", l)
	// }
	ll, e := man.DbAdapter.GetMempoolTransfer("tb1q3h9twrcz7s5mz7q2eu6pneex446tp3v5yasnp5", "")
	fmt.Println(e, len(ll))
}
func TestRarityScoreBinary(t *testing.T) {
	str := "00000000000000000000000000354712732267161417502043436707557310655121055015573522441662265776662610002362543123510570022146525640016535265733565315137521366643101110550222"
	//fmt.Println(pin.RarityScoreBinary("000001010101"))
	fmt.Println(pin.RarityScoreBinary("btc", str))

}

func TestMrc721Save(t *testing.T) {
	common.InitConfig()
	man.InitAdapter("btc", "mongo", "1", "1")
	man.DoIndexerRun("btc", int64(2874040))
}
func TestMempoolTransfer(t *testing.T) {
	common.InitConfig()
	man.InitAdapter("btc", "mongo", "1", "1")
	txId := "d076a9f456535b82cf2c8f9c9c59a7516dc040652f8ef41acd7c839bb98fdd4b"
	chain := &bitcoin.BitcoinChain{}
	txret, err := chain.GetTransaction(txId)
	if err != nil {
		return
	}
	tx := txret.(*btcutil.Tx)
	fmt.Println("HasWitness", tx.Hash())
	mm := man.ManMempool{}
	var list []interface{}

	list = append(list, tx.MsgTx())
	mm.CheckMempoolHadle("btc", list)
}
func TestCreateMetaId(t *testing.T) {
	fmt.Println(common.GetMetaIdByAddress("a"))
}
func TestEcdh(t *testing.T) {
	content := []byte(`oWg+H3pEuwqgFeLwTyQvch3h3jNpv4FTPutTQGw9C9aqGzyzrK5alIjpqW1fpxAkUYP5H9YGaMHII3UPNveox4jDEVyXQKnAukpD5Pn5Au11mGUISptLLh7kk1+k3L5uqhuXOm7JwiUY5oJ0yMtEjEgcjhmvfnFl/NWtjnUGQ0/4wCBUaRIgRFWotcR99gYKv2KmyOahj1ks0Jk2PhLV6uvoMHmaQTmy9RMVd8a8bFKP2ej2HaCyFADO24yyMrGt3iYe+Bjv3kU6Kd77vU5T+t+WfO2o1wrSct0HSD9hojEcGVavvlIbLszOAP9NYHLVx0XuMm/wjisM83g9wGloYh7AQHlkfTIQJlygxCQdA6qx/Kwtr/join1VFiAylaHr24DkAPZt/eYUn6sPoDpXFdfEmeSlFBjha2mwVLXjdikWcFYf81aNhAS1dSqw4tE47ZJ1nKeZBAacJ3D24ttod4VYbza89oYERAGEFCcX5/pxm87AMoQye0Pyb4YEExOSRcbY2Exf7DXK7WEr`)
	prikey := "5a34bc2e4edecd778faa6ed8dd38537f3152dec479ef95d0d608f388c3aa7aed"
	creatorPubkey := "04788e92954b89ecd4a149d2e3b2eca5ce58613ce712ad0901e56466a639f5e87d93c44fd2cf3b874ea2c5b963ceec5cb2f03e7c5a8f2b50462dfe7b63282c5a48"
	key := `1Pu9/39TWhjctvE6zXBlToqSQrp1djWfSBsNrIz4e50F4Nx43bnsf0H6Hd5fSO5FkAYS0lXQ9CZCuIGA4XrNagBBGX3gwHmS9ZSRda3pBIo=`
	r, f, err := metaaccess.DecryptionPin(content, [][]byte{}, prikey, creatorPubkey, key)
	fmt.Println(err, r, f)
	// publicKey := "03e090905baad30b208f29d57319a6fa9d2acc3578f2ac9152ebcacf9b3581f63a"
	// timestamp := int64(1731825669)
	// address := "2N1qfdmWkeREeoSbvco4zg2T1QunZZwc6ee"
	// sign := "6592346e329b5a86060a1c3a82e382e2a2ffd3168219dcd03921f7d1a4c9e96b"
	// err := metaaccess.CheckSign(publicKey, prikey, timestamp, address, sign)
	// fmt.Println(err)
}
func TestCheckIdcoin(t *testing.T) {
	data := `{"message":"","tickSign":"IOjNdOU1Sa1Xq0Uhzptim2IjKnBtLtEQayiFXW7EGaJiFY4dmIaY0pzrrSCNj97pWcyTFRN/MRwRaWKEFrgUth0="}`
	ret := metaso.CheckIdCoins("main", "OCEAN", data, 1723140496)
	fmt.Println(">>", ret)

}
func TestUrlEncode(t *testing.T) {
	s := "w&+l&q"
	e := url.PathEscape(s)
	fmt.Println(e)
}
func TestPinHost(t *testing.T) {
	b, h, p := pin.ValidHostPath("/aa/bb/cc")
	fmt.Println(b, h, p)
}
func octalStringToDecimal(octalStr string, intNum int, divisor float64) (*float64, error) {
	decimalNum := new(big.Int)
	base := big.NewInt(8)
	for _, char := range octalStr {
		digit := int64(char - '0')
		if digit < 0 || digit > 7 {
			return nil, fmt.Errorf("err: %c", char)
		}

		decimalNum.Mul(decimalNum, base)
		decimalNum.Add(decimalNum, big.NewInt(digit))
	}
	bigIntStrFull := decimalNum.String()
	bingIntStr := ""
	if len(bigIntStrFull) > intNum {
		bingIntStr = bigIntStrFull[:intNum]
	} else {
		bingIntStr = bigIntStrFull
	}
	firstFourInt, err := strconv.Atoi(bingIntStr)
	if err != nil {
		return nil, err
	}
	result := float64(firstFourInt) / divisor
	//rounded := math.Round(result*10000) / 10000
	return &result, nil
}

func TestPopValue(t *testing.T) {
	octalStr1 := "11042570125615234640004625102106357306043255550550076564537140365102132415230142366044306202554235231160276211343053103755346436433707052743553203325000000000000000000000"
	//octalStr1 = octalStr1[:20]
	decimalNum1, _ := octalStringToDecimal(octalStr1, 4, 10000)
	fmt.Println("mvc pop:", octalStr1)
	fmt.Println("MVC", len("000000000000000000000"), "0，Value=", *decimalNum1)
	octalStr1 = "25732332603521704665072554007343661227204215575466401372767200614232166343735662202410272547004153704147775065263545712557515043367451760000000000000000000000000000000000"
	//octalStr1 = octalStr1[:20]
	decimalNum1, _ = octalStringToDecimal(octalStr1, 4, 10000)
	fmt.Println("btc pop:", octalStr1)
	fmt.Println("BTC", len("000000000000000000000000000000000"), "0，Value=", *decimalNum1)
}
func TestMrc721SysnAddress(t *testing.T) {
	mrc721.SyncAddress()
}
