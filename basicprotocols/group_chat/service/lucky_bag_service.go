package service

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"manindexer/basicprotocols/group_chat/api/respond"
	"manindexer/basicprotocols/group_chat/common_util/logger"
	"manindexer/basicprotocols/group_chat/db"
	"manindexer/basicprotocols/group_chat/models"
	"manindexer/basicprotocols/group_chat/service/cache_service"
	"manindexer/basicprotocols/group_chat/service/common_service"
	"manindexer/common"
	"strconv"
	"strings"
	"sync"
	"time"

	chaincfg2 "github.com/bitcoinsv/bsvd/chaincfg"
	wire2 "github.com/bitcoinsv/bsvd/wire"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/cockroachdb/pebble"
	"github.com/libsv/go-bk/bec"
	"github.com/tyler-smith/go-bip32"
)

// UpdateLuckyBagCacheAfterSave Update lucky bag cache after saving to database
func UpdateLuckyBagCacheAfterSave(grabEntity *models.TalkGroupOpenLuckyBagV3, reclaimEntity *models.TalkGroupResidueLuckyBagV3) error {
	// Update individual open lucky bag cache if grabEntity is provided
	if grabEntity != nil {
		// Check if the open lucky bag exists in cache before updating
		_, err := cache_service.GetCacheOpenLuckyBag(grabEntity.GroupId, grabEntity.PinId)
		if err == nil {
			// Cache exists, update it
			_, err = cache_service.SetCacheOpenLuckyBag(grabEntity.GroupId, grabEntity.PinId, grabEntity)
			if err != nil {
				log.Printf("[UpdateLuckyBagCacheAfterSave] Failed to update open lucky bag cache for %s: %v", grabEntity.PinId, err)
			}
		}
	}

	// Update individual residue lucky bag cache if reclaimEntity is provided
	if reclaimEntity != nil {
		// Check if the residue lucky bag exists in cache before updating
		_, err := cache_service.GetCacheResidueLuckyBag(reclaimEntity.GroupId, reclaimEntity.PinId)
		if err == nil {
			// Cache exists, update it
			_, err = cache_service.SetCacheResidueLuckyBag(reclaimEntity.GroupId, reclaimEntity.PinId, reclaimEntity)
			if err != nil {
				log.Printf("[UpdateLuckyBagCacheAfterSave] Failed to update residue lucky bag cache for %s: %v", reclaimEntity.PinId, err)
			}
		}
	}

	return nil
}

// luckyBagGrabMutexItem lock item for lucky bag grab operations
type luckyBagGrabMutexItem struct {
	mutex       *sync.Mutex
	lastUsed    time.Time
	accessCount int64
}

// Global lucky bag grab mutex map
var luckyBagGrabMutexMap sync.Map

// getLuckyBagGrabMutex get or create lucky bag grab mutex
func getLuckyBagGrabMutex(luckyBagPinId string) *sync.Mutex {
	// try to get existing lock from sync.Map
	if value, exists := luckyBagGrabMutexMap.Load(luckyBagPinId); exists {
		if item, ok := value.(*luckyBagGrabMutexItem); ok {
			// update access statistics
			item.lastUsed = time.Now()
			item.accessCount++
			return item.mutex
		}
	}

	// if not exists, create new lock item
	newItem := &luckyBagGrabMutexItem{
		mutex:       &sync.Mutex{},
		lastUsed:    time.Now(),
		accessCount: 1,
	}

	// use LoadOrStore to ensure atomicity, avoid duplicate creation
	if value, loaded := luckyBagGrabMutexMap.LoadOrStore(luckyBagPinId, newItem); loaded {
		// if already exists, return existing lock and update statistics
		if item, ok := value.(*luckyBagGrabMutexItem); ok {
			item.lastUsed = time.Now()
			item.accessCount++
			return item.mutex
		}
	}

	// return new created lock
	return newItem.mutex
}

// cleanupUnusedLuckyBagGrabLocks cleanup unused lucky bag grab locks
func cleanupUnusedLuckyBagGrabLocks() {
	now := time.Now()
	cleanupThreshold := 30 * time.Minute // 30 minutes not used to clean up

	var keysToDelete []string

	// traverse all locks, find locks to clean up
	luckyBagGrabMutexMap.Range(func(key, value interface{}) bool {
		if item, ok := value.(*luckyBagGrabMutexItem); ok {
			// check if it exceeds the cleanup threshold
			if now.Sub(item.lastUsed) > cleanupThreshold {
				keysToDelete = append(keysToDelete, key.(string))
			}
		}
		return true
	})

	// delete unused locks
	for _, key := range keysToDelete {
		luckyBagGrabMutexMap.Delete(key)
	}

	if len(keysToDelete) > 0 {
		log.Printf("Cleaned up %d unused lucky bag grab locks", len(keysToDelete))
	}
}

// startLuckyBagGrabCleanupGoroutine start cleanup goroutine for lucky bag grab locks
func startLuckyBagGrabCleanupGoroutine() {
	go func() {
		ticker := time.NewTicker(5 * time.Minute) // every 5 minutes
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				cleanupUnusedLuckyBagGrabLocks()
			}
		}
	}()
}

// GetLuckyBagWithOpenList Get lucky bag object and claimed list by groupId and pinId
func GetLuckyBagWithOpenList(groupId, pinId string) (*respond.LuckyBagInfoResponse, error) {
	// 性能监控：记录开始时间
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
	// Get lucky bag object
	luckyBag, err := chatDB.GetLuckyBagByPinId(pinId)
	if err != nil {
		return nil, err
	}
	perfStats.getLuckyBagTime = time.Now().UnixMilli() - t

	if luckyBag == nil {
		return nil, errors.New("lucky bag not found")
	}

	// Verify if groupId matches
	if luckyBag.GroupId != groupId {
		return nil, errors.New("lucky bag not match")
	}

	t = time.Now().UnixMilli()
	// Get claimed lucky bag list
	openList, err := chatDB.GetOpenLuckyBagList(pinId)
	if err != nil {
		return nil, err
	}
	perfStats.getOpenListTime = time.Now().UnixMilli() - t

	t = time.Now().UnixMilli()
	// Get reclaimed lucky bag list
	residueList, err := chatDB.GetResidueLuckyBagList(pinId)
	if err != nil {
		return nil, err
	}
	perfStats.getResidueListTime = time.Now().UnixMilli() - t

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
			LuckyAmount:  payItem.LuckyAmount,
			LuckyFee:     payItem.LuckyFee,
			LuckyFeeRate: payItem.LuckyFeeRate,
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
		if infoPayList.LuckyAmount == "" || infoPayList.LuckyAmount == "0" {
			infoPayList.LuckyAmount = payItem.Amount
		}
		// Check claimed lucky bags
		if openList != nil {
			t = time.Now().UnixMilli()
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
			perfStats.processOpenListTime += time.Now().UnixMilli() - t
		}

		// Check reclaimed lucky bags
		if residueList != nil {
			t = time.Now().UnixMilli()
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
							if residueLuckyBag.ReclaimState == models.GrabStateReclaimAndSend || residueLuckyBag.ReclaimState == models.GrabStateChain {
								infoPayList.IsWithdraw = true
							}
							usedCount++
						}
					}
				}
			}
			perfStats.processResidueListTime += time.Now().UnixMilli() - t
		}

		response.PayList = append(response.PayList, infoPayList)
	}
	response.UsedCount = strconv.Itoa(usedCount)
	perfStats.responseFormatTime = time.Now().UnixMilli() - startTime.UnixMilli()
	perfStats.totalTime = time.Since(startTime).Milliseconds()

	// 统一性能日志记录
	logger.Info("[LUCKY_BAG_SERVICE][GET_LUCKY_BAG_WITH_OPEN_LIST] Performance Stats - "+
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

// GetLuckyBagWithUnusedList Get lucky bag object and unclaimed list by groupId and pinId
func GetLuckyBagWithUnusedList(groupId, pinId string) (*respond.LuckyBagUnusedResponse, error) {
	// 性能监控：记录开始时间
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
	// Get lucky bag object
	luckyBag, err := chatDB.GetLuckyBagByPinId(pinId)
	if err != nil {
		return nil, err
	}
	perfStats.getLuckyBagTime = time.Now().UnixMilli() - t
	if luckyBag == nil {
		return nil, errors.New("lucky bag not found")
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

	t = time.Now().UnixMilli()
	// Get claimed lucky bag list
	openList, err := chatDB.GetOpenLuckyBagList(pinId)
	if err != nil {
		return nil, err
	}
	perfStats.getOpenListTime = time.Now().UnixMilli() - t

	t = time.Now().UnixMilli()
	// Get reclaimed lucky bag list
	residueList, err := chatDB.GetResidueLuckyBagList(pinId)
	if err != nil {
		return nil, err
	}
	perfStats.getResidueListTime = time.Now().UnixMilli() - t
	// Build used UTXO index set
	usedIndices := make(map[int64]bool)

	t = time.Now().UnixMilli()
	// Add claimed lucky bag indices
	for _, openItem := range openList.Items {
		openLuckyBag, err := chatDB.GetOpenLuckyBagByPinId(openItem.OpenPinId)
		if err != nil || openLuckyBag == nil {
			continue
		}
		usedIndices[openLuckyBag.Index] = true
	}
	perfStats.processOpenListTime = time.Now().UnixMilli() - t

	t = time.Now().UnixMilli()
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
	perfStats.processResidueListTime = time.Now().UnixMilli() - t

	t = time.Now().UnixMilli()
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
	perfStats.buildUnusedListTime = time.Now().UnixMilli() - t
	perfStats.totalTime = time.Since(startTime).Milliseconds()

	// 统一性能日志记录
	logger.Info("[LUCKY_BAG_SERVICE][GET_LUCKY_BAG_WITH_UNUSED_LIST] Performance Stats - "+
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
	// 性能监控：记录开始时间
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

	isAddressGloballyBlocked, _ := globalBlockDB.IsAddressGloballyBlocked(address)
	// if err != nil {
	// 	return "", err
	// }
	if isAddressGloballyBlocked {
		return "", errors.New("address is blocked")
	}

	t := time.Now().UnixMilli()
	luckyBag, err := chatDB.GetLuckyBagByPinId(pinId)
	if err != nil {
		return "", err
	}
	perfStats.getLuckyBagTime = time.Now().UnixMilli() - t
	if luckyBag == nil {
		return "", errors.New("lucky bag not found")
	}

	// Verify if groupId matches
	if luckyBag.GroupId != groupId {
		return "", errors.New("lucky bag not match")
	}

	// if address == "1HYDwLC4myjVDtddUm7rcNB1FDwePwthaD" {
	// 	return "", errors.New("error")
	// }

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
		if strings.TrimSuffix(luckyBag.Domain, "/") != strings.TrimSuffix(common.Config.GroupChat.LuckyBagDomain, "/") {
			return "", errors.New("lucky bag domain not match")
		}
		if luckyBag.GenType == 1 && luckyBag.GenState != 1 {
			return "", errors.New("lucky bag is internal and failed")
		}
	}

	t = time.Now().UnixMilli()
	// Get claimed lucky bag list
	openList, err := chatDB.GetOpenLuckyBagList(pinId)
	if err != nil {
		return "", err
	}
	perfStats.getOpenListTime = time.Now().UnixMilli() - t

	t = time.Now().UnixMilli()
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
	perfStats.processOpenListTime = time.Now().UnixMilli() - t

	t = time.Now().UnixMilli()
	// Get reclaimed lucky bag list
	residueRedEnvelopeList, err := chatDB.GetResidueLuckyBagList(pinId)
	if err != nil {
		return "", err
	}
	perfStats.getResidueListTime = time.Now().UnixMilli() - t

	t = time.Now().UnixMilli()
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
			Index:        v.Index,
			Amount:       v.Amount,
			Address:      v.Address,
			LuckyAmount:  v.LuckyAmount,
			LuckyFee:     v.LuckyFee,
			LuckyFeeRate: v.LuckyFeeRate,
		}
		unusedList = append(unusedList, unused)
	}
	perfStats.getUnusedListTime = time.Now().UnixMilli() - t

	if len(unusedList) <= 0 {
		return "", errors.New("LuckyBag had been all grab.")
	}

	t = time.Now().UnixMilli()
	err = commonGrab(luckyBag, unusedList, metaId, address)
	if err != nil {
		return "", err
	}
	perfStats.commonGrabTime = time.Now().UnixMilli() - t
	perfStats.totalTime = time.Since(startTime).Milliseconds()

	// 统一性能日志记录
	logger.Info("[LUCKY_BAG_SERVICE][GRAB_LUCKY_BAG] Performance Stats - "+
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

func commonGrab(luckyBag *models.TalkGroupLuckyBagV3, unusedList []*respond.UnusedList, metaId, address string) error {
	// 性能监控：记录开始时间
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

		tn1 := time.Now().UnixMilli()
		// Check if this PinId has already been saved
		existingOpen, err := chatDB.GetOpenLuckyBagByPinId(pinId)
		if err == nil && existingOpen != nil {
			// Already exists, skip processing
			return errors.New("already grab")
		}
		tn2 := time.Now().UnixMilli()
		perfStats.getOpenLuckyBagTime = tn2 - tn1
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

		tn3 := time.Now().UnixMilli()

		// Save grab lucky bag record to TalkGroupOpenLuckyBagPinCollection
		err = chatDB.SaveOpenLuckyBag(openLuckyBag)
		if err != nil {
			log.Printf("SaveOpenLuckyBag err: %v", err)
			continue
		}
		tn4 := time.Now().UnixMilli()
		perfStats.saveOpenLuckyBagTime = tn4 - tn3

		// Save grab lucky bag list record to TalkGroupOpenLuckyBagListCollection
		// use optimistic lock mechanism to ensure atomicity, no distributed lock
		err = chatDB.SaveOpenLuckyBagListAtomic(luckyBag.PinId, openLuckyBag.PinId, luckyBag.GroupId, openLuckyBag.Timestamp, metaId, address, v.unusedIndex)
		if err != nil {
			log.Printf("SaveOpenLuckyBagList err: %v", err)
			continue
		}
		tn5 := time.Now().UnixMilli()
		perfStats.saveOpenLuckyBagListTime = tn5 - tn4

		tn6 := time.Now().UnixMilli()
		// Add grab lucky bag record to queue to be processed
		err = chatDB.EnqueueOpenLuckyBagMessage(openLuckyBag)
		if err != nil {
			log.Printf("EnqueueOpenLuckyBagMessage err: %v", err)
			continue
		}
		tn7 := time.Now().UnixMilli()
		perfStats.enqueueMessageTime = tn7 - tn6

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

		tn8 := time.Now().UnixMilli()
		// Save chat message to TalkGroupChatPinCollection
		err = chatDB.SaveChat(chat)
		if err != nil {
			return err
		}
		tn9 := time.Now().UnixMilli()
		perfStats.saveChatTime = tn9 - tn8

		hasSuccess = true
		break // Only process one lucky bag
	}
	perfStats.dbOperationsTime = time.Now().UnixMilli() - t
	perfStats.totalTime = time.Since(startTime).Milliseconds()

	// 统一性能日志记录
	logger.Info("[LUCKY_BAG_SERVICE][COMMON_GRAB] Performance Stats - "+
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

	if grabEntity.Address == "1HYDwLC4myjVDtddUm7rcNB1FDwePwthaD" {
		fmt.Printf("[LuckyBag] black list: %s\n", grabEntity.Address)
		return errors.New("black list")
	}

	// Check and clean up duplicate grab records for the same address
	shouldProcess, err := cleanupDuplicateGrabRecords(grabEntity)
	if err != nil {
		log.Printf("[disposingGrabLuckyBag] cleanupDuplicateGrabRecords err: %v", err)
	}
	if !shouldProcess {
		log.Printf("[disposingGrabLuckyBag] cleanupDuplicateGrabRecords shouldProcess: %v", shouldProcess)
		return nil
	}

	_ = grabEntity.Vins[0] // utxo, temporarily unused
	_, hexStr := getLuckyBagKey(grabEntity.SubId, grabEntity.Code, grabEntity.CreateTimeStr, grabEntity.LuckyBagAddress, grabEntity.GenType)
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

	// txFeeRate := int64(1)
	// if grabEntity.LuckyFeeRate != "" && grabEntity.LuckyFeeRate != "0" {
	// 	normalizedAmount = normalizeScientificNotation(grabEntity.LuckyFeeRate)
	// 	feeRate, err := strconv.ParseInt(normalizedAmount, 10, 64)
	// 	if err != nil {
	// 		return fmt.Errorf("failed to parse lucky fee rate: %v", err)
	// 	}
	// 	if feeRate >= 1 {
	// 		txFeeRate = feeRate
	// 	}
	// }

	outputAmount := int64(0)
	if grabEntity.LuckyAmount != "" && grabEntity.LuckyAmount != "0" {
		gradNormalizedAmount := normalizeScientificNotation(grabEntity.LuckyAmount)
		outValue, err := strconv.ParseUint(gradNormalizedAmount, 10, 64)
		if err != nil {
			return fmt.Errorf("failed to parse lucky amount: %v", err)
		}
		if outValue >= 546 {
			outputAmount = int64(outValue)
		}
	}

	input := common.TxInputUtxo{
		TxId:     grabEntity.LuckyBagTxId,
		TxIndex:  int64(grabEntity.Index),
		PkScript: grabEntity.PkScript,
		Amount:   value,
		PriHex:   hexStr,
		SignMode: common.SignModeLegacy,
	}
	output := common.TxOutput{
		Address:    toAddress,
		Amount:     int64(value),
		NeedAmount: int64(outputAmount),
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
	// if grabEntity.Chain == "btc" {
	// 	time.Sleep(1 * time.Second)
	// }

	// Broadcast transaction with retry mechanism
	resultTxId, broadcastErr := chainAdapter[grabEntity.Chain].BroadcastTx(txRaw)
	if resultTxId != "" {
		// Success case
		grabEntity.GrabState = models.GrabStateOpenAndSend
		grabEntity.GrabTxId = resultTxId
		grabEntity.GrabMsg = "success"
		// grabEntity.RetryCount = 0 // Reset retry count on success
		log.Printf("[Grad][%s]Success broadcast tx: %s, totalCount: %d", grabEntity.Chain, resultTxId, totalCount)

		if grabEntity.Chain == "btc" {
			net := ""
			if common.TestNet == "1" {
				net = "testnet"
			}
			_ = net
			// common_service.BroadcastTx(net, txRaw)
		}
	} else {
		// Check if error is "too-long-mempool-chain" - skip retry counting for this error
		if strings.Contains(strings.ToLower(broadcastErr.Error()), "too-long-mempool-chain") {
			log.Printf("[Grad][%s]Too-long-mempool-chain error, skipping retry count increment: %s, totalCount: %d", grabEntity.Chain, broadcastErr.Error(), totalCount)
			// Don't increment retry count, just log and continue
			grabEntity.GrabMsg = fmt.Sprintf("Too-long-mempool-chain error, will retry later: %s", broadcastErr.Error())

			// Update grab lucky bag record in database
			err = chatDB.SaveOpenLuckyBag(grabEntity)
			if err != nil {
				return fmt.Errorf("failed to save open lucky bag: %v", err)
			}

			// Return nil to continue processing in next cycle
			return fmt.Errorf("too-long-mempool-chain error, skipping retry count increment: %s", broadcastErr.Error())
		}

		// Failure case - increment retry count for other errors
		grabEntity.RetryCount++
		log.Printf("[Grad][%s]Failure broadcast tx: %s, retryCount: %d, totalCount: %d", grabEntity.Chain, broadcastErr.Error(), grabEntity.RetryCount, totalCount)

		// Update queue message with new retry count
		err = chatDB.UpdateOpenLuckyBagQueueMessage(grabEntity.PinId, grabEntity.RetryCount, "")
		if err != nil {
			log.Printf("[Grad][%s]Failed to update queue message retry count: %v", grabEntity.Chain, err)
		}

		// Check if we've exceeded maximum retry attempts
		if grabEntity.RetryCount >= 5 {
			// Max retries exceeded, record error and save to error collection
			grabEntity.GrabState = models.GrabStateOpenAndSendErr
			grabEntity.GrabMsg = fmt.Sprintf("Max retries exceeded (5), last error: %s", broadcastErr.Error())
			grabEntity.GrabTxRaw = txRaw

			// Save to error collection
			err = chatDB.SaveOpenLuckyBagError(grabEntity.PinId, grabEntity.LuckyBagPinId)
			if err != nil {
				log.Printf("[Grad][%s]Failed to save open lucky bag error: %v", grabEntity.Chain, err)
			} else {
				log.Printf("[Grad][%s]Saved open lucky bag error record for pinId: %s", grabEntity.Chain, grabEntity.PinId)
			}
		} else {
			fmt.Printf("[Grad][%s][%s]Retry broadcast tx: %s, retryCount: %d, totalCount: %d\n", grabEntity.Chain, grabEntity.PinId, broadcastErr.Error(), grabEntity.RetryCount, totalCount)
			// Still within retry limit, set error state but don't save to error collection yet
			// grabEntity.GrabState = models.GrabStateOpenAndSendErr
			grabEntity.GrabMsg = fmt.Sprintf("Retry %d/5, error: %s", grabEntity.RetryCount, broadcastErr.Error())

			// Update grab lucky bag record in database
			err = chatDB.SaveOpenLuckyBag(grabEntity)
			if err != nil {
				return fmt.Errorf("failed to save open lucky bag: %v", err)
			}

			// Check if error contains broadcast-related issues, if not, return the error
			if !isBroadcastError(broadcastErr) {
				return fmt.Errorf("broadcast transaction failed: %v", broadcastErr)
			}
		}

	}

	// Update grab lucky bag record in database
	err = chatDB.SaveOpenLuckyBag(grabEntity)
	if err != nil {
		return fmt.Errorf("failed to save open lucky bag: %v", err)
	}

	// Update cache after saving to database
	err = UpdateLuckyBagCacheAfterSave(grabEntity, nil)
	if err != nil {
		log.Printf("[disposingGrabLuckyBag] Failed to update cache after save: %v", err)
		// Don't return error for cache update failure, as the main operation succeeded
	}

	return nil
}

// cleanupDuplicateGrabRecords checks and cleans up duplicate grab records for the same address
// Returns true if the current entity should be processed, false if it should be skipped
func cleanupDuplicateGrabRecords(grabEntity *models.TalkGroupOpenLuckyBagV3) (bool, error) {
	// Get all grab records for this lucky bag from TalkGroupOpenLuckyBagListCollection
	openLuckyBagList, err := chatDB.GetOpenLuckyBagList(grabEntity.LuckyBagPinId)
	if err != nil {
		return false, fmt.Errorf("GetOpenLuckyBagList err: %v", err)
	}

	// Find records with same address but different pinId and grabState = 1
	var duplicateRecords []*models.OpenLuckyBagListItem
	for _, item := range openLuckyBagList.Items {
		if item.CreateAddress == grabEntity.Address &&
			item.CreateMetaId == grabEntity.MetaId &&
			item.OpenPinId != grabEntity.PinId {
			// Check if the corresponding grab record has grabState = 1
			openBag, err := chatDB.GetOpenLuckyBagByPinId(item.OpenPinId)
			if err == nil && openBag != nil && openBag.GrabState == models.GrabStateOpen {
				duplicateRecords = append(duplicateRecords, item)
			}
		}
	}

	// If no duplicate records found, no need to clean up
	if len(duplicateRecords) == 0 {
		log.Printf("[cleanupDuplicateGrabRecords] No duplicate records found for address %s in lucky bag %s",
			grabEntity.Address, grabEntity.LuckyBagPinId)
		return true, nil
	}

	log.Printf("[cleanupDuplicateGrabRecords] Found %d duplicate records for address %s in lucky bag %s, cleaning up",
		len(duplicateRecords), grabEntity.Address, grabEntity.LuckyBagPinId)

	// Remove current entity from TalkGroupOpenLuckyBagListCollection
	err = chatDB.RemoveOpenLuckyBagListItem(grabEntity.LuckyBagPinId, grabEntity.PinId)
	if err != nil {
		log.Printf("[cleanupDuplicateGrabRecords] RemoveOpenLuckyBagListItem err: %v", err)
	} else {
		log.Printf("[cleanupDuplicateGrabRecords] Removed current entity %s from list", grabEntity.PinId)
	}

	// Update current entity's grabState to 4 (duplicate)
	grabEntity.GrabState = models.GrabStateDuplicate
	err = chatDB.SaveOpenLuckyBag(grabEntity)
	if err != nil {
		log.Printf("[cleanupDuplicateGrabRecords] SaveOpenLuckyBag err: %v", err)
	} else {
		log.Printf("[cleanupDuplicateGrabRecords] Updated current entity %s grabState to 4", grabEntity.PinId)
	}

	// Skip processing this entity
	return false, nil
}

// Start grab lucky bag queue processor
func StartOpenLuckyBagQueueProcessor() {
	go func() {
		ticker := time.NewTicker(5 * time.Second) // Process every 5 seconds
		defer ticker.Stop()

		// Flag to track if the previous processing is still running
		isProcessing := false

		for {
			select {
			case <-ticker.C:
				if db.GlobalIsStop {
					log.Printf("[OpenLuckyBagQueueProcessor] Queue processor is stopped, skipping this cycle")
					continue
				}

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

func getLuckyBagKey(subId, code, createTimeStr, luckyBagAddress string, genType int64) (string, string) {
	if genType == 0 {
		// External lucky bag, generate key from parameters
		return makeGiftKey(subId, code, createTimeStr)
	} else if genType == 1 {
		// Internal lucky bag, get key from completed collection
		codeAddressKey, err := chatDB.GetLuckyBagCodeAddressKeyFromCompleted(code, luckyBagAddress)
		if err != nil {
			log.Printf("Failed to get lucky bag code address key from completed collection for code %s and address %s: %v", code, luckyBagAddress, err)
			return "", ""
		}
		if codeAddressKey == nil {
			log.Printf("Lucky bag code address key not found in completed collection for code %s and address %s", code, luckyBagAddress)
			return "", ""
		}
		return "", codeAddressKey.Key
	} else {
		return "", ""
	}
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

	// Get unused UTXO list (including both PayList and ErrLuckyBagVouts)
	unusedList := make([]*respond.UnusedList, 0)

	// Add unused items from PayList
	for _, v := range luckyBag.PayList {
		if !usedIndices[v.Index] {
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
	}

	// Add unused items from ErrLuckyBagVouts (these are also available for reclaim)
	for _, vout := range luckyBag.ErrLuckyBagVouts {
		if !usedIndices[vout.Index] {
			unused := &respond.UnusedList{
				Index:   vout.Index,
				Amount:  strconv.FormatUint(vout.Amount, 10),
				Address: vout.Address,
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
		unusedIndex        int64
		unusedAmount       string
		unusedAddress      string
		unusedLuckyAmount  string
		unusedLuckyFee     string
		unusedLuckyFeeRate string
	}
	reclaimEntityList := make([]*reclaimEntity, 0)

	// Collect all unused UTXOs
	for _, unused := range unusedList {
		reclaimEntityList = append(reclaimEntityList, &reclaimEntity{
			unusedIndex:        unused.Index,
			unusedAmount:       unused.Amount,
			unusedAddress:      unused.Address,
			unusedLuckyAmount:  unused.LuckyAmount,
			unusedLuckyFee:     unused.LuckyFee,
			unusedLuckyFeeRate: unused.LuckyFeeRate,
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
		for _, vout := range luckyBag.ErrLuckyBagVouts {
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
			Domain:              luckyBag.Domain,
			LuckyBagAddress:     luckyBag.LuckyBagAddress,
			GenType:             luckyBag.GenType,
			GenState:            luckyBag.GenState,
			Address:             address,
			PkScript:            pkScript,
			Amount:              v.unusedAmount,
			Index:               v.unusedIndex,
			LuckyAmount:         v.unusedLuckyAmount,
			LuckyFee:            v.unusedLuckyFee,
			LuckyFeeRate:        v.unusedLuckyFeeRate,
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
			ReclaimState:        models.GrabStateReclaim,
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
		err = chatDB.SaveResidueLuckyBagListAtomic(luckyBag.PinId, residueLuckyBag.PinId, luckyBag.GroupId, residueLuckyBag.Timestamp, metaId, address, []int64{v.unusedIndex})
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
	messages, err := chatDB.GetPendingResidueLuckyBagMessages(100) // Process 100 items each time
	if err != nil {
		log.Printf("GetPendingResidueLuckyBagMessages err: %v", err)
		return
	}

	for _, message := range messages {
		// Process reclaim lucky bag record
		err := disposingReclaimLuckyBag(message.ResidueLuckyBag)
		if err != nil {
			// log.Printf("disposingReclaimLuckyBag err: %v", err)
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

	if reclaimEntity.ReclaimState != models.GrabStateReclaim {
		return fmt.Errorf("reclaim state is not reclaim")
	}
	if reclaimEntity.Vins == nil || len(reclaimEntity.Vins) == 0 {
		return errors.New("no vins found")
	}

	_ = reclaimEntity.Vins[0] // utxo, temporarily unused
	_, hexStr := getLuckyBagKey(reclaimEntity.SubId, reclaimEntity.Code, reclaimEntity.CreateTimeStr, reclaimEntity.LuckyBagAddress, reclaimEntity.GenType)
	if hexStr == "" {
		return errors.New("failed to generate wif or hex")
	}

	if reclaimEntity.UsedList == nil || len(reclaimEntity.UsedList) == 0 {
		reclaimEntity.UsedList = make([]*models.ProInfoPayList, 0)
		luckyBag, err := chatDB.GetLuckyBagByPinId(reclaimEntity.LuckyBagPinId)
		if err != nil {
			return err
		}
		if luckyBag == nil {
			return errors.New("lucky bag not found")
		}
		for _, vout := range luckyBag.LuckyBagVouts {
			if vout.Index == reclaimEntity.Index {
				reclaimEntity.UsedList = append(reclaimEntity.UsedList, &models.ProInfoPayList{
					Amount:   strconv.FormatInt(int64(vout.Amount), 10),
					Address:  vout.Address,
					Index:    vout.Index,
					PkScript: vout.ScriptPubKey,
				})
			}
		}
		for _, vout := range luckyBag.ErrLuckyBagVouts {
			if vout.Index == reclaimEntity.Index {
				reclaimEntity.UsedList = append(reclaimEntity.UsedList, &models.ProInfoPayList{
					Amount:   strconv.FormatInt(int64(vout.Amount), 10),
					Address:  vout.Address,
					Index:    vout.Index,
					PkScript: vout.ScriptPubKey,
				})
			}
		}
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

	outputAmount := int64(0)
	if reclaimEntity.LuckyAmount != "" && reclaimEntity.LuckyAmount != "0" {
		reclaimNormalizedAmount := normalizeScientificNotation(reclaimEntity.LuckyAmount)
		outValue, err := strconv.ParseUint(reclaimNormalizedAmount, 10, 64)
		if err != nil {
			return fmt.Errorf("failed to parse lucky amount: %v", err)
		}
		if outValue >= 546 {
			outputAmount = int64(outValue)
		}
	}

	output := common.TxOutput{
		Address:    toAddress,
		Amount:     int64(totalAmount),
		NeedAmount: outputAmount,
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
	if reclaimEntity.Chain == "btc" {
		time.Sleep(1 * time.Second)
	}

	// Broadcast transaction with retry mechanism
	resultTxId, broadcastErr := chainAdapter[reclaimEntity.Chain].BroadcastTx(txRaw)
	if resultTxId != "" {
		// Success case
		reclaimEntity.ReclaimState = models.GrabStateReclaimAndSend
		reclaimEntity.ReclaimTxId = resultTxId
		reclaimEntity.ReclaimMsg = "success"
		// reclaimEntity.RetryCount = 0 // Reset retry count on success
		log.Printf("[Reclaim][%s]Success broadcast tx: %s", reclaimEntity.Chain, resultTxId)

		if reclaimEntity.Chain == "btc" {
			net := ""
			if common.TestNet == "1" {
				net = "testnet"
			}
			common_service.BroadcastTx(net, txRaw)
		}
	} else {
		// Failure case - increment retry count
		// Check if error is "too-long-mempool-chain" - skip retry counting for this error
		if strings.Contains(strings.ToLower(broadcastErr.Error()), "too-long-mempool-chain") {
			log.Printf("[Grad][%s]Too-long-mempool-chain error, skipping retry count increment: %s", reclaimEntity.Chain, broadcastErr.Error())
			// Don't increment retry count, just log and continue
			reclaimEntity.ReclaimMsg = fmt.Sprintf("Too-long-mempool-chain error, will retry later: %s", broadcastErr.Error())

			// Update residue lucky bag record in database
			err = chatDB.SaveResidueLuckyBag(reclaimEntity)
			if err != nil {
				return fmt.Errorf("failed to save residue lucky bag: %v", err)
			}

			// Return nil to continue processing in next cycle
			return fmt.Errorf("too-long-mempool-chain error, skipping retry count increment: %s", broadcastErr.Error())
		}

		reclaimEntity.RetryCount++
		log.Printf("[Reclaim][%s][%s]Failure broadcast tx: %s, retryCount: %d", reclaimEntity.Chain, reclaimEntity.PinId, broadcastErr.Error(), reclaimEntity.RetryCount)

		// Update queue message with new retry count
		err = chatDB.UpdateResidueLuckyBagQueueMessage(reclaimEntity.PinId, reclaimEntity.RetryCount, "")
		if err != nil {
			log.Printf("[Reclaim][%s]Failed to update queue message retry count: %v", reclaimEntity.Chain, err)
		}

		// Check if we've exceeded maximum retry attempts
		if reclaimEntity.RetryCount >= 5 {
			// Max retries exceeded, record error and save to error collection
			reclaimEntity.ReclaimState = models.GrabStateReclaimAndSendErr
			reclaimEntity.ReclaimMsg = fmt.Sprintf("Max retries exceeded (5), last error: %s", broadcastErr.Error())

			// Save to error collection
			err = chatDB.SaveResidueLuckyBagError(reclaimEntity.PinId, reclaimEntity.LuckyBagPinId)
			if err != nil {
				log.Printf("[Reclaim][%s]Failed to save residue lucky bag error: %v\n", reclaimEntity.Chain, err)
			} else {
				log.Printf("[Reclaim][%s]Saved residue lucky bag error record for pinId: %s\n", reclaimEntity.Chain, reclaimEntity.PinId)
			}
		} else {
			fmt.Printf("[Reclaim][%s][%s]Retry broadcast tx: %s, retryCount: %d\n", reclaimEntity.Chain, reclaimEntity.PinId, broadcastErr.Error(), reclaimEntity.RetryCount)
			// Still within retry limit, set error state but don't save to error collection yet
			// reclaimEntity.ReclaimState = models.GrabStateReclaimAndSendErr
			reclaimEntity.ReclaimMsg = fmt.Sprintf("Retry %d/5, error: %s", reclaimEntity.RetryCount, broadcastErr.Error())

			// Update reclaim lucky bag record in database
			err = chatDB.SaveResidueLuckyBag(reclaimEntity)
			if err != nil {
				return fmt.Errorf("failed to save residue lucky bag: %v", err)
			}

			// Check if error contains broadcast-related issues, if not, return the error
			if !isBroadcastError(broadcastErr) {
				return fmt.Errorf("broadcast transaction failed: %v", broadcastErr)
			}
		}

	}

	// Update reclaim lucky bag record in database
	err = chatDB.SaveResidueLuckyBag(reclaimEntity)
	if err != nil {
		return fmt.Errorf("failed to save residue lucky bag: %v", err)
	}

	// Update cache after saving to database
	err = UpdateLuckyBagCacheAfterSave(nil, reclaimEntity)
	if err != nil {
		log.Printf("[disposingReclaimLuckyBag] Failed to update cache after save: %v", err)
		// Don't return error for cache update failure, as the main operation succeeded
	}

	return nil
}

// Start reclaim lucky bag queue processor
func StartResidueLuckyBagQueueProcessor() {
	go func() {
		ticker := time.NewTicker(5 * time.Second) // Process every 5 seconds
		defer ticker.Stop()

		// Flag to track if the previous processing is still running
		isProcessing := false

		for {
			select {
			case <-ticker.C:
				if db.GlobalIsStop {
					log.Printf("[ResidueLuckyBagQueueProcessor] Queue processor is stopped, skipping this cycle")
					continue
				}

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

// UpdateLuckyBagValidation updates lucky bag validation counts and lists
func UpdateLuckyBagValidation(luckyBagPinId string) (map[string]interface{}, error) {
	// Get lucky bag from database
	luckyBag, err := chatDB.GetLuckyBagByPinId(luckyBagPinId)
	if err != nil {
		return nil, fmt.Errorf("failed to get lucky bag: %v", err)
	}
	if luckyBag == nil {
		return nil, fmt.Errorf("lucky bag not found: %s", luckyBagPinId)
	}

	if luckyBag.OriginalPayList == nil {
		luckyBag.OriginalPayList = luckyBag.PayList
	}
	if luckyBag.OriginalLuckyBagVouts == nil {
		luckyBag.OriginalLuckyBagVouts = luckyBag.LuckyBagVouts
	}

	// Get transaction data
	var txData *wire.MsgTx
	if luckyBag.TxId != "" {
		// Get transaction from chain adapter
		if chainAdapter != nil && chainAdapter[luckyBag.Chain] != nil {
			tx, err := chainAdapter[luckyBag.Chain].GetTransaction(luckyBag.TxId)
			if err == nil && tx != nil {
				fmt.Printf("tx: %+v\n", tx)
				// Convert to wire.MsgTx
				switch txType := tx.(type) {
				case *wire.MsgTx:
					txData = txType
					break
				case wire.MsgTx:
					txData = &txType
					break
				case *btcutil.Tx:
					txData = txType.MsgTx()
					break
				case btcutil.Tx:
					txData = txType.MsgTx()
					break
				default:
					fmt.Println("tx type:", txType)
					// Try to convert other types
					if msgTx, ok := tx.(*wire.MsgTx); ok {
						txData = msgTx
					} else if msgTx, ok := tx.(wire.MsgTx); ok {
						txData = &msgTx
					} else {
						log.Printf("[UpdateLuckyBagValidation]Failed to convert tx to *wire.MsgTx for lucky bag: %s", luckyBagPinId)
						return nil, fmt.Errorf("failed to convert tx to *wire.MsgTx for lucky bag: %s", luckyBagPinId)
					}
				}
			} else if err != nil {
				log.Printf("[UpdateLuckyBagValidation]Failed to get transaction data: %v", err)
				return nil, fmt.Errorf("failed to get transaction data: %v", err)
			} else {
				log.Printf("[UpdateLuckyBagValidation]Failed to get transaction data for lucky bag: %s, no err no tx", luckyBagPinId)
				return nil, fmt.Errorf("failed to get transaction data for lucky bag: %s, no err no tx", luckyBagPinId)
			}
		} else {
			log.Printf("[UpdateLuckyBagValidation]Chain adapter not found for chain: %s", luckyBag.Chain)
			return nil, fmt.Errorf("chain adapter not found for chain: %s", luckyBag.Chain)
		}
	}
	if txData == nil {
		return nil, fmt.Errorf("failed to get transaction data")
	} else {
		fmt.Printf("txData: %+v\n", txData)
	}

	// Convert payment list
	var (
		payList          []*models.ProInfoPayList = make([]*models.ProInfoPayList, 0)
		errPayList       []*models.ProInfoPayList = make([]*models.ProInfoPayList, 0)
		luckyBagVouts    []*models.LuckyBagOutput = make([]*models.LuckyBagOutput, 0)
		errLuckyBagVouts []*models.LuckyBagOutput = make([]*models.LuckyBagOutput, 0)
	)

	// Create a map to store PayList items by index for quick lookup
	payListByIndex := make(map[int64]*models.ProInfoPayList)
	luckyTxOutList := make([]*models.LuckyBagOutput, 0)

	payAddressList := make([]string, 0)
	// First, process all PayList items
	for _, pay := range luckyBag.PayList {
		payAddressList = append(payAddressList, pay.Address)
		payItem := &models.ProInfoPayList{
			Amount:  pay.Amount,
			Address: pay.Address,
			Index:   pay.Index,
		}
		if _, ok := payListByIndex[pay.Index]; ok {
			errPayList = append(errPayList, payItem)
		} else {
			payListByIndex[pay.Index] = payItem
		}
	}

	if txData != nil {
		// First, extract all TxOut addresses and find matching UTXOs
		for i, vout := range txData.TxOut {
			index := int64(i)

			// Extract address from vout
			voutAddress := ""
			// Get chain params based on chain name
			var netParams *chaincfg.Params = &chaincfg.MainNetParams
			if common.TestNet == "1" {
				netParams = &chaincfg.TestNet3Params
			} else if common.TestNet == "2" {
				netParams = &chaincfg.RegressionNetParams
			}

			class, addresses, _, _ := txscript.ExtractPkScriptAddrs(vout.PkScript, netParams)
			if class.String() != "nulldata" && class.String() != "nonstandard" && len(addresses) > 0 {
				voutAddress = addresses[0].String()
			}

			// Check if this vout address is in payAddressList (belongs to this lucky bag)
			if len(payAddressList) > 0 && contains(payAddressList, voutAddress) {
				// This is a lucky bag UTXO, add it to the list
				luckyBagVout := &models.LuckyBagOutput{
					ScriptPubKey: hex.EncodeToString(vout.PkScript),
					Amount:       uint64(vout.Value),
					Address:      voutAddress,
					Index:        index,
				}
				luckyTxOutList = append(luckyTxOutList, luckyBagVout)
			}
		}

		// Now process only the lucky bag UTXOs
		for _, luckyBagVout := range luckyTxOutList {
			// Check if this index exists in PayList
			if payItem, exists := payListByIndex[luckyBagVout.Index]; exists {
				// Index exists in PayList, check if address matches
				if payItem.Address == luckyBagVout.Address {
					// Both index and address match - this is correct
					payList = append(payList, payItem)
					luckyBagVouts = append(luckyBagVouts, luckyBagVout)
				} else {
					// Index exists but address doesn't match - this is an error
					errPayList = append(errPayList, payItem)
					errLuckyBagVouts = append(errLuckyBagVouts, luckyBagVout)
				}
			} else {
				// Index doesn't exist in PayList - this is an error
				errLuckyBagVouts = append(errLuckyBagVouts, luckyBagVout)
			}
		}

		// Check for PayList items that don't have corresponding TxOut
		for index, payItem := range payListByIndex {
			found := false
			for _, vout := range luckyBagVouts {
				if vout.Index == index {
					found = true
					break
				}
			}
			for _, vout := range errLuckyBagVouts {
				if vout.Index == index {
					found = true
					break
				}
			}

			if !found {
				// PayList item exists but no corresponding TxOut - this is an error
				errPayList = append(errPayList, payItem)
			}
		}
	} else {
		// No txData available, all PayList items are considered errors
		for _, payItem := range payListByIndex {
			errPayList = append(errPayList, payItem)
		}
	}

	fmt.Println("originalPayList len:", len(luckyBag.OriginalPayList))
	fmt.Println("originalLuckyBagVouts len:", len(luckyBag.OriginalLuckyBagVouts))
	fmt.Println("payList len:", len(payList))
	fmt.Println("errPayList len:", len(errPayList))
	fmt.Println("luckyBagVouts len:", len(luckyBagVouts))
	fmt.Println("errLuckyBagVouts len:", len(errLuckyBagVouts))

	// Calculate counts
	count, _ := strconv.ParseInt(luckyBag.Count, 10, 64)
	validCount := len(payList)
	errCount := count - int64(validCount)

	// Update lucky bag with new validation data
	luckyBag.ValidCount = strconv.FormatInt(int64(validCount), 10)
	luckyBag.ErrCount = strconv.FormatInt(errCount, 10)
	luckyBag.PayList = payList
	luckyBag.ErrPayList = errPayList
	luckyBag.LuckyBagVouts = luckyBagVouts
	luckyBag.ErrLuckyBagVouts = errLuckyBagVouts

	// Clean up error lucky bags if there are errors
	if len(errPayList) > 0 || len(errLuckyBagVouts) > 0 {
		err = cleanupErrorLuckyBags(luckyBag, errPayList, errLuckyBagVouts)
		if err != nil {
			log.Printf("[UpdateLuckyBagValidation] Failed to cleanup error lucky bags: %v", err)
		}
	}

	// Save updated lucky bag
	err = chatDB.SaveLuckyBag(luckyBag)
	if err != nil {
		return nil, fmt.Errorf("failed to save updated lucky bag: %v", err)
	}

	// Save lucky bag to appropriate collection based on error status
	err = chatDB.SaveLuckyBagPendingToCollection(luckyBag)
	if err != nil {
		return nil, fmt.Errorf("failed to save lucky bag to collection: %v", err)
	}

	// Return update result
	result := map[string]interface{}{
		"luckyBagPinId":         luckyBagPinId,
		"validCount":            validCount,
		"errCount":              errCount,
		"totalCount":            count,
		"payListCount":          len(payList),
		"errPayListCount":       len(errPayList),
		"luckyBagVoutsCount":    len(luckyBagVouts),
		"errLuckyBagVoutsCount": len(errLuckyBagVouts),
		"updated":               true,
	}

	return result, nil
}

// cleanupErrorLuckyBags cleans up error lucky bags from open lists and queue collections
func cleanupErrorLuckyBags(luckyBag *models.TalkGroupLuckyBagV3, errPayList []*models.ProInfoPayList, errLuckyBagVouts []*models.LuckyBagOutput) error {
	// Build error indices set for quick lookup
	errorIndices := make(map[int64]bool)
	for _, errPay := range errPayList {
		errorIndices[errPay.Index] = true
	}
	for _, errVout := range errLuckyBagVouts {
		errorIndices[errVout.Index] = true
	}

	// Get open lucky bag list
	openList, err := chatDB.GetOpenLuckyBagList(luckyBag.PinId)
	if err != nil {
		return fmt.Errorf("failed to get open lucky bag list: %v", err)
	}

	// Find and remove error lucky bags from open list
	var errorOpenPinIds []string
	for _, openItem := range openList.Items {
		openLuckyBag, err := chatDB.GetOpenLuckyBagByPinId(openItem.OpenPinId)
		if err != nil || openLuckyBag == nil {
			continue
		}

		// Check if this open lucky bag is for an error index
		if errorIndices[openLuckyBag.Index] {
			errorOpenPinIds = append(errorOpenPinIds, openItem.OpenPinId)
			log.Printf("[UpdateLuckyBagValidation] Found error open lucky bag: %s, index: %d", openItem.OpenPinId, openLuckyBag.Index)
		}
	}

	// Remove error open lucky bags from open list
	for _, errorOpenPinId := range errorOpenPinIds {
		err = chatDB.RemoveOpenLuckyBagListItem(luckyBag.PinId, errorOpenPinId)
		if err != nil {
			log.Printf("[UpdateLuckyBagValidation] Failed to remove error open lucky bag %s from list: %v", errorOpenPinId, err)
		} else {
			log.Printf("[UpdateLuckyBagValidation] Successfully removed error open lucky bag %s from list", errorOpenPinId)
		}
	}

	// Clean up error lucky bags from queue collection
	err = cleanupErrorLuckyBagsFromQueue(errorOpenPinIds)
	if err != nil {
		log.Printf("[UpdateLuckyBagValidation] Failed to cleanup error lucky bags from queue: %v", err)
	}

	return nil
}

// cleanupErrorLuckyBagsFromQueue removes error lucky bags from the queue collection
func cleanupErrorLuckyBagsFromQueue(errorOpenPinIds []string) error {
	if len(errorOpenPinIds) == 0 {
		return nil
	}

	iter, err := db.Pb[db.TalkGroupOpenLuckyBagQueueCollection].NewIter(nil)
	if err != nil {
		return fmt.Errorf("failed to create iterator for queue collection: %v", err)
	}
	defer iter.Close()

	// Find and remove error lucky bags from queue
	for iter.First(); iter.Valid(); iter.Next() {
		value := string(iter.Value())

		var queueMessage db.QueueOpenLuckyBagMessage
		err := json.Unmarshal([]byte(value), &queueMessage)
		if err != nil {
			continue
		}

		// Check if this queue message is for an error lucky bag
		for _, errorOpenPinId := range errorOpenPinIds {
			if queueMessage.PinId == errorOpenPinId {
				// Remove from queue collection
				err = db.Pb[db.TalkGroupOpenLuckyBagQueueCollection].Delete(iter.Key(), pebble.Sync)
				if err != nil {
					log.Printf("[UpdateLuckyBagValidation] Failed to remove error lucky bag %s from queue: %v", errorOpenPinId, err)
				} else {
					log.Printf("[UpdateLuckyBagValidation] Successfully removed error lucky bag %s from queue", errorOpenPinId)
				}
				break
			}
		}
	}

	return nil
}

// GenerateLuckyBagCodeAddressKey generates a new lucky bag code address key for frontend
// This method is called before creating a lucky bag to get the code and address
func GenerateLuckyBagCodeAddressKey() (*respond.LuckyBagCodeAddressKeyResponse, error) {
	// Call the database method to generate a new lucky bag code address key
	codeAddressKey, err := chatDB.GenerateLuckyBagCodeAddressKey()
	if err != nil {
		return nil, fmt.Errorf("failed to generate lucky bag code address key: %v", err)
	}

	// Create response without private key for security
	response := &respond.LuckyBagCodeAddressKeyResponse{
		Code:            codeAddressKey.Code,
		LuckyBagAddress: codeAddressKey.LuckyBagAddress,
		Timestamp:       codeAddressKey.Timestamp,
	}

	return response, nil
}

// Helper function to check if slice contains item
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// ProcessExpiredLuckyBags processes expired lucky bags from pending collections
func ProcessExpiredLuckyBags() {
	// Process pending collection
	err := processExpiredLuckyBagsFromCollection(db.TalkGroupLuckyBagPinPendingCollection, "")
	if err != nil {
		log.Printf("[ExpiredLuckyBag]ProcessExpiredLuckyBags pending collection err: %v", err)
	}

	// Process error pending collection
	err = processExpiredLuckyBagsFromCollection(db.TalkGroupLuckyBagPinErrPendingCollection, "")
	if err != nil {
		log.Printf("[ExpiredLuckyBag]ProcessExpiredLuckyBags error pending collection err: %v", err)
	}
}

// processExpiredLuckyBagsFromCollection processes expired lucky bags from a specific collection
func processExpiredLuckyBagsFromCollection(collection, pinId string) error {
	iter, err := db.Pb[collection].NewIter(nil)
	if err != nil {
		return fmt.Errorf("[ExpiredLuckyBag]failed to create iterator for collection %s: %v", collection, err)
	}
	defer iter.Close()

	expiredLuckyBags := make([]string, 0)
	now := time.Now().Unix()
	expirationTime := int64(24 * 60 * 60) // 24 hours in seconds

	// Iterate through all lucky bags in the collection
	for iter.First(); iter.Valid(); iter.Next() {
		luckyBagPinId := string(iter.Value())

		// Get lucky bag details
		luckyBag, err := chatDB.GetLuckyBagByPinId(luckyBagPinId)
		if err != nil {
			log.Printf("[ExpiredLuckyBag]Failed to get lucky bag %s: %v", luckyBagPinId, err)
			continue
		}
		if luckyBag == nil {
			log.Printf("[ExpiredLuckyBag]Lucky bag %s not found", luckyBagPinId)
			continue
		}

		// Check if lucky bag has expired (more than 24 hours old)
		if now-luckyBag.Timestamp > expirationTime {
			expiredLuckyBags = append(expiredLuckyBags, luckyBagPinId)
		}
	}

	if pinId != "" {
		expiredLuckyBags = append(expiredLuckyBags, pinId)
	}

	// Process expired lucky bags
	for _, luckyBagPinId := range expiredLuckyBags {
		log.Printf("[ExpiredLuckyBag]Processing expired lucky bag: %s", luckyBagPinId)

		// Get lucky bag details again for processing
		luckyBag, err := chatDB.GetLuckyBagByPinId(luckyBagPinId)
		if err != nil {
			log.Printf("[ExpiredLuckyBag]Failed to get lucky bag %s for processing: %v", luckyBagPinId, err)
			continue
		}
		if luckyBag == nil {
			continue
		}

		// Check lucky bag status and determine target collection
		targetCollection, shouldReclaim := determineLuckyBagStatus(luckyBag, collection)

		if shouldReclaim {
			// Execute reclaim logic
			err = autoReclaimExpiredLuckyBag(luckyBag)
			if err != nil {
				log.Printf("[ExpiredLuckyBag]Failed to auto reclaim expired lucky bag %s: %v", luckyBagPinId, err)
				continue
			}
		}

		// Update lucky bag state based on target collection
		err = updateLuckyBagState(luckyBag, targetCollection)
		if err != nil {
			log.Printf("[ExpiredLuckyBag]Failed to update lucky bag state for %s: %v", luckyBagPinId, err)
			continue
		}

		// Move to target collection
		err = moveLuckyBagToCollection(luckyBagPinId, collection, targetCollection)
		if err != nil {
			log.Printf("[ExpiredLuckyBag]Failed to move lucky bag %s from %s to %s: %v", luckyBagPinId, collection, targetCollection, err)
			continue
		}

		log.Printf("[ExpiredLuckyBag]Successfully processed expired lucky bag %s: moved from %s to %s, state updated to %d", luckyBagPinId, collection, targetCollection, luckyBag.State)
	}

	return nil
}

// ProcessExpiredLuckyBagByPinId processes a specific lucky bag by pinId as if it were expired
func ProcessExpiredLuckyBagByPinId(pinId string) (map[string]interface{}, error) {
	if pinId == "" {
		return nil, fmt.Errorf("pinId cannot be empty")
	}

	// Get lucky bag details
	luckyBag, err := chatDB.GetLuckyBagByPinId(pinId)
	if err != nil {
		return nil, fmt.Errorf("failed to get lucky bag %s: %v", pinId, err)
	}
	if luckyBag == nil {
		return nil, fmt.Errorf("lucky bag %s not found", pinId)
	}

	// Determine which collection the lucky bag is currently in
	var sourceCollection string
	var found bool

	// Check pending collection
	_, closer, err := db.Pb[db.TalkGroupLuckyBagPinPendingCollection].Get([]byte(pinId))
	if err == nil {
		closer.Close()
		sourceCollection = db.TalkGroupLuckyBagPinPendingCollection
		found = true
	}

	// Check error pending collection
	if !found {
		_, closer, err := db.Pb[db.TalkGroupLuckyBagPinErrPendingCollection].Get([]byte(pinId))
		if err == nil {
			closer.Close()
			sourceCollection = db.TalkGroupLuckyBagPinErrPendingCollection
			found = true
		}
	}

	if !found {
		return nil, fmt.Errorf("lucky bag %s not found in any pending collection", pinId)
	}

	// Process the lucky bag as if it were expired
	err = processExpiredLuckyBagsFromCollection(sourceCollection, pinId)
	if err != nil {
		return nil, fmt.Errorf("failed to process expired lucky bag %s: %v", pinId, err)
	}

	// Get updated lucky bag details after processing
	updatedLuckyBag, err := chatDB.GetLuckyBagByPinId(pinId)
	if err != nil {
		return nil, fmt.Errorf("failed to get updated lucky bag %s: %v", pinId, err)
	}

	result := map[string]interface{}{
		"pinId":            pinId,
		"sourceCollection": sourceCollection,
		"processed":        true,
		"newState":         updatedLuckyBag.State,
		"timestamp":        time.Now().Unix(),
	}

	// Determine target collection based on new state
	switch updatedLuckyBag.State {
	case 2: // completed
		result["targetCollection"] = db.TalkGroupLuckyBagPinCompletedCollection
	case 3: // timeout residue
		result["targetCollection"] = db.TalkGroupLuckyBagPinTimeoutResidueCollection
	case 5: // err timeout residue
		result["targetCollection"] = db.TalkGroupLuckyBagPinErrTimeoutResidueCollection
	default:
		result["targetCollection"] = "unknown"
	}

	return result, nil
}

// determineLuckyBagStatus determines the target collection and whether reclaim is needed
func determineLuckyBagStatus(luckyBag *models.TalkGroupLuckyBagV3, sourceCollection string) (string, bool) {
	// Get claimed lucky bag list
	openList, err := chatDB.GetOpenLuckyBagList(luckyBag.PinId)
	if err != nil {
		log.Printf("[ExpiredLuckyBag]Failed to get open lucky bag list for %s: %v", luckyBag.PinId, err)
		return sourceCollection, false
	}

	// Get reclaimed lucky bag list
	residueList, err := chatDB.GetResidueLuckyBagList(luckyBag.PinId)
	if err != nil {
		log.Printf("[ExpiredLuckyBag]Failed to get residue lucky bag list for %s: %v", luckyBag.PinId, err)
		return sourceCollection, false
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

	// Count total available UTXOs (PayList + ErrLuckyBagVouts)
	totalAvailable := len(luckyBag.PayList) + len(luckyBag.ErrLuckyBagVouts)
	usedCount := len(usedIndices)

	// Determine status based on source collection
	if sourceCollection == db.TalkGroupLuckyBagPinPendingCollection {
		// Normal pending collection
		if usedCount >= totalAvailable {
			// All UTXOs have been used, move to completed
			return db.TalkGroupLuckyBagPinCompletedCollection, false
		} else {
			// Some UTXOs are still available, move to timeout residue
			return db.TalkGroupLuckyBagPinTimeoutResidueCollection, true
		}
	} else if sourceCollection == db.TalkGroupLuckyBagPinErrPendingCollection {
		// Error pending collection
		// Some UTXOs are still available, move to error timeout residue
		return db.TalkGroupLuckyBagPinErrTimeoutResidueCollection, true
	}

	// Default case: stay in source collection
	return sourceCollection, false
}

// updateLuckyBagState updates the lucky bag state based on target collection
func updateLuckyBagState(luckyBag *models.TalkGroupLuckyBagV3, targetCollection string) error {
	// Update state based on target collection
	switch targetCollection {
	case db.TalkGroupLuckyBagPinCompletedCollection:
		luckyBag.State = 2 // completed
	case db.TalkGroupLuckyBagPinTimeoutResidueCollection:
		luckyBag.State = 3 // timeout residue
	case db.TalkGroupLuckyBagPinErrTimeoutResidueCollection:
		luckyBag.State = 5 // err timeout residue
	default:
		// Keep current state for other collections
		return nil
	}

	// Save updated lucky bag
	err := chatDB.SaveLuckyBag(luckyBag)
	if err != nil {
		return fmt.Errorf("failed to save lucky bag with updated state: %v", err)
	}

	return nil
}

// moveLuckyBagToCollection moves a lucky bag from source collection to target collection
func moveLuckyBagToCollection(luckyBagPinId, sourceCollection, targetCollection string) error {
	// Add to target collection
	err := db.Pb[targetCollection].Set([]byte(luckyBagPinId), []byte(luckyBagPinId), pebble.Sync)
	if err != nil {
		return fmt.Errorf("failed to add lucky bag to target collection %s: %v", targetCollection, err)
	}

	// Remove from source collection
	err = db.Pb[sourceCollection].Delete([]byte(luckyBagPinId), pebble.Sync)
	if err != nil {
		return fmt.Errorf("failed to remove lucky bag from source collection %s: %v", sourceCollection, err)
	}

	return nil
}

// autoReclaimExpiredLuckyBag automatically reclaims an expired lucky bag
func autoReclaimExpiredLuckyBag(luckyBag *models.TalkGroupLuckyBagV3) error {
	// Get claimed lucky bag list
	openList, err := chatDB.GetOpenLuckyBagList(luckyBag.PinId)
	if err != nil {
		return fmt.Errorf("[ExpiredLuckyBag]failed to get open lucky bag list: %v", err)
	}

	// Get reclaimed lucky bag list
	residueList, err := chatDB.GetResidueLuckyBagList(luckyBag.PinId)
	if err != nil {
		return fmt.Errorf("[ExpiredLuckyBag]failed to get residue lucky bag list: %v", err)
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

	// Get unused UTXO list (including both PayList and ErrLuckyBagVouts)
	unusedList := make([]*respond.UnusedList, 0)

	// Add unused items from PayList
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

	// Add unused items from ErrLuckyBagVouts (these are also available for reclaim)
	for _, vout := range luckyBag.ErrLuckyBagVouts {
		if !usedIndices[vout.Index] {
			unused := &respond.UnusedList{
				Index:   vout.Index,
				Amount:  strconv.FormatUint(vout.Amount, 10),
				Address: vout.Address,
			}
			unusedList = append(unusedList, unused)
		}
	}

	if len(unusedList) <= 0 {
		log.Printf("[ExpiredLuckyBag]No unused UTXOs to reclaim for lucky bag %s", luckyBag.PinId)
		return nil
	}

	// Execute reclaim logic
	err = commonReclaim(luckyBag, unusedList, luckyBag.MetaId, luckyBag.Address)
	if err != nil {
		return fmt.Errorf("[ExpiredLuckyBag]failed to execute common reclaim: %v", err)
	}

	log.Printf("[ExpiredLuckyBag]Successfully auto reclaimed expired lucky bag %s with %d unused UTXOs", luckyBag.PinId, len(unusedList))
	return nil
}

// StartExpiredLuckyBagProcessor starts the expired lucky bag processor
func StartExpiredLuckyBagProcessor() {
	go func() {
		ticker := time.NewTicker(10 * time.Minute) // Check every 10 minutes
		defer ticker.Stop()

		// Flag to track if the previous processing is still running
		isProcessing := false

		for {
			select {
			case <-ticker.C:

				if db.GlobalIsStop {
					log.Printf("[StartExpiredLuckyBagProcessor] Processor is stopped, skipping this cycle")
					continue
				}

				// Skip if previous processing is still running
				if isProcessing {
					log.Printf("[StartExpiredLuckyBagProcessor] Previous ProcessExpiredLuckyBags is still running, skipping this cycle")
					continue
				}

				// Set processing flag
				isProcessing = true

				// Process expired lucky bags in a goroutine
				go func() {
					defer func() {
						// Reset processing flag when done
						isProcessing = false
					}()

					ProcessExpiredLuckyBags()
				}()
			}
		}
	}()
}
