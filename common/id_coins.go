package common

import (
	"crypto/tls"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"
)

var IDCOINS map[string]string

func SyncIdCoins() {
	IDCOINS = make(map[string]string)
	// 启动定时任务，每10分钟执行一次
	GetIcCoinListFromNet()
	go func() {
		ticker := time.NewTicker(10 * time.Minute)
		defer ticker.Stop()
		for {
			GetIcCoinListFromNet()
			<-ticker.C
		}
	}()
}

type idconinRes struct {
	Code    int            `json:"code"`
	Message string         `json:"message"`
	Data    idconinResData `json:"data"`
}
type idconinResData struct {
	List []struct {
		Tick            string `json:"tick"`
		DeployerAddress string `json:"deployerAddress"`
	} `json:"list"`
}

func GetIcCoinListFromNet() {
	url := "https://www.metaso.network/api-base/v1/common/idcoin/simple-info-list?cursor=0&size=100000"
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	resp, err := client.Get(url)
	if err != nil {
		log.Println("get idcoin list from net error:", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Println("get idcoin list from net error: status code", resp.StatusCode)
		return
	}
	var res idconinRes
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		log.Println("get idcoin list from net error:", err)
		return
	}
	if res.Code != 0 {
		log.Println("get idcoin list from net error: code", res.Code, "message", res.Message)
		return
	}
	for _, item := range res.Data.List {
		key := strings.ToLower(item.Tick)
		IDCOINS[key] = item.DeployerAddress
	}
}
