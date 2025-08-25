package api

import (
	"fmt"
	"manindexer/basicprotocols/group_chat/api/respond"
	"manindexer/basicprotocols/group_chat/db"
	"manindexer/basicprotocols/group_chat/service"
	"net/http"
	"strconv"
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

	if timestamp != "" {
		prefix := timestamp + "_"
		results, queryErr = service.QueryByPrefix("talk_group_chat_queue", prefix, limit)
	} else {
		results, queryErr = service.QueryGroupChatQueue(limit)
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
