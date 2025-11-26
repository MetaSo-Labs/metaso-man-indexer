package api

import (
	"fmt"
	"log"
	"manindexer/basicprotocols/group_chat/api/request"
	"manindexer/basicprotocols/group_chat/api/respond"
	"manindexer/basicprotocols/group_chat/service"
	"manindexer/basicprotocols/group_chat/service/cache_service"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// @Summary Get group list
// @Description Get group list with pagination support
// @Produce json
// @Param metaId query string false "User MetaId"
// @Param cursor query int false "Cursor, default is 0"
// @Param size query int false "Page size, default is 20"
// @Param timestamp query int false "Timestamp"
// @Tags Group
// @Success 200 {object} respond.Message{data=respond.GroupResponse} "Successfully return group list"
// @Router /group-chat/group-list [get]
func GetGroupList(c *gin.Context) {
	var (
		t   = time.Now().UnixMilli()
		req = &request.FetchGroupListRequest{
			MetaId: c.DefaultQuery("metaId", ""),
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
	response, err := service.FetchGroupList(req)
	if err != nil {
		log.Printf("Failed to fetch group list: %v", err)
		c.JSONP(http.StatusInternalServerError, respond.RespErr(err, t, 1))
		return
	}

	c.IndentedJSON(http.StatusOK, respond.RespSuccess(response, t))
}

// @Summary Get latest chat group list
// @Description Get user's latest chat group list, sorted by latest chat time
// @Produce json
// @Param metaId query string true "User MetaId"
// @Param cursor query int false "Cursor, default is 0"
// @Param size query int false "Page size, default is 20"
// @Param timestamp query int false "Timestamp"
// @Tags Group
// @Success 200 {object} respond.Message{data=respond.GroupResponse} "Successfully return latest chat group list"
// @Router /group-chat/user/latest-group-list [get]
func GetLatestChatGroupList(c *gin.Context) {
	var (
		t   = time.Now().UnixMilli()
		req = &request.FetchLatestChatGroupListRequest{
			MetaId: c.DefaultQuery("metaId", ""),
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

	response, err := service.FetchLatestChatGroupList(req)
	if err != nil {
		log.Printf("Failed to fetch latest chat group list for metaId %s: %v", req.MetaId, err)
		c.JSONP(http.StatusInternalServerError, respond.RespErr(err, t, 1))
		return
	}

	c.IndentedJSON(http.StatusOK, respond.RespSuccess(response, t))
}

// @Summary Get group info
// @Description Get detailed information of a specified group
// @Produce json
// @Param groupId query string true "Group ID"
// @Tags Group
// @Success 200 {object} respond.Message{data=respond.GroupItem} "Successfully return group information"
// @Router /group-chat/group-info [get]
func GetGroupInfo(c *gin.Context) {
	var (
		t   = time.Now().UnixMilli()
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

// @Summary Get group chat record
// @Description Get chat records of a group, support timestamp pagination
// @Produce json
// @Param groupId query string true "Group ID"
// @Param metaId query string false "User MetaId"
// @Param cursor query int false "Cursor, default is 0"
// @Param size query int false "Page size, default is 20"
// @Param timestamp query int false "Timestamp for pagination"
// @Tags Group
// @Success 200 {object} respond.Message{data=respond.GroupChatResponse} "Successfully return group chat records"
// @Router /group-chat/group-chat-list [get]
func GetGroupChatList(c *gin.Context) {
	var (
		t   = time.Now().UnixMilli()
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

// @Summary Get group member list
// @Description Get member list of a group, support pagination
// @Produce json
// @Param groupId query string true "Group ID"
// @Param cursor query int false "Cursor, default is 0"
// @Param size query int false "Page size, default is 20"
// @Param timestamp query int false "Timestamp"
// @Param orderBy query string false "Order by field, use 'timestamp' for timestamp descending order"
// @Param orderType query string false "Order type, use 'desc' for descending order"
// @Tags Group
// @Success 200 {object} respond.Message{data=respond.GroupMemberResponse} "Successfully return group member list"
// @Router /group-chat/group-member-list [get]
func GetGroupMemberList(c *gin.Context) {
	var (
		t   = time.Now().UnixMilli()
		req = &request.FetchGroupMemberListRequest{
			GroupId: c.DefaultQuery("groupId", ""),
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
			OrderBy:   c.DefaultQuery("orderBy", ""),
			OrderType: c.DefaultQuery("orderType", ""),
		}
	)

	if req.GroupId == "" {
		c.JSONP(http.StatusBadRequest, respond.RespErr(fmt.Errorf("groupId is empty"), t, 1))
		return
	}

	// response, err := service.FetchGroupMemberList(req)
	response, err := service.FetchGroupMemberListV2(req)
	if err != nil {
		log.Printf("Failed to fetch group member list for groupId %s: %v", req.GroupId, err)
		c.JSONP(http.StatusInternalServerError, respond.RespErr(err, t, 1))
		return
	}

	c.IndentedJSON(http.StatusOK, respond.RespSuccess(response, t))
}

// @Summary Get group member info
// @Description Get TalkGroupPersonCollection information based on metaId and groupId to determine if the user is in the specified group
// @Produce json
// @Param metaId query string true "User MetaId"
// @Param groupId query string true "Group ID"
// @Tags Group
// @Success 200 {object} respond.Message{data=respond.GroupPersonResponse} "Successfully return group member information"
// @Router /group-chat/group-person [get]
func GetGroupPerson(c *gin.Context) {
	var (
		t   = time.Now().UnixMilli()
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

// @Summary Get latest chat info list (group chat + private chat)
// @Description Get user's latest chat info list, including group chats and private chats, sorted by latest chat time
// @Produce json
// @Param metaId query string true "User MetaId"
// @Param cursor query int false "Cursor, default is 0"
// @Param size query int false "Page size, default is 20"
// @Param timestamp query int false "Timestamp"
// @Tags Group
// @Success 200 {object} respond.Message{data=respond.ChatInfoResponse} "Successfully return latest chat info list"
// @Router /group-chat/user/latest-chat-info-list [get]
func GetLatestChatInfoList(c *gin.Context) {
	var (
		t   = time.Now().UnixMilli()
		req = &request.FetchLatestChatInfoListRequest{
			MetaId: c.DefaultQuery("metaId", ""),
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

	response, err := service.FetchLatestChatInfoList(req)
	if err != nil {
		log.Printf("Failed to fetch latest chat info list for metaId %s: %v", req.MetaId, err)
		c.JSONP(http.StatusInternalServerError, respond.RespErr(err, t, 1))
		return
	}

	c.IndentedJSON(http.StatusOK, respond.RespSuccess(response, t))
}

// @Summary Get private chat record
// @Description Get private chat records between two users, support timestamp pagination
// @Produce json
// @Param metaId query string true "Current user MetaId"
// @Param otherMetaId query string true "Other user MetaId"
// @Param cursor query int false "Cursor, default is 0"
// @Param size query int false "Page size, default is 20"
// @Param timestamp query int false "Timestamp for pagination"
// @Tags Group
// @Success 200 {object} respond.Message{data=respond.PrivateChatResponse} "Successfully return private chat records"
// @Router /group-chat/private-chat-list [get]
func GetPrivateChatList(c *gin.Context) {
	var (
		t   = time.Now().UnixMilli()
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

// @Summary Get private chat list by index range
// @Description Get private chat records by index range (ascending order) for pagination
// @Produce json
// @Param metaId query string true "Current user MetaId"
// @Param otherMetaId query string true "Other user MetaId"
// @Param startIndex query int false "Start index for pagination, default is 0"
// @Param size query int false "Page size, default is 20"
// @Tags Group
// @Success 200 {object} respond.Message{data=respond.PrivateChatResponse} "Successfully return private chat records by index"
// @Router /group-chat/private-chat-list-by-index [get]
func GetPrivateChatListByIndex(c *gin.Context) {
	var (
		t   = time.Now().UnixMilli()
		req = &request.FetchPrivateChatListByIndexRequest{
			MetaId:      c.DefaultQuery("metaId", ""),
			OtherMetaId: c.DefaultQuery("otherMetaId", ""),
			StartIndex: func() int64 {
				startIndex, _ := strconv.ParseInt(c.DefaultQuery("startIndex", "0"), 10, 64)
				return startIndex
			}(),
			Size: func() int64 {
				size, _ := strconv.ParseInt(c.DefaultQuery("size", "20"), 10, 64)
				return size
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

	response, err := service.FetchPrivateChatListByIndex(req)
	if err != nil {
		log.Printf("Failed to fetch private chat list by index for metaId %s and otherMetaId %s: %v", req.MetaId, req.OtherMetaId, err)
		c.JSONP(http.StatusInternalServerError, respond.RespErr(err, t, 1))
		return
	}

	c.IndentedJSON(http.StatusOK, respond.RespSuccess(response, t))
}

// @Summary Get lucky bag info
// @Description Get lucky bag object and unclaimed list based on groupId and pinId
// @Produce json
// @Param groupId query string true "Group ID"
// @Param pinId query string true "Lucky bag PinId"
// @Tags Group
// @Success 200 {object} respond.Message{data=respond.LuckyBagInfoResponse} "Successfully return lucky bag info"
// @Router /group-chat/lucky-bag-info [get]
func GetLuckyBagInfo(c *gin.Context) {
	var (
		t   = time.Now().UnixMilli()
		req = &request.FetchLuckyBagInfoRequest{
			GroupId: c.DefaultQuery("groupId", ""),
			PinId:   c.DefaultQuery("pinId", ""),
		}
		response *respond.LuckyBagInfoResponse
		err      error
	)

	if req.GroupId == "" {
		c.JSONP(http.StatusBadRequest, respond.RespErr(fmt.Errorf("groupId is empty"), t, 1))
		return
	}

	if req.PinId == "" {
		c.JSONP(http.StatusBadRequest, respond.RespErr(fmt.Errorf("pinId is empty"), t, 1))
		return
	}

	if cache_service.IsRedisInitialized() {
		log.Printf("Get lucky bag info from Redis: %s, %s", req.GroupId, req.PinId)
		response, err = service.GetLuckyBagWithOpenListV2(req.GroupId, req.PinId)
	} else {
		response, err = service.GetLuckyBagWithOpenList(req.GroupId, req.PinId)
	}
	if err != nil {
		log.Printf("Failed to get lucky bag info for groupId %s and pinId %s: %v", req.GroupId, req.PinId, err)
		c.JSONP(http.StatusInternalServerError, respond.RespErr(err, t, 1))
		return
	}

	c.IndentedJSON(http.StatusOK, respond.RespSuccess(response, t))
}

// @Summary Grab lucky bag
// @Description Grab lucky bag based on groupId, pinId, metaId, and address
// @Accept json
// @Produce json
// @Param request body request.GrabLuckyBagRequest true "Grab lucky bag request parameters"
// @Tags Group
// @Success 200 {object} respond.Message{data=string} "Successfully return grab lucky bag result"
// @Router /group-chat/grab-lucky-bag [post]
func GrabLuckyBag(c *gin.Context) {
	var (
		t      = time.Now().UnixMilli()
		req    = &request.GrabLuckyBagRequest{}
		result string
		err    error
	)

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

	if cache_service.IsRedisInitialized() {
		log.Printf("Grab lucky bag from Redis: %s, %s", req.GroupId, req.PinId)
		result, err = service.GrabLuckyBagV2(req.GroupId, req.PinId, req.MetaId, req.Address)
	} else {
		result, err = service.GrabLuckyBag(req.GroupId, req.PinId, req.MetaId, req.Address)
	}
	// result, err := service.GrabLuckyBag(req.GroupId, req.PinId, req.MetaId, req.Address)
	// result, err := service.GrabLuckyBagV2(req.GroupId, req.PinId, req.MetaId, req.Address)
	if err != nil {
		log.Printf("Failed to grab lucky bag for groupId %s, pinId %s, metaId %s: %v", req.GroupId, req.PinId, req.MetaId, err)
		c.JSONP(http.StatusInternalServerError, respond.RespErr(err, t, 1))
		return
	}

	c.IndentedJSON(http.StatusOK, respond.RespSuccess(result, t))
}

// @Summary Reclaim lucky bag
// @Description Reclaim UTXOs of expired lucky bags remaining for the person who sent the lucky bag
// @Accept json
// @Produce json
// @Param request body request.ReclaimLuckyBagRequest true "Reclaim lucky bag request parameters"
// @Tags Group
// @Success 200 {object} respond.Message{data=string} "Successfully return reclaim lucky bag result"
// @Router /group-chat/reclaim-lucky-bag [post]
func ReclaimLuckyBag(c *gin.Context) {
	var (
		t   = time.Now().UnixMilli()
		req = &request.ReclaimLuckyBagRequest{}
	)
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

// @Summary Get lucky bag unused info
// @Description Get lucky bag object and unclaimed list based on groupId and pinId
// @Produce json
// @Param groupId query string true "Group ID"
// @Param pinId query string true "Lucky bag PinId"
// @Tags Group
// @Success 200 {object} respond.Message{data=respond.LuckyBagUnusedResponse} "Successfully return lucky bag unused info"
// @Router /group-chat/lucky-bag-unused-info [get]
func GetLuckyBagUnusedInfo(c *gin.Context) {
	var (
		t   = time.Now().UnixMilli()
		req = &request.FetchLuckyBagInfoRequest{
			GroupId: c.DefaultQuery("groupId", ""),
			PinId:   c.DefaultQuery("pinId", ""),
		}
		response *respond.LuckyBagUnusedResponse
		err      error
	)

	if req.GroupId == "" {
		c.JSONP(http.StatusBadRequest, respond.RespErr(fmt.Errorf("groupId is empty"), t, 1))
		return
	}

	if req.PinId == "" {
		c.JSONP(http.StatusBadRequest, respond.RespErr(fmt.Errorf("pinId is empty"), t, 1))
		return
	}

	if cache_service.IsRedisInitialized() {
		log.Printf("Get lucky bag unused info from Redis: %s, %s", req.GroupId, req.PinId)
		response, err = service.GetLuckyBagWithUnusedListV2(req.GroupId, req.PinId)
	} else {
		response, err = service.GetLuckyBagWithUnusedList(req.GroupId, req.PinId)
	}
	if err != nil {
		log.Printf("Failed to get lucky bag unused info for groupId %s and pinId %s: %v", req.GroupId, req.PinId, err)
		c.JSONP(http.StatusInternalServerError, respond.RespErr(err, t, 1))
		return
	}

	c.IndentedJSON(http.StatusOK, respond.RespSuccess(response, t))
}

// @Summary Get lucky bag info (V2 with cache)
// @Description Get lucky bag object and claimed list based on groupId and pinId using cache optimization
// @Produce json
// @Param groupId query string true "Group ID"
// @Param pinId query string true "Lucky bag PinId"
// @Tags Group
// @Success 200 {object} respond.Message{data=respond.LuckyBagInfoResponse} "Successfully return lucky bag info"
// @Router /group-chat/lucky-bag-info-v2 [get]
func GetLuckyBagInfoV2(c *gin.Context) {
	var (
		t   = time.Now().UnixMilli()
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

	response, err := service.GetLuckyBagWithOpenListV2(req.GroupId, req.PinId)
	if err != nil {
		log.Printf("Failed to get lucky bag info V2 for groupId %s and pinId %s: %v", req.GroupId, req.PinId, err)
		c.JSONP(http.StatusInternalServerError, respond.RespErr(err, t, 1))
		return
	}

	c.IndentedJSON(http.StatusOK, respond.RespSuccess(response, t))
}

// @Summary Get lucky bag unused info (V2 with cache)
// @Description Get lucky bag object and unclaimed list based on groupId and pinId using cache optimization
// @Produce json
// @Param groupId query string true "Group ID"
// @Param pinId query string true "Lucky bag PinId"
// @Tags Group
// @Success 200 {object} respond.Message{data=respond.LuckyBagUnusedResponse} "Successfully return lucky bag unused info"
// @Router /group-chat/lucky-bag-unused-info-v2 [get]
func GetLuckyBagUnusedInfoV2(c *gin.Context) {
	var (
		t   = time.Now().UnixMilli()
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

	response, err := service.GetLuckyBagWithUnusedListV2(req.GroupId, req.PinId)
	if err != nil {
		log.Printf("Failed to get lucky bag unused info V2 for groupId %s and pinId %s: %v", req.GroupId, req.PinId, err)
		c.JSONP(http.StatusInternalServerError, respond.RespErr(err, t, 1))
		return
	}

	c.IndentedJSON(http.StatusOK, respond.RespSuccess(response, t))
}

// @Summary Grab lucky bag (V2 with cache)
// @Description Grab lucky bag based on groupId, pinId, metaId, and address using cache optimization
// @Accept json
// @Produce json
// @Param request body request.GrabLuckyBagRequest true "Grab lucky bag request parameters"
// @Tags Group
// @Success 200 {object} respond.Message{data=string} "Successfully return grab lucky bag result"
// @Router /group-chat/grab-lucky-bag-v2 [post]
func GrabLuckyBagV2(c *gin.Context) {
	var (
		t   = time.Now().UnixMilli()
		req = &request.GrabLuckyBagRequest{}
	)

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

	result, err := service.GrabLuckyBagV2(req.GroupId, req.PinId, req.MetaId, req.Address)
	if err != nil {
		log.Printf("Failed to grab lucky bag V2 for groupId %s, pinId %s, metaId %s: %v", req.GroupId, req.PinId, req.MetaId, err)
		c.JSONP(http.StatusInternalServerError, respond.RespErr(err, t, 1))
		return
	}

	c.IndentedJSON(http.StatusOK, respond.RespSuccess(result, t))
}

// @Summary Get group chat list (new format)
// @Description Get group chat records using TalkGroupChatTimestamp2Collection with improved key format
// @Produce json
// @Param groupId query string true "Group ID"
// @Param metaId query string false "User MetaId"
// @Param cursor query int false "Cursor, default is 0"
// @Param size query int false "Page size, default is 20"
// @Param timestamp query int false "Timestamp"
// @Tags Group
// @Success 200 {object} respond.Message{data=respond.GroupChatResponse} "Successfully return group chat list"
// @Router /group-chat/group-chat-list-v2 [get]
func GetGroupChatListV2(c *gin.Context) {
	var (
		t   = time.Now().UnixMilli()
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

	response, err := service.FetchGroupChatListV2(req)
	if err != nil {
		log.Printf("Failed to fetch group chat list v2 for groupId %s: %v", req.GroupId, err)
		c.JSONP(http.StatusInternalServerError, respond.RespErr(err, t, 1))
		return
	}

	c.IndentedJSON(http.StatusOK, respond.RespSuccess(response, t))
}

// @Summary Get group chat list (test version with IterOptions)
// @Description Get chat records of a group using GetChatsByGroupIdAndTimestampRange3 (test version with IterOptions)
// @Produce json
// @Param groupId query string true "Group ID"
// @Param metaId query string false "User MetaId"
// @Param cursor query int false "Cursor, default is 0"
// @Param size query int false "Page size, default is 20"
// @Param timestamp query int false "Timestamp for pagination"
// @Tags Group Management
// @Success 200 {object} respond.Message{data=respond.GroupChatResponse} "Successfully return group chat records"
// @Failure 400 {object} respond.Message{data=string} "Parameter error"
// @Router /group-chat/group-chat-list-v3 [get]
func GetGroupChatListV3(c *gin.Context) {
	var (
		t   = time.Now().UnixMilli()
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

	response, err := service.FetchGroupChatListV3(req)
	if err != nil {
		log.Printf("Failed to fetch group chat list v3 for groupId %s: %v", req.GroupId, err)
		c.JSONP(http.StatusInternalServerError, respond.RespErr(err, t, 1))
		return
	}

	c.IndentedJSON(http.StatusOK, respond.RespSuccess(response, t))
}

// @Summary Get user info by address or metaId
// @Description Get user information by address or metaId. If address is provided, it will be used; if address is empty but metaId is provided, metaId will be used; if both are empty, an error will be returned.
// @Produce json
// @Param address query string false "User address"
// @Param metaId query string false "User MetaId"
// @Tags Group
// @Success 200 {object} respond.Message{data=respond.UserInfoResponse} "Successfully return user information"
// @Failure 400 {object} respond.Message{data=string} "Parameter error"
// @Failure 500 {object} respond.Message{data=string} "Server error"
// @Router /group-chat/user-info [get]
func GetUserInfoByAddress(c *gin.Context) {
	var (
		t       = time.Now().UnixMilli()
		address = c.DefaultQuery("address", "")
		metaId  = c.DefaultQuery("metaId", "")
	)

	// Check if both address and metaId are empty
	if address == "" && metaId == "" {
		c.JSONP(http.StatusBadRequest, respond.RespErr(fmt.Errorf("either address or metaId must be provided"), t, 1))
		return
	}

	var response *respond.UserInfoResponse
	var err error

	// If address is provided, use it; otherwise use metaId
	if address != "" {
		response, err = service.GetUserInfoByAddress(address)
		if err != nil {
			log.Printf("Failed to get user info for address %s: %v", address, err)
			c.JSONP(http.StatusInternalServerError, respond.RespErr(err, t, 1))
			return
		}
	} else {
		// Use metaId when address is empty
		response, err = service.GetUserInfoByMetaId(metaId)
		if err != nil {
			log.Printf("Failed to get user info for metaId %s: %v", metaId, err)
			c.JSONP(http.StatusInternalServerError, respond.RespErr(err, t, 1))
			return
		}
	}

	c.IndentedJSON(http.StatusOK, respond.RespSuccess(response, t))
}

// @Summary Get batch user info by addresses or metaIds
// @Description Get user information by multiple addresses or metaIds in batch
// @Accept json
// @Produce json
// @Param request body request.BatchUserInfoRequest true "Batch user info request parameters"
// @Tags Group
// @Success 200 {object} respond.Message{data=respond.BatchUserInfoResponse} "Successfully return batch user information"
// @Failure 400 {object} respond.Message{data=string} "Parameter error"
// @Failure 500 {object} respond.Message{data=string} "Server error"
// @Router /group-chat/batch-user-info [post]
func GetBatchUserInfo(c *gin.Context) {
	var (
		t   = time.Now().UnixMilli()
		req = &request.BatchUserInfoRequest{}
	)

	if err := c.ShouldBindJSON(req); err != nil {
		c.JSONP(http.StatusBadRequest, respond.RespErr(fmt.Errorf("invalid request body: %v", err), t, 1))
		return
	}

	if len(req.Addresses) == 0 && len(req.MetaIds) == 0 {
		c.JSONP(http.StatusBadRequest, respond.RespErr(fmt.Errorf("both addresses and metaIds are empty"), t, 1))
		return
	}

	// limit total count to 100
	totalCount := len(req.Addresses) + len(req.MetaIds)
	if totalCount > 100 {
		c.JSONP(http.StatusBadRequest, respond.RespErr(fmt.Errorf("total count exceeds maximum limit of 100, got %d", totalCount), t, 1))
		return
	}

	response, err := service.GetBatchUserInfo(req.Addresses, req.MetaIds)
	if err != nil {
		log.Printf("Failed to get batch user info: %v", err)
		c.JSONP(http.StatusInternalServerError, respond.RespErr(err, t, 1))
		return
	}

	c.IndentedJSON(http.StatusOK, respond.RespSuccess(response, t))
}

// @Summary Get current maximum group chat index
// @Description Get the current maximum index for a group's chat records
// @Produce json
// @Param groupId query string true "Group ID"
// @Tags Group
// @Success 200 {object} respond.Message{data=respond.MaxIndexResponse} "Successfully return maximum group chat index"
// @Failure 400 {object} respond.Message{data=string} "Parameter error"
// @Failure 500 {object} respond.Message{data=string} "Server error"
// @Router /group-chat/max-group-chat-index [get]
func GetCurrentMaxGroupChatIndex(c *gin.Context) {
	var (
		t       = time.Now().UnixMilli()
		groupId = c.DefaultQuery("groupId", "")
	)

	if groupId == "" {
		c.JSONP(http.StatusBadRequest, respond.RespErr(fmt.Errorf("groupId is empty"), t, 1))
		return
	}

	response, err := service.GetCurrentMaxGroupChatIndex(groupId)
	if err != nil {
		log.Printf("Failed to get current max group chat index for groupId %s: %v", groupId, err)
		c.JSONP(http.StatusInternalServerError, respond.RespErr(err, t, 1))
		return
	}

	c.IndentedJSON(http.StatusOK, respond.RespSuccess(response, t))
}

// @Summary Get current maximum group channel chat index
// @Description Get the current maximum index for a group's chat records
// @Produce json
// @Param channelId query string true "Channel ID"
// @Tags Group
// @Success 200 {object} respond.Message{data=respond.MaxIndexResponse} "Successfully return maximum group chat index"
// @Failure 400 {object} respond.Message{data=string} "Parameter error"
// @Failure 500 {object} respond.Message{data=string} "Server error"
// @Router /group-chat/max-group-channel-chat-index [get]
func GetCurrentMaxGroupChannelChatIndex(c *gin.Context) {
	var (
		t         = time.Now().UnixMilli()
		channelId = c.DefaultQuery("channelId", "")
	)

	if channelId == "" {
		c.JSONP(http.StatusBadRequest, respond.RespErr(fmt.Errorf("channelId is empty"), t, 1))
		return
	}

	response, err := service.GetCurrentMaxGroupChannelChatIndex(channelId)
	if err != nil {
		log.Printf("Failed to get current max group channel chat index for channelId %s: %v", channelId, err)
		c.JSONP(http.StatusInternalServerError, respond.RespErr(err, t, 1))
		return
	}

	c.IndentedJSON(http.StatusOK, respond.RespSuccess(response, t))
}

// @Summary Get current maximum private chat index
// @Description Get the current maximum index for a private conversation between two users
// @Produce json
// @Param fromMetaId query string true "From user MetaId"
// @Param toMetaId query string true "To user MetaId"
// @Tags Group
// @Success 200 {object} respond.Message{data=respond.MaxIndexResponse} "Successfully return maximum private chat index"
// @Failure 400 {object} respond.Message{data=string} "Parameter error"
// @Failure 500 {object} respond.Message{data=string} "Server error"
// @Router /group-chat/max-private-chat-index [get]
func GetCurrentMaxPrivateChatIndex(c *gin.Context) {
	var (
		t          = time.Now().UnixMilli()
		fromMetaId = c.DefaultQuery("fromMetaId", "")
		toMetaId   = c.DefaultQuery("toMetaId", "")
	)

	if fromMetaId == "" {
		c.JSONP(http.StatusBadRequest, respond.RespErr(fmt.Errorf("fromMetaId is empty"), t, 1))
		return
	}

	if toMetaId == "" {
		c.JSONP(http.StatusBadRequest, respond.RespErr(fmt.Errorf("toMetaId is empty"), t, 1))
		return
	}

	response, err := service.GetCurrentMaxPrivateChatIndex(fromMetaId, toMetaId)
	if err != nil {
		log.Printf("Failed to get current max private chat index for fromMetaId %s and toMetaId %s: %v", fromMetaId, toMetaId, err)
		c.JSONP(http.StatusInternalServerError, respond.RespErr(err, t, 1))
		return
	}

	c.IndentedJSON(http.StatusOK, respond.RespSuccess(response, t))
}

// @Summary Get group chat list by index range
// @Description Get group chat records by index range (ascending order)
// @Produce json
// @Param groupId query string true "Group ID"
// @Param startIndex query int false "Start index for pagination, default is 0"
// @Param size query int false "Page size, default is 20"
// @Tags Group
// @Success 200 {object} respond.Message{data=respond.GroupChatResponse} "Successfully return group chat records by index"
// @Failure 400 {object} respond.Message{data=string} "Parameter error"
// @Failure 500 {object} respond.Message{data=string} "Server error"
// @Router /group-chat/group-chat-list-by-index [get]
func GetGroupChatListByIndex(c *gin.Context) {
	var (
		t   = time.Now().UnixMilli()
		req = &request.FetchGroupChatListByIndexRequest{
			GroupId: c.DefaultQuery("groupId", ""),
			StartIndex: func() int64 {
				startIndex, _ := strconv.ParseInt(c.DefaultQuery("startIndex", "0"), 10, 64)
				return startIndex
			}(),
			Size: func() int64 {
				size, _ := strconv.ParseInt(c.DefaultQuery("size", "20"), 10, 64)
				return size
			}(),
		}
	)

	if req.GroupId == "" {
		c.JSONP(http.StatusBadRequest, respond.RespErr(fmt.Errorf("groupId is empty"), t, 1))
		return
	}

	response, err := service.FetchGroupChatListByIndex(req)
	if err != nil {
		log.Printf("Failed to fetch group chat list by index for groupId %s: %v", req.GroupId, err)
		c.JSONP(http.StatusInternalServerError, respond.RespErr(err, t, 1))
		return
	}

	c.IndentedJSON(http.StatusOK, respond.RespSuccess(response, t))
}

// @Summary Get group chat list by start timestamp range
// @Description Get group chat records by start timestamp range (ascending order)
// @Produce json
// @Param groupId query string true "Group ID"
// @Param startTimestamp query int false "Start timestamp for pagination, default is 0"
// @Param size query int false "Page size, default is 20"
// @Tags Group
// @Success 200 {object} respond.Message{data=respond.GroupChatResponse} "Successfully return group chat records by start timestamp"
// @Failure 400 {object} respond.Message{data=string} "Parameter error"
// @Failure 500 {object} respond.Message{data=string} "Server error"
// @Router /group-chat/group-chat-list-by-start-time [get]
func GetGroupChatListByStartTime(c *gin.Context) {
	var (
		t   = time.Now().UnixMilli()
		req = &request.FetchGroupChatListByStartTimeRequest{
			GroupId: c.DefaultQuery("groupId", ""),
			StartTimestamp: func() int64 {
				startTimestamp, _ := strconv.ParseInt(c.DefaultQuery("startTimestamp", "0"), 10, 64)
				return startTimestamp
			}(),
			Size: func() int64 {
				size, _ := strconv.ParseInt(c.DefaultQuery("size", "20"), 10, 64)
				return size
			}(),
		}
	)

	if req.GroupId == "" {
		c.JSONP(http.StatusBadRequest, respond.RespErr(fmt.Errorf("groupId is empty"), t, 1))
		return
	}

	response, err := service.FetchGroupChatListByStartTime(req)
	if err != nil {
		log.Printf("Failed to fetch group chat list by start time for groupId %s: %v", req.GroupId, err)
		c.JSONP(http.StatusInternalServerError, respond.RespErr(err, t, 1))
		return
	}

	c.IndentedJSON(http.StatusOK, respond.RespSuccess(response, t))
}

// @Summary Search groups by name or ID
// @Description Search groups by name or ID using fuzzy search
// @Produce json
// @Param query query string true "Search query (group name or ID)"
// @Param size query int false "Page size, default is 20"
// @Tags Group
// @Success 200 {object} respond.Message{data=respond.GroupSearchResponse} "Successfully return search results"
// @Failure 400 {object} respond.Message{data=string} "Parameter error"
// @Failure 500 {object} respond.Message{data=string} "Server error"
// @Router /group-chat/search-groups [get]
func SearchGroups(c *gin.Context) {
	var (
		t   = time.Now().UnixMilli()
		req = &request.SearchGroupRequest{
			Query: c.DefaultQuery("query", ""),
			Size: func() int64 {
				size, _ := strconv.ParseInt(c.DefaultQuery("size", "20"), 10, 64)
				return size
			}(),
		}
	)

	if req.Query == "" {
		c.JSONP(http.StatusBadRequest, respond.RespErr(fmt.Errorf("query parameter is required"), t, 1))
		return
	}

	response, err := service.SearchGroupsByNameOrId(req)
	if err != nil {
		log.Printf("Failed to search groups for query %s: %v", req.Query, err)
		c.JSONP(http.StatusInternalServerError, respond.RespErr(err, t, 1))
		return
	}

	c.IndentedJSON(http.StatusOK, respond.RespSuccess(response, t))
}

// @Summary Search groups and users by name or ID
// @Description Search both groups and users by name or ID using fuzzy search
// @Produce json
// @Param query query string true "Search query (group name, group ID, user name, or metaId)"
// @Param size query int false "Page size, default is 5"
// @Tags Group
// @Success 200 {object} respond.Message{data=respond.GroupAndUserSearchResponse} "Successfully return combined search results"
// @Failure 400 {object} respond.Message{data=string} "Parameter error"
// @Failure 500 {object} respond.Message{data=string} "Server error"
// @Router /group-chat/search-groups-and-users [get]
func SearchGroupsAndUsers(c *gin.Context) {
	var (
		t   = time.Now().UnixMilli()
		req = &request.SearchGroupAndUserRequest{
			Query: c.DefaultQuery("query", ""),
			Size: func() int64 {
				size, _ := strconv.ParseInt(c.DefaultQuery("size", "5"), 10, 64)
				return size
			}(),
		}
	)

	if req.Query == "" {
		c.JSONP(http.StatusBadRequest, respond.RespErr(fmt.Errorf("query parameter is required"), t, 1))
		return
	}

	response, err := service.SearchGroupsAndUserByNameOrId(req)
	if err != nil {
		log.Printf("Failed to search groups and users for query %s: %v", req.Query, err)
		c.JSONP(http.StatusInternalServerError, respond.RespErr(err, t, 1))
		return
	}

	c.IndentedJSON(http.StatusOK, respond.RespSuccess(response, t))
}

// @Summary Search users by name or ID
// @Description Search users by name or ID using fuzzy search, returns results with chat public key
// @Produce json
// @Param query query string true "Search query (user name or metaId)"
// @Param size query int false "Page size, default is 20"
// @Tags Group
// @Success 200 {object} respond.Message{data=respond.UserSearchResponse} "Successfully return user search results"
// @Failure 400 {object} respond.Message{data=string} "Parameter error"
// @Failure 500 {object} respond.Message{data=string} "Server error"
// @Router /group-chat/search-users [get]
func SearchUsers(c *gin.Context) {
	var (
		t   = time.Now().UnixMilli()
		req = &request.SearchGroupAndUserRequest{
			Query: c.DefaultQuery("query", ""),
			Size: func() int64 {
				size, _ := strconv.ParseInt(c.DefaultQuery("size", "10"), 10, 64)
				return size
			}(),
		}
	)

	if req.Query == "" {
		c.JSONP(http.StatusBadRequest, respond.RespErr(fmt.Errorf("query parameter is required"), t, 1))
		return
	}

	response, err := service.SearchUserByNameOrId(req)
	if err != nil {
		log.Printf("Failed to search users for query %s: %v", req.Query, err)
		c.JSONP(http.StatusInternalServerError, respond.RespErr(err, t, 1))
		return
	}

	c.IndentedJSON(http.StatusOK, respond.RespSuccess(response, t))
}

// @Summary Get group search cache statistics
// @Description Get group search cache statistics
// @Produce json
// @Tags Group
// @Success 200 {object} respond.Message{data=map[string]interface{}} "Successfully return cache statistics"
// @Failure 500 {object} respond.Message{data=string} "Server error"
// @Router /group-chat/search-groups-cache-stats [get]
func GetGroupSearchCacheStats(c *gin.Context) {
	var t = time.Now().UnixMilli()

	response, err := service.GetGroupSearchCacheStats()
	if err != nil {
		log.Printf("Failed to get group search cache stats: %v", err)
		c.JSONP(http.StatusInternalServerError, respond.RespErr(err, t, 1))
		return
	}

	c.IndentedJSON(http.StatusOK, respond.RespSuccess(response, t))
}

// @Summary Check if chat is sendable
// @Description Check if the current system allows sending chat messages
// @Produce json
// @Tags Group
// @Success 200 {object} respond.Message{data=respond.ChatSendableResponse} "Successfully return chat sendable status"
// @Failure 500 {object} respond.Message{data=string} "Server error"
// @Router /group-chat/chat-sendable [get]
func CheckChatSendable(c *gin.Context) {
	var t = time.Now().UnixMilli()

	// For now, always return true as the system is operational
	// You can add more complex logic here based on your requirements
	// For example: check database connectivity, check blockchain status, etc.
	response := &respond.ChatSendableResponse{
		Sendable: true,
	}

	c.IndentedJSON(http.StatusOK, respond.RespSuccess(response, t))
}

// @Summary Search group members
// @Description Search group members by name, metaId, or address using fuzzy search
// @Produce json
// @Param groupId query string true "Group ID"
// @Param query query string true "Search query (user name, metaId, or address)"
// @Param size query int false "Page size, default is 20"
// @Tags Group
// @Success 200 {object} respond.Message{data=respond.GroupMemberSearchResponse} "Successfully return search results"
// @Failure 400 {object} respond.Message{data=string} "Parameter error"
// @Failure 500 {object} respond.Message{data=string} "Server error"
// @Router /group-chat/search-group-members [get]
func SearchGroupMembers(c *gin.Context) {
	var (
		t   = time.Now().UnixMilli()
		req = &request.SearchGroupMembersRequest{
			GroupId: c.DefaultQuery("groupId", ""),
			Query:   c.DefaultQuery("query", ""),
			Size: func() int64 {
				size, _ := strconv.ParseInt(c.DefaultQuery("size", "20"), 10, 64)
				return size
			}(),
		}
	)

	if req.GroupId == "" {
		c.JSONP(http.StatusBadRequest, respond.RespErr(fmt.Errorf("groupId parameter is required"), t, 1))
		return
	}

	if req.Query == "" {
		c.JSONP(http.StatusBadRequest, respond.RespErr(fmt.Errorf("query parameter is required"), t, 1))
		return
	}

	response, err := service.SearchGroupMembers(req)
	if err != nil {
		log.Printf("Failed to search group members for groupId %s, query %s: %v", req.GroupId, req.Query, err)
		c.JSONP(http.StatusInternalServerError, respond.RespErr(err, t, 1))
		return
	}

	c.IndentedJSON(http.StatusOK, respond.RespSuccess(response, t))
}

// @Summary Generate lucky bag code address key
// @Description Generate a new lucky bag code address key for frontend to use before creating a lucky bag
// @Produce json
// @Tags Group Management
// @Success 200 {object} respond.Message{data=respond.LuckyBagCodeAddressKeyResponse} "Successfully return code and address"
// @Failure 500 {object} respond.Message{data=string} "Server error"
// @Router /group-chat/generate-lucky-bag-code [get]
func GenerateLuckyBagCodeAddressKey(c *gin.Context) {
	var t = time.Now().UnixMilli()

	response, err := service.GenerateLuckyBagCodeAddressKey()
	if err != nil {
		log.Printf("Failed to generate lucky bag code address key: %v", err)
		c.JSONP(http.StatusInternalServerError, respond.RespErr(err, t, 1))
		return
	}

	c.IndentedJSON(http.StatusOK, respond.RespSuccess(response, t))
}

// @Summary Health check endpoint
// @Description Check if the group chat service is running properly
// @Produce json
// @Tags System
// @Success 200 {object} respond.Message{data=map[string]interface{}} "Service is healthy"
// @Router /health [get]
func HealthCheck(c *gin.Context) {
	var t = time.Now().UnixMilli()

	// Basic health check response
	healthData := map[string]interface{}{
		"status":    "healthy",
		"service":   "group-chat",
		"timestamp": t,
		"uptime":    time.Since(time.Unix(0, 0)).String(), // This is just a placeholder
	}

	c.IndentedJSON(http.StatusOK, respond.RespSuccess(healthData, t))
}

// ==================== Channel Chat Controller Methods ====================

// GetChannelChatListV3 Get channel chat records using GetChatsByChannelIdAndEndTimestampRange3 (test version with IterOptions for improved performance)
func GetChannelChatListV3(c *gin.Context) {
	var req request.FetchChannelChatListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, respond.RespErr(err, time.Now().UnixMilli(), 400))
		return
	}

	if req.ChannelId == "" {
		c.JSON(http.StatusBadRequest, respond.Message{
			Code:    400,
			Message: "ChannelId is required",
			Data:    nil,
		})
		return
	}

	result, err := service.FetchChannelChatListV3(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, respond.RespErr(err, time.Now().UnixMilli(), 500))
		return
	}

	c.JSON(http.StatusOK, respond.RespSuccess(result, time.Now().UnixMilli()))
}

// GetChannelChatListByIndex Get channel chat records by index range (ascending order) using TalkGroupChannelChatIndexCollection
func GetChannelChatListByIndex(c *gin.Context) {
	var req request.FetchChannelChatListByIndexRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, respond.RespErr(err, time.Now().UnixMilli(), 400))
		return
	}

	if req.ChannelId == "" {
		c.JSON(http.StatusBadRequest, respond.Message{
			Code:    400,
			Message: "ChannelId is required",
			Data:    nil,
		})
		return
	}

	result, err := service.FetchChannelChatListByIndex(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, respond.RespErr(err, time.Now().UnixMilli(), 500))
		return
	}

	c.JSON(http.StatusOK, respond.RespSuccess(result, time.Now().UnixMilli()))
}

// GetChannelChatListByStartTime Get channel chat records by start timestamp range (ascending order) using TalkGroupChannelChatTimestamp2Collection
func GetChannelChatListByStartTime(c *gin.Context) {
	var req request.FetchChannelChatListByStartTimeRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, respond.RespErr(err, time.Now().UnixMilli(), 400))
		return
	}

	if req.ChannelId == "" {
		c.JSON(http.StatusBadRequest, respond.Message{
			Code:    400,
			Message: "ChannelId is required",
			Data:    nil,
		})
		return
	}

	result, err := service.FetchChannelChatListByStartTime(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, respond.RespErr(err, time.Now().UnixMilli(), 500))
		return
	}

	c.JSON(http.StatusOK, respond.RespSuccess(result, time.Now().UnixMilli()))
}

// ==================== Group Channel Controller Methods ====================

// GetGroupChannelList Get group channel list by group ID
func GetGroupChannelList(c *gin.Context) {
	var req request.FetchGroupChannelListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, respond.RespErr(err, time.Now().UnixMilli(), 400))
		return
	}

	if req.GroupId == "" {
		c.JSON(http.StatusBadRequest, respond.Message{
			Code:    400,
			Message: "GroupId is required",
			Data:    nil,
		})
		return
	}

	result, err := service.FetchGroupChannelList(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, respond.RespErr(err, time.Now().UnixMilli(), 500))
		return
	}

	c.JSON(http.StatusOK, respond.RespSuccess(result, time.Now().UnixMilli()))
}

// GetGroupUserRoleInfo Fetch group user role info
// @Summary Fetch group user role info
// @Description Get user's role information in the group, including whether they are creator, admin, blacklisted, whitelisted, etc.
// @Tags Group Management
// @Accept json
// @Produce json
// @Param groupId query string true "Group ID"
// @Param channelId query string false "Channel ID (optional)"
// @Param metaId query string true "User MetaId"
// @Success 200 {object} respond.RespSuccess{data=respond.GroupUserRoleInfo} "Successfully get user role information"
// @Failure 400 {object} respond.RespError "Request parameter error"
// @Failure 500 {object} respond.RespError "Internal server error"
// @Router /group-chat/group-user-role [get]
func GetGroupUserRoleInfo(c *gin.Context) {
	var (
		t   = time.Now().UnixMilli()
		req = &request.FetchGroupUserRoleInfoRequest{
			GroupId:   c.DefaultQuery("groupId", ""),
			ChannelId: c.DefaultQuery("channelId", ""),
			MetaId:    c.DefaultQuery("metaId", ""),
		}
	)

	// Parameter validation
	if req.GroupId == "" {
		c.JSONP(http.StatusBadRequest, respond.RespErr(fmt.Errorf("groupId is empty"), t, 1))
		return
	}
	if req.MetaId == "" {
		c.JSONP(http.StatusBadRequest, respond.RespErr(fmt.Errorf("metaId is empty"), t, 1))
		return
	}

	result, err := service.FetchGroupUserRoleInfo(req)
	if err != nil {
		c.JSONP(http.StatusInternalServerError, respond.RespErr(err, t, 1))
		return
	}

	c.JSONP(http.StatusOK, respond.RespSuccess(result, t))
}

// IsSyncCompleted Check if sync is completed
// @Summary Check if sync is completed
// @Description Check if data synchronization is completed
// @Tags Group Management
// @Accept json
// @Produce json
// @Success 200 {object} respond.RespSuccess{data=bool} "Successfully return sync status"
// @Failure 500 {object} respond.RespError "Internal server error"
// @Router /group-chat/sync-completed [get]
func IsSyncCompleted(c *gin.Context) {
	var t = time.Now().UnixMilli()

	result := service.IsSyncCompleted()

	c.JSONP(http.StatusOK, respond.RespSuccess(result, t))
}

// GetGroupJoinControlList Get group join block and whitelist metaId list
// @Summary Get group join block and whitelist metaId list
// @Description Get the current effective join block and whitelist metaId lists for a group
// @Tags Group Management
// @Accept json
// @Produce json
// @Param groupId query string true "Group ID"
// @Success 200 {object} respond.RespSuccess{data=respond.GroupJoinControlListResponse} "Successfully return join control lists"
// @Failure 400 {object} respond.RespError "Request parameter error"
// @Failure 500 {object} respond.RespError "Internal server error"
// @Router /group-chat/group-join-control-list [get]
func GetGroupJoinControlList(c *gin.Context) {
	var (
		t   = time.Now().UnixMilli()
		req = &request.FetchGroupJoinControlListRequest{
			GroupId: c.DefaultQuery("groupId", ""),
		}
	)

	if req.GroupId == "" {
		c.JSONP(http.StatusBadRequest, respond.RespErr(fmt.Errorf("groupId is empty"), t, 1))
		return
	}

	result, err := service.FetchGroupJoinControlList(req)
	if err != nil {
		log.Printf("Failed to fetch group join control list for groupId %s: %v", req.GroupId, err)
		c.JSONP(http.StatusInternalServerError, respond.RespErr(err, t, 1))
		return
	}

	c.JSONP(http.StatusOK, respond.RespSuccess(result, t))
}

// GetPrivateGroupPaths Get private group paths by MetaId
// @Summary Get private group paths by MetaId
// @Description Get all private group paths (path, groupId, pinId) used by a user
// @Tags Group Management
// @Accept json
// @Produce json
// @Param metaId query string true "User MetaId"
// @Success 200 {object} respond.RespSuccess{data=respond.PrivateGroupPathsResponse} "Successfully return private group paths"
// @Failure 400 {object} respond.RespError "Request parameter error"
// @Failure 500 {object} respond.RespError "Internal server error"
// @Router /group-chat/private-group-paths [get]
func GetPrivateGroupPaths(c *gin.Context) {
	var (
		t   = time.Now().UnixMilli()
		req = &request.FetchPrivateGroupPathsRequest{
			MetaId: c.DefaultQuery("metaId", ""),
		}
	)

	if req.MetaId == "" {
		c.JSONP(http.StatusBadRequest, respond.RespErr(fmt.Errorf("metaId is required"), t, 1))
		return
	}

	result, err := service.FetchPrivateGroupPaths(req)
	if err != nil {
		log.Printf("Failed to fetch private group paths for metaId %s: %v", req.MetaId, err)
		c.JSONP(http.StatusInternalServerError, respond.RespErr(err, t, 1))
		return
	}

	c.JSONP(http.StatusOK, respond.RespSuccess(result, t))
}

// GetGroupMetaIdJoinList Get group MetaId join list
// @Summary Get group MetaId join list
// @Description Get user's group join records (create, join, leave, remove) for a specific group from TalkGroupMetaIdJoinCollection
// @Tags Group Management
// @Accept json
// @Produce json
// @Param metaId query string true "User MetaId"
// @Param groupId query string true "Group ID"
// @Success 200 {object} respond.RespSuccess{data=respond.GroupMetaIdJoinListResponse} "Successfully return group MetaId join list"
// @Failure 400 {object} respond.RespError "Request parameter error"
// @Failure 500 {object} respond.RespError "Internal server error"
// @Router /group-chat/group-metaid-join-list [get]
func GetGroupMetaIdJoinList(c *gin.Context) {
	var (
		t   = time.Now().UnixMilli()
		req = &request.FetchGroupMetaIdJoinListRequest{
			MetaId:  c.DefaultQuery("metaId", ""),
			GroupId: c.DefaultQuery("groupId", ""),
		}
	)

	if req.MetaId == "" {
		c.JSONP(http.StatusBadRequest, respond.RespErr(fmt.Errorf("metaId is required"), t, 1))
		return
	}

	if req.GroupId == "" {
		c.JSONP(http.StatusBadRequest, respond.RespErr(fmt.Errorf("groupId is required"), t, 1))
		return
	}

	result, err := service.FetchGroupMetaIdJoinList(req)
	if err != nil {
		log.Printf("Failed to fetch group MetaId join list for metaId %s, groupId %s: %v", req.MetaId, req.GroupId, err)
		c.JSONP(http.StatusInternalServerError, respond.RespErr(err, t, 1))
		return
	}

	c.JSONP(http.StatusOK, respond.RespSuccess(result, t))
}
