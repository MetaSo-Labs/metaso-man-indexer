package service

import (
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"manindexer/basicprotocols/group_chat/api/respond"
	"manindexer/basicprotocols/group_chat/models"
	"manindexer/common"
	"strconv"
	"strings"
	"time"

	"github.com/bitcoinsv/bsvd/chaincfg"
	chaincfg2 "github.com/bitcoinsv/bsvd/chaincfg"
	"github.com/bitcoinsv/bsvd/wire"
	"github.com/libsv/go-bk/bec"
	"github.com/tyler-smith/go-bip32"
)

// GetLuckyBagWithOpenList 根据groupId和pinId获取红包对象和已领取列表
func GetLuckyBagWithOpenList(groupId, pinId string) (*respond.LuckyBagInfoResponse, error) {
	// 获取红包对象
	luckyBag, err := chatDB.GetLuckyBagByPinId(pinId)
	if err != nil {
		return nil, err
	}

	if luckyBag == nil {
		return nil, errors.New("lucky bag not found")
	}

	// 验证groupId是否匹配
	if luckyBag.GroupId != groupId {
		return nil, errors.New("lucky bag not match")
	}

	// 获取已领取的抢红包列表
	openList, err := chatDB.GetOpenLuckyBagList(pinId)
	if err != nil {
		return nil, err
	}

	// 构建LuckyBagInfoResponse
	response := &respond.LuckyBagInfoResponse{
		TxId:                luckyBag.TxId,
		MetaId:              luckyBag.MetaId,
		UserInfo:            nil, // 需要从用户信息中获取
		SubId:               luckyBag.SubId,
		Code:                luckyBag.Code,
		CreateTime:          luckyBag.CreateTimeStr,
		Content:             luckyBag.Content,
		Img:                 luckyBag.Img,
		ImgType:             luckyBag.ImgType,
		Amount:              luckyBag.Amount,
		Count:               luckyBag.Count,
		UsedCount:           "0",
		PayList:             make([]*respond.InfoPayList, 0),
		Type:                luckyBag.Type,
		TokenCount:          0, // TalkGroupLuckyBagV3 没有 TokenCount 字段
		RequireType:         luckyBag.RequireType,
		RequireTickId:       luckyBag.RequireTickId,
		RequireCollectionId: luckyBag.RequireCollectionId,
		LimitAmount:         luckyBag.LimitAmount,
	}

	usedCount := 0
	// 转换PayList - 注意ProInfoPayList字段较少，需要填充默认值
	for _, payItem := range luckyBag.PayList {
		infoPayList := &respond.InfoPayList{
			TxId:         luckyBag.TxId,
			Index:        payItem.Index,
			Amount:       payItem.Amount,
			Address:      payItem.Address,
			Used:         false,
			GradTxId:     "",
			GradPinId:    "",
			GradMetaId:   "",
			GradAddress:  "",
			UserInfo:     nil,
			Timestamp:    0,
			ScriptPubKey: "",
			IsBest:       false,
			IsWithdraw:   false,
		}

		if openList != nil {
			for _, openItem := range openList.Items {
				if openItem.LuckyBagOutIndex == payItem.Index {
					infoPayList.Used = true
					infoPayList.GradTxId = openItem.OpenPinId[:len(openItem.OpenPinId)-2]
					infoPayList.GradPinId = openItem.OpenPinId
					infoPayList.GradMetaId = openItem.CreateMetaId
					infoPayList.GradAddress = openItem.CreateAddress
					infoPayList.IsWithdraw = true
					usedCount++
				}
			}
		}
		response.PayList = append(response.PayList, infoPayList)
	}
	response.UsedCount = strconv.Itoa(usedCount)

	return response, nil
}

// 临时的OpenRedMetaId结构，用于匹配参考代码
type OpenRedMetaId struct {
	MetaId             string
	ProOpenRedenvelope *ProOpenRedenvelope
	Vins               []*models.TxIn
	Timestamp          int64
}

type ProOpenRedenvelope struct {
	SubId      string
	Code       string
	CreateTime string
	Used       *ProUsed
}

type ProUsed struct {
	Amount  string
	Address string
	Index   int64
}

func GrabLuckyBag(groupId, pinId, metaId, address string) (string, error) {
	luckyBag, err := chatDB.GetLuckyBagByPinId(pinId)
	if err != nil {
		return "", err
	}
	if luckyBag == nil {
		return "", errors.New("lucky bag not found")
	}

	// 验证groupId是否匹配
	if luckyBag.GroupId != groupId {
		return "", errors.New("lucky bag not match")
	}

	// 获取已领取的抢红包列表
	openList, err := chatDB.GetOpenLuckyBagList(pinId)
	if err != nil {
		return "", err
	}

	// 构建已抢红包列表
	openRedList := make([]*OpenRedMetaId, 0)
	for _, v := range openList.Items {
		// 获取抢红包详细信息
		openLuckyBag, err := chatDB.GetOpenLuckyBagByPinId(v.OpenPinId)
		if err != nil || openLuckyBag == nil {
			continue
		}

		openMetaId := &OpenRedMetaId{
			MetaId: openLuckyBag.MetaId,
			ProOpenRedenvelope: &ProOpenRedenvelope{
				SubId:      openLuckyBag.SubId,
				Code:       openLuckyBag.Code,
				CreateTime: openLuckyBag.CreateTimeStr,
				Used: &ProUsed{
					Amount:  openLuckyBag.Amount,
					Address: openLuckyBag.Address,
					Index:   openLuckyBag.Index,
				},
			},
			Vins:      openLuckyBag.Vins,
			Timestamp: openLuckyBag.Timestamp,
		}
		openRedList = append(openRedList, openMetaId)

		// 检查当前用户是否已经抢过
		if openLuckyBag.Address == address || openLuckyBag.MetaId == metaId {
			return "", errors.New("already grab")
		}
	}

	// 获取回收红包列表
	residueRedEnvelopeList, err := chatDB.GetResidueLuckyBagList(pinId)
	if err != nil {
		return "", err
	}

	if residueRedEnvelopeList != nil && len(residueRedEnvelopeList.Items) != 0 {
		for _, residueItem := range residueRedEnvelopeList.Items {
			// 获取回收红包详细信息
			residueLuckyBag, err := chatDB.GetResidueLuckyBagByLuckyBagPinId(residueItem.ResiduePinId)
			if err != nil || residueLuckyBag == nil {
				continue
			}

			if residueLuckyBag.UsedList == nil || len(residueLuckyBag.UsedList) == 0 {
				continue
			}
			for _, v := range residueLuckyBag.UsedList {
				openMetaId := &OpenRedMetaId{
					MetaId: residueLuckyBag.MetaId,
					ProOpenRedenvelope: &ProOpenRedenvelope{
						SubId:      residueLuckyBag.SubId,
						Code:       residueLuckyBag.Code,
						CreateTime: residueLuckyBag.CreateTimeStr,
						Used: &ProUsed{
							Amount:  v.Amount,
							Address: v.Address,
							Index:   v.Index,
						},
					},
					Vins:      residueLuckyBag.Vins,
					Timestamp: residueLuckyBag.Timestamp,
				}

				openRedList = append(openRedList, openMetaId)
			}
		}
	}

	// 获取未抢的红包列表
	unusedList := make([]*respond.UnusedList, 0)
	for _, v := range luckyBag.PayList {
		used := false
		for _, openV := range openRedList {
			if openV.Vins == nil {
				continue
			}
			vins := openV.Vins
			isVaildRedOpen := false
			for _, in := range vins {
				if in.Index == uint64(v.Index) && in.OutTxID == luckyBag.TxId {
					isVaildRedOpen = true
				}
			}
			if !isVaildRedOpen {
				continue
			}

			if openV.ProOpenRedenvelope != nil && openV.ProOpenRedenvelope.Used != nil && openV.ProOpenRedenvelope.Used.Index == v.Index {
				used = true
			}
		}
		if used {
			continue
		}

		unused := &respond.UnusedList{
			Index:   v.Index,
			Amount:  v.Amount,
			Address: v.Address,
		}
		unusedList = append(unusedList, unused)
	}

	if len(unusedList) <= 0 {
		return "", errors.New("LuckyBag had been all grab.")
	}

	err = commonGrab(luckyBag, unusedList, metaId, address)
	if err != nil {
		return "", err
	}

	return "success", nil
}

func commonGrab(luckyBag *models.TalkGroupLuckyBagV3, unusedList []*respond.UnusedList, metaId, address string) error {
	type grabEntity struct {
		unusedIndex   int64
		unusedAmount  string
		unusedAddress string
		tokenIndex    string
	}
	grabEntityList := make([]*grabEntity, 0)
	has := false

	for _, unused := range unusedList {
		//redis
		// usedMetaId, _ := redis.GetRedisGiftInfo(giftEntity.GroupId, giftEntity.TxId, unused.Index)
		// if usedMetaId != "" {
		// 	if usedMetaId != metaId {
		// 		continue
		// 	}else {
		// 		has = true
		// 		grabEntityList = append(grabEntityList, &grabEntity{
		// 			unusedIndex:   unused.Index,
		// 			unusedAmount:  unused.Amount,
		// 			unusedAddress: unused.Address,
		// 			tokenIndex:    "",
		// 		})
		// 		break
		// 	}
		// }else {
		// 	_, err := redis.SetRedisGiftInfo(giftEntity.GroupId, giftEntity.TxId, metaId, unused.Index)
		// 	if err != nil {
		// 		major.Println(fmt.Sprintf("[REDIS] Set userinfo err:%s", err.Error()))
		// 		continue
		// 	}else {
		// 		has = true
		// 		grabEntityList = append(grabEntityList, &grabEntity{
		// 			unusedIndex:   unused.Index,
		// 			unusedAmount:  unused.Amount,
		// 			unusedAddress: unused.Address,
		// 			tokenIndex:    "",
		// 		})
		// 		break
		// 	}
		// }
		has = true
		grabEntityList = append(grabEntityList, &grabEntity{
			unusedIndex:   unused.Index,
			unusedAmount:  unused.Amount,
			unusedAddress: unused.Address,
			tokenIndex:    "",
		})
		break // 只抢一个红包
	}

	if !has {
		return errors.New("LuckyBag had been all grab.")
	}

	hasSuccess := false
	for _, v := range grabEntityList {
		vins := make([]*models.TxIn, 0)
		vins = append(vins, &models.TxIn{
			OutTxID: luckyBag.TxId,
			Index:   uint64(v.unusedIndex),
		})

		pkScript := ""
		for _, vout := range luckyBag.LuckyBagVouts {
			if vout.Index == v.unusedIndex {
				pkScript = vout.ScriptPubKey
			}
		}

		// 生成唯一的TxId和PinId
		txId := fmt.Sprintf("%s_%d_%s_%s", luckyBag.TxId, v.unusedIndex, v.unusedAddress, metaId)
		pinId := fmt.Sprintf("%s_%d_%s_%s_pin", luckyBag.TxId, v.unusedIndex, v.unusedAddress, metaId)

		// 检查是否已经保存过该 PinId
		existingOpen, err := chatDB.GetOpenLuckyBagByPinId(pinId)
		if err == nil && existingOpen != nil {
			// 已经存在，跳过处理
			return errors.New("already grab")
		}

		// 创建抢红包记录
		openLuckyBag := &models.TalkGroupOpenLuckyBagV3{
			CommunityId:         "", // 需要从群组信息中获取
			GroupId:             luckyBag.GroupId,
			TxId:                txId,
			PinId:               pinId,
			MetaId:              metaId,
			Protocol:            luckyBag.Protocol,
			SubId:               luckyBag.SubId,
			Code:                luckyBag.Code,
			CreateTimeStr:       luckyBag.CreateTimeStr,
			Address:             address,
			Index:               v.unusedIndex,
			Amount:              v.unusedAmount,
			PkScript:            pkScript,
			Vins:                vins,
			Type:                luckyBag.Type,
			RequireTickId:       luckyBag.RequireTickId,
			RequireCollectionId: luckyBag.RequireCollectionId,
			LuckyBagTxId:        luckyBag.TxId,
			LuckyBagPinId:       luckyBag.PinId,
			LuckyBagMetaId:      luckyBag.MetaId,
			IsWithdraw:          false,
			Timestamp:           time.Now().Unix(),
			BlockHeight:         0, // 需要从实际交易中获取
			Chain:               luckyBag.Chain,
			GrabState:           models.GrabStateOpen,
			GrabTxId:            "",
			GrabMsg:             "",
		}

		// 保存抢红包记录到 TalkGroupOpenLuckyBagPinCollection
		err = chatDB.SaveOpenLuckyBag(openLuckyBag)
		if err != nil {
			log.Printf("SaveOpenLuckyBag err: %v", err)
			continue
		}

		// 保存抢红包列表记录到 TalkGroupOpenLuckyBagListCollection
		err = chatDB.SaveOpenLuckyBagList(luckyBag.PinId, openLuckyBag.PinId, luckyBag.GroupId, openLuckyBag.Timestamp, metaId, address, v.unusedIndex)
		if err != nil {
			log.Printf("SaveOpenLuckyBagList err: %v", err)
			continue
		}

		// 将抢红包记录加入队列，等待处理
		err = chatDB.EnqueueOpenLuckyBagMessage(openLuckyBag)
		if err != nil {
			log.Printf("EnqueueOpenLuckyBagMessage err: %v", err)
			continue
		}

		// 创建聊天消息模型（用于群聊显示）
		chat := &models.TalkGroupChatV3{
			GroupId:     luckyBag.GroupId,
			TxId:        openLuckyBag.PinId[:len(openLuckyBag.PinId)-2],
			PinId:       openLuckyBag.PinId,
			MetaId:      openLuckyBag.MetaId,
			Address:     openLuckyBag.Address,
			Protocol:    openLuckyBag.Protocol,
			Content:     "[Grab LuckyBag]", // 可以根据实际金额显示
			ContentType: "text/plain",
			Encryption:  "",
			ChatType:    models.ChatTypeOpenLuckyBag, // 抢红包类型
			InsideIndex: models.ChatInsideIndexIn,    // 默认为进入状态
			ReplyPin:    luckyBag.PinId,
			Timestamp:   openLuckyBag.Timestamp,
			Chain:       openLuckyBag.Chain,
			BlockHeight: openLuckyBag.BlockHeight,
		}

		// 保存聊天消息到 TalkGroupChatPinCollection
		err = chatDB.SaveChat(chat)
		if err != nil {
			return err
		}

		// 保存时间戳索引（根据用户状态决定保存到哪个集合）
		err = chatDB.SaveChatTimestampWithState(chat)
		if err != nil {
			return err
		}

		// 将消息加入队列，异步更新群列表
		err = chatDB.EnqueueChatMessage(chat)
		if err != nil {
			return err
		}

		hasSuccess = true
		break // 只处理一个红包
	}

	if !hasSuccess {
		return errors.New("Grab err.")
	}

	return nil
}

// 处理抢红包队列中的记录
func ProcessOpenLuckyBagQueue() {
	// 获取待处理的抢红包消息
	messages, err := chatDB.GetPendingOpenLuckyBagMessages(10) // 每次处理10条
	if err != nil {
		log.Printf("GetPendingOpenLuckyBagMessages err: %v", err)
		return
	}

	for _, message := range messages {
		// 处理抢红包记录
		err := disposingGrabLuckyBag(message.OpenLuckyBag)
		if err != nil {
			log.Printf("disposingGrabLuckyBag err: %v", err)
			continue
		}

		// 处理成功，删除队列消息
		err = chatDB.DeleteOpenLuckyBagQueueMessage(message.PinId)
		if err != nil {
			log.Printf("deleteOpenLuckyBagQueueMessage err: %v", err)
		}
	}
}

// 处理抢红包逻辑
func disposingGrabLuckyBag(grabEntity *models.TalkGroupOpenLuckyBagV3) error {
	if chainAdapter == nil || chainAdapter[grabEntity.Chain] == nil {
		return fmt.Errorf("chain adapter not found")
	}

	if grabEntity.GrabState != models.GrabStateOpen {
		return nil
	}
	if grabEntity.Vins == nil || len(grabEntity.Vins) == 0 {
		return errors.New("no vins found")
	}

	_ = grabEntity.Vins[0] // utxo, 暂时未使用
	wifStr, hexStr := makeGiftKey(grabEntity.SubId, grabEntity.Code, grabEntity.CreateTimeStr)
	if wifStr == "" || hexStr == "" {
		return errors.New("failed to generate wif or hex")
	}

	value, err := strconv.ParseUint(grabEntity.Amount, 10, 64)
	if err != nil {
		return fmt.Errorf("failed to parse amount: %v", err)
	}

	toAddress := grabEntity.Address
	_ = toAddress

	input := common.TxInputUtxo{
		TxId:     grabEntity.LuckyBagTxId,
		TxIndex:  int64(grabEntity.Index),
		PkScript: grabEntity.PkScript,
		Amount:   value,
		PriHex:   hexStr,
	}
	output := common.TxOutput{
		Address: toAddress,
		Amount:  int64(value),
	}

	netParam := chainAdapter[grabEntity.Chain].GetNetParam()

	// 根据链类型进行类型转换
	var tx interface{}
	var buildErr error

	switch strings.ToLower(grabEntity.Chain) {
	case "mvc":
		// MVC链使用 chaincfg2.Params
		if mvcNetParam, ok := netParam.(*chaincfg2.Params); ok {
			tx, buildErr = common.BuildMvcTransferAllTx(mvcNetParam, []*common.TxInputUtxo{&input}, &output, 1, false)
		} else {
			return fmt.Errorf("failed to convert netParam to chaincfg2.Params for MVC chain")
		}
	case "btc":
		// BTC链使用 chaincfg.Params
		if btcNetParam, ok := netParam.(*chaincfg.Params); ok {
			tx, buildErr = common.BuildMvcTransferAllTx(btcNetParam, []*common.TxInputUtxo{&input}, &output, 1, false)
		} else {
			return fmt.Errorf("failed to convert netParam to chaincfg.Params for BTC chain")
		}
	default:
		return fmt.Errorf("unsupported chain type: %s", grabEntity.Chain)
	}
	if buildErr != nil {
		return fmt.Errorf("failed to build tx: %v", buildErr)
	}

	// 类型断言，确保tx是正确的类型
	msgTx, ok := tx.(*wire.MsgTx)
	if !ok {
		return fmt.Errorf("failed to convert tx to *wire.MsgTx")
	}

	txRaw, err := common.MvcToRaw(msgTx)
	if err != nil {
		return fmt.Errorf("failed to convert tx to raw: %v", err)
	}

	resultTxId, err := chainAdapter[grabEntity.Chain].BroadcastTx(txRaw)
	if resultTxId != "" {
		grabEntity.GrabState = models.GrabStateOpenAndSend
		grabEntity.GrabTxId = resultTxId
		grabEntity.GrabMsg = "success"
		log.Printf("Success: %s", resultTxId)
	} else {
		log.Printf("Failure: %s", err.Error())
		grabEntity.GrabState = models.GrabStateOpenAndSendErr
		grabEntity.GrabMsg = err.Error()
	}

	// 更新数据库中的抢红包记录
	err = chatDB.SaveOpenLuckyBag(grabEntity)
	if err != nil {
		return fmt.Errorf("failed to save open lucky bag: %v", err)
	}

	return nil
}

// 启动抢红包队列处理器
func StartOpenLuckyBagQueueProcessor() {
	go func() {
		ticker := time.NewTicker(10 * time.Second) // 每10秒处理一次
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				// 处理抢红包队列
				ProcessOpenLuckyBagQueue()
			}
		}
	}()
}

func makeGiftKey(subId, code, createTimeStr string) (string, string) {
	key := fmt.Sprintf("%s%s%s", strings.ToLower(subId), strings.ToLower(code), strings.ToLower(createTimeStr))
	masterKey, _ := bip32.NewMasterKey(common.SHA256([]byte(key)))
	xpri, _ := masterKey.NewChildKey(0)
	xpri, _ = xpri.NewChildKey(0)
	// net := &chaincfg2.MainNet
	// if conf.IsTestMVC() {
	// 	net = &chaincfg.TestNet
	// }
	priKey, _ := bec.PrivKeyFromBytes(bec.S256(), xpri.Key)
	// wifKey, _ := wif.NewWIF(priKey, net, true)
	//address, _ := bscript.NewAddressFromPublicKey(priKey.PubKey(), false)
	//fmt.Println(address.AddressString)
	//fmt.Println(wifKey.String())
	return "", hex.EncodeToString(priKey.Serialise())
	// return "", ""
}

// ReclaimExpiredLuckyBag 发红包的人回收过时红包剩余的UTXO
func ReclaimExpiredLuckyBag(groupId, pinId, metaId, address string) (string, error) {
	luckyBag, err := chatDB.GetLuckyBagByPinId(pinId)
	if err != nil {
		return "", err
	}
	if luckyBag == nil {
		return "", errors.New("lucky bag not found")
	}

	// 验证groupId是否匹配
	if luckyBag.GroupId != groupId {
		return "", errors.New("lucky bag not match")
	}

	// 验证是否为红包创建者
	if luckyBag.MetaId != metaId {
		return "", errors.New("only lucky bag creator can reclaim")
	}

	//如果未超过半个小时，则不能回收
	if time.Now().Unix()-luckyBag.Timestamp < 30*60 {
		return "", errors.New("lucky bag has not expired")
	}

	// 获取已领取的抢红包列表
	openList, err := chatDB.GetOpenLuckyBagList(pinId)
	if err != nil {
		return "", err
	}

	// 获取已回收的红包列表
	residueList, err := chatDB.GetResidueLuckyBagList(pinId)
	if err != nil {
		return "", err
	}

	// 构建已使用的UTXO索引集合
	usedIndices := make(map[int64]bool)

	// 添加已抢红包的索引
	for _, openItem := range openList.Items {
		openLuckyBag, err := chatDB.GetOpenLuckyBagByPinId(openItem.OpenPinId)
		if err != nil || openLuckyBag == nil {
			continue
		}
		usedIndices[openLuckyBag.Index] = true
	}

	// 添加已回收红包的索引
	for _, residueItem := range residueList.Items {
		residueLuckyBag, err := chatDB.GetResidueLuckyBagByLuckyBagPinId(residueItem.ResiduePinId)
		if err != nil || residueLuckyBag == nil {
			continue
		}
		if residueLuckyBag.UsedList != nil {
			for _, used := range residueLuckyBag.UsedList {
				usedIndices[used.Index] = true
			}
		}
	}

	// 获取未使用的UTXO列表
	unusedList := make([]*respond.UnusedList, 0)
	for _, v := range luckyBag.PayList {
		if !usedIndices[v.Index] {
			unused := &respond.UnusedList{
				Index:   v.Index,
				Amount:  v.Amount,
				Address: v.Address,
			}
			unusedList = append(unusedList, unused)
		}
	}

	if len(unusedList) <= 0 {
		return "", errors.New("no unused UTXOs to reclaim")
	}

	// 执行回收逻辑
	err = commonReclaim(luckyBag, unusedList, metaId, address)
	if err != nil {
		return "", err
	}

	return "success", nil
}

// commonReclaim 执行回收红包剩余UTXO的通用逻辑
func commonReclaim(luckyBag *models.TalkGroupLuckyBagV3, unusedList []*respond.UnusedList, metaId, address string) error {
	type reclaimEntity struct {
		unusedIndex   int64
		unusedAmount  string
		unusedAddress string
	}
	reclaimEntityList := make([]*reclaimEntity, 0)

	// 收集所有未使用的UTXO
	for _, unused := range unusedList {
		reclaimEntityList = append(reclaimEntityList, &reclaimEntity{
			unusedIndex:   unused.Index,
			unusedAmount:  unused.Amount,
			unusedAddress: unused.Address,
		})
	}

	hasSuccess := false
	for _, v := range reclaimEntityList {
		vins := make([]*models.TxIn, 0)
		vins = append(vins, &models.TxIn{
			OutTxID: luckyBag.TxId,
			Index:   uint64(v.unusedIndex),
		})
		proInfoPayList := make([]*models.ProInfoPayList, 0)

		pkScript := ""
		for _, vout := range luckyBag.LuckyBagVouts {
			if vout.Index == v.unusedIndex {
				pkScript = vout.ScriptPubKey
				proInfoPayList = append(proInfoPayList, &models.ProInfoPayList{
					Amount:   strconv.FormatInt(int64(vout.Amount), 10),
					Address:  vout.Address,
					Index:    vout.Index,
					PkScript: vout.ScriptPubKey,
				})
			}
		}

		// 生成唯一的TxId和PinId
		txId := fmt.Sprintf("%s_%d_%s_reclaim", luckyBag.TxId, v.unusedIndex, metaId)
		pinId := fmt.Sprintf("%s_%d_%s_reclaim_pin", luckyBag.TxId, v.unusedIndex, metaId)

		// 检查是否已经保存过该 PinId
		existingResidue, err := chatDB.GetResidueLuckyBagByLuckyBagPinId(pinId)
		if err == nil && existingResidue != nil {
			// 已经存在，跳过处理
			continue
		}

		// 创建回收红包记录
		residueLuckyBag := &models.TalkGroupResidueLuckyBagV3{
			CommunityId:         "", // 需要从群组信息中获取
			GroupId:             luckyBag.GroupId,
			TxId:                txId,
			PinId:               pinId,
			MetaId:              metaId,
			Protocol:            luckyBag.Protocol,
			SubId:               luckyBag.SubId,
			Code:                luckyBag.Code,
			CreateTimeStr:       luckyBag.CreateTimeStr,
			Address:             address,
			PkScript:            pkScript,
			Amount:              v.unusedAmount,
			Index:               v.unusedIndex,
			Vins:                vins,
			UsedList:            proInfoPayList, // 初始化为空列表
			Type:                luckyBag.Type,
			RequireTickId:       luckyBag.RequireTickId,
			RequireCollectionId: luckyBag.RequireCollectionId,
			LuckyBagTxId:        luckyBag.TxId,
			LuckyBagPinId:       luckyBag.PinId,
			LuckyBagMetaId:      luckyBag.MetaId,
			Timestamp:           time.Now().Unix(),
			BlockHeight:         0, // 需要从实际交易中获取
			Chain:               luckyBag.Chain,
			ReclaimState:        models.GrabStateOpen,
			ReclaimTxId:         "",
			ReclaimMsg:          "",
		}

		// 保存回收红包记录到 TalkGroupResidueLuckyBagPinCollection
		err = chatDB.SaveResidueLuckyBag(residueLuckyBag)
		if err != nil {
			log.Printf("SaveResidueLuckyBag err: %v", err)
			continue
		}

		// 保存回收红包列表记录到 TalkGroupResidueLuckyBagListCollection
		err = chatDB.SaveResidueLuckyBagList(luckyBag.PinId, residueLuckyBag.PinId, luckyBag.GroupId, residueLuckyBag.Timestamp, metaId, address, []int64{v.unusedIndex})
		if err != nil {
			log.Printf("SaveResidueLuckyBagList err: %v", err)
			continue
		}

		// 将回收红包记录加入队列，等待处理
		err = chatDB.EnqueueResidueLuckyBagMessage(residueLuckyBag)
		if err != nil {
			log.Printf("EnqueueResidueLuckyBagMessage err: %v", err)
			continue
		}

		hasSuccess = true
	}

	if !hasSuccess {
		return errors.New("Reclaim err.")
	}

	return nil
}

// 处理回收红包队列中的记录
func ProcessResidueLuckyBagQueue() {
	// 获取待处理的回收红包消息
	messages, err := chatDB.GetPendingResidueLuckyBagMessages(10) // 每次处理10条
	if err != nil {
		log.Printf("GetPendingResidueLuckyBagMessages err: %v", err)
		return
	}

	for _, message := range messages {
		// 处理回收红包记录
		err := disposingReclaimLuckyBag(message.ResidueLuckyBag)
		if err != nil {
			log.Printf("disposingReclaimLuckyBag err: %v", err)
			continue
		}

		// 处理成功，删除队列消息
		err = chatDB.DeleteResidueLuckyBagQueueMessage(message.PinId)
		if err != nil {
			log.Printf("deleteResidueLuckyBagQueueMessage err: %v", err)
		}
	}
}

// 处理回收红包逻辑
func disposingReclaimLuckyBag(reclaimEntity *models.TalkGroupResidueLuckyBagV3) error {
	if chainAdapter == nil || chainAdapter[reclaimEntity.Chain] == nil {
		return fmt.Errorf("chain adapter not found")
	}

	if reclaimEntity.ReclaimState != models.GrabStateOpen {
		return nil
	}
	if reclaimEntity.Vins == nil || len(reclaimEntity.Vins) == 0 {
		return errors.New("no vins found")
	}

	_ = reclaimEntity.Vins[0] // utxo, 暂时未使用
	wifStr, hexStr := makeGiftKey(reclaimEntity.SubId, reclaimEntity.Code, reclaimEntity.CreateTimeStr)
	if wifStr == "" || hexStr == "" {
		return errors.New("failed to generate wif or hex")
	}

	// 计算总金额（所有未使用的UTXO）
	totalAmount := uint64(0)
	for _, used := range reclaimEntity.UsedList {
		amount, err := strconv.ParseUint(used.Amount, 10, 64)
		if err != nil {
			continue
		}
		totalAmount += amount
	}

	if totalAmount == 0 {
		return errors.New("no amount to reclaim")
	}

	toAddress := reclaimEntity.Address

	// 构建输入
	inputs := make([]*common.TxInputUtxo, 0)
	for _, used := range reclaimEntity.UsedList {
		amount, err := strconv.ParseUint(used.Amount, 10, 64)
		if err != nil {
			continue
		}

		input := common.TxInputUtxo{
			TxId:     reclaimEntity.LuckyBagTxId,
			TxIndex:  used.Index,
			PkScript: reclaimEntity.PkScript,
			Amount:   amount,
			PriHex:   hexStr,
		}
		inputs = append(inputs, &input)
	}

	output := common.TxOutput{
		Address: toAddress,
		Amount:  int64(totalAmount),
	}

	netParam := chainAdapter[reclaimEntity.Chain].GetNetParam()

	// 根据链类型进行类型转换
	var tx interface{}
	var buildErr error

	switch strings.ToLower(reclaimEntity.Chain) {
	case "mvc":
		// MVC链使用 chaincfg2.Params
		if mvcNetParam, ok := netParam.(*chaincfg2.Params); ok {
			tx, buildErr = common.BuildMvcTransferAllTx(mvcNetParam, inputs, &output, 1, false)
		} else {
			return fmt.Errorf("failed to convert netParam to chaincfg2.Params for MVC chain")
		}
	case "btc":
		// BTC链使用 chaincfg.Params
		if btcNetParam, ok := netParam.(*chaincfg.Params); ok {
			tx, buildErr = common.BuildMvcTransferAllTx(btcNetParam, inputs, &output, 1, false)
		} else {
			return fmt.Errorf("failed to convert netParam to chaincfg.Params for BTC chain")
		}
	default:
		return fmt.Errorf("unsupported chain type: %s", reclaimEntity.Chain)
	}
	if buildErr != nil {
		return fmt.Errorf("failed to build tx: %v", buildErr)
	}

	// 类型断言，确保tx是正确的类型
	msgTx, ok := tx.(*wire.MsgTx)
	if !ok {
		return fmt.Errorf("failed to convert tx to *wire.MsgTx")
	}

	txRaw, err := common.MvcToRaw(msgTx)
	if err != nil {
		return fmt.Errorf("failed to convert tx to raw: %v", err)
	}

	resultTxId, err := chainAdapter[reclaimEntity.Chain].BroadcastTx(txRaw)
	if resultTxId != "" {
		reclaimEntity.ReclaimState = models.GrabStateOpenAndSend
		reclaimEntity.ReclaimTxId = resultTxId
		reclaimEntity.ReclaimMsg = "success"
		log.Printf("Success: %s", resultTxId)
	} else {
		log.Printf("Failure: %s", err.Error())
		reclaimEntity.ReclaimState = models.GrabStateOpenAndSendErr
		reclaimEntity.ReclaimMsg = err.Error()
	}

	// 更新数据库中的回收红包记录
	err = chatDB.SaveResidueLuckyBag(reclaimEntity)
	if err != nil {
		return fmt.Errorf("failed to save residue lucky bag: %v", err)
	}

	return nil
}

// 启动回收红包队列处理器
func StartResidueLuckyBagQueueProcessor() {
	go func() {
		ticker := time.NewTicker(10 * time.Second) // 每10秒处理一次
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				// 处理回收红包队列
				ProcessResidueLuckyBagQueue()
			}
		}
	}()
}
