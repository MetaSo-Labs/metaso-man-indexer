package man

import (
	"fmt"
	"log"

	"manindexer/common"
	"manindexer/database/mongodb"
	"manindexer/pebblestore"

	"manindexer/pin"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bytedance/sonic"
)

type PebbleData struct {
	Database *pebblestore.Database
}

func (pd *PebbleData) Init(shardNum int) (err error) {
	dbPath := filepath.Join("./man_base_data_pebble")
	err = os.MkdirAll(dbPath, 0755)
	if err != nil {
		return
	}
	pd.Database, err = pebblestore.NewDataBase(dbPath, shardNum)
	return
}

func (pd *PebbleData) DoIndexerRun(chainName string, height int64, reIndex bool) (err error) {
	go SaveBlockFileFromChain(chainName, height)
	//bT := time.Now()
	if !reIndex {
		MaxHeight[chainName] = height
	}
	log.Println("GetSaveData===>")
	startTime := time.Now()
	pinList, _, metaIdData,
		updatedData, mrc20List, txInList, mrc20TransferPinTx,
		followData, infoAdditional, _ := pd.GetSaveData(chainName, height)
	log.Println("GetSaveData:", time.Since(startTime), "Num:", len(pinList))
	//pinList, protocolsData, metaIdData, pinTreeData, updatedData, _, followData, infoAdditional, _ := GetSaveData(chainName, height)
	//fmt.Println("PIN NUM:", len(pinList), "PROTOCOLS NUM:", len(protocolsData), "METAID NUM:", len(metaIdData), "PIN TREE NUM:", 0, "UPDATE NUM:", len(updatedData), "FOLLOW NUM:", len(followData), "INFO ADDITIONAL NUM:", len(infoAdditional))
	startTime = time.Now()
	if len(metaIdData) > 0 {
		DbAdapter.BatchUpsertMetaIdInfo(metaIdData)
		if !reIndex {
			cnt, err := mongodb.CountMetaid()
			if err == nil {
				pd.Database.CountSet("metaids", cnt)
			}
		}
		//metaIdData = metaIdData[0:0]
		metaIdData = nil
	}
	log.Println("BatchUpsertMetaIdInfo:", time.Since(startTime))
	var pinNodeList []*pin.PinInscription
	if len(pinList) > 0 {
		//DbAdapter.BatchAddPins(pinList)
		// if err := batchProcessPins(pinList, DefaultBatchSize); err != nil {
		// 	return fmt.Errorf("failed to process pins: %v", err)
		// }
		startTime = time.Now()
		pd.Database.SetAllPins(height, pinList, 20000)
		log.Println("SetAllPins:", time.Since(startTime))
		//check transfer in this block
		//var idList []string
		tmp := pinList[0].(*pin.PinInscription)
		//blockKey := fmt.Sprintf("blocktime_mvc_%d", height)
		blockKey := fmt.Sprintf("blocktime_%s_%d", chainName, height)
		pd.Database.CountSet(blockKey, tmp.Timestamp)
		for _, item := range pinList {
			p := item.(*pin.PinInscription)
			//idList = append(idList, p.Output)
			if p.Path == "/metaaccess/accesscontrol" || p.Path == "/metaaccess/accesspass" {
				pinNodeList = append(pinNodeList, p)
			}
		}
		//先不检查转移
		//startTime = time.Now()
		// if common.Config.Sync.IsFullNode {
		// 	pd.handleTransfer(chainName, idList, height)
		// }
		//idList = idList[:0]
		//log.Println("handleTransfer:", time.Since(startTime))
		if !reIndex {
			num := int64(len(pinList))
			log.Println("Height:", height, "Pin:", num)
			pd.Database.CountAdd("pins", num)
			pd.Database.CountAdd("blocks", int64(1))
		}
	}
	pinList = pinList[:0]
	// if len(pinTreeData) > 0 {
	// 	DbAdapter.BatchAddPinTree(pinTreeData)
	// }
	// if len(protocolsData) > 0 {
	// 	//DbAdapter.BatchAddProtocolData(protocolsData)
	// 	if err := batchProcessProtocolsData(protocolsData, DefaultBatchSize); err != nil {
	// 		return fmt.Errorf("failed to process protocols data: %v", err)
	// 	}
	// }
	// protocolsData = protocolsData[:0]
	if len(updatedData) > 0 {
		startTime = time.Now()
		//DbAdapter.BatchUpdatePins(updatedData)
		pd.Database.BatchUpdatePins(updatedData)
		updatedData = updatedData[:0]
		log.Println("BatchUpdatePins:", time.Since(startTime))
	}
	if len(followData) > 0 {
		startTime = time.Now()
		DbAdapter.BatchUpsertFollowData(followData)
		followData = followData[:0]
		log.Println("BatchUpsertFollowData:", time.Since(startTime))
	}
	if len(infoAdditional) > 0 {
		startTime = time.Now()
		DbAdapter.BatchUpsertMetaIdInfoAddition(infoAdditional)
		infoAdditional = infoAdditional[:0]
		log.Println("BatchUpsertMetaIdInfoAddition:", time.Since(startTime))
	}
	//Handle MRC20 last.
	if height >= Mrc20HeightLimit[chainName] && common.ModuleExist("mrc20") {
		startTime = time.Now()
		Mrc20Handle(chainName, height, mrc20List, mrc20TransferPinTx, txInList, false)
		mrc20List = mrc20List[:0]
		mrc20TransferPinTx = make(map[string]struct{})
		log.Println("Mrc20Handle:", time.Since(startTime))
	}
	// if len(pinNodeList) > 0 && height >= Mrc20HeightLimit[chainName] {
	// 	m721 := Mrc721{}
	// 	m721.PinHandle(pinNodeList)
	// }
	//Handle MetaAccess
	if len(pinNodeList) > 0 {
		access := MetaAccess{}
		access.PinHandle(pinNodeList, false)
		pinNodeList = pinNodeList[:0]
	}
	//}
	//bar.Finish()
	if FirstCompleted {
		DeleteMempoolData(height, chainName)
	}
	//eT := time.Since(bT)
	//fmt.Println("Blok(", height, "),PIN NUM:", len(pinList), ",Run time: ", eT)
	return
}

// Set PinId from block data
func (pd *PebbleData) SetPinIdList(chainName string, height int64) (err error) {
	pins, _, _ := IndexerAdapter[chainName].CatchPins(height)
	var pinIdList []string
	if len(pins) <= 0 {
		return
	}
	for _, pinNode := range pins {
		pinIdList = append(pinIdList, pinNode.Id)
	}
	blockTime := pins[0].Timestamp
	publicKeyStr := common.ConcatBytesOptimized([]string{fmt.Sprintf("%010d", blockTime), "&", chainName, "&", fmt.Sprintf("%010d", height)}, "")
	pd.Database.InsertBlockTxs(publicKeyStr, strings.Join(pinIdList, ","))
	pinIdList = nil
	fmt.Println(">> SetPinIdList done for height:", chainName, height)
	return
}
func (pd *PebbleData) GetSaveData(chainName string, blockHeight int64) (
	pinList []interface{},
	protocolsData []*pin.PinInscription,
	metaIdData map[string]*pin.MetaIdInfo,
	updatedData []*pin.PinInscription,
	mrc20List []*pin.PinInscription,
	txInList []string,
	mrc20TransferPinTx map[string]struct{},
	followData []*pin.FollowData,
	infoAdditional []*pin.MetaIdInfoAdditional,
	err error) {
	metaIdData = make(map[string]*pin.MetaIdInfo)
	var pins []*pin.PinInscription
	st := time.Now()
	//var creatorMap map[string]string
	pins, txInList, _ = IndexerAdapter[chainName].CatchPins(blockHeight)
	log.Println("CatchPins time:", time.Since(st), "PIN NUM:", len(pins), chainName, blockHeight)
	//check transfer
	if common.Config.Sync.IsFullNode {
		//pd.Database.BatchInsertCreator(creatorMap, &pin.AllCreatorAddress)
		// 先不检查转移
		//pd.handleTransfer(chainName, txInList, blockHeight)
	}

	//pin validator
	mrc20TransferPinTx = make(map[string]struct{})
	for _, pinNode := range pins {
		err := ManValidator(pinNode)
		if err != nil {
			continue
		}
		//save all data or protocols data
		//=============Temporary comment, performance optimization.=========
		// s := handleProtocolsData(pinNode)
		// if s == -1 {
		// 	continue
		// } else if s == 1 {
		// 	protocolsData = append(protocolsData, pinNode)
		// }
		//==================================================================
		pinList = append(pinList, pinNode)
		//mrc20 pin
		if len(pinNode.Path) > 10 && pinNode.Path[0:10] == "/ft/mrc20/" {
			mrc20List = append(mrc20List, pinNode)
			if pinNode.Path == "/ft/mrc20/transfer" {
				mrc20TransferPinTx[pinNode.GenesisTransaction] = struct{}{}
			}
		}
	}
	//check mrc20 transfer
	// mrc20transferCheck, err := DbAdapter.GetMrc20UtxoByOutPutList(txInList, false)
	// if err == nil && len(mrc20transferCheck) > 0 {
	// 	mrc20TrasferList := IndexerAdapter[chainName].CatchNativeMrc20Transfer(blockHeight, mrc20transferCheck, mrc20TransferPinTx)
	// 	if len(mrc20TrasferList) > 0 {
	// 		DbAdapter.UpdateMrc20Utxo(mrc20TrasferList, false)
	// 	}
	// }
	pd.handlePathAndOperation(&pinList, &metaIdData, &updatedData, &followData, &infoAdditional)
	//pd.createPinNumber(&pinList)
	//pd.createMetaIdNumber(metaIdData)
	return
}
func (pd *PebbleData) handleTransfer(chainName string, outputList []string, blockHeight int64) {
	defer func() {
		outputList = outputList[:0]
	}()
	transferCheck, err := pd.Database.GetPinListByIdList(outputList, 1000, true)
	if err == nil && len(transferCheck) > 0 {
		idMap := make(map[string]string)
		for _, t := range transferCheck {
			idMap[t.Output] = t.Address
		}
		trasferMap := IndexerAdapter[chainName].CatchTransfer(idMap)
		pd.Database.UpdateTransferPin(trasferMap)
		var transferHistoryList []*pin.PinTransferHistory
		tranferTime := time.Now().Unix()
		for pinid, info := range trasferMap {
			transferHistoryList = append(transferHistoryList, &pin.PinTransferHistory{
				PinId:          strings.ReplaceAll(pinid, ":", "i"),
				TransferTime:   tranferTime,
				TransferHeight: blockHeight,
				TransferTx:     info.Location,
				ChainName:      chainName,
				FromAddress:    info.FromAddress,
				ToAddress:      info.Address,
			})
		}
		DbAdapter.AddTransferHistory(transferHistoryList)
		idMap = nil
		trasferMap = nil
		transferHistoryList = transferHistoryList[:0]
	}
}

func (pd *PebbleData) handlePathAndOperation(
	pinList *[]interface{},
	metaIdData *map[string]*pin.MetaIdInfo,
	updatedData *[]*pin.PinInscription,
	followData *[]*pin.FollowData,
	infoAdditional *[]*pin.MetaIdInfoAdditional) {
	var modifyPinIdList []string
	newPinMap := make(map[string]*pin.PinInscription)
	originalPinMap := make(map[string]*pin.PinInscription)

	defer func() {
		if newPinMap != nil {
			newPinMap = nil
		}
		if originalPinMap != nil {
			originalPinMap = nil
		}
	}()

	for _, p := range *pinList {
		pinNode := p.(*pin.PinInscription)
		if pinNode.MetaId == "" {
			pinNode.MetaId = common.GetMetaIdByAddress(pinNode.Address)
		}
		metaIdInfoParse(pinNode, "", metaIdData)
		switch pinNode.Operation {
		case "modify":
			updatePin := *pinNode
			updatePin.Status = 1
			updatePin.OriginalId = strings.Replace(pinNode.Path, "@", "", -1)
			modifyPinIdList = append(modifyPinIdList, updatePin.OriginalId)
			pinNode.OriginalId = updatePin.OriginalId
			newPinMap[updatePin.Id] = &updatePin
		case "revoke":
			updatePin := *pinNode
			updatePin.Status = -1
			updatePin.OriginalId = strings.Replace(pinNode.Path, "@", "", -1)
			modifyPinIdList = append(modifyPinIdList, updatePin.OriginalId)
			pinNode.OriginalId = updatePin.OriginalId
			newPinMap[updatePin.Id] = &updatePin
		}

		path := pinNode.Path
		// if len(path) > 5 && path[0:5] == "/info" {
		// 	metaIdInfo := metaIdInfoParse(pinNode, "")
		// 	*metaIdData = append(*metaIdData, metaIdInfo)
		// }
		pathArray := strings.Split(path, "/")
		if len(pathArray) > 1 && path != "/" {
			path = strings.Join(pathArray[0:len(pathArray)-1], "/")
		}
		//pinTree := pin.PinTreeCatalog{RootTxId: common.GetMetaIdByAddress(pinNode.Address), TreePath: path}
		//*pinTreeData = append(*pinTreeData, pinTree)
		//follow
		if pinNode.Path == "/follow" {
			*followData = append(*followData, creatFollowData(pinNode, true))
		}
		//infoAdditional
		additional := createInfoAdditional(pinNode, pinNode.Path)
		if additional != (pin.MetaIdInfoAdditional{}) {
			*infoAdditional = append(*infoAdditional, &additional)
		}
	}
	if len(modifyPinIdList) <= 0 {
		return
	}
	originalPins, err := pd.Database.GetPinListByIdList(modifyPinIdList, 1000, false)
	if err != nil {
		return
	}
	for _, mp := range originalPins {
		originalPinMap[mp.Id] = mp
	}
	statusMap := getModifyPinStatus(newPinMap, originalPinMap)
	for _, p := range *pinList {
		pinNode := p.(*pin.PinInscription)
		if pinNode.OriginalId == "" {
			pinNode.OriginalId = pinNode.Id
		}
		if pinNode.Operation == "modify" || pinNode.Operation == "revoke" {
			if v, ok := statusMap[pinNode.Id]; ok {
				pinNode.Status = v
			}
			if pinNode.Status >= 0 {
				*updatedData = append(*updatedData, newPinMap[pinNode.Id])
			}
			_, check := originalPinMap[pinNode.OriginalId]
			if check {
				pinNode.OriginalPath = originalPinMap[pinNode.OriginalId].OriginalPath
			}
			if pinNode.Operation == "modify" && pinNode.Status >= 0 && check {
				if len(originalPinMap[pinNode.OriginalId].OriginalPath) > 5 && originalPinMap[pinNode.OriginalId].OriginalPath[0:5] == "/info" {
					metaIdInfoParse(pinNode, originalPinMap[pinNode.OriginalId].OriginalPath, metaIdData)
				}
			}
			//unfollow
			if pinNode.Operation == "revoke" {
				isUnfollow := false
				if pinNode.OriginalPath == "/follow" {
					isUnfollow = true
				}
				arr := strings.Split(pinNode.OriginalPath, ":")
				if len(arr) == 2 {
					if arr[1] == "/follow" {
						isUnfollow = true
					}
				}
				if isUnfollow {
					*followData = append(*followData, creatFollowData(pinNode, false))
				}
			}
			//infoAdditional
			if pinNode.Operation == "modify" {
				additional := createInfoAdditional(pinNode, pinNode.OriginalPath)
				if additional != (pin.MetaIdInfoAdditional{}) {
					*infoAdditional = append(*infoAdditional, &additional)
				}
			}

		} else {
			metaIdInfoParse(pinNode, "", metaIdData)
		}
	}
}
func (pd *PebbleData) GetPinById(pinid string) (pinNode pin.PinInscription, err error) {
	result, err := pd.Database.GetPinByKey(pinid)
	if err != nil {
		return
	}
	err = sonic.Unmarshal(result, &pinNode)
	return
}
