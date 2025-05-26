package man

import (
	"manindexer/pebblestore"
	"manindexer/pin"
	"strconv"
	"strings"

	"github.com/bytedance/sonic"
)

func (pd *PebbleData) GetAllCount() (result pin.PinCount) {
	pinsVal, closer, err := pd.database.CountDB.Get([]byte("pins"))
	if err == nil {
		result.Pin, _ = strconv.ParseInt(string(pinsVal), 10, 64)
		closer.Close()
	}
	blockVal, closer2, err := pd.database.CountDB.Get([]byte("blocks"))
	if err == nil {
		result.Block, _ = strconv.ParseInt(string(blockVal), 10, 64)
		closer2.Close()
	}
	metaidVal, closer3, err := pd.database.CountDB.Get([]byte("metaids"))
	if err == nil {
		result.MetaId, _ = strconv.ParseInt(string(metaidVal), 10, 64)
		closer3.Close()
	}
	return
}
func (pd *PebbleData) PinPageList(page int, size int, lastId string) (list []pin.PinInscription, nextId string, err error) {
	q := pebblestore.PageQuery{Type: "pin", Page: page, Size: size, LastId: lastId}
	res, err := pd.database.QueryPageKeys(pd.database.PagesDB, q)
	if err != nil || len(res.List) <= 0 {
		return
	}
	pinResult := pd.database.BatchGetPinListByKeys(res.List, false)
	if len(pinResult) <= 0 || pinResult == nil {
		return
	}
	for _, val := range pinResult {
		var item pin.PinInscription
		err := sonic.Unmarshal(val, &item)
		if err == nil {
			list = append(list, item)
		}
	}
	nextId = res.NextId
	return
}

// QueryPageBlock
func (pd *PebbleData) QueryPageBlock(q pebblestore.PageQuery) (PageResult []pebblestore.PageBlock, err error) {
	keys, err := pd.database.QueryAllPinKeysByPageIndex(pd.database.PagesDB, q)
	if err != nil {
		return
	}
	for _, key := range keys {
		list, err := pd.database.GetBlockLimitPins(key, 100)
		if err != nil {
			continue
		}
		item := pebblestore.PageBlock{PinList: list}
		arr := strings.Split(key, "_")
		if len(arr) == 5 {
			item.BlockHeight = arr[3]
			item.ChainName = arr[4]
			item.BlockTime = arr[2]
		}
		PageResult = append(PageResult, item)
	}
	return
}
