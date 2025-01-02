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
	group.GET("/host/block/sync-newest", blockSyncNewest)
}
func blockSyncNewest(ctx *gin.Context) {
	metaBlock, _ := mongodb.GetSyncLastNumber("metablock")
	metaBlockData := getMetaBlock(metaBlock)
	firtBlockData := getMetaBlock(0)
	btc := bitcoin.BitcoinChain{}
	currentHeight := btc.GetBestHeight()
	syncHeight := int64(0)
	initHegiht := int64(0)
	for _, chain := range metaBlockData.Chains {
		if chain.Chain == "Bitcoin" {
			syncHeight, _ = strconv.ParseInt(chain.EndBlock, 10, 64)
			break
		}
	}
	for _, chain := range firtBlockData.Chains {
		if chain.Chain == "Bitcoin" {
			initHegiht, _ = strconv.ParseInt(chain.EndBlock, 10, 64)
			break
		}
	}
	ctx.JSON(http.StatusOK, ApiSuccess(1, "ok", gin.H{"currentHeight": currentHeight, "syncHeight": syncHeight, "initHeight": initHegiht}))
}
