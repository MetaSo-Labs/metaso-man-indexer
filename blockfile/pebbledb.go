package blockfile

import (
	"fmt"
	"log"
	"manindexer/common"
	"os"

	"github.com/cockroachdb/pebble"
)

var PbWriteOpts *pebble.WriteOptions
var PbMap = make(map[string]*pebble.DB)

type Logger struct{}

func (ml Logger) Infof(format string, args ...interface{}) {}
func (ml Logger) Fatalf(format string, args ...interface{}) {
	log.Println(format, args)
}
func (ml Logger) Errorf(format string, args ...interface{}) {
	log.Println(format, args)
}
func InitBlockFileDb() {

	dataPath = common.Config.Blockfile.DataPath
	syncHost = common.Config.Blockfile.SyncHost

	log.Println("InitBlockFileDb")
	dirPath := dataPath + "/pebble"
	//dirPath := "./user_operation_data"
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		err := os.MkdirAll(dirPath, os.ModePerm)
		if err != nil {
			log.Fatalf("Failed to create directory: %v", err)
		}
	}
	PbWriteOpts = &pebble.WriteOptions{
		Sync: true, // 同步写入到磁盘
	}
	collections := []string{"time_index", "meta"}
	for _, name := range collections {
		dbPath := dirPath + "/" + name
		var err error
		lg := Logger{}
		PbMap[name], err = pebble.Open(dbPath, &pebble.Options{
			Logger: lg,
			// MemTableSize: 128 << 20, // 128 MB
			// MaxConcurrentCompactions: func() int {
			// 	return 2
			// },
		})
		if err != nil {
			log.Printf("pebble %s init error:%v", name, err)
		} else {
			log.Printf("pebble %s open", name)
		}
	}
}
func CloseBlockFileDb() {
	for name, db := range PbMap {
		if db != nil {
			err := db.Close()
			if err != nil {
				log.Printf("pebble %s close error:%v", name, err)
			} else {
				log.Printf("pebble %s closed", name)
			}
		}
	}
}
func PebbleSetData(collection string, key string, value []byte) error {
	if db, ok := PbMap[collection]; ok {
		return db.Set([]byte(key), value, PbWriteOpts)
	}
	return nil
}
func PebbleGetData(collection string, key string) ([]byte, error) {
	if db, ok := PbMap[collection]; ok {
		value, closer, err := db.Get([]byte(key))
		if err != nil {
			return nil, err
		}
		defer closer.Close()
		return value, nil
	}
	return nil, nil
}
func QueryKeysByTimeRange(startTs, endTs int64) ([]string, error) {
	db, ok := PbMap["time_index"]
	if !ok {
		return nil, fmt.Errorf("collection %s not found", "time_index")
	}
	var keys []string
	startKey := fmt.Sprintf("%012d_", startTs) // 起始时间戳零填充
	endKey := fmt.Sprintf("%012d_", endTs)     // 结束时间戳零填充

	iter, err := db.NewIter(nil)
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	for ok := iter.SeekGE([]byte(startKey)); ok; ok = iter.Next() {
		k := string(iter.Key())
		// fmt.Println("Found key:", k)
		if k >= endKey {
			break
		}
		keys = append(keys, k)
	}
	return keys, nil
}
