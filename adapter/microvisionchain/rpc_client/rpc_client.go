package rpc_client

import (
	"encoding/base64"
	"errors"
	"fmt"

	"github.com/imroc/req"
	"github.com/tidwall/gjson"
)

type ClientInterface interface {
	Call(path string, request []interface{}) (*gjson.Result, error)
}

// A Client is a Bitcoin RPC client. It performs RPCs over HTTP using JSON
// request and responses. A Client must be configured with a secret token
// to authenticate with other Cores on the network.
type Client struct {
	URL         string
	AccessToken string
	Debug       bool
	client      *req.Req
}

type Response struct {
	Code    int         `json:"code, omitempty"`
	Error   interface{} `json:"error, omitempty"`
	Result  interface{} `json:"result, omitempty"`
	Message string      `json:"message, omitempty"`
	Id      string      `json:"id, omitempty"`
}

func NewClientNode(url string, accessToken string, debug bool) *Client {
	cli := &Client{
		URL:         url,
		AccessToken: accessToken,
		Debug:       debug,
	}

	api := req.New()
	cli.client = api
	return cli
}

func (cl *Client) Call(path string, request []interface{}) (*gjson.Result, error) {

	var body = make(map[string]interface{}, 0)

	if cl.client == nil {
		return nil, errors.New("Api url is not setup. ")
	}

	authHeader := req.Header{
		"Accept":        "Application/json",
		"Authorization": "Basic " + cl.AccessToken,
	}

	//json-rpc
	body["jsonrpc"] = "1.0"
	body["id"] = "1"
	body["method"] = path
	body["params"] = request

	if cl.Debug { //debug
		//log.Std.Info("Start Request API...")
		fmt.Println("Start Request API...")
	}

	r, err := cl.client.Post(cl.URL, req.BodyJSON(body), authHeader)

	if cl.Debug { //debug
		//log.Std.Info("Request API Completed")
		fmt.Println("Request API Completed")
	}

	if cl.Debug { //debug
		//log.Std.Info("%+v", r)
		fmt.Printf("%+v \n", r)
	}

	if err != nil {
		return nil, err
	}

	resp := gjson.ParseBytes(r.Bytes())
	err = IsError(&resp)
	if err != nil {
		return nil, err
	}

	result := resp.Get("result")
	return &result, nil
}

// See 2 (end of page 4) http://www.ietf.org/rfc/rfc2617.txt
// "To receive authorization, the client sends the userid and password,
// separated by a single colon (":") character, within a base64
// encoded string in the credentials."
// It is not meant to be urlencoded.
func BasicAuth(userName, password string) string {
	auth := userName + ":" + password
	return base64.StdEncoding.EncodeToString([]byte(auth))
}

// isError
func IsError(result *gjson.Result) error {
	/*
		{
			"result": null,
			"error": {
				"code": -8,
				"message": "Block height out of range"
			},
			"id": "foo"
		}
	*/
	var err error
	if !result.Get("error").IsObject() {
		if !result.Get("result").Exists() {
			return errors.New("Response is empty. ")
		}
		return nil
	}

	errInfo := fmt.Sprintf("[%d]%s",
		result.Get("error.code").Int(),
		result.Get("error.message").String())
	err = errors.New(errInfo)
	return err
}
