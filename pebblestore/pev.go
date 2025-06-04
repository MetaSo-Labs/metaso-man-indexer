package pebblestore

import (
	"fmt"
	"log"
	"manindexer/common"
	"manindexer/pin"
	"strconv"
	"strings"

	"github.com/bytedance/sonic"
)

func (idx *Database) GetMetaBlockData(from, to int64, chainName string,batchSize int, out chan<- []pin.PinInscription) (err error) {
	for i := from; i <= to; i++ {
		blockKey := fmt.Sprintf("blocktime_%s_%d", chainName,i)
		val, closer, err := idx.CountDB.Get([]byte(blockKey))
		if err != nil {
			continue
		}
		closer.Close()
		blockTime, err := strconv.ParseInt(string(val), 10, 64)
		if err != nil {
			continue
		}
		publicKeyStr := common.ConcatBytesOptimized([]string{fmt.Sprintf("%010d", blockTime), "&", chainName, "&", fmt.Sprintf("%010d", i)}, "")
		idVal, closer, err := idx.BlocksDB.Get([]byte(publicKeyStr))
		if err != nil {
			continue
		}
		defer closer.Close()
		pinIdList := strings.Split(string(idVal), ",")
		if len(pinIdList) <= 0 {
			continue
		}
		// result := idx.BatchGetPinListByKeys(pinIdList, false)
		// for _, val := range result {
		// 	var item pin.PinInscription
		// 	err := sonic.Unmarshal(val, &item)
		// 	if err == nil {
		// 		pinList = append(pinList, item)
		// 	}
		// }
	 // 分批处理
	 total := len(pinIdList)
	 for start := 0; start < total; start += batchSize {
		log.Println("Processing batch from index", start, "to", start+batchSize, "for block", i, "chain", chainName, "total pins", total)
		end := start + batchSize
		if end > total {
			end = total
		}
		batchIds := pinIdList[start:end]
		result := idx.BatchGetPinListByKeys(batchIds, false)
		var pinList []pin.PinInscription
		for _, val := range result {
			var item pin.PinInscription
			err := sonic.Unmarshal(val, &item)
			if err == nil {
				pinList = append(pinList, item)
			}
		}
		if len(pinList) > 0 {
			out <- pinList // 发送给调用者
		}
	}
	}
	return
}
