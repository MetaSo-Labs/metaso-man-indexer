package api

import (
	"fmt"
	"log"
	"manindexer/basicprotocols/group_chat/api/request"
	"manindexer/basicprotocols/group_chat/api/respond"
	"manindexer/basicprotocols/group_chat/service"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// @Summary 获取群组列表
// @Description 获取群组列表，支持分页
// @Produce json
// @Param metaId query string false "用户MetaId"
// @Param cursor query int false "游标，默认为1"
// @Param size query int false "每页大小，默认为20"
// @Param timestamp query int false "时间戳"
// @Tags Group
// @Success 200 {object} respond.Message{data=respond.GroupResponse} "成功返回群组列表"
// @Router /group-chat/group-list [get]
func GetGroupList(c *gin.Context) {
	var (
		t   = time.Now().Unix()
		req = &request.FetchGroupListRequest{
			MetaId: c.DefaultQuery("metaId", ""),
			Cursor: func() int64 {
				cursor, _ := strconv.ParseInt(c.DefaultQuery("cursor", "1"), 10, 64)
				return cursor
			}(),
			Size: func() int64 {
				size, _ := strconv.ParseInt(c.DefaultQuery("size", "20"), 10, 64)
				return size
			}(),
			Timestamp: func() int64 {
				timestamp, _ := strconv.ParseInt(c.DefaultQuery("timestamp", "0"), 10, 64)
				return timestamp
			}(),
		}
	)
	response, err := service.FetchGroupList(req)
	if err != nil {
		log.Printf("Failed to fetch group list: %v", err)
		c.JSONP(http.StatusInternalServerError, respond.RespErr(err, t, 1))
		return
	}

	c.IndentedJSON(http.StatusOK, respond.RespSuccess(response, t))
}

// @Summary 获取最新聊天群组列表
// @Description 获取用户的最新聊天群组列表，基于最新聊天时间排序
// @Produce json
// @Param metaId query string true "用户MetaId"
// @Param cursor query int false "游标，默认为1"
// @Param size query int false "每页大小，默认为20"
// @Param timestamp query int false "时间戳"
// @Tags Group
// @Success 200 {object} respond.Message{data=respond.GroupResponse} "成功返回最新聊天群组列表"
// @Router /group-chat/user/latest-group-list [get]
func GetLatestChatGroupList(c *gin.Context) {
	var (
		t   = time.Now().Unix()
		req = &request.FetchLatestChatGroupListRequest{
			MetaId: c.DefaultQuery("metaId", ""),
			Cursor: func() int64 {
				cursor, _ := strconv.ParseInt(c.DefaultQuery("cursor", "1"), 10, 64)
				return cursor
			}(),
			Size: func() int64 {
				size, _ := strconv.ParseInt(c.DefaultQuery("size", "20"), 10, 64)
				return size
			}(),
			Timestamp: func() int64 {
				timestamp, _ := strconv.ParseInt(c.DefaultQuery("timestamp", "0"), 10, 64)
				return timestamp
			}(),
		}
	)

	if req.MetaId == "" {
		c.JSONP(http.StatusBadRequest, respond.RespErr(fmt.Errorf("metaId is empty"), t, 1))
		return
	}

	response, err := service.FetchLatestChatGroupList(req)
	if err != nil {
		log.Printf("Failed to fetch latest chat group list for metaId %s: %v", req.MetaId, err)
		c.JSONP(http.StatusInternalServerError, respond.RespErr(err, t, 1))
		return
	}

	c.IndentedJSON(http.StatusOK, respond.RespSuccess(response, t))
}

// @Summary 获取群组信息
// @Description 获取指定群组的详细信息
// @Produce json
// @Param groupId query string true "群组ID"
// @Tags Group
// @Success 200 {object} respond.Message{data=respond.GroupItem} "成功返回群组信息"
// @Router /group-chat/group-info [get]
func GetGroupInfo(c *gin.Context) {
	var (
		t   = time.Now().Unix()
		req = &request.FetchGroupInfoRequest{
			GroupId: c.DefaultQuery("groupId", ""),
		}
	)

	if req.GroupId == "" {
		c.JSONP(http.StatusBadRequest, respond.RespErr(fmt.Errorf("groupId is empty"), t, 1))
		return
	}

	response, err := service.FetchGroupInfo(req)
	if err != nil {
		log.Printf("Failed to fetch group info for groupId %s: %v", req.GroupId, err)
		c.JSONP(http.StatusInternalServerError, respond.RespErr(err, t, 1))
		return
	}

	if response == nil {
		c.JSONP(http.StatusNotFound, respond.RespErr(fmt.Errorf("Group not found"), t, 1))
		return
	}

	c.IndentedJSON(http.StatusOK, respond.RespSuccess(response, t))
}

// @Summary 获取群组聊天记录
// @Description 获取群组的聊天记录，支持时间戳分页
// @Produce json
// @Param groupId query string true "群组ID"
// @Param metaId query string false "用户MetaId"
// @Param cursor query int false "游标，默认为0"
// @Param size query int false "每页大小，默认为20"
// @Param timestamp query int false "时间戳，用于分页"
// @Tags Group
// @Success 200 {object} respond.Message{data=respond.GroupChatResponse} "成功返回群组聊天记录"
// @Router /group-chat/group-chat-list [get]
func GetGroupChatList(c *gin.Context) {
	var (
		t   = time.Now().Unix()
		req = &request.FetchGroupChatListRequest{
			GroupId: c.DefaultQuery("groupId", ""),
			MetaId:  c.DefaultQuery("metaId", ""),
			Cursor: func() int64 {
				cursor, _ := strconv.ParseInt(c.DefaultQuery("cursor", "0"), 10, 64)
				return cursor
			}(),
			Size: func() int64 {
				size, _ := strconv.ParseInt(c.DefaultQuery("size", "20"), 10, 64)
				return size
			}(),
			Timestamp: func() int64 {
				timestamp, _ := strconv.ParseInt(c.DefaultQuery("timestamp", "0"), 10, 64)
				return timestamp
			}(),
		}
	)

	if req.GroupId == "" {
		c.JSONP(http.StatusBadRequest, respond.RespErr(fmt.Errorf("groupId is empty"), t, 1))
		return
	}

	response, err := service.FetchGroupChatList(req)
	if err != nil {
		log.Printf("Failed to fetch group chat list for groupId %s: %v", req.GroupId, err)
		c.JSONP(http.StatusInternalServerError, respond.RespErr(err, t, 1))
		return
	}

	c.IndentedJSON(http.StatusOK, respond.RespSuccess(response, t))
}

// @Summary 获取群组成员列表
// @Description 获取群组的成员列表，支持分页
// @Produce json
// @Param groupId query string true "群组ID"
// @Param cursor query int false "游标，默认为1"
// @Param size query int false "每页大小，默认为20"
// @Param timestamp query int false "时间戳"
// @Tags Group
// @Success 200 {object} respond.Message{data=respond.GroupMemberResponse} "成功返回群组成员列表"
// @Router /group-chat/group-member-list [get]
func GetGroupMemberList(c *gin.Context) {
	var (
		t   = time.Now().Unix()
		req = &request.FetchGroupMemberListRequest{
			GroupId: c.DefaultQuery("groupId", ""),
			Cursor: func() int64 {
				cursor, _ := strconv.ParseInt(c.DefaultQuery("cursor", "1"), 10, 64)
				return cursor
			}(),
			Size: func() int64 {
				size, _ := strconv.ParseInt(c.DefaultQuery("size", "20"), 10, 64)
				return size
			}(),
			Timestamp: func() int64 {
				timestamp, _ := strconv.ParseInt(c.DefaultQuery("timestamp", "0"), 10, 64)
				return timestamp
			}(),
		}
	)

	if req.GroupId == "" {
		c.JSONP(http.StatusBadRequest, respond.RespErr(fmt.Errorf("groupId is empty"), t, 1))
		return
	}

	response, err := service.FetchGroupMemberList(req)
	if err != nil {
		log.Printf("Failed to fetch group member list for groupId %s: %v", req.GroupId, err)
		c.JSONP(http.StatusInternalServerError, respond.RespErr(err, t, 1))
		return
	}

	c.IndentedJSON(http.StatusOK, respond.RespSuccess(response, t))
}

// @Summary 获取群组成员信息
// @Description 根据metaId和groupId获取TalkGroupPersonCollection信息，判断用户是否在指定群组中
// @Produce json
// @Param metaId query string true "用户MetaId"
// @Param groupId query string true "群组ID"
// @Tags Group
// @Success 200 {object} respond.Message{data=respond.GroupPersonResponse} "成功返回群组成员信息"
// @Router /group-chat/group-person [get]
func GetGroupPerson(c *gin.Context) {
	var (
		t   = time.Now().Unix()
		req = &request.FetchGroupPersonRequest{
			MetaId:  c.DefaultQuery("metaId", ""),
			GroupId: c.DefaultQuery("groupId", ""),
		}
	)

	if req.MetaId == "" {
		c.JSONP(http.StatusBadRequest, respond.RespErr(fmt.Errorf("metaId is empty"), t, 1))
		return
	}

	if req.GroupId == "" {
		c.JSONP(http.StatusBadRequest, respond.RespErr(fmt.Errorf("groupId is empty"), t, 1))
		return
	}

	response, err := service.FetchGroupPerson(req)
	if err != nil {
		log.Printf("Failed to fetch group person for metaId %s and groupId %s: %v", req.MetaId, req.GroupId, err)
		c.JSONP(http.StatusInternalServerError, respond.RespErr(err, t, 1))
		return
	}

	c.IndentedJSON(http.StatusOK, respond.RespSuccess(response, t))
}

// @Summary 获取最新聊天信息列表（群聊+私聊）
// @Description 获取用户的最新聊天信息列表，包括群聊和私聊，基于最新聊天时间排序
// @Produce json
// @Param metaId query string true "用户MetaId"
// @Param cursor query int false "游标，默认为1"
// @Param size query int false "每页大小，默认为20"
// @Param timestamp query int false "时间戳"
// @Tags Group
// @Success 200 {object} respond.Message{data=respond.ChatInfoResponse} "成功返回最新聊天信息列表"
// @Router /group-chat/user/latest-chat-info-list [get]
func GetLatestChatInfoList(c *gin.Context) {
	var (
		t   = time.Now().Unix()
		req = &request.FetchLatestChatInfoListRequest{
			MetaId: c.DefaultQuery("metaId", ""),
			Cursor: func() int64 {
				cursor, _ := strconv.ParseInt(c.DefaultQuery("cursor", "1"), 10, 64)
				return cursor
			}(),
			Size: func() int64 {
				size, _ := strconv.ParseInt(c.DefaultQuery("size", "20"), 10, 64)
				return size
			}(),
			Timestamp: func() int64 {
				timestamp, _ := strconv.ParseInt(c.DefaultQuery("timestamp", "0"), 10, 64)
				return timestamp
			}(),
		}
	)

	if req.MetaId == "" {
		c.JSONP(http.StatusBadRequest, respond.RespErr(fmt.Errorf("metaId is empty"), t, 1))
		return
	}

	response, err := service.FetchLatestChatInfoList(req)
	if err != nil {
		log.Printf("Failed to fetch latest chat info list for metaId %s: %v", req.MetaId, err)
		c.JSONP(http.StatusInternalServerError, respond.RespErr(err, t, 1))
		return
	}

	c.IndentedJSON(http.StatusOK, respond.RespSuccess(response, t))
}

// @Summary 获取私聊记录
// @Description 获取两个用户之间的私聊记录，支持时间戳分页
// @Produce json
// @Param metaId query string true "当前用户MetaId"
// @Param otherMetaId query string true "对方用户MetaId"
// @Param cursor query int false "游标，默认为0"
// @Param size query int false "每页大小，默认为20"
// @Param timestamp query int false "时间戳，用于分页"
// @Tags Group
// @Success 200 {object} respond.Message{data=respond.PrivateChatResponse} "成功返回私聊记录"
// @Router /group-chat/private-chat-list [get]
func GetPrivateChatList(c *gin.Context) {
	var (
		t   = time.Now().Unix()
		req = &request.FetchPrivateChatListRequest{
			MetaId:      c.DefaultQuery("metaId", ""),
			OtherMetaId: c.DefaultQuery("otherMetaId", ""),
			Cursor: func() int64 {
				cursor, _ := strconv.ParseInt(c.DefaultQuery("cursor", "0"), 10, 64)
				return cursor
			}(),
			Size: func() int64 {
				size, _ := strconv.ParseInt(c.DefaultQuery("size", "20"), 10, 64)
				return size
			}(),
			Timestamp: func() int64 {
				timestamp, _ := strconv.ParseInt(c.DefaultQuery("timestamp", "0"), 10, 64)
				return timestamp
			}(),
		}
	)

	if req.MetaId == "" {
		c.JSONP(http.StatusBadRequest, respond.RespErr(fmt.Errorf("metaId is empty"), t, 1))
		return
	}

	if req.OtherMetaId == "" {
		c.JSONP(http.StatusBadRequest, respond.RespErr(fmt.Errorf("otherMetaId is empty"), t, 1))
		return
	}

	response, err := service.FetchPrivateChatList(req)
	if err != nil {
		log.Printf("Failed to fetch private chat list for metaId %s and otherMetaId %s: %v", req.MetaId, req.OtherMetaId, err)
		c.JSONP(http.StatusInternalServerError, respond.RespErr(err, t, 1))
		return
	}

	c.IndentedJSON(http.StatusOK, respond.RespSuccess(response, t))
}

// @Summary 获取红包信息
// @Description 根据groupId和pinId获取红包对象和已领取列表
// @Produce json
// @Param groupId query string true "群组ID"
// @Param pinId query string true "红包PinId"
// @Tags Group
// @Success 200 {object} respond.Message{data=respond.LuckyBagInfoResponse} "成功返回红包信息"
// @Router /group-chat/lucky-bag-info [get]
func GetLuckyBagInfo(c *gin.Context) {
	var (
		t   = time.Now().Unix()
		req = &request.FetchLuckyBagInfoRequest{
			GroupId: c.DefaultQuery("groupId", ""),
			PinId:   c.DefaultQuery("pinId", ""),
		}
	)

	if req.GroupId == "" {
		c.JSONP(http.StatusBadRequest, respond.RespErr(fmt.Errorf("groupId is empty"), t, 1))
		return
	}

	if req.PinId == "" {
		c.JSONP(http.StatusBadRequest, respond.RespErr(fmt.Errorf("pinId is empty"), t, 1))
		return
	}

	response, err := service.GetLuckyBagWithOpenList(req.GroupId, req.PinId)
	if err != nil {
		log.Printf("Failed to get lucky bag info for groupId %s and pinId %s: %v", req.GroupId, req.PinId, err)
		c.JSONP(http.StatusInternalServerError, respond.RespErr(err, t, 1))
		return
	}

	c.IndentedJSON(http.StatusOK, respond.RespSuccess(response, t))
}

// @Summary 抢红包
// @Description 根据groupId、pinId、metaId和address抢红包
// @Accept json
// @Produce json
// @Param request body request.GrabLuckyBagRequest true "抢红包请求参数"
// @Tags Group
// @Success 200 {object} respond.Message{data=string} "成功返回抢红包结果"
// @Router /group-chat/grab-lucky-bag [post]
func GrabLuckyBag(c *gin.Context) {
	var (
		t   = time.Now().Unix()
		req = &request.GrabLuckyBagRequest{}
	)

	// 绑定JSON请求体
	if err := c.ShouldBindJSON(req); err != nil {
		c.JSONP(http.StatusBadRequest, respond.RespErr(fmt.Errorf("invalid request body: %v", err), t, 1))
		return
	}

	if req.GroupId == "" {
		c.JSONP(http.StatusBadRequest, respond.RespErr(fmt.Errorf("groupId is empty"), t, 1))
		return
	}

	if req.PinId == "" {
		c.JSONP(http.StatusBadRequest, respond.RespErr(fmt.Errorf("pinId is empty"), t, 1))
		return
	}

	if req.MetaId == "" {
		c.JSONP(http.StatusBadRequest, respond.RespErr(fmt.Errorf("metaId is empty"), t, 1))
		return
	}

	if req.Address == "" {
		c.JSONP(http.StatusBadRequest, respond.RespErr(fmt.Errorf("address is empty"), t, 1))
		return
	}

	result, err := service.GrabLuckyBag(req.GroupId, req.PinId, req.MetaId, req.Address)
	if err != nil {
		log.Printf("Failed to grab lucky bag for groupId %s, pinId %s, metaId %s: %v", req.GroupId, req.PinId, req.MetaId, err)
		c.JSONP(http.StatusInternalServerError, respond.RespErr(err, t, 1))
		return
	}

	c.IndentedJSON(http.StatusOK, respond.RespSuccess(result, t))
}

// @Summary 回收红包
// @Description 发红包的人回收过时红包剩余的UTXO
// @Accept json
// @Produce json
// @Param request body request.ReclaimLuckyBagRequest true "回收红包请求参数"
// @Tags Group
// @Success 200 {object} respond.Message{data=string} "成功返回回收红包结果"
// @Router /group-chat/reclaim-lucky-bag [post]
func ReclaimLuckyBag(c *gin.Context) {
	var (
		t   = time.Now().Unix()
		req = &request.ReclaimLuckyBagRequest{}
	)

	// 绑定JSON请求体
	if err := c.ShouldBindJSON(req); err != nil {
		c.JSONP(http.StatusBadRequest, respond.RespErr(fmt.Errorf("invalid request body: %v", err), t, 1))
		return
	}

	if req.GroupId == "" {
		c.JSONP(http.StatusBadRequest, respond.RespErr(fmt.Errorf("groupId is empty"), t, 1))
		return
	}

	if req.PinId == "" {
		c.JSONP(http.StatusBadRequest, respond.RespErr(fmt.Errorf("pinId is empty"), t, 1))
		return
	}

	if req.MetaId == "" {
		c.JSONP(http.StatusBadRequest, respond.RespErr(fmt.Errorf("metaId is empty"), t, 1))
		return
	}

	if req.Address == "" {
		c.JSONP(http.StatusBadRequest, respond.RespErr(fmt.Errorf("address is empty"), t, 1))
		return
	}

	result, err := service.ReclaimExpiredLuckyBag(req.GroupId, req.PinId, req.MetaId, req.Address)
	if err != nil {
		log.Printf("Failed to reclaim lucky bag for groupId %s, pinId %s, metaId %s: %v", req.GroupId, req.PinId, req.MetaId, err)
		c.JSONP(http.StatusInternalServerError, respond.RespErr(err, t, 1))
		return
	}

	c.IndentedJSON(http.StatusOK, respond.RespSuccess(result, t))
}

// @Summary 获取红包未领取信息
// @Description 根据groupId和pinId获取红包对象和未领取列表
// @Produce json
// @Param groupId query string true "群组ID"
// @Param pinId query string true "红包PinId"
// @Tags Group
// @Success 200 {object} respond.Message{data=respond.LuckyBagUnusedResponse} "成功返回红包未领取信息"
// @Router /group-chat/lucky-bag-unused-info [get]
func GetLuckyBagUnusedInfo(c *gin.Context) {
	var (
		t   = time.Now().Unix()
		req = &request.FetchLuckyBagInfoRequest{
			GroupId: c.DefaultQuery("groupId", ""),
			PinId:   c.DefaultQuery("pinId", ""),
		}
	)

	if req.GroupId == "" {
		c.JSONP(http.StatusBadRequest, respond.RespErr(fmt.Errorf("groupId is empty"), t, 1))
		return
	}

	if req.PinId == "" {
		c.JSONP(http.StatusBadRequest, respond.RespErr(fmt.Errorf("pinId is empty"), t, 1))
		return
	}

	response, err := service.GetLuckyBagWithUnusedList(req.GroupId, req.PinId)
	if err != nil {
		log.Printf("Failed to get lucky bag unused info for groupId %s and pinId %s: %v", req.GroupId, req.PinId, err)
		c.JSONP(http.StatusInternalServerError, respond.RespErr(err, t, 1))
		return
	}

	c.IndentedJSON(http.StatusOK, respond.RespSuccess(response, t))
}
