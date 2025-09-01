package microvisionchain

import (
	"log"
	"manindexer/adapter/microvisionchain/rpc_client"
	"manindexer/common"
	"manindexer/pin"
	"time"

	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/rpcclient"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"

	chaincfg2 "github.com/bitcoinsv/bsvd/chaincfg"
)

var (
	client *rpcclient.Client
)

type MicroVisionChain struct {
	IsTest bool
}

func (chain *MicroVisionChain) InitChain() {
	mvc := common.Config.Mvc
	rpcConfig := &rpcclient.ConnConfig{
		Host:                 mvc.RpcHost,
		User:                 mvc.RpcUser,
		Pass:                 mvc.RpcPass,
		HTTPPostMode:         mvc.RpcHTTPPostMode, //only supports HTTP POST mode
		DisableTLS:           mvc.RpcDisableTLS,   //core does not provide TLS by default
		DisableAutoReconnect: true,
		DisableConnectOnNew:  true,
	}
	var err error
	client, err = rpcclient.New(rpcConfig, nil)
	if err != nil {
		panic(err)
	}
	log.Println("mvc rpc  connect")
}
func (chain *MicroVisionChain) GetBlock(blockHeight int64) (block interface{}, err error) {
	blockhash, err := client.GetBlockHash(blockHeight)
	if err != nil {
		return
	}
	block, err = client.GetBlock(blockhash)
	return
}
func (chain *MicroVisionChain) GetBlock2(blockHeight int64) (block *wire.MsgBlock, err error) {
	blockhash, err := client.GetBlockHash(blockHeight)
	if err != nil {
		return
	}
	block, err = client.GetBlock(blockhash)
	return
}
func (chain *MicroVisionChain) GetBlockVerbose(blockHeight int64) (block *btcjson.GetBlockVerboseResult, err error) {
	blockhash, err := client.GetBlockHash(blockHeight)
	if err != nil {
		return
	}
	block, err = client.GetBlockVerbose(blockhash)
	return
}
func (chain *MicroVisionChain) GetBlockTime(blockHeight int64) (timestamp int64, err error) {
	block, err := chain.GetBlock(blockHeight)
	if err != nil {
		return
	}
	b := block.(*wire.MsgBlock)
	timestamp = b.Header.Timestamp.Unix()
	return
}
func (chain *MicroVisionChain) GetBlockByHash(hash string) (block *btcjson.GetBlockVerboseResult, err error) {
	blockhash, err := chainhash.NewHashFromStr(hash)
	if err != nil {
		return
	}
	block, err = client.GetBlockVerbose(blockhash)

	return
}
func (chain *MicroVisionChain) GetTransaction(txId string) (tx interface{}, err error) {
	txHash, _ := chainhash.NewHashFromStr(txId)
	return client.GetRawTransaction(txHash)
}
func (chain *MicroVisionChain) GetRawTransaction(txId string) (tx *btcutil.Tx, err error) {
	txHash, _ := chainhash.NewHashFromStr(txId)
	return client.GetRawTransaction(txHash)
}
func GetValueByTx(txId string, txIdx int) (value int64, err error) {
	txHash, _ := chainhash.NewHashFromStr(txId)
	tx, err := client.GetRawTransaction(txHash)
	if err != nil {
		return
	}
	value = tx.MsgTx().TxOut[txIdx].Value
	return
}
func (chain *MicroVisionChain) GetInitialHeight() (height int64) {
	return common.Config.Mvc.InitialHeight
}
func (chain *MicroVisionChain) GetBestHeight() (height int64) {
	blockhash, err := client.GetBestBlockHash()
	if err != nil {
		log.Println("GetBestHeight err:", err)
		return
	}
	block, err := client.GetBlockVerbose(blockhash)
	if err != nil {
		return
	}
	height = block.Height
	//fmt.Println(height)
	return
}
func (chain *MicroVisionChain) GetBlockMsg(height int64) (blockMsg *pin.BlockMsg) {
	blockhash, err := client.GetBlockHash(height)
	if err != nil {
		return
	}
	block, err := client.GetBlockVerbose(blockhash)
	if err != nil {
		return
	}
	blockMsg = &pin.BlockMsg{}
	blockMsg.BlockHash = block.Hash
	blockMsg.Target = block.MerkleRoot
	blockMsg.Weight = int64(block.Weight)
	blockMsg.Timestamp = time.Unix(block.Time, 0).Format("2006-01-02 15:04:05")
	blockMsg.Size = int64(block.Size)
	blockMsg.Transaction = block.Tx
	blockMsg.TransactionNum = len(block.Tx)
	return
}
func (chain *MicroVisionChain) GetCreatorAddress(txHashStr string, idx uint32, netParams *chaincfg.Params) (address string) {
	txHash, err := chainhash.NewHashFromStr(txHashStr)
	if err != nil {
		return "errorAddr"
	}
	tx, err := client.GetRawTransaction(txHash)
	if err != nil {
		return "errorAddr"
	}
	_, addresses, _, _ := txscript.ExtractPkScriptAddrs(tx.MsgTx().TxOut[idx].PkScript, netParams)
	if len(addresses) > 0 {
		address = addresses[0].String()
	} else {
		address = "errorAddr"
	}
	return
}
func (chain *MicroVisionChain) GetMempoolTransactionList() (list []interface{}, err error) {
	return
}

func (chain *MicroVisionChain) GetTxSizeAndFees(txHash string) (fee int64, size int64, blockHash string, err error) {
	hash, err := chainhash.NewHashFromStr(txHash)
	if err != nil {
		return
	}
	tx, err := client.GetRawTransactionVerbose(hash)
	if err != nil {
		return
	}
	var inputAmount int64
	for _, vin := range tx.Vin {
		inputTxHash, err := chainhash.NewHashFromStr(vin.Txid)
		if err != nil {
			continue
		}
		inputTx, err := client.GetRawTransactionVerbose(inputTxHash)
		if err != nil {
			continue
		}
		inputAmount += int64(inputTx.Vout[vin.Vout].Value * 1e8)
	}
	var outputAmount int64
	for _, vout := range tx.Vout {
		outputAmount += int64(vout.Value * 1e8)
	}
	fee = inputAmount - outputAmount
	size = int64(tx.Size)
	blockHash = tx.BlockHash
	return
}

func (chain *MicroVisionChain) BroadcastTx(txRaw string) (txId string, err error) {
	// log.Printf("[MVC]BroadcastTx broadcasting : %s", txRaw)
	rpcClient := rpc_client.NewClientController(common.Config.Mvc.RpcUser, common.Config.Mvc.RpcPass, "http://"+common.Config.Mvc.RpcHost)
	txId, err = rpcClient.BroadcastTx("mvc", txRaw)
	if err != nil {
		// log.Printf("[MVC]BroadcastTx broadcasting failed: %s", err)
		return "", err
	}
	return txId, nil

	// txByte, err := hex.DecodeString(txRaw)
	// if err != nil {
	// 	log.Printf("[MVC]BroadcastTx decoding txRaw failed: %s", err)
	// 	return "", err
	// }
	// tx := wire.NewMsgTx(2)
	// err = tx.Deserialize(bytes.NewReader(txByte))
	// if err != nil {
	// 	log.Printf("[MVC]BroadcastTx deserializing tx failed: %s", err)
	// 	return "", err
	// }
	// txHash, err := client.SendRawTransaction(tx, true)
	// if err != nil {
	// 	log.Printf("[MVC]BroadcastTx sending tx failed: %s", err)
	// 	return "", err
	// }
	// txId = txHash.String()
	// return
}

func (chain *MicroVisionChain) GetNetParam() interface{} {
	if common.TestNet == "1" {
		return &chaincfg2.TestNet3Params
	} else if common.TestNet == "2" {
		return &chaincfg2.RegressionNetParams
	} else {
		return &chaincfg2.MainNetParams
	}
}
