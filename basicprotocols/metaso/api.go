package metaso

import (
	"fmt"
	"manindexer/database/mongodb"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

func Api(r *gin.Engine) {
	accessGroup := r.Group("/social/buzz")
	accessGroup.Use(CorsMiddleware())
	accessGroup.GET("/newest", newest)
	accessGroup.GET("/hot", hot)
	accessGroup.GET("/info", info)
	accessGroup.GET("/follow", follow)
	hostGroup := r.Group("/host")
	hostGroup.Use(CorsMiddleware())
	hostGroup.GET("/block/sync-newest", syncNewest2)
	hostGroup.GET("/block/ndv", blockNDV)
	hostGroup.GET("/block/mdv", blockMDV)
	hostGroup.GET("/info", hostInfo)
	ftGroup := r.Group("/ft")
	ftGroup.Use(CorsMiddleware())
	ftGroup.GET("/mrc20/address/deploy-list", mrc20TickList)
	settingGroup := r.Group("/metaso/settings")
	settingGroup.Use(CorsMiddleware())
	settingGroup.GET("/blocked/list", blockedList)
	settingGroup.GET("/blocked/add", blockedAdd)
	settingGroup.GET("/blocked/delete", blockedDelete)
}
func CorsMiddleware() gin.HandlerFunc {
	return func(context *gin.Context) {
		method := context.Request.Method

		context.Header("Access-Control-Allow-Origin", "*")
		context.Header("Access-Control-Allow-Credentials", "true")
		context.Header("Access-Control-Allow-Headers", "*")
		context.Header("Access-Control-Allow-Methods", "GET,HEAD,POST,PUT,DELETE,OPTIONS")
		context.Header("Access-Control-Expose-Headers", "Content-Length, Access-Control-Allow-Origin, Access-Control-Allow-Headers, Content-Type")

		if method == "OPTIONS" {
			context.AbortWithStatus(http.StatusNoContent)
		}
		context.Next()
	}
}

type ApiResponse struct {
	Code int         `json:"code"`
	Msg  string      `json:"message"`
	Data interface{} `json:"data"`
}

func ApiError(code int, msg string) (res *ApiResponse) {
	return &ApiResponse{Code: code, Msg: msg}
}
func ApiNullData(code int, msg string) (res *ApiResponse) {
	return &ApiResponse{Code: code, Msg: msg, Data: []string{}}
}
func ApiSuccess(code int, msg string, data interface{}) (res *ApiResponse) {
	return &ApiResponse{Code: code, Msg: msg, Data: data}
}

func newest(ctx *gin.Context) {
	size, err := strconv.ParseInt(ctx.Query("size"), 10, 64)
	if err != nil {
		ctx.JSON(http.StatusOK, ApiError(-1, "size error"))
		return
	}
	if size == 0 {
		size = 10
	}
	list, total, err := getNewest(ctx.Query("lastId"), size, "_id", ctx.Query("metaid"), ctx.Query("followed"))
	lastId := ""
	if len(list) > 0 {
		lastId = list[len(list)-1].MogoID.Hex()
	}
	if err != nil {
		fmt.Println(err)
		ctx.JSON(http.StatusOK, ApiError(-1, "service exception."))
		return
	}
	var newList []*TweetWithLike
	for _, item := range list {
		hostKey := fmt.Sprintf("host_%s", item.Host)
		metaidKey := fmt.Sprintf("metaid_%s", item.MetaId)
		pinidKey := fmt.Sprintf("pinid_%s", item.Id)
		if _, ok := _blockedData[hostKey]; ok {
			item.Blocked = true
		}
		if _, ok := _blockedData[metaidKey]; ok {
			item.Blocked = true
		}
		if _, ok := _blockedData[pinidKey]; ok {
			item.Blocked = true
		}
		newList = append(newList, item)
	}

	ctx.JSON(http.StatusOK, ApiSuccess(1, "ok", gin.H{"list": newList, "total": total, "lastId": lastId}))
}
func hot(ctx *gin.Context) {
	size, err := strconv.ParseInt(ctx.Query("size"), 10, 64)
	if err != nil {
		ctx.JSON(http.StatusOK, ApiError(-1, "size error"))
		return
	}
	if size == 0 {
		size = 10
	}
	list, total, err := getNewest(ctx.Query("lastId"), size, "hot", "", "")
	lastId := ""
	if len(list) > 0 {
		lastId = list[len(list)-1].MogoID.Hex()
	}
	if err != nil {
		ctx.JSON(http.StatusOK, ApiError(-1, "service exception."))
		return
	}
	ctx.JSON(http.StatusOK, ApiSuccess(1, "ok", gin.H{"list": list, "total": total, "lastId": lastId}))
}
func info(ctx *gin.Context) {
	tweet, comments, like, donates, err := getInfo(ctx.Query("pinId"))
	if err != nil {
		ctx.JSON(http.StatusOK, ApiError(-1, "service exception"))
		return
	}
	blocked := false
	pinidKey := fmt.Sprintf("pinid_%s", tweet.Id)

	if _, ok := _blockedData[pinidKey]; ok {
		blocked = true
	}

	ctx.JSON(http.StatusOK, ApiSuccess(1, "ok", gin.H{"tweet": tweet, "comments": comments, "like": like, "donates": donates, "blocked": blocked}))
}

type followItem struct {
	Metaid   string `json:"metaid"`
	Mempool  int    `json:"mempool"`
	Unfollow int    `json:"unfollow"`
}

func follow(ctx *gin.Context) {
	if ctx.Query("metaid") == "" {
		ctx.JSON(http.StatusOK, ApiError(-1, "metaid id null"))
		return
	}
	mg := &mongodb.Mongodb{}
	list, _, err := mg.GetFollowDataByMetaId(ctx.Query("metaid"), true, false, int64(0), int64(10000))
	if err != nil {
		ctx.JSON(http.StatusOK, ApiError(-1, "service exception"))
		return
	}
	var ret []*followItem
	for _, metaid := range list {
		ret = append(ret, &followItem{Metaid: metaid.(string)})
	}
	mempoolList, err := getMempoolFollow(ctx.Query("metaid"))
	if err == nil {
		for _, metaid := range mempoolList {
			ret = append(ret, &followItem{Metaid: *metaid, Mempool: 1})
		}
	}
	ctx.JSON(http.StatusOK, ApiSuccess(1, "ok", gin.H{"list": ret}))
}
func syncNewest(ctx *gin.Context) {
	_, height := getSyncHeight()
	ctx.JSON(http.StatusOK, ApiSuccess(1, "ok", height))
}
func syncNewest2(ctx *gin.Context) {
	height, _ := mongodb.GetSyncLastNumber("metablock")
	ctx.JSON(http.StatusOK, ApiSuccess(1, "ok", height))
}
func blockInfo(ctx *gin.Context) {
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

	list, err := getBlockInfo(height, "", cursor, size, ctx.Query("orderby"))
	if err != nil {
		ctx.JSON(http.StatusOK, ApiError(-1, "service exception"))
		return
	}
	ctx.JSON(http.StatusOK, ApiSuccess(1, "ok", list))
}
func blockNDV(ctx *gin.Context) {
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

	list, err := getBlockNDV(height, ctx.Query("host"), cursor, size, ctx.Query("orderby"))
	if err != nil {
		ctx.JSON(http.StatusOK, ApiError(-1, "service exception"))
		return
	}
	ctx.JSON(http.StatusOK, ApiSuccess(1, "ok", list))
}
func blockMDV(ctx *gin.Context) {
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

	list, err := getBlockMDV(height, ctx.Query("address"), cursor, size, ctx.Query("orderby"))
	if err != nil {
		ctx.JSON(http.StatusOK, ApiError(-1, "service exception"))
		return
	}
	ctx.JSON(http.StatusOK, ApiSuccess(1, "ok", list))
}
func ndvPageList(ctx *gin.Context) {
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

	list, err := getNdvPageList(ctx.Query("host"), cursor, size, ctx.Query("orderby"))
	if err != nil {
		ctx.JSON(http.StatusOK, ApiError(-1, "service exception"))
		return
	}
	ctx.JSON(http.StatusOK, ApiSuccess(1, "ok", list))
}
func mdvPageList(ctx *gin.Context) {
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

	list, err := getMdvPageList(ctx.Query("address"), cursor, size, ctx.Query("orderby"))
	if err != nil {
		ctx.JSON(http.StatusOK, ApiError(-1, "service exception"))
		return
	}
	ctx.JSON(http.StatusOK, ApiSuccess(1, "ok", list))
}
func hostInfo(ctx *gin.Context) {
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
	list, err := getBlockInfo(0, ctx.Query("host"), cursor, size, ctx.Query("orderby"))
	if err != nil {
		ctx.JSON(http.StatusOK, ApiError(-1, "service exception"))
		return
	}
	ctx.JSON(http.StatusOK, ApiSuccess(1, "ok", list))
}
func mrc20TickList(ctx *gin.Context) {
	address := ctx.Query("address")
	if address == "" {
		ctx.JSON(http.StatusOK, ApiError(-1, "address is null"))
		return
	}
	list, err := getTickByAddress(address, ctx.Query("tickType"))
	if err != nil {
		ctx.JSON(http.StatusOK, ApiError(-1, "service exception"))
		return
	}
	ctx.JSON(http.StatusOK, ApiSuccess(1, "ok", list))
}

func blockedList(ctx *gin.Context) {
	blockType := ctx.Query("blockType")
	if blockType == "" {
		ctx.JSON(http.StatusOK, ApiError(-1, "blockType is null"))
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
	list, total, err := getBlockedList(blockType, cursor, size)
	if err != nil {
		ctx.JSON(http.StatusOK, ApiError(-1, "service exception"))
		return
	}
	ctx.JSON(http.StatusOK, ApiSuccess(1, "ok", gin.H{"list": list, "total": total}))
}
func blockedAdd(ctx *gin.Context) {
	blockType := ctx.Query("blockType")
	if blockType == "" {
		ctx.JSON(http.StatusOK, ApiError(-1, "blockType is null"))
		return
	}
	blockContent := ctx.Query("blockContent")
	if blockContent == "" {
		ctx.JSON(http.StatusOK, ApiError(-1, "blockContent is null"))
		return
	}
	err := addBlockedList(blockType, blockContent)
	if err != nil {
		ctx.JSON(http.StatusOK, ApiError(-1, "service exception"))
		return
	}
	ctx.JSON(http.StatusOK, ApiSuccess(1, "ok", nil))
}
func blockedDelete(ctx *gin.Context) {
	blockType := ctx.Query("blockType")
	if blockType == "" {
		ctx.JSON(http.StatusOK, ApiError(-1, "blockType is null"))
		return
	}
	blockContent := ctx.Query("blockContent")
	if blockContent == "" {
		ctx.JSON(http.StatusOK, ApiError(-1, "blockContent is null"))
		return
	}
	err := deleteBlockedList(blockType, blockContent)
	if err != nil {
		ctx.JSON(http.StatusOK, ApiError(-1, "service exception"))
		return
	}
	ctx.JSON(http.StatusOK, ApiSuccess(1, "ok", nil))
}
