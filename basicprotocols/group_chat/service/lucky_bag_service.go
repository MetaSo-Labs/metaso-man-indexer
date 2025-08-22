package service

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"manindexer/basicprotocols/group_chat/api/respond"
	"manindexer/basicprotocols/group_chat/models"
	"manindexer/basicprotocols/group_chat/service/cache_service"
	"manindexer/basicprotocols/group_chat/service/common_service"
	"manindexer/common"
	"strconv"
	"strings"
	"time"

	chaincfg2 "github.com/bitcoinsv/bsvd/chaincfg"
	wire2 "github.com/bitcoinsv/bsvd/wire"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/wire"
	"github.com/libsv/go-bk/bec"
	"github.com/tyler-smith/go-bip32"
)

// GetLuckyBagWithOpenList Get lucky bag object and claimed list by groupId and pinId
func GetLuckyBagWithOpenList(groupId, pinId string) (*respond.LuckyBagInfoResponse, error) {
	// Get lucky bag object
	luckyBag, err := chatDB.GetLuckyBagByPinId(pinId)
	if err != nil {
		return nil, err
	}

	if luckyBag == nil {
		return nil, errors.New("lucky bag not found")
	}

	// Verify if groupId matches
	if luckyBag.GroupId != groupId {
		return nil, errors.New("lucky bag not match")
	}

	// Get claimed lucky bag list
	openList, err := chatDB.GetOpenLuckyBagList(pinId)
	if err != nil {
		return nil, err
	}

	// Get reclaimed lucky bag list
	residueList, err := chatDB.GetResidueLuckyBagList(pinId)
	if err != nil {
		return nil, err
	}

	// Build LuckyBagInfoResponse
	response := &respond.LuckyBagInfoResponse{
		TxId:                luckyBag.TxId,
		PinId:               luckyBag.PinId,
		MetaId:              luckyBag.MetaId,
		Address:             luckyBag.Address,
		UserInfo:            common_service.FetchMetaIDUserInfo(luckyBag.Address),
		SubId:               luckyBag.SubId,
		Code:                luckyBag.Code,
		CreateTime:          normalizeScientificNotation(luckyBag.CreateTimeStr),
		Content:             luckyBag.Content,
		Img:                 luckyBag.Img,
		ImgType:             luckyBag.ImgType,
		Amount:              luckyBag.Amount,
		Count:               luckyBag.Count,
		UsedCount:           "0",
		PayList:             make([]*respond.InfoPayList, 0),
		Type:                luckyBag.Type,
		TokenCount:          0, // TalkGroupLuckyBagV3 doesn't have TokenCount field
		RequireType:         luckyBag.RequireType,
		RequireTickId:       luckyBag.RequireTickId,
		RequireCollectionId: luckyBag.RequireCollectionId,
		LimitAmount:         luckyBag.LimitAmount,
	}

	usedCount := 0
	// Convert PayList - Note that ProInfoPayList has fewer fields, need to fill default values
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

		// Check claimed lucky bags
		if openList != nil {
			for _, openItem := range openList.Items {
				if openItem.LuckyBagOutIndex == payItem.Index {
					// Get detailed open lucky bag info from TalkGroupOpenLuckyBagPinCollection
					openLuckyBag, err := chatDB.GetOpenLuckyBagByPinId(openItem.OpenPinId)
					if err == nil && openLuckyBag != nil {
						infoPayList.Used = true
						infoPayList.GradTxId = openLuckyBag.GrabTxId
						infoPayList.GradMsg = openLuckyBag.GrabMsg
						infoPayList.GradState = openLuckyBag.GrabState
						infoPayList.GradPinId = openItem.OpenPinId
						infoPayList.GradMetaId = openItem.CreateMetaId
						infoPayList.GradAddress = openItem.CreateAddress
						infoPayList.UserInfo = common_service.FetchMetaIDUserInfo(openItem.CreateAddress)
						infoPayList.Timestamp = openItem.Timestamp
						// infoPayList.IsBest = true
						if openLuckyBag.GrabState == models.GrabStateOpenAndSend || openLuckyBag.GrabState == models.GrabStateChain {
							infoPayList.IsWithdraw = true
						}
						usedCount++
					}
				}
			}
		}

		// Check reclaimed lucky bags
		if residueList != nil {
			for _, residueItem := range residueList.Items {
				// Get detailed residue lucky bag info from TalkGroupResidueLuckyBagPinCollection
				residueLuckyBag, err := chatDB.GetResidueLuckyBagByLuckyBagPinId(residueItem.ResiduePinId)
				if err == nil && residueLuckyBag != nil && residueLuckyBag.UsedList != nil {
					for _, used := range residueLuckyBag.UsedList {
						if used.Index == payItem.Index {
							infoPayList.Used = true
							infoPayList.GradPinId = residueItem.ResiduePinId
							infoPayList.GradMetaId = residueItem.CreateMetaId
							infoPayList.GradAddress = residueItem.CreateAddress
							infoPayList.GradState = residueLuckyBag.ReclaimState
							infoPayList.GradMsg = residueLuckyBag.ReclaimMsg
							infoPayList.GradTxId = residueLuckyBag.ReclaimTxId
							infoPayList.UserInfo = common_service.FetchMetaIDUserInfo(residueItem.CreateAddress)
							infoPayList.Timestamp = residueItem.Timestamp
							// infoPayList.IsBest = true
							if residueLuckyBag.ReclaimState == models.GrabStateOpenAndSend || residueLuckyBag.ReclaimState == models.GrabStateChain {
								infoPayList.IsWithdraw = true
							}
							usedCount++
						}
					}
				}
			}
		}

		response.PayList = append(response.PayList, infoPayList)
	}
	response.UsedCount = strconv.Itoa(usedCount)

	return response, nil
}

// GetLuckyBagWithUnusedList Get lucky bag object and unclaimed list by groupId and pinId
func GetLuckyBagWithUnusedList(groupId, pinId string) (*respond.LuckyBagUnusedResponse, error) {
	// Get lucky bag object
	luckyBag, err := chatDB.GetLuckyBagByPinId(pinId)
	if err != nil {
		return nil, err
	}

	if luckyBag == nil {
		return nil, errors.New("lucky bag not found")
	}

	// Verify if groupId matches
	if luckyBag.GroupId != groupId {
		return nil, errors.New("lucky bag not match")
	}

	// Get claimed lucky bag list
	openList, err := chatDB.GetOpenLuckyBagList(pinId)
	if err != nil {
		return nil, err
	}

	// Get reclaimed lucky bag list
	residueList, err := chatDB.GetResidueLuckyBagList(pinId)
	if err != nil {
		return nil, err
	}

	// Build used UTXO index set
	usedIndices := make(map[int64]bool)

	// Add claimed lucky bag indices
	for _, openItem := range openList.Items {
		openLuckyBag, err := chatDB.GetOpenLuckyBagByPinId(openItem.OpenPinId)
		if err != nil || openLuckyBag == nil {
			continue
		}
		usedIndices[openLuckyBag.Index] = true
	}

	// Add reclaimed lucky bag indices
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

	// Build LuckyBagUnusedResponse
	response := &respond.LuckyBagUnusedResponse{
		PinId:               luckyBag.PinId,
		MetaId:              luckyBag.MetaId,
		Address:             luckyBag.Address,
		UserInfo:            common_service.FetchMetaIDUserInfo(luckyBag.Address),
		SubId:               luckyBag.SubId,
		Code:                luckyBag.Code,
		CreateTime:          normalizeScientificNotation(luckyBag.CreateTimeStr),
		Amount:              luckyBag.Amount,
		Count:               luckyBag.Count,
		Content:             luckyBag.Content,
		Img:                 luckyBag.Img,
		ImgType:             luckyBag.ImgType,
		Unused:              make([]*respond.UnusedList, 0),
		Type:                luckyBag.Type,
		TokenCount:          0, // TalkGroupLuckyBagV3 doesn't have TokenCount field
		RequireType:         luckyBag.RequireType,
		RequireTickId:       luckyBag.RequireTickId,
		RequireCollectionId: luckyBag.RequireCollectionId,
		LimitAmount:         luckyBag.LimitAmount,
	}

	// Get unused UTXO list
	for _, v := range luckyBag.PayList {
		if !usedIndices[v.Index] {
			unused := &respond.UnusedList{
				Index:        v.Index,
				Amount:       v.Amount,
				Address:      v.Address,
				ScriptPubKey: "", // Need to get from LuckyBagVouts
			}

			// Get ScriptPubKey from LuckyBagVouts
			for _, vout := range luckyBag.LuckyBagVouts {
				if vout.Index == v.Index {
					unused.ScriptPubKey = vout.ScriptPubKey
					break
				}
			}

			response.Unused = append(response.Unused, unused)
		}
	}

	return response, nil
}

// Temporary OpenRedMetaId structure for matching reference code
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

	// Verify if groupId matches
	if luckyBag.GroupId != groupId {
		return "", errors.New("lucky bag not match")
	}

	// Check if user is in group
	isInGroup, err := chatDB.IsUserInGroup(metaId, groupId)
	if err != nil {
		return "", err
	}
	if !isInGroup {
		return "", errors.New("user not in group")
	}

	// Get claimed lucky bag list
	openList, err := chatDB.GetOpenLuckyBagList(pinId)
	if err != nil {
		return "", err
	}

	// Build claimed lucky bag list
	openRedList := make([]*OpenRedMetaId, 0)
	for _, v := range openList.Items {
		// Get lucky bag detailed info
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

		// Check if current user has already grabbed
		if openLuckyBag.Address == address || openLuckyBag.MetaId == metaId {
			return "", errors.New("already grab")
		}
	}

	// Get reclaimed lucky bag list
	residueRedEnvelopeList, err := chatDB.GetResidueLuckyBagList(pinId)
	if err != nil {
		return "", err
	}

	if residueRedEnvelopeList != nil && len(residueRedEnvelopeList.Items) != 0 {
		for _, residueItem := range residueRedEnvelopeList.Items {
			// Get reclaimed lucky bag detailed info
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

	// Get unclaimed lucky bag list
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
		// //redis
		//         // usedMetaId, _ := redis.GetRedisGiftInfo(giftEntity.GroupId, giftEntity.TxId, unused.Index)
		//         // if usedMetaId != "" {
		//         //  if usedMetaId != metaId {
		//         //      continue
		//         //  }else {
		//         //      has = true
		//         //      grabEntityList = append(grabEntityList, &grabEntity{
		//         //          unusedIndex:   unused.Index,
		//         //          unusedAmount:  unused.Amount,
		//         //          unusedAddress: unused.Address,
		//         //          tokenIndex:    "",
		//         //      })
		//         //      break
		//         //  }
		//         // }else {
		//         //  _, err := redis.SetRedisGiftInfo(giftEntity.GroupId, giftEntity.TxId, metaId, unused.Index)
		//         //  if err != nil {
		//         //      major.Println(fmt.Sprintf("[REDIS] Set userinfo err:%s", err.Error()))
		//         //      continue
		//         //  }else {
		//         //      has = true
		//         //      grabEntityList = append(grabEntityList, &grabEntity{
		//         //          unusedIndex:   unused.Index,
		//         //          unusedAmount:  unused.Amount,
		//         //          unusedAddress: unused.Address,
		//         //          tokenIndex:    "",
		//         //      })
		//         //      break
		//         //  }
		//         // }
		//         has = true
		//         grabEntityList = append(grabEntityList, &grabEntity{
		//             unusedIndex:   unused.Index,
		//             unusedAmount:  unused.Amount,
		//             unusedAddress: unused.Address,
		//             tokenIndex:    "",
		//         })
		//         break // Only grab one lucky bag

		// Use cache service to check if lucky bag has been grabbed
		usedMetaId, err := cache_service.GetCacheGiftInfo(luckyBag.GroupId, luckyBag.PinId, unused.Index)
		if err != nil {
			log.Printf("[CACHE] Get gift info err: %s", err.Error())
			continue
		}

		if usedMetaId != "" {
			if usedMetaId != metaId {
				// Already grabbed by another user
				continue
			} else {
				// Current user has already grabbed this lucky bag
				has = true
				grabEntityList = append(grabEntityList, &grabEntity{
					unusedIndex:   unused.Index,
					unusedAmount:  unused.Amount,
					unusedAddress: unused.Address,
					tokenIndex:    "",
				})
				break
			}
		} else {
			// Try to set cache lock
			success, err := cache_service.SetCacheGiftInfo(luckyBag.GroupId, luckyBag.PinId, metaId, unused.Index)
			if err != nil {
				log.Printf("[CACHE] Set gift info err: %s", err.Error())
				continue
			}

			if success {
				has = true
				grabEntityList = append(grabEntityList, &grabEntity{
					unusedIndex:   unused.Index,
					unusedAmount:  unused.Amount,
					unusedAddress: unused.Address,
					tokenIndex:    "",
				})
				break
			}
		}
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

		// Generate unique TxId and PinId
		txId := fmt.Sprintf("%s:%d:%s:%s", luckyBag.TxId, v.unusedIndex, v.unusedAddress, metaId)
		pinId := fmt.Sprintf("%s:%d:%s:%s:pin", luckyBag.TxId, v.unusedIndex, v.unusedAddress, metaId)

		// Check if this PinId has already been saved
		existingOpen, err := chatDB.GetOpenLuckyBagByPinId(pinId)
		if err == nil && existingOpen != nil {
			// Already exists, skip processing
			return errors.New("already grab")
		}

		// Create grab lucky bag record
		openLuckyBag := &models.TalkGroupOpenLuckyBagV3{
			CommunityId:         "", // Need to get from group info
			GroupId:             luckyBag.GroupId,
			TxId:                txId,
			PinId:               pinId,
			MetaId:              metaId,
			Protocol:            "/protocol/simplegroupopenLuckybag",
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
			BlockHeight:         0, // Need to get from actual transaction
			Chain:               luckyBag.Chain,
			GrabState:           models.GrabStateOpen,
			GrabTxId:            "",
			GrabMsg:             "",
		}

		// Save grab lucky bag record to TalkGroupOpenLuckyBagPinCollection
		err = chatDB.SaveOpenLuckyBag(openLuckyBag)
		if err != nil {
			log.Printf("SaveOpenLuckyBag err: %v", err)
			continue
		}

		// Save grab lucky bag list record to TalkGroupOpenLuckyBagListCollection
		err = chatDB.SaveOpenLuckyBagList(luckyBag.PinId, openLuckyBag.PinId, luckyBag.GroupId, openLuckyBag.Timestamp, metaId, address, v.unusedIndex)
		if err != nil {
			log.Printf("SaveOpenLuckyBagList err: %v", err)
			continue
		}

		// Add grab lucky bag record to queue to be processed
		err = chatDB.EnqueueOpenLuckyBagMessage(openLuckyBag)
		if err != nil {
			log.Printf("EnqueueOpenLuckyBagMessage err: %v", err)
			continue
		}

		// Create chat message model (for group chat display)
		chat := &models.TalkGroupChatV3{
			GroupId:     luckyBag.GroupId,
			TxId:        openLuckyBag.TxId,
			PinId:       openLuckyBag.PinId,
			MetaId:      openLuckyBag.MetaId,
			Address:     openLuckyBag.Address,
			Protocol:    openLuckyBag.Protocol,
			Content:     "[Grab LuckyBag]", // Can display based on actual amount
			ContentType: "application/json",
			Encryption:  "",
			ChatType:    models.ChatTypeOpenLuckyBag, // Grab lucky bag type
			InsideIndex: models.ChatInsideIndexIn,    // Default to enter state
			ReplyPin:    luckyBag.PinId,
			Timestamp:   openLuckyBag.Timestamp,
			Chain:       openLuckyBag.Chain,
			BlockHeight: openLuckyBag.BlockHeight,
		}

		// Save chat message to TalkGroupChatPinCollection
		err = chatDB.SaveChat(chat)
		if err != nil {
			return err
		}

		// Save timestamp index (save to which collection based on user state)
		// err = chatDB.SaveChatTimestampWithState(chat)
		// if err != nil {
		// 	return err
		// }

		// // Add message to queue to asynchronously update group list
		// err = chatDB.EnqueueChatMessage(chat)
		// if err != nil {
		// 	return err
		// }

		hasSuccess = true
		break // Only process one lucky bag
	}

	if !hasSuccess {
		return errors.New("Grab err.")
	}

	return nil
}

// Process records in grab lucky bag queue
func ProcessOpenLuckyBagQueue() {
	// Get total count of TalkGroupOpenLuckyBagQueueCollection
	totalCount, err := chatDB.GetOpenLuckyBagQueueCount()
	if err != nil {
		log.Printf("GetOpenLuckyBagQueueCount err: %v", err)
	} else {
		log.Printf("TalkGroupOpenLuckyBagQueueCollection total count: %d", totalCount)
	}

	// Get pending grab lucky bag messages
	messages, err := chatDB.GetPendingOpenLuckyBagMessages(100) // Process 100 items each time
	if err != nil {
		log.Printf("GetPendingOpenLuckyBagMessages err: %v", err)
		return
	}

	for _, message := range messages {
		// Process grab lucky bag record
		err := disposingGrabLuckyBag(message.OpenLuckyBag, totalCount)
		if err != nil {
			log.Printf("disposingGrabLuckyBag err: %v", err)
			continue
		}

		// Process successfully, delete queue message
		err = chatDB.DeleteOpenLuckyBagQueueMessage(message.PinId)
		if err != nil {
			log.Printf("deleteOpenLuckyBagQueueMessage err: %v", err)
		}
	}
}

// Process grab lucky bag logic
func disposingGrabLuckyBag(grabEntity *models.TalkGroupOpenLuckyBagV3, totalCount int64) error {
	if chainAdapter == nil || chainAdapter[grabEntity.Chain] == nil {
		return fmt.Errorf("chain adapter not found")
	}

	if grabEntity.GrabState != models.GrabStateOpen {
		return nil
	}
	if grabEntity.Vins == nil || len(grabEntity.Vins) == 0 {
		return errors.New("no vins found")
	}

	_ = grabEntity.Vins[0] // utxo, temporarily unused
	_, hexStr := makeGiftKey(grabEntity.SubId, grabEntity.Code, grabEntity.CreateTimeStr)
	if hexStr == "" {
		return errors.New("failed to generate wif or hex")
	}

	// Handle scientific notation in amount
	normalizedAmount := normalizeScientificNotation(grabEntity.Amount)
	value, err := strconv.ParseUint(normalizedAmount, 10, 64)
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
		SignMode: common.SignModeLegacy,
	}
	output := common.TxOutput{
		Address: toAddress,
		Amount:  int64(value),
	}

	netParam := chainAdapter[grabEntity.Chain].GetNetParam()

	// Type conversion based on chain type
	var tx interface{}
	var buildErr error

	switch strings.ToLower(grabEntity.Chain) {
	case "mvc":
		// MVC chain uses chaincfg2.Params
		if mvcNetParam, ok := netParam.(*chaincfg2.Params); ok {
			tx, buildErr = common.BuildMvcTransferAllTx(mvcNetParam, []*common.TxInputUtxo{&input}, &output, 1, false)
		} else {
			return fmt.Errorf("failed to convert netParam to chaincfg2.Params for MVC chain")
		}
		break
	case "btc":
		// BTC chain uses chaincfg.Params
		if btcNetParam, ok := netParam.(*chaincfg.Params); ok {
			tx, buildErr = common.BuildBtcTransferAllTx(btcNetParam, []*common.TxInputUtxo{&input}, &output, 1, false)
		} else {
			return fmt.Errorf("failed to convert netParam to chaincfg.Params for BTC chain")
		}
		break
	default:
		return fmt.Errorf("unsupported chain type: %s", grabEntity.Chain)
	}
	if buildErr != nil {
		return fmt.Errorf("failed to build tx: %v", buildErr)
	}

	var txRaw string
	switch strings.ToLower(grabEntity.Chain) {
	case "mvc":
		// Type assertion to ensure tx is the correct type
		msgTx, ok := tx.(*wire2.MsgTx)
		if !ok {
			return fmt.Errorf("[%s]failed to convert tx to *wire.MsgTx", grabEntity.Chain)
		}

		txRaw, err = common.MvcToRaw(msgTx)
		if err != nil {
			return fmt.Errorf("[%s]failed to convert tx to raw: %v", grabEntity.Chain, err)
		}
	case "btc":
		msgTx, ok := tx.(*wire.MsgTx)
		if !ok {
			return fmt.Errorf("[%s]failed to convert tx to *wire.MsgTx", grabEntity.Chain)
		}

		var b bytes.Buffer
		err = msgTx.Serialize(&b)
		if err != nil {
			return fmt.Errorf("[%s]failed to serialize tx: %v", grabEntity.Chain, err)
		}
		txRaw = hex.EncodeToString(b.Bytes())
	}

	resultTxId, err := chainAdapter[grabEntity.Chain].BroadcastTx(txRaw)
	if resultTxId != "" {
		grabEntity.GrabState = models.GrabStateOpenAndSend
		grabEntity.GrabTxId = resultTxId
		grabEntity.GrabMsg = "success"
		log.Printf("Success broadcast tx: %s, totalCount: %d", resultTxId, totalCount)
	} else {
		log.Printf("Failure broadcast tx: %s, totalCount: %d", err.Error(), totalCount)
		grabEntity.GrabState = models.GrabStateOpenAndSendErr
		grabEntity.GrabMsg = err.Error()

		// Check if error contains broadcast-related issues, if not, return the error
		if !isBroadcastError(err) {
			return fmt.Errorf("broadcast transaction failed: %v", err)
		}
	}

	// Update grab lucky bag record in database
	err = chatDB.SaveOpenLuckyBag(grabEntity)
	if err != nil {
		return fmt.Errorf("failed to save open lucky bag: %v", err)
	}

	return nil
}

// Start grab lucky bag queue processor
func StartOpenLuckyBagQueueProcessor() {
	go func() {
		ticker := time.NewTicker(10 * time.Second) // Process every 10 seconds
		defer ticker.Stop()

		// Flag to track if the previous processing is still running
		isProcessing := false

		for {
			select {
			case <-ticker.C:
				// Skip if previous processing is still running
				if isProcessing {
					log.Printf("Previous ProcessOpenLuckyBagQueue is still running, skipping this cycle")
					continue
				}

				// Set processing flag
				isProcessing = true

				// Process grab lucky bag queue in a goroutine
				go func() {
					defer func() {
						// Reset processing flag when done
						isProcessing = false
					}()

					ProcessOpenLuckyBagQueue()
				}()
			}
		}
	}()
}

func makeGiftKey(subId, code, createTimeStr string) (string, string) {
	// Check if createTimeStr is in scientific notation and convert it
	normalizedCreateTimeStr := normalizeScientificNotation(createTimeStr)

	key := fmt.Sprintf("%s%s%s", strings.ToLower(subId), strings.ToLower(code), strings.ToLower(normalizedCreateTimeStr))
	masterKey, _ := bip32.NewMasterKey(common.SHA256([]byte(key)))
	xpri, _ := masterKey.NewChildKey(0)
	xpri, _ = xpri.NewChildKey(0)
	priKey, _ := bec.PrivKeyFromBytes(bec.S256(), xpri.Key)

	return "", hex.EncodeToString(priKey.Serialise())
}

// normalizeScientificNotation converts scientific notation to normal decimal format
func normalizeScientificNotation(value string) string {
	// Check if the string contains scientific notation (e.g., "1.75525051088e+12")
	if strings.Contains(strings.ToLower(value), "e") {
		// Try to parse as float64 first
		if f, err := strconv.ParseFloat(value, 64); err == nil {
			// Convert to int64 to avoid decimal places
			return strconv.FormatInt(int64(f), 10)
		}
	}
	// If not scientific notation or parsing failed, return original value
	return value
}

// ReclaimExpiredLuckyBag Lucky bag creator reclaims remaining UTXOs from expired lucky bags
func ReclaimExpiredLuckyBag(groupId, pinId, metaId, address string) (string, error) {
	luckyBag, err := chatDB.GetLuckyBagByPinId(pinId)
	if err != nil {
		return "", err
	}
	if luckyBag == nil {
		return "", errors.New("lucky bag not found")
	}

	// Verify if groupId matches
	if luckyBag.GroupId != groupId {
		return "", errors.New("lucky bag not match")
	}

	// Verify if it's the lucky bag creator
	if luckyBag.MetaId != metaId {
		return "", errors.New("only lucky bag creator can reclaim")
	}

	// If not more than 30 minutes, cannot reclaim
	if time.Now().Unix()-luckyBag.Timestamp < 30*60 {
		return "", errors.New("lucky bag has not expired")
	}

	// Get claimed lucky bag list
	openList, err := chatDB.GetOpenLuckyBagList(pinId)
	if err != nil {
		return "", err
	}

	// Get reclaimed lucky bag list
	residueList, err := chatDB.GetResidueLuckyBagList(pinId)
	if err != nil {
		return "", err
	}

	// Build used UTXO index set
	usedIndices := make(map[int64]bool)

	// Add claimed lucky bag indices
	for _, openItem := range openList.Items {
		openLuckyBag, err := chatDB.GetOpenLuckyBagByPinId(openItem.OpenPinId)
		if err != nil || openLuckyBag == nil {
			continue
		}
		usedIndices[openLuckyBag.Index] = true
	}

	// Add reclaimed lucky bag indices
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

	// Get unused UTXO list
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

	// Execute reclaim logic
	err = commonReclaim(luckyBag, unusedList, metaId, address)
	if err != nil {
		return "", err
	}

	return "success", nil
}

// commonReclaim Execute common logic for reclaiming remaining UTXOs from lucky bags
func commonReclaim(luckyBag *models.TalkGroupLuckyBagV3, unusedList []*respond.UnusedList, metaId, address string) error {
	type reclaimEntity struct {
		unusedIndex   int64
		unusedAmount  string
		unusedAddress string
	}
	reclaimEntityList := make([]*reclaimEntity, 0)

	// Collect all unused UTXOs
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

		// Generate unique TxId and PinId
		txId := fmt.Sprintf("%s:%d:%s:reclaim", luckyBag.TxId, v.unusedIndex, metaId)
		pinId := fmt.Sprintf("%s:%d:%s:reclaim:pin", luckyBag.TxId, v.unusedIndex, metaId)

		// Check if this PinId has already been saved
		existingResidue, err := chatDB.GetResidueLuckyBagByLuckyBagPinId(pinId)
		if err == nil && existingResidue != nil {
			// Already exists, skip processing
			continue
		}

		// Create reclaim lucky bag record
		residueLuckyBag := &models.TalkGroupResidueLuckyBagV3{
			CommunityId:         "", // Need to get from group info
			GroupId:             luckyBag.GroupId,
			TxId:                txId,
			PinId:               pinId,
			MetaId:              metaId,
			Protocol:            "/protocol/simplegroupresidueLuckybag",
			SubId:               luckyBag.SubId,
			Code:                luckyBag.Code,
			CreateTimeStr:       luckyBag.CreateTimeStr,
			Address:             address,
			PkScript:            pkScript,
			Amount:              v.unusedAmount,
			Index:               v.unusedIndex,
			Vins:                vins,
			UsedList:            proInfoPayList, // Initialize as empty list
			Type:                luckyBag.Type,
			RequireTickId:       luckyBag.RequireTickId,
			RequireCollectionId: luckyBag.RequireCollectionId,
			LuckyBagTxId:        luckyBag.TxId,
			LuckyBagPinId:       luckyBag.PinId,
			LuckyBagMetaId:      luckyBag.MetaId,
			Timestamp:           time.Now().Unix(),
			BlockHeight:         0, // Need to get from actual transaction
			Chain:               luckyBag.Chain,
			ReclaimState:        models.GrabStateOpen,
			ReclaimTxId:         "",
			ReclaimMsg:          "",
		}

		// Save reclaim lucky bag record to TalkGroupResidueLuckyBagPinCollection
		err = chatDB.SaveResidueLuckyBag(residueLuckyBag)
		if err != nil {
			log.Printf("SaveResidueLuckyBag err: %v", err)
			continue
		}

		// Save reclaim lucky bag list record to TalkGroupResidueLuckyBagListCollection
		err = chatDB.SaveResidueLuckyBagList(luckyBag.PinId, residueLuckyBag.PinId, luckyBag.GroupId, residueLuckyBag.Timestamp, metaId, address, []int64{v.unusedIndex})
		if err != nil {
			log.Printf("SaveResidueLuckyBagList err: %v", err)
			continue
		}

		// Add reclaim lucky bag record to queue to be processed
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

// Process records in reclaim lucky bag queue
func ProcessResidueLuckyBagQueue() {
	// Get pending reclaim lucky bag messages
	messages, err := chatDB.GetPendingResidueLuckyBagMessages(10) // Process 10 items each time
	if err != nil {
		log.Printf("GetPendingResidueLuckyBagMessages err: %v", err)
		return
	}

	for _, message := range messages {
		// Process reclaim lucky bag record
		err := disposingReclaimLuckyBag(message.ResidueLuckyBag)
		if err != nil {
			log.Printf("disposingReclaimLuckyBag err: %v", err)
			continue
		}

		// Process successfully, delete queue message
		err = chatDB.DeleteResidueLuckyBagQueueMessage(message.PinId)
		if err != nil {
			log.Printf("deleteResidueLuckyBagQueueMessage err: %v", err)
		}
	}
}

// Process reclaim lucky bag logic
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

	_ = reclaimEntity.Vins[0] // utxo, temporarily unused
	_, hexStr := makeGiftKey(reclaimEntity.SubId, reclaimEntity.Code, reclaimEntity.CreateTimeStr)
	if hexStr == "" {
		return errors.New("failed to generate wif or hex")
	}

	// Calculate total amount (all unused UTXOs)
	totalAmount := uint64(0)
	for _, used := range reclaimEntity.UsedList {
		// Handle scientific notation in amount
		normalizedAmount := normalizeScientificNotation(used.Amount)
		amount, err := strconv.ParseUint(normalizedAmount, 10, 64)
		if err != nil {
			continue
		}
		totalAmount += amount
	}

	if totalAmount == 0 {
		return errors.New("no amount to reclaim")
	}

	toAddress := reclaimEntity.Address

	// Build inputs
	inputs := make([]*common.TxInputUtxo, 0)
	for _, used := range reclaimEntity.UsedList {
		// Handle scientific notation in amount
		normalizedAmount := normalizeScientificNotation(used.Amount)
		amount, err := strconv.ParseUint(normalizedAmount, 10, 64)
		if err != nil {
			continue
		}

		input := common.TxInputUtxo{
			TxId:     reclaimEntity.LuckyBagTxId,
			TxIndex:  used.Index,
			PkScript: reclaimEntity.PkScript,
			Amount:   amount,
			PriHex:   hexStr,
			SignMode: common.SignModeLegacy,
		}
		inputs = append(inputs, &input)
	}

	output := common.TxOutput{
		Address: toAddress,
		Amount:  int64(totalAmount),
	}

	netParam := chainAdapter[reclaimEntity.Chain].GetNetParam()

	// Type conversion based on chain type
	var tx interface{}
	var buildErr error

	switch strings.ToLower(reclaimEntity.Chain) {
	case "mvc":
		// MVC chain uses chaincfg2.Params
		if mvcNetParam, ok := netParam.(*chaincfg2.Params); ok {
			tx, buildErr = common.BuildMvcTransferAllTx(mvcNetParam, inputs, &output, 1, false)
		} else {
			return fmt.Errorf("failed to convert netParam to chaincfg2.Params for MVC chain")
		}
		break
	case "btc":
		// BTC chain uses chaincfg.Params
		if btcNetParam, ok := netParam.(*chaincfg.Params); ok {
			tx, buildErr = common.BuildBtcTransferAllTx(btcNetParam, inputs, &output, 1, false)
		} else {
			return fmt.Errorf("failed to convert netParam to chaincfg.Params for BTC chain")
		}
		break
	default:
		return fmt.Errorf("unsupported chain type: %s", reclaimEntity.Chain)
	}
	if buildErr != nil {
		return fmt.Errorf("failed to build tx: %v", buildErr)
	}

	var (
		txRaw string
		err   error
	)
	switch strings.ToLower(reclaimEntity.Chain) {
	case "mvc":
		// Type assertion to ensure tx is the correct type
		msgTx, ok := tx.(*wire2.MsgTx)
		if !ok {
			return fmt.Errorf("[%s]failed to convert tx to *wire.MsgTx", reclaimEntity.Chain)
		}

		txRaw, err = common.MvcToRaw(msgTx)
		if err != nil {
			return fmt.Errorf("[%s]failed to convert tx to raw: %v", reclaimEntity.Chain, err)
		}
		break
	case "btc":
		msgTx, ok := tx.(*wire.MsgTx)
		if !ok {
			return fmt.Errorf("[%s]failed to convert tx to *wire.MsgTx", reclaimEntity.Chain)
		}

		var b bytes.Buffer
		err = msgTx.Serialize(&b)
		if err != nil {
			return fmt.Errorf("[%s]failed to serialize tx: %v", reclaimEntity.Chain, err)
		}
		txRaw = hex.EncodeToString(b.Bytes())
		break
	}

	resultTxId, err := chainAdapter[reclaimEntity.Chain].BroadcastTx(txRaw)
	if resultTxId != "" {
		reclaimEntity.ReclaimState = models.GrabStateOpenAndSend
		reclaimEntity.ReclaimTxId = resultTxId
		reclaimEntity.ReclaimMsg = "success"
		log.Printf("Success broadcast tx: %s", resultTxId)
	} else {
		log.Printf("Failure broadcast tx: %s", err.Error())
		reclaimEntity.ReclaimState = models.GrabStateOpenAndSendErr
		reclaimEntity.ReclaimMsg = err.Error()

		// Check if error contains broadcast-related issues, if not, return the error
		if !isBroadcastError(err) {
			return fmt.Errorf("broadcast transaction failed: %v", err)
		}
	}

	// Update reclaim lucky bag record in database
	err = chatDB.SaveResidueLuckyBag(reclaimEntity)
	if err != nil {
		return fmt.Errorf("failed to save residue lucky bag: %v", err)
	}

	return nil
}

// Start reclaim lucky bag queue processor
func StartResidueLuckyBagQueueProcessor() {
	go func() {
		ticker := time.NewTicker(10 * time.Second) // Process every 10 seconds
		defer ticker.Stop()

		// Flag to track if the previous processing is still running
		isProcessing := false

		for {
			select {
			case <-ticker.C:
				// Skip if previous processing is still running
				if isProcessing {
					log.Printf("Previous ProcessResidueLuckyBagQueue is still running, skipping this cycle")
					continue
				}

				// Set processing flag
				isProcessing = true

				// Process reclaim lucky bag queue in a goroutine
				go func() {
					defer func() {
						// Reset processing flag when done
						isProcessing = false
					}()

					ProcessResidueLuckyBagQueue()
				}()
			}
		}
	}()
}

// isBroadcastError checks if the error contains broadcast-related issues that should be handled gracefully
func isBroadcastError(err error) bool {
	if err == nil {
		return false
	}

	errMsg := strings.ToLower(err.Error())

	// Check for common broadcast-related error messages
	broadcastErrors := []string{
		"missing inputs",
		"missing input",
		"missing-input",
		"txn-mempool-conflict",
		"txn mempool conflict",
	}

	for _, broadcastErr := range broadcastErrors {
		if strings.Contains(errMsg, broadcastErr) {
			return true
		}
	}

	return false
}
