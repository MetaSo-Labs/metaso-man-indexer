package metaso

import (
	"manindexer/adapter/bitcoin"
	"manindexer/database/mongodb"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

func StatisticsApi(r *gin.Engine) {
	group := r.Group("/statistics")
	group.Use(CorsMiddleware())
	group.GET("/host/metablock/sync-newest", blockSyncNewest)
	group.GET("/host/metablock/info", blockNDVPageList)
	group.GET("/metablock/address/info", blockMDVPageList)
	group.GET("/ndv", ndvPageList)
	group.GET("/mdv", mdvPageList)
	group.GET("/metablock/host/value", hostValuePageList)
	group.GET("/metablock/host/address/list", hostAddressValuePageList)
	group.GET("/metablock/host/address/value", hostAddressValue)
}

// @Summary      Get latest block sync status
// @Description  Retrieve the current synchronization status of meta blocks
// @Tags         Statistics
// @Accept       json
// @Produce      json
// @Success      200  {object}  ApiResponse{data=object{currentMetaBlockHeight=int,syncMetaBlockHeight=int,progressStartBlock=int,progressEndBlock=int,initBlockHeight=int,currentBlockHeight=int}}  "Block synchronization status"
// @Failure      500  {object}  ApiResponse  "Service exception"
// @Router       /statistics/host/metablock/sync-newest [get]
func blockSyncNewest(ctx *gin.Context) {
	currentMetaBlockHeight := int64(0)
	syncMetaBlockHeight := int64(0)
	progressStartBlock := int64(0)
	progressEndBlock := int64(0)
	initBlockHeight := int64(0)

	lastBlockInfo := getLastMetaBlock()
	if lastBlockInfo == nil {
		ctx.JSON(http.StatusOK, ApiError(-1, "no last meta block info"))
		return
	}
	preEnd := int64(0)
	for _, chain := range lastBlockInfo.BlockData.Chains {
		if chain.Chain == "Bitcoin" {
			preEnd, _ = strconv.ParseInt(chain.PreEndBlock, 10, 64)
			break
		}
	}

	currentMetaBlockHeight = lastBlockInfo.LastNumber + 1
	syncMetaBlockHeight, _ = mongodb.GetSyncLastNumber("metablock")
	progressStartBlock = preEnd + 1
	progressEndBlock = preEnd + int64(lastBlockInfo.Step)
	initBlockHeight = lastBlockInfo.Init
	btc := bitcoin.BitcoinChain{}
	currentBlockHeight := btc.GetBestHeight()
	ctx.JSON(http.StatusOK, ApiSuccess(1, "ok", gin.H{
		"currentMetaBlockHeight": currentMetaBlockHeight,
		"syncMetaBlockHeight":    syncMetaBlockHeight,
		"progressStartBlock":     progressStartBlock,
		"progressEndBlock":       progressEndBlock,
		"initBlockHeight":        initBlockHeight,
		"currentBlockHeight":     currentBlockHeight,
	}))
}

// @Summary      Get paginated NDV block list
// @Description  Retrieve paginated list of NDV (Node Data Verification) blocks
// @Tags         Statistics
// @Accept       json
// @Produce      json
// @Param        height  query  int  true   "Block height"
// @Param        cursor  query  int  false  "Pagination cursor"
// @Param        size    query  int  false  "Number of items per page"
// @Success      200  {object}  ApiResponse{data=object{info=object,total=int,list=array}}  "NDV block list with metadata"
// @Failure      400  {object}  ApiResponse  "Invalid query parameters"
// @Failure      500  {object}  ApiResponse  "Service exception"
// @Router       /statistics/host/metablock/info [get]
func blockNDVPageList(ctx *gin.Context) {
	height, err := strconv.ParseInt(ctx.Query("height"), 10, 64)
	if err != nil {
		ctx.JSON(http.StatusOK, ApiError(-1, "query height error"))
		return
	}
	cursor, err := strconv.ParseInt(ctx.Query("cursor"), 10, 64)
	if err != nil {
		ctx.JSON(http.StatusOK, ApiError(-1, "query cursor error"))
		return
	}
	size, err := strconv.ParseInt(ctx.Query("size"), 10, 64)
	if err != nil {
		ctx.JSON(http.StatusOK, ApiError(-1, "query size error"))
		return
	}
	info, list, err := getBlockNDVPageList(height, cursor, size)
	if err != nil {
		ctx.JSON(http.StatusOK, ApiError(-1, "service exception"))
		return
	}
	ctx.JSON(http.StatusOK, ApiSuccess(1, "ok", gin.H{"info": info, "total": info.Total, "list": list}))
}

// @Summary      Get paginated host value list
// @Description  Retrieve paginated list of host values with filtering options
// @Tags         Statistics
// @Accept       json
// @Produce      json
// @Param        heightBegin  query  int     false  "Starting block height filter"
// @Param        heightEnd    query  int     false  "Ending block height filter"
// @Param        timeBegin    query  int     false  "Starting timestamp filter"
// @Param        timeEnd      query  int     false  "Ending timestamp filter"
// @Param        host         query  string  false  "Host filter"
// @Param        cursor       query  int     false  "Pagination cursor"
// @Param        size         query  int     false  "Number of items per page"
// @Success      200  {object}  ApiResponse{data=object{total=int,list=array}}  "Host value list with total count"
// @Failure      400  {object}  ApiResponse  "Invalid query parameters"
// @Failure      500  {object}  ApiResponse  "Service exception"
// @Router       /statistics/metablock/host/value [get]
func hostValuePageList(ctx *gin.Context) {
	var err error
	heightBegin := int64(0)
	heightEnd := int64(0)
	timeBegin := int64(0)
	timeEnd := int64(0)
	if ctx.Query("heightBegin") != "" {
		heightBegin, err = strconv.ParseInt(ctx.Query("heightBegin"), 10, 64)
	}
	if err != nil {
		ctx.JSON(http.StatusOK, ApiError(-1, "query heightBegin error"))
		return
	}
	if ctx.Query("heightEnd") != "" {
		heightEnd, err = strconv.ParseInt(ctx.Query("heightEnd"), 10, 64)
	}
	if err != nil {
		ctx.JSON(http.StatusOK, ApiError(-1, "query heightEnd error"))
		return
	}
	if ctx.Query("timeBegin") != "" {
		timeBegin, err = strconv.ParseInt(ctx.Query("timeBegin"), 10, 64)
	}
	if err != nil {
		ctx.JSON(http.StatusOK, ApiError(-1, "query timeBegin error"))
		return
	}
	if ctx.Query("timeEnd") != "" {
		timeEnd, err = strconv.ParseInt(ctx.Query("timeEnd"), 10, 64)
	}
	if err != nil {
		ctx.JSON(http.StatusOK, ApiError(-1, "query timeEnd error"))
		return
	}
	cursor, err := strconv.ParseInt(ctx.Query("cursor"), 10, 64)
	if err != nil {
		ctx.JSON(http.StatusOK, ApiError(-1, "query cursor error"))
		return
	}
	size, err := strconv.ParseInt(ctx.Query("size"), 10, 64)
	if err != nil {
		ctx.JSON(http.StatusOK, ApiError(-1, "query size error"))
		return
	}
	list, total, err := getHostValuePageList(heightBegin, heightEnd, timeBegin, timeEnd, strings.ToLower(ctx.Query("host")), cursor, size)
	if err != nil {
		ctx.JSON(http.StatusOK, ApiError(-1, "service exception"))
		return
	}
	ctx.JSON(http.StatusOK, ApiSuccess(1, "ok", gin.H{"total": total, "list": list}))
}

// @Summary      Get paginated MDV block list
// @Description  Retrieve paginated list of MDV (Miner Data Verification) blocks
// @Tags         Statistics
// @Accept       json
// @Produce      json
// @Param        height  query  int  true   "Block height"
// @Param        cursor  query  int  false  "Pagination cursor"
// @Param        size    query  int  false  "Number of items per page"
// @Success      200  {object}  ApiResponse{data=object{info=object,total=int,list=array}}  "MDV block list with metadata"
// @Failure      400  {object}  ApiResponse  "Invalid query parameters"
// @Failure      500  {object}  ApiResponse  "Service exception"
// @Router       /statistics/metablock/address/info [get]
func blockMDVPageList(ctx *gin.Context) {
	height, err := strconv.ParseInt(ctx.Query("height"), 10, 64)
	if err != nil {
		ctx.JSON(http.StatusOK, ApiError(-1, "query height error"))
		return
	}
	cursor, err := strconv.ParseInt(ctx.Query("cursor"), 10, 64)
	if err != nil {
		ctx.JSON(http.StatusOK, ApiError(-1, "query cursor error"))
		return
	}
	size, err := strconv.ParseInt(ctx.Query("size"), 10, 64)
	if err != nil {
		ctx.JSON(http.StatusOK, ApiError(-1, "query size error"))
		return
	}
	info, list, err := getBlockMDVPageList(height, cursor, size)
	if err != nil {
		ctx.JSON(http.StatusOK, ApiError(-1, "service exception"))
		return
	}
	ctx.JSON(http.StatusOK, ApiSuccess(1, "ok", gin.H{"info": info, "total": info.Total, "list": list}))
}

// @Summary      Get paginated host address values
// @Description  Retrieve paginated list of host address values with filtering
// @Tags         Statistics
// @Accept       json
// @Produce      json
// @Param        heightBegin  query  int     false  "Starting block height filter"
// @Param        heightEnd    query  int     false  "Ending block height filter"
// @Param        timeBegin    query  int     false  "Starting timestamp filter"
// @Param        timeEnd      query  int     false  "Ending timestamp filter"
// @Param        host         query  string  false  "Host filter"
// @Param        cursor       query  int     false  "Pagination cursor"
// @Param        size         query  int     false  "Number of items per page"
// @Success      200  {object}  ApiResponse{data=object{total=int,list=array}}  "Host address values with total count"
// @Failure      400  {object}  ApiResponse  "Invalid query parameters"
// @Failure      500  {object}  ApiResponse  "Service exception"
// @Router       /statistics/metablock/host/address/list [get]
func hostAddressValuePageList(ctx *gin.Context) {
	var err error
	heightBegin := int64(0)
	heightEnd := int64(0)
	timeBegin := int64(0)
	timeEnd := int64(0)
	if ctx.Query("heightBegin") != "" {
		heightBegin, err = strconv.ParseInt(ctx.Query("heightBegin"), 10, 64)
	}
	if err != nil {
		ctx.JSON(http.StatusOK, ApiError(-1, "query heightBegin error"))
		return
	}
	if ctx.Query("heightEnd") != "" {
		heightEnd, err = strconv.ParseInt(ctx.Query("heightEnd"), 10, 64)
	}
	if err != nil {
		ctx.JSON(http.StatusOK, ApiError(-1, "query heightEnd error"))
		return
	}
	if ctx.Query("timeBegin") != "" {
		timeBegin, err = strconv.ParseInt(ctx.Query("timeBegin"), 10, 64)
	}
	if err != nil {
		ctx.JSON(http.StatusOK, ApiError(-1, "query timeBegin error"))
		return
	}
	if ctx.Query("timeEnd") != "" {
		timeEnd, err = strconv.ParseInt(ctx.Query("timeEnd"), 10, 64)
	}
	if err != nil {
		ctx.JSON(http.StatusOK, ApiError(-1, "query timeEnd error"))
		return
	}
	cursor, err := strconv.ParseInt(ctx.Query("cursor"), 10, 64)
	if err != nil {
		ctx.JSON(http.StatusOK, ApiError(-1, "query cursor error"))
		return
	}
	size, err := strconv.ParseInt(ctx.Query("size"), 10, 64)
	if err != nil {
		ctx.JSON(http.StatusOK, ApiError(-1, "query size error"))
		return
	}
	list, total, err := getHostAddressValuePageList(heightBegin, heightEnd, timeBegin, timeEnd, strings.ToLower(ctx.Query("host")), cursor, size)
	if err != nil {
		ctx.JSON(http.StatusOK, ApiError(-1, "service exception"))
		return
	}
	ctx.JSON(http.StatusOK, ApiSuccess(1, "ok", gin.H{"total": total, "list": list}))
}

// @Summary      Get host address values
// @Description  Retrieve values for specific host address with filtering
// @Tags         Statistics
// @Accept       json
// @Produce      json
// @Param        heightBegin  query  int     false  "Starting block height filter"
// @Param        heightEnd    query  int     false  "Ending block height filter"
// @Param        timeBegin    query  int     false  "Starting timestamp filter"
// @Param        timeEnd      query  int     false  "Ending timestamp filter"
// @Param        host         query  string  false  "Host filter"
// @Param        address      query  string  false  "Address filter"
// @Param        cursor       query  int     false  "Pagination cursor"
// @Param        size         query  int     false  "Number of items per page"
// @Success      200  {object}  ApiResponse{data=object{total=int,list=array}}  "Host address values with total count"
// @Failure      400  {object}  ApiResponse  "Invalid query parameters"
// @Failure      500  {object}  ApiResponse  "Service exception"
// @Router       /statistics/metablock/host/address/value [get]
func hostAddressValue(ctx *gin.Context) {
	var err error
	heightBegin := int64(0)
	heightEnd := int64(0)
	timeBegin := int64(0)
	timeEnd := int64(0)
	if ctx.Query("heightBegin") != "" {
		heightBegin, err = strconv.ParseInt(ctx.Query("heightBegin"), 10, 64)
	}
	if err != nil {
		ctx.JSON(http.StatusOK, ApiError(-1, "query heightBegin error"))
		return
	}
	if ctx.Query("heightEnd") != "" {
		heightEnd, err = strconv.ParseInt(ctx.Query("heightEnd"), 10, 64)
	}
	if err != nil {
		ctx.JSON(http.StatusOK, ApiError(-1, "query heightEnd error"))
		return
	}
	if ctx.Query("timeBegin") != "" {
		timeBegin, err = strconv.ParseInt(ctx.Query("timeBegin"), 10, 64)
	}
	if err != nil {
		ctx.JSON(http.StatusOK, ApiError(-1, "query timeBegin error"))
		return
	}
	if ctx.Query("timeEnd") != "" {
		timeEnd, err = strconv.ParseInt(ctx.Query("timeEnd"), 10, 64)
	}
	if err != nil {
		ctx.JSON(http.StatusOK, ApiError(-1, "query timeEnd error"))
		return
	}
	cursor, err := strconv.ParseInt(ctx.Query("cursor"), 10, 64)
	if err != nil {
		ctx.JSON(http.StatusOK, ApiError(-1, "query cursor error"))
		return
	}
	size, err := strconv.ParseInt(ctx.Query("size"), 10, 64)
	if err != nil {
		ctx.JSON(http.StatusOK, ApiError(-1, "query size error"))
		return
	}
	list, total, err := getHostAddressValue(heightBegin, heightEnd, timeBegin, timeEnd, strings.ToLower(ctx.Query("host")), ctx.Query("address"), cursor, size)
	if err != nil {
		ctx.JSON(http.StatusOK, ApiError(-1, "service exception"))
		return
	}
	ctx.JSON(http.StatusOK, ApiSuccess(1, "ok", gin.H{"total": total, "list": list}))
}
