package service

import (
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
)

// GetLuckyBagWithOpenListV2 Get lucky bag object and claimed list by groupId and pinId (V2 with cache)
func GetLuckyBagWithOpenListV2(groupId, pinId string) (*respond.LuckyBagInfoResponse, error) {
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

	// Verify if groupId matches
	if luckyBag.GroupId != groupId {
		return nil, errors.New("lucky bag not match")
	}

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
	// Convert PayList - Note that ProInfoPayList has fewer fields, need to fill default values
	for _, payItem := range luckyBag.PayList {
		infoPayList := &respond.InfoPayList{
			TxId:         luckyBag.TxId,
			Index:        payItem.Index,
			Amount:       payItem.Amount,
			Address:      payItem.Address,
			Used:         false,
			LuckyAmount:  payItem.LuckyAmount,
			LuckyFee:     payItem.LuckyFee,
			LuckyFeeRate: payItem.LuckyFeeRate,
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
		}

		// Check reclaimed lucky bags
		if residueList != nil {
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
		}

		response.PayList = append(response.PayList, infoPayList)
	}
	response.UsedCount = strconv.Itoa(usedCount)

	return response, nil
}

// GetLuckyBagWithUnusedListV2 Get lucky bag object and unclaimed list by groupId and pinId (V2 with cache)
func GetLuckyBagWithUnusedListV2(groupId, pinId string) (*respond.LuckyBagUnusedResponse, error) {
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

	// Verify if groupId matches
	if luckyBag.GroupId != groupId {
		return nil, errors.New("lucky bag not match")
	}

	if luckyBag.Domain != "" && luckyBag.LuckyBagAddress != "" {
		if luckyBag.GenType == 2 {
			return nil, errors.New("lucky bag is external")
		}
		if strings.TrimSuffix(luckyBag.Domain, "/") != strings.TrimSuffix(common.Config.GroupChat.LuckyBagDomain, "/") {
			return nil, errors.New("lucky bag domain not match")
		}
		if luckyBag.GenType == 1 && luckyBag.GenState != 1 {
			return nil, errors.New("lucky bag is internal and failed")
		}
	}

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

	// Build used UTXO index set
	usedIndices := make(map[int64]bool)

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
		TokenCount:          0, // TalkGroupLuckyBagV3 doesn't have TokenCount field
		RequireType:         luckyBag.RequireType,
		RequireTickId:       luckyBag.RequireTickId,
		RequireCollectionId: luckyBag.RequireCollectionId,
		LimitAmount:         luckyBag.LimitAmount,
	}
	if response.LuckyTotalAmount == "" || response.LuckyTotalAmount == "0" {
		response.LuckyTotalAmount = luckyBag.Amount
	}
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

	return response, nil
}

// GrabLuckyBagV2 Grab lucky bag (V2 with cache)
func GrabLuckyBagV2(groupId, pinId, metaId, address string) (string, error) {
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

	if luckyBag.Domain != "" && luckyBag.LuckyBagAddress != "" {
		if luckyBag.GenType == 2 {
			return "", errors.New("lucky bag is external")
		}
		if strings.TrimSuffix(luckyBag.Domain, "/") != strings.TrimSuffix(common.Config.GroupChat.LuckyBagDomain, "/") {
			return "", errors.New("lucky bag domain not match")
		}
		if luckyBag.GenType == 1 && luckyBag.GenState != 1 {
			return "", errors.New("lucky bag is internal and failed")
		}
	}

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
			Index:        v.Index,
			Amount:       v.Amount,
			Address:      v.Address,
			LuckyAmount:  v.LuckyAmount,
			LuckyFee:     v.LuckyFee,
			LuckyFeeRate: v.LuckyFeeRate,
		}
		unusedList = append(unusedList, unused)
	}

	if len(unusedList) <= 0 {
		return "", errors.New("LuckyBag had been all grab.")
	}

	err = commonGrabV2(luckyBag, unusedList, metaId, address)
	if err != nil {
		return "", err
	}

	return "success", nil
}

// commonGrabV2 Execute common logic for grabbing lucky bags (V2 with cache)
func commonGrabV2(luckyBag *models.TalkGroupLuckyBagV3, unusedList []*respond.UnusedList, metaId, address string) error {
	type grabEntity struct {
		unusedIndex        int64
		unusedAmount       string
		unusedAddress      string
		tokenIndex         string
		unusedLuckyAmount  string
		unusedLuckyFee     string
		unusedLuckyFeeRate string
	}
	grabEntityList := make([]*grabEntity, 0)
	has := false

	t := time.Now().UnixMilli()
	// Use lucky bag-specific mutex to prevent concurrent grabbing of the same lucky bag
	luckyBagMutex := getLuckyBagGrabMutex(luckyBag.PinId)
	luckyBagMutex.Lock()
	defer luckyBagMutex.Unlock()
	log.Printf("[commonGrabV2] luckyBagPinId: %s, metaId: %s, address: %s [Lock] %d", luckyBag.PinId, metaId, address, t)

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
				})
				break
			}
		}
	}

	if !has {
		log.Printf("[commonGrabV2] luckyBagPinId: %s, metaId: %s, address: %s [UnLock]NoLuckyBag [%d]", luckyBag.PinId, metaId, address, time.Now().UnixMilli()-t)
		return errors.New("LuckyBag had been all grab.")
	}

	log.Printf("[commonGrabV2] luckyBagPinId: %s, metaId: %s, address: %s [UnLock]Success [%d]", luckyBag.PinId, metaId, address, time.Now().UnixMilli()-t)

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
			Domain:              luckyBag.Domain,
			LuckyBagAddress:     luckyBag.LuckyBagAddress,
			GenType:             luckyBag.GenType,
			GenState:            luckyBag.GenState,
			Address:             address,
			Index:               v.unusedIndex,
			Amount:              v.unusedAmount,
			LuckyAmount:         v.unusedLuckyAmount,
			LuckyFee:            v.unusedLuckyFee,
			LuckyFeeRate:        v.unusedLuckyFeeRate,
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
		// use optimistic lock mechanism to ensure atomicity, no distributed lock
		err = chatDB.SaveOpenLuckyBagListAtomic(luckyBag.PinId, openLuckyBag.PinId, luckyBag.GroupId, openLuckyBag.Timestamp, metaId, address, v.unusedIndex)
		if err != nil {
			log.Printf("SaveOpenLuckyBagList err: %v", err)
			continue
		}

		// Update open list in cache
		// Re-fetch latest open list from database and update cache
		updatedOpenList, err := chatDB.GetOpenLuckyBagList(luckyBag.PinId)
		if err == nil && updatedOpenList != nil {
			cache_service.SetCacheOpenLuckyBagList(luckyBag.GroupId, luckyBag.PinId, updatedOpenList)
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

		hasSuccess = true
		break // Only process one lucky bag
	}

	if !hasSuccess {
		return errors.New("Grab err.")
	}

	return nil
}
