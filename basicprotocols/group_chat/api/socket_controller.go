package api

import (
	"fmt"
	"manindexer/basicprotocols/group_chat/api/respond"
	"manindexer/basicprotocols/group_chat/service/socket_service"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// @Summary Get connection statistics
// @Description Get Socket connection statistics including total connections, active connections, etc.
// @Tags Socket Management
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{} "Connection statistics"
// @Failure 500 {object} map[string]interface{} "Server error"
// @Router /group-chat/socket/stats [get]
func GetConnectionStats(ctx *gin.Context) {
	var t = time.Now().Unix()

	stats := socket_service.GetConnectionStats()
	if stats == nil {
		ctx.JSON(http.StatusInternalServerError, respond.RespErr(fmt.Errorf("failed to get connection statistics"), t, 1))
		return
	}

	ctx.JSON(http.StatusOK, respond.RespSuccess(gin.H{
		"totalConnections":    stats.TotalConnections,
		"activeConnections":   stats.ActiveConnections,
		"totalMessagesSent":   stats.TotalMessagesSent,
		"totalMessagesFailed": stats.TotalMessagesFailed,
	}, t))
}

// @Summary Check if user is online
// @Description Check if a specific user is currently online based on MetaId
// @Tags Socket Management
// @Accept json
// @Produce json
// @Param metaId query string true "User MetaId"
// @Success 200 {object} map[string]interface{} "User online status"
// @Failure 400 {object} map[string]interface{} "Parameter error"
// @Failure 500 {object} map[string]interface{} "Server error"
// @Router /group-chat/socket/user-online [get]
func IsUserOnline(ctx *gin.Context) {
	var t = time.Now().Unix()

	metaId := ctx.Query("metaId")
	if metaId == "" {
		ctx.JSON(http.StatusBadRequest, respond.RespErr(fmt.Errorf("metaId parameter cannot be empty"), t, 1))
		return
	}

	isOnline := socket_service.IsUserOnline(metaId)

	ctx.JSON(http.StatusOK, respond.RespSuccess(gin.H{
		"metaId":    metaId,
		"isOnline":  isOnline,
		"timestamp": t,
	}, t))
}
