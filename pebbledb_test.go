package main

import (
	"fmt"
	"manindexer/common"
	"manindexer/database/mongodb"
	"manindexer/man"
	"manindexer/pebblestore"
	"testing"
)

func TestPinPageList(t *testing.T) {
	common.InitConfig("./config_mvc.toml")
	man.InitAdapter("mvc", "mongo", "0", "1")
	list, nextId, err := man.PebbleStore.PinPageList(0, 1, "")
	fmt.Println(err, len(list), nextId)
	cnt, err := mongodb.CountMetaid()
	fmt.Println(err, cnt)
	//fmt.Println(list, nextId)
}
func TestQueryPageBlock(t *testing.T) {
	common.InitConfig("./config_mvc.toml")
	man.InitAdapter("mvc", "mongo", "0", "1")
	q := pebblestore.PageQuery{Type: "pin", Page: 0, Size: 2, LastId: ""}
	list, err := man.PebbleStore.QueryPageBlock(q)
	fmt.Println(err)
	fmt.Println(list)
}
