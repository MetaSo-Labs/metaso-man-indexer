package pebblestore

import (
	"fmt"
	"manindexer/common"
	"manindexer/pin"
	"os"
	"strconv"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/cespare/xxhash/v2"
	"github.com/cockroachdb/pebble/v2"
)

// ShardConfig 配置分片数量
var ShardConfig = 16
var noopLogger = &customLogger{}

// Indexer 封装 Pebble 多链索引，pins为分片db，pages/blocks为独立db
// PinsDBs: 每个分片一个pebble实例
// PagesDB: 分页信息独立pebble实例
// BlocksDB: 区块交易独立pebble实例
// CountDB: 一些缓冲的统计数据
// PathPinDB：按区块存储的pin_path数据，key是 path_blockTime_chainName_height,value是[]pinId
// AddressDB: 按地址存储的PIN ID列表，key是address转换后的metaid,value是[]pinId&path&outputValue
type Database struct {
	PinsDBs   []*pebble.DB
	PagesDB   *pebble.DB
	BlocksDB  *pebble.DB
	CountDB   *pebble.DB
	PathPinDB *pebble.DB
	AddressDB *pebble.DB
}
type customLogger struct{}

func (l *customLogger) Infof(format string, args ...interface{})  {}
func (l *customLogger) Fatalf(format string, args ...interface{}) {}
func (l *customLogger) Errorf(format string, args ...interface{}) {}

// NewDataBase 创建索引器，自动创建分片db、pages、blocks独立db
func NewDataBase(basePath string, shardNum int) (*Database, error) {
	pinsDBs := make([]*pebble.DB, shardNum)
	for i := 0; i < shardNum; i++ {
		dir := fmt.Sprintf("%s/pins_%d", basePath, i)
		os.MkdirAll(dir, 0755)
		db, err := pebble.Open(fmt.Sprintf("%s/db", dir), &pebble.Options{Logger: noopLogger})
		if err != nil {
			return nil, err
		}
		pinsDBs[i] = db
	}
	os.MkdirAll(fmt.Sprintf("%s/pages", basePath), 0755)
	pagesDB, err := pebble.Open(fmt.Sprintf("%s/pages/db", basePath), &pebble.Options{Logger: noopLogger})
	if err != nil {
		return nil, err
	}
	os.MkdirAll(fmt.Sprintf("%s/blocks", basePath), 0755)
	blocksDB, err := pebble.Open(fmt.Sprintf("%s/blocks/db", basePath), &pebble.Options{Logger: noopLogger})
	if err != nil {
		return nil, err
	}
	os.MkdirAll(fmt.Sprintf("%s/blocks", basePath), 0755)
	countDB, err := pebble.Open(fmt.Sprintf("%s/count/db", basePath), &pebble.Options{Logger: noopLogger})
	if err != nil {
		return nil, err
	}
	os.MkdirAll(fmt.Sprintf("%s/blocks", basePath), 0755)
	pathPinDB, err := pebble.Open(fmt.Sprintf("%s/path/db", basePath), &pebble.Options{Logger: noopLogger})
	if err != nil {
		return nil, err
	}
	os.MkdirAll(fmt.Sprintf("%s/blocks", basePath), 0755)
	addressDB, err := pebble.Open(fmt.Sprintf("%s/address/db", basePath), &pebble.Options{Logger: noopLogger})
	if err != nil {
		return nil, err
	}
	return &Database{PinsDBs: pinsDBs, PagesDB: pagesDB, BlocksDB: blocksDB, CountDB: countDB, PathPinDB: pathPinDB, AddressDB: addressDB}, nil
}

// Close 关闭所有数据库
func (idx *Database) Close() error {
	for _, db := range idx.PinsDBs {
		db.Close()
	}
	idx.PagesDB.Close()
	idx.BlocksDB.Close()
	return nil
}

// BatchInsertPins 分片批量插入pins主表
// pins: 交易信息列表，由调用方控制每次插入数量
func (idx *Database) BatchInsertPins(pins []pin.PinInscription) error {
	batches := make(map[*pebble.DB][]struct {
		key string
		val []byte
	})
	for _, pin := range pins {
		//key := BuildPinKey(pin.Txid, pin.OutputIndex)
		db := idx.getShard(pin.Id)
		//content,err := json.Marshal(pin)
		content, err := sonic.Marshal(pin)
		if err != nil {
			continue
		}
		batches[db] = append(batches[db], struct {
			key string
			val []byte
		}{pin.Id, content})
	}
	for db, kvs := range batches {
		batch := db.NewBatch()
		for _, kv := range kvs {
			batch.Set([]byte(kv.key), kv.val, nil)
		}
		if err := batch.Commit(nil); err != nil {
			batch.Close()
			return err
		}
		batch.Close()
	}
	return nil
}
func (idx *Database) BatchInsertPathPins(data map[string]string) {
	for k, v := range data {
		idx.PathPinDB.Set([]byte(k), []byte(v), pebble.Sync)
	}
}
func (idx *Database) BatchMergeAddressData(data map[string]string) {
	for k, v := range data {
		idx.AddressDB.Merge([]byte(k), []byte(v), pebble.Sync)
	}
}

// PageInfo 分页信息
// 用于二级索引统计
// type=pin, key: pin_n_出块时间_区块高度, value: num
// type=pin, key: pin_s_出块时间_区块高度, value: txid:outputindex列表
type PageInfo struct {
	ChainName   string
	BlockTime   int64
	BlockHeight int64
	Type        string   // pin
	Num         int      // 该分页数量
	Keys        []string // txid:outputindex 列表
}

// InsertPageInfo 插入分页信息及二级索引
func (idx *Database) InsertPageInfo(db *pebble.DB, page PageInfo) error {
	// keyS := fmt.Sprintf("%s_s_%d_%d", page.Type, page.BlockTime, page.BlockHeight)
	keyS := common.ConcatBytesOptimized([]string{page.Type, "_s_", fmt.Sprint(page.BlockTime), "_", fmt.Sprint(page.BlockHeight), "_", page.ChainName}, "")
	// keyN := fmt.Sprintf("%s_n_%d_%d", page.Type, page.BlockTime, page.BlockHeight)
	keyN := common.ConcatBytesOptimized([]string{page.Type, "_n_", fmt.Sprint(page.BlockTime), "_", fmt.Sprint(page.BlockHeight), "_", page.ChainName}, "")
	keyT := common.ConcatBytesOptimized([]string{page.Type, "_t_", fmt.Sprint(page.BlockHeight), "_", page.ChainName}, "")
	valS := []byte(common.ConcatBytesOptimized(page.Keys, "|"))
	valN := []byte(fmt.Sprintf("%d", page.Num))
	valT := []byte(fmt.Sprint(page.BlockTime))
	batch := db.NewBatch()
	batch.Set([]byte(keyS), valS, nil)
	batch.Set([]byte(keyN), valN, nil)
	batch.Set([]byte(keyT), valT, nil)
	err := batch.Commit(nil)
	batch.Close()
	return err
}

// InsertBlockTxs 插入区块交易表
func (idx *Database) InsertBlockTxs(chainName string, blockHeight int64, keys []string) error {
	// key := fmt.Sprintf("%s_block_%d", chainName, blockHeight)
	key := common.ConcatBytesOptimized([]string{chainName, "_block_", fmt.Sprint(blockHeight)}, "")
	val := []byte(common.ConcatBytesOptimized(keys, "|"))
	return idx.BlocksDB.Set([]byte(key), val, nil)
}

// PageQuery 分页查询参数
type PageQuery struct {
	Type   string // pin
	Page   int
	Size   int
	LastId string // 上次最后一个id
}

// PageResult 分页查询结果
type PageResult struct {
	List   []string // txid:outputindex
	NextId string   // 下次分页用
}

// QueryPageKeys 通用分页key查询，pages.db
func (idx *Database) QueryPageKeys(db *pebble.DB, q PageQuery) (PageResult, error) {
	prefix := fmt.Sprintf("%s_n_", q.Type)
	it, _ := idx.PagesDB.NewIter(nil)
	defer it.Close()
	var (
		pageKeys []string
	)
	for it.Last(); it.Valid(); it.Prev() {
		key := string(it.Key())
		if len(key) < len(prefix) || key[:len(prefix)] != prefix {
			continue
		}
		pageKeys = append(pageKeys, key)
	}
	if len(pageKeys) == 0 {
		return PageResult{}, nil
	}
	var allKeys []string
	for _, k := range pageKeys {
		k2 := k[:len(q.Type)] + "_s" + k[len(q.Type)+2:]
		val, closer, err := idx.PagesDB.Get([]byte(k2))
		if err == nil {
			keys := SplitBytesOptimized(string(val), "|")
			allKeys = append(allKeys, keys...)
			closer.Close()
		}
	}
	start := q.Page * q.Size
	if q.LastId != "" {
		for i, v := range allKeys {
			if v == q.LastId {
				start = i + 1
				break
			}
		}
	}
	end := start + q.Size
	if start > len(allKeys) {
		start = len(allKeys)
	}
	if end > len(allKeys) {
		end = len(allKeys)
	}
	res := PageResult{
		List:   allKeys[start:end],
		NextId: "",
	}
	if end < len(allKeys) {
		res.NextId = allKeys[end-1]
	}
	return res, nil
}

// QueryAllPinKeysByPageIndex 按PagesDB中的二级索引分页查询，返回所有一级索引key（按时间排序）
func (idx *Database) QueryAllPinKeysByPageIndex(db *pebble.DB, q PageQuery) ([]string, error) {
	prefix := fmt.Sprintf("%s_n_", q.Type)
	it, _ := idx.PagesDB.NewIter(nil)
	defer it.Close()
	var (
		pageKeys []string
	)
	// 倒序遍历二级索引，收集所有主索引key（如 pin_s_...），最新在前
	for it.Last(); it.Valid(); it.Prev() {
		key := string(it.Key())
		if len(key) < len(prefix) || key[:len(prefix)] != prefix {
			continue
		}
		mainKey := key[:len(q.Type)] + "_s" + key[len(q.Type)+2:]
		pageKeys = append(pageKeys, mainKey)
	}
	return pageKeys, nil
}
func (idx *Database) GetBlockLimitPins(pagekey string, size int) (list []pin.PinInscription, err error) {
	val, closer, err := idx.PagesDB.Get([]byte(pagekey))
	if err != nil {
		return
	}
	defer closer.Close()
	keys := SplitBytesOptimized(string(val), "|")
	var pinIdList []string
	if len(keys) > size {
		pinIdList = keys[0 : size-1]
	} else {
		pinIdList = keys
	}
	if len(pinIdList) <= 0 {
		return
	}
	result := idx.BatchGetPinListByKeys(pinIdList, false)
	for _, val := range result {
		var item pin.PinInscription
		err := sonic.Unmarshal(val, &item)
		if err == nil {
			list = append(list, item)
		}
	}
	return
}

type PageBlock struct {
	BlockHeight string
	BlockTime   string
	ChainName   string
	PinList     []pin.PinInscription
}

// GetPinByKey 通用分片主键查询
// key 格式: chainName_pins_txid:outputindex
func (idx *Database) GetPinByKey(key string) ([]byte, error) {
	db := idx.getShard(key)
	val, closer, err := db.Get([]byte(key))
	if err != nil {
		return nil, err
	}
	closer.Close() // 直接调用，避免 defer 带来的性能损耗
	return val, nil
}

// BatchGetPinByKeys 批量查询主键，返回 map[key]value 只包含查到的key
func (idx *Database) BatchGetPinByKeys(keys []string, replace bool) map[string][]byte {
	shardMap := make(map[*pebble.DB][]int)
	for i, key := range keys {
		if key == "" {
			continue
		}
		if replace {
			key = strings.Replace(key, ":", "i", -1)
		}
		db := idx.getShard(key)
		shardMap[db] = append(shardMap[db], i)
	}
	results := make(map[string][]byte, len(keys))
	for db, idxs := range shardMap {
		for _, i := range idxs {
			val, closer, err := db.Get([]byte(keys[i]))
			if err == nil {
				buf := make([]byte, len(val))
				copy(buf, val)
				results[keys[i]] = buf
				closer.Close()
			}
		}
	}
	return results
}
func (idx *Database) BatchGetPinListByKeys(keys []string, replace bool) [][]byte {
	shardMap := make(map[*pebble.DB][]int)
	for i, key := range keys {
		if key == "" {
			continue
		}
		if replace {
			key = strings.Replace(key, ":", "i", -1)
		}
		db := idx.getShard(key)
		shardMap[db] = append(shardMap[db], i)
	}
	results := make([][]byte, 0, len(keys))
	for db, idxs := range shardMap {
		for _, i := range idxs {
			val, closer, err := db.Get([]byte(keys[i]))
			if err == nil {
				buf := make([]byte, len(val))
				copy(buf, val)
				results = append(results, buf)
				closer.Close()
			}
		}
	}
	return results
}

// 统一主键生成函数，保证写入和查询一致
func BuildPinKey(txid string, outputIndex int) string {
	return common.ConcatBytesOptimized([]string{txid, ":", strconv.Itoa(outputIndex)}, "")
}

// getShard 使用 xxhash 分片，保证分布均匀
func (idx *Database) getShard(key string) *pebble.DB {
	h := xxhash.Sum64String(key)
	return idx.PinsDBs[h%uint64(len(idx.PinsDBs))]
}

func SplitBytesOptimized(s, sep string) []string {
	if s == "" {
		return nil
	}
	return splitFast(s, sep)
}

func splitFast(s, sep string) []string {
	var res []string
	sepLen := len(sep)
	start := 0
	for i := 0; i+sepLen <= len(s); {
		if s[i:i+sepLen] == sep {
			res = append(res, s[start:i])
			start = i + sepLen
			i = start
		} else {
			i++
		}
	}
	res = append(res, s[start:])
	return res
}
