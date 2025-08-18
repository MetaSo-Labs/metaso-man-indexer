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
		t   = time.Now().Unix()
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
		t   = time.Now().Unix()
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

// @Summary Get group member list
// @Description Get member list of a group, support pagination
// @Produce json
// @Param groupId query string true "Group ID"
// @Param cursor query int false "Cursor, default is 0"
// @Param size query int false "Page size, default is 20"
// @Param timestamp query int false "Timestamp"
// @Tags Group
// @Success 200 {object} respond.Message{data=respond.GroupMemberResponse} "Successfully return group member list"
// @Router /group-chat/group-member-list [get]
func GetGroupMemberList(c *gin.Context) {
	var (
		t   = time.Now().Unix()
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
		t   = time.Now().Unix()
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
		t   = time.Now().Unix()
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

	result, err := service.GrabLuckyBag(req.GroupId, req.PinId, req.MetaId, req.Address)
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
		t   = time.Now().Unix()
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

	response, err := service.FetchGroupChatListV2(req)
	if err != nil {
		log.Printf("Failed to fetch group chat list v2 for groupId %s: %v", req.GroupId, err)
		c.JSONP(http.StatusInternalServerError, respond.RespErr(err, t, 1))
		return
	}

	c.IndentedJSON(http.StatusOK, respond.RespSuccess(response, t))
}
