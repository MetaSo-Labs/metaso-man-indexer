package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"manindexer/basicprotocols/group_chat/api/respond"
	"manindexer/basicprotocols/group_chat/common_util/logger"
	"manindexer/basicprotocols/group_chat/models"
	"manindexer/basicprotocols/group_chat/service/cache_service"
	"manindexer/basicprotocols/group_chat/service/common_service"
	"manindexer/basicprotocols/group_chat/service/grpc_service/grpc_metacontract"
	"strconv"
	"strings"
	"time"
)

// GetLuckyBagWithOpenListV2 Get lucky bag object and claimed list by groupId and pinId (V2 with cache)
func GetLuckyBagWithOpenListV2(groupId, pinId string) (*respond.LuckyBagInfoResponse, error) {
	startTime := time.Now()
	var perfStats = struct {
		getLuckyBagTime        int64
		getOpenListTime        int64
		getResidueListTime     int64
		processOpenListTime    int64
		processResidueListTime int64
		responseFormatTime     int64
		totalTime              int64
	}{}

	t := time.Now().UnixMilli()
	// 1. First get lucky bag object from cache
	luckyBag, err := cache_service.GetCacheLuckyBag(groupId, pinId)
	if err != nil {
		log.Printf("[GetLuckyBagWithOpenListV2] Cache get error: %v", err)
		// Cache failed, get from database
		luckyBag, err = chatDB.GetLuckyBagByPinId(pinId)
		if err != nil {
			return nil, err
		}
		if luckyBag == nil {
			return nil, errors.New("lucky bag not found")
		}
		// Write data to cache
		cache_service.SetCacheLuckyBag(luckyBag)
	} else if luckyBag == nil {
		// Not in cache, get from database
		luckyBag, err = chatDB.GetLuckyBagByPinId(pinId)
		if err != nil {
			return nil, err
		}
		if luckyBag == nil {
			return nil, errors.New("lucky bag not found")
		}
		// Write data to cache
		cache_service.SetCacheLuckyBag(luckyBag)
	}
	perfStats.getLuckyBagTime = time.Now().UnixMilli() - t

	// Verify if groupId matches
	if luckyBag.GroupId != groupId {
		return nil, errors.New("lucky bag not match")
	}

	t = time.Now().UnixMilli()
	// 2. Get opened lucky bag list from cache
	openList, err := cache_service.GetCacheOpenLuckyBagList(groupId, pinId)
	if err != nil {
		log.Printf("[GetLuckyBagWithOpenListV2] Cache get open list error: %v", err)
		// Cache failed, get from database
		openList, err = chatDB.GetOpenLuckyBagList(pinId)
		if err != nil {
			return nil, err
		}
		// Write data to cache
		if openList != nil {
			cache_service.SetCacheOpenLuckyBagList(groupId, pinId, openList)
		}
	} else if openList == nil {
		// Not in cache, get from database
		openList, err = chatDB.GetOpenLuckyBagList(pinId)
		if err != nil {
			return nil, err
		}
		// Write data to cache
		if openList != nil {
			cache_service.SetCacheOpenLuckyBagList(groupId, pinId, openList)
		}
	}
	perfStats.getOpenListTime = time.Now().UnixMilli() - t

	t = time.Now().UnixMilli()
	// 3. Get reclaimed lucky bag list from cache
	residueList, err := cache_service.GetCacheResidueLuckyBagList(groupId, pinId)
	if err != nil {
		log.Printf("[GetLuckyBagWithOpenListV2] Cache get residue list error: %v", err)
		// Cache failed, get from database
		residueList, err = chatDB.GetResidueLuckyBagList(pinId)
		if err != nil {
			return nil, err
		}
		// Write data to cache
		if residueList != nil {
			cache_service.SetCacheResidueLuckyBagList(groupId, pinId, residueList)
		}
	} else if residueList == nil {
		// Not in cache, get from database
		residueList, err = chatDB.GetResidueLuckyBagList(pinId)
		if err != nil {
			return nil, err
		}
		// Write data to cache
		if residueList != nil {
			cache_service.SetCacheResidueLuckyBagList(groupId, pinId, residueList)
		}
	}
	perfStats.getResidueListTime = time.Now().UnixMilli() - t

	var tickInfo *respond.TickInfo
	if strings.ToLower(luckyBag.Type) == string(models.LuckyBagTypeMetacontractFT) && luckyBag.TickPinId != "" {
		// First get lucky bag extra from cache
		extra, err := cache_service.GetCacheLuckyBagExtra(luckyBag.TickTxId)
		if err != nil {
			log.Printf("[GetLuckyBagWithOpenListV2] Cache get lucky bag extra error: %v", err)
			// Cache failed, get from database
			extra, err = extraDB.GetLuckyBagExtraByTxId(luckyBag.TickTxId)
			if err != nil {
				return nil, err
			}
			// Write data to cache
			if extra != nil {
				cache_service.SetCacheLuckyBagExtra(luckyBag.TickTxId, extra)
			}
		} else if extra == nil {
			// Not in cache, get from database
			extra, err = extraDB.GetLuckyBagExtraByTxId(luckyBag.TickTxId)
			if err != nil {
				return nil, err
			}
			// Write data to cache
			if extra != nil {
				cache_service.SetCacheLuckyBagExtra(luckyBag.TickTxId, extra)
			}
		}
		if extra != nil {
			tickInfo = &respond.TickInfo{
				TickType:   extra.Type,
				Codehash:   extra.Codehash,
				GenesisId:  extra.GenesisId,
				Genesis:    extra.Genesis,
				SensibleId: extra.SensibleId,
				Name:       extra.Name,
				Symbol:     extra.Symbol,
				Decimal:    extra.Decimal,
			}
		}
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
		Domain:              luckyBag.Domain,
		LuckyBagAddress:     luckyBag.LuckyBagAddress,
		LuckyBagGasAddress:  luckyBag.LuckyBagGasAddress,
		TickPinId:           luckyBag.TickPinId,
		TickTxId:            luckyBag.TickTxId,
		GenType:             luckyBag.GenType,
		GenState:            luckyBag.GenState,
		Content:             luckyBag.Content,
		Img:                 luckyBag.Img,
		ImgType:             luckyBag.ImgType,
		Amount:              luckyBag.Amount,
		LuckyTotalAmount:    luckyBag.LuckyTotalAmount,
		LuckyTotalFee:       luckyBag.LuckyTotalFee,
		FeeRate:             luckyBag.FeeRate,
		Count:               luckyBag.Count,
		ValidCount:          luckyBag.ValidCount,
		UsedCount:           "0",
		PayList:             make([]*respond.InfoPayList, 0),
		ErrPayList:          make([]*respond.InfoPayList, 0),
		Type:                luckyBag.Type,
		TickId:              luckyBag.TickId,
		TickInfo:            tickInfo,
		CollectionId:        luckyBag.CollectionId,
		TokenCount:          0, // TalkGroupLuckyBagV3 doesn't have TokenCount field
		RequireType:         luckyBag.RequireType,
		RequireTickId:       luckyBag.RequireTickId,
		RequireCollectionId: luckyBag.RequireCollectionId,
		LimitAmount:         luckyBag.LimitAmount,
	}
	if response.LuckyTotalAmount == "" || response.LuckyTotalAmount == "0" {
		response.LuckyTotalAmount = luckyBag.Amount
	}

	usedCount := 0
	t = time.Now().UnixMilli()
	// Convert PayList - Note that ProInfoPayList has fewer fields, need to fill default values
	for _, payItem := range luckyBag.PayList {
		infoPayList := &respond.InfoPayList{
			TxId:         luckyBag.TxId,
			TokenTxId:    luckyBag.TickTxId,
			Index:        payItem.Index,
			Amount:       payItem.Amount,
			Address:      payItem.Address,
			Used:         false,
			LuckyAmount:  payItem.LuckyAmount,
			LuckyFee:     payItem.LuckyFee,
			LuckyFeeRate: payItem.LuckyFeeRate,
			GasAmount:    payItem.GasAmount,
			GasAddress:   payItem.GasAddress,
			GasIndex:     payItem.GasIndex,
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
		if infoPayList.LuckyAmount == "" || infoPayList.LuckyAmount == "0" {
			infoPayList.LuckyAmount = payItem.Amount
		}

		// Check claimed lucky bags
		if openList != nil {
			tn1 := time.Now().UnixMilli()
			for _, openItem := range openList.Items {
				if openItem.LuckyBagOutIndex == payItem.Index {
					// First get open lucky bag detail from cache
					openLuckyBag, err := cache_service.GetCacheOpenLuckyBag(groupId, openItem.OpenPinId)
					if err != nil {
						log.Printf("[GetLuckyBagWithOpenListV2] Cache get open lucky bag error: %v", err)
						// Cache failed, get from database
						openLuckyBag, err = chatDB.GetOpenLuckyBagByPinId(openItem.OpenPinId)
						if err == nil && openLuckyBag != nil {
							// Write data to cache
							cache_service.SetCacheOpenLuckyBag(groupId, openItem.OpenPinId, openLuckyBag)
						}
					} else if openLuckyBag == nil {
						// Not in cache, get from database
						openLuckyBag, err = chatDB.GetOpenLuckyBagByPinId(openItem.OpenPinId)
						if err == nil && openLuckyBag != nil {
							// Write data to cache
							cache_service.SetCacheOpenLuckyBag(groupId, openItem.OpenPinId, openLuckyBag)
						}
					}

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
			perfStats.processOpenListTime += time.Now().UnixMilli() - tn1
		}

		// Check reclaimed lucky bags
		if residueList != nil {
			tn2 := time.Now().UnixMilli()
			for _, residueItem := range residueList.Items {
				// First get residue lucky bag detail from cache
				residueLuckyBag, err := cache_service.GetCacheResidueLuckyBag(groupId, residueItem.ResiduePinId)
				if err != nil {
					log.Printf("[GetLuckyBagWithOpenListV2] Cache get residue lucky bag error: %v", err)
					// Cache failed, get from database
					residueLuckyBag, err = chatDB.GetResidueLuckyBagByLuckyBagPinId(residueItem.ResiduePinId)
					if err == nil && residueLuckyBag != nil {
						// Write data to cache
						cache_service.SetCacheResidueLuckyBag(groupId, residueItem.ResiduePinId, residueLuckyBag)
					}
				} else if residueLuckyBag == nil {
					// Not in cache, get from database
					residueLuckyBag, err = chatDB.GetResidueLuckyBagByLuckyBagPinId(residueItem.ResiduePinId)
					if err == nil && residueLuckyBag != nil {
						// Write data to cache
						cache_service.SetCacheResidueLuckyBag(groupId, residueItem.ResiduePinId, residueLuckyBag)
					}
				}

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
							if residueLuckyBag.ReclaimState == models.GrabStateReclaimAndSend || residueLuckyBag.ReclaimState == models.GrabStateChain {
								infoPayList.IsWithdraw = true
							}
							usedCount++
						}
					}
				}
			}
			perfStats.processResidueListTime += time.Now().UnixMilli() - tn2
		}

		response.PayList = append(response.PayList, infoPayList)
	}
	response.UsedCount = strconv.Itoa(usedCount)
	perfStats.responseFormatTime = time.Now().UnixMilli() - startTime.UnixMilli()
	perfStats.totalTime = time.Since(startTime).Milliseconds()

	logger.Info("[LUCKY_BAG_SERVICE_V2][GET_LUCKY_BAG_WITH_OPEN_LIST] Performance Stats - "+
		"Total: %dms, GetLuckyBag: %dms, GetOpenList: %dms, GetResidueList: %dms, "+
		"ProcessOpenList: %dms, ProcessResidueList: %dms, ResponseFormat: %dms, Items: %d",
		perfStats.totalTime,
		perfStats.getLuckyBagTime,
		perfStats.getOpenListTime,
		perfStats.getResidueListTime,
		perfStats.processOpenListTime,
		perfStats.processResidueListTime,
		perfStats.responseFormatTime,
		len(response.PayList))

	return response, nil
}

// GetLuckyBagWithUnusedListV2 Get lucky bag object and unclaimed list by groupId and pinId (V2 with cache)
func GetLuckyBagWithUnusedListV2(groupId, pinId string) (*respond.LuckyBagUnusedResponse, error) {
	startTime := time.Now()
	var perfStats = struct {
		getLuckyBagTime        int64
		getOpenListTime        int64
		getResidueListTime     int64
		processOpenListTime    int64
		processResidueListTime int64
		buildUnusedListTime    int64
		responseFormatTime     int64
		totalTime              int64
	}{}

	t := time.Now().UnixMilli()
	// 1. First get lucky bag object from cache
	luckyBag, err := cache_service.GetCacheLuckyBag(groupId, pinId)
	if err != nil {
		log.Printf("[GetLuckyBagWithUnusedListV2] Cache get error: %v", err)
		// Cache failed, get from database
		luckyBag, err = chatDB.GetLuckyBagByPinId(pinId)
		if err != nil {
			return nil, err
		}
		if luckyBag == nil {
			return nil, errors.New("lucky bag not found")
		}
		// Write data to cache
		cache_service.SetCacheLuckyBag(luckyBag)
	} else if luckyBag == nil {
		// Not in cache, get from database
		luckyBag, err = chatDB.GetLuckyBagByPinId(pinId)
		if err != nil {
			return nil, err
		}
		if luckyBag == nil {
			return nil, errors.New("lucky bag not found")
		}
		// Write data to cache
		cache_service.SetCacheLuckyBag(luckyBag)
	}
	perfStats.getLuckyBagTime = time.Now().UnixMilli() - t

	// Verify if groupId matches
	if luckyBag.GroupId != groupId {
		return nil, errors.New("lucky bag not match")
	}

	if luckyBag.Domain != "" && luckyBag.LuckyBagAddress != "" {
		if luckyBag.GenType == 2 {
			return nil, errors.New("lucky bag is external")
		}
		// if strings.TrimSuffix(luckyBag.Domain, "/") != strings.TrimSuffix(common.Config.GroupChat.LuckyBagDomain, "/") {
		// 	return nil, errors.New("lucky bag domain not match")
		// }
		if !checkLuckyBagDomain(luckyBag.Domain) {
			return nil, errors.New("lucky bag domain not match")
		}
		if luckyBag.GenType == 1 && luckyBag.GenState != 1 {
			return nil, errors.New("lucky bag is internal and failed")
		}
	}

	t = time.Now().UnixMilli()
	// 2. Get opened lucky bag list from cache
	openList, err := cache_service.GetCacheOpenLuckyBagList(groupId, pinId)
	if err != nil {
		log.Printf("[GetLuckyBagWithUnusedListV2] Cache get open list error: %v", err)
		// Cache failed, get from database
		openList, err = chatDB.GetOpenLuckyBagList(pinId)
		if err != nil {
			return nil, err
		}
		// Write data to cache
		if openList != nil {
			cache_service.SetCacheOpenLuckyBagList(groupId, pinId, openList)
		}
	} else if openList == nil {
		// Not in cache, get from database
		openList, err = chatDB.GetOpenLuckyBagList(pinId)
		if err != nil {
			return nil, err
		}
		// Write data to cache
		if openList != nil {
			cache_service.SetCacheOpenLuckyBagList(groupId, pinId, openList)
		}
	}
	perfStats.getOpenListTime = time.Now().UnixMilli() - t

	t = time.Now().UnixMilli()
	// 3. Get reclaimed lucky bag list from cache
	residueList, err := cache_service.GetCacheResidueLuckyBagList(groupId, pinId)
	if err != nil {
		log.Printf("[GetLuckyBagWithUnusedListV2] Cache get residue list error: %v", err)
		// Cache failed, get from database
		residueList, err = chatDB.GetResidueLuckyBagList(pinId)
		if err != nil {
			return nil, err
		}
		// Write data to cache
		if residueList != nil {
			cache_service.SetCacheResidueLuckyBagList(groupId, pinId, residueList)
		}
	} else if residueList == nil {
		// Not in cache, get from database
		residueList, err = chatDB.GetResidueLuckyBagList(pinId)
		if err != nil {
			return nil, err
		}
		// Write data to cache
		if residueList != nil {
			cache_service.SetCacheResidueLuckyBagList(groupId, pinId, residueList)
		}
	}
	perfStats.getResidueListTime = time.Now().UnixMilli() - t

	// Build used UTXO index set
	usedIndices := make(map[int64]bool)

	t = time.Now().UnixMilli()
	// Add claimed lucky bag indices
	for _, openItem := range openList.Items {
		// First get open lucky bag detail from cache
		openLuckyBag, err := cache_service.GetCacheOpenLuckyBag(groupId, openItem.OpenPinId)
		if err != nil {
			log.Printf("[GetLuckyBagWithUnusedListV2] Cache get open lucky bag error: %v", err)
			// Cache failed, get from database
			openLuckyBag, err = chatDB.GetOpenLuckyBagByPinId(openItem.OpenPinId)
			if err == nil && openLuckyBag != nil {
				// Write data to cache
				cache_service.SetCacheOpenLuckyBag(groupId, openItem.OpenPinId, openLuckyBag)
			}
		} else if openLuckyBag == nil {
			// Not in cache, get from database
			openLuckyBag, err = chatDB.GetOpenLuckyBagByPinId(openItem.OpenPinId)
			if err == nil && openLuckyBag != nil {
				// Write data to cache
				cache_service.SetCacheOpenLuckyBag(groupId, openItem.OpenPinId, openLuckyBag)
			}
		}

		if err != nil || openLuckyBag == nil {
			continue
		}
		usedIndices[openLuckyBag.Index] = true
	}
	perfStats.processOpenListTime = time.Now().UnixMilli() - t

	t = time.Now().UnixMilli()
	// Add reclaimed lucky bag indices
	for _, residueItem := range residueList.Items {
		// First get residue lucky bag detail from cache
		residueLuckyBag, err := cache_service.GetCacheResidueLuckyBag(groupId, residueItem.ResiduePinId)
		if err != nil {
			log.Printf("[GetLuckyBagWithUnusedListV2] Cache get residue lucky bag error: %v", err)
			// Cache failed, get from database
			residueLuckyBag, err = chatDB.GetResidueLuckyBagByLuckyBagPinId(residueItem.ResiduePinId)
			if err == nil && residueLuckyBag != nil {
				// Write data to cache
				cache_service.SetCacheResidueLuckyBag(groupId, residueItem.ResiduePinId, residueLuckyBag)
			}
		} else if residueLuckyBag == nil {
			// Not in cache, get from database
			residueLuckyBag, err = chatDB.GetResidueLuckyBagByLuckyBagPinId(residueItem.ResiduePinId)
			if err == nil && residueLuckyBag != nil {
				// Write data to cache
				cache_service.SetCacheResidueLuckyBag(groupId, residueItem.ResiduePinId, residueLuckyBag)
			}
		}

		if err != nil || residueLuckyBag == nil {
			continue
		}
		if residueLuckyBag.UsedList != nil {
			for _, used := range residueLuckyBag.UsedList {
				usedIndices[used.Index] = true
			}
		}
	}
	perfStats.processResidueListTime = time.Now().UnixMilli() - t

	var tickInfo *respond.TickInfo
	if strings.ToLower(luckyBag.Type) == string(models.LuckyBagTypeMetacontractFT) && luckyBag.TickPinId != "" {
		// First get lucky bag extra from cache
		extra, err := cache_service.GetCacheLuckyBagExtra(luckyBag.TickTxId)
		if err != nil {
			log.Printf("[GetLuckyBagWithUnusedListV2] Cache get lucky bag extra error: %v", err)
			// Cache failed, get from database
			extra, err = extraDB.GetLuckyBagExtraByTxId(luckyBag.TickTxId)
			if err != nil {
				return nil, err
			}
			// Write data to cache
			if extra != nil {
				cache_service.SetCacheLuckyBagExtra(luckyBag.TickTxId, extra)
			}
		} else if extra == nil {
			// Not in cache, get from database
			extra, err = extraDB.GetLuckyBagExtraByTxId(luckyBag.TickTxId)
			if err != nil {
				return nil, err
			}
			// Write data to cache
			if extra != nil {
				cache_service.SetCacheLuckyBagExtra(luckyBag.TickTxId, extra)
			}
		}
		if extra != nil {
			tickInfo = &respond.TickInfo{
				TickType:   extra.Type,
				Codehash:   extra.Codehash,
				GenesisId:  extra.GenesisId,
				Genesis:    extra.Genesis,
				SensibleId: extra.SensibleId,
				Name:       extra.Name,
				Symbol:     extra.Symbol,
				Decimal:    extra.Decimal,
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
		Domain:              luckyBag.Domain,
		LuckyBagAddress:     luckyBag.LuckyBagAddress,
		LuckyBagGasAddress:  luckyBag.LuckyBagGasAddress,
		TickPinId:           luckyBag.TickPinId,
		TickTxId:            luckyBag.TickTxId,
		GenType:             luckyBag.GenType,
		GenState:            luckyBag.GenState,
		Amount:              luckyBag.Amount,
		LuckyTotalAmount:    luckyBag.LuckyTotalAmount,
		LuckyTotalFee:       luckyBag.LuckyTotalFee,
		FeeRate:             luckyBag.FeeRate,
		Count:               luckyBag.Count,
		ValidCount:          luckyBag.ValidCount,
		Content:             luckyBag.Content,
		Img:                 luckyBag.Img,
		ImgType:             luckyBag.ImgType,
		Unused:              make([]*respond.UnusedList, 0),
		Type:                luckyBag.Type,
		TickId:              luckyBag.TickId,
		TickInfo:            tickInfo,
		CollectionId:        luckyBag.CollectionId,
		TokenCount:          0, // TalkGroupLuckyBagV3 doesn't have TokenCount field
		RequireType:         luckyBag.RequireType,
		RequireTickId:       luckyBag.RequireTickId,
		RequireCollectionId: luckyBag.RequireCollectionId,
		LimitAmount:         luckyBag.LimitAmount,
	}
	if response.LuckyTotalAmount == "" || response.LuckyTotalAmount == "0" {
		response.LuckyTotalAmount = luckyBag.Amount
	}
	t = time.Now().UnixMilli()
	// Get unused UTXO list
	for _, v := range luckyBag.PayList {
		if !usedIndices[v.Index] {
			unused := &respond.UnusedList{
				Index:        v.Index,
				Amount:       v.Amount,
				Address:      v.Address,
				ScriptPubKey: "", // Need to get from LuckyBagVouts
				LuckyAmount:  v.LuckyAmount,
				LuckyFee:     v.LuckyFee,
				LuckyFeeRate: v.LuckyFeeRate,

				GasAmount:  v.GasAmount,
				GasAddress: v.GasAddress,
				GasIndex:   v.GasIndex,
			}
			if unused.LuckyAmount == "" || unused.LuckyAmount == "0" {
				unused.LuckyAmount = v.Amount
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
	perfStats.buildUnusedListTime = time.Now().UnixMilli() - t
	perfStats.totalTime = time.Since(startTime).Milliseconds()

	logger.Info("[LUCKY_BAG_SERVICE_V2][GET_LUCKY_BAG_WITH_UNUSED_LIST] Performance Stats - "+
		"Total: %dms, GetLuckyBag: %dms, GetOpenList: %dms, GetResidueList: %dms, "+
		"ProcessOpenList: %dms, ProcessResidueList: %dms, BuildUnusedList: %dms, Items: %d",
		perfStats.totalTime,
		perfStats.getLuckyBagTime,
		perfStats.getOpenListTime,
		perfStats.getResidueListTime,
		perfStats.processOpenListTime,
		perfStats.processResidueListTime,
		perfStats.buildUnusedListTime,
		len(response.Unused))

	return response, nil
}

// GrabLuckyBagV2 Grab lucky bag (V2 with cache)
func GrabLuckyBagV2(groupId, pinId, metaId, address string) (string, error) {
	startTime := time.Now()
	var perfStats = struct {
		getLuckyBagTime        int64
		checkUserInGroupTime   int64
		getOpenListTime        int64
		processOpenListTime    int64
		getResidueListTime     int64
		processResidueListTime int64
		getUnusedListTime      int64
		commonGrabTime         int64
		totalTime              int64
	}{}

	isAddressGloballyBlocked, _ := globalBlockDB.IsAddressGloballyLuckBagBlocked(address)
	// if err != nil {
	// 	return "", err
	// }
	if isAddressGloballyBlocked {
		return "", errors.New("address is blocked")
	}

	t := time.Now().UnixMilli()
	// 1. First get lucky bag object from cache
	luckyBag, err := cache_service.GetCacheLuckyBag(groupId, pinId)
	if err != nil {
		log.Printf("[GrabLuckyBagV2] Cache get error: %v", err)
		// Cache failed, get from database
		luckyBag, err = chatDB.GetLuckyBagByPinId(pinId)
		if err != nil {
			return "", err
		}
		if luckyBag == nil {
			return "", errors.New("lucky bag not found")
		}
		// Write data to cache
		cache_service.SetCacheLuckyBag(luckyBag)
	} else if luckyBag == nil {
		// Not in cache, get from database
		luckyBag, err = chatDB.GetLuckyBagByPinId(pinId)
		if err != nil {
			return "", err
		}
		if luckyBag == nil {
			return "", errors.New("lucky bag not found")
		}
		// Write data to cache
		cache_service.SetCacheLuckyBag(luckyBag)
	}
	perfStats.getLuckyBagTime = time.Now().UnixMilli() - t

	// Verify if groupId matches
	if luckyBag.GroupId != groupId {
		return "", errors.New("lucky bag not match")
	}

	t = time.Now().UnixMilli()
	// Check if user is in group
	isInGroup, err := chatDB.IsUserInGroup(metaId, groupId)
	if err != nil {
		return "", err
	}
	if !isInGroup {
		return "", errors.New("user not in group")
	}
	perfStats.checkUserInGroupTime = time.Now().UnixMilli() - t

	if luckyBag.Domain != "" && luckyBag.LuckyBagAddress != "" {
		if luckyBag.GenType == 2 {
			return "", errors.New("lucky bag is external")
		}
		// if strings.TrimSuffix(luckyBag.Domain, "/") != strings.TrimSuffix(common.Config.GroupChat.LuckyBagDomain, "/") {
		// 	return "", errors.New("lucky bag domain not match")
		// }
		if !checkLuckyBagDomain(luckyBag.Domain) {
			return "", errors.New("lucky bag domain not match")
		}
		if luckyBag.GenType == 1 && luckyBag.GenState != 1 {
			return "", errors.New("lucky bag is internal and failed")
		}
	}

	t = time.Now().UnixMilli()
	// 2. Get opened lucky bag list from cache
	openList, err := cache_service.GetCacheOpenLuckyBagList(groupId, pinId)
	if err != nil {
		log.Printf("[GrabLuckyBagV2] Cache get open list error: %v", err)
		// Cache failed, get from database
		openList, err = chatDB.GetOpenLuckyBagList(pinId)
		if err != nil {
			return "", err
		}
		// Write data to cache
		if openList != nil {
			cache_service.SetCacheOpenLuckyBagList(groupId, pinId, openList)
		}
	} else if openList == nil {
		// Not in cache, get from database
		openList, err = chatDB.GetOpenLuckyBagList(pinId)
		if err != nil {
			return "", err
		}
		// Write data to cache
		if openList != nil {
			cache_service.SetCacheOpenLuckyBagList(groupId, pinId, openList)
		}
	}
	perfStats.getOpenListTime = time.Now().UnixMilli() - t

	t = time.Now().UnixMilli()
	// Build claimed lucky bag list
	openRedList := make([]*OpenRedMetaId, 0)
	for _, v := range openList.Items {
		// First get open lucky bag detail from cache
		openLuckyBag, err := cache_service.GetCacheOpenLuckyBag(groupId, v.OpenPinId)
		if err != nil {
			log.Printf("[GrabLuckyBagV2] Cache get open lucky bag error: %v", err)
			// Cache failed, get from database
			openLuckyBag, err = chatDB.GetOpenLuckyBagByPinId(v.OpenPinId)
			if err == nil && openLuckyBag != nil {
				// Write data to cache
				cache_service.SetCacheOpenLuckyBag(groupId, v.OpenPinId, openLuckyBag)
			}
		} else if openLuckyBag == nil {
			// Not in cache, get from database
			openLuckyBag, err = chatDB.GetOpenLuckyBagByPinId(v.OpenPinId)
			if err == nil && openLuckyBag != nil {
				// Write data to cache
				cache_service.SetCacheOpenLuckyBag(groupId, v.OpenPinId, openLuckyBag)
			}
		}

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
	perfStats.processOpenListTime = time.Now().UnixMilli() - t

	t = time.Now().UnixMilli()
	// 3. Get reclaimed lucky bag list from cache
	residueRedEnvelopeList, err := cache_service.GetCacheResidueLuckyBagList(groupId, pinId)
	if err != nil {
		log.Printf("[GrabLuckyBagV2] Cache get residue list error: %v", err)
		// Cache failed, get from database
		residueRedEnvelopeList, err = chatDB.GetResidueLuckyBagList(pinId)
		if err != nil {
			return "", err
		}
		// Write data to cache
		if residueRedEnvelopeList != nil {
			cache_service.SetCacheResidueLuckyBagList(groupId, pinId, residueRedEnvelopeList)
		}
	} else if residueRedEnvelopeList == nil {
		// Not in cache, get from database
		residueRedEnvelopeList, err = chatDB.GetResidueLuckyBagList(pinId)
		if err != nil {
			return "", err
		}
		// Write data to cache
		if residueRedEnvelopeList != nil {
			cache_service.SetCacheResidueLuckyBagList(groupId, pinId, residueRedEnvelopeList)
		}
	}
	perfStats.getResidueListTime = time.Now().UnixMilli() - t

	t = time.Now().UnixMilli()
	if residueRedEnvelopeList != nil && len(residueRedEnvelopeList.Items) != 0 {
		for _, residueItem := range residueRedEnvelopeList.Items {
			// First get residue lucky bag detail from cache
			residueLuckyBag, err := cache_service.GetCacheResidueLuckyBag(groupId, residueItem.ResiduePinId)
			if err != nil {
				log.Printf("[GrabLuckyBagV2] Cache get residue lucky bag error: %v", err)
				// Cache failed, get from database
				residueLuckyBag, err = chatDB.GetResidueLuckyBagByLuckyBagPinId(residueItem.ResiduePinId)
				if err == nil && residueLuckyBag != nil {
					// Write data to cache
					cache_service.SetCacheResidueLuckyBag(groupId, residueItem.ResiduePinId, residueLuckyBag)
				}
			} else if residueLuckyBag == nil {
				// Not in cache, get from database
				residueLuckyBag, err = chatDB.GetResidueLuckyBagByLuckyBagPinId(residueItem.ResiduePinId)
				if err == nil && residueLuckyBag != nil {
					// Write data to cache
					cache_service.SetCacheResidueLuckyBag(groupId, residueItem.ResiduePinId, residueLuckyBag)
				}
			}

			if err != nil || residueLuckyBag == nil {
				continue
			}

			if len(residueLuckyBag.UsedList) == 0 {
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
	perfStats.processResidueListTime = time.Now().UnixMilli() - t

	t = time.Now().UnixMilli()
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

			LuckyAmount:  v.LuckyAmount,
			LuckyFee:     v.LuckyFee,
			LuckyFeeRate: v.LuckyFeeRate,

			GasAmount:  v.GasAmount,
			GasAddress: v.GasAddress,
			GasIndex:   v.GasIndex,
		}
		if strings.ToLower(luckyBag.Type) == string(models.LuckyBagTypeMetacontractFT) {
			gasAmount, err := strconv.ParseInt(unused.GasAmount, 10, 64)
			if err != nil {
				continue
			}
			if gasAmount < 20000 {
				continue // skip if gas amount is less than 20000
			}
		}
		unusedList = append(unusedList, unused)
	}
	perfStats.getUnusedListTime = time.Now().UnixMilli() - t

	if len(unusedList) <= 0 {
		return "", errors.New("LuckyBag had been all grab.")
	}

	// check type if metacontract-ft, check tickTxId and tickPinId
	if strings.ToLower(luckyBag.Type) == string(models.LuckyBagTypeMetacontractFT) {
		if luckyBag.TickTxId == "" || luckyBag.TickPinId == "" || luckyBag.TickId == "" {
			return "", errors.New("tickTxId and tickPinId and tickId are required for metacontract-ft")
		}
		// if strings.Contains(luckyBag.TickTxId, "i") {
		// 	luckyBag.TickTxId = strings.Split(luckyBag.TickTxId, "i")[0]
		// }

		// First get lucky bag extra from cache
		extra, err := cache_service.GetCacheLuckyBagExtra(luckyBag.TickTxId)
		if err != nil {
			log.Printf("[GrabLuckyBagV2] Cache get lucky bag extra error: %v", err)
			// Cache failed, get from database
			extra, err = extraDB.GetLuckyBagExtraByTxId(luckyBag.TickTxId)
			if err != nil {
				return "", err
			}
			// Write data to cache
			if extra != nil {
				cache_service.SetCacheLuckyBagExtra(luckyBag.TickTxId, extra)
			}
		} else if extra == nil {
			// Not in cache, get from database
			extra, err = extraDB.GetLuckyBagExtraByTxId(luckyBag.TickTxId)
			if err != nil {
				return "", err
			}
			// Write data to cache
			if extra != nil {
				cache_service.SetCacheLuckyBagExtra(luckyBag.TickTxId, extra)
			}
		}
		if extra == nil {
			return "", errors.New("extra not found")
		}
		if luckyBag.TickId != extra.Codehash+"/"+extra.Genesis {
			return "", errors.New("tickId not match")
		}

		err = checkGrpcHealth()
		if err != nil {
			return "", errors.New("grpc health check failed: " + err.Error())
		}
	}

	t = time.Now().UnixMilli()
	err = commonGrabV2(luckyBag, unusedList, metaId, address)
	if err != nil {
		return "", err
	}
	perfStats.commonGrabTime = time.Now().UnixMilli() - t
	perfStats.totalTime = time.Since(startTime).Milliseconds()

	logger.Info("[LUCKY_BAG_SERVICE_V2][GRAB_LUCKY_BAG] Performance Stats - "+
		"Total: %dms, GetLuckyBag: %dms, CheckUserInGroup: %dms, GetOpenList: %dms, "+
		"ProcessOpenList: %dms, GetResidueList: %dms, ProcessResidueList: %dms, "+
		"GetUnusedList: %dms, CommonGrab: %dms, Items: %d",
		perfStats.totalTime,
		perfStats.getLuckyBagTime,
		perfStats.checkUserInGroupTime,
		perfStats.getOpenListTime,
		perfStats.processOpenListTime,
		perfStats.getResidueListTime,
		perfStats.processResidueListTime,
		perfStats.getUnusedListTime,
		perfStats.commonGrabTime,
		len(unusedList))

	return "success", nil
}

// commonGrabV2 Execute common logic for grabbing lucky bags (V2 with cache)
func commonGrabV2(luckyBag *models.TalkGroupLuckyBagV3, unusedList []*respond.UnusedList, metaId, address string) error {
	startTime := time.Now()
	var perfStats = struct {
		lockTime                 int64
		cacheCheckTime           int64
		dbOperationsTime         int64
		getOpenLuckyBagTime      int64
		saveOpenLuckyBagTime     int64
		saveOpenLuckyBagListTime int64
		enqueueMessageTime       int64
		saveChatTime             int64
		totalTime                int64
	}{}

	type grabEntity struct {
		unusedIndex        int64
		unusedAmount       string
		unusedAddress      string
		tokenIndex         string
		unusedLuckyAmount  string
		unusedLuckyFee     string
		unusedLuckyFeeRate string
		gasAmount          string
		gasAddress         string
		gasIndex           int64
	}
	grabEntityList := make([]*grabEntity, 0)
	has := false

	t := time.Now().UnixMilli()
	// Use lucky bag-specific mutex to prevent concurrent grabbing of the same lucky bag
	luckyBagMutex := getLuckyBagGrabMutex(luckyBag.PinId)
	luckyBagMutex.Lock()
	defer luckyBagMutex.Unlock()
	perfStats.lockTime = time.Now().UnixMilli() - t

	t = time.Now().UnixMilli()
	for _, unused := range unusedList {
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
					unusedIndex:        unused.Index,
					unusedAmount:       unused.Amount,
					unusedAddress:      unused.Address,
					tokenIndex:         "",
					unusedLuckyAmount:  unused.LuckyAmount,
					unusedLuckyFee:     unused.LuckyFee,
					unusedLuckyFeeRate: unused.LuckyFeeRate,
					gasAmount:          unused.GasAmount,
					gasAddress:         unused.GasAddress,
					gasIndex:           unused.GasIndex,
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
					unusedIndex:        unused.Index,
					unusedAmount:       unused.Amount,
					unusedAddress:      unused.Address,
					tokenIndex:         "",
					unusedLuckyAmount:  unused.LuckyAmount,
					unusedLuckyFee:     unused.LuckyFee,
					unusedLuckyFeeRate: unused.LuckyFeeRate,
					gasAmount:          unused.GasAmount,
					gasAddress:         unused.GasAddress,
					gasIndex:           unused.GasIndex,
				})
				break
			}
		}
	}
	perfStats.cacheCheckTime = time.Now().UnixMilli() - t

	if !has {
		return errors.New("LuckyBag had been all grab.")
	}

	t = time.Now().UnixMilli()
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
		tn1 := time.Now().UnixMilli()
		existingOpen, err := chatDB.GetOpenLuckyBagByPinId(pinId)
		if err == nil && existingOpen != nil {
			// Already exists, skip processing
			return errors.New("already grab")
		}
		tn2 := time.Now().UnixMilli()
		perfStats.getOpenLuckyBagTime = tn2 - tn1

		// Create grab lucky bag record
		openLuckyBag := &models.TalkGroupOpenLuckyBagV3{
			CommunityId:        "", // Need to get from group info
			GroupId:            luckyBag.GroupId,
			TxId:               txId,
			PinId:              pinId,
			MetaId:             metaId,
			Protocol:           "/protocol/simplegroupopenLuckybag",
			SubId:              luckyBag.SubId,
			Code:               luckyBag.Code,
			CreateTimeStr:      luckyBag.CreateTimeStr,
			Domain:             luckyBag.Domain,
			LuckyBagAddress:    luckyBag.LuckyBagAddress,
			LuckyBagGasAddress: luckyBag.LuckyBagGasAddress,
			TickPinId:          luckyBag.TickPinId,
			TickTxId:           luckyBag.TickTxId,
			GenType:            luckyBag.GenType,
			GenState:           luckyBag.GenState,
			Address:            address,
			Index:              v.unusedIndex,
			Amount:             v.unusedAmount,
			LuckyAmount:        v.unusedLuckyAmount,
			LuckyFee:           v.unusedLuckyFee,
			LuckyFeeRate:       v.unusedLuckyFeeRate,
			PkScript:           pkScript,
			Vins:               vins,

			GasAmount:   v.gasAmount,
			GasAddress:  v.gasAddress,
			GasIndex:    v.gasIndex,
			GasPkScript: "",

			Type:                luckyBag.Type,
			TickId:              luckyBag.TickId,
			CollectionId:        luckyBag.CollectionId,
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
		tn3 := time.Now().UnixMilli()
		err = chatDB.SaveOpenLuckyBag(openLuckyBag)
		if err != nil {
			log.Printf("SaveOpenLuckyBag err: %v", err)
			continue
		}
		tn4 := time.Now().UnixMilli()
		perfStats.saveOpenLuckyBagTime = tn4 - tn3

		// Save grab lucky bag list record to TalkGroupOpenLuckyBagListCollection
		// use optimistic lock mechanism to ensure atomicity, no distributed lock
		tn5 := time.Now().UnixMilli()
		err = chatDB.SaveOpenLuckyBagListAtomic(luckyBag.PinId, openLuckyBag.PinId, luckyBag.GroupId, openLuckyBag.Timestamp, metaId, address, v.unusedIndex)
		if err != nil {
			log.Printf("SaveOpenLuckyBagList err: %v", err)
			continue
		}
		tn6 := time.Now().UnixMilli()
		perfStats.saveOpenLuckyBagListTime = tn6 - tn5

		// Update open list in cache
		// Re-fetch latest open list from database and update cache
		updatedOpenList, err := chatDB.GetOpenLuckyBagList(luckyBag.PinId)
		if err == nil && updatedOpenList != nil {
			cache_service.SetCacheOpenLuckyBagList(luckyBag.GroupId, luckyBag.PinId, updatedOpenList)
		}

		// Add grab lucky bag record to queue to be processed
		tn7 := time.Now().UnixMilli()
		err = chatDB.EnqueueOpenLuckyBagMessage(openLuckyBag)
		if err != nil {
			log.Printf("EnqueueOpenLuckyBagMessage err: %v", err)
			continue
		}
		tn8 := time.Now().UnixMilli()
		perfStats.enqueueMessageTime = tn8 - tn7

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
		tn9 := time.Now().UnixMilli()
		err = chatDB.SaveChat(chat)
		if err != nil {
			return err
		}
		tn10 := time.Now().UnixMilli()
		perfStats.saveChatTime = tn10 - tn9

		hasSuccess = true
		break // Only process one lucky bag
	}
	perfStats.dbOperationsTime = time.Now().UnixMilli() - t
	perfStats.totalTime = time.Since(startTime).Milliseconds()

	logger.Info("[LUCKY_BAG_SERVICE_V2][COMMON_GRAB_V2] Performance Stats - "+
		"Total: %dms, Lock: %dms, CacheCheck: %dms, DBOperations: %dms, "+
		"GetOpenLuckyBag: %dms, SaveOpenLuckyBag: %dms, SaveOpenLuckyBagList: %dms, "+
		"EnqueueMessage: %dms, SaveChat: %dms, Items: %d",
		perfStats.totalTime,
		perfStats.lockTime,
		perfStats.cacheCheckTime,
		perfStats.dbOperationsTime,
		perfStats.getOpenLuckyBagTime,
		perfStats.saveOpenLuckyBagTime,
		perfStats.saveOpenLuckyBagListTime,
		perfStats.enqueueMessageTime,
		perfStats.saveChatTime,
		len(grabEntityList))

	if !hasSuccess {
		return errors.New("Grab err.")
	}

	return nil
}

func checkGrpcHealth() error {
	grpcClient, err := grpc_metacontract.NewClient()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	healthResponse, err := grpcClient.GetHealth(ctx)
	if err != nil {
		return err
	}
	if healthResponse.Status != "success" {
		return errors.New("grpc health check failed")
	}
	return nil
}
