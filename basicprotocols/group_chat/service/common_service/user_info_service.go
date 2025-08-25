package common_service

import (
	"fmt"
	"log"
	"manindexer/basicprotocols/group_chat/api/respond"
	"manindexer/basicprotocols/group_chat/service/cache_service"
	"manindexer/common"
	"sync"
	"time"
)

const (
	ManCodeSuccess = 1
)

var (
	userInfoPollingRunning bool
	pollingMutex           sync.Mutex
	updateInProgress       bool
	updateMutex            sync.Mutex
)

type ManResp struct {
	Code    int64       `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

type MetaIDUserInfo struct {
	Metaid  string `json:"metaid"`
	Name    string `json:"name"`
	Address string `json:"address"`
	Avatar  string `json:"avatar"`
}

func FetchMetaIDUserInfo(address string) *respond.UserInfo {
	// First try to get user info and update time from cache
	cachedUserInfo, updateTime, err := cache_service.GetCacheUserInfoWithTime(address)
	if err != nil {
		// Cache retrieval failed, log error but continue execution
		fmt.Printf("Failed to get user info from cache: %v\n", err)
	}

	// Check if cache is valid (exists and update time is within 5 minutes)
	if cachedUserInfo != nil && !updateTime.IsZero() {
		// Check if more than 5 minutes have passed
		if time.Since(updateTime) <= 5*time.Minute {
			// Cache exists and is within 5 minutes, return directly
			return cachedUserInfo
		}
	}

	// Cache doesn't exist, expired, or older than 5 minutes, get latest info from API
	userInfo, err := fetchMetaIDUserInfoInfo(address)
	if err != nil {
		if cachedUserInfo != nil {
			return cachedUserInfo
		}
		return nil
	}

	// Build user info
	avatarImage := ""
	if userInfo.Avatar != "" {
		avatarImage = common.Config.GroupChat.ManHost + userInfo.Avatar
	}

	userInfoResponse := &respond.UserInfo{
		Metaid:      userInfo.Metaid,
		AvatarImage: avatarImage,
		Avatar:      userInfo.Avatar,
		Name:        userInfo.Name,
	}

	// Update cache
	_, cacheErr := cache_service.SetCacheUserInfo(address, userInfoResponse)
	if cacheErr != nil {
		// Cache update failed, log error but don't affect return result
		fmt.Printf("Failed to set user info to cache: %v\n", cacheErr)
	}

	return userInfoResponse
}

func fetchMetaIDUserInfoInfo(address string) (*MetaIDUserInfo, error) {
	var (
		url    string
		result string
		resp   *ManResp
		data   *MetaIDUserInfo
		err    error
	)
	query := map[string]string{}
	if common.Config.GroupChat.ManHost == "" {
		return nil, fmt.Errorf("manHost is empty")
	}
	url = fmt.Sprintf("%s/api/info/address/%s", common.Config.GroupChat.ManHost, address)

	result, err = common.GetUrl(url, query, nil)
	if err != nil {
		return nil, err
	}
	if err = common.JsonToObject(result, &resp); err != nil {
		return nil, fmt.Errorf("get request err:%s", err.Error())
	}
	if resp.Code != ManCodeSuccess {
		return nil, fmt.Errorf("msg:%s", resp.Message)
	}

	if err = common.JsonToAny(resp.Data, &data); err != nil {
		return nil, fmt.Errorf("get request err:%s", err.Error())
	}
	return data, nil
}

// StartUserInfoPolling Start polling to update cached user info
func StartUserInfoPolling() {
	pollingMutex.Lock()
	defer pollingMutex.Unlock()

	if userInfoPollingRunning {
		log.Println("[CACHE_SERVICE]User info polling is already running")
		return
	}

	userInfoPollingRunning = true
	go func() {
		ticker := time.NewTicker(2 * time.Minute) // Poll every 2 minutes
		defer ticker.Stop()

		log.Println("[CACHE_SERVICE]User info polling started")
		for {
			select {
			case <-ticker.C:
				// Check if previous update is still in progress
				updateMutex.Lock()
				if updateInProgress {
					log.Println("[CACHE_SERVICE]Previous update still in progress, skipping this cycle")
					updateMutex.Unlock()
					continue
				}
				updateInProgress = true
				updateMutex.Unlock()

				// Start update in a separate goroutine to allow next cycle to check status
				go func() {
					defer func() {
						updateMutex.Lock()
						updateInProgress = false
						updateMutex.Unlock()
					}()
					updateExpiredUserInfo()
				}()
			}
		}
	}()
}

// StopUserInfoPolling Stop polling
func StopUserInfoPolling() {
	pollingMutex.Lock()
	defer pollingMutex.Unlock()

	if !userInfoPollingRunning {
		log.Println("[CACHE_SERVICE]User info polling is not running")
		return
	}

	userInfoPollingRunning = false
	log.Println("[CACHE_SERVICE]User info polling stopped")
}

// IsUserInfoPollingRunning Check if user info polling is running
func IsUserInfoPollingRunning() bool {
	pollingMutex.Lock()
	defer pollingMutex.Unlock()
	return userInfoPollingRunning
}

// IsUserInfoUpdateInProgress Check if user info update is currently in progress
func IsUserInfoUpdateInProgress() bool {
	updateMutex.Lock()
	defer updateMutex.Unlock()
	return updateInProgress
}

// updateExpiredUserInfo Update user info that is about to expire (within 5 minutes)
func updateExpiredUserInfo() {
	log.Println("[CACHE_SERVICE]Starting user info update cycle")

	// Get all cached user info addresses
	addresses := getAllCachedUserInfoAddresses()
	log.Printf("[CACHE_SERVICE]Found %d cached user info addresses to check", len(addresses))

	updateCount := 0
	for _, address := range addresses {
		// Check if this user info needs updating
		if shouldUpdateUserInfo(address) {
			updateSingleUserInfo(address)
			updateCount++
		}
	}

	log.Printf("[CACHE_SERVICE]User info update cycle completed. Updated %d addresses", updateCount)
}

// getAllCachedUserInfoAddresses Get all addresses from cached user info
func getAllCachedUserInfoAddresses() []string {
	return cache_service.GetAllCachedUserInfoAddresses()
}

// shouldUpdateUserInfo Check if user info for the given address should be updated
func shouldUpdateUserInfo(address string) bool {
	// Get cached user info with update time
	cachedUserInfo, updateTime, err := cache_service.GetCacheUserInfoWithTime(address)
	if err != nil {
		log.Printf("[CACHE_SERVICE]Failed to get user info for address %s: %v", address, err)
		return false
	}

	// If no cached data, no need to update
	if cachedUserInfo == nil || updateTime.IsZero() {
		return false
	}

	// Check if update time is older than 4 minutes (update before 5-minute expiry)
	return time.Since(updateTime) > 4*time.Minute
}

// updateSingleUserInfo Update user info for a single address
func updateSingleUserInfo(address string) {
	log.Printf("[CACHE_SERVICE]Updating user info for address: %s", address)

	// Fetch latest user info from API
	userInfo, err := fetchMetaIDUserInfoInfo(address)
	if err != nil {
		log.Printf("[CACHE_SERVICE]Failed to fetch user info for address %s: %v", address, err)
		return
	}

	// Build user info response
	avatarImage := ""
	if userInfo.Avatar != "" {
		avatarImage = common.Config.GroupChat.ManHost + userInfo.Avatar
	}

	userInfoResponse := &respond.UserInfo{
		Metaid:      userInfo.Metaid,
		AvatarImage: avatarImage,
		Avatar:      userInfo.Avatar,
		Name:        userInfo.Name,
	}

	// Update cache
	_, cacheErr := cache_service.SetCacheUserInfo(address, userInfoResponse)
	if cacheErr != nil {
		log.Printf("[CACHE_SERVICE]Failed to update cache for address %s: %v", address, cacheErr)
	} else {
		log.Printf("[CACHE_SERVICE]Successfully updated user info cache for address: %s", address)
	}
}
