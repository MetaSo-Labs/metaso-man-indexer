package api

import (
	"fmt"
	"manindexer/basicprotocols/group_chat/api/request"
	"manindexer/basicprotocols/group_chat/api/respond"
	"manindexer/basicprotocols/group_chat/db"
	"manindexer/basicprotocols/group_chat/service"
	lucky_bag_service "manindexer/basicprotocols/group_chat/service"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// @Summary Get community version info by communityId or pinId
// @Description Query TalkCommunityVersionInfoCollection data by communityId or pinId
// @Tags Database Query
// @Accept json
// @Produce json
// @Param communityId query string false "Community ID"
// @Param pinId query string false "Pin ID"
// @Param limit query int false "Limit count" default(10)
// @Success 200 {object} map[string]interface{} "Query result"
// @Failure 400 {object} map[string]interface{} "Parameter error"
// @Failure 500 {object} map[string]interface{} "Server error"
// @Router /api/db/community/version [get]
func GetCommunityVersionInfo(ctx *gin.Context) {
	var t = time.Now().UnixMilli()
	communityId := ctx.Query("communityId")
	pinId := ctx.Query("pinId")
	limitStr := ctx.DefaultQuery("limit", "10")

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, respond.RespErr(fmt.Errorf("limit parameter must be a number"), t, 1))
		return
	}

	var results []map[string]interface{}
	var queryErr error

	if communityId != "" {
		prefix := communityId + "_"
		results, queryErr = service.QueryByPrefix("talk_community_version_info", prefix, limit)
	} else if pinId != "" {
		prefix := pinId + "_"
		results, queryErr = service.QueryByPrefix("talk_community_version_info", prefix, limit)
	} else {
		ctx.JSON(http.StatusBadRequest, respond.RespErr(fmt.Errorf("must provide communityId or pinId parameter"), t, 1))
		return
	}

	if queryErr != nil {
		ctx.JSON(http.StatusInternalServerError, respond.RespErr(queryErr, t, 1))
		return
	}

	ctx.JSON(http.StatusOK, respond.RespSuccess(gin.H{
		"data":  results,
		"count": len(results),
	}, t))
}

// @Summary Get community info by communityId
// @Description Query TalkCommunityInfoCollection data by communityId
// @Tags Database Query
// @Accept json
// @Produce json
// @Param communityId query string true "Community ID"
// @Success 200 {object} map[string]interface{} "Query result"
// @Failure 400 {object} map[string]interface{} "Parameter error"
// @Failure 500 {object} map[string]interface{} "Server error"
// @Router /api/db/community/info [get]
func GetCommunityInfo(ctx *gin.Context) {
	var t = time.Now().UnixMilli()
	communityId := ctx.Query("communityId")
	if communityId == "" {
		ctx.JSON(http.StatusBadRequest, respond.RespErr(fmt.Errorf("communityId parameter cannot be empty"), t, 1))
		return
	}

	result, err := service.QueryCommunityInfo(communityId)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, respond.RespErr(err, t, 1))
		return
	}

	ctx.JSON(http.StatusOK, respond.RespSuccess(result, t))
}

// @Summary Get community join records by communityId or pinId
// @Description Query TalkCommunityJoinCollection data by communityId or pinId
// @Tags Database Query
// @Accept json
// @Produce json
// @Param communityId query string false "Community ID"
// @Param pinId query string false "Pin ID"
// @Param limit query int false "Limit count" default(10)
// @Success 200 {object} map[string]interface{} "Query result"
// @Failure 400 {object} map[string]interface{} "Parameter error"
// @Failure 500 {object} map[string]interface{} "Server error"
// @Router /api/db/community/join [get]
func GetCommunityJoin(ctx *gin.Context) {
	var t = time.Now().UnixMilli()
	communityId := ctx.Query("communityId")
	pinId := ctx.Query("pinId")
	limitStr := ctx.DefaultQuery("limit", "10")

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, respond.RespErr(fmt.Errorf("limit parameter must be a number"), t, 1))
		return
	}

	var results []map[string]interface{}
	var queryErr error

	if communityId != "" {
		results, queryErr = service.QueryCommunityJoin(communityId, limit)
	} else if pinId != "" {
		prefix := pinId + "_"
		results, queryErr = service.QueryByPrefix("talk_community_join", prefix, limit)
	} else {
		ctx.JSON(http.StatusBadRequest, respond.RespErr(fmt.Errorf("must provide communityId or pinId parameter"), t, 1))
		return
	}

	if queryErr != nil {
		ctx.JSON(http.StatusInternalServerError, respond.RespErr(queryErr, t, 1))
		return
	}

	ctx.JSON(http.StatusOK, respond.RespSuccess(gin.H{
		"data":  results,
		"count": len(results),
	}, t))
}

// @Summary Get community person list by communityId or metaId
// @Description Query TalkCommunityPersonCollection data by communityId or metaId
// @Tags Database Query
// @Accept json
// @Produce json
// @Param communityId query string false "Community ID"
// @Param metaId query string false "Meta ID"
// @Param limit query int false "Limit count" default(10)
// @Success 200 {object} map[string]interface{} "Query result"
// @Failure 400 {object} map[string]interface{} "Parameter error"
// @Failure 500 {object} map[string]interface{} "Server error"
// @Router /api/db/community/person [get]
func GetCommunityPerson(ctx *gin.Context) {
	var t = time.Now().UnixMilli()
	communityId := ctx.Query("communityId")
	metaId := ctx.Query("metaId")
	limitStr := ctx.DefaultQuery("limit", "10")

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, respond.RespErr(fmt.Errorf("limit parameter must be a number"), t, 1))
		return
	}

	var results []map[string]interface{}
	var queryErr error

	if communityId != "" {
		results, queryErr = service.QueryCommunityPerson(communityId, limit)
	} else if metaId != "" {
		prefix := metaId + "_"
		results, queryErr = service.QueryByPrefix("talk_community_person", prefix, limit)
	} else {
		ctx.JSON(http.StatusBadRequest, respond.RespErr(fmt.Errorf("must provide communityId or metaId parameter"), t, 1))
		return
	}

	if queryErr != nil {
		ctx.JSON(http.StatusInternalServerError, respond.RespErr(queryErr, t, 1))
		return
	}

	ctx.JSON(http.StatusOK, respond.RespSuccess(gin.H{
		"data":  results,
		"count": len(results),
	}, t))
}

// @Summary Get group info by groupId
// @Description Query TalkGroupInfoCollection data by groupId
// @Tags Database Query
// @Accept json
// @Produce json
// @Param groupId query string true "Group ID"
// @Success 200 {object} map[string]interface{} "Query result"
// @Failure 400 {object} map[string]interface{} "Parameter error"
// @Failure 500 {object} map[string]interface{} "Server error"
// @Router /api/db/group/info [get]
func GetDbGroupInfo(ctx *gin.Context) {
	var t = time.Now().UnixMilli()
	groupId := ctx.Query("groupId")
	if groupId == "" {
		ctx.JSON(http.StatusBadRequest, respond.RespErr(fmt.Errorf("groupId parameter cannot be empty"), t, 1))
		return
	}

	result, err := service.QueryGroupInfo(groupId)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, respond.RespErr(err, t, 1))
		return
	}

	ctx.JSON(http.StatusOK, respond.RespSuccess(result, t))
}

// @Summary Get group version info by groupId or pinId
// @Description Query TalkGroupVersionInfoCollection data by groupId or pinId
// @Tags Database Query
// @Accept json
// @Produce json
// @Param groupId query string false "Group ID"
// @Param pinId query string false "Pin ID"
// @Param limit query int false "Limit count" default(10)
// @Success 200 {object} map[string]interface{} "Query result"
// @Failure 400 {object} map[string]interface{} "Parameter error"
// @Failure 500 {object} map[string]interface{} "Server error"
// @Router /api/db/group/version [get]
func GetGroupVersionInfo(ctx *gin.Context) {
	var t = time.Now().UnixMilli()
	groupId := ctx.Query("groupId")
	pinId := ctx.Query("pinId")
	limitStr := ctx.DefaultQuery("limit", "10")

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, respond.RespErr(fmt.Errorf("limit parameter must be a number"), t, 1))
		return
	}

	var results []map[string]interface{}
	var queryErr error

	if groupId != "" {
		results, queryErr = service.QueryGroupVersionInfo(groupId, limit)
	} else if pinId != "" {
		prefix := pinId + "_"
		results, queryErr = service.QueryByPrefix("talk_group_version_info", prefix, limit)
	} else {
		ctx.JSON(http.StatusBadRequest, respond.RespErr(fmt.Errorf("must provide groupId or pinId parameter"), t, 1))
		return
	}

	if queryErr != nil {
		ctx.JSON(http.StatusInternalServerError, respond.RespErr(queryErr, t, 1))
		return
	}

	ctx.JSON(http.StatusOK, respond.RespSuccess(gin.H{
		"data":  results,
		"count": len(results),
	}, t))
}

// @Summary Get group join records by groupId or pinId
// @Description Query TalkGroupJoinCollection data by groupId or pinId
// @Tags Database Query
// @Accept json
// @Produce json
// @Param groupId query string false "Group ID"
// @Param pinId query string false "Pin ID"
// @Param limit query int false "Limit count" default(10)
// @Success 200 {object} map[string]interface{} "Query result"
// @Failure 400 {object} map[string]interface{} "Parameter error"
// @Failure 500 {object} map[string]interface{} "Server error"
// @Router /api/db/group/join [get]
func GetGroupJoin(ctx *gin.Context) {
	var t = time.Now().UnixMilli()
	groupId := ctx.Query("groupId")
	pinId := ctx.Query("pinId")
	limitStr := ctx.DefaultQuery("limit", "10")

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, respond.RespErr(fmt.Errorf("limit parameter must be a number"), t, 1))
		return
	}

	var results []map[string]interface{}
	var queryErr error

	if groupId != "" {
		results, queryErr = service.QueryGroupJoin(groupId, limit)
	} else if pinId != "" {
		prefix := pinId + "_"
		results, queryErr = service.QueryByPrefix("talk_group_join", prefix, limit)
	} else {
		ctx.JSON(http.StatusBadRequest, respond.RespErr(fmt.Errorf("must provide groupId or pinId parameter"), t, 1))
		return
	}

	if queryErr != nil {
		ctx.JSON(http.StatusInternalServerError, respond.RespErr(queryErr, t, 1))
		return
	}

	ctx.JSON(http.StatusOK, respond.RespSuccess(gin.H{
		"data":  results,
		"count": len(results),
	}, t))
}

// @Summary Get group person list by groupId or metaId
// @Description Query TalkGroupPersonCollection data by groupId or metaId
// @Tags Database Query
// @Accept json
// @Produce json
// @Param groupId query string false "Group ID"
// @Param metaId query string false "Meta ID"
// @Param limit query int false "Limit count" default(10)
// @Success 200 {object} map[string]interface{} "Query result"
// @Failure 400 {object} map[string]interface{} "Parameter error"
// @Failure 500 {object} map[string]interface{} "Server error"
// @Router /api/db/group/person [get]
func GetDbGroupPerson(ctx *gin.Context) {
	var t = time.Now().UnixMilli()
	groupId := ctx.Query("groupId")
	metaId := ctx.Query("metaId")
	limitStr := ctx.DefaultQuery("limit", "10")

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, respond.RespErr(fmt.Errorf("limit parameter must be a number"), t, 1))
		return
	}

	var results []map[string]interface{}
	var queryErr error

	if groupId != "" {
		results, queryErr = service.QueryGroupPerson(groupId, limit)
	} else if metaId != "" {
		prefix := metaId + "_"
		results, queryErr = service.QueryByPrefix("talk_group_person", prefix, limit)
	} else {
		ctx.JSON(http.StatusBadRequest, respond.RespErr(fmt.Errorf("must provide groupId or metaId parameter"), t, 1))
		return
	}

	if queryErr != nil {
		ctx.JSON(http.StatusInternalServerError, respond.RespErr(queryErr, t, 1))
		return
	}

	ctx.JSON(http.StatusOK, respond.RespSuccess(gin.H{
		"data":  results,
		"count": len(results),
	}, t))
}

// @Summary Get chat queue list by timestamp
// @Description Query TalkGroupChatQueueCollection data by timestamp, or get all data without timestamp
// @Tags Database Query
// @Accept json
// @Produce json
// @Param timestamp query string false "Timestamp"
// @Param limit query int false "Limit count" default(10)
// @Success 200 {object} map[string]interface{} "Query result"
// @Failure 400 {object} map[string]interface{} "Parameter error"
// @Failure 500 {object} map[string]interface{} "Server error"
// @Router /api/db/chat/queue [get]
func GetChatQueue(ctx *gin.Context) {
	var t = time.Now().UnixMilli()
	timestamp := ctx.Query("timestamp")
	limitStr := ctx.DefaultQuery("limit", "10")

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, respond.RespErr(fmt.Errorf("limit parameter must be a number"), t, 1))
		return
	}

	var results []map[string]interface{}
	var queryErr error
	var totalCount int

	if timestamp != "" {
		prefix := timestamp + "_"
		results, queryErr = service.QueryByPrefix("talk_group_chat_queue", prefix, limit)
		// For prefix query, we need to count all records with that prefix
		if queryErr == nil {
			allResults, countErr := service.QueryByPrefix("talk_group_chat_queue", prefix, 0) // Get all with prefix
			if countErr == nil {
				totalCount = len(allResults)
			}
		}
	} else {
		results, queryErr = service.QueryGroupChatQueue(limit)
		// Get total count for the entire collection
		if queryErr == nil {
			totalCount, err = service.GetCollectionCount("talk_group_chat_queue")
			if err != nil {
				totalCount = 0
			}
		}
	}

	if queryErr != nil {
		ctx.JSON(http.StatusInternalServerError, respond.RespErr(queryErr, t, 1))
		return
	}

	ctx.JSON(http.StatusOK, respond.RespSuccess(gin.H{
		"data":       results,
		"count":      len(results),
		"totalCount": totalCount,
	}, t))
}

// @Summary Get chat message by pinId
// @Description Query TalkGroupChatPinCollection data by pinId
// @Tags Database Query
// @Accept json
// @Produce json
// @Param pinId query string true "Pin ID"
// @Success 200 {object} map[string]interface{} "Query result"
// @Failure 400 {object} map[string]interface{} "Parameter error"
// @Failure 500 {object} map[string]interface{} "Server error"
// @Router /api/db/chat/pin [get]
func GetChatPin(ctx *gin.Context) {
	var t = time.Now().UnixMilli()
	pinId := ctx.Query("pinId")
	if pinId == "" {
		ctx.JSON(http.StatusBadRequest, respond.RespErr(fmt.Errorf("pinId parameter cannot be empty"), t, 1))
		return
	}

	result, err := service.QueryGroupChatPin(pinId)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, respond.RespErr(err, t, 1))
		return
	}

	ctx.JSON(http.StatusOK, respond.RespSuccess(result, t))
}

// @Summary Get chat timestamp list by groupId
// @Description Query TalkGroupChatTimestampCollection data by groupId
// @Tags Database Query
// @Accept json
// @Produce json
// @Param groupId query string true "Group ID"
// @Param limit query int false "Limit count" default(10)
// @Success 200 {object} map[string]interface{} "Query result"
// @Failure 400 {object} map[string]interface{} "Parameter error"
// @Failure 500 {object} map[string]interface{} "Server error"
// @Router /api/db/chat/timestamp [get]
func GetChatTimestamp(ctx *gin.Context) {
	var t = time.Now().UnixMilli()
	groupId := ctx.Query("groupId")
	if groupId == "" {
		ctx.JSON(http.StatusBadRequest, respond.RespErr(fmt.Errorf("groupId parameter cannot be empty"), t, 1))
		return
	}

	limitStr := ctx.DefaultQuery("limit", "10")
	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, respond.RespErr(fmt.Errorf("limit parameter must be a number"), t, 1))
		return
	}

	results, err := service.QueryGroupChatByTimestamp(groupId, 0, 0, limit)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, respond.RespErr(err, t, 1))
		return
	}

	ctx.JSON(http.StatusOK, respond.RespSuccess(gin.H{
		"data":  results,
		"count": len(results),
	}, t))
}

// @Summary Get user group list by metaId
// @Description Query TalkMetaIdContextListCollection data by metaId
// @Tags Database Query
// @Accept json
// @Produce json
// @Param metaId query string true "Meta ID"
// @Success 200 {object} map[string]interface{} "Query result"
// @Failure 400 {object} map[string]interface{} "Parameter error"
// @Failure 500 {object} map[string]interface{} "Server error"
// @Router /api/db/user/context [get]
func GetUserContext(ctx *gin.Context) {
	var t = time.Now().UnixMilli()
	metaId := ctx.Query("metaId")
	if metaId == "" {
		ctx.JSON(http.StatusBadRequest, respond.RespErr(fmt.Errorf("metaId parameter cannot be empty"), t, 1))
		return
	}

	result, err := service.QueryMetaIdContextList(metaId)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, respond.RespErr(err, t, 1))
		return
	}

	ctx.JSON(http.StatusOK, respond.RespSuccess(result, t))
}

// @Summary Get database statistics
// @Description Get statistics for all database collections
// @Tags Database Query
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{} "Statistics"
// @Failure 500 {object} map[string]interface{} "Server error"
// @Router /api/db/stats [get]
func GetDatabaseStats(ctx *gin.Context) {
	var t = time.Now().UnixMilli()
	stats, err := service.GetDatabaseStats()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, respond.RespErr(err, t, 1))
		return
	}

	ctx.JSON(http.StatusOK, respond.RespSuccess(stats, t))
}

// @Summary Get all available database collections
// @Description Get all available database collection names
// @Tags Database Query
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{} "Collection list"
// @Router /api/db/collections [get]
func GetCollections(ctx *gin.Context) {
	var t = time.Now().UnixMilli()
	collections := service.GetAvailableCollections()

	ctx.JSON(http.StatusOK, respond.RespSuccess(gin.H{
		"data":  collections,
		"count": len(collections),
	}, t))
}

// @Summary Get all group version info list (pagination)
// @Description Get all data of TalkGroupVersionInfoCollection, support pagination
// @Tags Database Query
// @Accept json
// @Produce json
// @Param page query int false "Page number, starting from 1" default(1)
// @Param size query int false "Number of items per page" default(20)
// @Success 200 {object} map[string]interface{} "Query result"
// @Failure 400 {object} map[string]interface{} "Parameter error"
// @Failure 500 {object} map[string]interface{} "Server error"
// @Router /api/db/group/version/all [get]
func GetAllGroupVersionInfo(ctx *gin.Context) {
	var t = time.Now().UnixMilli()
	pageStr := ctx.DefaultQuery("page", "1")
	sizeStr := ctx.DefaultQuery("size", "20")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		ctx.JSON(http.StatusBadRequest, respond.RespErr(fmt.Errorf("page parameter must be a number greater than 0"), t, 1))
		return
	}

	size, err := strconv.Atoi(sizeStr)
	if err != nil || size < 1 || size > 100 {
		ctx.JSON(http.StatusBadRequest, respond.RespErr(fmt.Errorf("size parameter must be a number between 1-100"), t, 1))
		return
	}

	// Calculate the number of records to skip
	skip := (page - 1) * size

	// Get total data count (for pagination info)
	totalCount, err := service.GetCollectionCount("talk_group_version_info")
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, respond.RespErr(err, t, 1))
		return
	}

	// Get paginated data
	results, err := service.QueryAll("talk_group_version_info", skip+size)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, respond.RespErr(err, t, 1))
		return
	}

	// Skip previous records to get the data for the current page
	var pageResults []map[string]interface{}
	if skip < len(results) {
		end := skip + size
		if end > len(results) {
			end = len(results)
		}
		pageResults = results[skip:end]
	}

	// Calculate pagination info
	totalPages := (totalCount + size - 1) / size

	ctx.JSON(http.StatusOK, respond.RespSuccess(gin.H{
		"data": pageResults,
		"pagination": gin.H{
			"page":        page,
			"size":        size,
			"total":       totalCount,
			"total_pages": totalPages,
			"has_next":    page < totalPages,
			"has_prev":    page > 1,
		},
	}, t))
}

// @Summary Get all chat message list (pagination)
// @Description Get all data of TalkGroupChatPinCollection, support pagination
// @Tags Database Query
// @Accept json
// @Produce json
// @Param page query int false "Page number, starting from 1" default(1)
// @Param size query int false "Number of items per page" default(20)
// @Success 200 {object} map[string]interface{} "Query result"
// @Failure 400 {object} map[string]interface{} "Parameter error"
// @Failure 500 {object} map[string]interface{} "Server error"
// @Router /api/db/chat/pin/all [get]
func GetAllChatPin(ctx *gin.Context) {
	var t = time.Now().UnixMilli()
	pageStr := ctx.DefaultQuery("page", "1")
	sizeStr := ctx.DefaultQuery("size", "20")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		ctx.JSON(http.StatusBadRequest, respond.RespErr(fmt.Errorf("page parameter must be a number greater than 0"), t, 1))
		return
	}

	size, err := strconv.Atoi(sizeStr)
	if err != nil || size < 1 || size > 100 {
		ctx.JSON(http.StatusBadRequest, respond.RespErr(fmt.Errorf("size parameter must be a number between 1-100"), t, 1))
		return
	}

	// Calculate the number of records to skip
	skip := (page - 1) * size

	// Get total data count (for pagination info)
	totalCount, err := service.GetCollectionCount("talk_group_chat_pin")
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, respond.RespErr(err, t, 1))
		return
	}

	// Get paginated data
	results, err := service.QueryAll("talk_group_chat_pin", skip+size)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, respond.RespErr(err, t, 1))
		return
	}

	// Skip previous records to get the data for the current page
	var pageResults []map[string]interface{}
	if skip < len(results) {
		end := skip + size
		if end > len(results) {
			end = len(results)
		}
		pageResults = results[skip:end]
	}

	// Calculate pagination info
	totalPages := (totalCount + size - 1) / size

	ctx.JSON(http.StatusOK, respond.RespSuccess(gin.H{
		"data": pageResults,
		"pagination": gin.H{
			"page":        page,
			"size":        size,
			"total":       totalCount,
			"total_pages": totalPages,
			"has_next":    page < totalPages,
			"has_prev":    page > 1,
		},
	}, t))
}

// @Summary Get lucky bag statistics by group and time range
// @Description Get comprehensive lucky bag statistics for a specific group or all groups within a time range
// @Tags Database Query
// @Accept json
// @Produce json
// @Param groupId query string false "Group ID (leave empty to get statistics for all groups)"
// @Param startTime query int64 true "Start timestamp (Unix timestamp)"
// @Param endTime query int64 true "End timestamp (Unix timestamp)"
// @Success 200 {object} map[string]interface{} "Lucky bag statistics"
// @Failure 400 {object} map[string]interface{} "Parameter error"
// @Failure 500 {object} map[string]interface{} "Server error"
// @Router /api/db/luckybag/statistics [get]
func GetLuckyBagStatistics(ctx *gin.Context) {
	var t = time.Now().UnixMilli()

	groupId := ctx.Query("groupId")
	// groupId can be empty to get statistics for all groups

	startTimeStr := ctx.Query("startTime")
	if startTimeStr == "" {
		ctx.JSON(http.StatusBadRequest, respond.RespErr(fmt.Errorf("startTime parameter cannot be empty"), t, 1))
		return
	}

	endTimeStr := ctx.Query("endTime")
	if endTimeStr == "" {
		ctx.JSON(http.StatusBadRequest, respond.RespErr(fmt.Errorf("endTime parameter cannot be empty"), t, 1))
		return
	}

	startTime, err := strconv.ParseInt(startTimeStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, respond.RespErr(fmt.Errorf("startTime parameter must be a valid timestamp"), t, 1))
		return
	}

	endTime, err := strconv.ParseInt(endTimeStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, respond.RespErr(fmt.Errorf("endTime parameter must be a valid timestamp"), t, 1))
		return
	}

	if startTime >= endTime {
		ctx.JSON(http.StatusBadRequest, respond.RespErr(fmt.Errorf("startTime must be less than endTime"), t, 1))
		return
	}

	stats, err := service.GetLuckyBagStatisticsByGroupAndTimeRange(groupId, startTime, endTime)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, respond.RespErr(err, t, 1))
		return
	}

	ctx.JSON(http.StatusOK, respond.RespSuccess(stats, t))
}

// @Summary Get database migration information
// @Description Get comprehensive database migration information including current status, supported migrations, and migration history
// @Tags Database Query
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{} "Migration information"
// @Failure 500 {object} map[string]interface{} "Server error"
// @Router /api/db/migration/info [get]
func GetMigrationInfo(ctx *gin.Context) {
	var t = time.Now().UnixMilli()

	info, err := db.GetMigrationInfo()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, respond.RespErr(err, t, 1))
		return
	}

	ctx.JSON(http.StatusOK, respond.RespSuccess(info, t))
}

// @Summary Get lucky bag lock statistics
// @Description Get comprehensive statistics about lucky bag locks including total locks, active locks, and inactive locks
// @Tags Database Query
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{} "Lucky bag lock statistics"
// @Failure 500 {object} map[string]interface{} "Server error"
// @Router /api/db/luckybag/lock-stats [get]
func GetLuckyBagLockStats(ctx *gin.Context) {
	var t = time.Now().UnixMilli()

	stats, err := service.GetLuckyBagLockStats()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, respond.RespErr(err, t, 1))
		return
	}

	ctx.JSON(http.StatusOK, respond.RespSuccess(stats, t))
}

// @Summary Get group chat index list
// @Description Get TalkGroupChatIndexCollection list with cursor pagination and reverse order
// @Tags Database Query
// @Accept json
// @Produce json
// @Param cursor query int false "Cursor, starting from 0" default(0)
// @Param size query int false "Number of items per page" default(20)
// @Param groupId query string false "Group ID for filtering (optional)"
// @Success 200 {object} map[string]interface{} "Group chat index list"
// @Failure 400 {object} map[string]interface{} "Parameter error"
// @Failure 500 {object} map[string]interface{} "Server error"
// @Router /api/db/chat/index/group [get]
func GetGroupChatIndexList(ctx *gin.Context) {
	var t = time.Now().UnixMilli()

	cursorStr := ctx.DefaultQuery("cursor", "0")
	sizeStr := ctx.DefaultQuery("size", "20")
	groupId := ctx.Query("groupId")

	cursor, err := strconv.Atoi(cursorStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, respond.RespErr(fmt.Errorf("cursor parameter must be a number"), t, 1))
		return
	}

	size, err := strconv.Atoi(sizeStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, respond.RespErr(fmt.Errorf("size parameter must be a number"), t, 1))
		return
	}

	results, err := service.GetGroupChatIndexList(cursor, size, groupId)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, respond.RespErr(err, t, 1))
		return
	}

	ctx.JSON(http.StatusOK, respond.RespSuccess(results, t))
}

// @Summary Get group chat index list
// @Description Get TalkGroupChatIndexCollection list with cursor pagination and reverse order
// @Tags Database Query
// @Accept json
// @Produce json
// @Param cursor query int false "Cursor, starting from 0" default(0)
// @Param size query int false "Number of items per page" default(20)
// @Param groupId query string false "Group ID for filtering (optional)"
// @Success 200 {object} map[string]interface{} "Group chat index list"
// @Failure 400 {object} map[string]interface{} "Parameter error"
// @Failure 500 {object} map[string]interface{} "Server error"
// @Router /api/db/chat/index/group [get]
func GetGroupChannelChatIndexList(ctx *gin.Context) {
	var t = time.Now().UnixMilli()

	cursorStr := ctx.DefaultQuery("cursor", "0")
	sizeStr := ctx.DefaultQuery("size", "20")
	channelId := ctx.Query("channelId")

	cursor, err := strconv.Atoi(cursorStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, respond.RespErr(fmt.Errorf("cursor parameter must be a number"), t, 1))
		return
	}

	size, err := strconv.Atoi(sizeStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, respond.RespErr(fmt.Errorf("size parameter must be a number"), t, 1))
		return
	}

	results, err := service.GetGroupChannelChatIndexList(cursor, size, channelId)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, respond.RespErr(err, t, 1))
		return
	}

	ctx.JSON(http.StatusOK, respond.RespSuccess(results, t))
}

// @Summary Get private chat index list
// @Description Get TalkPrivateChatIndexCollection list with cursor pagination and reverse order
// @Tags Database Query
// @Accept json
// @Produce json
// @Param cursor query int false "Cursor, starting from 0" default(0)
// @Param size query int false "Number of items per page" default(20)
// @Param fromTo query string false "From MetaId to To MetaId for filtering (format: fromMetaId_toMetaId, optional)"
// @Success 200 {object} map[string]interface{} "Private chat index list"
// @Failure 400 {object} map[string]interface{} "Parameter error"
// @Failure 500 {object} map[string]interface{} "Server error"
// @Router /api/db/chat/index/private [get]
func GetPrivateChatIndexList(ctx *gin.Context) {
	var t = time.Now().UnixMilli()

	cursorStr := ctx.DefaultQuery("cursor", "0")
	sizeStr := ctx.DefaultQuery("size", "20")
	fromTo := ctx.Query("fromTo")

	cursor, err := strconv.Atoi(cursorStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, respond.RespErr(fmt.Errorf("cursor parameter must be a number"), t, 1))
		return
	}

	size, err := strconv.Atoi(sizeStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, respond.RespErr(fmt.Errorf("size parameter must be a number"), t, 1))
		return
	}

	results, err := service.GetPrivateChatIndexList(cursor, size, fromTo)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, respond.RespErr(err, t, 1))
		return
	}

	ctx.JSON(http.StatusOK, respond.RespSuccess(results, t))
}

// @Summary Get group chat index keys
// @Description Get TalkGroupChatIndexCollection key list with cursor pagination
// @Tags Database Query
// @Accept json
// @Produce json
// @Param cursor query int false "Cursor, starting from 0" default(0)
// @Param size query int false "Number of items per page" default(20)
// @Param groupId query string false "Group ID for filtering (optional)"
// @Success 200 {object} map[string]interface{} "Group chat index keys"
// @Failure 400 {object} map[string]interface{} "Parameter error"
// @Failure 500 {object} map[string]interface{} "Server error"
// @Router /api/db/chat/index/group/keys [get]
func GetGroupChatIndexKeys(ctx *gin.Context) {
	var t = time.Now().UnixMilli()

	cursorStr := ctx.DefaultQuery("cursor", "0")
	sizeStr := ctx.DefaultQuery("size", "20")
	groupId := ctx.Query("groupId")

	cursor, err := strconv.Atoi(cursorStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, respond.RespErr(fmt.Errorf("cursor parameter must be a number"), t, 1))
		return
	}

	size, err := strconv.Atoi(sizeStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, respond.RespErr(fmt.Errorf("size parameter must be a number"), t, 1))
		return
	}

	results, err := service.GetGroupChatIndexKeys(cursor, size, groupId)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, respond.RespErr(err, t, 1))
		return
	}

	ctx.JSON(http.StatusOK, respond.RespSuccess(results, t))
}

// @Summary Get group channel chat index keys
// @Description Get TalkGroupChatIndexCollection key list with cursor pagination
// @Tags Database Query
// @Accept json
// @Produce json
// @Param cursor query int false "Cursor, starting from 0" default(0)
// @Param size query int false "Number of items per page" default(20)
// @Param channelId query string false "Channel ID for filtering (optional)"
// @Success 200 {object} map[string]interface{} "Group channel chat index keys"
// @Failure 400 {object} map[string]interface{} "Parameter error"
// @Failure 500 {object} map[string]interface{} "Server error"
// @Router /api/db/chat/index/channel/keys [get]
func GetGroupChannelChatIndexKeys(ctx *gin.Context) {
	var t = time.Now().UnixMilli()

	cursorStr := ctx.DefaultQuery("cursor", "0")
	sizeStr := ctx.DefaultQuery("size", "20")
	channelId := ctx.Query("channelId")

	cursor, err := strconv.Atoi(cursorStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, respond.RespErr(fmt.Errorf("cursor parameter must be a number"), t, 1))
		return
	}

	size, err := strconv.Atoi(sizeStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, respond.RespErr(fmt.Errorf("size parameter must be a number"), t, 1))
		return
	}

	results, err := service.GetGroupChannelChatIndexKeys(cursor, size, channelId)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, respond.RespErr(err, t, 1))
		return
	}

	ctx.JSON(http.StatusOK, respond.RespSuccess(results, t))
}

// @Summary Get group chat timestamp2 out list
// @Description Get TalkGroupChatTimestamp2OutCollection list with cursor pagination and reverse order
// @Tags Database Query
// @Accept json
// @Produce json
// @Param cursor query int false "Cursor, starting from 0" default(0)
// @Param size query int false "Number of items per page" default(20)
// @Param groupId query string false "Group ID for filtering (optional)"
// @Success 200 {object} map[string]interface{} "Group chat timestamp2 out list"
// @Failure 400 {object} map[string]interface{} "Parameter error"
// @Failure 500 {object} map[string]interface{} "Server error"
// @Router /api/db/chat/timestamp2/out [get]
func GetGroupChatTimestamp2OutList(ctx *gin.Context) {
	var t = time.Now().UnixMilli()

	cursorStr := ctx.DefaultQuery("cursor", "0")
	sizeStr := ctx.DefaultQuery("size", "20")
	groupId := ctx.Query("groupId")

	cursor, err := strconv.Atoi(cursorStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, respond.RespErr(fmt.Errorf("cursor parameter must be a number"), t, 1))
		return
	}

	size, err := strconv.Atoi(sizeStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, respond.RespErr(fmt.Errorf("size parameter must be a number"), t, 1))
		return
	}

	results, err := service.GetGroupChatTimestamp2OutList(cursor, size, groupId)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, respond.RespErr(err, t, 1))
		return
	}

	ctx.JSON(http.StatusOK, respond.RespSuccess(results, t))
}

// @Summary Get group channel chat timestamp2 out list
// @Description Get TalkGroupChannelChatTimestamp2OutCollection list with cursor pagination and reverse order
// @Tags Database Query
// @Accept json
// @Produce json
// @Param cursor query int false "Cursor, starting from 0" default(0)
// @Param size query int false "Number of items per page" default(20)
// @Param channelId query string false "Channel ID for filtering (optional)"
// @Success 200 {object} map[string]interface{} "Group channel chat timestamp2 out list"
// @Failure 400 {object} map[string]interface{} "Parameter error"
// @Failure 500 {object} map[string]interface{} "Server error"
// @Router /api/db/chat/timestamp2/out/channel [get]
func GetGroupChannelChatTimestampOutList(ctx *gin.Context) {
	var t = time.Now().UnixMilli()

	cursorStr := ctx.DefaultQuery("cursor", "0")
	sizeStr := ctx.DefaultQuery("size", "20")
	channelId := ctx.Query("channelId")

	cursor, err := strconv.Atoi(cursorStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, respond.RespErr(fmt.Errorf("cursor parameter must be a number"), t, 1))
		return
	}

	size, err := strconv.Atoi(sizeStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, respond.RespErr(fmt.Errorf("size parameter must be a number"), t, 1))
		return
	}

	results, err := service.GetGroupChannelChatTimestampOutList(cursor, size, channelId)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, respond.RespErr(err, t, 1))
		return
	}

	ctx.JSON(http.StatusOK, respond.RespSuccess(results, t))
}

// @Summary Get detailed open lucky bag list by lucky bag PinId
// @Description Get detailed open lucky bag list with grab state, user info, and lucky bag details
// @Tags Database Query
// @Accept json
// @Produce json
// @Param luckyBagPinId query string true "Lucky bag PinId"
// @Success 200 {object} map[string]interface{} "Detailed query result with grab state, user info, and lucky bag details"
// @Failure 400 {object} map[string]interface{} "Parameter error"
// @Failure 500 {object} map[string]interface{} "Server error"
// @Router /api/db/luckybag/open/list [get]
func GetOpenLuckyBagList(ctx *gin.Context) {
	var t = time.Now().UnixMilli()
	luckyBagPinId := ctx.Query("luckyBagPinId")
	if luckyBagPinId == "" {
		ctx.JSON(http.StatusBadRequest, respond.RespErr(fmt.Errorf("luckyBagPinId parameter cannot be empty"), t, 1))
		return
	}

	result, err := service.GetDetailedOpenLuckyBagList(luckyBagPinId)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, respond.RespErr(err, t, 1))
		return
	}

	ctx.JSON(http.StatusOK, respond.RespSuccess(result, t))
}

// @Summary Get MetaId join list by metaId
// @Description Query TalkGroupMetaIdJoinCollection data by metaId
// @Tags Database Query
// @Accept json
// @Produce json
// @Param metaId query string true "MetaId"
// @Param groupId query string false "Group ID to filter by (optional)"
// @Success 200 {object} map[string]interface{} "MetaId join list with detailed information"
// @Failure 400 {object} map[string]interface{} "Parameter error"
// @Failure 500 {object} map[string]interface{} "Server error"
// @Router /api/db/metaid/join [get]
func GetMetaIdJoinList(ctx *gin.Context) {
	var t = time.Now().UnixMilli()
	metaId := ctx.Query("metaId")
	groupId := ctx.Query("groupId") // Optional groupId parameter

	if metaId == "" {
		ctx.JSON(http.StatusBadRequest, respond.RespErr(fmt.Errorf("metaId parameter cannot be empty"), t, 1))
		return
	}

	result, err := service.QueryMetaIdJoinList(metaId, groupId)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, respond.RespErr(err, t, 1))
		return
	}

	ctx.JSON(http.StatusOK, respond.RespSuccess(result, t))
}

// @Summary Update lucky bag validation
// @Description Update lucky bag validation counts and lists by lucky bag PinId
// @Tags Database Query
// @Accept json
// @Produce json
// @Param luckyBagPinId query string true "Lucky bag PinId"
// @Success 200 {object} map[string]interface{} "Update result with validation counts and lists"
// @Failure 400 {object} map[string]interface{} "Parameter error"
// @Failure 500 {object} map[string]interface{} "Server error"
// @Router /api/db/luckybag/update-validation [get]
func UpdateLuckyBagValidation(ctx *gin.Context) {
	var t = time.Now().UnixMilli()
	luckyBagPinId := ctx.Query("luckyBagPinId")
	if luckyBagPinId == "" {
		ctx.JSON(http.StatusBadRequest, respond.RespErr(fmt.Errorf("luckyBagPinId parameter cannot be empty"), t, 1))
		return
	}

	result, err := lucky_bag_service.UpdateLuckyBagValidation(luckyBagPinId)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, respond.RespErr(err, t, 1))
		return
	}

	ctx.JSON(http.StatusOK, respond.RespSuccess(result, t))
}

// @Summary Process expired lucky bag by pinId
// @Description Process a specific lucky bag by pinId as if it were expired, simulating the expired lucky bag processing logic
// @Tags Database Query
// @Accept json
// @Produce json
// @Param pinId query string true "Lucky bag PinId"
// @Success 200 {object} map[string]interface{} "Process result with source collection, target collection, and new state"
// @Failure 400 {object} map[string]interface{} "Parameter error"
// @Failure 500 {object} map[string]interface{} "Server error"
// @Router /api/db/luckybag/process-expired [get]
func ProcessExpiredLuckyBagByPinId(ctx *gin.Context) {
	var t = time.Now().UnixMilli()
	pinId := ctx.Query("pinId")
	if pinId == "" {
		ctx.JSON(http.StatusBadRequest, respond.RespErr(fmt.Errorf("pinId parameter cannot be empty"), t, 1))
		return
	}

	result, err := lucky_bag_service.ProcessExpiredLuckyBagByPinId(pinId)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, respond.RespErr(err, t, 1))
		return
	}

	ctx.JSON(http.StatusOK, respond.RespSuccess(result, t))
}

// @Summary Get lucky bag queue collection list with pagination
// @Description Get lucky bag queue collection list with pagination support for open lucky bag queue and residue lucky bag queue collections
// @Tags Database Query
// @Accept json
// @Produce json
// @Param collection query string true "Collection name (talk_group_open_lucky_bag_queue, talk_group_residue_lucky_bag_queue)"
// @Param cursor query int false "Cursor, starting from 0" default(0)
// @Param size query int false "Number of items per page" default(20)
// @Success 200 {object} map[string]interface{} "Lucky bag queue collection list with pagination"
// @Failure 400 {object} map[string]interface{} "Parameter error"
// @Failure 500 {object} map[string]interface{} "Server error"
// @Router /api/db/luckybag/queue/list [get]
func GetLuckyBagQueueList(ctx *gin.Context) {
	var t = time.Now().UnixMilli()
	collection := ctx.Query("collection")
	if collection == "" {
		ctx.JSON(http.StatusBadRequest, respond.RespErr(fmt.Errorf("collection parameter cannot be empty"), t, 1))
		return
	}

	cursorStr := ctx.DefaultQuery("cursor", "0")
	sizeStr := ctx.DefaultQuery("size", "20")

	cursor, err := strconv.Atoi(cursorStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, respond.RespErr(fmt.Errorf("cursor parameter must be a number"), t, 1))
		return
	}

	size, err := strconv.Atoi(sizeStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, respond.RespErr(fmt.Errorf("size parameter must be a number"), t, 1))
		return
	}

	result, err := service.GetLuckyBagQueueList(collection, cursor, size)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, respond.RespErr(err, t, 1))
		return
	}

	ctx.JSON(http.StatusOK, respond.RespSuccess(result, t))
}

// @Summary Get residue lucky bag data by pinId
// @Description Get residue lucky bag data by specific pinId from TalkGroupResidueLuckyBagPinCollection
// @Tags Database Query
// @Accept json
// @Produce json
// @Param pinId query string true "Residue lucky bag PinId"
// @Success 200 {object} map[string]interface{} "Residue lucky bag data"
// @Failure 400 {object} map[string]interface{} "Parameter error"
// @Failure 500 {object} map[string]interface{} "Server error"
// @Router /api/db/residue-luckybag/pinid [get]
func GetResidueLuckyBagByPinId(ctx *gin.Context) {
	var t = time.Now().UnixMilli()
	pinId := ctx.Query("pinId")
	if pinId == "" {
		ctx.JSON(http.StatusBadRequest, respond.RespErr(fmt.Errorf("pinId parameter cannot be empty"), t, 1))
		return
	}

	result, err := service.GetResidueLuckyBagByPinId(pinId)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, respond.RespErr(err, t, 1))
		return
	}

	ctx.JSON(http.StatusOK, respond.RespSuccess(result, t))
}

// @Summary Get residue lucky bag list with pagination
// @Description Get residue lucky bag collection list with pagination support
// @Tags Database Query
// @Accept json
// @Produce json
// @Param cursor query int false "Cursor, starting from 0" default(0)
// @Param size query int false "Number of items per page" default(20)
// @Success 200 {object} map[string]interface{} "Residue lucky bag collection list with pagination"
// @Failure 400 {object} map[string]interface{} "Parameter error"
// @Failure 500 {object} map[string]interface{} "Server error"
// @Router /api/db/residue-luckybag/list [get]
func GetResidueLuckyBagList(ctx *gin.Context) {
	var t = time.Now().UnixMilli()

	cursorStr := ctx.DefaultQuery("cursor", "0")
	sizeStr := ctx.DefaultQuery("size", "20")

	cursor, err := strconv.Atoi(cursorStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, respond.RespErr(fmt.Errorf("cursor parameter must be a number"), t, 1))
		return
	}

	size, err := strconv.Atoi(sizeStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, respond.RespErr(fmt.Errorf("size parameter must be a number"), t, 1))
		return
	}

	result, err := service.GetResidueLuckyBagList(cursor, size)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, respond.RespErr(err, t, 1))
		return
	}

	ctx.JSON(http.StatusOK, respond.RespSuccess(result, t))
}

// @Summary Get private chat timestamp list with pagination
// @Description Get private chat timestamp collection list with pagination support for specific from and to users
// @Tags Database Query
// @Accept json
// @Produce json
// @Param from query string true "From user MetaId"
// @Param to query string true "To user MetaId"
// @Param cursor query int false "Cursor, starting from 0" default(0)
// @Param size query int false "Number of items per page" default(20)
// @Success 200 {object} map[string]interface{} "Private chat timestamp collection list with pagination"
// @Failure 400 {object} map[string]interface{} "Parameter error"
// @Failure 500 {object} map[string]interface{} "Server error"
// @Router /api/db/private-chat/timestamp/list [get]
func GetPrivateChatTimestampList(ctx *gin.Context) {
	var t = time.Now().UnixMilli()
	from := ctx.Query("from")
	if from == "" {
		ctx.JSON(http.StatusBadRequest, respond.RespErr(fmt.Errorf("from parameter cannot be empty"), t, 1))
		return
	}

	to := ctx.Query("to")
	if to == "" {
		ctx.JSON(http.StatusBadRequest, respond.RespErr(fmt.Errorf("to parameter cannot be empty"), t, 1))
		return
	}

	cursorStr := ctx.DefaultQuery("cursor", "0")
	sizeStr := ctx.DefaultQuery("size", "20")

	cursor, err := strconv.Atoi(cursorStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, respond.RespErr(fmt.Errorf("cursor parameter must be a number"), t, 1))
		return
	}

	size, err := strconv.Atoi(sizeStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, respond.RespErr(fmt.Errorf("size parameter must be a number"), t, 1))
		return
	}

	result, err := service.GetPrivateChatTimestampList(from, to, cursor, size)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, respond.RespErr(err, t, 1))
		return
	}

	ctx.JSON(http.StatusOK, respond.RespSuccess(result, t))
}

// @Summary Get lucky bag collection list with pagination
// @Description Get lucky bag collection list with pagination support for pending, completed, timeout residue, error pending, and error timeout residue collections
// @Tags Database Query
// @Accept json
// @Produce json
// @Param collection query string true "Collection name (talk_group_lucky_bag_pin_pending, talk_group_lucky_bag_pin_completed, talk_group_lucky_bag_pin_timeout_residue, talk_group_lucky_bag_pin_err_pending, talk_group_lucky_bag_pin_err_timeout_residue)"
// @Param cursor query int false "Cursor, starting from 0" default(0)
// @Param size query int false "Number of items per page" default(20)
// @Success 200 {object} map[string]interface{} "Lucky bag collection list with pagination"
// @Failure 400 {object} map[string]interface{} "Parameter error"
// @Failure 500 {object} map[string]interface{} "Server error"
// @Router /api/db/luckybag/collection/list [get]
func GetLuckyBagCollectionList(ctx *gin.Context) {
	var t = time.Now().UnixMilli()
	collection := ctx.Query("collection")
	if collection == "" {
		ctx.JSON(http.StatusBadRequest, respond.RespErr(fmt.Errorf("collection parameter cannot be empty"), t, 1))
		return
	}

	cursorStr := ctx.DefaultQuery("cursor", "0")
	sizeStr := ctx.DefaultQuery("size", "20")

	cursor, err := strconv.Atoi(cursorStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, respond.RespErr(fmt.Errorf("cursor parameter must be a number"), t, 1))
		return
	}

	size, err := strconv.Atoi(sizeStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, respond.RespErr(fmt.Errorf("size parameter must be a number"), t, 1))
		return
	}

	result, err := service.GetLuckyBagCollectionList(collection, cursor, size)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, respond.RespErr(err, t, 1))
		return
	}

	ctx.JSON(http.StatusOK, respond.RespSuccess(result, t))
}

// @Summary Get lucky bag collection data by specific pinId
// @Description Get lucky bag collection data by specific pinId from any of the lucky bag collections
// @Tags Database Query
// @Accept json
// @Produce json
// @Param collection query string true "Collection name (talk_group_lucky_bag_pin_pending, talk_group_lucky_bag_pin_completed, talk_group_lucky_bag_pin_timeout_residue, talk_group_lucky_bag_pin_err_pending, talk_group_lucky_bag_pin_err_timeout_residue)"
// @Param pinId query string true "Lucky bag PinId"
// @Success 200 {object} map[string]interface{} "Lucky bag collection data by pinId"
// @Failure 400 {object} map[string]interface{} "Parameter error"
// @Failure 500 {object} map[string]interface{} "Server error"
// @Router /api/db/luckybag/collection/pinid [get]
func GetLuckyBagCollectionByPinId(ctx *gin.Context) {
	var t = time.Now().UnixMilli()
	collection := ctx.Query("collection")
	if collection == "" {
		ctx.JSON(http.StatusBadRequest, respond.RespErr(fmt.Errorf("collection parameter cannot be empty"), t, 1))
		return
	}

	pinId := ctx.Query("pinId")
	if pinId == "" {
		ctx.JSON(http.StatusBadRequest, respond.RespErr(fmt.Errorf("pinId parameter cannot be empty"), t, 1))
		return
	}

	result, err := service.GetLuckyBagCollectionByPinId(collection, pinId)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, respond.RespErr(err, t, 1))
		return
	}

	ctx.JSON(http.StatusOK, respond.RespSuccess(result, t))
}

// @Summary Get MetaId context list by metaId
// @Description Get TalkMetaIdContextListCollection data by metaId
// @Tags Database Query
// @Accept json
// @Produce json
// @Param metaId query string true "MetaId"
// @Success 200 {object} map[string]interface{} "MetaId context list data"
// @Failure 400 {object} map[string]interface{} "Parameter error"
// @Failure 500 {object} map[string]interface{} "Server error"
// @Router /api/db/metaid/context [get]
func GetMetaIdContextListByMetaId(ctx *gin.Context) {
	var t = time.Now().UnixMilli()
	metaId := ctx.Query("metaId")
	if metaId == "" {
		ctx.JSON(http.StatusBadRequest, respond.RespErr(fmt.Errorf("metaId parameter cannot be empty"), t, 1))
		return
	}

	result, err := service.GetMetaIdContextListByMetaId(metaId)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, respond.RespErr(err, t, 1))
		return
	}

	ctx.JSON(http.StatusOK, respond.RespSuccess(result, t))
}

// @Summary Get private chat timestamp out list with pagination
// @Description Get private chat timestamp out collection list with pagination support for specific from and to users
// @Tags Database Query
// @Accept json
// @Produce json
// @Param from query string true "From user MetaId"
// @Param to query string true "To user MetaId"
// @Param cursor query int false "Cursor, starting from 0" default(0)
// @Param size query int false "Number of items per page" default(20)
// @Success 200 {object} map[string]interface{} "Private chat timestamp out collection list with pagination"
// @Failure 400 {object} map[string]interface{} "Parameter error"
// @Failure 500 {object} map[string]interface{} "Server error"
// @Router /api/db/private-chat/timestamp-out/list [get]
func GetPrivateChatTimestampOutList(ctx *gin.Context) {
	var t = time.Now().UnixMilli()
	from := ctx.Query("from")
	if from == "" {
		ctx.JSON(http.StatusBadRequest, respond.RespErr(fmt.Errorf("from parameter cannot be empty"), t, 1))
		return
	}

	to := ctx.Query("to")
	if to == "" {
		ctx.JSON(http.StatusBadRequest, respond.RespErr(fmt.Errorf("to parameter cannot be empty"), t, 1))
		return
	}

	cursorStr := ctx.DefaultQuery("cursor", "0")
	sizeStr := ctx.DefaultQuery("size", "20")

	cursor, err := strconv.Atoi(cursorStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, respond.RespErr(fmt.Errorf("cursor parameter must be a number"), t, 1))
		return
	}

	size, err := strconv.Atoi(sizeStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, respond.RespErr(fmt.Errorf("size parameter must be a number"), t, 1))
		return
	}

	result, err := service.GetPrivateChatTimestampOutList(from, to, cursor, size)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, respond.RespErr(err, t, 1))
		return
	}

	ctx.JSON(http.StatusOK, respond.RespSuccess(result, t))
}

// @Summary Get group member list (DB version)
// @Description Get group member list with pagination support from database
// @Tags Database Query
// @Accept json
// @Produce json
// @Param groupId query string true "Group ID"
// @Param cursor query int false "Cursor, starting from 0" default(0)
// @Param size query int false "Number of items per page" default(20)
// @Param orderBy query string false "Order by field, use 'timestamp' for timestamp descending order"
// @Param orderType query string false "Order type, use 'desc' for descending order"
// @Success 200 {object} map[string]interface{} "Group member list"
// @Failure 400 {object} map[string]interface{} "Parameter error"
// @Failure 500 {object} map[string]interface{} "Server error"
// @Router /api/db/group/member-list [get]
func GetDbGroupMemberList(ctx *gin.Context) {
	var t = time.Now().UnixMilli()
	groupId := ctx.Query("groupId")
	if groupId == "" {
		ctx.JSON(http.StatusBadRequest, respond.RespErr(fmt.Errorf("groupId parameter cannot be empty"), t, 1))
		return
	}

	cursorStr := ctx.DefaultQuery("cursor", "0")
	sizeStr := ctx.DefaultQuery("size", "20")
	orderBy := ctx.Query("orderBy")
	orderType := ctx.Query("orderType")

	cursor, err := strconv.Atoi(cursorStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, respond.RespErr(fmt.Errorf("cursor parameter must be a number"), t, 1))
		return
	}

	size, err := strconv.Atoi(sizeStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, respond.RespErr(fmt.Errorf("size parameter must be a number"), t, 1))
		return
	}

	// Create request object
	req := &request.FetchGroupMemberListRequest{
		GroupId:   groupId,
		Cursor:    int64(cursor),
		Size:      int64(size),
		OrderBy:   orderBy,
		OrderType: orderType,
	}

	result, err := service.FetchGroupMemberList(req)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, respond.RespErr(err, t, 1))
		return
	}

	ctx.JSON(http.StatusOK, respond.RespSuccess(result, t))
}

// @Summary Get group member list V2
// @Description Get group member list using TalkGroupPersonListCollection (already sorted by timestamp descending)
// @Tags Database Query
// @Accept json
// @Produce json
// @Param groupId query string true "Group ID"
// @Param cursor query int false "Cursor, starting from 0" default(0)
// @Param size query int false "Number of items per page" default(20)
// @Success 200 {object} map[string]interface{} "Group member list V2"
// @Failure 400 {object} map[string]interface{} "Parameter error"
// @Failure 500 {object} map[string]interface{} "Server error"
// @Router /api/db/group/member-list-v2 [get]
func GetGroupMemberListV2(ctx *gin.Context) {
	var t = time.Now().UnixMilli()
	groupId := ctx.Query("groupId")
	if groupId == "" {
		ctx.JSON(http.StatusBadRequest, respond.RespErr(fmt.Errorf("groupId parameter cannot be empty"), t, 1))
		return
	}

	cursorStr := ctx.DefaultQuery("cursor", "0")
	sizeStr := ctx.DefaultQuery("size", "20")

	cursor, err := strconv.Atoi(cursorStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, respond.RespErr(fmt.Errorf("cursor parameter must be a number"), t, 1))
		return
	}

	size, err := strconv.Atoi(sizeStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, respond.RespErr(fmt.Errorf("size parameter must be a number"), t, 1))
		return
	}

	// Create request object
	req := &request.FetchGroupMemberListRequest{
		GroupId: groupId,
		Cursor:  int64(cursor),
		Size:    int64(size),
	}

	result, err := service.FetchGroupMemberListV2(req)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, respond.RespErr(err, t, 1))
		return
	}

	ctx.JSON(http.StatusOK, respond.RespSuccess(result, t))
}

// @Summary Get group person list collection with pagination
// @Description Get TalkGroupPersonListCollection data with pagination support, returns groupId and member count
// @Tags Database Query
// @Accept json
// @Produce json
// @Param cursor query int false "Cursor, starting from 0" default(0)
// @Param size query int false "Number of items per page" default(20)
// @Success 200 {object} map[string]interface{} "Group person list collection with pagination"
// @Failure 400 {object} map[string]interface{} "Parameter error"
// @Failure 500 {object} map[string]interface{} "Server error"
// @Router /api/db/group/person-list-collection [get]
func GetGroupPersonListCollection(ctx *gin.Context) {
	var t = time.Now().UnixMilli()

	cursorStr := ctx.DefaultQuery("cursor", "0")
	sizeStr := ctx.DefaultQuery("size", "20")

	cursor, err := strconv.Atoi(cursorStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, respond.RespErr(fmt.Errorf("cursor parameter must be a number"), t, 1))
		return
	}

	size, err := strconv.Atoi(sizeStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, respond.RespErr(fmt.Errorf("size parameter must be a number"), t, 1))
		return
	}

	result, err := service.GetGroupPersonListCollection(cursor, size)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, respond.RespErr(err, t, 1))
		return
	}

	ctx.JSON(http.StatusOK, respond.RespSuccess(result, t))
}

// GetLuckyBagErrorCollectionKeys Get keys from lucky bag error collections with pagination
// @Summary Get lucky bag error collection keys
// @Description Get paginated list of keys from TalkGroupOpenLuckyBagErrCollection or TalkGroupResidueLuckyBagErrCollection
// @Tags Database
// @Accept json
// @Produce json
// @Param collection query string true "Collection name (talk_group_open_lucky_bag_err or talk_group_residue_lucky_bag_err)"
// @Param cursor query int false "Cursor for pagination (default: 0)"
// @Param size query int false "Page size (default: 20)"
// @Success 200 {object} map[string]interface{} "Success response with keys list"
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /db/lucky-bag-error-keys [get]
func GetLuckyBagErrorCollectionKeys(ctx *gin.Context) {
	collection := ctx.Query("collection")
	if collection == "" {
		ctx.JSON(400, gin.H{
			"success": false,
			"error":   "collection parameter is required",
		})
		return
	}

	cursor := 0
	if cursorStr := ctx.Query("cursor"); cursorStr != "" {
		if c, err := strconv.Atoi(cursorStr); err == nil {
			cursor = c
		}
	}

	size := 20
	if sizeStr := ctx.Query("size"); sizeStr != "" {
		if s, err := strconv.Atoi(sizeStr); err == nil {
			size = s
		}
	}

	result, err := service.GetLuckyBagErrorCollectionKeys(collection, cursor, size)
	if err != nil {
		ctx.JSON(500, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	ctx.JSON(200, gin.H{
		"success": true,
		"data":    result,
	})
}

// GetLuckyBagPinByPinId Get lucky bag pin data by pinId from specified collection
// @Summary Get lucky bag pin data by pinId
// @Description Get lucky bag pin data from TalkGroupOpenLuckyBagPinCollection or TalkGroupResidueLuckyBagPinCollection by pinId
// @Tags Database
// @Accept json
// @Produce json
// @Param collection query string true "Collection name (talk_group_open_lucky_bag_pin or talk_group_residue_lucky_bag_pin)"
// @Param pinId query string true "PinId to search for"
// @Success 200 {object} map[string]interface{} "Success response with pin data"
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Failure 404 {object} map[string]interface{} "PinId not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /db/lucky-bag-pin [get]
func GetLuckyBagPinByPinId(ctx *gin.Context) {
	collection := ctx.Query("collection")
	if collection == "" {
		ctx.JSON(400, gin.H{
			"success": false,
			"error":   "collection parameter is required",
		})
		return
	}

	pinId := ctx.Query("pinId")
	if pinId == "" {
		ctx.JSON(400, gin.H{
			"success": false,
			"error":   "pinId parameter is required",
		})
		return
	}

	result, err := service.GetLuckyBagPinByPinId(collection, pinId)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			ctx.JSON(404, gin.H{
				"success": false,
				"error":   err.Error(),
			})
		} else {
			ctx.JSON(500, gin.H{
				"success": false,
				"error":   err.Error(),
			})
		}
		return
	}

	ctx.JSON(200, gin.H{
		"success": true,
		"data":    result,
	})
}

// GetLuckyBagCodeAddressKeyFromCompleted Get lucky bag code address key from completed collection
// @Summary Get lucky bag code address key from completed collection
// @Description Get lucky bag code address key from completed collection by code and address
// @Tags Database Query
// @Accept json
// @Produce json
// @Param code query string true "Lucky bag code"
// @Param address query string true "Lucky bag address"
// @Success 200 {object} map[string]interface{} "Successfully return code address key data"
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Failure 404 {object} map[string]interface{} "Code address key not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/db/luckybag/code-address-key [get]
func GetLuckyBagCodeAddressKeyFromCompleted(ctx *gin.Context) {
	// Get query parameters
	code := ctx.Query("code")
	address := ctx.Query("address")

	// Validate required parameters
	if code == "" {
		ctx.JSON(400, gin.H{
			"success": false,
			"error":   "code parameter is required",
		})
		return
	}

	if address == "" {
		ctx.JSON(400, gin.H{
			"success": false,
			"error":   "address parameter is required",
		})
		return
	}

	// Call service method
	result, err := service.GetLuckyBagCodeAddressKeyFromCompleted(code, address)
	if err != nil {
		// Check if it's a not found error
		if strings.Contains(err.Error(), "not found") {
			ctx.JSON(404, gin.H{
				"success": false,
				"error":   err.Error(),
			})
		} else {
			ctx.JSON(500, gin.H{
				"success": false,
				"error":   err.Error(),
			})
		}
		return
	}

	ctx.JSON(200, gin.H{
		"success": true,
		"data":    result,
	})
}

// RetryFailedLuckyBagOperation Retry failed lucky bag operation by pinId
// @Summary Retry failed lucky bag operation
// @Description Retry failed lucky bag operation by pinId from error collections
// @Tags Database Query
// @Accept json
// @Produce json
// @Param pinId query string true "PinId of the failed lucky bag operation"
// @Success 200 {object} map[string]interface{} "Successfully retried lucky bag operation"
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Failure 404 {object} map[string]interface{} "PinId not found in error collections"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/db/luckybag/retry [post]
func RetryFailedLuckyBagOperation(ctx *gin.Context) {
	// Get query parameter
	pinId := ctx.Query("pinId")

	// Validate required parameter
	if pinId == "" {
		ctx.JSON(400, gin.H{
			"success": false,
			"error":   "pinId parameter is required",
		})
		return
	}

	// Call service method
	result, err := service.RetryFailedLuckyBagOperation(pinId)
	if err != nil {
		// Check if it's a not found error
		if strings.Contains(err.Error(), "not found") {
			ctx.JSON(404, gin.H{
				"success": false,
				"error":   err.Error(),
			})
		} else {
			ctx.JSON(500, gin.H{
				"success": false,
				"error":   err.Error(),
			})
		}
		return
	}

	ctx.JSON(200, gin.H{
		"success": true,
		"data":    result,
	})
}

// RetryFailedLuckyBagOperationsByLuckyBagId Retry failed lucky bag operations by lucky bag ID
// @Summary Retry failed lucky bag operations by lucky bag ID
// @Description Retry failed lucky bag operations by lucky bag ID from error collections
// @Tags Database Query
// @Accept json
// @Produce json
// @Param luckyBagId query string true "Lucky bag ID to retry failed operations for"
// @Success 200 {object} map[string]interface{} "Successfully retried lucky bag operations"
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/db/luckybag/retry-by-luckybag-id [post]
func RetryFailedLuckyBagOperationsByLuckyBagId(ctx *gin.Context) {
	// Get query parameter
	luckyBagId := ctx.Query("luckyBagId")

	// Validate required parameter
	if luckyBagId == "" {
		ctx.JSON(400, gin.H{
			"success": false,
			"error":   "luckyBagId parameter is required",
		})
		return
	}

	// Call service method
	result, err := service.RetryFailedLuckyBagOperationsByLuckyBagId(luckyBagId)
	if err != nil {
		ctx.JSON(500, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	ctx.JSON(200, gin.H{
		"success": true,
		"data":    result,
	})
}

// GetGroupAdminCollection Get TalkGroupAdminCollection data with pagination
// @Summary Get group admin collection data
// @Description Get TalkGroupAdminCollection data with pagination support
// @Tags Database
// @Accept json
// @Produce json
// @Param cursor query int false "Cursor for pagination" default(0)
// @Param size query int false "Number of items per page" default(20)
// @Success 200 {object} map[string]interface{} "Success response with paginated data"
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/db/group-admin-collection [get]
func GetGroupAdminCollection(ctx *gin.Context) {
	cursor := 0
	size := 20

	if cursorStr := ctx.Query("cursor"); cursorStr != "" {
		if parsed, err := strconv.Atoi(cursorStr); err == nil {
			cursor = parsed
		}
	}

	if sizeStr := ctx.Query("size"); sizeStr != "" {
		if parsed, err := strconv.Atoi(sizeStr); err == nil {
			size = parsed
		}
	}

	result, err := service.QueryGroupAdminCollection(cursor, size)
	if err != nil {
		ctx.JSON(500, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	ctx.JSON(200, gin.H{
		"success": true,
		"data":    result,
	})
}

// GetGroupBlockCollection Get TalkGroupBlockCollection data with pagination
// @Summary Get group block collection data
// @Description Get TalkGroupBlockCollection data with pagination support
// @Tags Database
// @Accept json
// @Produce json
// @Param cursor query int false "Cursor for pagination" default(0)
// @Param size query int false "Number of items per page" default(20)
// @Success 200 {object} map[string]interface{} "Success response with paginated data"
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/db/group-block-collection [get]
func GetGroupBlockCollection(ctx *gin.Context) {
	cursor := 0
	size := 20

	if cursorStr := ctx.Query("cursor"); cursorStr != "" {
		if parsed, err := strconv.Atoi(cursorStr); err == nil {
			cursor = parsed
		}
	}

	if sizeStr := ctx.Query("size"); sizeStr != "" {
		if parsed, err := strconv.Atoi(sizeStr); err == nil {
			size = parsed
		}
	}

	result, err := service.QueryGroupBlockCollection(cursor, size)
	if err != nil {
		ctx.JSON(500, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	ctx.JSON(200, gin.H{
		"success": true,
		"data":    result,
	})
}

// GetGroupWhitelistCollection Get TalkGroupWhitelistCollection data with pagination
// @Summary Get group whitelist collection data
// @Description Get TalkGroupWhitelistCollection data with pagination support
// @Tags Database
// @Accept json
// @Produce json
// @Param cursor query int false "Cursor for pagination" default(0)
// @Param size query int false "Number of items per page" default(20)
// @Success 200 {object} map[string]interface{} "Success response with paginated data"
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/db/group-whitelist-collection [get]
func GetGroupWhitelistCollection(ctx *gin.Context) {
	cursor := 0
	size := 20

	if cursorStr := ctx.Query("cursor"); cursorStr != "" {
		if parsed, err := strconv.Atoi(cursorStr); err == nil {
			cursor = parsed
		}
	}

	if sizeStr := ctx.Query("size"); sizeStr != "" {
		if parsed, err := strconv.Atoi(sizeStr); err == nil {
			size = parsed
		}
	}

	result, err := service.QueryGroupWhitelistCollection(cursor, size)
	if err != nil {
		ctx.JSON(500, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	ctx.JSON(200, gin.H{
		"success": true,
		"data":    result,
	})
}

// GetGroupAdminByGroupId Get TalkGroupAdminCollection data by groupId
// @Summary Get group admin data by group ID
// @Description Get TalkGroupAdminCollection data for a specific group ID
// @Tags Database
// @Accept json
// @Produce json
// @Param groupId path string true "Group ID"
// @Success 200 {object} map[string]interface{} "Success response with group admin data"
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/db/group-admin/{groupId} [get]
func GetGroupAdminByGroupId(ctx *gin.Context) {
	groupId := ctx.Param("groupId")
	if groupId == "" {
		ctx.JSON(400, gin.H{
			"success": false,
			"error":   "groupId parameter is required",
		})
		return
	}

	result, err := service.QueryGroupAdminByGroupId(groupId)
	if err != nil {
		ctx.JSON(500, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	ctx.JSON(200, gin.H{
		"success": true,
		"data":    result,
	})
}

// GetGroupBlockByGroupId Get TalkGroupBlockCollection data by groupId
// @Summary Get group block data by group ID
// @Description Get TalkGroupBlockCollection data for a specific group ID
// @Tags Database
// @Accept json
// @Produce json
// @Param groupId path string true "Group ID"
// @Success 200 {object} map[string]interface{} "Success response with group block data"
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/db/group-block/{groupId} [get]
func GetGroupBlockByGroupId(ctx *gin.Context) {
	groupId := ctx.Param("groupId")
	if groupId == "" {
		ctx.JSON(400, gin.H{
			"success": false,
			"error":   "groupId parameter is required",
		})
		return
	}

	result, err := service.QueryGroupBlockByGroupId(groupId)
	if err != nil {
		ctx.JSON(500, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	ctx.JSON(200, gin.H{
		"success": true,
		"data":    result,
	})
}

// GetGroupWhitelistByGroupId Get TalkGroupWhitelistCollection data by groupId
// @Summary Get group whitelist data by group ID
// @Description Get TalkGroupWhitelistCollection data for a specific group ID
// @Tags Database
// @Accept json
// @Produce json
// @Param groupId path string true "Group ID"
// @Success 200 {object} map[string]interface{} "Success response with group whitelist data"
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/db/group-whitelist/{groupId} [get]
func GetGroupWhitelistByGroupId(ctx *gin.Context) {
	groupId := ctx.Param("groupId")
	if groupId == "" {
		ctx.JSON(400, gin.H{
			"success": false,
			"error":   "groupId parameter is required",
		})
		return
	}

	result, err := service.QueryGroupWhitelistByGroupId(groupId)
	if err != nil {
		ctx.JSON(500, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	ctx.JSON(200, gin.H{
		"success": true,
		"data":    result,
	})
}

// SetGlobalBlockAddress Set a global block address
// @Summary Set global block address
// @Description Set an address to the global block list
// @Tags Global Block
// @Accept json
// @Produce json
// @Param request body map[string]interface{} true "Request body with address and reason"
// @Success 200 {object} map[string]interface{} "Success response"
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/db/global-block/set [post]
func SetGlobalBlockAddress(ctx *gin.Context) {
	var req struct {
		Address string `json:"address" binding:"required"`
		Reason  string `json:"reason"`
	}

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(400, gin.H{
			"success": false,
			"error":   fmt.Sprintf("Invalid request parameters: %v", err),
		})
		return
	}

	result, err := service.SetGlobalBlockAddress(req.Address, req.Reason)
	if err != nil {
		ctx.JSON(500, gin.H{
			"success": false,
			"error":   fmt.Sprintf("Failed to set global block address: %v", err),
		})
		return
	}

	ctx.JSON(200, gin.H{
		"success": true,
		"data":    result,
	})
}

// DeleteGlobalBlockAddress Delete a global block address
// @Summary Delete global block address
// @Description Remove an address from the global block list
// @Tags Global Block
// @Accept json
// @Produce json
// @Param request body map[string]interface{} true "Request body with address"
// @Success 200 {object} map[string]interface{} "Success response"
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/db/global-block/delete [post]
func DeleteGlobalBlockAddress(ctx *gin.Context) {
	var req struct {
		Address string `json:"address" binding:"required"`
	}

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(400, gin.H{
			"success": false,
			"error":   fmt.Sprintf("Invalid request parameters: %v", err),
		})
		return
	}

	result, err := service.DeleteGlobalBlockAddress(req.Address)
	if err != nil {
		ctx.JSON(500, gin.H{
			"success": false,
			"error":   fmt.Sprintf("Failed to delete global block address: %v", err),
		})
		return
	}

	ctx.JSON(200, gin.H{
		"success": true,
		"data":    result,
	})
}

// GetGlobalBlockStats Get global block list statistics
// @Summary Get global block statistics
// @Description Get statistics about the global block list
// @Tags Global Block
// @Produce json
// @Success 200 {object} map[string]interface{} "Success response with statistics"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/db/global-block/stats [get]
func GetGlobalBlockStats(ctx *gin.Context) {
	result, err := service.GetGlobalBlockStats()
	if err != nil {
		ctx.JSON(500, gin.H{
			"success": false,
			"error":   fmt.Sprintf("Failed to get global block stats: %v", err),
		})
		return
	}

	ctx.JSON(200, gin.H{
		"success": true,
		"data":    result,
	})
}

// GetGlobalBlockAddresses Get all global block addresses with pagination
// @Summary Get global block addresses
// @Description Get paginated list of all global block addresses
// @Tags Global Block
// @Produce json
// @Param cursor query int false "Cursor for pagination" default(0)
// @Param size query int false "Page size" default(20)
// @Success 200 {object} map[string]interface{} "Success response with paginated data"
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/db/global-block/addresses [get]
func GetGlobalBlockAddresses(ctx *gin.Context) {
	cursorStr := ctx.DefaultQuery("cursor", "0")
	sizeStr := ctx.DefaultQuery("size", "20")

	cursor, err := strconv.Atoi(cursorStr)
	if err != nil {
		ctx.JSON(400, gin.H{
			"success": false,
			"error":   "Invalid cursor parameter",
		})
		return
	}

	size, err := strconv.Atoi(sizeStr)
	if err != nil {
		ctx.JSON(400, gin.H{
			"success": false,
			"error":   "Invalid size parameter",
		})
		return
	}

	result, err := service.GetGlobalBlockAddresses(cursor, size)
	if err != nil {
		ctx.JSON(500, gin.H{
			"success": false,
			"error":   fmt.Sprintf("Failed to get global block addresses: %v", err),
		})
		return
	}

	ctx.JSON(200, gin.H{
		"success": true,
		"data":    result,
	})
}

// CheckGlobalBlockAddress Check if an address is globally blocked
// @Summary Check global block status
// @Description Check if an address is in the global block list
// @Tags Global Block
// @Produce json
// @Param address query string true "Address to check"
// @Success 200 {object} map[string]interface{} "Success response with block status"
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/db/global-block/check [get]
func CheckGlobalBlockAddress(ctx *gin.Context) {
	address := ctx.Query("address")
	if address == "" {
		ctx.JSON(400, gin.H{
			"success": false,
			"error":   "Address parameter is required",
		})
		return
	}

	result, err := service.CheckGlobalBlockAddress(address)
	if err != nil {
		ctx.JSON(500, gin.H{
			"success": false,
			"error":   fmt.Sprintf("Failed to check global block address: %v", err),
		})
		return
	}

	ctx.JSON(200, gin.H{
		"success": true,
		"data":    result,
	})
}

// SetGlobalLuckBagBlockAddress Set a global luck bag block address
// @Summary Set global luck bag block address
// @Description Add an address to the global luck bag block list
// @Tags Global Luck Bag Block
// @Accept json
// @Produce json
// @Param request body map[string]string true "Request body with address and reason"
// @Success 200 {object} map[string]interface{} "Success response"
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/db/global-luck-bag-block/set [post]
func SetGlobalLuckBagBlockAddress(ctx *gin.Context) {
	var request map[string]string
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(400, gin.H{
			"success": false,
			"error":   "Invalid request body",
		})
		return
	}

	address, exists := request["address"]
	if !exists || address == "" {
		ctx.JSON(400, gin.H{
			"success": false,
			"error":   "Address is required",
		})
		return
	}

	reason := request["reason"]
	if reason == "" {
		reason = "No reason provided"
	}

	result, err := service.SetGlobalLuckBagBlockAddress(address, reason)
	if err != nil {
		ctx.JSON(500, gin.H{
			"success": false,
			"error":   fmt.Sprintf("Failed to set global luck bag block address: %v", err),
		})
		return
	}

	ctx.JSON(200, gin.H{
		"success": true,
		"data":    result,
	})
}

// DeleteGlobalLuckBagBlockAddress Delete a global luck bag block address
// @Summary Delete global luck bag block address
// @Description Remove an address from the global luck bag block list
// @Tags Global Luck Bag Block
// @Accept json
// @Produce json
// @Param request body map[string]string true "Request body with address"
// @Success 200 {object} map[string]interface{} "Success response"
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/db/global-luck-bag-block/delete [post]
func DeleteGlobalLuckBagBlockAddress(ctx *gin.Context) {
	var request map[string]string
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(400, gin.H{
			"success": false,
			"error":   "Invalid request body",
		})
		return
	}

	address, exists := request["address"]
	if !exists || address == "" {
		ctx.JSON(400, gin.H{
			"success": false,
			"error":   "Address is required",
		})
		return
	}

	result, err := service.DeleteGlobalLuckBagBlockAddress(address)
	if err != nil {
		ctx.JSON(500, gin.H{
			"success": false,
			"error":   fmt.Sprintf("Failed to delete global luck bag block address: %v", err),
		})
		return
	}

	ctx.JSON(200, gin.H{
		"success": true,
		"data":    result,
	})
}

// GetGlobalLuckBagBlockStats Get global luck bag block list statistics
// @Summary Get global luck bag block statistics
// @Description Get statistics about the global luck bag block list
// @Tags Global Luck Bag Block
// @Produce json
// @Success 200 {object} map[string]interface{} "Success response with statistics"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/db/global-luck-bag-block/stats [get]
func GetGlobalLuckBagBlockStats(ctx *gin.Context) {
	result, err := service.GetGlobalLuckBagBlockStats()
	if err != nil {
		ctx.JSON(500, gin.H{
			"success": false,
			"error":   fmt.Sprintf("Failed to get global luck bag block stats: %v", err),
		})
		return
	}

	ctx.JSON(200, gin.H{
		"success": true,
		"data":    result,
	})
}

// GetGlobalLuckBagBlockAddresses Get all global luck bag block addresses with pagination
// @Summary Get global luck bag block addresses
// @Description Get all addresses in the global luck bag block list with pagination
// @Tags Global Luck Bag Block
// @Produce json
// @Param cursor query int false "Cursor for pagination (default: 0)"
// @Param size query int false "Number of items per page (default: 20)"
// @Success 200 {object} map[string]interface{} "Success response with paginated data"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/db/global-luck-bag-block/addresses [get]
func GetGlobalLuckBagBlockAddresses(ctx *gin.Context) {
	cursorStr := ctx.Query("cursor")
	sizeStr := ctx.Query("size")

	cursor := 0
	size := 20

	if cursorStr != "" {
		if c, err := strconv.Atoi(cursorStr); err == nil {
			cursor = c
		}
	}

	if sizeStr != "" {
		if s, err := strconv.Atoi(sizeStr); err == nil {
			size = s
		}
	}

	result, err := service.GetGlobalLuckBagBlockAddresses(cursor, size)
	if err != nil {
		ctx.JSON(500, gin.H{
			"success": false,
			"error":   fmt.Sprintf("Failed to get global luck bag block addresses: %v", err),
		})
		return
	}

	ctx.JSON(200, gin.H{
		"success": true,
		"data":    result,
	})
}

// CheckGlobalLuckBagBlockAddress Check if an address is globally luck bag blocked
// @Summary Check global luck bag block status
// @Description Check if an address is in the global luck bag block list
// @Tags Global Luck Bag Block
// @Produce json
// @Param address query string true "Address to check"
// @Success 200 {object} map[string]interface{} "Success response with block status"
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/db/global-luck-bag-block/check [get]
func CheckGlobalLuckBagBlockAddress(ctx *gin.Context) {
	address := ctx.Query("address")
	if address == "" {
		ctx.JSON(400, gin.H{
			"success": false,
			"error":   "Address parameter is required",
		})
		return
	}

	result, err := service.CheckGlobalLuckBagBlockAddress(address)
	if err != nil {
		ctx.JSON(500, gin.H{
			"success": false,
			"error":   fmt.Sprintf("Failed to check global luck bag block address: %v", err),
		})
		return
	}

	ctx.JSON(200, gin.H{
		"success": true,
		"data":    result,
	})
}

// GetChatStatistics Get chat statistics within a time range
// @Summary Get chat statistics
// @Description Get comprehensive chat statistics including group chat, private chat, channel chat, group creation, and total counts within a time range
// @Tags Statistics
// @Produce json
// @Param startTime query int64 true "Start timestamp (Unix timestamp)"
// @Param endTime query int64 true "End timestamp (Unix timestamp)"
// @Param groupId query string false "Group ID (leave empty to get statistics for all groups)"
// @Success 200 {object} map[string]interface{} "Success response with chat statistics"
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/db/chat-statistics [get]
func GetChatStatistics(ctx *gin.Context) {
	startTimeStr := ctx.Query("startTime")
	endTimeStr := ctx.Query("endTime")
	groupId := ctx.Query("groupId") // Optional groupId parameter

	if startTimeStr == "" || endTimeStr == "" {
		ctx.JSON(400, gin.H{
			"success": false,
			"error":   "startTime and endTime parameters are required",
		})
		return
	}

	startTime, err := strconv.ParseInt(startTimeStr, 10, 64)
	if err != nil {
		ctx.JSON(400, gin.H{
			"success": false,
			"error":   "Invalid startTime parameter",
		})
		return
	}

	endTime, err := strconv.ParseInt(endTimeStr, 10, 64)
	if err != nil {
		ctx.JSON(400, gin.H{
			"success": false,
			"error":   "Invalid endTime parameter",
		})
		return
	}

	if startTime > endTime {
		ctx.JSON(400, gin.H{
			"success": false,
			"error":   "startTime must be less than or equal to endTime",
		})
		return
	}

	result, err := service.GetChatStatisticsByTimeRange(startTime, endTime, groupId)
	if err != nil {
		ctx.JSON(500, gin.H{
			"success": false,
			"error":   fmt.Sprintf("Failed to get chat statistics: %v", err),
		})
		return
	}

	ctx.JSON(200, gin.H{
		"success": true,
		"data":    result,
	})
}

// GetDbSyncStats Get sync statistics from database service
// @Summary Get sync statistics from database service
// @Description Get synchronization statistics including sync status, block heights, and progress
// @Tags Database
// @Produce json
// @Success 200 {object} map[string]interface{} "Success response with sync statistics"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/db/sync-stats [get]
func GetDbSyncStats(ctx *gin.Context) {
	result, err := service.GetSyncStats()
	if err != nil {
		ctx.JSON(500, gin.H{
			"success": false,
			"error":   fmt.Sprintf("Failed to get sync statistics: %v", err),
		})
		return
	}

	ctx.JSON(200, gin.H{
		"success": true,
		"data":    result,
	})
}

// GetPinSyncStatus Get pin sync status by pinId
// @Summary Get pin sync status by pinId
// @Description Get synchronization status for a specific pin ID
// @Tags Database
// @Produce json
// @Param pinId query string true "Pin ID to check sync status"
// @Success 200 {object} map[string]interface{} "Success response with pin sync status"
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/db/pin-sync/status [get]
func GetPinSyncStatus(ctx *gin.Context) {
	pinId := ctx.Query("pinId")
	if pinId == "" {
		ctx.JSON(400, gin.H{
			"success": false,
			"error":   "pinId parameter is required",
		})
		return
	}

	result, err := service.GetPinSyncStatus(pinId)
	if err != nil {
		ctx.JSON(500, gin.H{
			"success": false,
			"error":   fmt.Sprintf("Failed to get pin sync status: %v", err),
		})
		return
	}

	ctx.JSON(200, gin.H{
		"success": true,
		"data":    result,
	})
}

// GetAllSyncedPins Get all synced pins with pagination
// @Summary Get all synced pins with pagination
// @Description Get paginated list of all synced pins
// @Tags Database
// @Produce json
// @Param cursor query int false "Cursor for pagination (default: 0)"
// @Param size query int false "Number of items to return (default: 20, max: 100)"
// @Success 200 {object} map[string]interface{} "Success response with synced pins list"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/db/pin-sync/synced-pins [get]
func GetAllSyncedPins(ctx *gin.Context) {
	cursor := 0
	size := 20

	if cursorStr := ctx.Query("cursor"); cursorStr != "" {
		if c, err := strconv.Atoi(cursorStr); err == nil && c >= 0 {
			cursor = c
		}
	}

	if sizeStr := ctx.Query("size"); sizeStr != "" {
		if s, err := strconv.Atoi(sizeStr); err == nil && s > 0 && s <= 100 {
			size = s
		}
	}

	result, err := service.GetAllSyncedPins(cursor, size)
	if err != nil {
		ctx.JSON(500, gin.H{
			"success": false,
			"error":   fmt.Sprintf("Failed to get synced pins: %v", err),
		})
		return
	}

	ctx.JSON(200, gin.H{
		"success": true,
		"data":    result,
	})
}

// @Summary Get pins count by time range
// @Description Get pins count within a specified time range
// @Tags Database Operations
// @Accept json
// @Produce json
// @Param startTime query int64 true "Start timestamp (seconds)"
// @Param endTime query int64 true "End timestamp (seconds)"
// @Success 200 {object} map[string]interface{} "Success response with pins count"
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/db/pin-sync/pins-count-by-time-range [get]
func GetPinsCountByTimeRange(ctx *gin.Context) {
	var t = time.Now().UnixMilli()
	startTimeStr := ctx.Query("startTime")
	endTimeStr := ctx.Query("endTime")

	if startTimeStr == "" || endTimeStr == "" {
		ctx.JSON(http.StatusBadRequest, respond.RespErr(fmt.Errorf("startTime and endTime parameters are required"), t, 1))
		return
	}

	startTime, err := strconv.ParseInt(startTimeStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, respond.RespErr(fmt.Errorf("startTime parameter must be a valid timestamp"), t, 1))
		return
	}

	endTime, err := strconv.ParseInt(endTimeStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, respond.RespErr(fmt.Errorf("endTime parameter must be a valid timestamp"), t, 1))
		return
	}

	result, err := service.GetPinsCountByTimeRange(startTime, endTime)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, respond.RespErr(err, t, 1))
		return
	}

	ctx.JSON(http.StatusOK, respond.RespSuccess(result, t))
}

// @Summary Check pin existence by chain and block height
// @Description Check if a pin exists by chain name, block height and pinId
// @Tags Database Operations
// @Accept json
// @Produce json
// @Param chainName query string true "Chain name (e.g., btc, mvc)"
// @Param blockHeight query int64 true "Block height"
// @Param pinId query string true "Pin ID"
// @Success 200 {object} map[string]interface{} "Success response with pin existence status"
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/db/pin-sync/check-pin-exists [get]
func CheckPinExistsByChainAndHeight(ctx *gin.Context) {
	var t = time.Now().UnixMilli()
	chainName := ctx.Query("chainName")
	blockHeightStr := ctx.Query("blockHeight")
	pinId := ctx.Query("pinId")

	if chainName == "" {
		ctx.JSON(http.StatusBadRequest, respond.RespErr(fmt.Errorf("chainName parameter is required"), t, 1))
		return
	}

	if blockHeightStr == "" {
		ctx.JSON(http.StatusBadRequest, respond.RespErr(fmt.Errorf("blockHeight parameter is required"), t, 1))
		return
	}

	if pinId == "" {
		ctx.JSON(http.StatusBadRequest, respond.RespErr(fmt.Errorf("pinId parameter is required"), t, 1))
		return
	}

	blockHeight, err := strconv.ParseInt(blockHeightStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, respond.RespErr(fmt.Errorf("blockHeight parameter must be a valid integer"), t, 1))
		return
	}

	result, err := service.CheckPinExistsByChainAndHeight(chainName, blockHeight, pinId)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, respond.RespErr(err, t, 1))
		return
	}

	ctx.JSON(http.StatusOK, respond.RespSuccess(result, t))
}

// @Summary Get pin IDs by chain and block height
// @Description Get all pin IDs for a specific chain and block height
// @Tags Database Operations
// @Accept json
// @Produce json
// @Param chainName query string true "Chain name (e.g., btc, mvc)"
// @Param blockHeight query int64 true "Block height"
// @Success 200 {object} map[string]interface{} "Success response with pin IDs list"
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/db/pin-sync/get-pin-ids-by-height [get]
func GetPinIdsByChainAndHeight(ctx *gin.Context) {
	var t = time.Now().UnixMilli()
	chainName := ctx.Query("chainName")
	blockHeightStr := ctx.Query("blockHeight")

	if chainName == "" {
		ctx.JSON(http.StatusBadRequest, respond.RespErr(fmt.Errorf("chainName parameter is required"), t, 1))
		return
	}

	if blockHeightStr == "" {
		ctx.JSON(http.StatusBadRequest, respond.RespErr(fmt.Errorf("blockHeight parameter is required"), t, 1))
		return
	}

	blockHeight, err := strconv.ParseInt(blockHeightStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, respond.RespErr(fmt.Errorf("blockHeight parameter must be a valid integer"), t, 1))
		return
	}

	result, err := service.GetPinIdsByChainAndHeight(chainName, blockHeight)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, respond.RespErr(err, t, 1))
		return
	}

	ctx.JSON(http.StatusOK, respond.RespSuccess(result, t))
}
