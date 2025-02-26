package adapter

import "manindexer/pin"

type Chain interface {
	InitChain()
	GetBlock(blockHeight int64) (block interface{}, err error)
	GetBlockTime(blockHeight int64) (timestamp int64, err error)
	GetTransaction(txId string) (tx interface{}, err error)
	GetInitialHeight() (height int64)
	GetBestHeight() (height int64)
	GetBlockMsg(height int64) (blockMsg *pin.BlockMsg)
	GetMempoolTransactionList() (list []interface{}, err error)
	GetTxSizeAndFees(txHash string) (fee int64, size int64, blockHash string, err error)
}
