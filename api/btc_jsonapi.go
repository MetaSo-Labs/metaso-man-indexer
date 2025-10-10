package api

import (
	"encoding/json"
	"fmt"
	"manindexer/api/respond"
	"manindexer/common"
	"manindexer/database"
	"manindexer/database/mongodb"
	"manindexer/man"
	"manindexer/pin"
	"math/rand"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/mongo"
)

type ApiResponse struct {
	Code int         `json:"code"`
	Msg  string      `json:"message"`
	Data interface{} `json:"data"`
}

func btcJsonApi(r *gin.Engine) {
	btcGroup := r.Group("/api")
	btcGroup.Use(CorsMiddleware())
	btcGroup.GET("/metaid/list", metaidList)
	btcGroup.GET("/metaid/list/limit", metaidListLimit)
	btcGroup.GET("/pin/list", pinList)
	btcGroup.POST("/pin/check", pinCheck)
	btcGroup.GET("/block/list", blockList)
	btcGroup.GET("/mempool/list", mempoolList)
	btcGroup.GET("/node/list", nodeList)
	btcGroup.GET("/reindex/:chain/:from/:to", reindex)
	btcGroup.GET("/notifcation/list", notifcationList)
	btcGroup.GET("/dict/set", dictSet)
	btcGroup.GET("/dict/get", dictGet)
	btcGroup.GET("/block/file", blockFileGet)
	btcGroup.GET("/block/file/partCount", blockPartCount)
	btcGroup.GET("/block/file/create", blockFileCreate)
	btcGroup.GET("/block/id/list", blockIdList)
	btcGroup.GET("/block/id/create", setPinIdList)

	btcGroup.GET("/pin/:numberOrId", getPinById)
	btcGroup.GET("/address/pin/utxo/count/:address", getPinUtxoCountByAddress)
	btcGroup.GET("/address/pin/list/:addressType/:address", getPinListByAddress)
	btcGroup.GET("/node/child/:pinId", getChildNodeById)
	btcGroup.GET("/node/parent/:pinId", getParentNodeById)
	btcGroup.GET("/info/address/:address", getInfoByAddress)
	btcGroup.GET("/info/metaid/:metaId", getInfoByMetaId)
	btcGroup.GET("/info/search", infoSearch)
	btcGroup.GET("/info/metaidUpdate", infometaidUpdate)
	btcGroup.GET("/getAllPinByPath", getAllPinByPath)
	btcGroup.POST("/generalQuery", generalQuery)
	btcGroup.GET("/pin/ByOutput/:output", getPinByOutput)
	btcGroup.GET("/follow/record", getFollowRecord)
	btcGroup.GET("/metaid/followerList/:metaid", getFollowerListByMetaId)
	btcGroup.GET("/metaid/followingList/:metaid", getFollowingListByMetaId)
	btcGroup.GET("/metaid/recommended", getRecommendedList)
	btcGroup.POST("/getAllPinByPathAndMetaId", getAllPinByPathAndMetaId)
	btcGroup.POST("/metaid/dataValue", getDataValueByMetaIdList)
}

func metaidList(ctx *gin.Context) {
	page, err := strconv.ParseInt(ctx.Query("page"), 10, 64)
	if err != nil {
		ctx.JSON(http.StatusOK, respond.ErrParameterError)
		return
	}
	size, err := strconv.ParseInt(ctx.Query("size"), 10, 64)
	if err != nil {
		ctx.JSON(http.StatusOK, respond.ErrParameterError)
		return
	}
	order := ctx.Query("order")
	list, err := man.DbAdapter.GetMetaIdPageList(page, size, order)
	if err != nil || list == nil {
		if err == mongo.ErrNoDocuments {
			ctx.JSON(http.StatusOK, respond.ErrNoDataFound)
		} else {
			ctx.JSON(http.StatusOK, respond.ErrServiceError)
		}
		return
	}
	count := man.DbAdapter.Count()
	ctx.JSON(http.StatusOK, respond.ApiSuccess(1, "ok", gin.H{"list": list, "count": &count}))
}
func metaidListLimit(ctx *gin.Context) {
	lastupdate := ctx.Query("lastupdate")
	if lastupdate == "" {
		ctx.JSON(http.StatusOK, respond.ErrParameterError)
		return
	}
	limit := ctx.Query("limit")
	if limit == "" {
		limit = "100"
	}
	limitInt, err := strconv.Atoi(limit)
	if err != nil {
		ctx.JSON(http.StatusOK, respond.ErrParameterError)
		return
	}
	if limitInt > 2000 {
		limitInt = 2000
	}
	lastupdateInt, err := strconv.ParseInt(lastupdate, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusOK, respond.ErrParameterError)
		return
	}
	infoList, err := mongodb.BatchGetMetaIdInfo(lastupdateInt, limitInt)
	if err != nil {
		ctx.JSON(http.StatusOK, respond.ErrServiceError)
		return
	}
	ctx.JSON(http.StatusOK, respond.ApiSuccess(1, "ok", gin.H{"list": infoList}))
}
func pinList(ctx *gin.Context) {
	//page, err := strconv.ParseInt(ctx.Query("page"), 10, 64)
	page, err := strconv.Atoi(ctx.Query("page"))
	if err != nil {
		ctx.JSON(http.StatusOK, respond.ErrParameterError)
		return
	}
	//size, err := strconv.ParseInt(ctx.Query("size"), 10, 64)
	size, err := strconv.Atoi(ctx.Query("size"))
	if err != nil {
		ctx.JSON(http.StatusOK, respond.ErrParameterError)
		return
	}
	//list, err := man.DbAdapter.GetPinPageList(page, size)
	list, lastId, err := man.PebbleStore.PinPageList(page-1, size, ctx.Query("lastId"))
	if err != nil || list == nil {
		if err == mongo.ErrNoDocuments {
			ctx.JSON(http.StatusOK, respond.ErrNoDataFound)
		} else {
			ctx.JSON(http.StatusOK, respond.ErrServiceError)
		}
		return
	}
	var msg []*pin.PinMsg
	for _, p := range list {
		pmsg := &pin.PinMsg{
			Content: p.ContentSummary, Number: p.Number, Operation: p.Operation,
			Id: p.Id, Type: p.ContentTypeDetect, Path: p.Path, MetaId: p.MetaId,
			Pop: p.Pop, ChainName: p.ChainName,
			InitialOwner: p.InitialOwner, Address: p.Address, CreateAddress: p.CreateAddress,
			Timestamp: p.Timestamp,
		}
		msg = append(msg, pmsg)
	}
	//count := man.DbAdapter.Count()
	count := man.PebbleStore.GetAllCount()
	ctx.JSON(http.StatusOK, respond.ApiSuccess(1, "ok", gin.H{"Pins": msg, "Count": &count, "Active": "index", "LastId": lastId}))
}
func mempoolList(ctx *gin.Context) {
	page, err := strconv.ParseInt(ctx.Query("page"), 10, 64)
	if err != nil {
		ctx.JSON(http.StatusOK, respond.ErrParameterError)
		return
	}
	size, err := strconv.ParseInt(ctx.Query("size"), 10, 64)
	if err != nil {
		ctx.JSON(http.StatusOK, respond.ErrParameterError)
		return
	}
	list, err := man.DbAdapter.GetMempoolPinPageList(page, size)
	if err != nil || list == nil {
		if err == mongo.ErrNoDocuments || list == nil {
			ctx.JSON(http.StatusOK, respond.ErrNoDataFound)
		} else {
			ctx.JSON(http.StatusOK, respond.ErrServiceError)
		}
		return
	}
	var msg []*pin.PinMsg
	for _, p := range list {
		pmsg := &pin.PinMsg{Content: p.ContentSummary, Number: p.Number, Operation: p.Operation, Id: p.Id, Type: p.ContentTypeDetect, Path: p.Path, MetaId: p.MetaId}
		msg = append(msg, pmsg)
	}
	count := man.DbAdapter.Count()
	ctx.JSON(http.StatusOK, respond.ApiSuccess(1, "ok", gin.H{"Pins": msg, "Count": &count, "Active": "mempool"}))
}
func nodeList(ctx *gin.Context) {
	page, err := strconv.ParseInt(ctx.Query("page"), 10, 64)
	if err != nil {
		ctx.JSON(http.StatusOK, respond.ErrParameterError)
		return
	}
	size, err := strconv.ParseInt(ctx.Query("size"), 10, 64)
	if err != nil {
		ctx.JSON(http.StatusOK, respond.ErrParameterError)
		return
	}
	rootid := ctx.Query("rootid")
	list, total, err := man.DbAdapter.GetMetaIdPin(rootid, page, size)
	if err != nil || list == nil {
		if err == mongo.ErrNoDocuments {
			ctx.JSON(http.StatusOK, respond.ErrNoDataFound)
		} else {
			ctx.JSON(http.StatusOK, respond.ErrServiceError)
		}
		return
	}
	ctx.JSON(http.StatusOK, respond.ApiSuccess(1, "ok", gin.H{"RootId": rootid, "Total": total, "Pins": list}))
}

// get pin by id
func getPinById(ctx *gin.Context) {
	pinMsg, err := man.DbAdapter.GetPinByNumberOrId(ctx.Param("numberOrId"))
	if err != nil || pinMsg == nil {
		if err == mongo.ErrNoDocuments {
			ctx.JSON(http.StatusOK, respond.ErrNoPinFound)
		} else {
			ctx.JSON(http.StatusOK, respond.ErrServiceError)
		}
		return
	}
	//pinMsg.ContentBody = []byte{}
	pinMsg.ContentSummary = string(pinMsg.ContentBody)
	pinMsg.PopLv, _ = pin.PopLevelCount(pinMsg.ChainName, pinMsg.Pop)
	pinMsg.Preview = common.Config.Web.Host + "/pin/" + pinMsg.Id
	pinMsg.Content = common.Config.Web.Host + "/content/" + pinMsg.Id
	check, err := man.DbAdapter.GetMempoolTransferById(pinMsg.Id)
	if err == nil && check != nil {
		pinMsg.Status = -9
	}
	ctx.JSON(http.StatusOK, respond.ApiSuccess(1, "ok", pinMsg))
}
func getPinByOutput(ctx *gin.Context) {
	pinMsg, err := man.DbAdapter.GetPinByOutput(ctx.Param("output"))
	if err != nil || pinMsg == nil {
		if err == mongo.ErrNoDocuments {
			ctx.JSON(http.StatusOK, respond.ErrNoPinFound)
		} else {
			ctx.JSON(http.StatusOK, respond.ErrServiceError)
		}
		return
	}
	//pinMsg.ContentBody = []byte{}
	pinMsg.ContentSummary = string(pinMsg.ContentBody)
	pinMsg.Preview = common.Config.Web.Host + "/pin/" + pinMsg.Id
	pinMsg.Content = common.Config.Web.Host + "/content/" + pinMsg.Id
	pinMsg.PopLv, _ = pin.PopLevelCount(pinMsg.ChainName, pinMsg.Pop)
	ctx.JSON(http.StatusOK, respond.ApiSuccess(1, "ok", pinMsg))
}

func blockList(ctx *gin.Context) {
	page, err := strconv.ParseInt(ctx.Query("page"), 10, 64)
	if err != nil {
		ctx.JSON(http.StatusOK, respond.ErrParameterError)
		return
	}
	size, err := strconv.ParseInt(ctx.Query("size"), 10, 64)
	if err != nil {
		ctx.JSON(http.StatusOK, respond.ErrParameterError)
		return
	}
	list, err := man.DbAdapter.GetPinPageList(page, size)
	if err != nil || list == nil {
		if err == mongo.ErrNoDocuments {
			ctx.JSON(http.StatusOK, respond.ErrNoDataFound)
		} else {
			ctx.JSON(http.StatusOK, respond.ErrServiceError)
		}
		return
	}
	msgMap := make(map[int64][]*pin.PinMsg)
	var msgList []int64
	for _, p := range list {
		pmsg := &pin.PinMsg{Operation: p.Operation, Path: p.Path, Content: p.ContentSummary, Number: p.Number, Id: p.Id, Type: p.ContentTypeDetect, MetaId: p.MetaId, Height: p.GenesisHeight, Pop: p.Pop}
		if _, ok := msgMap[pmsg.Height]; ok {
			msgMap[pmsg.Height] = append(msgMap[pmsg.Height], pmsg)
		} else {
			msgMap[pmsg.Height] = []*pin.PinMsg{pmsg}
			msgList = append(msgList, pmsg.Height)
		}
	}
	ctx.JSON(http.StatusOK, respond.ApiSuccess(1, "ok", gin.H{"msgMap": msgMap, "msgList": msgList, "Active": "blocks"}))
}

// get Pin Utxo Count By Address
func getPinUtxoCountByAddress(ctx *gin.Context) {
	if ctx.Param("address") == "" {
		ctx.JSON(http.StatusOK, respond.ErrAddressIsEmpty)
	}
	utxoNum, utxoSum, err := man.DbAdapter.GetPinUtxoCountByAddress(ctx.Param("address"))
	if err != nil {
		if err == mongo.ErrNoDocuments {
			ctx.JSON(http.StatusOK, respond.ErrNoDataFound)
		} else {
			ctx.JSON(http.StatusOK, respond.ErrServiceError)
		}
		return
	}
	ctx.JSON(http.StatusOK, respond.ApiSuccess(1, "ok", gin.H{"utxoNum": utxoNum, "utxoSum": utxoSum}))
}

// get pin list by address
func getPinListByAddress(ctx *gin.Context) {
	cursorStr := ctx.Query("cursor")
	sizeStr := ctx.Query("size")
	cnt := ctx.Query("cnt")
	path := ctx.Query("path")
	cursor := int64(0)
	size := int64(10000)
	if cursorStr != "" && sizeStr != "" {
		cursor, _ = strconv.ParseInt(cursorStr, 10, 64)
		size, _ = strconv.ParseInt(sizeStr, 10, 64)
	}
	pinList, total, err := man.DbAdapter.GetPinListByAddress(ctx.Param("address"), ctx.Param("addressType"), cursor, size, cnt, path)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			ctx.JSON(http.StatusOK, respond.ErrNoPinFound)
		} else {
			ctx.JSON(http.StatusOK, respond.ErrServiceError)
		}
		return
	}
	//get mempool transfer pin
	memTransRecive := make(map[string]struct{})
	memTransSend := make(map[string]struct{})
	mempoolTransferList, err := man.DbAdapter.GetMempoolTransfer(ctx.Param("address"), "")
	if err == nil {
		for _, transfer := range mempoolTransferList {
			if transfer.FromAddress == ctx.Param("address") {
				memTransSend[transfer.PinId] = struct{}{}
			} else if transfer.ToAddress == ctx.Param("address") {
				memTransRecive[transfer.PinId] = struct{}{}
			}
		}
		total -= int64(len(memTransSend))
	}
	var result []*pin.PinInscription
	if cursor == 0 && len(memTransRecive) > 0 {
		var idList []string
		for k := range memTransRecive {
			idList = append(idList, k)
		}
		list, err := man.DbAdapter.GetPinListByIdList(idList)
		if err == nil && len(list) > 0 {
			for _, p := range list {
				p.Status = -9
				p.ContentBody = []byte{}
				p.Preview = common.Config.Web.Host + "/pin/" + p.Id
				p.Content = common.Config.Web.Host + "/content/" + p.Id
				p.PopLv, _ = pin.PopLevelCount(p.ChainName, p.Pop)
				result = append(result, p)
			}
		}
		total += int64(len(list))
	}
	var fixPinList []*pin.PinInscription
	for _, pinNode := range pinList {
		_, ok := memTransSend[pinNode.Id]
		if ok {
			continue
		}
		pinNode.ContentSummary = string(pinNode.ContentBody)
		//pinNode.ContentBody = []byte{}
		pinNode.Preview = common.Config.Web.Host + "/pin/" + pinNode.Id
		pinNode.Content = common.Config.Web.Host + "/content/" + pinNode.Id
		pinNode.PopLv, _ = pin.PopLevelCount(pinNode.ChainName, pinNode.Pop)
		fixPinList = append(fixPinList, pinNode)
	}
	result = append(result, fixPinList...)
	if cnt == "true" {
		ctx.JSON(http.StatusOK, respond.ApiSuccess(1, "ok", gin.H{"list": result, "total": total}))
	} else {
		ctx.JSON(http.StatusOK, respond.ApiSuccess(1, "ok", result))
	}

}

// get child node by id
func getChildNodeById(ctx *gin.Context) {
	pinList, err := man.DbAdapter.GetChildNodeById(ctx.Param("pinId"))
	if err != nil {
		if err == mongo.ErrNoDocuments {
			ctx.JSON(http.StatusOK, respond.ErrNoChildFound)
		} else {
			ctx.JSON(http.StatusOK, respond.ErrServiceError)
		}
		return
	}
	for _, pin := range pinList {
		pin.ContentBody = []byte{}
	}
	ctx.JSON(http.StatusOK, respond.ApiSuccess(1, "ok", pinList))
}

// get parent node by id
func getParentNodeById(ctx *gin.Context) {
	pinMsg, err := man.DbAdapter.GetParentNodeById(ctx.Param("pinId"))
	if err != nil || pinMsg == nil {
		if err == mongo.ErrNoDocuments {
			ctx.JSON(http.StatusOK, respond.ErrNoNodeFound)
		} else {
			ctx.JSON(http.StatusOK, respond.ErrServiceError)
		}
		return
	}
	pinMsg.ContentBody = []byte{}
	ctx.JSON(http.StatusOK, respond.ApiSuccess(1, "ok", pinMsg))
}

type metaInfo struct {
	*pin.MetaIdInfo
	Unconfirmed string `json:"unconfirmed"`
	Blocked     bool   `json:"blocked"`
}

func getInfoByAddress(ctx *gin.Context) {
	if common.Config.CacheUrl != "" && ctx.Query("cache") == "" {
		getCacheInfoByAddress(ctx)
		return
	}
	metaid, unconfirmed, err := man.DbAdapter.GetMetaIdInfo(ctx.Param("address"), true, "")
	if err != nil {
		ctx.JSON(http.StatusOK, respond.ErrServiceError)
		return
	}
	if metaid == nil {
		metaid = &pin.MetaIdInfo{MetaId: common.GetMetaIdByAddress(ctx.Param("address")), Address: ctx.Param("address")}
		//ctx.JSON(200, apiError(100, "no metaid found."))
		ctx.JSON(http.StatusOK, respond.ApiSuccess(1, "ok", metaInfo{metaid, "", false}))
		return
	}
	if metaid.Address == "" {
		metaid.Address = ctx.Param("address")
	}

	metaid.MetaId = common.GetMetaIdByAddress(ctx.Param("address"))
	metaidKey := fmt.Sprintf("metaid_%s", metaid.MetaId)
	blocked := false
	if _, ok := common.BlockedData[metaidKey]; ok {
		blocked = true
	}
	ctx.JSON(http.StatusOK, respond.ApiSuccess(1, "ok", metaInfo{metaid, unconfirmed, blocked}))
}
func getCacheInfoByAddress(ctx *gin.Context) {
	address := ctx.Param("address")
	// 拼接目标 URL
	targetURL, err := url.Parse(common.Config.CacheUrl)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid target URL"})
		return
	}
	// 创建反向代理
	proxy := httputil.NewSingleHostReverseProxy(targetURL)

	// 修改请求路径
	ctx.Request.URL.Path = "/v1/users/" + address
	ctx.Request.Host = targetURL.Host

	// 使用代理处理请求
	proxy.ServeHTTP(ctx.Writer, ctx.Request)
}
func infometaidUpdate(ctx *gin.Context) {
	err := mongodb.UpdateAllMetaId()
	ctx.JSON(http.StatusOK, respond.ApiSuccess(1, "ok", err))
}
func infoSearch(ctx *gin.Context) {
	keyword := ctx.Query("keyword")
	if keyword == "" {
		ctx.JSON(http.StatusOK, respond.ErrParameterError)
		return
	}
	keytype := ctx.Query("keytype")
	if keytype == "" {
		ctx.JSON(http.StatusOK, respond.ErrParameterError)
		return
	}
	// 拼接目标 URL
	targetURL, err := url.Parse(common.Config.CacheUrl)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid target URL"})
		return
	}
	// 创建反向代理
	proxy := httputil.NewSingleHostReverseProxy(targetURL)

	// 修改请求路径
	ctx.Request.URL.Path = "/v1/search"
	ctx.Request.Host = targetURL.Host

	// 使用代理处理请求
	proxy.ServeHTTP(ctx.Writer, ctx.Request)
}
func getCacheInfoByMetaid(ctx *gin.Context) {
	metaid := ctx.Param("metaId")
	// 拼接目标 URL
	targetURL, err := url.Parse(common.Config.CacheUrl)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid target URL"})
		return
	}
	// 创建反向代理
	proxy := httputil.NewSingleHostReverseProxy(targetURL)

	// 修改请求路径
	ctx.Request.URL.Path = "/v1/users/info/metaid/" + metaid
	ctx.Request.Host = targetURL.Host

	// 使用代理处理请求
	proxy.ServeHTTP(ctx.Writer, ctx.Request)
}
func getInfoByMetaId(ctx *gin.Context) {
	if common.Config.CacheUrl != "" && ctx.Query("cache") == "" {
		getCacheInfoByMetaid(ctx)
		return
	}
	metaid, unconfirmed, err := man.DbAdapter.GetMetaIdInfo("", true, ctx.Param("metaId"))
	if err != nil {
		ctx.JSON(http.StatusOK, respond.ErrServiceError)
		return
	}
	if metaid == nil {
		metaid = &pin.MetaIdInfo{MetaId: ctx.Param("metaId"), Address: ""}
		//ctx.JSON(200, apiError(100, "no metaid found."))
		ctx.JSON(http.StatusOK, respond.ApiSuccess(1, "ok", metaInfo{metaid, "", false}))
		return
	}

	if metaid.MetaId == "" {
		metaid.MetaId = ctx.Param("metaId")
	}
	metaidKey := fmt.Sprintf("metaid_%s", metaid.MetaId)
	blocked := false
	if _, ok := common.BlockedData[metaidKey]; ok {
		blocked = true
	}
	ctx.JSON(http.StatusOK, respond.ApiSuccess(1, "ok", metaInfo{metaid, unconfirmed, blocked}))
}
func generalQuery(ctx *gin.Context) {
	var g database.Generator
	if err := ctx.BindJSON(&g); err != nil {
		ctx.JSON(http.StatusOK, respond.ErrParameterError)
		return
	}
	ret, err := man.DbAdapter.GeneratorFind(g)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			ctx.JSON(http.StatusOK, respond.ErrNoResultFound)
		} else {
			ctx.JSON(http.StatusOK, respond.ErrServiceError)
		}
		return
	}
	ctx.JSON(http.StatusOK, respond.ApiSuccess(1, "ok", ret))
}
func getAllPinByPath(ctx *gin.Context) {
	page, err := strconv.ParseInt(ctx.Query("page"), 10, 64)
	if err != nil {
		ctx.JSON(http.StatusOK, respond.ApiError(101, "page parameter error"))
		return
	}
	limit, err := strconv.ParseInt(ctx.Query("limit"), 10, 64)
	if err != nil {
		ctx.JSON(http.StatusOK, respond.ApiError(101, "limit parameter error"))
		return
	}
	if ctx.Query("path") == "" {
		ctx.JSON(http.StatusOK, respond.ApiError(101, "parentPath parameter error"))
		return
	}
	pinList1, total, err := man.DbAdapter.GetAllPinByPath(page, limit, ctx.Query("path"), []string{})
	if err != nil {
		if err == mongo.ErrNoDocuments {
			ctx.JSON(http.StatusOK, respond.ErrNoPinFound)
		} else {
			ctx.JSON(http.StatusOK, respond.ErrServiceError)
		}
		return
	}
	var pinList []*pin.PinInscription
	for _, pinNode := range pinList1 {
		pinNode.ContentSummary = string(pinNode.ContentBody)
		pinList = append(pinList, pinNode)
	}
	ctx.JSON(http.StatusOK, respond.ApiSuccess(1, "ok", gin.H{"list": pinList, "total": total}))
}

// getAllPinByPathAndMetaId
type pinQuery struct {
	Page       int64    `json:"page"`
	Size       int64    `json:"size"`
	Path       string   `json:"path"`
	MetaIdList []string `json:"metaIdList"`
}

func getAllPinByPathAndMetaId(ctx *gin.Context) {
	var q pinQuery
	if err := ctx.BindJSON(&q); err != nil {
		ctx.JSON(http.StatusOK, respond.ErrParameterError)
		return
	}
	pinList1, total, err := man.DbAdapter.GetAllPinByPath(q.Page, q.Size, q.Path, q.MetaIdList)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			ctx.JSON(http.StatusOK, respond.ErrNoPinFound)
		} else {
			ctx.JSON(http.StatusOK, respond.ErrServiceError)
		}
		return
	}
	var pinList []*pin.PinInscription
	for _, pinNode := range pinList1 {
		pinNode.ContentSummary = string(pinNode.ContentBody)
		pinList = append(pinList, pinNode)
	}
	ctx.JSON(http.StatusOK, respond.ApiSuccess(1, "ok", gin.H{"list": pinList, "total": total}))
}

// getDataValueByMetaIdList
type stringListQuery struct {
	List []string `json:"list"`
}

func getDataValueByMetaIdList(ctx *gin.Context) {
	var q stringListQuery
	if err := ctx.BindJSON(&q); err != nil {
		ctx.JSON(http.StatusOK, respond.ErrParameterError)
		return
	}
	result, err := man.DbAdapter.GetDataValueByMetaIdList(q.List)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			ctx.JSON(http.StatusOK, respond.ErrNoResultFound)
		} else {
			ctx.JSON(http.StatusOK, respond.ErrServiceError)
		}
		return
	}
	ctx.JSON(http.StatusOK, respond.ApiSuccess(1, "ok", result))

}

type pinChecktQuery struct {
	PinList []string `json:"pinList"`
}

// pinCheck
func pinCheck(ctx *gin.Context) {
	var q pinChecktQuery
	if err := ctx.BindJSON(&q); err != nil {
		ctx.JSON(http.StatusOK, respond.ErrParameterError)
		return
	}
	result, err := man.DbAdapter.GetPinCheckListByIdList(q.PinList)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			ctx.JSON(http.StatusOK, respond.ErrNoResultFound)
		} else {
			ctx.JSON(http.StatusOK, respond.ErrServiceError)
		}
		return
	}
	ctx.JSON(http.StatusOK, respond.ApiSuccess(1, "ok", result))

}

// getFollowListByMetaId
func getFollowerListByMetaId(ctx *gin.Context) {
	cursorStr := ctx.Query("cursor")
	sizeStr := ctx.Query("size")
	cursor := int64(0)
	size := int64(100)
	myFollow := false
	followDetail := false
	if ctx.Query("followDetail") == "true" {
		followDetail = true
	}
	if cursorStr != "" && sizeStr != "" {
		cursor, _ = strconv.ParseInt(cursorStr, 10, 64)
		size, _ = strconv.ParseInt(sizeStr, 10, 64)
	}
	list, total, err := man.DbAdapter.GetFollowDataByMetaId(ctx.Param("metaid"), myFollow, followDetail, cursor, size)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			ctx.JSON(http.StatusOK, respond.ErrNoResultFound)
		} else {
			ctx.JSON(http.StatusOK, respond.ErrServiceError)
		}
		return
	}
	ctx.JSON(http.StatusOK, respond.ApiSuccess(1, "ok", gin.H{"list": list, "total": total}))

}

// getFollowListByMetaId
func getFollowingListByMetaId(ctx *gin.Context) {
	cursorStr := ctx.Query("cursor")
	sizeStr := ctx.Query("size")
	cursor := int64(0)
	size := int64(100)
	myFollow := true
	followDetail := false
	if ctx.Query("followDetail") == "true" {
		followDetail = true
	}
	if cursorStr != "" && sizeStr != "" {
		cursor, _ = strconv.ParseInt(cursorStr, 10, 64)
		size, _ = strconv.ParseInt(sizeStr, 10, 64)
	}
	list, total, err := man.DbAdapter.GetFollowDataByMetaId(ctx.Param("metaid"), myFollow, followDetail, cursor, size)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			ctx.JSON(http.StatusOK, respond.ErrNoResultFound)
		} else {
			ctx.JSON(http.StatusOK, respond.ErrServiceError)
		}
		return
	}
	ctx.JSON(http.StatusOK, respond.ApiSuccess(1, "ok", gin.H{"list": list, "total": total}))

}

// getRecommendedList
func getRecommendedList(ctx *gin.Context) {
	limit := int(100)
	num := int(6)
	if ctx.Query("top") != "" {
		limit, _ = strconv.Atoi(ctx.Query("limit"))
	}
	if ctx.Query("num") != "" {
		num, _ = strconv.Atoi(ctx.Query("num"))
	}
	if limit > 500 {
		limit = 500
	}
	if num > limit {
		num = limit
	}
	list, err := mongodb.GetRecommendedList(limit)
	if err != nil {
		ctx.JSON(http.StatusOK, respond.ErrServiceError)
		return
	}

	// 使用局部随机数生成器打乱 list 的顺序
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	r.Shuffle(len(list), func(i, j int) {
		list[i], list[j] = list[j], list[i]
	})

	// 取前 num 个元素
	if len(list) > num {
		list = list[:num]
	}
	ctx.JSON(http.StatusOK, respond.ApiSuccess(1, "ok", list))

}

// getFollowRecord
func getFollowRecord(ctx *gin.Context) {
	metaId := ctx.Query("metaId")
	followMetaId := ctx.Query("followerMetaId")
	if metaId == "" || followMetaId == "" {
		ctx.JSON(http.StatusOK, respond.ErrParameterError)
		return
	}
	info, err := man.DbAdapter.GetFollowRecord(metaId, followMetaId)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			ctx.JSON(http.StatusOK, respond.ErrNoResultFound)
		} else {
			ctx.JSON(http.StatusOK, respond.ErrServiceError)
		}
		return
	}
	ctx.JSON(http.StatusOK, respond.ApiSuccess(1, "ok", info))

}
func reindex(ctx *gin.Context) {
	token := ctx.Query("token")
	if token != common.Config.AdminToken || token == "" {
		ctx.JSON(http.StatusOK, "error token")
		return
	}
	chain := ctx.Param("chain")
	from, _ := strconv.ParseInt(ctx.Param("from"), 10, 64)
	to, _ := strconv.ParseInt(ctx.Param("to"), 10, 64)
	for i := from; i <= to; i++ {
		man.DoIndexerRun(chain, i, true)
		fmt.Println("reindex", i)
	}

	ctx.String(http.StatusOK, "reindex finish")
}

// notifcationList address=xx&lastId=100&size=10
func notifcationList(ctx *gin.Context) {
	address := ctx.Query("address")
	lastId, _ := strconv.ParseInt(ctx.Query("lastId"), 10, 64)
	//size, _ := strconv.ParseInt(ctx.Query("size"), 10, 64)
	result, err := man.PebbleStore.Database.GetNotifcation(address)
	if err != nil {
		ctx.JSON(http.StatusOK, respond.ErrServiceError)
		return
	}
	var list []pin.NotifcationData
	arr := strings.Split(string(result), "@*@")
	checkMap := make(map[string]struct{})
	for _, item := range arr {
		if item == "" {
			continue
		}
		var notif pin.NotifcationData
		if err := json.Unmarshal([]byte(item), &notif); err == nil {
			if _, ok := checkMap[notif.FromPinId]; ok {
				continue
			}
			checkMap[notif.FromPinId] = struct{}{}
			list = append(list, notif)
		}
	}
	var lastList []pin.NotifcationData
	for _, notif := range list {
		if notif.NotifcationId <= lastId {
			continue
		}
		// if len(lastList) >= int(size) {
		// 	break
		// }
		lastList = append(lastList, notif)
	}
	total := int64(len(list))
	checkMap = nil
	list = nil
	sort.Slice(lastList, func(i, j int) bool {
		return lastList[i].NotifcationId > lastList[j].NotifcationId
	})
	ctx.JSON(http.StatusOK, gin.H{"code": 200, "message": "ok", "data": lastList, "total": total})
}
func dictSet(ctx *gin.Context) {
	dictKey := ctx.Query("key")
	if dictKey == "" {
		ctx.JSON(http.StatusOK, respond.ErrParameterError)
		return
	}
	dictValue := ctx.Query("value")
	if dictValue == "" {
		ctx.JSON(http.StatusOK, respond.ErrParameterError)
		return
	}
	err := common.SaveToDictDB(dictKey, []byte(dictValue))
	if err != nil {
		ctx.JSON(http.StatusOK, respond.ErrServiceError)
		return
	}
	ctx.JSON(http.StatusOK, respond.ApiSuccess(1, "ok", nil))
}
func dictGet(ctx *gin.Context) {
	dictKey := ctx.Query("key")
	if dictKey == "" {
		ctx.JSON(http.StatusOK, respond.ErrParameterError)
		return
	}
	value, err := common.LoadFromDictDB(dictKey)
	if err != nil {
		ctx.JSON(http.StatusOK, respond.ErrServiceError)
		return
	}
	ctx.JSON(http.StatusOK, respond.ApiSuccess(1, "ok", string(value)))
}

func blockFileGet(ctx *gin.Context) {
	heightStr := ctx.Query("height")
	if heightStr == "" {
		ctx.JSON(http.StatusOK, respond.ErrParameterError)
		return
	}
	height, err := strconv.ParseInt(heightStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusOK, respond.ErrParameterError)
		return
	}
	chainName := ctx.Query("chain")
	if chainName == "" {
		ctx.JSON(http.StatusOK, respond.ErrParameterError)
		return
	}
	partIndexStr := ctx.Query("part")
	partIndex := 0
	if partIndexStr != "" {
		partIndex, err = strconv.Atoi(partIndexStr)
		if err != nil {
			ctx.JSON(http.StatusOK, respond.ErrParameterError)
			return
		}
	}

	// 获取文件路径
	filePath := man.GetBlockFilePath(chainName, height, partIndex)
	if _, err := os.Stat(filePath); err != nil {
		ctx.JSON(http.StatusOK, respond.ApiError(404, "File does not exist"))
		return
	}

	// 设置下载响应头
	ctx.Header("Content-Type", "application/octet-stream")
	ctx.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filepath.Base(filePath)))
	ctx.File(filePath)
}

// 查询某区块分片文件数量
func blockPartCount(ctx *gin.Context) {
	heightStr := ctx.Query("height")
	if heightStr == "" {
		ctx.JSON(http.StatusOK, respond.ErrParameterError)
		return
	}
	height, err := strconv.ParseInt(heightStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusOK, respond.ErrParameterError)
		return
	}
	chainName := ctx.Query("chain")
	if chainName == "" {
		ctx.JSON(http.StatusOK, respond.ErrParameterError)
		return
	}

	// 遍历分片文件，统计数量
	dirPath := filepath.Join(
		common.Config.Pebble.Dir+"/blockFiles",
		strconv.FormatInt(height/1000000, 10),
		strconv.FormatInt((height%1000000)/1000, 10),
	)
	prefix := chainName + "_" + strconv.FormatInt(height, 10) + "_"
	count := 0
	files, err := os.ReadDir(dirPath)
	if err == nil {
		for _, f := range files {
			if !f.IsDir() && strings.HasPrefix(f.Name(), prefix) && strings.HasSuffix(f.Name(), ".dat.zst") {
				count++
			}
		}
	}
	btcMin, btcMax, _ := man.GetFileMetaHeight("btc")
	mvcMin, mvcMax, _ := man.GetFileMetaHeight("mvc")

	ctx.JSON(http.StatusOK, respond.ApiSuccess(1, "ok", gin.H{"partCount": count, "btcMin": btcMin, "btcMax": btcMax, "mvcMin": mvcMin, "mvcMax": mvcMax}))
}
func blockFileCreate(ctx *gin.Context) {
	token := ctx.Query("token")
	if token != common.Config.AdminToken || token == "" {
		ctx.JSON(http.StatusOK, "error token")
		return
	}
	chainName := ctx.Query("chain")
	if chainName == "" {
		ctx.JSON(http.StatusOK, respond.ErrParameterError)
		return
	}
	from, _ := strconv.ParseInt(ctx.Query("from"), 10, 64)
	to, _ := strconv.ParseInt(ctx.Query("to"), 10, 64)
	for i := from; i <= to; i++ {
		man.SaveBlockFileFromChain(chainName, i)
	}
	ctx.String(http.StatusOK, "block file create finish")
}

// SetPinIdList
func setPinIdList(ctx *gin.Context) {
	token := ctx.Query("token")
	if token != common.Config.AdminToken || token == "" {
		ctx.JSON(http.StatusOK, "error token")
		return
	}
	chainName := ctx.Query("chain")
	if chainName == "" {
		ctx.JSON(http.StatusOK, respond.ErrParameterError)
		return
	}
	from, _ := strconv.ParseInt(ctx.Query("from"), 10, 64)
	to, _ := strconv.ParseInt(ctx.Query("to"), 10, 64)
	for i := from; i <= to; i++ {
		man.PebbleStore.SetPinIdList(chainName, i)
	}
	ctx.String(http.StatusOK, "block file pin id list create finish")
}
func blockIdList(ctx *gin.Context) {
	token := ctx.Query("token")
	if token != common.Config.AdminToken || token == "" {
		ctx.JSON(http.StatusOK, "error token")
		return
	}
	chainName := ctx.Query("chain")
	if chainName == "" {
		ctx.JSON(http.StatusOK, respond.ErrParameterError)
		return
	}
	height, err := strconv.ParseInt(ctx.Query("height"), 10, 64)
	if err != nil {
		ctx.JSON(http.StatusOK, respond.ErrParameterError)
		return
	}
	blockIds, err := man.GetBlockIdList(chainName, int(height))
	if err != nil {
		ctx.JSON(http.StatusOK, respond.ErrServiceError)
		return
	}
	ctx.JSON(http.StatusOK, respond.ApiSuccess(1, "ok", gin.H{"data": blockIds}))
}
