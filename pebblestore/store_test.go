package pebblestore

import (
	"fmt"
	"manindexer/common"
	"manindexer/pin"
	"strconv"
	"testing"
)

func TestBatchInsertPins(t *testing.T) {
	// 使用临时目录
	dir := "../data_test"
	idx, err := NewDataBase(dir, 4)
	if err != nil {
		t.Fatalf("NewDataBase err: %v", err)
	}
	defer idx.Close()

	pins := []pin.PinInscription{
		{Id: "txid1", ChainName: "chainA", ContentSummary: "111"},
		{Id: "txid2", ChainName: "chainA", ContentSummary: "222"},
		{Id: "txid3", ChainName: "chainB", ContentSummary: "333"},
	}
	err = idx.BatchInsertPins(pins)
	if err != nil {
		t.Fatalf("BatchInsertPins err: %v", err)
	}

	// 查询主键（自动分片）
	key := BuildPinKey("txid1", 0)
	val, err := idx.GetPinByKey(key)
	if err != nil {
		t.Fatalf("主键查询失败: %v", err)
	}
	t.Logf("主键%s查询结果: %+v", key, string(val))
	if string(val) != "111" {
		t.Fatalf("主键查询内容不符: %+v", string(val))
	}

	// 插入分页信息
	page := PageInfo{
		BlockTime:   111,
		BlockHeight: 1,
		Type:        "pin",
		Num:         2,
		Keys:        []string{"txid1:0", "txid2:1"},
	}
	err = idx.InsertPageInfo(idx.PagesDB, page)
	if err != nil {
		t.Fatalf("InsertPageInfo err: %v", err)
	}

	// 分页查询
	q := PageQuery{Type: "pin", Page: 0, Size: 2, LastId: ""}
	t.Logf("分页查询条件: %+v", q)
	res, err := idx.QueryPageKeys(idx.PagesDB, q)
	if err != nil {
		t.Fatalf("分页查询失败: %v", err)
	}
	t.Logf("分页查询结果: %+v", res)
	if len(res.List) != 2 || res.List[0] != "txid1:0" {
		t.Fatalf("分页查询结果不符: %+v", res)
	}

	// 批量查询主键
	keys := []string{
		BuildPinKey("txid1", 0),
		BuildPinKey("txid2", 1),
		BuildPinKey("txid3", 0),
		BuildPinKey("notfound", 0), // 不存在的key
	}
	vals := idx.BatchGetPinByKeys(keys, false)
	for _, k := range keys {
		if v, ok := vals[k]; ok {
			t.Logf("批量主键查询: %s => %s", k, string(v))
		} else {
			t.Logf("批量主键查询: %s => not found", k)
		}
	}

	// 测试区块交易表写入和读取
	blockKeys := []string{"txid1:0", "txid2:1"}
	err = idx.InsertBlockTxs("chainA", 1, blockKeys)
	if err != nil {
		t.Fatalf("InsertBlockTxs err: %v", err)
	}
	blockKey := common.ConcatBytesOptimized([]string{"chainA", "_block_", strconv.Itoa(1)}, "")
	val, closer, err := idx.BlocksDB.Get([]byte(blockKey))
	if err != nil {
		t.Fatalf("区块交易表查询失败: %v", err)
	}
	blockTxs := SplitBytesOptimized(string(val), "|")
	closer.Close()
	if len(blockTxs) != 2 || blockTxs[0] != "txid1:0" {
		t.Fatalf("区块交易表内容不符: %+v", blockTxs)
	}
	t.Logf("区块交易表内容: %+v", blockTxs)
}
func TestPebbleMerge(t *testing.T) {
	dir := "../data_test"
	idx, err := NewDataBase(dir, 4)
	if err != nil {
		t.Fatalf("NewDataBase err: %v", err)
	}
	defer idx.Close()
	data := make(map[string]string)
	data["a"] = "1"
	idx.BatchMergeAddressData(data)
	data2 := make(map[string]string)
	data2["a"] = "2"
	idx.BatchMergeAddressData(data2)
	v, closer, err := idx.AddressDB.Get([]byte("a"))
	defer closer.Close()
	fmt.Println(err, string(v))
}
