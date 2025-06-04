package man

import (
	"manindexer/pebblestore"
	"manindexer/pin"
	"strconv"

	"github.com/bytedance/sonic"
)

func (pd *PebbleData) GetAllCount() (result pin.PinCount) {
	pinsVal, closer, err := pd.Database.CountDB.Get([]byte("pins"))
	if err == nil {
		result.Pin, _ = strconv.ParseInt(string(pinsVal), 10, 64)
		closer.Close()
	}
	blockVal, closer2, err := pd.Database.CountDB.Get([]byte("blocks"))
	if err == nil {
		result.Block, _ = strconv.ParseInt(string(blockVal), 10, 64)
		closer2.Close()
	}
	metaidVal, closer3, err := pd.Database.CountDB.Get([]byte("metaids"))
	if err == nil {
		result.MetaId, _ = strconv.ParseInt(string(metaidVal), 10, 64)
		closer3.Close()
	}
	return
}
func (pd *PebbleData) PinPageList(page int, size int, lastId string) (list []pin.PinInscription, nextId string, err error) {
	q := pebblestore.PageQuery{Type: "pin", Page: page, Size: size, LastId: lastId}
	res, err := pd.Database.QueryPinPageList(pd.Database.PinSort, q)
	if err != nil || len(res.List) <= 0 {
		return
	}
	pinResult := pd.Database.BatchGetPinListByKeys(res.List, false)
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
	PageResult, err = pd.Database.GetBlockPageList(q.Page, q.Size, 100)
	return
}
