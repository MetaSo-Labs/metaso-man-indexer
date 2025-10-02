package man

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"manindexer/common"
	"manindexer/pin"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/cockroachdb/pebble"
	"github.com/klauspost/compress/zstd"
	"google.golang.org/protobuf/proto"
)

func GetBlockIdList(chainName string, height int) (*string, error) {
	result, err := PebbleStore.Database.GetlBlocksDB(chainName, height)
	return result, err
}
func SaveBlockFile(chainName string, height int) error {
	result, err := PebbleStore.Database.GetlBlocksDB(chainName, height)
	updateLastBlockHeight(chainName, height)
	if result == nil || err != nil {
		return errors.New("noData")
	}
	keys := strings.Split(*result, ",")
	batchSize := 20000
	total := len(keys)
	partIndex := 0

	for i := 0; i < total; i += batchSize {
		end := i + batchSize
		if end > total {
			end = total
		}
		batchKeys := keys[i:end]
		pings := PebbleStore.Database.BatchGetPinByKeys(batchKeys, false)
		var pingsData [][]byte
		for _, v := range pings {
			pingsData = append(pingsData, v)
		}
		// 保存区块数据到文件
		err = SaveFBlockPart(pingsData, chainName, int64(height), partIndex)
		if err != nil {
			log.Fatalf("保存区块文件失败: %v", err)
			return err
		}
		pingsData = nil
		pings = nil
		partIndex++
	}
	fmt.Println("==>SaveBlockFile done:", chainName, height)

	result = nil
	runtime.GC()
	return nil
}
func SaveBlockFileFromChain(chainName string, height int64) error {
	pins, _, _ := IndexerAdapter[chainName].CatchPins(height)
	batchSize := 20000
	total := len(pins)
	partIndex := 0

	for i := 0; i < total; i += batchSize {
		end := i + batchSize
		if end > total {
			end = total
		}
		batchKeys := pins[i:end]
		var pingsData [][]byte
		for _, v := range batchKeys {
			j, err := json.Marshal(v)
			if err != nil {
				continue
			}
			pingsData = append(pingsData, j)
		}
		// 保存区块数据到文件
		err := SaveFBlockPart(pingsData, chainName, int64(height), partIndex)
		if err != nil {
			log.Fatalf("保存区块文件失败: %v", err)
			return err
		}
		pingsData = nil
		batchKeys = nil
		partIndex++
	}
	fmt.Println("==>SaveBlockFile from Chain done:", chainName, height)
	updateLastBlockHeight(chainName, int(height))
	pins = nil
	runtime.GC()
	return nil
}
func updateLastBlockHeight(chainName string, height int) error {
	minKey := "blockFile_minHeight_" + chainName
	maxKey := "blockFile_maxHeight_" + chainName
	minValue, closer, err := PebbleStore.Database.MetaDb.Get([]byte(minKey))
	if err != nil {
		if errors.Is(err, pebble.ErrNotFound) {
			PebbleStore.Database.MetaDb.Set([]byte(minKey), []byte(strconv.Itoa(height)), pebble.Sync)
		}
	}
	if closer != nil {
		closer.Close()
	}
	minHeight, _ := strconv.Atoi(string(minValue))
	if height < minHeight {
		PebbleStore.Database.MetaDb.Set([]byte(minKey), []byte(strconv.Itoa(height)), pebble.Sync)
	}
	maxValue, closer, err := PebbleStore.Database.MetaDb.Get([]byte(maxKey))
	if err != nil {
		if errors.Is(err, pebble.ErrNotFound) {
			PebbleStore.Database.MetaDb.Set([]byte(maxKey), []byte(strconv.Itoa(height)), pebble.Sync)
		}
	}
	if closer != nil {
		closer.Close()
	}
	maxHeight, _ := strconv.Atoi(string(maxValue))
	if height > maxHeight {
		PebbleStore.Database.MetaDb.Set([]byte(maxKey), []byte(strconv.Itoa(height)), pebble.Sync)
	}
	return nil
}

func GetFileMetaHeight(chainName string) (minHeight int, maxHeight int, err error) {
	minKey := "blockFile_minHeight_" + chainName
	maxKey := "blockFile_maxHeight_" + chainName
	minValue, closer, err := PebbleStore.Database.MetaDb.Get([]byte(minKey))
	if err != nil {
		if errors.Is(err, pebble.ErrNotFound) {
			err = nil
			return
		}
		return
	}
	if closer != nil {
		closer.Close()
	}
	minHeight, _ = strconv.Atoi(string(minValue))

	maxValue, closer, err := PebbleStore.Database.MetaDb.Get([]byte(maxKey))
	if err != nil {
		if errors.Is(err, pebble.ErrNotFound) {
			err = nil
			return
		}
		return
	}
	if closer != nil {
		closer.Close()
	}
	maxHeight, _ = strconv.Atoi(string(maxValue))
	return
}

// GetBlockFilePath 根据区块高度计算存储路径
// 采用 /百万位/千位/高度.dat.zst 的结构
func GetBlockFilePath(chainName string, height int64, partIndex int) string {
	million := height / 1000000
	thousand := (height % 1000000) / 1000
	lastName := chainName + "_" + strconv.FormatInt(height, 10) + "_" + strconv.Itoa(partIndex) + ".dat.zst"
	return filepath.Join(
		common.Config.Pebble.Dir+"/blockFiles",
		strconv.FormatInt(million, 10),
		strconv.FormatInt(thousand, 10),
		lastName,
	)
}

// SaveBlock 将一个区块序列化、压缩并保存到文件
func SaveFBlockPart(blockData [][]byte, chainName string, height int64, partIndex int) error {
	// 1. 使用 Protobuf 序列化
	bytesList := &pin.BlockBytesList{Items: blockData}
	serialized, err := proto.Marshal(bytesList)
	if err != nil {
		return err
	}

	// 2. 使用 Zstandard 压缩
	var compressedData []byte
	encoder, _ := zstd.NewWriter(nil)
	compressedData = encoder.EncodeAll(serialized, make([]byte, 0, len(serialized)))
	encoder.Close()
	// 3. 计算并创建存储路径
	filePath := GetBlockFilePath(chainName, height, partIndex)
	dirPath := filepath.Dir(filePath)
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return fmt.Errorf("创建目录 %s 失败: %w", dirPath, err)
	}

	// 4. 写入文件
	if err := os.WriteFile(filePath, compressedData, 0644); err != nil {
		return fmt.Errorf("写入文件 %s 失败: %w", filePath, err)
	}

	//log.Printf("成功保存区块 %d 到 %s (原始大小: %d, 压缩后: %d)", block.Height, filePath, len(data), len(compressedData))
	return nil
}

// LoadFBlock 从文件加载、解压并反序列化一个区块
func LoadFBlockPart(chainName string, height int64, partIndex int) ([][]byte, error) {
	filePath := GetBlockFilePath(chainName, height, partIndex)
	if _, err := os.Stat(filePath); err != nil {
		return nil, errors.New("noFile")
	}
	// 1. 读取压缩文件
	compressedData, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("读取文件 %s 失败: %w", filePath, err)
	}

	// 2. 使用 Zstandard 解压
	decoder, _ := zstd.NewReader(nil)
	decompressedData, err := decoder.DecodeAll(compressedData, nil)
	if err != nil {
		return nil, fmt.Errorf("解压区块 %d 失败: %w", height, err)
	}
	decoder.Close()

	// 3. 使用 Protobuf 反序列化
	var bytesList pin.BlockBytesList
	if err := proto.Unmarshal(decompressedData, &bytesList); err != nil {
		return nil, fmt.Errorf("反序列化区块 %d 失败: %w", height, err)
	}
	log.Printf("成功从 %s 加载区块 %d", filePath, height)
	return bytesList.Items, nil
}
