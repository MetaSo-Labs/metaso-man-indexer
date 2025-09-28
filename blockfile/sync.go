package blockfile

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"manindexer/common"
	"manindexer/pin"
	"time"

	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/cockroachdb/pebble"
	"github.com/klauspost/compress/zstd"
	"google.golang.org/protobuf/proto"
)

type BlockData struct {
}

var syncHost = "https://man.metaid.io"
var dataPath = "./blockfile_data"
var defaultBeginHeight map[string]int64
var fileChainName []string

func InitBlockFile() {
	defaultBeginHeight = make(map[string]int64)
	// defaultBeginHeight["mvc"] = 139410
	// defaultBeginHeight["btc"] = 910000
	syncHost = common.Config.Blockfile.SyncHost
	defaultBeginHeight["mvc"] = common.Config.Blockfile.DefaultBeginHeightMVC
	defaultBeginHeight["btc"] = common.Config.Blockfile.DefaultBeginHeightBTC
	fileChainName = []string{"mvc", "btc"}

	//打印defaultBeginHeight
	log.Printf("[BLOCKFILE]DefaultBeginHeight: %v", defaultBeginHeight)
	log.Printf("[BLOCKFILE]SyncHost: %v", syncHost)
}

func (b *BlockData) Sync() error {
	for {
		err := DoSync(10000)
		if err != nil {
			// return err
			log.Printf("[BLOCKFILE]Sync failed: %v", err)
		}
		time.Sleep(3 * time.Minute)
	}
}
func DoSync(step int64) error {
	for _, chain := range fileChainName {
		var localLastHeight int64
		localDbLastHeight, err := PebbleGetData("meta", chain+"_lastheight")
		if err != nil {
			if err == pebble.ErrNotFound {
				localLastHeight = defaultBeginHeight[chain]
			} else {
				log.Printf("[BLOCKFILE]PebbleGetData failed: %v", err)
				return err
			}
		} else {
			localLastHeight, err = strconv.ParseInt(string(localDbLastHeight), 10, 64)
			if err != nil {
				log.Printf("[BLOCKFILE]ParseInt failed: %v", err)
				return err
			}
		}
		if localLastHeight <= 0 {
			log.Printf("[BLOCKFILE]localLastHeight <= 0")
			return errors.New("localLastHeight <= 0")
		}
		for h := localLastHeight + 1; h <= localLastHeight+step; h++ {
			err := DownloadFile(chain, int(h))
			if err != nil {
				if err.Error() == "noFile" {
					// log.Printf("[BLOCKFILE]链 %s 同步完成, 正在同步高度:%d 没有文件，当前local高度 %d", chain, h, localLastHeight)
					continue
				} else if err.Error() == "height is greater than MaxHeight" {
					log.Printf("[BLOCKFILE]链 %s 已同步完成，到最新高度 %d", chain, h-1)
					break
				} else {
					log.Printf("[BLOCKFILE]链 %s 同步高度 %d 失败: %v", chain, h, err)
					break
				}
			} else {
				log.Printf("[BLOCKFILE]链 %s 成功同步高度 %d", chain, h)
			}
		}
	}
	return nil
}

// GetBlockFilePath 根据区块高度计算存储路径
// 采用 /百万位/千位/高度.dat.zst 的结构
func GetBlockFilePath(chainName string, height int64, partIndex int) string {
	million := height / 1000000
	thousand := (height % 1000000) / 1000
	lastName := chainName + "_" + strconv.FormatInt(height, 10) + "_" + strconv.Itoa(partIndex) + ".dat.zst"
	return filepath.Join(
		dataPath+"/file",
		strconv.FormatInt(million, 10),
		strconv.FormatInt(thousand, 10),
		lastName,
	)
}
func DownloadFile(chainName string, height int) error {
	partUrl := fmt.Sprintf("%s/api/block/file/partCount?chain=%s&height=%d", syncHost, chainName, height)
	resp, err := http.Get(partUrl)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	type partResp struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			BtcMax    int64 `json:"btcMax"`
			BtcMin    int64 `json:"btcMin"`
			MvcMax    int64 `json:"mvcMax"`
			MvcMin    int64 `json:"mvcMin"`
			PartCount int   `json:"partCount"`
		} `json:"data"`
	}
	var pr partResp
	if err := json.NewDecoder(resp.Body).Decode(&pr); err != nil {
		return err
	}
	if pr.Code != 1 {
		return fmt.Errorf("partCount api error: %s", pr.Message)
	}
	if pr.Data.PartCount == 0 {
		if chainName == "btc" {
			if int64(height) > pr.Data.BtcMax {
				return errors.New("height is greater than MaxHeight")
			}
		} else if chainName == "mvc" {
			if int64(height) > pr.Data.MvcMax {
				return errors.New("height is greater than MaxHeight")
			}
		}
		PebbleSetData("meta", chainName+"_lastheight", []byte(strconv.FormatInt(int64(height), 10)))

		//log.Printf("链 %s 高度 %d 没有分片", chainName, height)
		// return errors.New("partCount is 0")
		return errors.New("noFile")
	}
	for i := 0; i < pr.Data.PartCount; i++ {
		fileUrl := fmt.Sprintf("%s/api/block/file?chain=%s&height=%d&part=%d", syncHost, chainName, height, i)
		fileResp, err := http.Get(fileUrl)
		if err != nil {
			return err
		}
		defer fileResp.Body.Close()
		filePath := GetBlockFilePath(chainName, int64(height), i)
		dirPath := filepath.Dir(filePath)
		if err := os.MkdirAll(dirPath, 0755); err != nil {
			return fmt.Errorf("创建目录 %s 失败: %w", dirPath, err)
		}

		out, err := os.Create(filePath)
		if err != nil {
			return err
		}
		defer out.Close()
		_, err = io.Copy(out, fileResp.Body)
		if err != nil {
			return err
		}
		if i == 0 {
			blocks, err := LoadFBlockPart(chainName, int64(height), 0)
			if err == nil && len(blocks) > 0 {
				var pinNode pin.PinInscription
				err := json.Unmarshal(blocks[0], &pinNode)
				if err == nil {
					SaveBlockTimeIndex(pinNode)
				}
			}
		}
	}
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
	var bytesList BlockBytesList
	if err := proto.Unmarshal(decompressedData, &bytesList); err != nil {
		return nil, fmt.Errorf("反序列化区块 %d 失败: %w", height, err)
	}
	// log.Printf("成功从 %s 加载区块 %d", filePath, height)
	return bytesList.Items, nil
}
func SaveBlockTimeIndex(pinNode pin.PinInscription) error {
	// 时间戳和高度零填充到12位，链名保持原样
	key := fmt.Sprintf("%012d_%012d_%s", pinNode.Timestamp, pinNode.GenesisHeight, pinNode.ChainName)
	PebbleSetData("time_index", key, nil)
	PebbleSetData("meta", pinNode.ChainName+"_lastheight", []byte(strconv.FormatInt(pinNode.GenesisHeight, 10)))
	return nil
}
func arrayContains(arr []string, str string) bool {
	for _, a := range arr {
		if a == str {
			return true
		}
	}
	return false
}

// 获取时间范围内的指定host和协议的PIN列表
func QueryPinsByTimeRange(startTs, endTs int64, host, protocol []string) ([]pin.PinInscription, error) {
	keys, err := QueryKeysByTimeRange(startTs, endTs)
	if err != nil {
		return nil, err
	}
	var result []pin.PinInscription
	for _, key := range keys {
		// key格式: 时间戳_高度_链名
		parts := len(key)
		if parts < 26 { // 最小长度检查
			continue
		}
		chainName := key[26:]
		heightStr := key[13:25]
		height, err := strconv.ParseInt(heightStr, 10, 64)
		if err != nil {
			continue
		}
		fmt.Println("chainName:", chainName, "height:", height)
		partIndex := 0
		for {
			blocks, err := LoadFBlockPart(chainName, height, partIndex)
			if err != nil {
				break
			}
			for _, blockData := range blocks {
				var pinNode pin.PinInscription
				err := json.Unmarshal(blockData, &pinNode)
				if err != nil {
					continue
				}
				if len(host) > 0 {
					if !arrayContains(host, pinNode.Host) {
						continue
					}
				}
				if len(protocol) > 0 {
					if !arrayContains(protocol, pinNode.Path) {
						continue
					}
				}
				result = append(result, pinNode)
			}
			partIndex++
		}
	}
	return result, nil
}

// GetPinsByChainAndHeight 根据链名和高度获取所有pin数据列表
func GetPinsByChainAndHeight(chainName string, height int64, hosts []string, protocols []string) ([]pin.PinInscription, error) {
	if chainName == "" || height <= 0 {
		return nil, fmt.Errorf("invalid parameters: chainName and height must be provided")
	}

	var result []pin.PinInscription
	partIndex := 0

	// 遍历所有分片
	for {
		blocks, err := LoadFBlockPart(chainName, height, partIndex)
		if err != nil {
			// 如果没有更多分片，退出循环
			if err.Error() == "noFile" {
				fmt.Println("noFile for chain %s height %d", chainName, height)
				break
			}
			return nil, fmt.Errorf("failed to load block part %d for chain %s height %d: %v", partIndex, chainName, height, err)
		}
		fmt.Println("blocks:", len(blocks))

		// 处理当前分片中的所有pin
		for _, blockData := range blocks {
			var pinNode pin.PinInscription
			err := json.Unmarshal(blockData, &pinNode)
			if err != nil {
				log.Printf("[BLOCKFILE]Failed to unmarshal pin data: %v", err)
				continue
			}

			// 过滤host（如果提供了hosts参数）
			if len(hosts) > 0 {
				if !arrayContains(hosts, pinNode.Host) {
					continue
				}
			}

			// 过滤protocol（如果提供了protocols参数）
			if len(protocols) > 0 {
				if !arrayContains(protocols, pinNode.Path) {
					continue
				}
			}

			result = append(result, pinNode)
		}

		partIndex++
	}

	log.Printf("[BLOCKFILE]Found %d pins for chain %s height %d", len(result), chainName, height)
	return result, nil
}
