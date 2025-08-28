package adapter

import (
	"manindexer/mrc20"
	"manindexer/pin"

	"github.com/btcsuite/btcd/wire"
)

type Indexer interface {
	InitIndexer()
	CatchPins(blockHeight int64) (pinInscriptions []*pin.PinInscription, txInList []string, creatorMap map[string]string)
	CatchPinsByTx(msgTx *wire.MsgTx, blockHeight int64, timestamp int64, blockHash string, merkleRoot string, txIndex int) (pinInscriptions []*pin.PinInscription)
	CatchMempoolPins(txList []interface{}) (pinInscriptions []*pin.PinInscription, txInList []string)
	CatchTransfer(idMap map[string]string) (trasferMap map[string]*pin.PinTransferInfo)
	GetAddress(pkScript []byte) (address string)
	ZmqRun(chanMsg chan pin.MempollChanMsg)
	GetBlockTxHash(blockHeight int64) (txhashList []string, pinIdList []string)
	ZmqHashblock()
	CatchNativeMrc20Transfer(blockHeight int64, utxoList []*mrc20.Mrc20Utxo, mrc20TransferPinTx map[string]struct{}) (savelist []*mrc20.Mrc20Utxo)
	CatchMempoolNativeMrc20Transfer(txList []interface{}, utxoList []*mrc20.Mrc20Utxo, mrc20TransferPinTx map[string]struct{}) (savelist []*mrc20.Mrc20Utxo)
}
