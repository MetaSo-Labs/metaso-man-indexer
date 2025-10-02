package pebblestore

import (
	"fmt"
	"log"
	"manindexer/common"
	"manindexer/pin"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/bytedance/sonic"
	"github.com/cespare/xxhash/v2"
	"github.com/cockroachdb/pebble"
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
	PinsDBs       []*pebble.DB
	PinSort       *pebble.DB
	BlocksDB      *pebble.DB
	CountDB       *pebble.DB
	PathPinDB     *pebble.DB
	AddressDB     *pebble.DB
	CreatorDb     *pebble.DB
	MrcDb         *pebble.DB
	PinsMempoolDb *pebble.DB // 用于存储mempool中的pins数据
	NotifcationDb *pebble.DB // 用于存储通知数据
	MetaDb        *pebble.DB // 用于存储meta数据
}
type customLogger struct{}

func (l *customLogger) Infof(format string, args ...interface{})  {}
func (l *customLogger) Fatalf(format string, args ...interface{}) {}
func (l *customLogger) Errorf(format string, args ...interface{}) {}

// NewDataBase 创建索引器，自动创建分片db、pages、blocks独立db
func NewDataBase(basePath string, shardNum int) (*Database, error) {
	log.Println("=========NEW PEBBLE DATABASE========")
	dbOptions := &pebble.Options{
		//Logger: noopLogger,
		Levels: []pebble.LevelOptions{
			{
				Compression: pebble.NoCompression,
			},
		},
		MemTableSize:                32 << 20, // 降低为32MB (默认64MB)
		MemTableStopWritesThreshold: 2,        // 默认4
		// 限制 block cache 大小（比如 128MB，可根据机器内存调整）
		Cache: pebble.NewCache(128 << 20), // 128MB
		// 限制 table cache 数量（比如 64）
		MaxOpenFiles: 64,
	}
	pinsDBs := make([]*pebble.DB, shardNum)
	for i := 0; i < shardNum; i++ {
		dir := fmt.Sprintf("%s/pins_%d", basePath, i)
		os.MkdirAll(dir, 0755)
		db, err := pebble.Open(fmt.Sprintf("%s/db", dir), dbOptions)
		if err != nil {
			log.Println(err)
			return nil, err
		}
		pinsDBs[i] = db
	}
	os.MkdirAll(fmt.Sprintf("%s/pinsort", basePath), 0755)
	pinSortDb, err := pebble.Open(fmt.Sprintf("%s/pinsort/db", basePath), dbOptions)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	os.MkdirAll(fmt.Sprintf("%s/blocks", basePath), 0755)
	blocksDB, err := pebble.Open(fmt.Sprintf("%s/blocks/db", basePath), dbOptions)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	os.MkdirAll(fmt.Sprintf("%s/blocks", basePath), 0755)
	countDB, err := pebble.Open(fmt.Sprintf("%s/count/db", basePath), dbOptions)
	if err != nil {
		return nil, err
	}
	os.MkdirAll(fmt.Sprintf("%s/blocks", basePath), 0755)
	pathPinDB, err := pebble.Open(fmt.Sprintf("%s/path/db", basePath), dbOptions)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	os.MkdirAll(fmt.Sprintf("%s/blocks", basePath), 0755)
	addressDB, err := pebble.Open(fmt.Sprintf("%s/address/db", basePath), dbOptions)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	os.MkdirAll(fmt.Sprintf("%s/creator", basePath), 0755)
	creatorDb, err := pebble.Open(fmt.Sprintf("%s/creator/db", basePath), dbOptions)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	os.MkdirAll(fmt.Sprintf("%s/mempool", basePath), 0755)
	mempoolDb, err := pebble.Open(fmt.Sprintf("%s/mempool/db", basePath), dbOptions)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	os.MkdirAll(fmt.Sprintf("%s/notifcation", basePath), 0755)
	notifcationDb, err := pebble.Open(fmt.Sprintf("%s/notifcation/db", basePath), dbOptions)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	os.MkdirAll(fmt.Sprintf("%s/meta", basePath), 0755)
	metaDb, err := pebble.Open(fmt.Sprintf("%s/meta/db", basePath), dbOptions)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	return &Database{PinsDBs: pinsDBs, PinSort: pinSortDb, BlocksDB: blocksDB,
		CountDB: countDB, PathPinDB: pathPinDB, AddressDB: addressDB,
		CreatorDb: creatorDb, PinsMempoolDb: mempoolDb, NotifcationDb: notifcationDb, MetaDb: metaDb}, nil
}

// Close 关闭所有数据库
func (idx *Database) Close() error {
	for _, db := range idx.PinsDBs {
		db.Close()
	}
	idx.PinSort.Close()
	idx.BlocksDB.Close()
	idx.PathPinDB.Close()
	idx.AddressDB.Close()
	idx.CountDB.Close()
	idx.CreatorDb.Close()
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
func (idx *Database) BatchInsertPathPins(data map[string]string) error {
	batch := idx.PathPinDB.NewBatch()
	for k, v := range data {
		batch.Set([]byte(k), []byte(v), nil)
	}
	if err := batch.Commit(nil); err != nil {
		batch.Close()
		return err
	}
	batch.Close()
	return nil

}
func (idx *Database) BatchMergeAddressData(data map[string]string) error {
	batch := idx.AddressDB.NewBatch()
	for k, v := range data {
		batch.Merge([]byte(k), []byte(v), nil)
	}
	if err := batch.Commit(nil); err != nil {
		batch.Close()
		return err
	}
	batch.Close()
	return nil
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

// InsertPinSort 插入pin的排序
func (idx *Database) InsertPinSort(db *pebble.DB, sortLsit []string) error {
	batch := db.NewBatch()
	for _, key := range sortLsit {
		batch.Set([]byte(key), nil, nil)
	}
	if err := batch.Commit(nil); err != nil {
		batch.Close()
		return err
	}
	batch.Close()
	return nil
}

// InsertBlockTxs 插入区块交易表
func (idx *Database) InsertBlockTxs(blockKey string, data string) error {
	return idx.BlocksDB.Set([]byte(blockKey), []byte(data), pebble.Sync)
}

// 获取BlocksDB所有的数据
func (idx *Database) GetlBlocksDB(chainName string, height int) (*string, error) {
	it, err := idx.BlocksDB.NewIter(nil)
	if err != nil {
		return nil, err
	}
	defer it.Close()
	searchKey := fmt.Sprintf("&%s&%010d", chainName, height)
	for it.First(); it.Valid(); it.Next() {
		key := string(it.Key())
		if strings.Contains(key, searchKey) {
			val := string(it.Value())
			return &val, nil
		}
	}
	return nil, nil
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
func (idx *Database) QueryPinPageList(db *pebble.DB, q PageQuery) (PageResult, error) {
	it, _ := idx.PinSort.NewIter(nil)
	defer it.Close()
	var keys []string
	skip := q.Page * q.Size
	count := 0
	// 从最后一个 key 开始倒序遍历
	for it.Last(); it.Valid(); it.Prev() {
		if skip > 0 {
			skip--
			continue
		}
		arr := strings.Split(string(it.Key()), "&")
		key := "err"
		if len(arr) >= 4 {
			key = arr[3]
		}
		keys = append(keys, key)
		count++
		if count >= q.Size {
			break
		}
	}
	res := PageResult{
		List:   keys,
		NextId: "",
	}
	if len(keys) > 0 {
		res.NextId = keys[len(keys)-1]
	}
	return res, nil
}

func (idx *Database) GetBlockPageList(page int, size int, limit int) (PageResult []PageBlock, err error) {
	it, _ := idx.BlocksDB.NewIter(nil)
	defer it.Close()
	skip := page * size
	count := 0
	// 从最后一个 key 开始倒序遍历
	for it.Last(); it.Valid(); it.Prev() {
		if skip > 0 {
			skip--
			continue
		}
		pinIdList := strings.Split(string(it.Value()), ",")
		if len(pinIdList) < 0 {
			continue
		}
		if len(pinIdList) > limit {
			pinIdList = pinIdList[0:limit]
		}
		result := idx.BatchGetPinListByKeys(pinIdList, false)
		blockData := PageBlock{}
		keyArr := strings.Split(string(it.Key()), "&")
		blockData.BlockTime = keyArr[0]
		blockData.ChainName = keyArr[1]
		blockData.BlockHeight = keyArr[2]
		for _, val := range result {
			var item pin.PinInscription
			err := sonic.Unmarshal(val, &item)
			if err == nil {
				blockData.PinList = append(blockData.PinList, item)
			}
		}
		PageResult = append(PageResult, blockData)
		count++
		if count >= size {
			break
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
func (idx *Database) GetPinInscriptionByKey(key string) (pinNode pin.PinInscription, err error) {
	db := idx.getShard(key)
	val, closer, err := db.Get([]byte(key))
	if err != nil {
		return
	}
	closer.Close() // 直接调用，避免 defer 带来的性能损耗
	if val != nil && len(val) == 0 {
		return
	}
	err = sonic.Unmarshal(val, &pinNode)
	return
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
func (idx *Database) BatchGetPinListByKeys_bak(keys []string, replace bool) [][]byte {
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
	results := make([][]byte, len(keys))
	var wg sync.WaitGroup

	for db, idxs := range shardMap {
		wg.Add(1)
		go func(db *pebble.DB, idxs []int) {
			defer wg.Done()
			// 分片内再并发
			const innerBatch = 32 // 可根据实际情况调整，32，64，128
			var innerWg sync.WaitGroup
			for i := 0; i < len(idxs); i += innerBatch {
				end := i + innerBatch
				if end > len(idxs) {
					end = len(idxs)
				}
				innerWg.Add(1)
				go func(batchIdxs []int) {
					defer innerWg.Done()
					for _, idx := range batchIdxs {
						val, closer, err := db.Get([]byte(keys[idx]))
						if err == nil {
							buf := make([]byte, len(val))
							copy(buf, val)
							results[idx] = buf
							closer.Close()
						}
					}
				}(idxs[i:end])
			}
			innerWg.Wait()
		}(db, idxs)
	}
	wg.Wait()

	// 去除未命中的 nil
	final := make([][]byte, 0, len(keys))
	for _, v := range results {
		if v != nil {
			final = append(final, v)
		}
	}
	return final
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
func CountKeys(db *pebble.DB, prefix []byte) (int, error) {
	it, err := db.NewIter(nil)
	if err != nil {
		return 0, err
	}
	defer it.Close()
	count := 0
	for it.First(); it.Valid(); it.Next() {
		if prefix == nil || len(prefix) == 0 || strings.HasPrefix(string(it.Key()), string(prefix)) {
			count++
		}
	}
	return count, nil
}

func CountAllShards(dbs []*pebble.DB, prefix []byte) (int, error) {
	var total int64
	var wg sync.WaitGroup
	var errOnce sync.Once
	var firstErr error
	for _, db := range dbs {
		wg.Add(1)
		go func(db *pebble.DB) {
			defer wg.Done()
			n, err := CountKeys(db, prefix)
			if err != nil {
				errOnce.Do(func() { firstErr = err })
				return
			}
			atomic.AddInt64(&total, int64(n))
		}(db)
	}
	wg.Wait()
	return int(total), firstErr
}
func GetAllPinId(dbs []*pebble.DB, allPinIdMap *sync.Map) error {
	var wg sync.WaitGroup
	var errOnce sync.Once
	var firstErr error
	ch := make(chan map[string]struct{}, len(dbs))

	for _, db := range dbs {
		wg.Add(1)
		go func(db *pebble.DB) {
			defer wg.Done()
			localMap := make(map[string]struct{})
			it, err := db.NewIter(nil)
			if err != nil {
				errOnce.Do(func() { firstErr = err })
				return
			}
			defer it.Close()
			for it.First(); it.Valid(); it.Next() {
				localMap[string(it.Key())] = struct{}{}
			}
			ch <- localMap
		}(db)
	}
	wg.Wait()
	close(ch)
	for m := range ch {
		for k := range m {
			allPinIdMap.Store(k, struct{}{})
		}
	}
	return firstErr
}
