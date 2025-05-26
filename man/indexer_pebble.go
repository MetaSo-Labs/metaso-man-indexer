package man

import (
	"manindexer/common"
	"manindexer/pebblestore"
	"manindexer/pin"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bytedance/sonic"
)

type PebbleData struct {
	database *pebblestore.Database
}

func (pd *PebbleData) Init(shardNum int) (err error) {
	dbPath := filepath.Join("./man_base_data_pebble")
	err = os.MkdirAll(dbPath, 0755)
	if err != nil {
		return
	}
	pd.database, err = pebblestore.NewDataBase(dbPath, shardNum)
	return
}

func (pd *PebbleData) DoIndexerRun(chainName string, height int64, reIndex bool) (err error) {
	//bT := time.Now()
	if !reIndex {
		MaxHeight[chainName] = height
	}
	pinList, _, metaIdData,
		updatedData, mrc20List, txInList, mrc20TransferPinTx,
		followData, infoAdditional, _ := pd.GetSaveData(chainName, height)
	//pinList, protocolsData, metaIdData, pinTreeData, updatedData, _, followData, infoAdditional, _ := GetSaveData(chainName, height)
	//fmt.Println("PIN NUM:", len(pinList), "PROTOCOLS NUM:", len(protocolsData), "METAID NUM:", len(metaIdData), "PIN TREE NUM:", 0, "UPDATE NUM:", len(updatedData), "FOLLOW NUM:", len(followData), "INFO ADDITIONAL NUM:", len(infoAdditional))
	if len(metaIdData) > 0 {
		DbAdapter.BatchUpsertMetaIdInfo(metaIdData)
		if !reIndex {
			pd.database.CountAdd("metaids", int64(len(metaIdData)))
		}
		//metaIdData = metaIdData[0:0]
		metaIdData = nil
	}
	var pinNodeList []*pin.PinInscription
	if len(pinList) > 0 {
		//DbAdapter.BatchAddPins(pinList)
		// if err := batchProcessPins(pinList, DefaultBatchSize); err != nil {
		// 	return fmt.Errorf("failed to process pins: %v", err)
		// }
		pd.database.SetAllPins(height, pinList, 1000)
		//check transfer in this block
		var idList []string
		for _, item := range pinList {
			p := item.(*pin.PinInscription)
			idList = append(idList, p.Output)
			if p.Path == "/metaaccess/accesscontrol" || p.Path == "/metaaccess/accesspass" {
				pinNodeList = append(pinNodeList, p)
			}
		}
		if common.Config.Sync.IsFullNode {
			pd.handleTransfer(chainName, idList, height)
			idList = idList[:0]
		}
		if !reIndex {
			pd.database.CountAdd("pins", int64(len(pinList)))
			pd.database.CountAdd("blocks", int64(1))
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
		//DbAdapter.BatchUpdatePins(updatedData)
		pd.database.BatchUpdatePins(updatedData)
		updatedData = updatedData[:0]
	}
	if len(followData) > 0 {
		DbAdapter.BatchUpsertFollowData(followData)
		followData = followData[:0]
	}
	if len(infoAdditional) > 0 {
		DbAdapter.BatchUpsertMetaIdInfoAddition(infoAdditional)
		infoAdditional = infoAdditional[:0]
	}
	//Handle MRC20 last.
	if height >= Mrc20HeightLimit[chainName] && common.ModuleExist("mrc20") {
		Mrc20Handle(chainName, height, mrc20List, mrc20TransferPinTx, txInList, false)
		mrc20List = mrc20List[:0]
		mrc20TransferPinTx = make(map[string]struct{})
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
	pins, txInList = IndexerAdapter[chainName].CatchPins(blockHeight)
	//fmt.Println("PIN NUM:", len(pins), chainName, blockHeight)
	//check transfer
	if common.Config.Sync.IsFullNode {
		pd.handleTransfer(chainName, txInList, blockHeight)
		txInList = txInList[:0]
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
	transferCheck, err := pd.database.GetPinListByIdList(outputList, 1000, true)
	if err == nil && len(transferCheck) > 0 {
		idMap := make(map[string]string)
		for _, t := range transferCheck {
			idMap[t.Output] = t.Address
		}
		trasferMap := IndexerAdapter[chainName].CatchTransfer(idMap)
		pd.database.UpdateTransferPin(trasferMap)
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
	originalPins, err := pd.database.GetPinListByIdList(modifyPinIdList, 1000, false)
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
	result, err := pd.database.GetPinByKey(pinid)
	if err != nil {
		return
	}
	err = sonic.Unmarshal(result, &pinNode)
	return
}
