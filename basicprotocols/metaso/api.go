package metaso

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"manindexer/common"
	"manindexer/database/mongodb"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

var serverUrl string

func Api(r *gin.Engine) {
	accessGroup := r.Group("/social/buzz")
	accessGroup.Use(CorsMiddleware())
	accessGroup.GET("/newest", newest)
	accessGroup.GET("/recommended", recommended)
	accessGroup.GET("/updater", updater)
	accessGroup.GET("/hot", hot)
	accessGroup.GET("/search", search)
	accessGroup.GET("/info", info)
	accessGroup.GET("/follow", follow)
	hostGroup := r.Group("/host")
	hostGroup.Use(CorsMiddleware())
	hostGroup.GET("/block/sync-newest", syncNewest2)
	hostGroup.GET("/block/ndv", blockNDV)
	hostGroup.GET("/block/mdv", blockMDV)
	hostGroup.GET("/info", hostInfo)
	hostGroup.POST("/viewed/add", buzzViewedAdd)
	ftGroup := r.Group("/ft")
	ftGroup.Use(CorsMiddleware())
	ftGroup.GET("/mrc20/address/deploy-list", mrc20TickList)
	settingGroup := r.Group("/metaso/settings")
	settingGroup.Use(CorsMiddleware())
	settingGroup.GET("/blocked/list", blockedList)
	settingGroup.GET("/blocked/add", blockedAdd)
	settingGroup.GET("/blocked/delete", blockedDelete)
	settingGroup.GET("/recommended/list", listRecommendedAuthor)
	settingGroup.GET("/recommended/add", addRecommendedAuthor)
	settingGroup.GET("/recommended/delete", deleteRecommendedAuthor)
	//settingGroup.GET("/recommended/delete", deleteRecommendedAuthor)
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

// @Summary      Get latest buzz feed
// @Description  Retrieve paginated list of newest buzz items with filtering options
// @Tags         Buzz
// @Accept       json
// @Produce      json
// @Param        lastId    query    string  false  "Last record ID for pagination (cursor)"
// @Param        size      query    int     false  "Items per page (default: 10)"
// @Param        metaid    query    string  false  "Filter by meta ID"
// @Param        followed  query    string  false  "Filter followed content only (true/false)"
// @Success      200  {object}  ApiResponse{data=object{list=[]TweetWithLike,total=int,lastId=string}}  "Successfully retrieved buzz list"
// @Failure      400  {object}  ApiResponse  "Invalid parameters"
// @Failure      500  {object}  ApiResponse  "Service exception"
// @Router       /social/buzz/newest [get]
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
	// var newList []*TweetWithLike
	// for _, item := range list {
	// 	hostKey := fmt.Sprintf("host_%s", item.Host)
	// 	metaidKey := fmt.Sprintf("metaid_%s", item.CreateMetaId)
	// 	pinidKey := fmt.Sprintf("pinid_%s", item.Id)
	// 	if _, ok := BlockedData[hostKey]; ok {
	// 		item.Blocked = true
	// 	}
	// 	if _, ok := BlockedData[metaidKey]; ok {
	// 		item.Blocked = true
	// 	}
	// 	if _, ok := BlockedData[pinidKey]; ok {
	// 		item.Blocked = true
	// 	}
	// 	newList = append(newList, item)
	// }

	ctx.JSON(http.StatusOK, ApiSuccess(1, "ok", gin.H{"list": list, "total": total, "lastId": lastId}))
}

// @Summary      Get recommended buzz feed
// @Description  Retrieve paginated list of recommended buzz items with filtering options
// @Tags         Buzz
// @Accept       json
// @Produce      json
// @Param        lastId    query    string  false  "Last record ID for pagination (cursor)"
// @Param        size      query    int     false  "Items per page (default: 10)"
// @Param        userAddress  query    string  false  "Filter by userAddress"
// @Success      200  {object}  ApiResponse{data=object{list=[]TweetWithLike,total=int,lastId=string}}  "Successfully retrieved buzz list"
// @Failure      400  {object}  ApiResponse  "Invalid parameters"
// @Failure      500  {object}  ApiResponse  "Service exception"
// @Router       /social/buzz/recommended [get]
func recommended(ctx *gin.Context) {
	userAddress := ctx.Query("userAddress")
	size, err := strconv.ParseInt(ctx.Query("size"), 10, 64)
	if err != nil {
		ctx.JSON(http.StatusOK, ApiError(-1, "size error"))
		return
	}
	if size == 0 {
		size = 10
	}
	ms := &MetaSo{}
	list, total, err := ms.GetRecommendedPostsNew(ctx, ctx.Query("lastId"), userAddress, size)
	lastId := ""
	if len(list) > 0 {
		lastId = list[len(list)-1].MogoID.Hex()
	}
	if err != nil {
		fmt.Println(err)
		ctx.JSON(http.StatusOK, ApiError(-1, "service exception."))
		return
	}
	ctx.JSON(http.StatusOK, ApiSuccess(1, "ok", gin.H{"list": list, "total": total, "lastId": lastId}))
}

type updaterRes struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data updaterInfo `json:"data"`
}
type updaterInfo struct {
	Version   string `json:"Ver"`
	BuildNo   int64  `json:"BuildNo"`
	Mandatory bool   `json:"mandatory"`
}

// @Summary      Get update information
// @Description  Retrieves current and latest version information along with server details
// @Tags         Updater
// @Accept       json
// @Produce      json
// @Success      200  {object}  ApiResponse{data=object{lastNo=string,lastVer=string,curNo=string,curVer=string,serverUrl=string,mandatory=bool}}  "Successfully retrieved update information"
// @Failure      400  {object}  ApiResponse  "Error retrieving update information"
// @Failure      500  {object}  ApiResponse  "Server error"
// @Router       /social/buzz/updater [get]
func updater(ctx *gin.Context) {
	lastNo, lastVer, mandatory, err := getUpdaterInfo(true)
	if err != nil {
		ctx.JSON(http.StatusOK, ApiError(-1, err.Error()))
		return
	}
	curNo, curVer, _, err := getUpdaterInfo(false)
	if err != nil {
		ctx.JSON(http.StatusOK, ApiError(-1, err.Error()))
		return
	}
	if serverUrl == "" {
		ip, err := GetExternalIP()
		if err == nil {
			serverUrl = fmt.Sprintf("http://%s:7171", ip)
		}
	}

	ctx.JSON(http.StatusOK, ApiSuccess(1, "ok", gin.H{"lastNo": lastNo, "lastVer": lastVer, "curNo": curNo, "curVer": curVer, "serverUrl": serverUrl, "mandatory": mandatory}))
}

var services = []string{
	"https://icanhazip.com",
	"https://ipinfo.io/ip",
	"https://api.ipify.org",
}

func GetExternalIP() (string, error) {
	for _, service := range services {
		cmd := exec.Command("curl", "-s", service)
		output, err := cmd.Output()
		if err != nil {
			continue
		}
		ip := string(output)
		ip = strings.ReplaceAll(ip, " ", "")
		ip = strings.ReplaceAll(ip, "\n", "")
		ip = strings.ReplaceAll(ip, "\r", "")
		if isValidIP(ip) {
			return ip, nil
		}
	}

	return "", fmt.Errorf("failed to get external IP from all services")
}

func isValidIP(ip string) bool {
	arr := strings.Split(ip, ".")
	return len(arr) == 4
}

func getUpdaterInfo(last bool) (buildNo int64, ver string, mandatory bool, err error) {
	versionUrl := "http://host.docker.internal:7171/api/lastVersion"
	if !last {
		versionUrl = "http://host.docker.internal:7171/api/checkStatus"
	}
	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	resp, err := client.Get(versionUrl)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}
	var data updaterRes
	err = json.Unmarshal(body, &data)
	if err != nil {
		return
	}
	buildNo = data.Data.BuildNo
	ver = data.Data.Version
	mandatory = data.Data.Mandatory
	return
}

// @Summary      Get hot buzz feed
// @Description  Retrieve paginated list of hottest buzz items ranked by popularity
// @Tags         Buzz
// @Accept       json
// @Produce      json
// @Param        lastId    query    string  false  "Last record ID for pagination (cursor-based)"
// @Param        size      query    int     false  "Number of items per page (default: 10, max: 50)"
// @Success      200  {object}  ApiResponse{data=object{list=[]TweetWithLike,total=int,lastId=string}}  "Successfully retrieved hot buzz list"
// @Failure      400  {object}  ApiResponse  "Invalid size parameter"
// @Failure      500  {object}  ApiResponse  "Service exception"
// @Router       /social/buzz/hot [get]
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

// @Summary      Search buzz items
// @Description  Search buzz items by keyword with pagination support
// @Tags         Buzz
// @Accept       json
// @Produce      json
// @Param        lastId  query  string  false  "Last record ID for pagination"
// @Param        size    query  int     false  "Number of items per page (default: 10)"
// @Param        key     query  string  true   "Search keyword"
// @Success      200  {object}  ApiResponse{data=object{list=[]TweetWithLike,total=int,lastId=string}}  "Search results"
// @Failure      400  {object}  ApiResponse  "Invalid parameters"
// @Failure      500  {object}  ApiResponse  "Service exception"
// @Router       /social/buzz/search [get]
func search(ctx *gin.Context) {
	size, err := strconv.ParseInt(ctx.Query("size"), 10, 64)
	if err != nil {
		ctx.JSON(http.StatusOK, ApiError(-1, "size error"))
		return
	}
	if size == 0 {
		size = 10
	}
	list, total, err := textSearch(ctx.Query("lastId"), size, ctx.Query("key"))
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

// @Summary      Get buzz item details
// @Description  Get complete information about a specific buzz item including comments, likes and donations
// @Tags         Buzz
// @Accept       json
// @Produce      json
// @Param        pinId  query  string  true  "ID of the buzz item to retrieve"
// @Success      200  {object}  ApiResponse{data=object{tweet=Tweet,comments=[]TweetComment,like=int,donates=[]MetasoDonate,blocked=bool}}  "Buzz item details"
// @Failure      400  {object}  ApiResponse  "Missing pinId parameter"
// @Failure      500  {object}  ApiResponse  "Service exception"
// @Router       /social/buzz/info [get]
func info(ctx *gin.Context) {
	tweet, comments, like, donates, err := getInfo(ctx.Query("pinId"))
	if err != nil || tweet == nil {
		ctx.JSON(http.StatusOK, ApiError(-1, "service exception"))
		return
	}
	blocked := false
	hostKey := fmt.Sprintf("host_%s", tweet.Host)
	metaidKey := fmt.Sprintf("metaid_%s", tweet.CreateMetaId)
	pinidKey := fmt.Sprintf("pinid_%s", tweet.Id)
	if _, ok := common.BlockedData[hostKey]; ok {
		blocked = true
	}
	if _, ok := common.BlockedData[metaidKey]; ok {
		blocked = true
	}
	if _, ok := common.BlockedData[pinidKey]; ok {
		blocked = true
	}

	ctx.JSON(http.StatusOK, ApiSuccess(1, "ok", gin.H{"tweet": tweet, "comments": comments, "like": like, "donates": donates, "blocked": blocked}))
}

type followItem struct {
	Metaid   string `json:"metaid"`
	Mempool  int    `json:"mempool"`
	Unfollow int    `json:"unfollow"`
}

// @Summary      Get follow information
// @Description  Retrieve follow data for a specific meta ID including mempool status
// @Tags         Buzz
// @Accept       json
// @Produce      json
// @Param        metaid  query  string  true  "Meta ID to query follow data"
// @Success      200  {object}  ApiResponse{data=object{list=[]followItem}}  "Follow information"
// @Failure      400  {object}  ApiResponse  "Missing metaid parameter"
// @Failure      500  {object}  ApiResponse  "Service exception"
// @Router       /social/buzz/follow [get]
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

// @Summary      Get block NDV data
// @Description  Retrieve NDV (Node Data Verification) information for blocks with pagination
// @Tags         Statistics
// @Accept       json
// @Produce      json
// @Param        height  query  int     true   "Block height"
// @Param        host    query  string  false  "Host filter"
// @Param        cursor  query  int     false  "Pagination cursor"
// @Param        size    query  int     false  "Number of items per page"
// @Param        orderby query  string  false  "Sorting field"
// @Success      200  {object}  ApiResponse  "NDV data for requested block"
// @Failure      400  {object}  ApiResponse  "Invalid query parameters"
// @Failure      500  {object}  ApiResponse  "Service exception"
// @Router       /host/block/ndv [get]
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

// @Summary      Get block MDV data
// @Description  Retrieve MDV (Miner Data Verification) information for blocks with pagination
// @Tags         Statistics
// @Accept       json
// @Produce      json
// @Param        height  query  int     true   "Block height"
// @Param        address query  string  false  "Miner address filter"
// @Param        cursor  query  int     false  "Pagination cursor"
// @Param        size    query  int     false  "Number of items per page"
// @Param        orderby query  string  false  "Sorting field"
// @Success      200  {object}  ApiResponse  "MDV data for requested block"
// @Failure      400  {object}  ApiResponse  "Invalid query parameters"
// @Failure      500  {object}  ApiResponse  "Service exception"
// @Router       /host/block/mdv [get]
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

// @Summary      Get paginated NDV list
// @Description  Retrieve paginated list of NDV (Node Data Verification) records
// @Tags         Statistics
// @Accept       json
// @Produce      json
// @Param        host    query  string  false  "Host filter"
// @Param        cursor  query  int     false  "Pagination cursor"
// @Param        size    query  int     false  "Number of items per page"
// @Param        orderby query  string  false  "Sorting field"
// @Success      200  {object}  ApiResponse  "List of NDV records"
// @Failure      400  {object}  ApiResponse  "Invalid query parameters"
// @Failure      500  {object}  ApiResponse  "Service exception"
// @Router       /statistics/ndv [get]
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

	list, err := getNdvPageList(strings.ToLower(ctx.Query("host")), cursor, size, ctx.Query("orderby"))
	if err != nil {
		ctx.JSON(http.StatusOK, ApiError(-1, "service exception"))
		return
	}
	ctx.JSON(http.StatusOK, ApiSuccess(1, "ok", list))
}

// @Summary      Get paginated MDV list
// @Description  Retrieve paginated list of MDV (Miner Data Verification) records
// @Tags         Statistics
// @Accept       json
// @Produce      json
// @Param        address query  string  false  "Miner address filter"
// @Param        cursor  query  int     false  "Pagination cursor"
// @Param        size    query  int     false  "Number of items per page"
// @Param        orderby query  string  false  "Sorting field"
// @Success      200  {object}  ApiResponse  "List of MDV records"
// @Failure      400  {object}  ApiResponse  "Invalid query parameters"
// @Failure      500  {object}  ApiResponse  "Service exception"
// @Router       /statistics/mdv [get]
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

// @Summary      Get host information
// @Description  Retrieve block information for a specific host
// @Tags         Host
// @Accept       json
// @Produce      json
// @Param        host    query  string  true   "Host identifier"
// @Param        cursor  query  int     false  "Pagination cursor"
// @Param        size    query  int     false  "Number of items per page"
// @Param        orderby query  string  false  "Sorting field"
// @Success      200  {object}  ApiResponse  "Host block information"
// @Failure      400  {object}  ApiResponse  "Invalid query parameters"
// @Failure      500  {object}  ApiResponse  "Service exception"
// @Router       /host/info [get]
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

// @Summary      Get MRC20 token list
// @Description  Retrieve list of MRC20 tokens for an address
// @Tags         Tokens
// @Accept       json
// @Produce      json
// @Param        address  query  string  true   "Wallet address"
// @Param        tickType query  string  false  "Token type filter"
// @Success      200  {object}  ApiResponse  "List of MRC20 tokens"
// @Failure      400  {object}  ApiResponse  "Missing address parameter"
// @Failure      500  {object}  ApiResponse  "Service exception"
// @Router       /ft/mrc20/address/deploy-list [get]
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

// @Summary      Get blocked items list
// @Description  Retrieve paginated list of blocked items
// @Tags         Settings
// @Accept       json
// @Produce      json
// @Param        blockType query  string  true   "Type of blocked items (host/pinid/etc)"
// @Param        cursor    query  int     false  "Pagination cursor"
// @Param        size      query  int     false  "Number of items per page"
// @Success      200  {object}  ApiResponse{data=object{list=array,total=int}}  "Blocked items list with total count"
// @Failure      400  {object}  ApiResponse  "Missing or invalid parameters"
// @Failure      500  {object}  ApiResponse  "Service exception"
// @Router       /metaso/settings/blocked/list [get]
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

// @Summary      Add item to blocked list
// @Description  Add new item to the blocked items list
// @Tags         Settings
// @Accept       json
// @Produce      json
// @Param        blockType      query  string  true  "Type of item to block"
// @Param        blockContent   query  string  true  "Content to block"
// @Success      200  {object}  ApiResponse  "Success response"
// @Failure      400  {object}  ApiResponse  "Missing required parameters"
// @Failure      500  {object}  ApiResponse  "Service exception"
// @Router       /metaso/settings/blocked/add [get]
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
	originalContent := ctx.Query("blockContent")
	if blockType == "host" {
		blockContent = strings.ToLower(blockContent)
	}
	err := addBlockedList(blockType, blockContent, originalContent)
	if err != nil {
		ctx.JSON(http.StatusOK, ApiError(-1, "service exception"))
		return
	}
	ctx.JSON(http.StatusOK, ApiSuccess(1, "ok", nil))
}

// @Summary      Remove item from blocked list
// @Description  Remove item from the blocked items list
// @Tags         Settings
// @Accept       json
// @Produce      json
// @Param        blockType    query  string  true  "Type of blocked item"
// @Param        blockContent query  string  true  "Content to unblock"
// @Success      200  {object}  ApiResponse  "Success response"
// @Failure      400  {object}  ApiResponse  "Missing required parameters"
// @Failure      500  {object}  ApiResponse  "Service exception"
// @Router       /metaso/settings/blocked/delete [get]
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

// @Summary      Get recommended author list
// @Description  Retrieve paginated list of recommended author
// @Tags         Settings
// @Accept       json
// @Produce      json
// @Param        cursor    query  int     false  "Pagination cursor"
// @Param        size      query  int     false  "Number of items per page"
// @Success      200  {object}  ApiResponse{data=object{list=array,total=int}}  "recommended author list with total count"
// @Failure      400  {object}  ApiResponse  "Missing or invalid parameters"
// @Failure      500  {object}  ApiResponse  "Service exception"
// @Router       /metaso/settings/recommended/list [get]
func listRecommendedAuthor(ctx *gin.Context) {
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
	ms := &MetaSo{}
	list, total, err := ms.GetRecommendedAuthors(ctx, cursor, size)
	if err != nil {
		ctx.JSON(http.StatusOK, ApiError(-1, "service exception"))
		return
	}
	ctx.JSON(http.StatusOK, ApiSuccess(1, "ok", gin.H{"list": list, "total": total}))
}

// @Summary      Add item to RecommendedAuthor list
// @Description  Add new item to the RecommendedAuthor items list
// @Tags         Settings
// @Accept       json
// @Produce      json
// @Param        authorAddress      query  string  true  "Address of author to list"
// @Param        authorNmae      query  string  true  "Name of author to list"
// @Success      200  {object}  ApiResponse  "Success response"
// @Failure      400  {object}  ApiResponse  "Missing required parameters"
// @Failure      500  {object}  ApiResponse  "Service exception"
// @Router       /metaso/settings/recommended/add [get]
func addRecommendedAuthor(ctx *gin.Context) {
	authorAddress := ctx.Query("authorAddress")
	if authorAddress == "" {
		ctx.JSON(http.StatusOK, ApiError(-1, "authorAddress is null"))
		return
	}
	authorName := ctx.Query("authorName")
	if authorName == "" {
		ctx.JSON(http.StatusOK, ApiError(-1, "authorName is null"))
		return
	}
	ms := &MetaSo{}
	err := ms.AddRecommendedAuthor(ctx, authorAddress, authorName)
	if err != nil {
		ctx.JSON(http.StatusOK, ApiError(-1, "service exception"))
		return
	}
	ctx.JSON(http.StatusOK, ApiSuccess(1, "ok", nil))
}

// @Summary      Remove item from RecommendedAuthor list
// @Description  Remove item from the RecommendedAuthor items list
// @Tags         Settings
// @Accept       json
// @Produce      json
// @Param        authorAddress      query  string  true  "Address of author to list"
// @Success      200  {object}  ApiResponse  "Success response"
// @Failure      400  {object}  ApiResponse  "Missing required parameters"
// @Failure      500  {object}  ApiResponse  "Service exception"
// @Router       /metaso/settings/recommended/delete [get]
func deleteRecommendedAuthor(ctx *gin.Context) {
	authorAddress := ctx.Query("authorAddress")
	if authorAddress == "" {
		ctx.JSON(http.StatusOK, ApiError(-1, "authorAddress is null"))
		return
	}
	ms := &MetaSo{}
	err := ms.RemoveRecommendedAuthor(ctx, authorAddress)
	if err != nil {
		ctx.JSON(http.StatusOK, ApiError(-1, "service exception"))
		return
	}
	ctx.JSON(http.StatusOK, ApiSuccess(1, "ok", nil))
}

type buzzViewedAddReq struct {
	PinIdList []string `json:"pinIdList" binding:"required"`
	Address   string   `json:"address" binding:"required"`
}

func buzzViewedAdd(ctx *gin.Context) {
	var req buzzViewedAddReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusOK, ApiError(-1, "invalid request"))
		return
	}
	if len(req.PinIdList) == 0 {
		ctx.JSON(http.StatusOK, ApiError(-1, "pinIdList is null"))
		return
	}
	if req.Address == "" {
		ctx.JSON(http.StatusOK, ApiError(-1, "address is null"))
		return
	}
	v := []string{}
	for _, pinId := range req.PinIdList {
		item := fmt.Sprintf("%s_%d", pinId, time.Now().Unix())
		v = append(v, item)
	}
	err := MergeUserOperationData("readed_log", req.Address, fmt.Sprintf("%s,", strings.Join(v, ",")))
	if err != nil {
		ctx.JSON(http.StatusOK, ApiError(-1, "service exception"))
		return
	}
	go CleanOldUserOperationData("readed_log", req.Address) // Clean up after 10 days
	ctx.JSON(http.StatusOK, ApiSuccess(1, "ok", nil))
}
