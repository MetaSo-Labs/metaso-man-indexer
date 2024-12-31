package metaso

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"manindexer/common"
	"manindexer/database/mongodb"
	"net/http"
)

func (metaso *MetaSo) syncPEV() {
	if common.Config.Statistics.MetaChainHost == "" || common.Config.Statistics.AllowHost == nil || common.Config.Statistics.AllowProtocols == nil {
		return
	}
	metaBlock, _ := metaso.getLastMetaBlock()
	if metaBlock == nil {
		return
	}
	if metaBlock.MetablockHeight <= 0 {
		return
	}
	log.Println("count metaBlock:", metaBlock.MetablockHeight)
	mongodb.UpdateSyncLastNumber("metablock", metaBlock.MetablockHeight)
	for _, chain := range metaBlock.Chains {
		CountBlockPEV(metaBlock.MetablockHeight, &chain)
	}
}
func (metaso *MetaSo) getLastMetaBlock() (metaBlock *MetaBlockData, err error) {
	localHeight, err := mongodb.GetSyncLastNumber("metablock")
	if err != nil {
		return
	}
	metaBlock = getMetaBlock(localHeight + 1)
	//fmt.Println("metaBlock:", metaBlock)
	return
}

type metaBlockRes struct {
	Code     int           `json:"code"`
	Data     MetaBlockData `json:"data"`
	Messsage string        `json:"messsage"`
}

func getMetaBlock(height int64) (metaBlock *MetaBlockData) {
	url := fmt.Sprintf("%s/api/block/info?number=%d", common.Config.Statistics.MetaChainHost, height)
	resp, err := http.Get(url)
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
	var data metaBlockRes
	err = json.Unmarshal(body, &data)
	if err != nil {
		fmt.Println("Error unmarshalling JSON:", err)
		return
	}
	metaBlock = &data.Data
	return
}
