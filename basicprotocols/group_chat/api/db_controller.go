package api

import (
	"fmt"
	"manindexer/basicprotocols/group_chat/api/respond"
	"manindexer/basicprotocols/group_chat/service"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type DbController struct {
	dbService *service.DbService
}

func NewDbController() *DbController {
	return &DbController{
		dbService: &service.DbService{},
	}
}

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
func (c *DbController) GetCommunityVersionInfo(ctx *gin.Context) {
	var t = time.Now().Unix()
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
		results, queryErr = c.dbService.QueryByPrefix("talk_community_version_info", prefix, limit)
	} else if pinId != "" {
		prefix := pinId + "_"
		results, queryErr = c.dbService.QueryByPrefix("talk_community_version_info", prefix, limit)
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
func (c *DbController) GetCommunityInfo(ctx *gin.Context) {
	var t = time.Now().Unix()
	communityId := ctx.Query("communityId")
	if communityId == "" {
		ctx.JSON(http.StatusBadRequest, respond.RespErr(fmt.Errorf("communityId parameter cannot be empty"), t, 1))
		return
	}

	result, err := c.dbService.QueryCommunityInfo(communityId)
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
func (c *DbController) GetCommunityJoin(ctx *gin.Context) {
	var t = time.Now().Unix()
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
		results, queryErr = c.dbService.QueryCommunityJoin(communityId, limit)
	} else if pinId != "" {
		prefix := pinId + "_"
		results, queryErr = c.dbService.QueryByPrefix("talk_community_join", prefix, limit)
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
func (c *DbController) GetCommunityPerson(ctx *gin.Context) {
	var t = time.Now().Unix()
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
		results, queryErr = c.dbService.QueryCommunityPerson(communityId, limit)
	} else if metaId != "" {
		prefix := metaId + "_"
		results, queryErr = c.dbService.QueryByPrefix("talk_community_person", prefix, limit)
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
func (c *DbController) GetGroupInfo(ctx *gin.Context) {
	var t = time.Now().Unix()
	groupId := ctx.Query("groupId")
	if groupId == "" {
		ctx.JSON(http.StatusBadRequest, respond.RespErr(fmt.Errorf("groupId parameter cannot be empty"), t, 1))
		return
	}

	result, err := c.dbService.QueryGroupInfo(groupId)
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
func (c *DbController) GetGroupVersionInfo(ctx *gin.Context) {
	var t = time.Now().Unix()
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
		results, queryErr = c.dbService.QueryGroupVersionInfo(groupId, limit)
	} else if pinId != "" {
		prefix := pinId + "_"
		results, queryErr = c.dbService.QueryByPrefix("talk_group_version_info", prefix, limit)
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
func (c *DbController) GetGroupJoin(ctx *gin.Context) {
	var t = time.Now().Unix()
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
		results, queryErr = c.dbService.QueryGroupJoin(groupId, limit)
	} else if pinId != "" {
		prefix := pinId + "_"
		results, queryErr = c.dbService.QueryByPrefix("talk_group_join", prefix, limit)
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
func (c *DbController) GetGroupPerson(ctx *gin.Context) {
	var t = time.Now().Unix()
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
		results, queryErr = c.dbService.QueryGroupPerson(groupId, limit)
	} else if metaId != "" {
		prefix := metaId + "_"
		results, queryErr = c.dbService.QueryByPrefix("talk_group_person", prefix, limit)
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
func (c *DbController) GetChatQueue(ctx *gin.Context) {
	var t = time.Now().Unix()
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
		results, queryErr = c.dbService.QueryByPrefix("talk_group_chat_queue", prefix, limit)
	} else {
		results, queryErr = c.dbService.QueryGroupChatQueue(limit)
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
func (c *DbController) GetChatPin(ctx *gin.Context) {
	var t = time.Now().Unix()
	pinId := ctx.Query("pinId")
	if pinId == "" {
		ctx.JSON(http.StatusBadRequest, respond.RespErr(fmt.Errorf("pinId parameter cannot be empty"), t, 1))
		return
	}

	result, err := c.dbService.QueryGroupChatPin(pinId)
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
func (c *DbController) GetChatTimestamp(ctx *gin.Context) {
	var t = time.Now().Unix()
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

	results, err := c.dbService.QueryGroupChatByTimestamp(groupId, 0, 0, limit)
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
func (c *DbController) GetUserContext(ctx *gin.Context) {
	var t = time.Now().Unix()
	metaId := ctx.Query("metaId")
	if metaId == "" {
		ctx.JSON(http.StatusBadRequest, respond.RespErr(fmt.Errorf("metaId parameter cannot be empty"), t, 1))
		return
	}

	result, err := c.dbService.QueryMetaIdContextList(metaId)
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
func (c *DbController) GetDatabaseStats(ctx *gin.Context) {
	var t = time.Now().Unix()
	stats, err := c.dbService.GetDatabaseStats()
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
func (c *DbController) GetCollections(ctx *gin.Context) {
	var t = time.Now().Unix()
	collections := c.dbService.GetAvailableCollections()

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
func (c *DbController) GetAllGroupVersionInfo(ctx *gin.Context) {
	var t = time.Now().Unix()
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
	totalCount, err := c.dbService.GetCollectionCount("talk_group_version_info")
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, respond.RespErr(err, t, 1))
		return
	}

	// Get paginated data
	results, err := c.dbService.QueryAll("talk_group_version_info", skip+size)
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
func (c *DbController) GetAllChatPin(ctx *gin.Context) {
	var t = time.Now().Unix()
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
	totalCount, err := c.dbService.GetCollectionCount("talk_group_chat_pin")
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, respond.RespErr(err, t, 1))
		return
	}

	// Get paginated data
	results, err := c.dbService.QueryAll("talk_group_chat_pin", skip+size)
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
