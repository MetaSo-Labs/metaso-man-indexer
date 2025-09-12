package common_service

import (
	"errors"
	"fmt"
	"manindexer/common"
)

var (
	MempoolSpace string = "https://mempool.space"
)

func BroadcastTx(net, hex string) (string, error) {
	var (
		url     string
		code    int
		result  string
		err     error
		headers map[string]string = map[string]string{}
	)

	fmt.Println(hex)
	url = fmt.Sprintf("%s/api/tx", MempoolSpace)
	if net == "testnet" {
		url = fmt.Sprintf("%s/%s/api/tx", MempoolSpace, net)
	}
	fmt.Println(url)
	result, code, err = common.PostUrlAndCode(url, hex, headers)
	if err != nil {
		return "", err
	}
	fmt.Println(result)
	fmt.Println(code)
	//return result, nil
	if code != 200 {
		return "", errors.New(fmt.Sprintf("Post err: code not 200, msg:%s", result))
	}

	return result, nil
}
