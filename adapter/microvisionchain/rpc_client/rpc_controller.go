package rpc_client

type ClientController struct {
	ClientMap map[string]*Client
}

var (
	MyClientController *ClientController
)

func NewClientController(username, password, url string) *ClientController {
	if MyClientController != nil {
		return MyClientController
	} else {
		MyClientController = &ClientController{
			ClientMap: make(map[string]*Client),
		}
	}

	accessToken := BasicAuth(username, password)
	MyClientController.ClientMap["mvc"] = NewClientNode(url, accessToken, false)

	return MyClientController
}

func (c *ClientController) BroadcastTx(chain, txHexStr string) (string, error) {
	request := []interface{}{
		txHexStr,
		false,
	}

	result, err := c.ClientMap[chain].Call("sendrawtransaction", request)
	if err != nil {
		return "", err
	}
	return result.String(), nil
}
