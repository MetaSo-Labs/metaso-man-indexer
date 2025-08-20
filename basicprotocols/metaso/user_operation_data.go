package metaso

import (
	"log"
	"os"
	"strings"
	"time"

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
func InitOperationDb() {
	dirPath := "./user_operation_data"
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		err := os.MkdirAll(dirPath, os.ModePerm)
		if err != nil {
			log.Fatalf("Failed to create directory: %v", err)
		}
	}
	PbWriteOpts = &pebble.WriteOptions{
		Sync: true, // 同步写入到磁盘
	}
	collections := []string{"readed_log"}
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

func AddUserOperationData(collection string, key string, value []byte) error {
	if db, ok := PbMap[collection]; ok {
		return db.Set([]byte(key), value, PbWriteOpts)
	}
	return nil
}
func GetUserOperationData(collection string, key string) ([]byte, error) {
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
func DeleteUserOperationData(collection string, key string) error {
	if db, ok := PbMap[collection]; ok {
		return db.Delete([]byte(key), PbWriteOpts)
	}
	return nil
}
func MergeUserOperationData(collection string, key string, value string) error {
	if db, ok := PbMap[collection]; ok {
		return db.Merge([]byte(key), []byte(value), PbWriteOpts)
	}
	return nil
}
func CleanOldUserOperationData(collection string, key string) error {
	if db, ok := PbMap[collection]; ok {
		value, closer, err := db.Get([]byte(key))
		if err != nil {
			return err
		}
		defer closer.Close()

		// 获取当前时间并计算10天前的时间戳
		threshold := time.Now().AddDate(0, 0, -10)

		// 解析 value 并过滤掉 10 天前的数据
		entries := strings.Split(string(value), ",")
		var updatedEntries []string
		for _, entry := range entries {
			parts := strings.Split(entry, "_")
			if len(parts) != 2 {
				continue
			}
			if date, err := time.Parse("2006-01-02", parts[1]); err == nil {
				if date.After(threshold) {
					updatedEntries = append(updatedEntries, entry)
				}
			}
		}

		// 将过滤后的数据重新写入数据库
		newValue := strings.Join(updatedEntries, ",")
		return db.Set([]byte(key), []byte(newValue), PbWriteOpts)
	}
	return nil
}
