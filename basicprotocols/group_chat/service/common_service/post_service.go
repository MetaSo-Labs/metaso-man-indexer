package common_service

import (
	"encoding/json"
	"fmt"
	"manindexer/common"
)

const (
	code_success = int64(0)
)

func WsPost(pinId string, msgData interface{}, metaIdList []string) {
	var (
		url    string
		result string
		err    error
		resp   *wsPostResponse
		req    map[string]interface{} = make(map[string]interface{})
	)
	req["msgData"] = msgData
	req["postType"] = 22
	req["metaIdList"] = metaIdList

	url = fmt.Sprintf("%s/v1/ws/post/msg", "http://172.31.165.162:8298")
	result, err = common.PostUrl(url, req, nil)
	if err != nil {
		return
	}

	resp = &wsPostResponse{}
	if err = json.Unmarshal([]byte(result), resp); err != nil {
		return
	}

	if resp.Code != code_success {
		//fmt.Printf("Set err:%s\n", resp)
	} else {
		fmt.Printf("Ws Post RoomChat success[%s]\n %+v\n", pinId, msgData)
	}
	return
}
