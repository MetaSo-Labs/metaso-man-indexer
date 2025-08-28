package microvisionchain

import (
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"manindexer/common"
	"manindexer/database"
	"manindexer/mrc20"
	"manindexer/pin"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
)

var PopCutNum int = 0

type Indexer struct {
	ChainParams *chaincfg.Params
	Block       interface{}
	PopCutNum   int
	DbAdapter   *database.Db
	ChainName   string
}

func (indexer *Indexer) InitIndexer() {
	PopCutNum = common.Config.Mvc.PopCutNum
}
func (indexer *Indexer) GetCurHeight() (height int64) {
	return
}
func (indexer *Indexer) GetAddress(pkScript []byte) (address string) {
	_, addresses, _, _ := txscript.ExtractPkScriptAddrs(pkScript, indexer.ChainParams)
	if len(addresses) > 0 {
		address = addresses[0].String()
	}
	return
}
func (indexer *Indexer) _bak(blockHeight int64) (pinInscriptions []*pin.PinInscription, txInList []string, creatorMap map[string]string) {
	chain := MicroVisionChain{}
	blockMsg, err := chain.GetBlock(blockHeight)
	if err != nil {
		return
	}
	indexer.Block = blockMsg
	block := blockMsg.(*wire.MsgBlock)

	timestamp := block.Header.Timestamp.Unix()
	blockHash := block.BlockHash().String()
	merkleRoot := block.Header.MerkleRoot.String()
	creatorMap = make(map[string]string)
	for i, tx := range block.Transactions {
		for _, in := range tx.TxIn {
			//id := fmt.Sprintf("%s:%d", in.PreviousOutPoint.Hash.String(), in.PreviousOutPoint.Index)
			id := common.ConcatBytesOptimized([]string{in.PreviousOutPoint.Hash.String(), ":", strconv.FormatUint(uint64(in.PreviousOutPoint.Index), 10)}, "")
			txInList = append(txInList, id)
		}
		txPins := indexer.CatchPinsByTx(tx, blockHeight, timestamp, blockHash, merkleRoot, i)
		if len(txPins) > 0 {
			pinInscriptions = append(pinInscriptions, txPins...)
		}
	}

	return
}
func (indexer *Indexer) CatchPins_BAK(blockHeight int64) (pinInscriptions []*pin.PinInscription, txInList []string, creatorMap map[string]string) {
	chain := MicroVisionChain{}
	blockMsg, err := chain.GetBlockVerbose(blockHeight)
	if err != nil {
		return
	}
	//indexer.Block = blockMsg
	//block := blockMsg.(*wire.MsgBlock)

	timestamp := blockMsg.Time
	blockHash := blockMsg.Hash
	merkleRoot := blockMsg.MerkleRoot
	creatorMap = make(map[string]string)

	txids := blockMsg.Tx

	batchSize := 1000
	for i := 0; i < len(txids); i += batchSize {
		end := i + batchSize
		if end > len(txids) {
			end = len(txids)
		}
		batch := txids[i:end]
		// 并发获取这一批原始交易
		var wg sync.WaitGroup
		for _, txid := range batch {
			wg.Add(1)
			go func(txid string) {
				defer wg.Done()
				tx, _ := chain.GetRawTransaction(txid)
				if tx == nil {
					return
				}
				// 这里直接处理 tx
				for _, in := range tx.MsgTx().TxIn {
					//id := fmt.Sprintf("%s:%d", in.PreviousOutPoint.Hash.String(), in.PreviousOutPoint.Index)
					id := common.ConcatBytesOptimized([]string{in.PreviousOutPoint.Hash.String(), ":", strconv.FormatUint(uint64(in.PreviousOutPoint.Index), 10)}, "")
					txInList = append(txInList, id)
				}
				txPins := indexer.CatchPinsByTx(tx.MsgTx(), blockHeight, timestamp, blockHash, merkleRoot, i)
				if len(txPins) > 0 {
					pinInscriptions = append(pinInscriptions, txPins...)
				}
			}(txid)
		}
		wg.Wait()
	}

	return
}
func (indexer *Indexer) CatchPins(blockHeight int64) (pinInscriptions []*pin.PinInscription, txInList []string, creatorMap map[string]string) {
	chain := MicroVisionChain{}
	st := time.Now()
	blockMsg, err := chain.GetBlock2(blockHeight)
	if err != nil {
		log.Println("GetBlock2 Error:", err)
		return
	}
	fmt.Println("GetBlock Data from Node:", time.Since(st))
	//indexer.Block = blockMsg
	//block := blockMsg.(*wire.MsgBlock)

	timestamp := blockMsg.Header.Timestamp.Unix()
	blockHash := blockMsg.BlockHash().String()
	merkleRoot := blockMsg.Header.MerkleRoot.String()
	creatorMap = make(map[string]string)
	st = time.Now()
	for i, tx := range blockMsg.Transactions {
		// if i%10000 == 0 {
		// 	fmt.Println("Catch Block Pins By Tx:", i, "/", len(blockMsg.Transactions), "Time:", time.Since(st))
		// 	st = time.Now()
		// }
		for _, in := range tx.TxIn {
			//id := fmt.Sprintf("%s:%d", in.PreviousOutPoint.Hash.String(), in.PreviousOutPoint.Index)
			id := common.ConcatBytesOptimized([]string{in.PreviousOutPoint.Hash.String(), ":", strconv.FormatUint(uint64(in.PreviousOutPoint.Index), 10)}, "")
			txInList = append(txInList, id)
		}
		txPins := indexer.CatchPinsByTx(tx, blockHeight, timestamp, blockHash, merkleRoot, i)
		if len(txPins) > 0 {
			pinInscriptions = append(pinInscriptions, txPins...)
		}
	}
	fmt.Println("Catch Block Pins By Tx:", time.Since(st))
	// txids := blockMsg.Tx

	// batchSize := 1000
	// for i := 0; i < len(txids); i += batchSize {
	// 	end := i + batchSize
	// 	if end > len(txids) {
	// 		end = len(txids)
	// 	}
	// 	batch := txids[i:end]
	// 	// 并发获取这一批原始交易
	// 	var wg sync.WaitGroup
	// 	for _, txid := range batch {
	// 		wg.Add(1)
	// 		go func(txid string) {
	// 			defer wg.Done()
	// 			tx, _ := chain.GetRawTransaction(txid)
	// 			if tx == nil {
	// 				return
	// 			}
	// 			// 这里直接处理 tx
	// 			for _, in := range tx.MsgTx().TxIn {
	// 				//id := fmt.Sprintf("%s:%d", in.PreviousOutPoint.Hash.String(), in.PreviousOutPoint.Index)
	// 				id := common.ConcatBytesOptimized([]string{in.PreviousOutPoint.Hash.String(), ":", strconv.FormatUint(uint64(in.PreviousOutPoint.Index), 10)}, "")
	// 				txInList = append(txInList, id)
	// 			}
	// 			txPins := indexer.CatchPinsByTx(tx.MsgTx(), blockHeight, timestamp, blockHash, merkleRoot, i)
	// 			if len(txPins) > 0 {
	// 				pinInscriptions = append(pinInscriptions, txPins...)
	// 			}
	// 		}(txid)
	// 	}
	// 	wg.Wait()
	// }
	return
}
func (indexer *Indexer) CatchMempoolPins(txList []interface{}) (pinInscriptions []*pin.PinInscription, txInList []string) {
	//TODO
	return
}
func (indexer *Indexer) CatchTransfer(idMap map[string]string) (trasferMap map[string]*pin.PinTransferInfo) {
	trasferMap = make(map[string]*pin.PinTransferInfo)
	block := indexer.Block.(*wire.MsgBlock)
	for _, tx := range block.Transactions {
		for _, in := range tx.TxIn {
			id := fmt.Sprintf("%s:%d", in.PreviousOutPoint.Hash.String(), in.PreviousOutPoint.Index)
			if fromAddress, ok := idMap[id]; ok {
				info, err := indexer.GetOWnerAddress(id, tx)
				if err == nil && info != nil {
					info.FromAddress = fromAddress
					trasferMap[id] = info
				}
			}
		}
	}
	return
}
func (indexer *Indexer) GetOWnerAddress(inputId string, tx *wire.MsgTx) (info *pin.PinTransferInfo, err error) {
	//fmt.Println("tx:", tx.TxHash().String(), inputId)
	info = &pin.PinTransferInfo{}
	firstInputId := fmt.Sprintf("%s:%d", tx.TxIn[0].PreviousOutPoint.Hash, tx.TxIn[0].PreviousOutPoint.Index)
	//!!!加速索引，都给第一个
	if len(tx.TxIn) == 1 || firstInputId == inputId || 1 == 1 {
		class, addresses, _, _ := txscript.ExtractPkScriptAddrs(tx.TxOut[0].PkScript, indexer.ChainParams)
		if len(addresses) > 0 {
			info.Address = addresses[0].String()
		} else if class.String() == "nulldata" {
			info.Address = hex.EncodeToString(tx.TxOut[0].PkScript)
		}
		info.Location = fmt.Sprintf("%s:%d:%d", tx.TxHash().String(), 0, 0)
		info.Offset = 0
		info.Output = fmt.Sprintf("%s:%d", tx.TxHash().String(), 0)
		info.OutputValue = tx.TxOut[0].Value
		return
	}
	totalOutputValue := int64(0)
	for _, out := range tx.TxOut {
		totalOutputValue += out.Value
	}
	inputValue := int64(0)
	for _, in := range tx.TxIn {
		id := fmt.Sprintf("%s:%d", in.PreviousOutPoint.Hash, in.PreviousOutPoint.Index)
		if id == inputId {
			break
		}
		value, err1 := GetValueByTx(in.PreviousOutPoint.Hash.String(), int(in.PreviousOutPoint.Index))
		if err1 != nil {
			err = errors.New("get value error")
			return
		}
		inputValue += value
		if inputValue > totalOutputValue {
			return
		}
	}
	outputValue := int64(0)
	for i, out := range tx.TxOut {
		outputValue += out.Value
		if outputValue > inputValue {
			class, addresses, _, _ := txscript.ExtractPkScriptAddrs(out.PkScript, indexer.ChainParams)
			if len(addresses) > 0 {
				info.Address = addresses[0].String()
			} else if class.String() == "nulldata" {
				info.Address = hex.EncodeToString(out.PkScript)
			}
			info.Output = fmt.Sprintf("%s:%d", tx.TxHash().String(), i)
			//count offset
			info.Location = fmt.Sprintf("%s:%d", info.Output, out.Value-(outputValue-inputValue))
			info.Offset = uint64(i)
			info.OutputValue = out.Value
			break
		}
	}

	return
}
func (indexer *Indexer) CatchPinsByTx(msgTx *wire.MsgTx, blockHeight int64, timestamp int64, blockHash string, merkleRoot string, txIndex int) (pinInscriptions []*pin.PinInscription) {
	//check OpReturn data
	haveOpReturn := false
	//chain := MicroVisionChain{}
	for i, out := range msgTx.TxOut {
		class, _, _, _ := txscript.ExtractPkScriptAddrs(out.PkScript, indexer.ChainParams)
		//fmt.Println(class.String())
		if class.String() == "nonstandard" {
			pinInscription := indexer.ParsePin(out.PkScript)
			if pinInscription == nil {
				continue
			}
			_, host, path := pin.ValidHostPath(pinInscription.Path)
			if common.CheckBlockedHost(host) {
				continue //blocked host
			}
			if !common.CheckHost(host) {
				continue //not in host list
			}
			//address, outIdx, locationIdx := indexer.GetPinOwner(msgTx, i-1)
			address, outIdx, locationIdx := indexer.GetPinOwner(msgTx, 0)
			//recalculate txhash
			txHash, err := GetNewHash(msgTx)
			if err != nil {
				continue
			}
			id := fmt.Sprintf("%si%d", txHash, outIdx)
			metaId := common.GetMetaIdByAddress(address)
			contentTypeDetect := common.DetectContentType(&pinInscription.ContentBody)
			pop := ""
			if merkleRoot != "" && blockHash != "" {
				pop, _ = common.GenPop(id, merkleRoot, blockHash)
			}
			popLv, _ := pin.PopLevelCount(indexer.ChainName, pop)
			creator := address
			if common.Config.Sync.IsFullNode {
				//creator = chain.GetCreatorAddress(msgTx.TxIn[0].PreviousOutPoint.Hash.String(), msgTx.TxIn[0].PreviousOutPoint.Index, indexer.ChainParams)
				// if v, ok := pin.AllCreatorAddress.Load(msgTx.TxIn[0].PreviousOutPoint.Hash.String()); ok {
				// 	creator = v.(string)
				// }
			}

			pinInscriptions = append(pinInscriptions, &pin.PinInscription{
				//Pin:                pinInscription,
				ChainName:          indexer.ChainName,
				Id:                 id,
				MetaId:             metaId,
				Number:             0,
				Address:            address,
				InitialOwner:       address,
				CreateAddress:      creator,
				CreateMetaId:       common.GetMetaIdByAddress(creator),
				Timestamp:          timestamp,
				GenesisHeight:      blockHeight,
				GenesisTransaction: txHash,
				Output:             fmt.Sprintf("%s:%d", txHash, outIdx),
				OutputValue:        msgTx.TxOut[outIdx].Value,
				TxInIndex:          uint32(i - 1),
				Offset:             uint64(outIdx),
				TxIndex:            txIndex,
				Operation:          pinInscription.Operation,
				Location:           fmt.Sprintf("%s:%d:%d", txHash, outIdx, locationIdx),
				Path:               strings.TrimSpace(path),
				OriginalPath:       strings.TrimSpace(pinInscription.Path),
				ParentPath:         strings.TrimSpace(pinInscription.ParentPath),
				Encryption:         pinInscription.Encryption,
				Version:            pinInscription.Version,
				ContentType:        pinInscription.ContentType,
				ContentTypeDetect:  contentTypeDetect,
				ContentBody:        pinInscription.ContentBody,
				ContentLength:      pinInscription.ContentLength,
				ContentSummary:     getContentSummary(pinInscription, id, contentTypeDetect),
				Pop:                pop,
				PopLv:              popLv,
				PoPScore:           pin.GetPoPScore(pop, int64(popLv), common.Config.Mvc.PopCutNum),
				PoPScoreV1:         pin.GetPoPScoreV1(pop, popLv),
				DataValue:          pin.RarityScoreBinary(indexer.ChainName, pop),
				Mrc20MintId:        []string{},
				Host:               host,
			})
			haveOpReturn = true
			break
		}
	}
	if !haveOpReturn {
		return nil
	}
	return
}
func getParentPath(path string) (parentPath string) {
	arr := strings.Split(path, "/")
	if len(arr) < 3 {
		return
	}
	parentPath = strings.Join(arr[0:len(arr)-1], "/")
	return
}
func getContentSummary(pinode *pin.PersonalInformationNode, id string, contentTypeDetect string) (content string) {
	if contentTypeDetect[0:4] != "text" {
		return fmt.Sprintf("/content/%s", id)
	} else {
		c := string(pinode.ContentBody)
		if len(c) > 150 {
			return c[0:150]
		} else {
			return string(pinode.ContentBody)
		}
	}
}
func (indexer *Indexer) GetPinOwner(tx *wire.MsgTx, inIdx int) (address string, outIdx int, locationIdx int64) {
	for i, out := range tx.TxOut {
		class, addresses, _, _ := txscript.ExtractPkScriptAddrs(out.PkScript, indexer.ChainParams)
		if class.String() != "nulldata" && class.String() != "nonstandard" && len(addresses) > 0 {
			address = addresses[0].String()
			outIdx = i
			locationIdx = 0
			break
		}
	}
	return
	// if len(tx.TxIn) == 1 || len(tx.TxOut) == 1 || inIdx == 0 {
	// 	_, addresses, _, _ := txscript.ExtractPkScriptAddrs(tx.TxOut[0].PkScript, indexer.ChainParams)
	// 	if len(addresses) > 0 {
	// 		address = addresses[0].String()
	// 	}
	// 	return
	// }
	// inputValue := int64(0)
	// for i, in := range tx.TxIn {
	// 	if i == inIdx {
	// 		break
	// 	}
	// 	value, err := GetValueByTx(in.PreviousOutPoint.Hash.String(), int(in.PreviousOutPoint.Index))
	// 	if err != nil {
	// 		return
	// 	}
	// 	inputValue += value
	// }
	// outputValue := int64(0)
	// for x, out := range tx.TxOut {
	// 	outputValue += out.Value
	// 	if outputValue > inputValue {
	// 		locationIdx = outputValue - inputValue
	// 		_, addresses, _, _ := txscript.ExtractPkScriptAddrs(out.PkScript, indexer.ChainParams)
	// 		if len(addresses) > 0 {
	// 			address = addresses[0].String()
	// 			outIdx = x
	// 		}
	// 		break
	// 	}
	// }
	// return
}
func (indexer *Indexer) ParsePins(pkScript []byte) (pins []*pin.PersonalInformationNode) {
	// Parse pins content from witness script
	tokenizer := txscript.MakeScriptTokenizer(0, pkScript)
	for tokenizer.Next() {
		// Check inscription envelop header: OP_FALSE(0x00), OP_IF(0x63), PROTOCOL_ID
		if tokenizer.Opcode() == txscript.OP_FALSE {
			if !tokenizer.Next() || tokenizer.Opcode() != txscript.OP_IF {
				return
			}
			if !tokenizer.Next() || hex.EncodeToString(tokenizer.Data()) != common.Config.ProtocolID {
				return
			}
			pinode := indexer.parseOnePin(&tokenizer)
			if pinode != nil {
				pins = append(pins, pinode)
			}
		}
	}
	return
}
func (indexer *Indexer) ParsePin(pkScript []byte) (pinode *pin.PersonalInformationNode) {
	// Parse pins content from witness script
	tokenizer := txscript.MakeScriptTokenizer(0, pkScript)
	for tokenizer.Next() {
		// Check inscription envelop header: OP_FALSE(0x00), OP_IF(0x63), PROTOCOL_ID
		if tokenizer.Opcode() == txscript.OP_RETURN {
			if !tokenizer.Next() || hex.EncodeToString(tokenizer.Data()) != common.Config.ProtocolID {
				return
			}
			pinode = indexer.parseOnePin(&tokenizer)
		}
	}
	return
}
func (indexer *Indexer) parseOnePin(tokenizer *txscript.ScriptTokenizer) *pin.PersonalInformationNode {
	// Find any pushed data in the script. This includes OP_0, but not OP_1 - OP_16.
	var infoList [][]byte
	for tokenizer.Next() {
		infoList = append(infoList, tokenizer.Data())
	}
	// Error occurred
	if err := tokenizer.Err(); err != nil {
		return nil
	}
	if len(infoList) < 1 {
		return nil
	}

	pinode := pin.PersonalInformationNode{}
	pinode.Operation = strings.ToLower(string(infoList[0]))
	if pinode.Operation == "init" {
		pinode.Path = "/"
		return &pinode
	}
	if len(infoList) < 6 && pinode.Operation != "revoke" {
		return nil
	}
	if pinode.Operation == "revoke" && len(infoList) < 5 {
		return nil
	}
	pinode.Path = strings.ToLower(string(infoList[1]))
	pinode.ParentPath = getParentPath(pinode.Path)
	encryption := "0"
	if infoList[2] != nil {
		encryption = string(infoList[2])
	}
	pinode.Encryption = encryption
	version := "0"
	if infoList[3] != nil {
		version = string(infoList[3])
	}
	pinode.Version = version
	contentType := "application/json"
	if infoList[4] != nil {
		contentType = strings.ToLower(string(infoList[4]))
	}
	pinode.ContentType = contentType
	var body []byte
	for i := 5; i < len(infoList); i++ {
		body = append(body, infoList[i]...)
	}
	pinode.ContentBody = body
	pinode.ContentLength = uint64(len(body))
	return &pinode
}
func (indexer *Indexer) GetBlockTxHash(blockHeight int64) (txhashList []string, pinIdList []string) {
	chain := MicroVisionChain{}
	blockMsg, err := chain.GetBlock(blockHeight)
	if err != nil {
		return
	}
	block := blockMsg.(*wire.MsgBlock)
	for _, tx := range block.Transactions {
		//recalculate txhash
		txHash, err := GetNewHash(tx)
		if err != nil {
			continue
		}
		for i := range tx.Copy().TxOut {
			var pinId strings.Builder
			pinId.WriteString(txHash)
			pinId.WriteString("i")
			pinId.WriteString(strconv.Itoa(i))
			pinIdList = append(pinIdList, pinId.String())
		}
		txhashList = append(txhashList, tx.TxHash().String())
	}
	return
}
func (indexer *Indexer) CatchNativeMrc20Transfer(blockHeight int64, utxoList []*mrc20.Mrc20Utxo, mrc20TransferPinTx map[string]struct{}) (savelist []*mrc20.Mrc20Utxo) {
	pointMap := make(map[string][]*mrc20.Mrc20Utxo)
	keyMap := make(map[string]*mrc20.Mrc20Utxo) //key point-tickid
	for _, u := range utxoList {
		if u.MrcOption == "deploy" {
			continue
		}
		pointMap[u.TxPoint] = append(pointMap[u.TxPoint], u)
	}
	block := indexer.Block.(*wire.MsgBlock)
	t := block.Header.Timestamp.Unix()
	for _, tx := range block.Transactions {
		//if have data transfer
		haveOpReturn := false
		for _, out := range tx.TxOut {
			class, _, _, _ := txscript.ExtractPkScriptAddrs(out.PkScript, indexer.ChainParams)
			if class.String() == "nulldata" || class.String() == "nonstandard" {
				haveOpReturn = true
				break
			}
		}
		if haveOpReturn {
			continue
		}
		for _, in := range tx.TxIn {
			id := fmt.Sprintf("%s:%d", in.PreviousOutPoint.Hash.String(), in.PreviousOutPoint.Index)
			if v, ok := pointMap[id]; ok {
				for _, utxo := range v {
					send := mrc20.Mrc20Utxo{TxPoint: id, Index: utxo.Index, Mrc20Id: utxo.Mrc20Id, Verify: true, Status: -1}
					savelist = append(savelist, &send)
					key := fmt.Sprintf("%s-%s", send.Mrc20Id, send.TxPoint)
					_, find := keyMap[key]
					if find {
						//keyMap[key].AmtChange += send.AmtChange
						keyMap[key].AmtChange = keyMap[key].AmtChange.Add(send.AmtChange)
					} else {
						recive := *utxo
						recive.MrcOption = "native-transfer"
						recive.ToAddress = indexer.GetAddress(tx.TxOut[0].PkScript)
						recive.BlockHeight = blockHeight
						recive.TxPoint = fmt.Sprintf("%s:%d", tx.TxHash().String(), 0)
						recive.Chain = "mvc"
						recive.Timestamp = t
						keyMap[key] = &recive
					}
				}
			}
		}
	}
	for _, u := range keyMap {
		savelist = append(savelist, u)
	}
	return
}
func (indexer *Indexer) CatchMempoolNativeMrc20Transfer(txList []interface{}, utxoList []*mrc20.Mrc20Utxo, mrc20TransferPinTx map[string]struct{}) (savelist []*mrc20.Mrc20Utxo) {
	//TODO
	return
}
