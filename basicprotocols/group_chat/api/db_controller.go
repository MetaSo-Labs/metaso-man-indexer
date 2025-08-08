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

// @Summary 根据communityId或pinId获取社区版本信息
// @Description 根据communityId或pinId查询TalkCommunityVersionInfoCollection数据
// @Tags 数据库查询
// @Accept json
// @Produce json
// @Param communityId query string false "社区ID"
// @Param pinId query string false "PinID"
// @Param limit query int false "限制数量" default(10)
// @Success 200 {object} map[string]interface{} "查询结果"
// @Failure 400 {object} map[string]interface{} "参数错误"
// @Failure 500 {object} map[string]interface{} "服务器错误"
// @Router /api/db/community/version [get]
func (c *DbController) GetCommunityVersionInfo(ctx *gin.Context) {
	var t = time.Now().Unix()
	communityId := ctx.Query("communityId")
	pinId := ctx.Query("pinId")
	limitStr := ctx.DefaultQuery("limit", "10")

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, respond.RespErr(fmt.Errorf("limit参数必须是数字"), t, 1))
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
		ctx.JSON(http.StatusBadRequest, respond.RespErr(fmt.Errorf("必须提供communityId或pinId参数"), t, 1))
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

// @Summary 根据communityId获取社区信息
// @Description 根据communityId查询TalkCommunityInfoCollection数据
// @Tags 数据库查询
// @Accept json
// @Produce json
// @Param communityId query string true "社区ID"
// @Success 200 {object} map[string]interface{} "查询结果"
// @Failure 400 {object} map[string]interface{} "参数错误"
// @Failure 500 {object} map[string]interface{} "服务器错误"
// @Router /api/db/community/info [get]
func (c *DbController) GetCommunityInfo(ctx *gin.Context) {
	var t = time.Now().Unix()
	communityId := ctx.Query("communityId")
	if communityId == "" {
		ctx.JSON(http.StatusBadRequest, respond.RespErr(fmt.Errorf("communityId参数不能为空"), t, 1))
		return
	}

	result, err := c.dbService.QueryCommunityInfo(communityId)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, respond.RespErr(err, t, 1))
		return
	}

	ctx.JSON(http.StatusOK, respond.RespSuccess(result, t))
}

// @Summary 根据communityId或pinId获取社区加入记录
// @Description 根据communityId或pinId查询TalkCommunityJoinCollection数据
// @Tags 数据库查询
// @Accept json
// @Produce json
// @Param communityId query string false "社区ID"
// @Param pinId query string false "PinID"
// @Param limit query int false "限制数量" default(10)
// @Success 200 {object} map[string]interface{} "查询结果"
// @Failure 400 {object} map[string]interface{} "参数错误"
// @Failure 500 {object} map[string]interface{} "服务器错误"
// @Router /api/db/community/join [get]
func (c *DbController) GetCommunityJoin(ctx *gin.Context) {
	var t = time.Now().Unix()
	communityId := ctx.Query("communityId")
	pinId := ctx.Query("pinId")
	limitStr := ctx.DefaultQuery("limit", "10")

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, respond.RespErr(fmt.Errorf("limit参数必须是数字"), t, 1))
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
		ctx.JSON(http.StatusBadRequest, respond.RespErr(fmt.Errorf("必须提供communityId或pinId参数"), t, 1))
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

// @Summary 根据communityId或metaId获取社区成员列表
// @Description 根据communityId或metaId查询TalkCommunityPersonCollection数据
// @Tags 数据库查询
// @Accept json
// @Produce json
// @Param communityId query string false "社区ID"
// @Param metaId query string false "MetaID"
// @Param limit query int false "限制数量" default(10)
// @Success 200 {object} map[string]interface{} "查询结果"
// @Failure 400 {object} map[string]interface{} "参数错误"
// @Failure 500 {object} map[string]interface{} "服务器错误"
// @Router /api/db/community/person [get]
func (c *DbController) GetCommunityPerson(ctx *gin.Context) {
	var t = time.Now().Unix()
	communityId := ctx.Query("communityId")
	metaId := ctx.Query("metaId")
	limitStr := ctx.DefaultQuery("limit", "10")

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, respond.RespErr(fmt.Errorf("limit参数必须是数字"), t, 1))
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
		ctx.JSON(http.StatusBadRequest, respond.RespErr(fmt.Errorf("必须提供communityId或metaId参数"), t, 1))
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

// @Summary 根据groupId获取群组信息
// @Description 根据groupId查询TalkGroupInfoCollection数据
// @Tags 数据库查询
// @Accept json
// @Produce json
// @Param groupId query string true "群组ID"
// @Success 200 {object} map[string]interface{} "查询结果"
// @Failure 400 {object} map[string]interface{} "参数错误"
// @Failure 500 {object} map[string]interface{} "服务器错误"
// @Router /api/db/group/info [get]
func (c *DbController) GetGroupInfo(ctx *gin.Context) {
	var t = time.Now().Unix()
	groupId := ctx.Query("groupId")
	if groupId == "" {
		ctx.JSON(http.StatusBadRequest, respond.RespErr(fmt.Errorf("groupId参数不能为空"), t, 1))
		return
	}

	result, err := c.dbService.QueryGroupInfo(groupId)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, respond.RespErr(err, t, 1))
		return
	}

	ctx.JSON(http.StatusOK, respond.RespSuccess(result, t))
}

// @Summary 根据groupId或pinId获取群组版本信息
// @Description 根据groupId或pinId查询TalkGroupVersionInfoCollection数据
// @Tags 数据库查询
// @Accept json
// @Produce json
// @Param groupId query string false "群组ID"
// @Param pinId query string false "PinID"
// @Param limit query int false "限制数量" default(10)
// @Success 200 {object} map[string]interface{} "查询结果"
// @Failure 400 {object} map[string]interface{} "参数错误"
// @Failure 500 {object} map[string]interface{} "服务器错误"
// @Router /api/db/group/version [get]
func (c *DbController) GetGroupVersionInfo(ctx *gin.Context) {
	var t = time.Now().Unix()
	groupId := ctx.Query("groupId")
	pinId := ctx.Query("pinId")
	limitStr := ctx.DefaultQuery("limit", "10")

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, respond.RespErr(fmt.Errorf("limit参数必须是数字"), t, 1))
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
		ctx.JSON(http.StatusBadRequest, respond.RespErr(fmt.Errorf("必须提供groupId或pinId参数"), t, 1))
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

// @Summary 根据groupId或pinId获取群组加入记录
// @Description 根据groupId或pinId查询TalkGroupJoinCollection数据
// @Tags 数据库查询
// @Accept json
// @Produce json
// @Param groupId query string false "群组ID"
// @Param pinId query string false "PinID"
// @Param limit query int false "限制数量" default(10)
// @Success 200 {object} map[string]interface{} "查询结果"
// @Failure 400 {object} map[string]interface{} "参数错误"
// @Failure 500 {object} map[string]interface{} "服务器错误"
// @Router /api/db/group/join [get]
func (c *DbController) GetGroupJoin(ctx *gin.Context) {
	var t = time.Now().Unix()
	groupId := ctx.Query("groupId")
	pinId := ctx.Query("pinId")
	limitStr := ctx.DefaultQuery("limit", "10")

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, respond.RespErr(fmt.Errorf("limit参数必须是数字"), t, 1))
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
		ctx.JSON(http.StatusBadRequest, respond.RespErr(fmt.Errorf("必须提供groupId或pinId参数"), t, 1))
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

// @Summary 根据groupId或metaId获取群组成员列表
// @Description 根据groupId或metaId查询TalkGroupPersonCollection数据
// @Tags 数据库查询
// @Accept json
// @Produce json
// @Param groupId query string false "群组ID"
// @Param metaId query string false "MetaID"
// @Param limit query int false "限制数量" default(10)
// @Success 200 {object} map[string]interface{} "查询结果"
// @Failure 400 {object} map[string]interface{} "参数错误"
// @Failure 500 {object} map[string]interface{} "服务器错误"
// @Router /api/db/group/person [get]
func (c *DbController) GetGroupPerson(ctx *gin.Context) {
	var t = time.Now().Unix()
	groupId := ctx.Query("groupId")
	metaId := ctx.Query("metaId")
	limitStr := ctx.DefaultQuery("limit", "10")

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, respond.RespErr(fmt.Errorf("limit参数必须是数字"), t, 1))
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
		ctx.JSON(http.StatusBadRequest, respond.RespErr(fmt.Errorf("必须提供groupId或metaId参数"), t, 1))
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

// @Summary 根据timestamp获取聊天队列列表
// @Description 根据timestamp查询TalkGroupChatQueueCollection数据，或不传timestamp获取所有数据
// @Tags 数据库查询
// @Accept json
// @Produce json
// @Param timestamp query string false "时间戳"
// @Param limit query int false "限制数量" default(10)
// @Success 200 {object} map[string]interface{} "查询结果"
// @Failure 400 {object} map[string]interface{} "参数错误"
// @Failure 500 {object} map[string]interface{} "服务器错误"
// @Router /api/db/chat/queue [get]
func (c *DbController) GetChatQueue(ctx *gin.Context) {
	var t = time.Now().Unix()
	timestamp := ctx.Query("timestamp")
	limitStr := ctx.DefaultQuery("limit", "10")

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, respond.RespErr(fmt.Errorf("limit参数必须是数字"), t, 1))
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

// @Summary 根据pinId获取聊天消息
// @Description 根据pinId查询TalkGroupChatPinCollection数据
// @Tags 数据库查询
// @Accept json
// @Produce json
// @Param pinId query string true "PinID"
// @Success 200 {object} map[string]interface{} "查询结果"
// @Failure 400 {object} map[string]interface{} "参数错误"
// @Failure 500 {object} map[string]interface{} "服务器错误"
// @Router /api/db/chat/pin [get]
func (c *DbController) GetChatPin(ctx *gin.Context) {
	var t = time.Now().Unix()
	pinId := ctx.Query("pinId")
	if pinId == "" {
		ctx.JSON(http.StatusBadRequest, respond.RespErr(fmt.Errorf("pinId参数不能为空"), t, 1))
		return
	}

	result, err := c.dbService.QueryGroupChatPin(pinId)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, respond.RespErr(err, t, 1))
		return
	}

	ctx.JSON(http.StatusOK, respond.RespSuccess(result, t))
}

// @Summary 根据groupId获取聊天时间戳列表
// @Description 根据groupId查询TalkGroupChatTimestampCollection数据
// @Tags 数据库查询
// @Accept json
// @Produce json
// @Param groupId query string true "群组ID"
// @Param limit query int false "限制数量" default(10)
// @Success 200 {object} map[string]interface{} "查询结果"
// @Failure 400 {object} map[string]interface{} "参数错误"
// @Failure 500 {object} map[string]interface{} "服务器错误"
// @Router /api/db/chat/timestamp [get]
func (c *DbController) GetChatTimestamp(ctx *gin.Context) {
	var t = time.Now().Unix()
	groupId := ctx.Query("groupId")
	if groupId == "" {
		ctx.JSON(http.StatusBadRequest, respond.RespErr(fmt.Errorf("groupId参数不能为空"), t, 1))
		return
	}

	limitStr := ctx.DefaultQuery("limit", "10")
	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, respond.RespErr(fmt.Errorf("limit参数必须是数字"), t, 1))
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

// @Summary 根据metaId获取用户群列表
// @Description 根据metaId查询TalkMetaIdContextListCollection数据
// @Tags 数据库查询
// @Accept json
// @Produce json
// @Param metaId query string true "MetaID"
// @Success 200 {object} map[string]interface{} "查询结果"
// @Failure 400 {object} map[string]interface{} "参数错误"
// @Failure 500 {object} map[string]interface{} "服务器错误"
// @Router /api/db/user/context [get]
func (c *DbController) GetUserContext(ctx *gin.Context) {
	var t = time.Now().Unix()
	metaId := ctx.Query("metaId")
	if metaId == "" {
		ctx.JSON(http.StatusBadRequest, respond.RespErr(fmt.Errorf("metaId参数不能为空"), t, 1))
		return
	}

	result, err := c.dbService.QueryMetaIdContextList(metaId)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, respond.RespErr(err, t, 1))
		return
	}

	ctx.JSON(http.StatusOK, respond.RespSuccess(result, t))
}

// @Summary 获取数据库统计信息
// @Description 获取所有数据库集合的统计信息
// @Tags 数据库查询
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{} "统计信息"
// @Failure 500 {object} map[string]interface{} "服务器错误"
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

// @Summary 获取所有可用的数据库集合
// @Description 获取所有可用的数据库集合名称
// @Tags 数据库查询
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{} "集合列表"
// @Router /api/db/collections [get]
func (c *DbController) GetCollections(ctx *gin.Context) {
	var t = time.Now().Unix()
	collections := c.dbService.GetAvailableCollections()

	ctx.JSON(http.StatusOK, respond.RespSuccess(gin.H{
		"data":  collections,
		"count": len(collections),
	}, t))
}

// @Summary 获取所有群组版本信息列表（分页）
// @Description 获取TalkGroupVersionInfoCollection的所有数据，支持分页
// @Tags 数据库查询
// @Accept json
// @Produce json
// @Param page query int false "页码，从1开始" default(1)
// @Param size query int false "每页数量" default(20)
// @Success 200 {object} map[string]interface{} "查询结果"
// @Failure 400 {object} map[string]interface{} "参数错误"
// @Failure 500 {object} map[string]interface{} "服务器错误"
// @Router /api/db/group/version/all [get]
func (c *DbController) GetAllGroupVersionInfo(ctx *gin.Context) {
	var t = time.Now().Unix()
	pageStr := ctx.DefaultQuery("page", "1")
	sizeStr := ctx.DefaultQuery("size", "20")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		ctx.JSON(http.StatusBadRequest, respond.RespErr(fmt.Errorf("page参数必须是大于0的数字"), t, 1))
		return
	}

	size, err := strconv.Atoi(sizeStr)
	if err != nil || size < 1 || size > 100 {
		ctx.JSON(http.StatusBadRequest, respond.RespErr(fmt.Errorf("size参数必须是1-100之间的数字"), t, 1))
		return
	}

	// 计算要跳过的记录数
	skip := (page - 1) * size

	// 获取总数据量（用于分页信息）
	totalCount, err := c.dbService.GetCollectionCount("talk_group_version_info")
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, respond.RespErr(err, t, 1))
		return
	}

	// 获取分页数据
	results, err := c.dbService.QueryAll("talk_group_version_info", skip+size)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, respond.RespErr(err, t, 1))
		return
	}

	// 跳过前面的记录，获取当前页的数据
	var pageResults []map[string]interface{}
	if skip < len(results) {
		end := skip + size
		if end > len(results) {
			end = len(results)
		}
		pageResults = results[skip:end]
	}

	// 计算分页信息
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

// @Summary 获取所有聊天消息列表（分页）
// @Description 获取TalkGroupChatPinCollection的所有数据，支持分页
// @Tags 数据库查询
// @Accept json
// @Produce json
// @Param page query int false "页码，从1开始" default(1)
// @Param size query int false "每页数量" default(20)
// @Success 200 {object} map[string]interface{} "查询结果"
// @Failure 400 {object} map[string]interface{} "参数错误"
// @Failure 500 {object} map[string]interface{} "服务器错误"
// @Router /api/db/chat/pin/all [get]
func (c *DbController) GetAllChatPin(ctx *gin.Context) {
	var t = time.Now().Unix()
	pageStr := ctx.DefaultQuery("page", "1")
	sizeStr := ctx.DefaultQuery("size", "20")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		ctx.JSON(http.StatusBadRequest, respond.RespErr(fmt.Errorf("page参数必须是大于0的数字"), t, 1))
		return
	}

	size, err := strconv.Atoi(sizeStr)
	if err != nil || size < 1 || size > 100 {
		ctx.JSON(http.StatusBadRequest, respond.RespErr(fmt.Errorf("size参数必须是1-100之间的数字"), t, 1))
		return
	}

	// 计算要跳过的记录数
	skip := (page - 1) * size

	// 获取总数据量（用于分页信息）
	totalCount, err := c.dbService.GetCollectionCount("talk_group_chat_pin")
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, respond.RespErr(err, t, 1))
		return
	}

	// 获取分页数据
	results, err := c.dbService.QueryAll("talk_group_chat_pin", skip+size)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, respond.RespErr(err, t, 1))
		return
	}

	// 跳过前面的记录，获取当前页的数据
	var pageResults []map[string]interface{}
	if skip < len(results) {
		end := skip + size
		if end > len(results) {
			end = len(results)
		}
		pageResults = results[skip:end]
	}

	// 计算分页信息
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
