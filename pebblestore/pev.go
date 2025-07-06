package pebblestore

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"manindexer/common"
	"manindexer/pin"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/bytedance/sonic"
)

func (idx *Database) GetMetaBlockData(metaBlockHeight, from, to int64, chainName string, batchSize int, out chan<- []pin.PinInscription) (err error) {
	for i := from; i <= to; i++ {

		blockKey := fmt.Sprintf("blocktime_%s_%d", chainName, i)
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
		//获取费率
		feeRateInfo := make(map[string]int, total)
		if chainName == "mvc" {
			if metaBlockHeight >= 68 || metaBlockHeight == -1 {
				err := GetMvcBlockFee(i, &feeRateInfo)
				if err != nil {
					log.Println("Error getting MVC block fee:", err)
					return err
				}
			}
		}
		log.Println("PEV Processing batch from index,", "for block:", i, "chain", chainName, "total pins", total)
		for start := 0; start < total; start += batchSize {
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
				//费率过滤
				if chainName == "mvc" {
					if metaBlockHeight >= 68 || metaBlockHeight == -1 {
						if feeRate, ok := feeRateInfo[item.GenesisTransaction]; ok {
							if feeRate < int(common.Config.MetaSo.FeeLimit) {
								continue // 过滤掉费率低于阈值的交易
							}
						} else {
							continue // 如果没有费率信息，则跳过
						}
					}
				}
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

func GetMvcBlockFee(height int64, resultData *map[string]int) (err error) {
	url := fmt.Sprintf("%s/block/txall/%d", common.Config.MetaSo.FeeRateHost, height)
	client := &http.Client{
		Timeout: 10 * time.Minute,
	}
	resp, err := client.Get(url)
	if err != nil {
		fmt.Println("Error making GET request:", err)
		return
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return
	}
	var data []string
	err = json.Unmarshal(body, &data)
	if err != nil {
		fmt.Println("Error unmarshalling JSON:", err)
		return
	}
	cnt := 0
	passCnt := 0
	resultLength := len(data)
	for _, item := range data {
		parts := strings.Split(item, ":")
		if len(parts) != 3 {
			continue
		}
		txId := parts[0]
		feeRate, err := strconv.Atoi(parts[2])
		if err != nil {
			continue
		}
		(*resultData)[txId] = feeRate
		cnt += 1
		if feeRate >= int(common.Config.MetaSo.FeeLimit) {
			passCnt += 1
		}
	}
	log.Println("GetMvcBlockFee==>", height, "ResultNum:", resultLength, "---->: map transactions:", cnt, "passed fee Num:", passCnt, "fee config:", common.Config.MetaSo.FeeLimit)
	return nil
}
