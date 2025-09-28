package api

import (
	"fmt"
	"manindexer/basicprotocols/group_chat/api/request"
	"manindexer/basicprotocols/group_chat/api/respond"
	"manindexer/basicprotocols/group_chat/protocols"
	"manindexer/basicprotocols/group_chat/service"
	"manindexer/common"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// SyncPinsByTimeRange Sync pin data by timestamp range
// @Summary Sync pin data by timestamp range
// @Description Sync pin data by start and end timestamps, supports custom batch size and filter conditions
// @Tags Sync Management
// @Accept json
// @Produce json
// @Param request body request.SyncPinsByTimeRangeRequest true "Sync request parameters"
// @Success 200 {object} respond.Message{data=object{message=string,startTime=string}} "Sync started successfully"
// @Failure 400 {object} respond.Message "Request parameter error"
// @Failure 500 {object} respond.Message "Internal server error"
// @Router /group-chat/sync/pins-by-time-range [post]
func SyncPinsByTimeRange(c *gin.Context) {
	var (
		t   = time.Now().UnixMilli()
		req request.SyncPinsByTimeRangeRequest

		host     = []string{}
		protocol = []string{}
	)

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, respond.RespErr(err, t, 1))
		return
	}

	// Validate timestamps
	if req.StartTs >= req.EndTs {
		c.JSON(http.StatusBadRequest, respond.RespErr(fmt.Errorf("start timestamp must be less than end timestamp"), t, 1))
		return
	}

	// Validate batch size
	if req.BatchDurationSeconds <= 0 {
		req.BatchDurationSeconds = 3600 // Default 1 hour
	}
	host = common.Config.SyncHost
	protocol = protocols.ProtocolList

	//not yet
	c.JSON(http.StatusOK, respond.RespSuccess(gin.H{
		"message": "Not yet implemented",
	}, t))
	return

	// Start sync
	err := service.GetSyncService().SyncPinsByTimeRangeWithBatch(req.StartTs, req.EndTs, host, protocol, req.BatchDurationSeconds)
	if err != nil {
		c.JSON(http.StatusInternalServerError, respond.RespErr(err, t, 1))
		return
	}

	c.JSON(http.StatusOK, respond.RespSuccess(gin.H{
		"message":              "Batch sync started, processing...",
		"startTime":            time.Now().Format("2006-01-02 15:04:05"),
		"startTs":              req.StartTs,
		"endTs":                req.EndTs,
		"batchDurationSeconds": req.BatchDurationSeconds,
	}, t))
}

// GetSyncStats Get sync status and progress
// @Summary Get sync status and progress
// @Description Get current sync service status, progress and statistics
// @Tags Sync Management
// @Produce json
// @Success 200 {object} respond.Message{data=respond.SyncStatsResponse} "Successfully get sync status"
// @Failure 500 {object} respond.Message "Internal server error"
// @Router /group-chat/sync/stats [get]
func GetSyncStats(c *gin.Context) {
	var t = time.Now().UnixMilli()
	stats := service.GetSyncService().GetSyncStats()

	// Calculate progress
	var progress float64
	var status string

	if stats.TotalPins == 0 {
		if stats.IsCompleted {
			status = "Completed"
			progress = 100.0
		} else {
			status = "Waiting"
			progress = 0.0
		}
	} else {
		progress = float64(stats.ProcessedPins) / float64(stats.TotalPins) * 100
		if stats.IsCompleted {
			status = "Completed"
		} else {
			status = "In Progress"
		}
	}

	// Return response with progress information
	responseWithProgress := map[string]interface{}{
		"totalPins":     stats.TotalPins,
		"processedPins": stats.ProcessedPins,
		"successPins":   stats.SuccessPins,
		"failedPins":    stats.FailedPins,
		"startTime":     stats.StartTime,
		"endTime":       stats.EndTime,
		"isCompleted":   stats.IsCompleted,
		"errorMessage":  stats.ErrorMessage,
		"progress":      progress,
		"status":        status,
	}

	c.JSON(http.StatusOK, respond.RespSuccess(responseWithProgress, t))
}

// GetSyncStatus Get sync running status
// @Summary Get sync running status
// @Description Check if sync service is running
// @Tags Sync Management
// @Produce json
// @Success 200 {object} respond.CommonResponse{data=object{isRunning=bool}} "Successfully get running status"
// @Failure 500 {object} respond.CommonResponse "Internal server error"
// @Router /group-chat/sync/status [get]
func GetSyncStatus(c *gin.Context) {
	var t = time.Now().UnixMilli()
	isRunning := service.GetSyncService().IsRunning()
	c.JSON(http.StatusOK, respond.RespSuccess(respond.SyncStatusResponse{
		IsRunning: isRunning,
	}, t))
}

// StopSync Stop sync service
// @Summary Stop sync service
// @Description Stop running sync service, gracefully cancel ongoing sync operations
// @Tags Sync Management
// @Produce json
// @Success 200 {object} respond.CommonResponse{data=object{message=string}} "Stop sync successfully"
// @Failure 400 {object} respond.CommonResponse "Sync service is not running"
// @Failure 500 {object} respond.CommonResponse "Internal server error"
// @Router /group-chat/sync/stop [post]
func StopSync(c *gin.Context) {
	var t = time.Now().UnixMilli()
	err := service.GetSyncService().StopSync()
	if err != nil {
		c.JSON(http.StatusBadRequest, respond.RespErr(err, t, 1))
		return
	}

	c.JSON(http.StatusOK, respond.RespSuccess(gin.H{
		"message": "Sync stopped",
	}, t))
}

// GetBlockHeightByTimestamp Query block height by timestamp
// @Summary Query block height by timestamp
// @Description Find block heights closest to the specified timestamp on MVC and BTC chains
// @Tags Block Query
// @Accept json
// @Produce json
// @Param request body request.GetBlockHeightByTimestampRequest true "Query request parameters"
// @Success 200 {object} respond.Message{data=object{timestamp=int64,results=[]respond.BlockHeightResultResponse,count=int}} "Successfully get block height information"
// @Failure 400 {object} respond.Message "Request parameter error"
// @Failure 500 {object} respond.Message "Internal server error"
// @Router /group-chat/sync/block-height-by-timestamp [post]
func GetBlockHeightByTimestamp(c *gin.Context) {
	var (
		t   = time.Now().UnixMilli()
		req request.GetBlockHeightByTimestampRequest
	)

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, respond.RespErr(err, t, 1))
		return
	}

	// Validate timestamp
	if req.Timestamp <= 0 {
		c.JSON(http.StatusBadRequest, respond.RespErr(fmt.Errorf("timestamp must be greater than 0"), t, 1))
		return
	}

	// Call service method to get block height
	results, err := service.GetBlockHeightByTimestamp(req.Timestamp)
	if err != nil {
		c.JSON(http.StatusInternalServerError, respond.RespErr(err, t, 1))
		return
	}

	// Convert to response format
	var responseResults []respond.BlockHeightResultResponse
	for _, result := range results {
		responseResults = append(responseResults, respond.BlockHeightResultResponse{
			ChainName: result.ChainName,
			Height:    result.Height,
			Timestamp: result.Timestamp,
		})
	}

	c.JSON(http.StatusOK, respond.RespSuccess(gin.H{
		"timestamp": req.Timestamp,
		"results":   responseResults,
		"count":     len(responseResults),
	}, t))
}
