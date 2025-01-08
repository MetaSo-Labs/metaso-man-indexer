package metaso

import (
	"manindexer/adapter/bitcoin"
	"manindexer/database/mongodb"
	"net/http"
	"strconv"

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
func blockSyncNewest(ctx *gin.Context) {
	currentMetaBlockHeight := int64(0)
	syncMetaBlockHeight := int64(0)
	progressStartBlock := int64(0)
	progressEndBlock := int64(0)
	initBlockHeight := int64(0)

	lastBlockInfo := getLastMetaBlock()
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
	list, total, err := getHostValuePageList(heightBegin, heightEnd, timeBegin, timeEnd, ctx.Query("host"), cursor, size)
	if err != nil {
		ctx.JSON(http.StatusOK, ApiError(-1, "service exception"))
		return
	}
	ctx.JSON(http.StatusOK, ApiSuccess(1, "ok", gin.H{"total": total, "list": list}))
}
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
	list, total, err := getHostAddressValuePageList(heightBegin, heightEnd, timeBegin, timeEnd, ctx.Query("host"), cursor, size)
	if err != nil {
		ctx.JSON(http.StatusOK, ApiError(-1, "service exception"))
		return
	}
	ctx.JSON(http.StatusOK, ApiSuccess(1, "ok", gin.H{"total": total, "list": list}))
}
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
	list, total, err := getHostAddressValue(heightBegin, heightEnd, timeBegin, timeEnd, ctx.Query("host"), ctx.Query("address"), cursor, size)
	if err != nil {
		ctx.JSON(http.StatusOK, ApiError(-1, "service exception"))
		return
	}
	ctx.JSON(http.StatusOK, ApiSuccess(1, "ok", gin.H{"total": total, "list": list}))
}
