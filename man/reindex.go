package man

import (
	"bytes"
	"encoding/json"
	"fmt"
	"manindexer/common"
	"manindexer/database/mongodb"
	"manindexer/pin"
	"net/http"
	"time"
)

func SetAllMetaidToCache() {
	for {
		doSetAllMetaidToCache()
		time.Sleep(30 * time.Minute) // 每30分钟执行一次
	}
}
func doSetAllMetaidToCache() {
	var lastID string       // 初始为空，表示从头开始
	batchSize := int64(500) // 每次获取 500 条记录
	apiURL := common.Config.CacheUrl + "/v1/users/batchset/"
	for {
		// 获取一批数据
		pins, nextLastID, err := mongodb.FetchMetaIdInfoBatch(lastID, batchSize)
		if err != nil {
			fmt.Println("Error fetching data:", err)
			break
		}
		// 如果没有数据，退出循环
		if len(pins) == 0 {
			fmt.Println("No more data to process.")
			break
		}
		// 将当前批次数据提交给目标系统
		err = submitPinsToAPI(apiURL, pins)
		if err != nil {
			fmt.Printf("Error submitting data to API: %v\n", err)
			break
		}
		// 更新 lastID 为当前批次的最后一个 _id
		lastID = nextLastID
	}
}

// 提交数据到目标系统的接口
func submitPinsToAPI(apiURL string, pins []*pin.MetaIdInfo) error {
	// 将 pins 转换为 JSON
	jsonData, err := json.Marshal(pins)
	if err != nil {
		return fmt.Errorf("failed to marshal pins: %v", err)
	}

	// 创建 HTTP POST 请求
	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// 发送请求
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send HTTP request: %v", err)
	}
	defer resp.Body.Close()

	// 检查响应状态码
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("received non-OK response: %s", resp.Status)
	}

	fmt.Printf("Successfully submitted %d records to API\n", len(pins))
	return nil
}
